// Package server wires up and runs the OAuth/OIDC HTTP server.
package server

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
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
	"github.com/jackc/pgx/v5/pgxpool"
)

// Server encapsulates the OAuth HTTP server.
type Server struct {
	cfg      *conf.Config
	httpSrv  *http.Server
	oauthSvc *service.OAuthService
	pool     *pgxpool.Pool
}

// keyProvider implements domain.KeyProvider by loading RSA keys from disk.
type keyProvider struct {
	priv *rsa.PrivateKey
	pub  *rsa.PublicKey
	kid  string
}

func (kp *keyProvider) PublicKey() *rsa.PublicKey   { return kp.pub }
func (kp *keyProvider) PrivateKey() *rsa.PrivateKey { return kp.priv }
func (kp *keyProvider) KeyID() string               { return kp.kid }

// New constructs and wires up the OAuth server.
func New(cfg *conf.Config) (*Server, error) {
	ctx := context.Background()

	// Load or create RSA keys — shares same paths as Auth Service.
	kp, err := loadOrCreateKeyProvider(cfg.PrivateKeyPath, cfg.PublicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("load keys: %w", err)
	}
	log.Printf("OAuth key loaded (kid=%s)", kp.KeyID())

	var (
		clientRepo repository.ClientRepository
		codeRepo   repository.AuthorizationCodeRepository
		tokenRepo  repository.IDTokenRepository
		pool       *pgxpool.Pool
	)

	if cfg.Database.URL != "" {
		poolCfg, err := pgxpool.ParseConfig(cfg.Database.URL)
		if err == nil {
			if cfg.Database.MaxConns > 0 {
				poolCfg.MaxConns = cfg.Database.MaxConns
			}
			if cfg.Database.MinConns > 0 {
				poolCfg.MinConns = cfg.Database.MinConns
			}
			p, err := pgxpool.NewWithConfig(ctx, poolCfg)
			if err == nil && p.Ping(ctx) == nil {
				pool = p
				clientRepo = repository.NewPGClientRepository(pool)
				codeRepo = repository.NewPGAuthorizationCodeRepository(pool)
				tokenRepo = repository.NewPGIDTokenRepository(pool)
				log.Println("OAuth database connected")
			} else if err != nil {
				log.Printf("warning: failed to connect database: %v (running without DB)", err)
			} else {
				log.Printf("warning: database ping failed (running without DB)")
				p.Close()
			}
		}
	}

	// Create the OAuth service.
	oauthSvc := service.NewOAuthService(clientRepo, codeRepo, tokenRepo, kp, cfg.Issuer)

	// Build HTTP handler.
	handler := buildHandler(oauthSvc, cfg)

	httpSrv := &http.Server{
		Addr:         cfg.HTTP.Addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return &Server{cfg: cfg, httpSrv: httpSrv, oauthSvc: oauthSvc, pool: pool}, nil
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
		if s.pool != nil {
			s.pool.Close()
		}
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

		ctx := tenant.WithContext(r.Context(), &tenant.Context{
			TenantID:       uuid.Nil,
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
		writeJSON(w, http.StatusOK, map[string]string{"status": "requires_bearer_token"})
	})

	// Token revocation
	mux.HandleFunc("/oauth/revoke", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
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

	// --- SAML 2.0 IdP skeleton ---

	mux.HandleFunc("/saml/metadata", func(w http.ResponseWriter, r *http.Request) {
		entityID := cfg.Issuer + "/saml"
		meta := samlMetadata(entityID, cfg.Issuer)
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(meta))
	})

	mux.HandleFunc("/saml/acs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		// POST Binding: SAMLResponse in form body
		_ = r.ParseForm()
		samlResponse := r.FormValue("SAMLResponse")
		if samlResponse == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing SAMLResponse"})
			return
		}
		// Skeleton: in production, parse and validate SAML assertion here.
		writeJSON(w, http.StatusOK, map[string]string{
			"status": "saml_response_received",
			"note":   "SAML assertion processing not yet implemented",
		})
	})

	mux.HandleFunc("/saml/sso", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{
			"status": "saml_sso_initiated",
			"note":   "SP-initiated SSO redirect placeholder",
		})
	})

	mux.HandleFunc("/saml/slo", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "saml_slo_initiated"})
	})

	// Health check
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	return mux
}

// samlMetadata generates a minimal SAML 2.0 IdP metadata XML.
func samlMetadata(entityID, issuer string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata" entityID="%s">
  <IDPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
    <KeyDescriptor use="signing">
      <KeyInfo xmlns="http://www.w3.org/2000/09/xmldsig#">
        <KeyName>%s</KeyName>
      </KeyInfo>
    </KeyDescriptor>
    <NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress</NameIDFormat>
    <SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" Location="%s/saml/sso"/>
    <SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" Location="%s/saml/sso"/>
    <SingleLogoutService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" Location="%s/saml/slo"/>
  </IDPSSODescriptor>
</EntityDescriptor>`, entityID, issuer, issuer, issuer, issuer)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// loadOrCreateKeyProvider loads RSA keys from disk, or generates them if missing.
// This shares the same key files as the Auth Service (configs/rsa_private.pem).
func loadOrCreateKeyProvider(privPath, pubPath string) (*keyProvider, error) {
	priv, err := loadOrCreatePrivateKey(privPath)
	if err != nil {
		return nil, fmt.Errorf("private key: %w", err)
	}
	pub := &priv.PublicKey
	kid := computeKID(pub)

	// Try to load public key from file for JWKS verification.
	if _, err := os.Stat(pubPath); err == nil {
		loadedPub, err := loadPublicKey(pubPath)
		if err == nil {
			pub = loadedPub
		}
	}

	return &keyProvider{priv: priv, pub: pub, kid: kid}, nil
}

// loadOrCreatePrivateKey mirrors Auth Service's pattern: load if exists, generate if not.
func loadOrCreatePrivateKey(path string) (*rsa.PrivateKey, error) {
	if data, err := os.ReadFile(path); err == nil {
		return parsePrivateKeyPEM(data)
	}
	// Generate new 2048-bit RSA key.
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	// Write to disk so Auth Service can also read it.
	_ = os.MkdirAll("configs", 0o700)
	data := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return nil, fmt.Errorf("write private key: %w", err)
	}
	// Also write the public key for JWKS.
	pubPath := path
	if len(pubPath) > len("private.pem") && pubPath[len(pubPath)-11:] == "private.pem" {
		pubPath = pubPath[:len(pubPath)-11] + "public.pem"
	} else {
		pubPath = "configs/rsa_public.pem"
	}
	pubBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("marshal public key: %w", err)
	}
	pubData := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	})
	_ = os.WriteFile(pubPath, pubData, 0o644)
	log.Printf("Generated new RSA key pair: %s + %s", path, pubPath)
	return key, nil
}

func parsePrivateKeyPEM(data []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		keyAny, err2 := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err2 != nil {
			return nil, fmt.Errorf("parse private key: %w (pkcs8: %v)", err, err2)
		}
		rsaKey, ok := keyAny.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("key is not RSA")
		}
		return rsaKey, nil
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
		// Try PKCS1
		rsaPub, err2 := x509.ParsePKCS1PublicKey(block.Bytes)
		if err2 != nil {
			return nil, fmt.Errorf("parse public key: %w", err)
		}
		return rsaPub, nil
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("key is not RSA")
	}
	return rsaPub, nil
}

func computeKID(pub *rsa.PublicKey) string {
	data, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return ""
	}
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h[:8])
}

// Suppress unused imports.
var (
	_ = domain.ClientTypeConfidential
	_ = big.NewInt
)