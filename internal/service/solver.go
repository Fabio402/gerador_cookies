package service

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"gerador_cookies/internal/config"
	"gerador_cookies/internal/response"
	"gerador_cookies/scraper"
)

type SolverService struct {
	config *config.Config
}

func NewSolverService(cfg *config.Config) *SolverService {
	return &SolverService{
		config: cfg,
	}
}

// getProfile retorna o nome do perfil TLS (será usado pelo scraper internamente)
func (s *SolverService) getProfile(profileType string) string {
	if profileType == "" {
		return "chrome_144"
	}
	return profileType
}

// collectCookies coleta cookies parciais do scraper
func (s *SolverService) collectCookies(sc *scraper.Scraper, domain string) *response.Cookies {
	cookies := sc.GetCookies()
	if cookies == nil {
		return nil
	}
	return s.cookiesToResponse(cookies, domain)
}

// cookiesToResponse converte []*http.Cookie para response.Cookies
func (s *SolverService) cookiesToResponse(cookies []*http.Cookie, domain string) *response.Cookies {
	if len(cookies) == 0 {
		return nil
	}

	items := make([]response.CookieItem, 0, len(cookies))
	var parts []string

	for _, c := range cookies {
		items = append(items, response.CookieItem{
			Name:   c.Name,
			Value:  c.Value,
			Domain: c.Domain,
		})
		parts = append(parts, fmt.Sprintf("%s=%s", c.Name, c.Value))
	}

	return &response.Cookies{
		FullString: strings.Join(parts, "; "),
		Items:      items,
	}
}

// buildTelemetry constrói telemetria básica dos cookies
func (s *SolverService) buildTelemetry(cookies []*http.Cookie) *response.Telemetry {
	var abck, bmsz string
	for _, c := range cookies {
		if strings.HasPrefix(c.Name, "_abck") {
			abck = c.Value
		} else if strings.HasPrefix(c.Name, "bm_sz") {
			bmsz = c.Value
		}
	}

	var abckToken string
	if parts := strings.Split(abck, "~"); len(parts) > 0 {
		abckToken = parts[0]
	}

	return &response.Telemetry{
		AbckToken:   abckToken,
		BmSzEncoded: base64.StdEncoding.EncodeToString([]byte(bmsz)),
	}
}
