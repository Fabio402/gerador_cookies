package scraper

import (
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/andybalholm/brotli"
)

type Config struct {
	Domain              string
	SensorUrl           string
	SensorPostLimit     int
	Language            string
	LowSecurity         bool
	UseScript           bool
	ForceUpdateDynamics bool
	EncodedData         string
	AkamaiProvider      string
	SbSdProvider        string
	SbSd                bool
	UserAgent           string
	SecChUa             string
	ProfileType         string
	GenerateReport      bool
	// TLS-API specific fields
	TLSAPIBrowser string // Browser profile for TLS-API (e.g., "chrome_133")
	Proxy         string // Proxy URL for TLS-API requests
}

var countryLanguageJSON = `{"ae":"ar-AE","ar":"es-AR","at":"de-AT","au":"en-AU","be":"nl-BE","bg":"bg-BG","bh":"ar-BH","bo":"es-BO","br":"pt-BR","by":"ru-BY","ca":"en-CA","ch":"de-CH","cl":"es-CL","cn":"zh-CN","co":"es-CO","cr":"es-CR","cz":"cs-CZ","de":"de-DE","dk":"da-DK","do":"es-DO","dz":"ar-DZ","ec":"es-EC","eg":"ar-EG","es":"es-ES","fi":"fi-FI","fr":"fr-FR","gb":"en-GB","gr":"el-GR","gt":"es-GT","hk":"zh-HK","hn":"es-HN","hu":"hu-HU","id":"id-ID","ie":"en-IE","il":"he-IL","in":"en-IN","iq":"ar-IQ","is":"is-IS","it":"it-IT","jo":"ar-JO","jp":"ja-JP","kr":"ko-KR","kw":"ar-KW","lb":"ar-LB","lu":"fr-LU","ma":"ar-MA","mx":"es-MX","my":"ms-MY","ni":"es-NI","nl":"nl-NL","no":"nb-NO","nz":"en-NZ","om":"ar-OM","pa":"es-PA","pe":"es-PE","ph":"en-PH","pl":"pl-PL","pr":"es-PR","pt":"pt-PT","py":"es-PY","qa":"ar-QA","ro":"ro-RO","ru":"ru-RU","sa":"ar-SA","se":"sv-SE","sg":"en-SG","sk":"sk-SK","sv":"es-SV","sy":"ar-SY","th":"th-TH","tn":"ar-TN","tr":"tr-TR","tw":"zh-TW","ua":"uk-UA","us":"en-US","uy":"es-UY","ve":"es-VE","vn":"vi-VN","za":"en-ZA"}`

var countryToLanguage = map[string]string{
	"ae": "ar-AE", "ar": "es-AR", "at": "de-AT", "au": "en-AU", "be": "nl-BE",
	"bg": "bg-BG", "bh": "ar-BH", "bo": "es-BO", "br": "pt-BR", "by": "ru-BY",
	"ca": "en-CA", "ch": "de-CH", "cl": "es-CL", "cn": "zh-CN", "co": "es-CO",
	"cr": "es-CR", "cz": "cs-CZ", "de": "de-DE", "dk": "da-DK", "do": "es-DO",
	"dz": "ar-DZ", "ec": "es-EC", "eg": "ar-EG", "es": "es-ES", "fi": "fi-FI",
	"fr": "fr-FR", "gb": "en-GB", "gr": "el-GR", "gt": "es-GT", "hk": "zh-HK",
	"hn": "es-HN", "hu": "hu-HU", "id": "id-ID", "ie": "en-IE", "il": "he-IL",
	"in": "en-IN", "iq": "ar-IQ", "is": "is-IS", "it": "it-IT", "jo": "ar-JO",
	"jp": "ja-JP", "kr": "ko-KR", "kw": "ar-KW", "lb": "ar-LB", "lu": "fr-LU",
	"ma": "ar-MA", "mx": "es-MX", "my": "ms-MY", "ni": "es-NI", "nl": "nl-NL",
	"no": "nb-NO", "nz": "en-NZ", "om": "ar-OM", "pa": "es-PA", "pe": "es-PE",
	"ph": "en-PH", "pl": "pl-PL", "pr": "es-PR", "pt": "pt-PT", "py": "es-PY",
	"qa": "ar-QA", "ro": "ro-RO", "ru": "ru-RU", "sa": "ar-SA", "se": "sv-SE",
	"sg": "en-SG", "sk": "sk-SK", "sv": "es-SV", "sy": "ar-SY", "th": "th-TH",
	"tn": "ar-TN", "tr": "tr-TR", "tw": "zh-TW", "ua": "uk-UA", "us": "en-US",
	"uy": "es-UY", "ve": "es-VE", "vn": "vi-VN", "za": "en-ZA",
}

