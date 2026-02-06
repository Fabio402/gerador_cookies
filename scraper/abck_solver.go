package scraper

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"
)

// ABCKSolver handles the ABCK cookie generation flow
type ABCKSolver struct {
	tlsClient     *TLSAPIClient
	cookieJar     *CookieJar
	providerCache *ProviderCache
	config        *Config
	userAgent     string
	browser       string
	proxy         string
}

// NewABCKSolver creates a new ABCK solver
func NewABCKSolver(
	config *Config,
	tlsClient *TLSAPIClient,
	cookieJar *CookieJar,
	providerCache *ProviderCache,
	userAgent string,
	browser string,
	proxy string,
) *ABCKSolver {
	if browser == "" {
		browser = DefaultBrowser
	}
	return &ABCKSolver{
		tlsClient:     tlsClient,
		cookieJar:     cookieJar,
		providerCache: providerCache,
		config:        config,
		userAgent:     userAgent,
		browser:       browser,
		proxy:         proxy,
	}
}

// Solve executes the complete ABCK generation flow
func (s *ABCKSolver) Solve(script string) (*ABCKResult, error) {
	log.Printf("→ Starting ABCK solve flow (provider=%s)", s.config.AkamaiProvider)

	result := &ABCKResult{
		Session: SessionInfo{
			Proxy:       s.proxy,
			UserAgent:   s.userAgent,
			Browser:     s.browser,
			Domain:      s.config.Domain,
			Provider:    s.config.AkamaiProvider,
			GeneratedAt: time.Now(),
		},
	}

	var success bool
	var err error

	switch s.config.AkamaiProvider {
	case "jevi":
		success, err = s.solveWithJevi(script)
	case "n4s":
		success, err = s.solveWithN4S(script)
	case "roolink":
		success, err = s.solveWithRoolink(script)
	default:
		err = NewError(PhaseInit, "invalid provider", fmt.Errorf("unknown provider: %s", s.config.AkamaiProvider))
	}

	if err != nil {
		result.Success = false
		if solverErr, ok := err.(*SolverError); ok {
			result.Error = solverErr
		} else {
			result.Error = NewError(PhaseProviderCall, "solve failed", err)
		}
		return result, err
	}

	if !success {
		result.Success = false
		result.Error = NewError(PhaseCookieValidation, "validation failed", fmt.Errorf("_abck cookie validation failed after %d attempts", s.config.SensorPostLimit))
		return result, nil
	}

	result.Success = true
	result.Cookies = s.cookieJar.GetCookies(s.config.Domain)
	result.CookieString = s.cookieJar.GetCookieString(s.config.Domain)

	log.Printf("✓ ABCK solve succeeded (provider=%s)", s.config.AkamaiProvider)
	return result, nil
}

// solveWithJevi implements the Jevi provider flow
func (s *ABCKSolver) solveWithJevi(script string) (bool, error) {
	// Check cache for encoded data
	encodedData := ""
	if !s.config.ForceUpdateDynamics {
		if entry, ok := s.providerCache.Get(s.config.Domain, "jevi", "sensor"); ok && entry.Dynamic != "" {
			log.Printf("→ Using cached dynamic (provider=jevi, len=%d)", len(entry.Dynamic))
			encodedData = entry.Dynamic
		}
	}

	for i := 0; i < s.config.SensorPostLimit; i++ {
		sensorData, newEncodedData, err := s.callJeviAPI(script, encodedData, i)
		if err != nil {
			return false, NewErrorWithProvider(PhaseProviderCall, "jevi sensor generation", "jevi", err)
		}

		// Update cached encoded data
		if newEncodedData != "" && newEncodedData != encodedData {
			encodedData = newEncodedData
			s.providerCache.Upsert(s.config.Domain, "jevi", "sensor", nil, &encodedData)
		}

		success, err := s.postSensor(sensorData, i)
		if err != nil {
			return false, err
		}
		if success {
			return true, nil
		}
	}

	return false, nil
}

