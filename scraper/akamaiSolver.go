package scraper

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	fhttp "github.com/bogdanfinn/fhttp"
)

func providerLanguage(lang string) string {
	lang = strings.TrimSpace(lang)
	if lang == "" {
		return ""
	}
	// Accept-Language can look like: pt-BR,pt;q=0.9,en-US;q=0.8,en;q=0.7
	// Providers should receive only the primary language tag (e.g. pt-BR).
	first := strings.Split(lang, ",")[0]
	first = strings.Split(first, ";")[0]
	return strings.TrimSpace(first)
}

type Response struct {
	Data map[string]interface{} `json:"data"` // A map to hold dynamic keys inside "data"
}

type SbSdPostRequest struct {
	Body string `json:"body"`
}

type AkamaiSolver struct {
	scraper      *Scraper
	apiType      string // "localhost" or "hwk"
	apiKey       string // API key for HWK API
	requestCount int32  // Counter for HWK API requests
}

func (as *AkamaiSolver) roolinkAPIKey() string {
	// Preferred env var
	return "2710d9bf-26fd-4add-8172-805ba613d66b"
}

type LocalhostRequestN4S struct {
	Site        string          `json:"targetURL"`
	Abck        string          `json:"abck"`
	Bmsz        string          `json:"bm_sz"`
	UserAgent   string          `json:"user_agent"`
	EncodedData json.RawMessage `json:"dynamic"`
	FirstSensor bool            `json:"first_sensor"`
	ReqNumber   int             `json:"req_number"`
}

type LocalhostRequest struct {
	Site           string `json:"site"`
	Abck           string `json:"abck"`
	Bmsz           string `json:"bmsz"`
	UserAgent      string `json:"userAgent"`
	Language       string `json:"language"`
	Script         string `json:"script"`
	EncodedData    string `json:"encodedData"`
	PayloadCounter int    `json:"payloadCounter"`
}

type GetHashData struct {
	Script string `json:"script"`
}

type SolverRequest struct {
	Mode             int              `json:"mode"`
	LocalhostRequest LocalhostRequest `json:"akamaiRequest"`
}

type SbsdRequest struct {
	NewVersion bool   `json:"NewVersion"`
	ScriptHash string `json:"ScriptHash"`
	Script     string `json:"Script"`
	Site       string `json:"Site"`
	SbsdO      string `json:"sbsd_o"`
	UserAgent  string `json:"userAgent"`
	Uuid       string `json:"uuid"`
}

type SbsdSolverRequest struct {
	Mode        int         `json:"mode"`
	SbsdRequest SbsdRequest `json:"SbsdRequest"`
}

type HWKRequest struct {
	Abck      string `json:"abck"`
	Bmsz      string `json:"bm_sz"`
	Config    string `json:"config"`
	Events    string `json:"events"`
	Site      string `json:"site"`
	UserAgent string `json:"user_agent"`
}

func NewAkamaiSolver(scraper *Scraper, apiType string, apiKey string) *AkamaiSolver {
	return &AkamaiSolver{
		scraper:      scraper,
		apiType:      apiType,
		apiKey:       apiKey,
		requestCount: 0,
	}
}

