package api

import (
	"errors"
	"fmt"
	"net/http"

	"gerador_cookies/scraper"
)

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleABCK(w http.ResponseWriter, r *http.Request) {
	var req abckRequest
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid payload: %v", err))
		return
	}
	if err := req.validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	cfg := req.configCopy()
	cfg.SbSd = false
	ensureDefaults(&cfg)

	solver, err := s.scraperFactory(req.ProxyURL, &cfg)
	if err != nil {
		s.logf("scraper init failed: %v", err)
		writeError(w, http.StatusBadGateway, "failed to initialize scraper")
		return
	}
	defer solver.CloseReport()

	result, genErr := solver.GenerateABCK(req.Script)
	if result == nil {
		result = &scraper.ABCKResult{}
	}

	status := http.StatusOK
	if genErr != nil {
		status = statusFromError(genErr)
		s.logf("abck generation failed: %v", genErr)
	}

	writeJSON(w, status, buildABCKResponse(result))
}

func (s *Server) handleSBSD(w http.ResponseWriter, r *http.Request) {
	var req sbsdRequest
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid payload: %v", err))
		return
	}
	if err := req.validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	cfg := req.configCopy()
	cfg.SbSd = true
	ensureDefaults(&cfg)

	solver, err := s.scraperFactory(req.ProxyURL, &cfg)
	if err != nil {
		s.logf("scraper init failed: %v", err)
		writeError(w, http.StatusBadGateway, "failed to initialize scraper")
		return
	}
	defer solver.CloseReport()

	result, genErr := solver.GenerateSBSD(req.Script, req.BmSo)
	if result == nil {
		result = &scraper.SBSDResult{}
	}

	status := http.StatusOK
	if genErr != nil {
		status = statusFromError(genErr)
		s.logf("sbsd generation failed: %v", genErr)
	}

	writeJSON(w, status, buildSBSDResponse(result))
}

func statusFromError(err error) int {
	var solverErr *scraper.SolverError
	if errors.As(err, &solverErr) {
		return http.StatusBadGateway
	}
	return http.StatusInternalServerError
}
