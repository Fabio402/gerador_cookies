package scraper

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/andybalholm/brotli"
)

// SiteClient handles HTTP requests to target sites via TLS-API
// Preserves headers, headerOrder, and browser profile for proper TLS fingerprinting
type SiteClient struct {
	tlsClient   *TLSAPIClient
	cookieJar   *CookieJar
	config      *Config
	userAgent   UserAgent
	browser     string
	proxy       string
}

// NewSiteClient creates a new site client for making requests to target sites
func NewSiteClient(
	tlsClient *TLSAPIClient,
	cookieJar *CookieJar,
	config *Config,
	userAgent UserAgent,
	browser string,
	proxy string,
) *SiteClient {
	if browser == "" {
		browser = DefaultBrowser
	}
	return &SiteClient{
		tlsClient: tlsClient,
		cookieJar: cookieJar,
		config:    config,
		userAgent: userAgent,
		browser:   browser,
		proxy:     proxy,
	}
}

// SiteResponse wraps the response from a site request
type SiteResponse struct {
	Status     int
	StatusText string
	Headers    map[string]string
	Body       []byte
	Cookies    []Cookie
}

// Request makes a request to a target site via TLS-API
func (c *SiteClient) Request(method, url string, body string, customHeaders map[string]string, customHeadersOrder []string) (*SiteResponse, error) {
	headers, headersOrder := c.buildHeaders(customHeaders, customHeadersOrder)

	req := TLSRequest{
		URL:           url,
		Method:        method,
		Browser:       c.browser,
		Headers:       headers,
		HeadersOrder:  headersOrder,
		Body:          body,
		Cookies:       c.cookieJar.ToTLSAPICookies(c.config.Domain),
		Proxy:         c.proxy,
		ReturnCookies: true,
	}

	resp, err := c.tlsClient.Request(req)
	if err != nil {
		return nil, err
	}

	// Store cookies from response
	if resp.GetCookies() != nil {
		c.cookieJar.FromTLSAPICookies(c.config.Domain, resp.GetCookies())
	}

	// Decompress body if needed
	bodyBytes := c.decompressBody([]byte(resp.GetBody()), resp.GetHeaders())

	return &SiteResponse{
		Status:     resp.GetStatus(),
		StatusText: fmt.Sprintf("%d", resp.GetStatus()),
		Headers:    resp.GetHeaders(),
		Body:       bodyBytes,
		Cookies:    resp.GetCookies(),
	}, nil
}

// GetHomepage fetches the homepage of the target site
func (c *SiteClient) GetHomepage(customURL string) (*SiteResponse, error) {
	var homeURL string
	if customURL != "" {
		homeURL = customURL
	} else if c.config.Domain == "www.voeazul.com.br" {
		homeURL = "https://www.voeazul.com.br/br/pt/home"
	} else {
		homeURL = fmt.Sprintf("https://%s", c.config.Domain)
	}

	log.Printf("→ Fetching homepage via TLS-API: %s", homeURL)

	headers, headersOrder := c.buildHomepageHeaders()
	return c.Request("GET", homeURL, "", headers, headersOrder)
}

// GetScript fetches the anti-bot script from the target site
func (c *SiteClient) GetScript(scriptURL string) (*SiteResponse, error) {
	log.Printf("→ Fetching script via TLS-API: %s", scriptURL)

	headers := map[string]string{
		"Accept":          "*/*",
		"Accept-Encoding": "gzip, deflate, br",
		"Accept-Language": c.config.Language,
		"Cache-Control":   "no-cache",
		"Referer":         fmt.Sprintf("https://%s/", c.config.Domain),
		"User-Agent":      c.userAgent.Full,
	}

	headersOrder := []string{
		"Accept",
		"Accept-Encoding",
		"Accept-Language",
		"Cache-Control",
		"Referer",
		"User-Agent",
	}

	return c.Request("GET", scriptURL, "", headers, headersOrder)
}

