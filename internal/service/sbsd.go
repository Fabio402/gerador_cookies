package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"gerador_cookies/internal/errors"
	"gerador_cookies/internal/response"
	"gerador_cookies/scraper"
)

type SbsdInput struct {
	Domain         string
	AkamaiURL      string
	Proxy          string
	ProfileType    string
	UserAgent      string
	SecChUa        string
	Language       string
	AkamaiProvider string
	GenerateReport bool
}

type SbsdOutput struct {
	Cookies        *response.Cookies
	Telemetry      *response.Telemetry
	Session        *response.Session
	PartialCookies *response.Cookies
	ReportPath     string
}

func (s *SolverService) GenerateSbsd(ctx context.Context, input *SbsdInput) (*SbsdOutput, error) {
	output := &SbsdOutput{}

	// STEP 1: Criar Scraper
	config := &scraper.Config{
		Domain:         input.Domain,
		SensorUrl:      input.AkamaiURL,
		Language:       input.Language,
		AkamaiProvider: input.AkamaiProvider,
		SbSdProvider:   input.AkamaiProvider,
		SbSd:           true,
		UserAgent:      input.UserAgent,
		SecChUa:        input.SecChUa,
		ProfileType:    input.ProfileType,
		GenerateReport: input.GenerateReport,
		Proxy:          input.Proxy,
		JeviAPIKey:     s.config.JeviAPIKey,
		N4SAPIKey:      s.config.N4SAPIKey,
		RoolinkAPIKey:  s.config.RoolinkAPIKey,
	}

	sc, err := scraper.NewScraper(input.Proxy, config)
	if err != nil {
		return output, errors.NewScraperInitError(err, input.Domain)
	}

	// STEP 2: GetAntiBotScriptURL
	akamaiURL, err := sc.GetAntiBotScriptURL(input.AkamaiURL)
	if err != nil {
		output.PartialCookies = s.collectCookies(sc, input.Domain)
		return output, errors.NewScriptURLExtractionError(err, input.Domain)
	}
	if akamaiURL == "" {
		output.PartialCookies = s.collectCookies(sc, input.Domain)
		return output, errors.NewScriptURLExtractionError(
			fmt.Errorf("script URL not found in page"),
			input.Domain,
		)
	}
	config.SensorUrl = akamaiURL

	// STEP 3: GetAntiBotScript
	scriptB64, err := sc.GetAntiBotScript()
	if err != nil {
		output.PartialCookies = s.collectCookies(sc, input.Domain)
		return output, errors.NewScriptFetchError(err, input.Domain)
	}

	// STEP 4: Decodificar script
	decodedScript, err := base64.StdEncoding.DecodeString(scriptB64)
	if err != nil {
		output.PartialCookies = s.collectCookies(sc, input.Domain)
		return output, errors.NewScriptDecodeError(err, input.Domain)
	}
	rawScript := string(decodedScript)

	// STEP 5: Extrair bm_so ou sbsd_o
	cookies := sc.GetCookies()
	var bmSo string
	for _, cookie := range cookies {
		if cookie.Name == "bm_so" {
			bmSo = cookie.Value
			break
		} else if cookie.Name == "sbsd_o" && bmSo == "" {
			bmSo = cookie.Value
		}
	}

	if bmSo == "" {
		output.PartialCookies = s.collectCookies(sc, input.Domain)
		return output, errors.NewBmSoExtractionError(input.Domain)
	}

	// STEP 6 & 7: GenerateSBSD (fluxo completo do scraper)
	result, err := sc.GenerateSBSD(rawScript, bmSo)
	if err != nil {
		output.PartialCookies = s.collectCookies(sc, input.Domain)
		return output, errors.NewSbsdGenerationError(err, input.AkamaiProvider, input.Domain)
	}

	if !result.Success {
		output.PartialCookies = s.collectCookies(sc, input.Domain)
		return output, errors.NewSbsdPostError(
			fmt.Errorf("%s", result.Error.RawError),
			input.AkamaiProvider,
			input.Domain,
		)
	}

	// STEP 8: Coletar todos os cookies
	finalCookies := sc.GetCookies()

	// Montar response
	output.Cookies = s.cookiesToResponse(finalCookies, input.Domain)
	output.Telemetry = s.buildSbsdTelemetry(finalCookies)
	output.Session = &response.Session{
		Provider: result.Session.Provider,
		Profile:  input.ProfileType,
	}

	return output, nil
}

func (s *SolverService) buildSbsdTelemetry(cookies []*http.Cookie) *response.Telemetry {
	var abck, bmsz, bms string
	for _, c := range cookies {
		if strings.HasPrefix(c.Name, "_abck") {
			abck = c.Value
		} else if strings.HasPrefix(c.Name, "bm_sz") {
			bmsz = c.Value
		} else if c.Name == "bm_s" {
			bms = c.Value
		}
	}

	var abckToken string
	if parts := strings.Split(abck, "~"); len(parts) > 0 {
		abckToken = parts[0]
	}

	return &response.Telemetry{
		AbckToken:   abckToken,
		BmSzEncoded: base64.StdEncoding.EncodeToString([]byte(bmsz)),
		BmSEncoded:  base64.StdEncoding.EncodeToString([]byte(bms)),
	}
}