func (as *AkamaiSolver) Solve(script string) (bool, error) {
	log.Printf("→ Starting solve flow (akamaiProvider=%s)", as.scraper.config.AkamaiProvider)
	log.Printf("→ Provider language: %s (raw=%s)", providerLanguage(as.scraper.config.Language), as.scraper.config.Language)
	defer func() {
		log.Printf("→ Finished solve flow (akamaiProvider=%s)", as.scraper.config.AkamaiProvider)
	}()

	switch as.scraper.config.AkamaiProvider {
	case "n4s":
		hash := ""
		if !as.scraper.config.ForceUpdateDynamics {
			if entry, ok := as.scraper.cacheGet(); ok && entry.Dynamic != "" {
				log.Printf("→ Using cached provider dynamic (provider=n4s, len=%d)", len(entry.Dynamic))
				hash = entry.Dynamic
			}
		}
		if hash == "" {
			var err error
			hash, err = as.generateDynamic(script)
			if err != nil {
				return false, err
			}
			log.Printf("→ Saving provider dynamic to cache (provider=n4s, len=%d)", len(hash))
			as.scraper.cacheUpsert(nil, &hash)
		}

		for i := 0; i < as.scraper.config.SensorPostLimit; i++ {
			success, err := as.solveSingleN4S(hash, i)
			if err != nil {
				return false, fmt.Errorf("error in solving iteration %d: %v", i+1, err)
			}
			if success {
				return true, nil
			}
		}
		log.Printf("✗ Solve flow failed (akamaiProvider=%s)", as.scraper.config.AkamaiProvider)
		return false, nil
	case "jevi":
		if as.scraper.config.ForceUpdateDynamics {
			as.scraper.config.EncodedData = ""
		}
		if as.scraper.config.EncodedData == "" {
			if !as.scraper.config.ForceUpdateDynamics {
				if entry, ok := as.scraper.cacheGet(); ok && entry.Dynamic != "" {
					log.Printf("→ Using cached provider dynamic (provider=jevi, len=%d)", len(entry.Dynamic))
					as.scraper.config.EncodedData = entry.Dynamic
				}
			}
		}
		for i := 0; i < as.scraper.config.SensorPostLimit; i++ {
			success, err := as.solveSingle(script, i)
			if err != nil {
				return false, fmt.Errorf("error in solving iteration %d: %v", i+1, err)
			}
			if success {
				log.Printf("✓ Solve flow succeeded (akamaiProvider=%s)", as.scraper.config.AkamaiProvider)
				return true, nil
			}
		}
		log.Printf("✗ Solve flow failed (akamaiProvider=%s)", as.scraper.config.AkamaiProvider)
		return false, nil
	case "roolink":
		apiKey := as.roolinkAPIKey()
		var scriptData *roolinkScriptData
		if !as.scraper.config.ForceUpdateDynamics {
			if entry, ok := as.scraper.cacheGet(); ok && entry.Dynamic != "" {
				var cached roolinkScriptData
				if json.Unmarshal([]byte(entry.Dynamic), &cached) == nil && cached.Ver != "" {
					log.Printf("→ Using cached provider dynamic (provider=roolink, len=%d)", len(entry.Dynamic))
					scriptData = &cached
				}
			}
		}
		if scriptData == nil {
			var parseErr error
			scriptData, parseErr = as.roolinkParseScript(apiKey, script)
			if parseErr != nil {
				log.Printf("✗ RooLink parse failed: %v", parseErr)
				// Keep going without scriptData. RooLink can still generate sensors without it.
				scriptData = nil
			} else if scriptData != nil {
				if b, err := json.Marshal(scriptData); err == nil {
					v := string(b)
					log.Printf("→ Saving provider dynamic to cache (provider=roolink, len=%d)", len(v))
					as.scraper.cacheUpsert(nil, &v)
				}
			}
		}
		for i := 0; i < as.scraper.config.SensorPostLimit; i++ {
			success, err := as.solveSingleRoolink(script, scriptData, i)
			if err != nil {
				return false, fmt.Errorf("error in solving iteration %d: %v", i+1, err)
			}
			if success {
				log.Printf("✓ Solve flow succeeded (akamaiProvider=%s)", as.scraper.config.AkamaiProvider)
				return true, nil
			}
		}
		log.Printf("✗ Solve flow failed (akamaiProvider=%s)", as.scraper.config.AkamaiProvider)
		return false, nil
	default:
		return false, fmt.Errorf("invalid akamaiProvider: %s", as.scraper.config.AkamaiProvider)
	}
}

