// Package main is the entry point for the GGID Auth Service.
package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/smtp"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/ggid/ggid/pkg/authprovider"
	"github.com/ggid/ggid/pkg/crypto"
	"github.com/ggid/ggid/pkg/sysconfig"
	"github.com/ggid/ggid/pkg/truststore"
	"github.com/ggid/ggid/services/auth/internal/conf"
	"github.com/ggid/ggid/services/auth/internal/repository"
	"github.com/ggid/ggid/services/auth/internal/server"
	"github.com/ggid/ggid/services/auth/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc"
)

func main() {
	cfg := conf.LoadFromEnv(conf.Default())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Connect to PostgreSQL
	pool, err := pgxpool.New(ctx, cfg.Database.URL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("database ping failed: %v", err)
	}
	log.Println("connected to PostgreSQL")

	// 2. Connect to Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("redis ping failed: %v", err)
	}
	log.Println("connected to Redis")

	// 3. Build repositories
	credRepo := repository.NewCredentialRepository(pool)
	sessionRepo := repository.NewSessionRepository(pool)
	refreshRepo := repository.NewRefreshTokenRepository(pool)

	// 3a. Initialize key provider (local default, PKCS#11 via GGID_KEY_PROVIDER env)
	keyProvider, err := initKeyProvider(ctx, cfg.JWT)
	if err != nil {
		log.Fatalf("failed to initialize key provider: %v", err)
	}
	defer keyProvider.Close()
	log.Printf("key provider ready (kid=%s)", keyProvider.Metadata().KeyID)

	// 4. Build services
	tokenService, err := service.NewTokenService(
		keyProvider,
		cfg.JWT.Issuer,
		cfg.JWT.Audience,
		cfg.JWT.AccessTokenTTL,
		refreshRepo,
		rdb,
	)
	if err != nil {
		log.Fatalf("failed to create token service: %v", err)
	}
	log.Printf("JWT key loaded (kid=%s)", tokenService.KeyID())

	sessionService := service.NewSessionService(sessionRepo)
	passwordService := service.NewPasswordService(cfg.Password, credRepo, rdb)
	rateLimiter := service.NewRateLimiter(rdb)

	// 5. Build auth provider chain (local + optional LDAP)
	// Wire password pepper from env var (security hardening, P0).
	if pepper := os.Getenv("PASSWORD_PEPPER"); pepper != "" {
		crypto.SetPepper(pepper)
		log.Printf("Password pepper enabled (HMAC-SHA256 pre-hash)")
	}
	localProvider := service.NewLocalProvider(credRepo, cfg.Password)

	var providers []authprovider.Provider
	providers = append(providers, localProvider)

	// Wire LDAP provider when LDAP_URL is configured.
	if ldapURL := os.Getenv("LDAP_URL"); ldapURL != "" {
		ldapCfg := authprovider.LDAPConfig{
			ServerURL:     ldapURL,
			BindDN:        os.Getenv("LDAP_BIND_DN"),
			BindPassword:  os.Getenv("LDAP_BIND_PASSWORD"),
			BaseDN:        os.Getenv("LDAP_BASE_DN"),
			UserFilter:    os.Getenv("LDAP_USER_FILTER"),
			StartTLS:      os.Getenv("LDAP_START_TLS") == "true",
			AutoProvision: os.Getenv("LDAP_AUTO_PROVISION") == "true",
		}
		if ldapCfg.BaseDN == "" {
			ldapCfg.BaseDN = "dc=corp,dc=local"
		}
		if ldapCfg.UserFilter == "" {
			ldapCfg.UserFilter = "(&(objectClass=inetOrgPerson)(uid=%s))"
		}

		ldapProvider, err := authprovider.NewLDAPProvider(ldapCfg)
		if err != nil {
			log.Printf("WARNING: failed to create LDAP provider, skipping: %v", err)
		} else {
			// Wire trust store CA pool for custom CA support
			if ts := loadTrustStoreCAs(ctx, pool); ts != nil {
				if cp, err := ts.CertPool(); err == nil {
					ldapProvider.SetCAPool(cp)
					log.Printf("Trust store CA pool wired to LDAP provider")
				}
			}
			providers = append(providers, ldapProvider)
			log.Printf("LDAP provider configured: server=%s base=%s filter=%s",
				ldapCfg.ServerURL, ldapCfg.BaseDN, ldapCfg.UserFilter)
		}
	}

	chain := authprovider.NewChain(providers...)
	log.Printf("Auth provider chain: %d provider(s) configured", len(providers))

	// 5a. Build MFA service
	mfaRepo := repository.NewPGMFADeviceRepository(pool)
	mfaService := service.NewMFAService(mfaRepo)

	// 5b. Backup code service (PostgreSQL-backed for persistence)
	backupCodeRepo := service.NewPgBackupCodeRepo(pool)
	backupCodeService := service.NewBackupCodeService(backupCodeRepo)

	// 6. Build identity client (HTTP-based, connects to Identity Service)
	var identityClient service.IdentityClient
	identityURL := os.Getenv("IDENTITY_SERVICE_URL")
	if identityURL != "" {
		identityClient = service.NewHTTPIdentityClient(identityURL)
		log.Printf("Identity client configured: %s", identityURL)
	} else {
		identityClient = service.NewNoopIdentityClient()
		log.Printf("Identity client not configured (IDENTITY_SERVICE_URL not set), using degraded mode")
	}

	// 7. Build auth service
	authSvc := service.NewAuthService(
		cfg, chain, credRepo,
		tokenService, sessionService, passwordService,
		rateLimiter, identityClient,
		mfaService,
	)

	// 7a. Configure email sender if SMTP is configured
	smtpHost := os.Getenv("SMTP_HOST")
	if smtpHost != "" {
		smtpPort := 587
		if p := os.Getenv("SMTP_PORT"); p != "" {
			fmt.Sscanf(p, "%d", &smtpPort)
		}
		sender := &smtpEmailSender{
			host:     smtpHost,
			port:     smtpPort,
			from:     os.Getenv("SMTP_FROM"),
		}
		// Wire trust store CA pool for custom CA support
		if ts := loadTrustStoreCAs(ctx, pool); ts != nil {
			if cp, err := ts.CertPool(); err == nil {
				sender.SetCAPool(cp)
				log.Printf("Trust store CA pool wired to email sender")
			}
		}
		authSvc.SetEmailSender(sender)
		log.Printf("SMTP email sender configured: %s:%d", smtpHost, smtpPort)
	}

	// 7. Start session cleanup goroutine
	go startSessionCleanup(ctx, authSvc)

	// 7b. Initialize system config store (hot-reloadable via DB + Redis Pub/Sub)
	sysconfigStore := sysconfig.NewStore(pool, rdb)
	log.Printf("System config store initialized (hot-reload via Redis Pub/Sub)")

	// 8. Start HTTP server
	authSvc.SetBackupCodeService(backupCodeService)

	handler := server.New(authSvc)
	handler.SetSysconfigStore(sysconfigStore)
	httpServer := &http.Server{
		Addr:         cfg.Server.HTTP.Addr,
		Handler:      handler,
		ReadTimeout:  cfg.Server.HTTP.ReadTimeout,
		WriteTimeout: cfg.Server.HTTP.WriteTimeout,
	}

	go func() {
		log.Printf("Auth Service listening on %s", cfg.Server.HTTP.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// 8b. Start gRPC server (optional, if GRPC_ADDR is set)
	// Uses same TLS-aware pattern as org/policy/audit/identity services.
	grpcAddr := os.Getenv("AUTH_GRPC_ADDR")
	if grpcAddr == "" {
		grpcAddr = ":50052"
	}
	go func() {
		lis, err := net.Listen("tcp", grpcAddr)
		if err != nil {
			log.Printf("Auth gRPC: failed to listen on %s: %v (continuing HTTP-only)", grpcAddr, err)
			return
		}

		grpcServer := grpc.NewServer()

		// TLS support: when GRPC_TLS_ENABLED=true, attempt TLS credentials.
		if os.Getenv("GRPC_TLS_ENABLED") == "true" {
			certFile := os.Getenv("GRPC_TLS_CERT")
			keyFile := os.Getenv("GRPC_TLS_KEY")
			if certFile != "" && keyFile != "" {
				creds, err := credentials.NewServerTLSFromFile(certFile, keyFile)
				if err != nil {
					if os.Getenv("GRPC_TLS_ALLOW_PLAINTEXT_FALLBACK") != "true" {
						log.Fatalf("GRPC_TLS_ENABLED but cert/key invalid: %v; refusing to start. Set GRPC_TLS_ALLOW_PLAINTEXT_FALLBACK=true only in dev.", err)
					}
					log.Printf("Warning: GRPC_TLS_ENABLED but cert/key invalid: %v, falling back to plaintext (GRPC_TLS_ALLOW_PLAINTEXT_FALLBACK=true)", err)
				} else {
					// Recreate server with TLS credentials.
					lis.Close()
					lis, err = net.Listen("tcp", grpcAddr)
					if err != nil {
						log.Printf("Auth gRPC: failed to re-listen on %s: %v", grpcAddr, err)
						return
					}
					grpcServer = grpc.NewServer(grpc.Creds(creds))
					log.Printf("Auth gRPC: TLS enabled (cert=%s)", certFile)
				}
			}
		}

		authGRPCHandler := server.NewAuthGRPCHandler(authSvc)
		authGRPCHandler.RegisterGRPC(grpcServer)
		log.Printf("Auth gRPC server listening on %s", grpcAddr)
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("Auth gRPC server error: %v", err)
		}
	}()

	// 9. Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down Auth Service...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("forced shutdown: %v", err)
	}
	log.Println("Auth Service stopped")
}

