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

	"github.com/redis/go-redis/v9"
	"github.com/ggid/ggid/pkg/sysconfig"
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

	// Connect to Redis for sysconfig hot-reload (optional — gateway degrades gracefully without it)
	// TODO: wire sysconfigStore into gateway middleware for rate-limit / session-timeout hot-reload
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Redis not available for gateway sysconfig (using defaults): %v", err)
		_ = sysconfig.NewStore(nil, nil) // DB-less, defaults only
	} else {
		log.Printf("connected to Redis for sysconfig")
		_ = sysconfig.NewStore(nil, rdb) // Gateway uses Redis cache only; DB writes go through auth service
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

	// Graceful shutdown: SIGTERM → stop accepting new requests →
	// wait for in-flight requests (max 30s) → exit
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("received shutdown signal, draining in-flight requests...")

	// srv.Shutdown closes the listener (no new connections accepted)
	// and waits for active requests to complete.
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("forced shutdown after 30s timeout: %v", err)
	}

	// Cancel background context (JWKS refresh goroutines)
	cancel()
	log.Println("API Gateway stopped gracefully")
}
