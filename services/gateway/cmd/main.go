// Package main is the entry point for the GGID API Gateway.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ggid/ggid/services/gateway/internal/config"
	"github.com/ggid/ggid/services/gateway/internal/middleware"
	"github.com/ggid/ggid/services/gateway/internal/router"
)

func main() {
	cfg := config.LoadFromEnv(config.Default())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize JWT key client (JWKS or static public key)
	jwks, err := middleware.NewJWKSClient(cfg.JWKSURL, cfg.PublicKeyPath)
	if err != nil {
		log.Fatalf("failed to initialize JWKS client: %v", err)
	}
	log.Printf("JWT key loaded (kid=%s)", jwks.KeyID())

	// Start background JWKS refresh (every 15 min) if using JWKS URL
	if cfg.JWKSURL != "" {
		jwks.StartRefresh(ctx, 15*time.Minute)
	}

	// Create gateway router
	gw := router.New(cfg, jwks)
	gw.PrintRoutes()

	// HTTP server
	srv := &http.Server{
		Addr:         cfg.Addr,
		Handler:      gw.Handler(),
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	go func() {
		log.Printf("API Gateway listening on %s", cfg.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down API Gateway...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("forced shutdown: %v", err)
	}
	log.Println("API Gateway stopped")
}