// solveWithN4S implements the N4S provider flow
func (s *ABCKSolver) solveWithN4S(script string) (bool, error) {
	// Get or generate dynamic data
	dynamicData := ""
	if !s.config.ForceUpdateDynamics {
		if entry, ok := s.providerCache.Get(s.config.Domain, "n4s", "sensor"); ok && entry.Dynamic != "" {
			log.Printf("→ Using cached dynamic (provider=n4s, len=%d)", len(entry.Dynamic))
			dynamicData = entry.Dynamic
		}
	}

	if dynamicData == "" {
		var err error
		dynamicData, err = s.generateN4SDynamic(script)
		if err != nil {
			return false, NewErrorWithProvider(PhaseProviderCall, "n4s dynamic generation", "n4s", err)
		}
		s.providerCache.Upsert(s.config.Domain, "n4s", "sensor", nil, &dynamicData)
	}

	for i := 0; i < s.config.SensorPostLimit; i++ {
		sensorData, err := s.callN4SAPI(dynamicData, i)
		if err != nil {
			return false, NewErrorWithProvider(PhaseProviderCall, "n4s sensor generation", "n4s", err)
		}

		success, err := s.postSensor(sensorData, i)
		if err != nil {
			return false, err
		}
		if success {
			return true, nil
		}
	}

	return false, nil
}

// solveWithRoolink implements the Roolink provider flow
func (s *ABCKSolver) solveWithRoolink(script string) (bool, error) {
	// Parse script to get script data
	var scriptData *abckRoolinkScriptData
	if !s.config.ForceUpdateDynamics {
		if entry, ok := s.providerCache.Get(s.config.Domain, "roolink", "sensor"); ok && entry.Dynamic != "" {
			var cached abckRoolinkScriptData
			if json.Unmarshal([]byte(entry.Dynamic), &cached) == nil && cached.Ver != "" {
				log.Printf("→ Using cached dynamic (provider=roolink)")
				scriptData = &cached
			}
		}
	}

	if scriptData == nil {
		var err error
		scriptData, err = s.parseRoolinkScript(script)
		if err != nil {
			log.Printf("→ Roolink parse failed: %v (continuing without scriptData)", err)
			scriptData = nil
		} else if scriptData != nil {
			if b, err := json.Marshal(scriptData); err == nil {
				dynamicStr := string(b)
				s.providerCache.Upsert(s.config.Domain, "roolink", "sensor", nil, &dynamicStr)
			}
		}
	}

	for i := 0; i < s.config.SensorPostLimit; i++ {
		sensorData, err := s.callRoolinkAPI(scriptData, i)
		if err != nil {
			// Retry without scriptData
			sensorData, err = s.callRoolinkAPI(nil, i)
			if err != nil {
				return false, NewErrorWithProvider(PhaseProviderCall, "roolink sensor generation", "roolink", err)
			}
		}

		success, err := s.postSensorRoolink(sensorData, i)
		if err != nil {
			return false, err
		}
		if success {
			return true, nil
		}
	}

	return false, nil
}

