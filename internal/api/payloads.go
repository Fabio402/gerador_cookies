package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"gerador_cookies/scraper"
)

const maxBodyBytes = 2 << 20 // 2 MiB

type abckRequest struct {
	Config  *scraper.Config `json:"config"`
	Script  string          `json:"script"`
	ProxyURL string         `json:"proxyUrl,omitempty"`
}

type sbsdRequest struct {
	Config  *scraper.Config `json:"config"`
	Script  string          `json:"script"`
	BmSo    string          `json:"bmSo"`
	ProxyURL string         `json:"proxyUrl,omitempty"`
}

func (r *abckRequest) validate() error {
	if r.Config == nil {
		return errors.New("config is required")
	}
	if strings.TrimSpace(r.Config.Domain) == "" {
		return errors.New("config.domain is required")
	}
	if strings.TrimSpace(r.Config.SensorUrl) == "" {
		return errors.New("config.sensorUrl is required")
	}
	if strings.TrimSpace(r.Config.AkamaiProvider) == "" {
		return errors.New("config.akamaiProvider is required")
	}
	if strings.TrimSpace(r.Script) == "" {
		return errors.New("script is required")
	}
	return nil
}

func (r *sbsdRequest) validate() error {
	if err := (&abckRequest{Config: r.Config, Script: r.Script}).validate(); err != nil {
		return err
	}
	if strings.TrimSpace(r.BmSo) == "" {
		return errors.New("bmSo is required")
	}
	return nil
}

func (r *abckRequest) configCopy() scraper.Config {
	if r.Config == nil {
		return scraper.Config{}
	}
	cfg := *r.Config
	return cfg
}

func (r *sbsdRequest) configCopy() scraper.Config {
	if r.Config == nil {
		return scraper.Config{}
	}
	cfg := *r.Config
	return cfg
}

func ensureDefaults(cfg *scraper.Config) {
	if cfg.SensorPostLimit <= 0 {
		cfg.SensorPostLimit = 5
	}
	if strings.TrimSpace(cfg.Language) == "" {
		cfg.Language = "en-US"
	}
}

func parseJSON(r *http.Request, dst interface{}) error {
	defer r.Body.Close()
	limited := io.LimitReader(r.Body, maxBodyBytes)
	dec := json.NewDecoder(limited)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return errors.New("unexpected trailing data")
		}
		return err
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	_ = enc.Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

type cookieDTO struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Domain   string `json:"domain,omitempty"`
	Path     string `json:"path,omitempty"`
	Expires  string `json:"expires,omitempty"`
	MaxAge   int    `json:"maxAge,omitempty"`
	Secure   bool   `json:"secure,omitempty"`
	HTTPOnly bool   `json:"httpOnly,omitempty"`
	SameSite string `json:"sameSite,omitempty"`
}

type solverErrorDTO struct {
	Phase      string `json:"phase"`
	Step       string `json:"step"`
	Provider   string `json:"provider,omitempty"`
	Domain     string `json:"domain,omitempty"`
	StatusCode int    `json:"statusCode,omitempty"`
	RawError   string `json:"rawError,omitempty"`
	Retryable  bool   `json:"retryable"`
}

type generationResponse struct {
	Success      bool                 `json:"success"`
	Cookies      []cookieDTO          `json:"cookies"`
	CookieString string               `json:"cookieString"`
	Session      scraper.SessionInfo  `json:"session"`
	Error        *solverErrorDTO      `json:"error,omitempty"`
}

func buildABCKResponse(res *scraper.ABCKResult) generationResponse {
	return generationResponse{
		Success:      res.Success,
		Cookies:      mapCookies(res.Cookies),
		CookieString: res.CookieString,
		Session:      res.Session,
		Error:        mapSolverError(res.Error),
	}
}

func buildSBSDResponse(res *scraper.SBSDResult) generationResponse {
	return generationResponse{
		Success:      res.Success,
		Cookies:      mapCookies(res.Cookies),
		CookieString: res.CookieString,
		Session:      res.Session,
		Error:        mapSolverError(res.Error),
	}
}

func mapCookies(cookies []*http.Cookie) []cookieDTO {
	if len(cookies) == 0 {
		return nil
	}
	result := make([]cookieDTO, 0, len(cookies))
	for _, c := range cookies {
		if c == nil {
			continue
		}
		dto := cookieDTO{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			MaxAge:   c.MaxAge,
			Secure:   c.Secure,
			HTTPOnly: c.HttpOnly,
			SameSite: formatSameSite(c.SameSite),
		}
		if !c.Expires.IsZero() {
			dto.Expires = c.Expires.UTC().Format(time.RFC3339)
		} else if c.RawExpires != "" {
			dto.Expires = c.RawExpires
		}
		result = append(result, dto)
	}
	return result
}

func formatSameSite(value http.SameSite) string {
	switch value {
	case http.SameSiteDefaultMode:
		return "default"
	case http.SameSiteLaxMode:
		return "lax"
	case http.SameSiteStrictMode:
		return "strict"
	case http.SameSiteNoneMode:
		return "none"
	default:
		return ""
	}
}

func mapSolverError(err *scraper.SolverError) *solverErrorDTO {
	if err == nil {
		return nil
	}
	return &solverErrorDTO{
		Phase:      string(err.Phase),
		Step:       err.Step,
		Provider:   err.Provider,
		Domain:     err.Domain,
		StatusCode: err.StatusCode,
		RawError:   err.RawError,
		Retryable:  err.Retryable,
	}
}
