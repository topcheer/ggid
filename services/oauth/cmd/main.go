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

	"github.com/ggid/ggid/pkg/crypto"
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

	if err := srv.Run(ctx); err != nil {
		log.Fatalf("oauth server error: %v", err)
	}

	log.Println("OAuth/OIDC service stopped")
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
