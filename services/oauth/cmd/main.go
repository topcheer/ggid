// Package main is the entry point for the OAuth/OIDC Service.
package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/ggid/ggid/pkg/crypto"
	"github.com/ggid/ggid/services/oauth/internal/conf"
	"github.com/ggid/ggid/services/oauth/internal/server"
	"github.com/jackc/pgx/v5/pgxpool"
)

// envOrDefault returns the env var value or default if not set.
func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func init() {
	// Password pepper must match auth service for PasswordGrant to verify hashes.
	if pepper := os.Getenv("PASSWORD_PEPPER"); pepper != "" {
		crypto.SetPepper(pepper)
	} else {
		env := os.Getenv("GGID_ENV")
		if env != "test" && env != "dev" {
			log.Fatal("PASSWORD_PEPPER must be set in non-dev environments")
		}
		log.Println("WARNING: PASSWORD_PEPPER not set — password verification may be inconsistent with auth service")
	}
}

func main() {
	var (
		addr           = flag.String("addr", ":9005", "HTTP listen address")
		issuer         = flag.String("issuer", envOrDefault("OAUTH_ISSUER", "http://localhost:9005"), "OIDC issuer URL")
		privateKeyPath = flag.String("private-key", os.Getenv("OAUTH_PRIVATE_KEY_PATH"), "RSA private key path")
		publicKeyPath  = flag.String("public-key", os.Getenv("OAUTH_PUBLIC_KEY_PATH"), "RSA public key path")
		dbURL          = flag.String("db-url", os.Getenv("DATABASE_URL"), "PostgreSQL connection URL")
	)
	flag.Parse()

	if *privateKeyPath == "" {
		*privateKeyPath = "configs/rsa_private.pem"
	}
	if *publicKeyPath == "" {
		*publicKeyPath = "configs/rsa_public.pem"
	}
	if *dbURL == "" {
		*dbURL = "postgres://ggid:ggid@localhost:5432/ggid?sslmode=disable"
	}

	cfg := conf.Default()
	cfg.HTTP.Addr = *addr
	cfg.Issuer = *issuer
	cfg.PrivateKeyPath = *privateKeyPath
	cfg.PublicKeyPath = *publicKeyPath
	cfg.Database.URL = *dbURL

	// Initialize key provider (local default, PKCS#11 via GGID_KEY_PROVIDER env).
	providerType := envOrDefault("GGID_KEY_PROVIDER", "local")
	if providerType == "local" {
		if err := ensureLocalKeyPair(cfg.PrivateKeyPath, cfg.PublicKeyPath); err != nil {
			log.Fatalf("failed to ensure local key pair: %v", err)
		}
	}
	keyProvider, err := crypto.NewKeyProvider(context.Background(), crypto.KeyProviderConfig{
		Provider: providerType,
		Local: crypto.LocalKeyProviderConfig{
			PrivateKeyPath: cfg.PrivateKeyPath,
			PublicKeyPath:  cfg.PublicKeyPath,
		},
	})
	if err != nil {
		log.Fatalf("failed to initialize key provider: %v", err)
	}
	defer keyProvider.Close()

	srv, err := server.NewWithKeyProvider(cfg, keyProvider)
	if err != nil {
		log.Fatalf("failed to create oauth server: %v", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Start cleanup goroutine for expired tokens, auth codes, and id-token records.
	go startTokenCleanup(ctx, os.Getenv("DATABASE_URL"))

	if err := srv.Run(ctx); err != nil {
		log.Fatalf("oauth server error: %v", err)
	}

	log.Println("OAuth/OIDC service stopped")
}

// ensureLocalKeyPair generates an RSA key pair on disk ONLY in dev mode.
// In production, missing keys are a fatal error.
func ensureLocalKeyPair(privateKeyPath, publicKeyPath string) error {
	if _, err := os.Stat(privateKeyPath); err == nil {
		return nil
	}
	if os.Getenv("GGID_DEV_MODE") != "true" {
		return fmt.Errorf("RSA private key not found at %s — inject via secret mount (do not auto-generate in production)", privateKeyPath)
	}
	log.Printf("[DEV] Generating RSA key pair for local development: %s", privateKeyPath)
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

// startTokenCleanup periodically deletes expired tokens, auth codes, and id-token records.
func startTokenCleanup(ctx context.Context, dbURL string) {
	if dbURL == "" {
		return
	}
	// Use ParseConfig for explicit pool settings (max 4 conns for cleanup goroutine)
	cfg, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		log.Printf("token cleanup: failed to parse DB config: %v", err)
		return
	}
	if cfg.MaxConns == 0 {
		cfg.MaxConns = 4 // small pool for background cleanup
	}
	if cfg.MaxConnLifetime == 0 {
		cfg.MaxConnLifetime = 30 * time.Minute
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		log.Printf("token cleanup: failed to connect DB: %v", err)
		return
	}
	defer pool.Close()

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Delete expired/revoked refresh tokens (keep 7 days for audit).
			if tag, err := pool.Exec(ctx, `
				DELETE FROM refresh_tokens
				WHERE (expires_at < NOW() OR (revoked = true AND revoked_at < NOW() - INTERVAL '7 days'))
			`); err == nil {
				log.Printf("cleanup: deleted %d expired/revoked refresh tokens", tag.RowsAffected())
			}
			// Delete used/expired auth codes.
			if tag, err := pool.Exec(ctx, `
				DELETE FROM oauth_authorization_codes WHERE used = true OR expires_at < NOW()
			`); err == nil {
				log.Printf("cleanup: deleted %d expired auth codes", tag.RowsAffected())
			}
			// Delete expired id-token records.
			if tag, err := pool.Exec(ctx, `
				DELETE FROM oidc_id_tokens WHERE expires_at < NOW()
			`); err == nil {
				log.Printf("cleanup: deleted %d expired id-token records", tag.RowsAffected())
			}
			// Delete revoked OIDC refresh tokens after 7-day retention window,
			// and expired tokens after 30 days. Keeps revoked_at for forensics.
			if tag, err := pool.Exec(ctx, `
				DELETE FROM oidc_refresh_tokens
				WHERE (revoked = true AND revoked_at < NOW() - INTERVAL '7 days')
				OR created_at < NOW() - INTERVAL '30 days'
			`); err == nil {
				log.Printf("cleanup: deleted %d expired/revoked oidc_refresh_tokens", tag.RowsAffected())
			}
		}
	}
}
