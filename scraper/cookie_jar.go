package scraper

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

// CookieJar provides thread-safe cookie storage per domain
type CookieJar struct {
	mu      sync.RWMutex
	cookies map[string][]*http.Cookie // domain -> cookies
}

// NewCookieJar creates a new empty CookieJar
func NewCookieJar() *CookieJar {
	return &CookieJar{
		cookies: make(map[string][]*http.Cookie),
	}
}

// SetCookies replaces all cookies for a domain
func (j *CookieJar) SetCookies(domain string, cookies []*http.Cookie) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.cookies[domain] = cookies
}

// AddCookie adds or updates a single cookie for a domain
// If a cookie with the same name exists, it will be updated
func (j *CookieJar) AddCookie(domain string, cookie *http.Cookie) {
	j.mu.Lock()
	defer j.mu.Unlock()

	existing := j.cookies[domain]
	for i, c := range existing {
		if c.Name == cookie.Name {
			existing[i] = cookie
			return
		}
	}
	j.cookies[domain] = append(existing, cookie)
}

// MergeCookies adds multiple cookies, updating existing ones by name
func (j *CookieJar) MergeCookies(domain string, cookies []*http.Cookie) {
	j.mu.Lock()
	defer j.mu.Unlock()

	existing := j.cookies[domain]
	cookieMap := make(map[string]*http.Cookie)

	// Index existing cookies
	for _, c := range existing {
		cookieMap[c.Name] = c
	}

	// Merge new cookies (overwrite existing)
	for _, c := range cookies {
		cookieMap[c.Name] = c
	}

	// Rebuild slice
	result := make([]*http.Cookie, 0, len(cookieMap))
	for _, c := range cookieMap {
		result = append(result, c)
	}
	j.cookies[domain] = result
}

// GetCookies returns all cookies for a domain
func (j *CookieJar) GetCookies(domain string) []*http.Cookie {
	j.mu.RLock()
	defer j.mu.RUnlock()

	if cookies, ok := j.cookies[domain]; ok {
		// Return a copy to prevent external modification
		result := make([]*http.Cookie, len(cookies))
		copy(result, cookies)
		return result
	}
	return nil
}

// GetCookie returns a specific cookie by name for a domain
func (j *CookieJar) GetCookie(domain, name string) *http.Cookie {
	j.mu.RLock()
	defer j.mu.RUnlock()

	for _, c := range j.cookies[domain] {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// GetCookieString returns cookies formatted as "name=value; name2=value2"
func (j *CookieJar) GetCookieString(domain string) string {
	j.mu.RLock()
	defer j.mu.RUnlock()

	cookies := j.cookies[domain]
	if len(cookies) == 0 {
		return ""
	}

	parts := make([]string, 0, len(cookies))
	for _, c := range cookies {
		parts = append(parts, c.Name+"="+c.Value)
	}
	return strings.Join(parts, "; ")
}

// ToTLSAPICookies converts stored http.Cookies to TLS-API Cookie format
func (j *CookieJar) ToTLSAPICookies(domain string) []Cookie {
	j.mu.RLock()
	defer j.mu.RUnlock()

	httpCookies := j.cookies[domain]
	if len(httpCookies) == 0 {
		return nil
	}

	result := make([]Cookie, 0, len(httpCookies))
	for _, c := range httpCookies {
		result = append(result, httpCookieToTLSAPI(c))
	}
	return result
}

// FromTLSAPICookies converts TLS-API cookies and stores them
func (j *CookieJar) FromTLSAPICookies(domain string, cookies []Cookie) {
	if len(cookies) == 0 {
		return
	}

	httpCookies := make([]*http.Cookie, 0, len(cookies))
	for _, c := range cookies {
		httpCookies = append(httpCookies, tlsAPICookieToHTTP(c))
	}
	j.MergeCookies(domain, httpCookies)
}

// Clear removes all cookies for a domain
func (j *CookieJar) Clear(domain string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	delete(j.cookies, domain)
}

// ClearAll removes all cookies from all domains
func (j *CookieJar) ClearAll() {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.cookies = make(map[string][]*http.Cookie)
}

// httpCookieToTLSAPI converts an http.Cookie to TLS-API Cookie format
func httpCookieToTLSAPI(c *http.Cookie) Cookie {
	cookie := Cookie{
		Name:     c.Name,
		Value:    c.Value,
		Domain:   c.Domain,
		Path:     c.Path,
		Secure:   c.Secure,
		HTTPOnly: c.HttpOnly,
	}

	if !c.Expires.IsZero() {
		cookie.Expires = c.Expires.Format(time.RFC1123)
	}

	switch c.SameSite {
	case http.SameSiteLaxMode:
		cookie.SameSite = "Lax"
	case http.SameSiteStrictMode:
		cookie.SameSite = "Strict"
	case http.SameSiteNoneMode:
		cookie.SameSite = "None"
	}

	return cookie
}

// tlsAPICookieToHTTP converts a TLS-API Cookie to http.Cookie
func tlsAPICookieToHTTP(c Cookie) *http.Cookie {
	cookie := &http.Cookie{
		Name:     c.Name,
		Value:    c.Value,
		Domain:   c.Domain,
		Path:     c.Path,
		Secure:   c.Secure,
		HttpOnly: c.HTTPOnly,
	}

	if c.Expires != "" {
		if t, err := time.Parse(time.RFC1123, c.Expires); err == nil {
			cookie.Expires = t
		}
	}

	switch strings.ToLower(c.SameSite) {
	case "lax":
		cookie.SameSite = http.SameSiteLaxMode
	case "strict":
		cookie.SameSite = http.SameSiteStrictMode
	case "none":
		cookie.SameSite = http.SameSiteNoneMode
	}

	return cookie
}