// SeedCookies makes a minimal request to seed cookies
func (c *SiteClient) SeedCookies(url string) error {
	log.Printf("→ Seeding cookies via TLS-API: %s", url)

	headers := map[string]string{
		"Accept":          "*/*",
		"Accept-Encoding": "identity",
		"Accept-Language": c.config.Language,
		"Range":           "bytes=0-1023",
		"Referer":         fmt.Sprintf("https://%s/", c.config.Domain),
		"User-Agent":      c.userAgent.Full,
	}

	headersOrder := []string{
		"Accept",
		"Accept-Encoding",
		"Accept-Language",
		"Range",
		"Referer",
		"User-Agent",
	}

	_, err := c.Request("GET", url, "", headers, headersOrder)
	return err
}

// buildHeaders creates the headers map and order based on profile type
func (c *SiteClient) buildHeaders(custom map[string]string, customOrder []string) (map[string]string, []string) {
	if custom != nil && len(custom) > 0 {
		return custom, customOrder
	}

	// Default sensor POST headers based on profile type
	switch c.config.ProfileType {
	case "safari_ios_18_5":
		return c.buildSafariHeaders()
	case "firefox_135":
		return c.buildFirefoxHeaders()
	default:
		return c.buildChromeHeaders()
	}
}

// buildHomepageHeaders creates headers for homepage requests based on profile type
func (c *SiteClient) buildHomepageHeaders() (map[string]string, []string) {
	switch c.config.ProfileType {
	case "safari_ios_18_5":
		headers := map[string]string{
			"sec-fetch-dest":  "document",
			"user-agent":      "Mozilla/5.0 (iPhone; CPU iPhone OS 18_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.5 Mobile/15E148 Safari/604.1",
			"accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
			"sec-fetch-site":  "none",
			"sec-fetch-mode":  "navigate",
			"accept-language": "en-CA,en-US;q=0.9,en;q=0.8",
			"priority":        "u=0, i",
			"accept-encoding": "gzip, deflate, br",
		}
		order := []string{
			"sec-fetch-dest",
			"user-agent",
			"accept",
			"sec-fetch-site",
			"sec-fetch-mode",
			"accept-language",
			"priority",
			"accept-encoding",
		}
		return headers, order

	case "firefox_135":
		headers := map[string]string{
			"user-agent":                "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:145.0) Gecko/20100101 Firefox/145.0",
			"accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
			"accept-language":           "en-US,en;q=0.5",
			"accept-encoding":           "gzip, deflate, br, zstd",
			"upgrade-insecure-requests": "1",
			"sec-fetch-dest":            "document",
			"sec-fetch-mode":            "navigate",
			"sec-fetch-site":            "none",
			"sec-fetch-user":            "?1",
			"priority":                  "u=0, i",
			"te":                        "trailers",
		}
		order := []string{
			"user-agent",
			"accept",
			"accept-language",
			"accept-encoding",
			"upgrade-insecure-requests",
			"sec-fetch-dest",
			"sec-fetch-mode",
			"sec-fetch-site",
			"sec-fetch-user",
			"priority",
			"te",
		}
		return headers, order

	default:
		// Chrome default
		headers := map[string]string{
			"sec-ch-ua":                 `"Google Chrome";v="143", "Chromium";v="143", "Not A(Brand";v="24"`,
			"sec-ch-ua-mobile":          "?0",
			"sec-ch-ua-platform":        `"Windows"`,
			"upgrade-insecure-requests": "1",
			"user-agent":                "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36",
			"accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7",
			"sec-fetch-site":            "none",
			"sec-fetch-mode":            "navigate",
			"sec-fetch-user":            "?1",
			"sec-fetch-dest":            "document",
			"accept-encoding":           "gzip, deflate, br, zstd",
			"accept-language":           "en-US,en;q=0.9",
			"priority":                  "u=0, i",
		}
		order := []string{
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
		}
		return headers, order
	}
}

// buildSafariHeaders creates Safari iOS headers for sensor POST
func (c *SiteClient) buildSafariHeaders() (map[string]string, []string) {
	safariUA := "Mozilla/5.0 (iPhone; CPU iPhone OS 18_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.5 Mobile/15E148 Safari/604.1"
	headers := map[string]string{
		"accept":          "*/*",
		"content-type":    "text/plain;charset=UTF-8",
		"sec-fetch-site":  "same-origin",
		"origin":          fmt.Sprintf("https://%s", c.config.Domain),
		"sec-fetch-mode":  "cors",
		"user-agent":      safariUA,
		"referer":         fmt.Sprintf("https://%s/", c.config.Domain),
		"sec-fetch-dest":  "empty",
		"accept-language": "en-CA,en-US;q=0.9,en;q=0.8",
		"priority":        "u=3, i",
		"accept-encoding": "gzip, deflate, br",
	}
	order := []string{
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
	}
	return headers, order
}