var proxyCountryRe = regexp.MustCompile(`(?i)(?:^|-)country-([a-z]{2})(?:-|$)`)

func languageFromProxy(proxyURL string) (string, bool) {
	if proxyURL == "" {
		return "", false
	}
	u, err := url.Parse(proxyURL)
	if err != nil {
		return "", false
	}
	if u.User == nil {
		return "", false
	}
	username := u.User.Username()
	m := proxyCountryRe.FindStringSubmatch(username)
	if len(m) < 2 {
		return "", false
	}
	cc := strings.ToLower(m[1])
	lang, ok := countryToLanguage[cc]
	return lang, ok
}

type Scraper struct {
	simpleClient   *http.Client // Simple HTTP client for provider API calls (n4s, jevi, roolink)
	userAgent      UserAgent
	config         *Config
	SensorDataOnly string
	report         *requestReport
	providerCache  *ProviderCache
	// TLS-API components
	tlsAPIClient *TLSAPIClient
	cookieJar    *CookieJar
	siteClient   *SiteClient
	abckSolver   *ABCKSolver
	sbsdSolver   *SBSDSolver
	browser      string
	proxy        string
}

func (s *Scraper) HasCachedProviderDynamic() bool {
	if s == nil || s.config == nil {
		return false
	}
	if s.config.ForceUpdateDynamics {
		return false
	}
	if s.config.SbSd {
		return false
	}
	entry, ok := s.cacheGet()
	return ok && entry.Dynamic != ""
}

func (s *Scraper) providerKeyMode() (provider string, mode string) {
	provider = ""
	mode = "sensor"
	if s == nil || s.config == nil {
		return provider, mode
	}
	if s.config.SbSd {
		mode = "sbsd"
		provider = s.config.SbSdProvider
		if provider == "" {
			provider = s.config.AkamaiProvider
		}
		return provider, mode
	}
	provider = s.config.AkamaiProvider
	return provider, mode
}

func (s *Scraper) cacheGet() (providerCacheEntry, bool) {
	if s == nil || s.providerCache == nil || s.config == nil {
		return providerCacheEntry{}, false
	}
	provider, mode := s.providerKeyMode()
	if provider == "" {
		return providerCacheEntry{}, false
	}
	return s.providerCache.Get(s.config.Domain, provider, mode)
}

func (s *Scraper) cacheUpsert(scriptURL *string, dynamic *string) {
	if s == nil || s.providerCache == nil || s.config == nil {
		return
	}
	provider, mode := s.providerKeyMode()
	if provider == "" {
		return
	}
	s.providerCache.Upsert(s.config.Domain, provider, mode, scriptURL, dynamic)
}

func (s *Scraper) ReportPath() string {
	if s == nil || s.report == nil {
		return ""
	}
	return s.report.Path()
}

func (s *Scraper) CloseReport() {
	if s == nil || s.report == nil {
		return
	}
	_ = s.report.Close()
}

type requestReport struct {
	mu   sync.Mutex
	f    *os.File
	path string
}

func newRequestReport() (*requestReport, error) {
	name := fmt.Sprintf("/tmp/getsensor-report-%d.txt", time.Now().UnixNano())
	f, err := os.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return nil, err
	}
	return &requestReport{f: f, path: name}, nil
}

func (rr *requestReport) Path() string {
	return rr.path
}

func (rr *requestReport) Close() error {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	if rr.f == nil {
		return nil
	}
	err := rr.f.Close()
	rr.f = nil
	return err
}

func (rr *requestReport) WriteBlock(block string) {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	if rr.f == nil {
		return
	}
	_, _ = rr.f.WriteString(block)
}

