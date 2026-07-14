// Package main is the entry point for the GGID API Gateway.
package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/ggid/ggid/pkg/crypto"
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

	// Initialize key provider for dynamic JWKS serving (local default, PKCS#11 via GGID_KEY_PROVIDER).
	privateKeyPath := defaultPrivateKeyPath(cfg.PublicKeyPath)
	providerType := envOrDefault("GGID_KEY_PROVIDER", "local")
	if providerType == "local" {
		if err := ensureLocalKeyPair(privateKeyPath, cfg.PublicKeyPath); err != nil {
			log.Fatalf("failed to ensure local key pair: %v", err)
		}
	}
	keyProvider, err := crypto.NewKeyProvider(ctx, crypto.KeyProviderConfig{
		Provider: providerType,
		Local: crypto.LocalKeyProviderConfig{
			PrivateKeyPath: privateKeyPath,
			PublicKeyPath:  cfg.PublicKeyPath,
		},
	})
	if err != nil {
		log.Fatalf("failed to initialize key provider: %v", err)
	}
	defer keyProvider.Close()
	jwks.SetKeyProvider(keyProvider)
	log.Printf("key provider wired to JWKS endpoint (kid=%s)", keyProvider.Metadata().KeyID)

	// Connect to Redis for sysconfig hot-reload (optional — gateway degrades gracefully without it)
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	var store sysconfig.Store
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Redis not available for gateway sysconfig (using defaults): %v", err)
		store = sysconfig.NewStore(nil, nil) // DB-less, defaults only
	} else {
		log.Printf("connected to Redis for sysconfig")
		store = sysconfig.NewStore(nil, rdb) // Gateway uses Redis cache only; DB writes go through auth service
	}

	// Create gateway router
	gw := router.New(cfg, jwks)
	gw.SetSysconfigStore(store)
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

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func defaultPrivateKeyPath(publicKeyPath string) string {
	if len(publicKeyPath) > 11 && publicKeyPath[len(publicKeyPath)-11:] == "public.pem" {
		return publicKeyPath[:len(publicKeyPath)-11] + "private.pem"
	}
	return "configs/rsa_private.pem"
}

// ensureLocalKeyPair generates an RSA key pair on disk if the private key is missing.
func ensureLocalKeyPair(privateKeyPath, publicKeyPath string) error {
	if _, err := os.Stat(privateKeyPath); err == nil {
		return nil
	}
	_ = os.MkdirAll(filepath.Dir(privateKeyPath), 0o700)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("generate RSA key: %w", err)
	}
	privData := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	if err := os.WriteFile(privateKeyPath, privData, 0o600); err != nil {
		return fmt.Errorf("write private key: %w", err)
	}
	pubBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return fmt.Errorf("marshal public key: %w", err)
	}
	pubData := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	})
	_ = os.MkdirAll(filepath.Dir(publicKeyPath), 0o700)
	if err := os.WriteFile(publicKeyPath, pubData, 0o644); err != nil {
		return fmt.Errorf("write public key: %w", err)
	}
	log.Printf("Generated new RSA key pair: %s + %s", privateKeyPath, publicKeyPath)
	return nil
}
