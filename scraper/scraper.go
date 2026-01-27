package scraper

import (
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	_http "net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/andybalholm/brotli"
	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
)

type Config struct {
	Domain          string
	SensorUrl       string
	SensorPostLimit int
	Language        string
	LowSecurity     bool
	UseScript       bool
	ForceUpdateDynamics bool
	EncodedData     string
	AkamaiProvider  string
	SbSdProvider    string
	SbSd            bool
	UserAgent       string
	SecChUa         string
	ProfileType     string
	GenerateReport  bool
}

var countryLanguageJSON = `{"ae":"ar-AE","ar":"es-AR","at":"de-AT","au":"en-AU","be":"nl-BE","bg":"bg-BG","bh":"ar-BH","bo":"es-BO","br":"pt-BR","by":"ru-BY","ca":"en-CA","ch":"de-CH","cl":"es-CL","cn":"zh-CN","co":"es-CO","cr":"es-CR","cz":"cs-CZ","de":"de-DE","dk":"da-DK","do":"es-DO","dz":"ar-DZ","ec":"es-EC","eg":"ar-EG","es":"es-ES","fi":"fi-FI","fr":"fr-FR","gb":"en-GB","gr":"el-GR","gt":"es-GT","hk":"zh-HK","hn":"es-HN","hu":"hu-HU","id":"id-ID","ie":"en-IE","il":"he-IL","in":"en-IN","iq":"ar-IQ","is":"is-IS","it":"it-IT","jo":"ar-JO","jp":"ja-JP","kr":"ko-KR","kw":"ar-KW","lb":"ar-LB","lu":"fr-LU","ma":"ar-MA","mx":"es-MX","my":"ms-MY","ni":"es-NI","nl":"nl-NL","no":"nb-NO","nz":"en-NZ","om":"ar-OM","pa":"es-PA","pe":"es-PE","ph":"en-PH","pl":"pl-PL","pr":"es-PR","pt":"pt-PT","py":"es-PY","qa":"ar-QA","ro":"ro-RO","ru":"ru-RU","sa":"ar-SA","se":"sv-SE","sg":"en-SG","sk":"sk-SK","sv":"es-SV","sy":"ar-SY","th":"th-TH","tn":"ar-TN","tr":"tr-TR","tw":"zh-TW","ua":"uk-UA","us":"en-US","uy":"es-UY","ve":"es-VE","vn":"vi-VN","za":"en-ZA"}`

var countryToLanguage = map[string]string{
	"ae": "ar-AE",
	"ar": "es-AR",
	"at": "de-AT",
	"au": "en-AU",
	"be": "nl-BE",
	"bg": "bg-BG",
	"bh": "ar-BH",
	"bo": "es-BO",
	"br": "pt-BR",
	"by": "ru-BY",
	"ca": "en-CA",
	"ch": "de-CH",
	"cl": "es-CL",
	"cn": "zh-CN",
	"co": "es-CO",
	"cr": "es-CR",
	"cz": "cs-CZ",
	"de": "de-DE",
	"dk": "da-DK",
	"do": "es-DO",
	"dz": "ar-DZ",
	"ec": "es-EC",
	"eg": "ar-EG",
	"es": "es-ES",
	"fi": "fi-FI",
	"fr": "fr-FR",
	"gb": "en-GB",
	"gr": "el-GR",
	"gt": "es-GT",
	"hk": "zh-HK",
	"hn": "es-HN",
	"hu": "hu-HU",
	"id": "id-ID",
	"ie": "en-IE",
	"il": "he-IL",
	"in": "en-IN",
	"iq": "ar-IQ",
	"is": "is-IS",
	"it": "it-IT",
	"jo": "ar-JO",
	"jp": "ja-JP",
	"kr": "ko-KR",
	"kw": "ar-KW",
	"lb": "ar-LB",
	"lu": "fr-LU",
	"ma": "ar-MA",
	"mx": "es-MX",
	"my": "ms-MY",
	"ni": "es-NI",
	"nl": "nl-NL",
	"no": "nb-NO",
	"nz": "en-NZ",
	"om": "ar-OM",
	"pa": "es-PA",
	"pe": "es-PE",
	"ph": "en-PH",
	"pl": "pl-PL",
	"pr": "es-PR",
	"pt": "pt-PT",
	"py": "es-PY",
	"qa": "ar-QA",
	"ro": "ro-RO",
	"ru": "ru-RU",
	"sa": "ar-SA",
	"se": "sv-SE",
	"sg": "en-SG",
	"sk": "sk-SK",
	"sv": "es-SV",
	"sy": "ar-SY",
	"th": "th-TH",
	"tn": "ar-TN",
	"tr": "tr-TR",
	"tw": "zh-TW",
	"ua": "uk-UA",
	"us": "en-US",
	"uy": "es-UY",
	"ve": "es-VE",
	"vn": "vi-VN",
	"za": "en-ZA",
}

