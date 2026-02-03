package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"gerador_cookies/internal/api"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	addr := resolveAddr()
	server := api.NewServer()

	httpServer := &http.Server{
		Addr:              addr,
		Handler:           server.Router(),
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      2 * time.Minute,
		IdleTimeout:       90 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	go func() {
		log.Printf("⇨ HTTP API listening on %s", addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log.Printf("⇨ shutting down HTTP API")
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("graceful shutdown failed: %v", err)
	}
}

func resolveAddr() string {
	if addr := os.Getenv("API_ADDR"); addr != "" {
		return addr
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if strings.HasPrefix(port, ":") {
		return port
	}
	return ":" + port
}
