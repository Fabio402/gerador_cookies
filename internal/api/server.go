package api

import (
	"log"
	"net/http"
	"os"

	"gerador_cookies/scraper"
)

// Server wraps HTTP handlers for cookie generation endpoints.
type Server struct {
	mux            *http.ServeMux
	logger         *log.Logger
	scraperFactory scraperFactory
}

// Option customizes server behaviour.
type Option func(*Server)

// WithLogger overrides default logger.
func WithLogger(logger *log.Logger) Option {
	return func(s *Server) {
		if logger != nil {
			s.logger = logger
		}
	}
}

// WithScraperFactory overrides scraper constructor (useful for tests).
func WithScraperFactory(factory scraperFactory) Option {
	return func(s *Server) {
		if factory != nil {
			s.scraperFactory = factory
		}
	}
}

// NewServer builds a Server with default configuration.
func NewServer(opts ...Option) *Server {
	s := &Server{
		mux:            http.NewServeMux(),
		logger:         log.New(os.Stdout, "[api] ", log.LstdFlags),
		scraperFactory: defaultScraperFactory,
	}

	for _, opt := range opts {
		opt(s)
	}

	s.registerRoutes()
	return s
}

// Router exposes the configured mux.
func (s *Server) Router() http.Handler {
	return s.mux
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("GET /healthz", s.handleHealth)
	s.mux.HandleFunc("POST /api/v1/abck", s.handleABCK)
	s.mux.HandleFunc("POST /api/v1/sbsd", s.handleSBSD)
}

type solver interface {
	GenerateABCK(script string) (*scraper.ABCKResult, error)
	GenerateSBSD(script string, bmSo string) (*scraper.SBSDResult, error)
	CloseReport()
}

type scraperFactory func(proxyURL string, cfg *scraper.Config) (solver, error)

func defaultScraperFactory(proxyURL string, cfg *scraper.Config) (solver, error) {
	return scraper.NewScraper(proxyURL, cfg)
}

func (s *Server) logf(format string, args ...interface{}) {
	if s.logger != nil {
		s.logger.Printf(format, args...)
	}
}