var proxyCountryRe = regexp.MustCompile(`(?i)(?:^|-)country-([a-z]{2})(?:-|$)`) // e.g. 06lJR...-country-BR-session-...

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
	client         tls_client.HttpClient
	simpleClient   *_http.Client // Simple HTTP client for Akamai API calls
	userAgent      UserAgent
	solver         *AkamaiSolver
	config         *Config
	SensorDataOnly string
	report         *requestReport
	providerCache  *ProviderCache
}

func (s *Scraper) HasCachedProviderDynamic() bool {
	if s == nil || s.config == nil {
		return false
	}
	if s.config.ForceUpdateDynamics {
		return false
	}
	// Only used to skip fetching the normal (abck) script body.
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

// Helper functions to convert between net/http and fhttp cookies
func stdCookieToFhttp(c *_http.Cookie) *http.Cookie {
	return &http.Cookie{
		Name:       c.Name,
		Value:      c.Value,
		Path:       c.Path,
		Domain:     c.Domain,
		Expires:    c.Expires,
		RawExpires: c.RawExpires,
		MaxAge:     c.MaxAge,
		Secure:     c.Secure,
		HttpOnly:   c.HttpOnly,
		SameSite:   http.SameSite(c.SameSite),
		Raw:        c.Raw,
		Unparsed:   c.Unparsed,
	}
}

func NewScraper(proxyURL string, config *Config, profile profiles.ClientProfile) (*Scraper, error) {
	log.Printf("→ Using bogdanfinn TLS client")
	log.Printf("→ DEBUG_PROXY env: %q", os.Getenv("DEBUG_PROXY"))
	jar := tls_client.NewCookieJar()

	if config != nil {
		if lang, ok := languageFromProxy(proxyURL); ok {
			prev := config.Language
			config.Language = lang
			log.Printf("→ Proxy country language override: %s -> %s", prev, config.Language)
		}
	}

	options := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(30),
		tls_client.WithClientProfile(profile),
		tls_client.WithCookieJar(jar),
		tls_client.WithDisableHttp3(),
		tls_client.WithInsecureSkipVerify(),
	}

	// Check for DEBUG_PROXY env var (e.g., http://127.0.0.1:8888 for Charles)
	// DEBUG_PROXY takes precedence over proxyURL for debugging purposes
	debugProxy := os.Getenv("DEBUG_PROXY")
	if debugProxy != "" {
		log.Printf("→ DEBUG_PROXY enabled for tls_client: %s", debugProxy)
		options = append(options, tls_client.WithProxyUrl(debugProxy))
	} else if proxyURL != "" {
		parsedURL, err := url.Parse(proxyURL)
		if err != nil {
			return nil, fmt.Errorf("error parsing proxy URL: %v", err)
		}
		options = append(options, tls_client.WithProxyUrl(parsedURL.String()))
	}

	client, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)
	if err != nil {
		return nil, err
	}

	// Use custom UserAgent if provided in config, otherwise use default
	ua := defaultUserAgent
	if config.UserAgent != "" {
		ua.Full = config.UserAgent
	}
	if config.SecChUa != "" {
		ua.SecChUA = config.SecChUa
	}

	// Create a simple HTTP client for Akamai API calls (no TLS fingerprinting needed)
	// Reuse debugProxy from above for simpleClient
	var simpleTransport *_http.Transport
	if debugProxy != "" {
		debugProxyURL, err := url.Parse(debugProxy)
		if err != nil {
			return nil, fmt.Errorf("error parsing DEBUG_PROXY URL: %v", err)
		}
		log.Printf("→ DEBUG_PROXY enabled for simpleClient: %s", debugProxy)
		simpleTransport = &_http.Transport{
			Proxy:           _http.ProxyURL(debugProxyURL),
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	} else {
		simpleTransport = &_http.Transport{
			Proxy: nil,
		}
	}
	simpleClient := &_http.Client{
		Timeout:   30 * _http.DefaultClient.Timeout,
		Transport: simpleTransport,
	}

	scraper := &Scraper{
		client:       client,
		simpleClient: simpleClient,
		userAgent:    ua,
		config:       config,
	}
	pc, _ := LoadProviderCacheDefault()
	scraper.providerCache = pc
	if config != nil && config.GenerateReport {
		rep, err := newRequestReport()
		if err != nil {
			return nil, fmt.Errorf("failed to create report file: %v", err)
		}
		scraper.report = rep
		log.Printf("→ generateReport enabled: writing request report to %s", rep.Path())
	}
	scraper.solver = NewAkamaiSolver(scraper, "localhost", "")

	return scraper, nil
}

