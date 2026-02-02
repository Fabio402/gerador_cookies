package scraper

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

// SBSDSolver handles the SBSD challenge generation flow
type SBSDSolver struct {
	tlsClient     *TLSAPIClient
	cookieJar     *CookieJar
	providerCache *ProviderCache
	config        *Config
	userAgent     string
	browser       string
	proxy         string
}

// NewSBSDSolver creates a new SBSD solver
func NewSBSDSolver(
	config *Config,
	tlsClient *TLSAPIClient,
	cookieJar *CookieJar,
	providerCache *ProviderCache,
	userAgent string,
	browser string,
	proxy string,
) *SBSDSolver {
	if browser == "" {
		browser = DefaultBrowser
	}
	return &SBSDSolver{
		tlsClient:     tlsClient,
		cookieJar:     cookieJar,
		providerCache: providerCache,
		config:        config,
		userAgent:     userAgent,
		browser:       browser,
		proxy:         proxy,
	}
}

// Solve executes the complete SBSD challenge flow
func (s *SBSDSolver) Solve(script string, bmSo string) (*SBSDResult, error) {
	provider := GetEffectiveSbSdProvider(s.config.SbSdProvider, s.config.AkamaiProvider)
	log.Printf("→ Starting SBSD solve flow (provider=%s)", provider)

	result := &SBSDResult{
		Session: SessionInfo{
			Proxy:       s.proxy,
			UserAgent:   s.userAgent,
			Browser:     s.browser,
			Domain:      s.config.Domain,
			Provider:    provider,
			GeneratedAt: time.Now(),
		},
	}

	var sbsdBody string
	var err error

	switch provider {
	case "jevi":
		sbsdBody, err = s.generateWithJevi(script, bmSo)
	case "n4s":
		sbsdBody, err = s.generateWithN4S(script, bmSo)
	case "roolink":
		sbsdBody, err = s.generateWithRoolink(bmSo)
	default:
		err = NewError(PhaseInit, "invalid provider", fmt.Errorf("unknown SBSD provider: %s", provider))
	}

	if err != nil {
		result.Success = false
		if solverErr, ok := err.(*SolverError); ok {
			result.Error = solverErr
		} else {
			result.Error = NewErrorWithProvider(PhaseProviderCall, "sbsd generation", provider, err)
		}
		return result, err
	}

	// Post the SBSD challenge to Akamai
	if err := s.postChallenge(sbsdBody); err != nil {
		result.Success = false
		if solverErr, ok := err.(*SolverError); ok {
			result.Error = solverErr
		} else {
			result.Error = NewError(PhaseSBSDPost, "post challenge", err)
		}
		return result, err
	}

	result.Success = true
	result.Cookies = s.cookieJar.GetCookies(s.config.Domain)
	result.CookieString = s.cookieJar.GetCookieString(s.config.Domain)

	log.Printf("✓ SBSD solve succeeded (provider=%s)", provider)
	return result, nil
}

// postChallenge sends the SBSD body to Akamai
func (s *SBSDSolver) postChallenge(sbsdBody string) error {
	log.Printf("→ Posting SBSD challenge to Akamai")

	challengeURL := fmt.Sprintf("https://%s%s", s.config.Domain, s.config.SensorUrl)

	// SBSD uses JSON body format
	payload, _ := json.Marshal(map[string]string{"body": sbsdBody})

	headers := s.buildHeaders()
	headers["Content-Type"] = "application/json"

	resp, err := s.tlsClient.Request(TLSRequest{
		URL:           challengeURL,
		Method:        "POST",
		Browser:       s.browser,
		Headers:       headers,
		HeadersOrder:  s.buildHeadersOrder(),
		Body:          string(payload),
		Cookies:       s.cookieJar.ToTLSAPICookies(s.config.Domain),
		Proxy:         s.proxy,
		ReturnCookies: true,
	})

	if err != nil {
		return NewError(PhaseSBSDPost, "send challenge", err)
	}

	// Validate response: must be 200 or 202
	status := resp.GetStatus()
	if status != 200 && status != 202 {
		return NewErrorWithStatus(PhaseSBSDPost, "unexpected status", status, fmt.Errorf("expected 200 or 202, got %d: %s", status, resp.GetBody()))
	}

	// Store cookies from response
	if resp.GetCookies() != nil {
		s.cookieJar.FromTLSAPICookies(s.config.Domain, resp.GetCookies())
	}

	log.Printf("✓ SBSD challenge accepted (status=%d)", status)
	return nil
}

// ============================================================================
// Jevi Provider
// ============================================================================

