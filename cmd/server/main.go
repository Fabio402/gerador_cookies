package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"gerador_cookies/internal/config"
	"gerador_cookies/internal/handler"
	"gerador_cookies/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Criar service
	solverService := service.NewSolverService(cfg)

	// Criar handlers
	sbsdHandler := handler.NewSbsdHandler(cfg, solverService)

	mux := http.NewServeMux()

	// Registrar rotas
	mux.HandleFunc("POST /sbsd", sbsdHandler.Handle)

	// Handlers serão adicionados nas próximas issues
	// mux.HandleFunc("POST /abck", abckHandler.Handle)
	// mux.HandleFunc("POST /search", searchHandler.Handle)
	// mux.HandleFunc("GET /health", healthHandler.Handle)
	// mux.HandleFunc("GET /ready", healthHandler.Ready)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      mux,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	go func() {
		log.Printf("Starting server on :%d", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Graceful shutdown (detalhes na issue #7)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	log.Println("Shutting down server...")
}