func (s *Scraper) GetHomepage() (*http.Response, error) {
	return s.makeRequest(http.MethodGet, fmt.Sprintf("https://%s", s.config.Domain))
}

func (s *Scraper) SeedAbckScriptCookies() error {
	if s.config == nil {
		return fmt.Errorf("config is nil")
	}
	if s.config.SbSd {
		return nil
	}
	urlStr := fmt.Sprintf("https://%s%s", s.config.Domain, s.config.SensorUrl)
	log.Printf("→ _abck script cookie seed only (proxy): url=%s range=%s", urlStr, "bytes=0-1023")
	seedReq, err := http.NewRequest(http.MethodGet, urlStr, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}
	s.setHeaders(seedReq)
	seedReq.Header.Set("Range", "bytes=0-1023")
	seedReq.Header.Set("Accept-Encoding", "identity")
	seedResp, err := s.doRequest(seedReq)
	if err != nil {
		return fmt.Errorf("error sending cookie-seed request: %v", err)
	}
	defer seedResp.Body.Close()
	log.Printf("→ _abck script cookie seed only response: status=%s", seedResp.Status)
	_, _ = io.Copy(io.Discard, io.LimitReader(seedResp.Body, 1024))
	return nil
}

func (s *Scraper) GetAntiBotScript() (string, error) {
	urlStr := fmt.Sprintf("https://%s%s", s.config.Domain, s.config.SensorUrl)
	if s.config != nil && !s.config.SbSd && s.HasCachedProviderDynamic() {
		log.Printf("→ Using cached provider dynamic; _abck script fetch reduced to cookie seed only")
		if err := s.SeedAbckScriptCookies(); err != nil {
			return "", err
		}
		return "", nil
	}

	var rawBody []byte
	var encoding string
	var err error

	if s.config != nil && s.config.SbSd {
		// SbSd script: fetch without proxy using the simple client.
		req, reqErr := _http.NewRequest(_http.MethodGet, urlStr, nil)
		if reqErr != nil {
			return "", fmt.Errorf("error creating request: %v", reqErr)
		}
		req.Header.Set("User-Agent", s.userAgent.Full)
		req.Header.Set("Accept", "*/*")
		req.Header.Set("Accept-Encoding", "gzip, br")
		req.Header.Set("Cache-Control", "no-cache")
		if cookieStr := s.GetCookieString(urlStr); cookieStr != "" {
			req.Header.Set("Cookie", cookieStr)
		}

		stdResp, stdErr := s.doSimpleRequest(req)
		if stdErr != nil {
			return "", stdErr
		}
		defer stdResp.Body.Close()
		log.Printf("→ _abck script full download response: status=%s", stdResp.Status)
		rawBody, err = io.ReadAll(stdResp.Body)
		if err != nil {
			return "", fmt.Errorf("error reading response body: %v", err)
		}
		encoding = stdResp.Header.Get("Content-Encoding")
	} else {
		// Non-SbSd (_abck) script:
		// 1) Do a minimal proxied request to this URL to seed any IP-bound cookies into the tls-client jar.
		log.Printf("→ _abck script cookie seed (proxy): url=%s range=%s", urlStr, "bytes=0-1023")
		seedReq, reqErr := http.NewRequest(http.MethodGet, urlStr, nil)
		if reqErr != nil {
			return "", fmt.Errorf("error creating request: %v", reqErr)
		}
		s.setHeaders(seedReq)
		seedReq.Header.Set("Range", "bytes=0-1023")
		seedReq.Header.Set("Accept-Encoding", "identity")
		seedResp, seedErr := s.doRequest(seedReq)
		if seedErr != nil {
			return "", fmt.Errorf("error sending cookie-seed request: %v", seedErr)
		}
		log.Printf("→ _abck script cookie seed response: status=%s", seedResp.Status)
		_, _ = io.Copy(io.Discard, io.LimitReader(seedResp.Body, 1024))
		_ = seedResp.Body.Close()

		// 2) Download the full script without proxy using the simple client, but send cookies from the tls-client jar.
		cookieStr := s.GetCookieString(urlStr)
		log.Printf("→ _abck script full download (no proxy): url=%s has_cookie=%t", urlStr, cookieStr != "")
		req, reqErr := _http.NewRequest(_http.MethodGet, urlStr, nil)
		if reqErr != nil {
			return "", fmt.Errorf("error creating request: %v", reqErr)
		}
		req.Header.Set("User-Agent", s.userAgent.Full)
		req.Header.Set("Accept", "*/*")
		req.Header.Set("Accept-Encoding", "gzip, br")
		req.Header.Set("Cache-Control", "no-cache")
		if cookieStr != "" {
			req.Header.Set("Cookie", cookieStr)
		}

		stdResp, stdErr := s.doSimpleRequest(req)
		if stdErr != nil {
			return "", stdErr
		}
		defer stdResp.Body.Close()
		rawBody, err = io.ReadAll(stdResp.Body)
		if err != nil {
			return "", fmt.Errorf("error reading response body: %v", err)
		}
		encoding = stdResp.Header.Get("Content-Encoding")
	}

	var body []byte

	// Try to decompress based on Content-Encoding header
	// Note: tls-client library may auto-decompress, so we try and fallback silently
	switch encoding {
	case "gzip":
		gzipReader, err := gzip.NewReader(bytes.NewReader(rawBody))
		if err != nil {
			// Already decompressed or invalid gzip, use raw body
			body = rawBody
		} else {
			defer gzipReader.Close()
			body, err = io.ReadAll(gzipReader)
			if err != nil {
				// Decompression failed, use raw body
				body = rawBody
			}
		}
	case "br":
		brReader := brotli.NewReader(bytes.NewReader(rawBody))
		body, err = io.ReadAll(brReader)
		if err != nil {
			// Already decompressed or invalid brotli, use raw body
			body = rawBody
		}
	default:
		body = rawBody
	}

	encoded := base64.StdEncoding.EncodeToString(body)

	return encoded, nil
}