func (as *AkamaiSolver) solveSingleRoolink(script string, scriptData *roolinkScriptData, index int) (bool, error) {
	apiKey := as.roolinkAPIKey()

	cookies, err := as.scraper.GetCookies(fmt.Sprintf("https://%s", as.scraper.config.Domain))
	if err != nil {
		return false, fmt.Errorf("error getting cookies: %v", err)
	}

	var abck, bmsz string
	for _, cookie := range cookies {
		if strings.HasPrefix(cookie.Name, "_abck") {
			abck = cookie.Value
		} else if strings.HasPrefix(cookie.Name, "bm_sz") {
			bmsz = cookie.Value
		}
	}
	if abck == "" {
		return false, fmt.Errorf("missing _abck cookie")
	}
	if bmsz == "" {
		return false, fmt.Errorf("missing bm_sz cookie")
	}

	log.Printf("→ Getting sensor (roolink) [%d/%d]", index+1, as.scraper.config.SensorPostLimit)

	sensor, err := as.roolinkGenerateSensor(apiKey, abck, bmsz, scriptData, index)
	if err != nil {
		log.Printf("✗ RooLink sensor (with scriptData) failed: %v", err)
		sensor, err = as.roolinkGenerateSensor(apiKey, abck, bmsz, nil, index)
		if err != nil {
			log.Printf("✗ RooLink sensor (without scriptData) failed: %v", err)
			return false, err
		}
	}
	previewLen := 250
	if len(sensor) < previewLen {
		previewLen = len(sensor)
	}
	log.Printf("→ RooLink sensor generated (len=%d) preview=%q", len(sensor), sensor[:previewLen])

	log.Printf("→ Sending sensor to Akamai")

	payloadBytes, err := json.Marshal(map[string]string{"sensor_data": sensor})
	if err != nil {
		return false, fmt.Errorf("error marshaling Akamai sensor payload: %v", err)
	}

	antiBotReq, err := fhttp.NewRequest(fhttp.MethodPost, fmt.Sprintf("https://%s%s", as.scraper.config.Domain, as.scraper.config.SensorUrl), bytes.NewBuffer(payloadBytes))
	if err != nil {
		return false, fmt.Errorf("error creating anti-bot request: %v", err)
	}

	as.scraper.setHeaders(antiBotReq)
	antiBotReq.Header.Set("Content-Type", "text/plain;charset=UTF-8")

	antiBotResp, err := as.scraper.doRequest(antiBotReq)
	if err != nil {
		log.Printf("✗ Akamai post failed: %v", err)
		return false, fmt.Errorf("error sending anti-bot request: %v", err)
	}
	defer antiBotResp.Body.Close()
	antiBotBody, _ := io.ReadAll(antiBotResp.Body)

	isValid := !strings.Contains(string(antiBotBody), "\n")
	for _, cookie := range antiBotResp.Cookies() {
		if cookie.Name == "_abck" {
			if strings.Contains(cookie.Value, "~0~") || (as.scraper.config.LowSecurity == true && len(cookie.Value) == 541) {
				isValid = true
			}
		}
	}

	err = as.scraper.SetCookies(fmt.Sprintf("https://%s", as.scraper.config.Domain), antiBotResp.Cookies())
	if err != nil {
		return false, fmt.Errorf("error setting cookies: %v", err)
	}

	return isValid, nil
}

type roolinkScriptData struct {
	Ver string `json:"ver"`
	Key int    `json:"key"`
	Dvc string `json:"dvc"`
	Din []int  `json:"din"`
}

func (as *AkamaiSolver) roolinkParseScript(apiKey string, script string) (*roolinkScriptData, error) {
	// RooLink expects the raw JS (text/plain). In our pipeline the script is often base64-encoded.
	// Try base64 decode and accept it when it looks like mostly-text content.
	if decoded, err := base64.StdEncoding.DecodeString(script); err == nil {
		if len(decoded) > 0 {
			printable := 0
			limit := len(decoded)
			if limit > 1024 {
				limit = 1024
			}
			for i := 0; i < limit; i++ {
				b := decoded[i]
				if b == '\n' || b == '\r' || b == '\t' || (b >= 32 && b <= 126) {
					printable++
				}
			}
			// If it's mostly printable text, treat it as decoded JS.
			if float64(printable)/float64(limit) > 0.85 {
				script = string(decoded)
			}
		}
	}

	req, err := http.NewRequest(http.MethodPost, "https://www.roolink.io/api/v1/parse", bytes.NewBufferString(script))
	if err != nil {
		return nil, fmt.Errorf("error creating roolink parse request: %v", err)
	}
	req.Header.Set("content-type", "text/plain")
	req.Header.Set("x-api-key", apiKey)

	resp, err := as.scraper.doSimpleRequest(req)
	if err != nil {
		return nil, fmt.Errorf("error making roolink parse request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading roolink parse response: %v", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("roolink parse failed: status=%d body=%s", resp.StatusCode, string(body))
	}

	var errResp struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(body, &errResp) == nil && errResp.Error != "" {
		return nil, fmt.Errorf("roolink parse error: %s", errResp.Error)
	}

	var out roolinkScriptData
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("error unmarshaling roolink parse response: %v. Body: %s", err, string(body))
	}
	if out.Ver == "" {
		return nil, fmt.Errorf("roolink parse response missing 'ver'. Body: %s", string(body))
	}

	return &out, nil
}