// buildFirefoxHeaders creates Firefox headers for sensor POST
func (c *SiteClient) buildFirefoxHeaders() (map[string]string, []string) {
	firefoxUA := "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:145.0) Gecko/20100101 Firefox/145.0"
	headers := map[string]string{
		"user-agent":      firefoxUA,
		"content-type":    "text/plain;charset=UTF-8",
		"accept":          "*/*",
		"accept-language": "en-US,en;q=0.5",
		"accept-encoding": "gzip, deflate, br, zstd",
		"origin":          fmt.Sprintf("https://%s", c.config.Domain),
		"sec-fetch-dest":  "empty",
		"sec-fetch-mode":  "cors",
		"sec-fetch-site":  "same-origin",
		"referer":         fmt.Sprintf("https://%s/", c.config.Domain),
		"priority":        "u=4",
		"te":              "trailers",
	}
	order := []string{
		"user-agent",
		"content-type",
		"accept",
		"accept-language",
		"accept-encoding",
		"origin",
		"sec-fetch-dest",
		"sec-fetch-mode",
		"sec-fetch-site",
		"referer",
		"priority",
		"te",
	}
	return headers, order
}

// buildChromeHeaders creates Chrome headers for sensor POST
func (c *SiteClient) buildChromeHeaders() (map[string]string, []string) {
	headers := map[string]string{
		"sec-ch-ua-platform": `"` + c.userAgent.Platform + `"`,
		"user-agent":         c.userAgent.Full,
		"sec-ch-ua":          c.userAgent.SecChUA,
		"content-type":       "text/plain;charset=UTF-8",
		"sec-ch-ua-mobile":   "?0",
		"accept":             "*/*",
		"origin":             fmt.Sprintf("https://%s", c.config.Domain),
		"sec-fetch-site":     "same-origin",
		"sec-fetch-mode":     "cors",
		"sec-fetch-dest":     "empty",
		"referer":            fmt.Sprintf("https://%s/", c.config.Domain),
		"accept-encoding":    "gzip, deflate, br, zstd",
		"accept-language":    "pt-BR,pt;q=0.9,en-US;q=0.8,en;q=0.7",
		"priority":           "u=1, i",
	}
	order := []string{
		"content-length",
		"sec-ch-ua-platform",
		"user-agent",
		"sec-ch-ua",
		"content-type",
		"sec-ch-ua-mobile",
		"accept",
		"origin",
		"sec-fetch-site",
		"sec-fetch-mode",
		"sec-fetch-dest",
		"referer",
		"accept-encoding",
		"accept-language",
		"cookie",
		"priority",
	}
	return headers, order
}

// decompressBody decompresses the response body based on Content-Encoding
func (c *SiteClient) decompressBody(body []byte, headers map[string]string) []byte {
	if len(body) == 0 {
		return body
	}

	encoding := ""
	if headers != nil {
		encoding = headers["Content-Encoding"]
		if encoding == "" {
			encoding = headers["content-encoding"]
		}
	}

	switch encoding {
	case "gzip":
		reader, err := gzip.NewReader(bytes.NewReader(body))
		if err != nil {
			return body // Already decompressed or invalid
		}
		defer reader.Close()
		decompressed, err := io.ReadAll(reader)
		if err != nil {
			return body
		}
		return decompressed

	case "br":
		reader := brotli.NewReader(bytes.NewReader(body))
		decompressed, err := io.ReadAll(reader)
		if err != nil {
			return body
		}
		return decompressed

	default:
		return body
	}
}

// GetCookieString returns cookies as a string for the configured domain
func (c *SiteClient) GetCookieString() string {
	return c.cookieJar.GetCookieString(c.config.Domain)
}

// GetCookies returns cookies for the configured domain
func (c *SiteClient) GetCookies() []*http.Cookie {
	return c.cookieJar.GetCookies(c.config.Domain)
}
