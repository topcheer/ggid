// Package main is the entry point for the OAuth/OIDC Service.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ggid/ggid/services/oauth/internal/conf"
	"github.com/ggid/ggid/services/oauth/internal/server"
)

// envOrDefault returns the env var value or default if not set.
func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
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

	srv, err := server.New(cfg)
	if err != nil {
		log.Fatalf("failed to create oauth server: %v", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := srv.Run(ctx); err != nil {
		log.Fatalf("oauth server error: %v", err)
	}

	log.Println("OAuth/OIDC service stopped")
}