type UserAgent struct {
	Full      string
	SecChUA   string
	Platform  string
	ChromeVer string
}

var defaultUserAgent = UserAgent{
	Full:      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36",
	SecChUA:   "\"Google Chrome\";v=\"143\", \"Chromium\";v=\"143\", \"Not A(Brand\";v=\"24\"",
	Platform:  "Windows",
	ChromeVer: "143",
}

// NewScraper creates a new scraper instance using TLS-API for all site requests
// proxyURL is used for TLS-API site requests
// config contains the scraper configuration
func NewScraper(proxyURL string, config *Config) (*Scraper, error) {
	log.Printf("→ Using TLS-API client")

	if config != nil {
		if lang, ok := languageFromProxy(proxyURL); ok {
			prev := config.Language
			config.Language = lang
			log.Printf("→ Proxy country language override: %s -> %s", prev, config.Language)
		}
	}

	// Use custom UserAgent if provided in config, otherwise use default
	ua := defaultUserAgent
	if config != nil && config.UserAgent != "" {
		ua.Full = config.UserAgent
	}
	if config != nil && config.SecChUa != "" {
		ua.SecChUA = config.SecChUa
	}

	// Create a simple HTTP client for provider API calls (no TLS fingerprinting needed)
	debugProxy := os.Getenv("DEBUG_PROXY")
	var simpleTransport *http.Transport
	if debugProxy != "" {
		debugProxyURL, err := url.Parse(debugProxy)
		if err != nil {
			return nil, fmt.Errorf("error parsing DEBUG_PROXY URL: %v", err)
		}
		log.Printf("→ DEBUG_PROXY enabled for simpleClient: %s", debugProxy)
		simpleTransport = &http.Transport{
			Proxy:           http.ProxyURL(debugProxyURL),
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	} else {
		simpleTransport = &http.Transport{
			Proxy: nil,
		}
	}
	simpleClient := &http.Client{
		Timeout:   30 * time.Second,
		Transport: simpleTransport,
	}

	// Determine browser and proxy
	browser := DefaultBrowser
	if config != nil && config.TLSAPIBrowser != "" {
		browser = config.TLSAPIBrowser
	}
	proxy := ""
	if config != nil && config.Proxy != "" {
		proxy = config.Proxy
	} else if proxyURL != "" {
		proxy = proxyURL
	}

	// Initialize TLS-API client and cookie jar
	tlsAPIClient := NewTLSAPIClient()
	cookieJar := NewCookieJar()

	// Load provider cache
	pc, _ := LoadProviderCacheDefault()

	scraper := &Scraper{
		simpleClient:  simpleClient,
		userAgent:     ua,
		config:        config,
		providerCache: pc,
		tlsAPIClient:  tlsAPIClient,
		cookieJar:     cookieJar,
		browser:       browser,
		proxy:         proxy,
	}

	// Initialize report if enabled
	if config != nil && config.GenerateReport {
		rep, err := newRequestReport()
		if err != nil {
			return nil, fmt.Errorf("failed to create report file: %v", err)
		}
		scraper.report = rep
		log.Printf("→ generateReport enabled: writing request report to %s", rep.Path())
	}

	// Initialize site client for making requests to target sites
	scraper.siteClient = NewSiteClient(
		tlsAPIClient,
		cookieJar,
		config,
		ua,
		browser,
		proxy,
	)

	// Initialize ABCK and SBSD solvers
	scraper.abckSolver = NewABCKSolver(
		config,
		tlsAPIClient,
		cookieJar,
		pc,
		ua.Full,
		browser,
		proxy,
	)
	scraper.sbsdSolver = NewSBSDSolver(
		config,
		tlsAPIClient,
		cookieJar,
		pc,
		ua.Full,
		browser,
		proxy,
	)

	return scraper, nil
}

// GetHomepage fetches the homepage via TLS-API
func (s *Scraper) GetHomepage() (*SiteResponse, error) {
	return s.siteClient.GetHomepage("")
}

// SeedAbckScriptCookies seeds cookies by making a minimal request
func (s *Scraper) SeedAbckScriptCookies() error {
	if s.config == nil {
		return fmt.Errorf("config is nil")
	}
	if s.config.SbSd {
		return nil
	}
	urlStr := fmt.Sprintf("https://%s%s", s.config.Domain, s.config.SensorUrl)
	return s.siteClient.SeedCookies(urlStr)
}

// GetAntiBotScript fetches and returns the base64-encoded anti-bot script
func (s *Scraper) GetAntiBotScript() (string, error) {
	urlStr := fmt.Sprintf("https://%s%s", s.config.Domain, s.config.SensorUrl)

	if s.config != nil && !s.config.SbSd && s.HasCachedProviderDynamic() {
		log.Printf("→ Using cached provider dynamic; _abck script fetch reduced to cookie seed only")
		if err := s.SeedAbckScriptCookies(); err != nil {
			return "", err
		}
		return "", nil
	}

	// Fetch the script via TLS-API
	resp, err := s.siteClient.GetScript(urlStr)
	if err != nil {
		return "", fmt.Errorf("error fetching script: %v", err)
	}

	log.Printf("→ Script downloaded: status=%d size=%d", resp.Status, len(resp.Body))

	encoded := base64.StdEncoding.EncodeToString(resp.Body)
	return encoded, nil
}

// GetAntiBotScriptURL fetches the homepage and extracts the anti-bot script URL
func (s *Scraper) GetAntiBotScriptURL(providedUrl string) (string, error) {
	log.Printf("→ Getting home page via TLS-API")

	resp, err := s.siteClient.GetHomepage(providedUrl)
	if err != nil {
		return "", fmt.Errorf("failed to fetch homepage: %v", err)
	}

	log.Printf("→ Home page fetched: status=%d", resp.Status)

	if resp.Status < 200 || resp.Status > 299 {
		return "", fmt.Errorf("homepage blocked: status=%d body_preview=%s", resp.Status, string(resp.Body[:min(len(resp.Body), 2048)]))
	}

	// Check cache for existing script URL
	if s.config != nil && !s.config.SbSd {
		if entry, ok := s.cacheGet(); ok && entry.ScriptURL != "" {
			log.Printf("→ Found cached sensor URL: %s", entry.ScriptURL)
			return entry.ScriptURL, nil
		}
	}

	bodyString := string(resp.Body)

	// Parse the response body with goquery
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(bodyString))
	if err != nil {
		return "", fmt.Errorf("failed to parse response body with goquery: %v", err)
	}

	akamaiUrl := ""
	var candidateUrls []string

	doc.Find("script").Each(func(index int, item *goquery.Selection) {
		src, exists := item.Attr("src")
		_, existsDefer := item.Attr("defer")

		if exists {
			parts := strings.Split(src, "/")

			if s.config.SbSd && strings.Contains(src, "?v=") && len(parts) > 3 {
				baseUrl := src
				if strings.Contains(src, "?") {
					baseUrl = strings.Split(src, "?")[0]
				}
				hasExtension := strings.Contains(baseUrl, ".js") || strings.Contains(baseUrl, ".css")
				if !hasExtension {
					candidateUrls = append(candidateUrls, src)
				}
			} else if !s.config.SbSd && !existsDefer && !strings.Contains(src, "?v=") && len(parts) > 3 {
				hasExtension := strings.Contains(src, ".js") || strings.Contains(src, ".css")
				if !hasExtension {
					candidateUrls = append(candidateUrls, src)
				}
			}
		}
	})

	// Select the last matching URL (the dynamic one)
	if len(candidateUrls) > 0 {
		akamaiUrl = candidateUrls[len(candidateUrls)-1]
	}

	if s.config.SbSd {
		log.Printf("→ Found SbSd URL: %s", akamaiUrl)
	} else {
		log.Printf("→ Found sensor URL: %s", akamaiUrl)
	}

	if akamaiUrl != "" && s.config != nil && !s.config.SbSd {
		s.cacheUpsert(&akamaiUrl, nil)
	}

	return akamaiUrl, nil
}

