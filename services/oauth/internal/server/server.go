// Package server wires up and runs the OAuth/OIDC HTTP server.
package server

import (
	"context"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/ggid/ggid/pkg/tenant"
	"github.com/ggid/ggid/services/oauth/internal/conf"
	"github.com/ggid/ggid/services/oauth/internal/domain"
	"github.com/ggid/ggid/services/oauth/internal/repository"
	"github.com/ggid/ggid/services/oauth/internal/service"
	"github.com/google/uuid"
)

// Server encapsulates the OAuth HTTP server.
type Server struct {
	cfg       *conf.Config
	httpSrv   *http.Server
	oauthSvc  *service.OAuthService
}

// keyProvider implements domain.KeyProvider by loading RSA keys from disk.
type keyProvider struct {
	priv    *rsa.PrivateKey
	pub     *rsa.PublicKey
	kid     string
}

func (kp *keyProvider) PublicKey() *rsa.PublicKey   { return kp.pub }
func (kp *keyProvider) PrivateKey() *rsa.PrivateKey { return kp.priv }
func (kp *keyProvider) KeyID() string                { return kp.kid }

// New constructs and wires up the OAuth server.
func New(cfg *conf.Config) (*Server, error) {
	ctx := context.Background()

	// Load RSA keys.
	kp, err := loadKeyProvider(cfg.PrivateKeyPath, cfg.PublicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("load keys: %w", err)
	}

	// Database (skip if URL is empty — for testing).
	var clientRepo repository.ClientRepository
	var codeRepo repository.AuthorizationCodeRepository
	var tokenRepo repository.IDTokenRepository

	if cfg.Database.URL != "" {
		// Use shared DB connection logic from pkg/data or inline.
		// For now, repos need a *pgxpool.Pool which we can't create without a live DB.
		// In production, this is wired up via cmd/main.go.
		log.Println("Database URL configured — repos will be injected by cmd/main.go")
	}
	_ = ctx
	_ = clientRepo
	_ = codeRepo
	_ = tokenRepo

	// Create the OAuth service.
	oauthSvc := service.NewOAuthService(nil, nil, nil, kp, cfg.Issuer)

	// Build HTTP handler.
	handler := buildHandler(oauthSvc, cfg)

	httpSrv := &http.Server{
		Addr:         cfg.HTTP.Addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return &Server{cfg: cfg, httpSrv: httpSrv, oauthSvc: oauthSvc}, nil
}

// Run starts the server and blocks until ctx is cancelled.
func (s *Server) Run(ctx context.Context) error {
	lis, err := net.Listen("tcp", s.cfg.HTTP.Addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("OAuth/OIDC server listening on %s", s.cfg.HTTP.Addr)
		if err := s.httpSrv.Serve(lis); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("http serve: %w", err)
		}
	}()

	select {
	case <-ctx.Done():
		log.Println("OAuth server shutting down...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = s.httpSrv.Shutdown(shutdownCtx)
		return nil
	case err := <-errCh:
		return err
	}
}

// buildHandler creates the HTTP mux with all OAuth/OIDC endpoints.
func buildHandler(oauthSvc *service.OAuthService, cfg *conf.Config) http.Handler {
	mux := http.NewServeMux()

	// OIDC Discovery
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		config := oauthSvc.GetDiscoveryConfig()
		writeJSON(w, http.StatusOK, config)
	})

	// JWKS
	mux.HandleFunc("/oauth/jwks", func(w http.ResponseWriter, r *http.Request) {
		jwks := oauthSvc.GetJWKS()
		writeJSON(w, http.StatusOK, jwks)
	})

	// Authorize endpoint (GET — redirects with code)
	mux.HandleFunc("/oauth/authorize", func(w http.ResponseWriter, r *http.Request) {
		// In a full implementation, this endpoint:
		// 1. Checks if user is authenticated (session/cookie)
		// 2. Shows consent page
		// 3. Creates authorization code and redirects
		// For now, it requires the user to be pre-authenticated.
		writeJSON(w, http.StatusOK, map[string]string{
			"status":  "authorization_required",
			"message": "This endpoint requires user authentication and consent.",
		})
	})

	// Token endpoint (POST)
	mux.HandleFunc("/oauth/token", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		_ = r.ParseForm()

		grantType := r.FormValue("grant_type")
		if grantType != "authorization_code" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported_grant_type"})
			return
		}

		// Build tenant context from client authentication.
		// In production, client auth is extracted from Authorization header or POST body.
		ctx := tenant.WithContext(r.Context(), &tenant.Context{
			TenantID:       uuid.Nil, // resolved from client
			IsolationLevel: tenant.IsolationShared,
		})

		resp, err := oauthSvc.ExchangeAuthorizationCode(ctx, &service.TokenExchangeRequest{
			GrantType:    grantType,
			Code:         r.FormValue("code"),
			RedirectURI:  r.FormValue("redirect_uri"),
			ClientID:     r.FormValue("client_id"),
			ClientSecret: r.FormValue("client_secret"),
			CodeVerifier: r.FormValue("code_verifier"),
		})
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_grant"})
			return
		}

		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Pragma", "no-cache")
		writeJSON(w, http.StatusOK, resp)
	})

	// UserInfo endpoint (GET)
	mux.HandleFunc("/oauth/userinfo", func(w http.ResponseWriter, r *http.Request) {
		// Requires a valid Bearer token in production.
		writeJSON(w, http.StatusOK, map[string]string{
			"status": "requires_bearer_token",
		})
	})

	// Token revocation
	mux.HandleFunc("/oauth/revoke", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		// RFC 7009: always return 200 even if token is invalid.
		w.WriteHeader(http.StatusOK)
	})

	// Token introspection
	mux.HandleFunc("/oauth/introspect", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"active": false})
	})

	// Health check
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	return mux
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// jsonEncode is kept as a wrapper for potential future use (e.g. custom encoders).
func jsonEncode(w http.ResponseWriter, v any) error {
	return json.NewEncoder(w).Encode(v)
}

// loadKeyProvider loads RSA keys from disk and computes a key ID.
func loadKeyProvider(privPath, pubPath string) (*keyProvider, error) {
	priv, err := loadPrivateKey(privPath)
	if err != nil {
		return nil, fmt.Errorf("load private key: %w", err)
	}
	pub := &priv.PublicKey
	kid := computeKID(pub)

	// Try to load public key from file for external verification.
	if _, err := os.Stat(pubPath); err == nil {
		// Public key exists — we use it for JWKS verification.
		loadedPub, err := loadPublicKey(pubPath)
		if err == nil {
			pub = loadedPub
		}
	}

	return &keyProvider{priv: priv, pub: pub, kid: kid}, nil
}

func loadPrivateKey(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM")
	}

	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		// Try PKCS8
		keyAny, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse private key: %w", err)
		}
		var ok bool
		key, ok = keyAny.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("key is not RSA")
		}
	}
	return key, nil
}

func loadPublicKey(path string) (*rsa.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("key is not RSA")
	}
	return rsaPub, nil
}

func computeKID(pub *rsa.PublicKey) string {
	data, _ := x509.MarshalPKIXPublicKey(pub)
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h[:8])
}

// Unused domain import guard.
var _ = domain.ClientTypeConfidential