// postSensor sends sensor data to Akamai and validates response
func (s *ABCKSolver) postSensor(sensorData string, index int) (bool, error) {
	log.Printf("→ Posting sensor to Akamai [%d/%d]", index+1, s.config.SensorPostLimit)

	// Build sensor URL (remove v= parameter for ABCK)
	sensorPath := s.config.SensorUrl
	if u, parseErr := url.ParseRequestURI(sensorPath); parseErr == nil {
		q := u.Query()
		q.Del("v")
		u.RawQuery = q.Encode()
		sensorPath = u.String()
	}

	sensorURL := fmt.Sprintf("https://%s%s", s.config.Domain, sensorPath)

	// Build payload
	payload, _ := json.Marshal(map[string]string{"sensor_data": sensorData})

	// Build headers
	headers := s.buildHeaders()
	headers["Content-Type"] = "text/plain;charset=UTF-8"

	resp, err := s.tlsClient.Request(TLSRequest{
		URL:           sensorURL,
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
		return false, NewErrorWithProvider(PhaseSensorPost, "send sensor", s.config.AkamaiProvider, err)
	}

	// Store cookies from response
	if resp.GetCookies() != nil {
		s.cookieJar.FromTLSAPICookies(s.config.Domain, resp.GetCookies())
	}

	// Validate response
	return s.validateSensorResponse(resp), nil
}

// postSensorRoolink sends sensor data to Akamai for Roolink (slightly different format)
func (s *ABCKSolver) postSensorRoolink(sensorData string, index int) (bool, error) {
	log.Printf("→ Posting sensor to Akamai [%d/%d]", index+1, s.config.SensorPostLimit)

	sensorURL := fmt.Sprintf("https://%s%s", s.config.Domain, s.config.SensorUrl)

	// Roolink uses sensor_data in JSON
	payload, _ := json.Marshal(map[string]string{"sensor_data": sensorData})

	headers := s.buildHeaders()
	headers["Content-Type"] = "text/plain;charset=UTF-8"

	resp, err := s.tlsClient.Request(TLSRequest{
		URL:           sensorURL,
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
		return false, NewErrorWithProvider(PhaseSensorPost, "send sensor", "roolink", err)
	}

	// Store cookies from response
	if resp.GetCookies() != nil {
		s.cookieJar.FromTLSAPICookies(s.config.Domain, resp.GetCookies())
	}

	return s.validateSensorResponse(resp), nil
}

// validateSensorResponse checks if the sensor was accepted
func (s *ABCKSolver) validateSensorResponse(resp *TLSResponse) bool {
	body := resp.GetBody()

	// Check response body doesn't contain newlines
	isValid := !strings.Contains(body, "\n")

	// Check _abck cookie
	for _, cookie := range resp.GetCookies() {
		if cookie.Name == "_abck" {
			if strings.Contains(cookie.Value, "~0~") {
				isValid = true
				break
			}
			if s.config.LowSecurity && len(cookie.Value) == 541 {
				isValid = true
				break
			}
		}
	}

	return isValid
}

// ============================================================================
// Jevi Provider
// ============================================================================

type abckJeviSolverRequest struct {
	Mode             int                  `json:"mode"`
	LocalhostRequest abckJeviLocalRequest `json:"akamaiRequest"`
}

type abckJeviLocalRequest struct {
	Site           string `json:"site"`
	Abck           string `json:"abck"`
	Bmsz           string `json:"bmsz"`
	UserAgent      string `json:"userAgent"`
	Language       string `json:"language"`
	Script         string `json:"script"`
	EncodedData    string `json:"encodedData"`
	PayloadCounter int    `json:"payloadCounter"`
}

func (s *ABCKSolver) callJeviAPI(script, encodedData string, index int) (string, string, error) {
	log.Printf("→ Calling Jevi API [%d/%d]", index+1, s.config.SensorPostLimit)

	abck, bmsz := s.getAkamaiCookies()

	var scriptValue string
	if encodedData == "" && s.config.UseScript {
		scriptValue = script
	}

	req := abckJeviSolverRequest{
		Mode: 1,
		LocalhostRequest: abckJeviLocalRequest{
			Site:           s.config.Domain,
			Abck:           abck,
			Bmsz:           bmsz,
			UserAgent:      s.userAgent,
			Language:       abckProviderLanguage(s.config.Language),
			Script:         scriptValue,
			EncodedData:    encodedData,
			PayloadCounter: index,
		},
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return "", "", fmt.Errorf("marshal request: %w", err)
	}

	compressedData, err := abckCompressGzip(jsonData)
	if err != nil {
		return "", "", fmt.Errorf("compress payload: %w", err)
	}

	apiKey := s.config.JeviAPIKey
	if apiKey == "" {
		return "", "", fmt.Errorf("JEVI_API_KEY not configured")
	}
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
		return "", "", fmt.Errorf("request failed: %w", err)
	}

	if !resp.IsSuccess() {
		return "", "", fmt.Errorf("jevi returned status %d: %s", resp.GetStatus(), resp.GetBody())
	}

	// Extract EncodedData from response headers
	newEncodedData := ""
	if headers := resp.GetHeaders(); headers != nil {
		if ed, ok := headers["EncodedData"]; ok {
			newEncodedData = ed
		}
		if ed, ok := headers["encodeddata"]; ok {
			newEncodedData = ed
		}
	}

	// Parse sensor from response
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(resp.GetBody()), &result); err != nil {
		return "", "", fmt.Errorf("parse response: %w", err)
	}

	sensorData, ok := result["sensor_data"].(string)
	if !ok {
		return "", "", fmt.Errorf("sensor_data not found in response")
	}

	return sensorData, newEncodedData, nil
}

