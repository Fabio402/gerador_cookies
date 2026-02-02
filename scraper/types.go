package scraper

import (
	"net/http"
	"time"
)

// NOTE: Config is defined in scraper.go and will be consolidated during Phase 6
// New fields to be added to Config in Phase 6:
// - TLSAPIBrowser string // Browser profile: "chrome_133" (default), "firefox_135", "safari_ios_18_5"
// - Proxy string (already exists)

// DefaultBrowser is the default TLS profile when none specified
const DefaultBrowser = "chrome_133"

// GetBrowser returns the configured browser or default from Config
// This will be a method on Config after Phase 6 refactoring
func GetBrowser(browser string) string {
	if browser == "" {
		return DefaultBrowser
	}
	return browser
}

// GetEffectiveSbSdProvider returns SbSdProvider or falls back to AkamaiProvider
func GetEffectiveSbSdProvider(sbsdProvider, akamaiProvider string) string {
	if sbsdProvider != "" {
		return sbsdProvider
	}
	return akamaiProvider
}

// ABCKResult holds the result of an ABCK cookie generation
type ABCKResult struct {
	Success      bool           // Whether generation succeeded
	Cookies      []*http.Cookie // Generated cookies
	CookieString string         // Formatted cookie string for direct use
	Session      SessionInfo    // Information about the session used
	Error        *SolverError   // Error details if Success is false
}

// SBSDResult holds the result of an SBSD challenge generation
type SBSDResult struct {
	Success      bool           // Whether generation succeeded
	Cookies      []*http.Cookie // Generated cookies
	CookieString string         // Formatted cookie string for direct use
	Session      SessionInfo    // Information about the session used
	Error        *SolverError   // Error details if Success is false
}

// SessionInfo contains metadata about the generation session
type SessionInfo struct {
	Proxy       string    // Proxy used for generation
	UserAgent   string    // User-Agent used
	Browser     string    // TLS browser profile used
	Domain      string    // Target domain
	Provider    string    // Provider used (jevi, n4s, roolink)
	GeneratedAt time.Time // Timestamp of generation
}

// Cookie represents a cookie in TLS-API format
type Cookie struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Domain   string `json:"domain,omitempty"`
	Path     string `json:"path,omitempty"`
	Expires  string `json:"expires,omitempty"`
	Secure   bool   `json:"secure,omitempty"`
	HTTPOnly bool   `json:"httpOnly,omitempty"`
	SameSite string `json:"sameSite,omitempty"`
}

// TLSRequest represents a request to the TLS-API
type TLSRequest struct {
	URL           string            `json:"url"`
	Method        string            `json:"method"`
	Browser       string            `json:"browser"`
	Headers       map[string]string `json:"headers,omitempty"`
	HeadersOrder  []string          `json:"headersOrder,omitempty"`
	Body          string            `json:"body,omitempty"`
	Cookies       []Cookie          `json:"cookies,omitempty"`
	Proxy         string            `json:"proxy,omitempty"`
	Timeout       int               `json:"timeout,omitempty"`
	ReturnCookies bool              `json:"returnCookies"`
}

// TLSResponse represents a response from the TLS-API
type TLSResponse struct {
	Success  bool             `json:"success"`
	Data     *TLSResponseData `json:"data,omitempty"`
	Error    *TLSAPIError     `json:"error,omitempty"`
	Metadata *TLSMetadata     `json:"metadata,omitempty"`
}

// TLSResponseData contains the actual response data
type TLSResponseData struct {
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
	Cookies []Cookie          `json:"cookies,omitempty"`
}

// TLSAPIError represents an error from TLS-API
type TLSAPIError struct {
	Code     string                 `json:"code"`
	Type     string                 `json:"type"`
	Category string                 `json:"category"` // TLS, PROXY, SITE, VALIDATION
	Message  string                 `json:"message"`
	Details  map[string]interface{} `json:"details,omitempty"`
}

// TLSMetadata contains request metadata from TLS-API
type TLSMetadata struct {
	RequestID   string `json:"requestId"`
	Duration    int    `json:"duration"`
	Fingerprint string `json:"fingerprint"`
	Attempts    int    `json:"attempts"`
	Timestamp   string `json:"timestamp"`
}
