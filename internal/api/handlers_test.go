package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gerador_cookies/scraper"
)

type fakeScraper struct {
	abckResult *scraper.ABCKResult
	abckErr    error
	sbsdResult *scraper.SBSDResult
	sbsdErr    error
	lastScript string
	lastBmSo   string
}

func (f *fakeScraper) GenerateABCK(script string) (*scraper.ABCKResult, error) {
	f.lastScript = script
	return f.abckResult, f.abckErr
}

func (f *fakeScraper) GenerateSBSD(script string, bmSo string) (*scraper.SBSDResult, error) {
	f.lastScript = script
	f.lastBmSo = bmSo
	return f.sbsdResult, f.sbsdErr
}

func (f *fakeScraper) CloseReport() {}

func TestHandleABCKSuccess(t *testing.T) {
	fake := &fakeScraper{
		abckResult: &scraper.ABCKResult{
			Success:      true,
			CookieString: "1~2",
			Session: scraper.SessionInfo{
				Provider: "jevi",
			},
		},
	}

	var capturedCfg *scraper.Config
	server := NewServer(WithScraperFactory(func(proxy string, cfg *scraper.Config) (solver, error) {
		copied := *cfg
		capturedCfg = &copied
		return fake, nil
	}))

	body := `{"config":{"domain":"example.com","sensorUrl":"/akam/123","akamaiProvider":"jevi"},"script":"abc123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/abck", strings.NewReader(body))
	rec := httptest.NewRecorder()

	server.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	if capturedCfg == nil {
		t.Fatalf("config not passed to scraper")
	}
	if capturedCfg.SbSd {
		t.Fatalf("expected SbSd to be false for ABCK")
	}
	if fake.lastScript != "abc123" {
		t.Fatalf("script not forwarded to scraper")
	}

	var payload generationResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("invalid response json: %v", err)
	}
	if !payload.Success {
		t.Fatalf("expected success true")
	}
	if payload.CookieString != "1~2" {
		t.Fatalf("unexpected cookie string: %s", payload.CookieString)
	}
	if payload.Session.Provider != "jevi" {
		t.Fatalf("unexpected provider %s", payload.Session.Provider)
	}
}

func TestHandleABCKValidationError(t *testing.T) {
	server := NewServer(WithScraperFactory(func(proxy string, cfg *scraper.Config) (solver, error) {
		return &fakeScraper{}, nil
	}))

	body := `{"config":{"domain":"example.com","sensorUrl":"/akam/123","akamaiProvider":""},"script":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/abck", strings.NewReader(body))
	rec := httptest.NewRecorder()

	server.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestHandleSBSDErrorResponse(t *testing.T) {
	fake := &fakeScraper{
		sbsdResult: &scraper.SBSDResult{
			Success: false,
			Error: &scraper.SolverError{
				Phase: scraper.PhaseProviderCall,
				Step:  "sbsd",
			},
		},
		sbsdErr: &scraper.SolverError{
			Phase: scraper.PhaseProviderCall,
			Step:  "sbsd",
		},
	}

	var capturedCfg *scraper.Config
	server := NewServer(WithScraperFactory(func(proxy string, cfg *scraper.Config) (solver, error) {
		copied := *cfg
		capturedCfg = &copied
		return fake, nil
	}))

	body := `{"config":{"domain":"shop.com","sensorUrl":"/sbsd","akamaiProvider":"n4s"},"script":"abc123","bmSo":"bm_value"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sbsd", strings.NewReader(body))
	rec := httptest.NewRecorder()

	server.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected status 502, got %d", rec.Code)
	}
	if capturedCfg == nil || !capturedCfg.SbSd {
		t.Fatalf("expected SbSd flag to be true")
	}
	if fake.lastBmSo != "bm_value" {
		t.Fatalf("bmSo not forwarded to scraper")
	}

	var payload generationResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if payload.Error == nil || payload.Error.Phase != string(scraper.PhaseProviderCall) {
		t.Fatalf("expected solver error in payload")
	}
}

func TestHandleSBSDValidationError(t *testing.T) {
	server := NewServer()
	body := `{"config":{"domain":"shop.com","sensorUrl":"/sbsd","akamaiProvider":"jevi"},"script":"abc"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sbsd", strings.NewReader(body))
	rec := httptest.NewRecorder()

	server.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}