type sbsdJeviRequest struct {
	Mode        int             `json:"mode"`
	SbsdRequest sbsdJeviPayload `json:"SbsdRequest"`
}

type sbsdJeviPayload struct {
	NewVersion bool   `json:"NewVersion"`
	ScriptHash string `json:"ScriptHash"`
	Script     string `json:"Script"`
	Site       string `json:"Site"`
	SbsdO      string `json:"sbsd_o"`
	UserAgent  string `json:"userAgent"`
	Uuid       string `json:"uuid"`
}

func (s *SBSDSolver) generateWithJevi(script string, bmSo string) (string, error) {
	log.Printf("→ Generating SBSD with Jevi")

	// Extract uuid from sensor URL (v= parameter)
	uuid := s.extractVidFromSensorURL()
	if uuid == "" {
		return "", fmt.Errorf("could not extract vid from sensor URL")
	}

	// First attempt without script
	body, err := s.callJeviSBSD("", bmSo, uuid)
	if err != nil {
		// Retry with base64-encoded script
		log.Printf("→ Jevi SBSD retry with script")
		base64Script := base64.StdEncoding.EncodeToString([]byte(script))
		body, err = s.callJeviSBSD(base64Script, bmSo, uuid)
		if err != nil {
			return "", err
		}
	}

	return body, nil
}

func (s *SBSDSolver) callJeviSBSD(script string, bmSo string, uuid string) (string, error) {
	req := sbsdJeviRequest{
		Mode: 3,
		SbsdRequest: sbsdJeviPayload{
			NewVersion: true,
			ScriptHash: "",
			Script:     script,
			Site:       fmt.Sprintf("https://%s/", s.config.Domain),
			SbsdO:      bmSo,
			UserAgent:  s.userAgent,
			Uuid:       uuid,
		},
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	compressedData, err := sbsdCompressGzip(jsonData)
	if err != nil {
		return "", fmt.Errorf("compress payload: %w", err)
	}

	apiKey := "curiousT-a23f417f-096e-4258-adea-7ea874a57e56"
	userAgentPrefix := strings.Split(apiKey, "-")[0]

	resp, err := s.tlsClient.Request(TLSRequest{
		URL:     "https://new.jevi.dev/Solver/solve",
		Method:  "POST",
		Browser: s.browser,
		Headers: map[string]string{
			"Content-Type":     "application/json",
			"Content-Encoding": "gzip",
			"User-Agent":       userAgentPrefix,
			"x-key":            apiKey,
		},
		Body:  string(compressedData),
		Proxy: s.proxy,
	})

	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}

	body := resp.GetBody()

	// Parse JSON to extract body field
	var jeviResp struct {
		Body string `json:"body"`
	}
	if json.Unmarshal([]byte(body), &jeviResp) == nil && jeviResp.Body != "" {
		body = jeviResp.Body
	}

	// Check for errors
	if resp.GetStatus() == 400 || strings.Contains(body, "Script hash or script content must be provided") || strings.Contains(body, "Error processing SBSD request") {
		return "", fmt.Errorf("jevi requires script: %s", body)
	}

	if !resp.IsSuccess() {
		return "", fmt.Errorf("jevi returned status %d: %s", resp.GetStatus(), body)
	}

	if strings.Contains(body, "Invalid credentials") {
		return "", fmt.Errorf("invalid API key")
	}

	return body, nil
}

// ============================================================================
// N4S Provider
// ============================================================================

type sbsdN4SRequest struct {
	UserAgent string `json:"user_agent"`
	TargetURL string `json:"targetURL"`
	VUrl      string `json:"v_url"`
	BmSo      string `json:"bm_so"`
	Language  string `json:"language"`
	Script    string `json:"script"`
}

func (s *SBSDSolver) generateWithN4S(script string, bmSo string) (string, error) {
	log.Printf("→ Generating SBSD with N4S")

	req := sbsdN4SRequest{
		UserAgent: s.userAgent,
		TargetURL: fmt.Sprintf("https://%s", s.config.Domain),
		VUrl:      fmt.Sprintf("https://%s%s", s.config.Domain, s.config.SensorUrl),
		BmSo:      bmSo,
		Language:  sbsdProviderLanguage(s.config.Language),
		Script:    script,
	}

	jsonData, _ := json.Marshal(req)

	resp, err := s.tlsClient.Request(TLSRequest{
		URL:     "https://n4s.xyz/sbsd",
		Method:  "POST",
		Browser: s.browser,
		Headers: map[string]string{
			"Content-Type": "application/json",
			"X-API-KEY":    "4DD7-F8F7-A935-972F-45B4-1A04",
		},
		Body:  string(jsonData),
		Proxy: s.proxy,
	})

	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(resp.GetBody()), &result); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	// Check for error
	if errMsg, ok := result["error"].(string); ok {
		return "", fmt.Errorf("n4s error: %s", errMsg)
	}

	// N4S returns the payload in "body" field
	body, ok := result["body"].(string)
	if !ok {
		return "", fmt.Errorf("body field not found in response")
	}

	return body, nil
}