func (as *AkamaiSolver) roolinkGenerateSensor(apiKey string, abck string, bmsz string, scriptData *roolinkScriptData, index int) (string, error) {
	type roolinkSensorReq struct {
		URL        string             `json:"url"`
		UserAgent  string             `json:"userAgent"`
		Language   string             `json:"language"`
		Abck       string             `json:"_abck"`
		BmSz       string             `json:"bm_sz"`
		ScriptUrl  string             `json:"scriptUrl"`
		ScriptData *roolinkScriptData `json:"scriptData,omitempty"`
		Index      int                `json:"index"`
		Stepper    bool               `json:"stepper,omitempty"`
	}

	log.Printf("→ Provider request (provider=roolink, type=sensor, index=%d) language=%s (raw=%s)", index, providerLanguage(as.scraper.config.Language), as.scraper.config.Language)

	payload := roolinkSensorReq{
		URL:        fmt.Sprintf("https://%s", as.scraper.config.Domain),
		UserAgent:  as.scraper.userAgent.Full,
		Language:   providerLanguage(as.scraper.config.Language),
		Abck:       abck,
		BmSz:       bmsz,
		ScriptUrl:  fmt.Sprintf("https://%s%s", as.scraper.config.Domain, as.scraper.config.SensorUrl),
		ScriptData: scriptData,
		Index:      index,
		Stepper:    true,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("error marshaling roolink sensor request: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, "https://www.roolink.io/api/v1/sensor", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("error creating roolink sensor request: %v", err)
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-api-key", apiKey)

	resp, err := as.scraper.doSimpleRequest(req)
	if err != nil {
		return "", fmt.Errorf("error making roolink sensor request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading roolink sensor response: %v", err)
	}

	var errResp struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(body, &errResp) == nil && errResp.Error != "" {
		return "", fmt.Errorf("roolink sensor error: %s", errResp.Error)
	}

	var okResp struct {
		Sensor string `json:"sensor"`
	}
	if err := json.Unmarshal(body, &okResp); err != nil {
		return "", fmt.Errorf("error unmarshaling roolink sensor response: %v. Body: %s", err, string(body))
	}
	if okResp.Sensor == "" {
		return "", fmt.Errorf("roolink sensor response missing 'sensor'. Body: %s", string(body))
	}

	return okResp.Sensor, nil
}

func (as *AkamaiSolver) solveSingleN4S(hash string, index int) (bool, error) {
	cookies, err := as.scraper.GetCookies(fmt.Sprintf("https://%s", as.scraper.config.Domain))
	if err != nil {
		return false, fmt.Errorf("error getting cookies: %v", err)
	}

	var abck, bmsz string
	for _, cookie := range cookies {
		if strings.HasPrefix(cookie.Name, "_abck") {
			abck = cookie.Value
		} else if strings.HasPrefix(cookie.Name, "bm_sz") {
			bmsz = cookie.Value
		}
	}

	var sensorData string
	switch as.apiType {
	case "localhost":
		sensorData, err = as.callLocalhostAPIN4S(abck, bmsz, hash, index)
	default:
		return false, fmt.Errorf("invalid API type: %s", as.apiType)
	}

	if err != nil {
		return false, err
	}

	var result map[string]interface{}
	err = json.Unmarshal([]byte(sensorData), &result)

	if err != nil {
		return false, err
	}

	sensorDataOnly, ok := result["sensor_data"].(string)
	if !ok {
		return false, fmt.Errorf("sensor_data is not a string")
	}

	as.scraper.SensorDataOnly = sensorDataOnly

	success, err := as.sendAntiBotRequest(sensorDataOnly)
	if err != nil {
		return false, err
	}

	return success, nil
}

func (as *AkamaiSolver) solveSingle(script string, index int) (bool, error) {
	cookies, err := as.scraper.GetCookies(fmt.Sprintf("https://%s", as.scraper.config.Domain))
	if err != nil {
		return false, fmt.Errorf("error getting cookies: %v", err)
	}

	var abck, bmsz string
	for _, cookie := range cookies {
		if strings.HasPrefix(cookie.Name, "_abck") {
			abck = cookie.Value
		} else if strings.HasPrefix(cookie.Name, "bm_sz") {
			bmsz = cookie.Value
		}
	}

	var sensorData string
	switch as.apiType {
	case "localhost":
		sensorData, err = as.callLocalhostAPI(abck, bmsz, script, index)
	default:
		return false, fmt.Errorf("invalid API type: %s", as.apiType)
	}

	if err != nil {
		return false, err
	}

	var result map[string]interface{}
	err = json.Unmarshal([]byte(sensorData), &result)

	if err != nil {
		return false, err
	}

	sensorDataOnly, ok := result["sensor_data"].(string)
	if !ok {
		return false, fmt.Errorf("sensor_data is not a string")
	}

	as.scraper.SensorDataOnly = sensorDataOnly

	success, err := as.sendAntiBotRequest(sensorDataOnly)
	if err != nil {
		return false, err
	}

	return success, nil
}

func (as *AkamaiSolver) generateDynamic(script string) (string, error) {
	log.Printf("→ Generating dynamic data")
	solverReq := GetHashData{
		Script: script,
	}

	jsonData, err := json.Marshal(solverReq)
	if err != nil {
		return "", fmt.Errorf("error marshaling solver request: %v", err)
	}

	req, err := http.NewRequest("POST", "https://n4s.xyz/v3_values", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("error creating solver API request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-KEY", "4DD7-F8F7-A935-972F-45B4-1A04")

	resp, err := as.scraper.doSimpleRequest(req)
	if err != nil {
		return "", fmt.Errorf("error making solver API request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result Response
	err = json.Unmarshal(body, &result)
	if err != nil {
		return "", fmt.Errorf("error unmarshaling v3_values response: %v, body: %s", err, string(body))
	}

	if len(result.Data) == 0 {
		return "", fmt.Errorf("v3_values API returned empty data field. Status: %d, Response: %s", resp.StatusCode, string(body))
	}

	dataJSON, err := json.MarshalIndent(result.Data, "", "  ")
	if err != nil {
		return "", err
	}

	return string(dataJSON), nil
}

func (as *AkamaiSolver) callLocalhostAPIN4S(abck, bmsz string, hash string, index int) (string, error) {
	solverReq := LocalhostRequestN4S{
		Site:        fmt.Sprintf("https://%s", as.scraper.config.Domain),
		Abck:        abck,
		Bmsz:        bmsz,
		UserAgent:   as.scraper.userAgent.Full,
		EncodedData: json.RawMessage(hash),
		FirstSensor: index == 0,
		ReqNumber:   index,
	}

	jsonData, err := json.Marshal(solverReq)
	if err != nil {
		return "", fmt.Errorf("error marshaling solver request: %v", err)
	}

	req, err := http.NewRequest("POST", "https://n4s.xyz/sensor", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("error creating solver API request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-KEY", "4DD7-F8F7-A935-972F-45B4-1A04")

	resp, err := as.scraper.doSimpleRequest(req)
	if err != nil {
		return "", fmt.Errorf("error making solver API request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func (as *AkamaiSolver) callLocalhostAPI(abck, bmsz string, script string, index int) (string, error) {
	var scriptValue string
	var encodedValue string

	if len(as.scraper.config.EncodedData) == 0 {
		if as.scraper.config.UseScript {
			scriptValue = script
		} else {
			scriptValue = ""
		}
	} else {
		encodedValue = as.scraper.config.EncodedData
	}

	solverReq := SolverRequest{
		Mode: 1,
		LocalhostRequest: LocalhostRequest{
			Site:           as.scraper.config.Domain,
			Abck:           abck,
			Bmsz:           bmsz,
			UserAgent:      as.scraper.userAgent.Full,
			Language:       providerLanguage(as.scraper.config.Language),
			Script:         scriptValue,
			EncodedData:    encodedValue,
			PayloadCounter: index,
		},
	}
	log.Printf("→ Provider request (provider=localhost, type=akamaiRequest, index=%d) language=%s (raw=%s)", index, solverReq.LocalhostRequest.Language, as.scraper.config.Language)

	jsonData, err := json.Marshal(solverReq)
	if err != nil {
		return "", fmt.Errorf("error marshaling solver request: %v", err)
	}

	compressedData, err := compressPayload(jsonData)
	if err != nil {
		return "", fmt.Errorf("error compressing payload: %v", err)
	}

	req, err := http.NewRequest("POST", "https://new.jevi.dev/Solver/solve", bytes.NewBuffer(compressedData))
	if err != nil {
		return "", fmt.Errorf("error creating solver API request: %v", err)
	}

	apiKey := "curiousT-a23f417f-096e-4258-adea-7ea874a57e56"
	userAgentPrefix := strings.Split(apiKey, "-")[0]

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("User-Agent", userAgentPrefix)
	req.Header.Set("x-key", apiKey)

	resp, err := as.scraper.doSimpleRequest(req)
	if err != nil {
		return "", fmt.Errorf("error making solver API request: %v", err)
	}
	defer resp.Body.Close()

	encodedData := resp.Header.Get("EncodedData")
	as.scraper.config.EncodedData = encodedData
	if encodedData != "" {
		as.scraper.cacheUpsert(nil, &encodedData)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("jevi solver failed: status=%d body=%s", resp.StatusCode, string(body))
	}

	return string(body), nil
}

func (as *AkamaiSolver) GenerateSbSd(script string, bmSo string) (string, error) {
	provider := as.scraper.config.AkamaiProvider
	if as.scraper.config.SbSdProvider != "" {
		provider = as.scraper.config.SbSdProvider
	}
	log.Printf("→ Provider request (provider=%s, type=sbsd) language=%s (raw=%s)", provider, providerLanguage(as.scraper.config.Language), as.scraper.config.Language)
	log.Printf("→ Generating SbSd (provider=%s)", provider)

	switch provider {
	case "n4s":
		return as.GenerateSbSdN4S(script, bmSo)
	case "jevi":
		return as.GenerateSbSdJevi(script, bmSo)
	case "roolink":
		return as.GenerateSbSdRoolink(bmSo)
	default:
		return "", fmt.Errorf("invalid SbSd provider: %s", provider)
	}
}

func (as *AkamaiSolver) GenerateSbSdRoolink(bmSo string) (string, error) {
	apiKey := as.roolinkAPIKey()

	// RooLink expects bm_o; if we got bm_iso^{ts} strip the timestamp suffix.
	if strings.Contains(bmSo, "^") {
		bmSo = strings.Split(bmSo, "^")[0]
	}

	vid := ""
	if strings.Contains(as.scraper.config.SensorUrl, "v=") {
		parts := strings.Split(as.scraper.config.SensorUrl, "v=")
		if len(parts) > 1 {
			vid = strings.Split(parts[1], "&")[0]
		}
	}
	if vid == "" {
		return "", fmt.Errorf("could not parse vid from sensor URL")
	}

	type roolinkSbsdReq struct {
		UserAgent string `json:"userAgent"`
		Language  string `json:"language"`
		Vid       string `json:"vid"`
		BmO       string `json:"bm_o"`
		URL       string `json:"url"`
		Static    bool   `json:"static"`
	}

	payload := roolinkSbsdReq{
		UserAgent: as.scraper.userAgent.Full,
		Language:  providerLanguage(as.scraper.config.Language),
		Vid:       vid,
		BmO:       bmSo,
		URL:       fmt.Sprintf("https://%s", as.scraper.config.Domain),
		Static:    false,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("error marshaling roolink sbsd request: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, "https://www.roolink.io/api/v1/sbsd", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("error creating roolink sbsd request: %v", err)
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-api-key", apiKey)

	resp, err := as.scraper.doSimpleRequest(req)
	if err != nil {
		return "", fmt.Errorf("error making roolink sbsd request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading roolink sbsd response: %v", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("roolink sbsd failed: status=%d body=%s", resp.StatusCode, string(body))
	}

	var errResp struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(body, &errResp) == nil && errResp.Error != "" {
		return "", fmt.Errorf("roolink sbsd error: %s", errResp.Error)
	}

	var okResp struct {
		Body string `json:"body"`
	}
	if err := json.Unmarshal(body, &okResp); err != nil {
		return "", fmt.Errorf("error unmarshaling roolink sbsd response: %v. Body: %s", err, string(body))
	}
	if okResp.Body == "" {
		return "", fmt.Errorf("roolink sbsd response missing 'body'. Body: %s", string(body))
	}

	return okResp.Body, nil
}

func (as *AkamaiSolver) GenerateSbSdN4S(script string, bmSo string) (string, error) {
	log.Printf("→ Generating SbSd payload")
	type SbSdRequestN4S struct {
		UserAgent string `json:"user_agent"`
		TargetURL string `json:"targetURL"`
		VUrl      string `json:"v_url"`
		BmSo      string `json:"bm_so"`
		Language  string `json:"language"`
		Script    string `json:"script"`
	}

	solverReq := SbSdRequestN4S{
		UserAgent: as.scraper.userAgent.Full,
		TargetURL: fmt.Sprintf("https://%s", as.scraper.config.Domain),
		VUrl:      fmt.Sprintf("https://%s%s", as.scraper.config.Domain, as.scraper.config.SensorUrl),
		BmSo:      bmSo,
		Language:  providerLanguage(as.scraper.config.Language),
		Script:    script,
	}

	jsonData, err := json.Marshal(solverReq)
	if err != nil {
		return "", fmt.Errorf("error marshaling SbSd request: %v", err)
	}

	// Log what we're sending to N4S
	log.Printf("→ N4S SbSd request: user_agent=%s, targetURL=%s, v_url=%s, bm_so=%s, language=%s, script_len=%d",
		solverReq.UserAgent,
		solverReq.TargetURL,
		solverReq.VUrl,
		solverReq.BmSo,
		solverReq.Language,
		len(solverReq.Script))

	req, err := http.NewRequest("POST", "https://n4s.xyz/sbsd", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("error creating SbSd API request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-KEY", "4DD7-F8F7-A935-972F-45B4-1A04")

	resp, err := as.scraper.doSimpleRequest(req)
	if err != nil {
		return "", fmt.Errorf("error making SbSd API request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading SbSd response: %v", err)
	}

	// Parse the response to extract the "body" field (N4S returns payload in "body")
	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return "", fmt.Errorf("error unmarshaling SbSd response: %v. Body: %s", err, string(body))
	}

	// Check for error in response
	if errMsg, hasError := result["error"].(string); hasError {
		return "", fmt.Errorf("SbSd API error: %s. Response: %s", errMsg, string(body))
	}

	// N4S returns the payload in "body" field, ready to be sent
	data, ok := result["body"].(string)
	if !ok {
		return "", fmt.Errorf("body field not found in SbSd response. Status: %d, Response: %s", resp.StatusCode, string(body))
	}

	return data, nil
}

func compressPayload(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)

	_, err := writer.Write(data)
	if err != nil {
		return nil, err
	}

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (as *AkamaiSolver) GenerateSbSdJevi(script string, bmSo string) (string, error) {
	// Call jevi.dev SBSD API (mode 3)
	// Extract uuid from sensor URL (v= parameter)
	uuid := ""
	if strings.Contains(as.scraper.config.SensorUrl, "v=") {
		parts := strings.Split(as.scraper.config.SensorUrl, "v=")
		if len(parts) > 1 {
			uuid = strings.Split(parts[1], "&")[0]
		}
	}

	// Calculate SHA256 hash of the raw script (jevi-sdk does this)
	// Note: script parameter is the RAW JavaScript, not base64
	scriptBytes := []byte(script)

	// Try first without script (empty), if 400 then retry with script
	solverReq := SbsdSolverRequest{
		Mode: 3,
		SbsdRequest: SbsdRequest{
			NewVersion: true,
			ScriptHash: "", // Leave empty on first attempt
			Script:     "", // Empty on first attempt
			Site:       fmt.Sprintf("https://%s/", as.scraper.config.Domain),
			SbsdO:      bmSo,
			UserAgent:  as.scraper.userAgent.Full,
			Uuid:       uuid,
		},
	}

	jsonData, err := json.Marshal(solverReq)
	if err != nil {
		return "", fmt.Errorf("error marshaling SBSD request: %v", err)
	}

	// Compress the payload with gzip as required by jevi API
	compressedData, err := compressPayload(jsonData)
	if err != nil {
		return "", fmt.Errorf("error compressing SBSD payload: %v", err)
	}

	req, err := http.NewRequest("POST", "https://new.jevi.dev/Solver/solve", bytes.NewBuffer(compressedData))
	if err != nil {
		return "", fmt.Errorf("error creating SBSD API request: %v", err)
	}

	apiKey := "curiousT-a23f417f-096e-4258-adea-7ea874a57e56"
	userAgentPrefix := strings.Split(apiKey, "-")[0]

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("User-Agent", userAgentPrefix)
	req.Header.Set("x-key", apiKey)

	resp, err := as.scraper.doSimpleRequest(req)
	if err != nil {
		return "", fmt.Errorf("error making SBSD API request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading SBSD response: %v", err)
	}

	// Parse JSON response to extract body field
	var jeviResponse struct {
		Body string `json:"body"`
	}
	if err := json.Unmarshal(body, &jeviResponse); err == nil && jeviResponse.Body != "" {
		body = []byte(jeviResponse.Body)
	}

	// If status is 400 or contains error, retry with base64-encoded script
	if resp.StatusCode == 400 || strings.Contains(string(body), "Script hash or script content must be provided") || strings.Contains(string(body), "Error processing SBSD request") {
		// Base64 encode the raw script
		base64Script := base64.StdEncoding.EncodeToString(scriptBytes)

		solverReq.SbsdRequest.Script = base64Script

		jsonData, err = json.Marshal(solverReq)
		if err != nil {
			return "", fmt.Errorf("error marshaling SBSD request (retry): %v", err)
		}

		// Compress the payload
		compressedData, err = compressPayload(jsonData)
		if err != nil {
			return "", fmt.Errorf("error compressing SBSD payload (retry): %v", err)
		}

		req, err = http.NewRequest("POST", "https://new.jevi.dev/Solver/solve", bytes.NewBuffer(compressedData))
		if err != nil {
			return "", fmt.Errorf("error creating SBSD API request (retry): %v", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Content-Encoding", "gzip")
		req.Header.Set("User-Agent", userAgentPrefix)
		req.Header.Set("x-key", apiKey)

		resp, err = as.scraper.doSimpleRequest(req)
		if err != nil {
			return "", fmt.Errorf("error making SBSD API request (retry): %v", err)
		}
		defer resp.Body.Close()

		body, err = io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("error reading SBSD response (retry): %v", err)
		}

		// Parse JSON response to extract body field
		if err := json.Unmarshal(body, &jeviResponse); err == nil && jeviResponse.Body != "" {
			body = []byte(jeviResponse.Body)
		}
	}

	// Check for errors in response body even with 200 status
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("SBSD API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Check if response still contains error message
	if strings.Contains(string(body), "Invalid credentials") {
		return "", fmt.Errorf("SBSD API error: Invalid credentials - check API key")
	}
	if strings.Contains(string(body), "Error processing SBSD request") {
		return "", fmt.Errorf("SBSD API error: %s", string(body))
	}

	return string(body), nil
}

func (as *AkamaiSolver) PostSbSdChallenge(sbSdData string) error {
	log.Printf("→ Sending SbSd to Akamai")
	url := fmt.Sprintf("https://%s%s", as.scraper.config.Domain, as.scraper.config.SensorUrl)

	postReq := SbSdPostRequest{
		Body: sbSdData,
	}

	jsonData, err := json.Marshal(postReq)
	if err != nil {
		return fmt.Errorf("error marshaling SbSd post request: %v", err)
	}

	req, err := fhttp.NewRequest(fhttp.MethodPost, url, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("error creating SbSd post request: %v", err)
	}

	as.scraper.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := as.scraper.doRequest(req)
	if err != nil {
		return fmt.Errorf("error posting SbSd challenge: %v", err)
	}
	defer resp.Body.Close()

	// Validate response: must be 200 or 202
	if resp.StatusCode != 200 && resp.StatusCode != 202 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("SbSd challenge failed: expected status 200 or 202, got %d. Response: %s", resp.StatusCode, string(body))
	}

	// Check Content-Length from response header must be 0
	// if resp.ContentLength != 0 {
	// 	body, _ := io.ReadAll(resp.Body)
	// 	return fmt.Errorf("SbSd challenge failed: expected Content-Length 0, got %d. Response: %s", resp.ContentLength, string(body))
	// }

	log.Printf("✓ SbSd challenge accepted (status: %d, content-length: %d)", resp.StatusCode, resp.ContentLength)

	// Set cookies from response
	err = as.scraper.SetCookies(fmt.Sprintf("https://%s", as.scraper.config.Domain), resp.Cookies())
	if err != nil {
		return fmt.Errorf("error setting SbSd cookies: %v", err)
	}

	return nil
}

func (as *AkamaiSolver) sendAntiBotRequest(sensorData string) (bool, error) {
	log.Printf("→ Sending sensor to Akamai")
	payloadBytes, err := json.Marshal(map[string]string{"sensor_data": sensorData})
	if err != nil {
		return false, fmt.Errorf("error marshaling Akamai sensor payload: %v", err)
	}

	sensorPath := as.scraper.config.SensorUrl
	if u, parseErr := url.ParseRequestURI(sensorPath); parseErr == nil {
		q := u.Query()
		q.Del("v")
		u.RawQuery = q.Encode()
		sensorPath = u.String()
	}

	antiBotReq, err := fhttp.NewRequest(http.MethodPost, fmt.Sprintf("https://%s%s", as.scraper.config.Domain, sensorPath), bytes.NewBuffer(payloadBytes))
	if err != nil {
		return false, fmt.Errorf("error creating anti-bot request: %v", err)
	}

	as.scraper.setHeaders(antiBotReq)
	antiBotReq.Header.Set("Content-Type", "text/plain;charset=UTF-8")

	antiBotResp, err := as.scraper.doRequest(antiBotReq)
	if err != nil {
		return false, fmt.Errorf("error sending anti-bot request: %v", err)
	}
	defer antiBotResp.Body.Close()

	antiBotBody, err := io.ReadAll(antiBotResp.Body)
	if err != nil {
		return false, fmt.Errorf("error reading anti-bot response body: %v", err)
	}

	isValid := !strings.Contains(string(antiBotBody), "\n")

	for _, cookie := range antiBotResp.Cookies() {
		if cookie.Name == "_abck" {
			if strings.Contains(cookie.Value, "~0~") || (as.scraper.config.LowSecurity == true && len(cookie.Value) == 541) {
				isValid = true

				break
			}
		}
	}

	if isValid {
		err = as.scraper.SetCookies(fmt.Sprintf("https://%s", as.scraper.config.Domain), antiBotResp.Cookies())
		if err != nil {
			return false, fmt.Errorf("error setting cookies: %v", err)
		}
		return true, nil
	}

	err = as.scraper.SetCookies(fmt.Sprintf("https://%s", as.scraper.config.Domain), antiBotResp.Cookies())
	if err != nil {
		return false, fmt.Errorf("error setting cookies: %v", err)
	}

	return false, nil
}
