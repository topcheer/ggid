// Package main is the entry point for the GGID Auth Service.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ggid/ggid/pkg/authprovider"
	"github.com/ggid/ggid/pkg/crypto"
	"github.com/ggid/ggid/services/auth/internal/conf"
	"github.com/ggid/ggid/services/auth/internal/repository"
	"github.com/ggid/ggid/services/auth/internal/server"
	"github.com/ggid/ggid/services/auth/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
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

	// 4. Build services
	tokenService, err := service.NewTokenService(cfg.JWT, refreshRepo, rdb)
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

	// 6. Build identity client (HTTP-based, connects to Identity Service)
	var identityClient service.IdentityClient
	identityURL := os.Getenv("IDENTITY_SERVICE_URL")
	if identityURL != "" {
		identityClient = service.NewHTTPIdentityClient(identityURL)
		log.Printf("Identity client configured: %s", identityURL)
	} else {
		identityClient = &service.NoopIdentityClient{}
		log.Printf("Identity client not configured (IDENTITY_SERVICE_URL not set)")
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
		authSvc.SetEmailSender(&smtpEmailSender{
			host:     smtpHost,
		port:     smtpPort,
		from:     os.Getenv("SMTP_FROM"),
		})
		log.Printf("SMTP email sender configured: %s:%d", smtpHost, smtpPort)
	}

	// 7. Start session cleanup goroutine
	go startSessionCleanup(ctx, authSvc)

	// 8. Start HTTP server
	handler := server.New(authSvc)
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
	host string
	port int
	from string
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