// ============================================================================
// N4S Provider
// ============================================================================

type abckN4sSensorRequest struct {
	Site        string          `json:"targetURL"`
	Abck        string          `json:"abck"`
	Bmsz        string          `json:"bm_sz"`
	UserAgent   string          `json:"user_agent"`
	EncodedData json.RawMessage `json:"dynamic"`
	FirstSensor bool            `json:"first_sensor"`
	ReqNumber   int             `json:"req_number"`
}

func (s *ABCKSolver) generateN4SDynamic(script string) (string, error) {
	log.Printf("→ Generating N4S dynamic data")

	req := map[string]string{"script": script}
	jsonData, _ := json.Marshal(req)

	resp, err := s.tlsClient.Request(TLSRequest{
		URL:     "https://n4s.xyz/v3_values",
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

	var result struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal([]byte(resp.GetBody()), &result); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	if len(result.Data) == 0 {
		return "", fmt.Errorf("empty data in response")
	}

	dataJSON, _ := json.MarshalIndent(result.Data, "", "  ")
	return string(dataJSON), nil
}

func (s *ABCKSolver) callN4SAPI(dynamicData string, index int) (string, error) {
	log.Printf("→ Calling N4S API [%d/%d]", index+1, s.config.SensorPostLimit)

	abck, bmsz := s.getAkamaiCookies()

	req := abckN4sSensorRequest{
		Site:        fmt.Sprintf("https://%s", s.config.Domain),
		Abck:        abck,
		Bmsz:        bmsz,
		UserAgent:   s.userAgent,
		EncodedData: json.RawMessage(dynamicData),
		FirstSensor: index == 0,
		ReqNumber:   index,
	}

	jsonData, _ := json.Marshal(req)

	resp, err := s.tlsClient.Request(TLSRequest{
		URL:     "https://n4s.xyz/sensor",
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

	sensorData, ok := result["sensor_data"].(string)
	if !ok {
		return "", fmt.Errorf("sensor_data not found in response")
	}

	return sensorData, nil
}

// ============================================================================
// Roolink Provider
// ============================================================================

type abckRoolinkScriptData struct {
	Ver string `json:"ver"`
	Key int    `json:"key"`
	Dvc string `json:"dvc"`
	Din []int  `json:"din"`
}

type abckRoolinkSensorRequest struct {
	URL        string                 `json:"url"`
	UserAgent  string                 `json:"userAgent"`
	Language   string                 `json:"language"`
	Abck       string                 `json:"_abck"`
	BmSz       string                 `json:"bm_sz"`
	ScriptUrl  string                 `json:"scriptUrl"`
	ScriptData *abckRoolinkScriptData `json:"scriptData,omitempty"`
	Index      int                    `json:"index"`
	Stepper    bool                   `json:"stepper,omitempty"`
}

func (s *ABCKSolver) roolinkAPIKey() string {
	return s.config.RoolinkAPIKey
}

func (s *ABCKSolver) parseRoolinkScript(script string) (*abckRoolinkScriptData, error) {
	// Try base64 decode if script appears to be encoded
	if decoded, err := base64.StdEncoding.DecodeString(script); err == nil {
		if len(decoded) > 0 && abckIsPrintableText(decoded) {
			script = string(decoded)
		}
	}

	resp, err := s.tlsClient.Request(TLSRequest{
		URL:     "https://www.roolink.io/api/v1/parse",
		Method:  "POST",
		Browser: s.browser,
		Headers: map[string]string{
			"Content-Type": "text/plain",
			"x-api-key":    s.roolinkAPIKey(),
		},
		Body:  script,
		Proxy: s.proxy,
	})

	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("roolink parse failed: status=%d", resp.GetStatus())
	}

	// Check for error in response
	var errResp struct {
		Error string `json:"error"`
	}
	if json.Unmarshal([]byte(resp.GetBody()), &errResp) == nil && errResp.Error != "" {
		return nil, fmt.Errorf("roolink error: %s", errResp.Error)
	}

	var data abckRoolinkScriptData
	if err := json.Unmarshal([]byte(resp.GetBody()), &data); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if data.Ver == "" {
		return nil, fmt.Errorf("missing 'ver' in response")
	}

	return &data, nil
}

func (s *ABCKSolver) callRoolinkAPI(scriptData *abckRoolinkScriptData, index int) (string, error) {
	log.Printf("→ Calling Roolink API [%d/%d]", index+1, s.config.SensorPostLimit)

	abck, bmsz := s.getAkamaiCookies()

	req := abckRoolinkSensorRequest{
		URL:        fmt.Sprintf("https://%s", s.config.Domain),
		UserAgent:  s.userAgent,
		Language:   abckProviderLanguage(s.config.Language),
		Abck:       abck,
		BmSz:       bmsz,
		ScriptUrl:  fmt.Sprintf("https://%s%s", s.config.Domain, s.config.SensorUrl),
		ScriptData: scriptData,
		Index:      index,
		Stepper:    true,
	}

	jsonData, _ := json.Marshal(req)

	resp, err := s.tlsClient.Request(TLSRequest{
		URL:     "https://www.roolink.io/api/v1/sensor",
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

	// Check for error in response
	var errResp struct {
		Error string `json:"error"`
	}
	if json.Unmarshal([]byte(resp.GetBody()), &errResp) == nil && errResp.Error != "" {
		return "", fmt.Errorf("roolink error: %s", errResp.Error)
	}

	var okResp struct {
		Sensor string `json:"sensor"`
	}
	if err := json.Unmarshal([]byte(resp.GetBody()), &okResp); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	if okResp.Sensor == "" {
		return "", fmt.Errorf("missing 'sensor' in response")
	}

	return okResp.Sensor, nil
}

// ============================================================================
// Helper Functions
// ============================================================================

func (s *ABCKSolver) getAkamaiCookies() (abck, bmsz string) {
	cookies := s.cookieJar.GetCookies(s.config.Domain)
	for _, c := range cookies {
		if strings.HasPrefix(c.Name, "_abck") {
			abck = c.Value
		} else if strings.HasPrefix(c.Name, "bm_sz") {
			bmsz = c.Value
		}
	}
	return
}

func (s *ABCKSolver) buildHeaders() map[string]string {
	return map[string]string{
		"Accept":          "*/*",
		"Accept-Language": s.config.Language,
		"Accept-Encoding": "gzip, deflate, br",
		"Origin":          fmt.Sprintf("https://%s", s.config.Domain),
		"Referer":         fmt.Sprintf("https://%s/", s.config.Domain),
		"User-Agent":      s.userAgent,
	}
}

func (s *ABCKSolver) buildHeadersOrder() []string {
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

// abckProviderLanguage extracts the primary language tag from Accept-Language
func abckProviderLanguage(lang string) string {
	lang = strings.TrimSpace(lang)
	if lang == "" {
		return ""
	}
	first := strings.Split(lang, ",")[0]
	first = strings.Split(first, ";")[0]
	return strings.TrimSpace(first)
}

// abckCompressGzip compresses data with gzip
func abckCompressGzip(data []byte) ([]byte, error) {
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

// abckIsPrintableText checks if data is mostly printable ASCII
func abckIsPrintableText(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	limit := len(data)
	if limit > 1024 {
		limit = 1024
	}
	printable := 0
	for i := 0; i < limit; i++ {
		b := data[i]
		if b == '\n' || b == '\r' || b == '\t' || (b >= 32 && b <= 126) {
			printable++
		}
	}
	return float64(printable)/float64(limit) > 0.85
}