// ============================================================================
// Roolink Provider
// ============================================================================

type sbsdRoolinkRequest struct {
	UserAgent string `json:"userAgent"`
	Language  string `json:"language"`
	Vid       string `json:"vid"`
	BmO       string `json:"bm_o"`
	URL       string `json:"url"`
	Static    bool   `json:"static"`
}

func (s *SBSDSolver) generateWithRoolink(bmSo string) (string, error) {
	log.Printf("→ Generating SBSD with Roolink")

	// Roolink expects bm_o; if we got bm_iso^{ts} strip the timestamp suffix
	if strings.Contains(bmSo, "^") {
		bmSo = strings.Split(bmSo, "^")[0]
	}

	vid := s.extractVidFromSensorURL()
	if vid == "" {
		return "", fmt.Errorf("could not extract vid from sensor URL")
	}

	req := sbsdRoolinkRequest{
		UserAgent: s.userAgent,
		Language:  sbsdProviderLanguage(s.config.Language),
		Vid:       vid,
		BmO:       bmSo,
		URL:       fmt.Sprintf("https://%s", s.config.Domain),
		Static:    false,
	}

	jsonData, _ := json.Marshal(req)

	resp, err := s.tlsClient.Request(TLSRequest{
		URL:     "https://www.roolink.io/api/v1/sbsd",
		Method:  "POST",
		Browser: s.browser,
		Headers: map[string]string{
			"Content-Type": "application/json",
			"x-api-key":    s.roolinkAPIKey(),
		},
		Body:  string(jsonData),
		Proxy: s.proxy,
	})

	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}

	if !resp.IsSuccess() {
		return "", fmt.Errorf("roolink returned status %d: %s", resp.GetStatus(), resp.GetBody())
	}

	// Check for error in response
	var errResp struct {
		Error string `json:"error"`
	}
	if json.Unmarshal([]byte(resp.GetBody()), &errResp) == nil && errResp.Error != "" {
		return "", fmt.Errorf("roolink error: %s", errResp.Error)
	}

	var okResp struct {
		Body string `json:"body"`
	}
	if err := json.Unmarshal([]byte(resp.GetBody()), &okResp); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	if okResp.Body == "" {
		return "", fmt.Errorf("missing 'body' in response")
	}

	return okResp.Body, nil
}

func (s *SBSDSolver) roolinkAPIKey() string {
	return "2710d9bf-26fd-4add-8172-805ba613d66b"
}

// ============================================================================
// Helper Functions
// ============================================================================

func (s *SBSDSolver) extractVidFromSensorURL() string {
	if strings.Contains(s.config.SensorUrl, "v=") {
		parts := strings.Split(s.config.SensorUrl, "v=")
		if len(parts) > 1 {
			return strings.Split(parts[1], "&")[0]
		}
	}
	return ""
}

func (s *SBSDSolver) buildHeaders() map[string]string {
	return map[string]string{
		"Accept":          "*/*",
		"Accept-Language": s.config.Language,
		"Accept-Encoding": "gzip, deflate, br",
		"Origin":          fmt.Sprintf("https://%s", s.config.Domain),
		"Referer":         fmt.Sprintf("https://%s/", s.config.Domain),
		"User-Agent":      s.userAgent,
	}
}

func (s *SBSDSolver) buildHeadersOrder() []string {
	return []string{
		"Accept",
		"Accept-Language",
		"Accept-Encoding",
		"Content-Type",
		"Origin",
		"Referer",
		"User-Agent",
	}
}

// sbsdProviderLanguage extracts the primary language tag from Accept-Language
func sbsdProviderLanguage(lang string) string {
	lang = strings.TrimSpace(lang)
	if lang == "" {
		return ""
	}
	first := strings.Split(lang, ",")[0]
	first = strings.Split(first, ";")[0]
	return strings.TrimSpace(first)
}

// sbsdCompressGzip compresses data with gzip
func sbsdCompressGzip(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)
	if _, err := writer.Write(data); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