func (s *Scraper) GetAntiBotScriptURL(providedUrl string) (string, error) {
	log.Printf("→ Getting home page")
	var homeUrl string
	if len(providedUrl) > 0 {
		homeUrl = providedUrl
	} else {
		if s.config.Domain == "www.voeazul.com.br" {
			homeUrl = "https://www.voeazul.com.br/br/pt/home"
		} else {
			homeUrl = fmt.Sprintf("https://%s", s.config.Domain)
		}
	}


	// Create request with custom headers based on profile type
	req, err := http.NewRequest(http.MethodGet, homeUrl, nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	// Set headers based on profile type
	switch s.config.ProfileType {
	case "safari_ios_18_5":
		req.Header = http.Header{
			"sec-fetch-dest":  {"document"},
			"user-agent":      {"Mozilla/5.0 (iPhone; CPU iPhone OS 18_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.5 Mobile/15E148 Safari/604.1"},
			"accept":          {"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"},
			"sec-fetch-site":  {"none"},
			"sec-fetch-mode":  {"navigate"},
			"accept-language": {"en-CA,en-US;q=0.9,en;q=0.8"},
			"priority":        {"u=0, i"},
			"accept-encoding": {"gzip, deflate, br"},
			http.HeaderOrderKey: {
				"sec-fetch-dest",
				"user-agent",
				"accept",
				"sec-fetch-site",
				"sec-fetch-mode",
				"accept-language",
				"priority",
				"accept-encoding",
			},
		}
	case "firefox_135":
		req.Header = http.Header{
			"user-agent":                {"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:145.0) Gecko/20100101 Firefox/145.0"},
			"accept":                    {"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"},
			"accept-language":           {"en-US,en;q=0.5"},
			"accept-encoding":           {"gzip, deflate, br, zstd"},
			"upgrade-insecure-requests": {"1"},
			"sec-fetch-dest":            {"document"},
			"sec-fetch-mode":            {"navigate"},
			"sec-fetch-site":            {"none"},
			"sec-fetch-user":            {"?1"},
			"priority":                  {"u=0, i"},
			"te":                        {"trailers"},
			http.HeaderOrderKey:         {},
		}
	default:
		// Default Chrome headers
		req.Header = http.Header{
			"sec-ch-ua": {"\"Google Chrome\";v=\"143\", \"Chromium\";v=\"143\", \"Not A(Brand\";v=\"24\""},
			"sec-ch-ua-mobile": {"?0"},
			"sec-ch-ua-platform": {"\"Windows\""},
			"upgrade-insecure-requests": {"1"},
			"user-agent": {"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36"},
			"accept": {"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7"},
			"sec-fetch-site": {"none"},
			"sec-fetch-mode": {"navigate"},
			"sec-fetch-user": {"?1"},
			"sec-fetch-dest": {"document"},
			"accept-encoding": {"gzip, deflate, br, zstd"},
			"accept-language": {"en-US,en;q=0.9"},
			"priority": {"u=0, i"},
			http.HeaderOrderKey: {
				"sec-ch-ua",
				"sec-ch-ua-mobile",
				"sec-ch-ua-platform",
				"upgrade-insecure-requests",
				"user-agent",
				"accept",
				"sec-fetch-site",
				"sec-fetch-mode",
				"sec-fetch-user",
				"sec-fetch-dest",
				"accept-encoding",
				"accept-language",
				"priority",
			},
		}

	}

	response, err := s.doRequest(req)
	if err != nil {
		return "", fmt.Errorf("failed to make GET request to %s: %v", s.config.Domain, err)
	}
	defer response.Body.Close()
	log.Printf("→ Home page fetched: url=%s status=%s", homeUrl, response.Status)
	if response.StatusCode < 200 || response.StatusCode > 299 {
		b, _ := io.ReadAll(io.LimitReader(response.Body, 2048))
		return "", fmt.Errorf("homepage blocked: url=%s status=%s body_preview=%s", homeUrl, response.Status, string(b))
	}

	// Even when the non-SbSd (abck) script URL is cached, we still GET the home page first
	// to seed cookies/session, then we can reuse the cached script URL.
	if s.config != nil && !s.config.SbSd {
		if entry, ok := s.cacheGet(); ok && entry.ScriptURL != "" {
			log.Printf("→ Found sensor URL: %s", entry.ScriptURL)
			return entry.ScriptURL, nil
		}
	}

	bodyBytes, err := ReadBody(response)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}

	bodyString := string(bodyBytes)

	println(bodyString, "!!!!!!!!")

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

func (s *Scraper) GenerateSession(script string) (bool, error) {
	return s.solver.Solve(script)
}

func (s *Scraper) GenerateSbSdChallenge(script string, bmSo string) (string, error) {
	return s.solver.GenerateSbSd(script, bmSo)
}

func (s *Scraper) PostSbSdChallenge(data string) error {
	return s.solver.PostSbSdChallenge(data)
}

// doSimpleRequest uses regular HTTP client for Akamai API calls (no TLS fingerprinting)
func (s *Scraper) doSimpleRequest(req *_http.Request) (*_http.Response, error) {
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

	if s.report != nil {
		bodyBytes, truncNote, rawLen := readAndReplaceBody(&resp.Body, 256*1024)
		block := s.formatStdResponseBlock(resp, bodyBytes, rawLen, truncNote)
		s.report.WriteBlock(block)
	}

	return resp, nil
}

func (s *Scraper) doRequest(req *http.Request) (*http.Response, error) {
	if s.report != nil {
		block := s.formatFhttpRequestBlock(req)
		s.report.WriteBlock(block)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		if s.report != nil {
			s.report.WriteBlock(fmt.Sprintf("--- RESPONSE ERROR ---\n%v\n\n", err))
		}
		return nil, err
	}

	if s.report != nil {
		bodyBytes, truncNote, rawLen := readAndReplaceBody(&resp.Body, 256*1024)
		block := s.formatFhttpResponseBlock(resp, bodyBytes, rawLen, truncNote)
		s.report.WriteBlock(block)
	}

	return resp, nil
}

func (s *Scraper) formatStdRequestBlock(req *_http.Request) string {
	var b strings.Builder
	b.WriteString("--- REQUEST (provider/http) ---\n")
	b.WriteString(fmt.Sprintf("%s %s\n", req.Method, req.URL.String()))
	b.WriteString("\n")

	bodyBytes := []byte(nil)
	if req.Body != nil {
		raw, _ := io.ReadAll(req.Body)
		_ = req.Body.Close()
		req.Body = io.NopCloser(bytes.NewReader(raw))
		bodyBytes = raw
	}

	b.WriteString(s.curlFromStdRequest(req, bodyBytes))
	b.WriteString("\n")
	for k, vv := range req.Header {
		for _, v := range vv {
			b.WriteString(fmt.Sprintf("%s: %s\n", k, v))
		}
	}
	if len(bodyBytes) > 0 {
		b.WriteString("\n")
		b.WriteString(string(bodyBytes))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	return b.String()
}

func (s *Scraper) formatFhttpRequestBlock(req *http.Request) string {
	var b strings.Builder
	b.WriteString("--- REQUEST (site/tls-client) ---\n")
	b.WriteString(fmt.Sprintf("%s %s\n", req.Method, req.URL.String()))
	b.WriteString("\n")

	bodyBytes := []byte(nil)
	if req.Body != nil {
		raw, _ := io.ReadAll(req.Body)
		_ = req.Body.Close()
		req.Body = io.NopCloser(bytes.NewReader(raw))
		bodyBytes = raw
	}

	b.WriteString(s.curlFromFhttpRequest(req, bodyBytes))
	b.WriteString("\n")
	for k, vv := range req.Header {
		for _, v := range vv {
			b.WriteString(fmt.Sprintf("%s: %s\n", k, v))
		}
	}
	if len(bodyBytes) > 0 {
		b.WriteString("\n")
		b.WriteString(string(bodyBytes))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	return b.String()
}

func (s *Scraper) formatFhttpResponseBlock(resp *http.Response, body []byte, rawLen int, truncNote string) string {
	var b strings.Builder
	b.WriteString("--- RESPONSE (site/tls-client) ---\n")
	b.WriteString(fmt.Sprintf("Status: %s\n", resp.Status))
	if cl := resp.Header.Get("Content-Length"); cl != "" {
		b.WriteString(fmt.Sprintf("Content-Length: %s\n", cl))
	}
	b.WriteString(fmt.Sprintf("Content-Length-Parsed: %d\n", resp.ContentLength))
	b.WriteString(fmt.Sprintf("Content-Length-Body-Original: %d\n", rawLen))
	b.WriteString(fmt.Sprintf("Content-Length-Body-Captured: %d\n", len(body)))
	for k, vv := range resp.Header {
		for _, v := range vv {
			b.WriteString(fmt.Sprintf("%s: %s\n", k, v))
		}
	}
	if truncNote != "" {
		b.WriteString(fmt.Sprintf("\n%s\n", truncNote))
	}
	b.WriteString("\n")
	if len(body) > 0 {
		b.Write(body)
		b.WriteString("\n")
	}
	b.WriteString("\n")
	return b.String()
}

func (s *Scraper) formatStdResponseBlock(resp *_http.Response, body []byte, rawLen int, truncNote string) string {
	var b strings.Builder
	b.WriteString("--- RESPONSE (provider/http) ---\n")
	b.WriteString(fmt.Sprintf("Status: %s\n", resp.Status))
	if cl := resp.Header.Get("Content-Length"); cl != "" {
		b.WriteString(fmt.Sprintf("Content-Length: %s\n", cl))
	}
	b.WriteString(fmt.Sprintf("Content-Length-Parsed: %d\n", resp.ContentLength))
	b.WriteString(fmt.Sprintf("Content-Length-Body-Original: %d\n", rawLen))
	b.WriteString(fmt.Sprintf("Content-Length-Body-Captured: %d\n", len(body)))
	for k, vv := range resp.Header {
		for _, v := range vv {
			b.WriteString(fmt.Sprintf("%s: %s\n", k, v))
		}
	}
	if truncNote != "" {
		b.WriteString(fmt.Sprintf("\n%s\n", truncNote))
	}
	b.WriteString("\n")
	if len(body) > 0 {
		b.Write(body)
		b.WriteString("\n")
	}
	b.WriteString("\n")
	return b.String()
}

func readAndReplaceBody(rc *io.ReadCloser, max int) ([]byte, string, int) {
	if rc == nil || *rc == nil {
		return nil, "", 0
	}
	raw, _ := io.ReadAll(*rc)
	_ = (*rc).Close()
	*rc = io.NopCloser(bytes.NewReader(raw))
	if max <= 0 || len(raw) <= max {
		return raw, "", len(raw)
	}
	return raw[:max], fmt.Sprintf("(body truncated to %s bytes; original %s bytes)", strconv.Itoa(max), strconv.Itoa(len(raw))), len(raw)
}

func escapeSingleQuotes(s string) string {
	return strings.ReplaceAll(s, "'", "'\\''")
}

func (s *Scraper) curlFromStdRequest(req *_http.Request, body []byte) string {
	var parts []string
	parts = append(parts, "curl -i")
	parts = append(parts, "-X "+escapeSingleQuotes(req.Method))
	parts = append(parts, "'"+escapeSingleQuotes(req.URL.String())+"'")
	for k, vv := range req.Header {
		for _, v := range vv {
			parts = append(parts, "-H '"+escapeSingleQuotes(k+": "+v)+"'")
		}
	}
	if len(body) > 0 {
		parts = append(parts, "--data-raw '"+escapeSingleQuotes(string(body))+"'")
	}
	return strings.Join(parts, " ") + "\n"
}

func (s *Scraper) curlFromFhttpRequest(req *http.Request, body []byte) string {
	var parts []string
	parts = append(parts, "curl -i")
	parts = append(parts, "-X "+escapeSingleQuotes(req.Method))
	parts = append(parts, "'"+escapeSingleQuotes(req.URL.String())+"'")
	if cookieStr := s.GetCookieString(req.URL.String()); cookieStr != "" {
		parts = append(parts, "-H '"+escapeSingleQuotes("Cookie: "+cookieStr)+"'")
	}
	for k, vv := range req.Header {
		for _, v := range vv {
			parts = append(parts, "-H '"+escapeSingleQuotes(k+": "+v)+"'")
		}
	}
	if len(body) > 0 {
		parts = append(parts, "--data-raw '"+escapeSingleQuotes(string(body))+"'")
	}
	return strings.Join(parts, " ") + "\n"
}

func (s *Scraper) makeRequest(method, urlStr string) (*http.Response, error) {
	req, err := http.NewRequest(method, urlStr, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	s.setHeaders(req)

	resp, err := s.doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}

	return resp, nil
}

func (s *Scraper) setHeaders(req *http.Request) {
	switch s.config.ProfileType {
	case "safari_ios_18_5":
		safariUA := "Mozilla/5.0 (iPhone; CPU iPhone OS 18_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.5 Mobile/15E148 Safari/604.1"
		log.Printf("→ Using Safari iOS 18.5 headers (UA: %s)", safariUA)
		req.Header = http.Header{
			"accept":          {"*/*"},
			"content-type":    {"text/plain;charset=UTF-8"},
			"sec-fetch-site":  {"same-origin"},
			"origin":          {fmt.Sprintf("https://%s", s.config.Domain)},
			"sec-fetch-mode":  {"cors"},
			"user-agent":      {safariUA},
			"referer":         {fmt.Sprintf("https://%s/", s.config.Domain)},
			"sec-fetch-dest":  {"empty"},
			"accept-language": {"en-CA,en-US;q=0.9,en;q=0.8"},
			"priority":        {"u=3, i"},
			"accept-encoding": {"gzip, deflate, br"},
			http.HeaderOrderKey: {
				"accept",
				"content-type",
				"sec-fetch-site",
				"origin",
				"sec-fetch-mode",
				"user-agent",
				"referer",
				"sec-fetch-dest",
				"content-length",
				"accept-language",
				"priority",
				"accept-encoding",
				"cookie",
			},
		}
	case "firefox_135":
		firefoxUA := "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:145.0) Gecko/20100101 Firefox/145.0"
		log.Printf("→ Using Firefox 135 headers (UA: %s)", firefoxUA)
		req.Header = http.Header{
			"user-agent":        {firefoxUA},
			"content-type":      {"text/plain;charset=UTF-8"},
			"accept":            {"*/*"},
			"accept-language":   {"en-US,en;q=0.5"},
			"accept-encoding":   {"gzip, deflate, br, zstd"},
			"origin":            {fmt.Sprintf("https://%s", s.config.Domain)},
			"sec-fetch-dest":    {"empty"},
			"sec-fetch-mode":    {"cors"},
			"sec-fetch-site":    {"same-origin"},
			"referer":           {fmt.Sprintf("https://%s/", s.config.Domain)},
			"priority":          {"u=4"},
			"te":                {"trailers"},
			http.HeaderOrderKey: {},
		}
	default:
		req.Header = http.Header{
			"sec-ch-ua-platform": {`"` + s.userAgent.Platform + `"`},
			"user-agent":         {s.userAgent.Full},
			"sec-ch-ua":          {s.userAgent.SecChUA},
			"content-type":       {"text/plain;charset=UTF-8"},
			"sec-ch-ua-mobile":   {"?0"},
			"accept":             {"*/*"},
			"origin":             {fmt.Sprintf("https://%s", s.config.Domain)},
			"sec-fetch-site":     {"same-origin"},
			"sec-fetch-mode":     {"cors"},
			"sec-fetch-dest":     {"empty"},
			"referer":            {fmt.Sprintf("https://%s/", s.config.Domain)},
			"accept-encoding":    {"gzip, deflate, br, zstd"},
			"accept-language":    {"pt-BR,pt;q=0.9,en-US;q=0.8,en;q=0.7"},
			"priority":           {"u=1, i"},
			http.HeaderOrderKey: {
				"content-length", "sec-ch-ua-platform", "user-agent", "sec-ch-ua",
				"content-type", "sec-ch-ua-mobile", "accept", "origin",
				"sec-fetch-site", "sec-fetch-mode", "sec-fetch-dest", "referer",
				"accept-encoding", "accept-language", "cookie", "priority",
			},
		}
	}
}

func (s *Scraper) SetUserAgent(ua UserAgent) {
	s.userAgent = ua
}

func (s *Scraper) GetCookies(urlStr string) ([]*http.Cookie, error) {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("error parsing URL: %v", err)
	}
	return s.client.GetCookies(parsedURL), nil
}

func (s *Scraper) GetCookieString(urlStr string) string {
	cookies, _ := s.GetCookies(urlStr)

	cookieString := ""
	for i, cookie := range cookies {
		if i > 0 {
			cookieString += "; "
		}
		cookieString += fmt.Sprintf("%s=%s", cookie.Name, cookie.Value)
	}

	return cookieString
}

func (s *Scraper) SetCookies(urlStr string, cookies []*http.Cookie) error {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("error parsing URL: %v", err)
	}
	s.client.SetCookies(parsedURL, cookies)
	return nil
}

// Helper function to read and decompress the response body if needed
func ReadBody(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close() // Ensure the response body is always closed

	var reader io.Reader
	var err error

	// Check for content encoding
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		// Try creating a GZIP reader
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			// Log the raw body preview for debugging
			bodyPreview, _ := io.ReadAll(io.LimitReader(resp.Body, 100)) // Read first 100 bytes for inspection
			log.Printf("Failed to create GZIP reader: %v. Body Preview: %s", err, bodyPreview)

			// Fallback: assume it's not actually gzipped if there's an error
			reader = resp.Body
		} else {
			defer gzipReader.Close()
			reader = gzipReader
		}

	case "br":
		// Brotli decompression
		reader = brotli.NewReader(resp.Body)

	default:
		// Fallback for uncompressed response bodies
		reader = resp.Body
	}

	// Read the entire body
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	return body, nil
}