// startSessionCleanup periodically removes expired sessions.
func startSessionCleanup(ctx context.Context, authSvc *service.AuthService) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			count, err := authSvc.CleanupExpired(ctx)
			if err != nil {
				log.Printf("session cleanup error: %v", err)
				continue
			}
			if count > 0 {
				log.Printf("cleaned up %d expired sessions", count)
			}
		}
	}
}

// init ensures configs directory exists for RSA key generation.
func init() {
	if err := os.MkdirAll("configs", 0o700); err != nil {
		panic(fmt.Sprintf("failed to create configs directory: %v", err))
	}
}

// smtpEmailSender implements service.EmailSender using net/smtp.
type smtpEmailSender struct {
	host   string
	port   int
	from   string
	caPool *x509.CertPool
}

// initKeyProvider creates a crypto.KeyProvider from auth service JWT config.
// GGID_KEY_PROVIDER selects "local" (default) or "pkcs11"; PKCS#11 env vars are read by the provider.
func initKeyProvider(ctx context.Context, jwtCfg conf.JWTConfig) (crypto.KeyProvider, error) {
	providerType := os.Getenv("GGID_KEY_PROVIDER")
	if providerType == "" {
		providerType = "local"
	}

	if providerType == "local" {
		if err := ensureLocalKeyPair(jwtCfg.PrivateKeyPath, jwtCfg.PublicKeyPath); err != nil {
			return nil, fmt.Errorf("ensure local key pair: %w", err)
		}
	}

	return crypto.NewKeyProvider(ctx, crypto.KeyProviderConfig{
		Provider: providerType,
		Local: crypto.LocalKeyProviderConfig{
			PrivateKeyPath: jwtCfg.PrivateKeyPath,
			PublicKeyPath:  jwtCfg.PublicKeyPath,
		},
	})
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

func (s *smtpEmailSender) SetCAPool(cp *x509.CertPool) {
	s.caPool = cp
}

func (s *smtpEmailSender) Send(ctx context.Context, to, subject, body string) error {
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		s.from, to, subject, body)
	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	return smtpSendMail(addr, s.from, []string{to}, []byte(msg))
}

// smtpSendMail sends plain SMTP (no auth, suitable for MailHog).
func smtpSendMail(addr, from string, to []string, msg []byte) error {
	return smtp.SendMail(addr, nil, from, to, msg)
}

// loadTrustStoreCAs loads trusted CA certificates from the database into a truststore.MemoryStore.
// Returns nil if no CAs are found (non-fatal — services start without custom CAs).
func loadTrustStoreCAs(ctx context.Context, pool *pgxpool.Pool) *truststore.MemoryStore {
	if pool == nil {
		return nil
	}
	ts := truststore.NewMemoryStore()
	rows, err := pool.Query(ctx, "SELECT name, pem_data FROM trusted_ca_certs")
	if err != nil {
		log.Printf("Trust store: failed to query CAs (non-fatal): %v", err)
		return nil
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var name, pemData string
		if err := rows.Scan(&name, &pemData); err != nil {
			continue
		}
		if _, err := ts.AddCA(name, pemData, "system"); err != nil {
			log.Printf("Trust store: failed to add CA %s: %v", name, err)
			continue
		}
		count++
	}
	if count == 0 {
		return nil
	}
	log.Printf("Trust store: %d CA(s) loaded", count)
	return ts
}