// GenerateSession generates the _abck cookie (legacy API, uses TLS-API internally)
// Deprecated: Use GenerateABCK() instead for full result details
func (s *Scraper) GenerateSession(script string) (bool, error) {
	result, err := s.GenerateABCK(script)
	if err != nil {
		return false, err
	}
	return result.Success, nil
}

// GenerateSbSdChallenge generates AND posts the SBSD challenge (legacy API)
// Deprecated: Use GenerateSBSD() instead for full result details
func (s *Scraper) GenerateSbSdChallenge(script string, bmSo string) (string, error) {
	result, err := s.GenerateSBSD(script, bmSo)
	if err != nil {
		return "", err
	}
	return result.CookieString, nil
}

// PostSbSdChallenge is a no-op (legacy API)
// Deprecated: SBSD posting is now integrated into GenerateSbSdChallenge/GenerateSBSD
func (s *Scraper) PostSbSdChallenge(data string) error {
	log.Printf("→ PostSbSdChallenge is deprecated; challenge already posted by GenerateSbSdChallenge")
	return nil
}

// ============================================================================
// TLS-API Based Methods
// ============================================================================

// GenerateABCK generates the _abck cookie using TLS-API service
func (s *Scraper) GenerateABCK(script string) (*ABCKResult, error) {
	if s.abckSolver == nil {
		return nil, NewError(PhaseInit, "abck solver not initialized", nil)
	}
	return s.abckSolver.Solve(script)
}

// GenerateSBSD generates the SBSD challenge and posts it using TLS-API service
func (s *Scraper) GenerateSBSD(script string, bmSo string) (*SBSDResult, error) {
	if s.sbsdSolver == nil {
		return nil, NewError(PhaseInit, "sbsd solver not initialized", nil)
	}
	return s.sbsdSolver.Solve(script, bmSo)
}

// GetCookies returns cookies from the cookie jar for the configured domain
func (s *Scraper) GetCookies() []*http.Cookie {
	if s.cookieJar == nil || s.config == nil {
		return nil
	}
	return s.cookieJar.GetCookies(s.config.Domain)
}

// GetCookieString returns cookies as a string from the cookie jar
func (s *Scraper) GetCookieString(urlStr string) string {
	if s.cookieJar == nil || s.config == nil {
		return ""
	}
	return s.cookieJar.GetCookieString(s.config.Domain)
}

// SetCookies sets cookies in the cookie jar
func (s *Scraper) SetCookies(urlStr string, cookies []*http.Cookie) error {
	if s.cookieJar == nil || s.config == nil {
		return fmt.Errorf("cookie jar not initialized")
	}
	s.cookieJar.SetCookies(s.config.Domain, cookies)
	return nil
}

// SetUserAgent updates the user agent
func (s *Scraper) SetUserAgent(ua UserAgent) {
	s.userAgent = ua
}

// ============================================================================
// Helper Functions
// ============================================================================

// doSimpleRequest uses regular HTTP client for provider API calls (no TLS fingerprinting)
func (s *Scraper) doSimpleRequest(req *http.Request) (*http.Response, error) {
	if s.report != nil {
		block := s.formatStdRequestBlock(req)
		s.report.WriteBlock(block)
	}

	resp, err := s.simpleClient.Do(req)
	if err != nil {
		if s.report != nil {
			s.report.WriteBlock(fmt.Sprintf("--- RESPONSE ERROR ---\n%v\n\n", err))
		}
		return nil, err
	}

	return resp, nil
}

func (s *Scraper) formatStdRequestBlock(req *http.Request) string {
	var b strings.Builder
	b.WriteString("--- REQUEST (provider/http) ---\n")
	b.WriteString(fmt.Sprintf("%s %s\n", req.Method, req.URL.String()))
	b.WriteString("\n")

	for k, vv := range req.Header {
		for _, v := range vv {
			b.WriteString(fmt.Sprintf("%s: %s\n", k, v))
		}
	}
	b.WriteString("\n")
	return b.String()
}

// ReadBody reads and decompresses response body
func ReadBody(body []byte, contentEncoding string) ([]byte, error) {
	if len(body) == 0 {
		return body, nil
	}

	switch contentEncoding {
	case "gzip":
		reader, err := gzip.NewReader(bytes.NewReader(body))
		if err != nil {
			return body, nil // Already decompressed
		}
		defer reader.Close()
		return io.ReadAll(reader)

	case "br":
		reader := brotli.NewReader(bytes.NewReader(body))
		return io.ReadAll(reader)

	default:
		return body, nil
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
