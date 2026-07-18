// Package server wires up and runs the OAuth/OIDC HTTP server.
package server

import (
	"bytes"
	"context"
	stdcrypto "crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"compress/flate"
	"fmt"
	"log"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ggid/ggid/pkg/audit"
	"github.com/ggid/ggid/pkg/middleware"
	"github.com/ggid/ggid/pkg/crypto"
	"github.com/ggid/ggid/pkg/saml"
	"github.com/ggid/ggid/pkg/tenant"
	"github.com/ggid/ggid/services/oauth/internal/conf"
	"github.com/ggid/ggid/services/oauth/internal/domain"
	"github.com/ggid/ggid/services/oauth/internal/repository"
	"github.com/ggid/ggid/services/oauth/internal/service"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

// Server encapsulates the OAuth HTTP server.
type Server struct {
	cfg           *conf.Config
	httpSrv      *http.Server
	oauthSvc     *service.OAuthService
	pool         *pgxpool.Pool
	stopTicker   func()
	rotatingKP   *service.RotatingKeyProvider
	auditPub     *audit.Publisher
	mapRepo      *oauthMapRepo
}

func (s *Server) SetMapRepo(repo *oauthMapRepo) {
	s.mapRepo = repo
}

// keyProvider implements pkg/crypto.KeyProvider by loading RSA keys from disk.
type keyProvider struct {
	priv *rsa.PrivateKey
	pub  *rsa.PublicKey
	kid  string
}

func (kp *keyProvider) Metadata() crypto.KeyMetadata {
	return crypto.KeyMetadata{
		KeyID:     kp.kid,
		Algorithm: crypto.RS256,
		Use:       "sig",
	}
}
func (kp *keyProvider) Public() stdcrypto.PublicKey   { return kp.pub }
func (kp *keyProvider) Signer() stdcrypto.Signer        { return kp.priv }
func (kp *keyProvider) Close() error               { return nil }

// New constructs and wires up the OAuth server using a local key provider by default.
func New(cfg *conf.Config) (*Server, error) {
	// Load or create RSA keys — shares same paths as Auth Service.
	kp, err := loadOrCreateKeyProvider(cfg.PrivateKeyPath, cfg.PublicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("load keys: %w", err)
	}
	log.Printf("OAuth key loaded (kid=%s)", kp.Metadata().KeyID)

	return NewWithKeyProvider(cfg, kp)
}

// NewWithKeyProvider constructs and wires up the OAuth server using the supplied KeyProvider.
func NewWithKeyProvider(cfg *conf.Config, kp crypto.KeyProvider) (*Server, error) {
	ctx := context.Background()

	var rotatingKP *service.RotatingKeyProvider
	stopTicker := func() {}

	// Wrap local RSA keys in a RotatingKeyProvider for automatic rotation.
	// HSM/KMS providers cannot be rotated in-process, so use them directly.
	if rsaPriv, ok := kp.Signer().(*rsa.PrivateKey); ok {
		rotatingKP = service.NewRotatingKeyProvider(rsaPriv, 24*time.Hour)
		stopTicker = rotatingKP.StartRotationTicker(24 * time.Hour)
		log.Printf("OAuth key rotation enabled (24h interval, 24h grace period)")
		kp = rotatingKP
	} else {
		log.Printf("OAuth key provider is not a local RSA key; rotation disabled")
	}


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
				// Initialize persistent stores (PG-first, in-memory fallback)
				brandingAdapterVar = newBrandingAdapter(pool)
				parAdapterVar = newPARAdapter(pool)
				dpopAdapterVar = newDPoPAdapter(pool)
				delegationAdapterVar = newDelegationAdapter(pool)
				reviewAdapterVar = newReviewAdapter(pool)
				scopeLifecycleAdapterVar = newScopeLifecycleAdapter(pool)
				// Initialize map repo for remaining in-memory stores.
				mapRepoVar = newOAuthMapRepo(pool)
				if err := mapRepoVar.EnsureSchema(ctx); err != nil {
					log.Printf("warning: oauth map repo schema error: %v", err)
				}
				log.Println("OAuth database connected")
			} else if err != nil {
				log.Printf("warning: failed to connect database: %v (running without DB)", err)
			} else {
				log.Printf("warning: database ping failed (running without DB)")
				p.Close()
			}
		}
	}

	// Fallback to in-memory repos when DB is not connected.
	if clientRepo == nil {
		log.Println("OAuth: using in-memory client repository (no DB)")
		clientRepo = repository.NewMemoryClientRepository()
	}
	if codeRepo == nil {
		codeRepo = repository.NewMemoryCodeRepository()
	}
	if tokenRepo == nil {
		tokenRepo = repository.NewMemoryIDTokenRepository()
	}

	// Create the OAuth service with rotating key provider.
	oauthSvc := service.NewOAuthService(clientRepo, codeRepo, tokenRepo, kp, cfg.Issuer)

	// Initialize Redis client for refresh token lookup (shared with Auth service).
	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		opts, err := redis.ParseURL(redisURL)
		if err == nil {
			rdb := redis.NewClient(opts)
			if err := rdb.Ping(ctx).Err(); err == nil {
				oauthSvc.SetRedisClient(&redisAdapter{rdb: rdb})
				log.Println("OAuth Redis connected for refresh token lookup")
			} else {
				log.Printf("warning: Redis ping failed: %v (refresh token fallback disabled)", err)
			}
		} else {
			log.Printf("warning: invalid REDIS_URL: %v", err)
		}
	}

	// Initialize audit publisher before building handler so it's available in routes.
	var auditPub *audit.Publisher
	if natsURL := os.Getenv("NATS_URL"); natsURL != "" {
		if pub, err := audit.NewPublisher(context.Background(), natsURL); err == nil {
			auditPub = pub
			log.Println("OAuth: audit publisher connected to NATS")
		} else {
			log.Printf("OAuth: audit publisher disabled (%v)", err)
		}
	}

	// Build HTTP handler.
	trustValidator := NewTrustChainValidator(pool)
	handler := buildHandler(oauthSvc, cfg, rotatingKP, auditPub, trustValidator)

	// Wrap with panic recovery so a single bad request cannot crash the process.
	wrappedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rvr := recover(); rvr != nil {
				slog.Error("PANIC recovered in oauth handler", "error", rvr)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": "internal server error"})
			}
		}()
		handler.ServeHTTP(w, r)
	})

	mwSecret, mwPrevSecret := middleware.LoadInternalSecrets()
	protectedHandler := middleware.InternalAuthPathOnly(middleware.InternalAuthConfig{
		Secret:     mwSecret,
		PrevSecret: mwPrevSecret,
	})(wrappedHandler)

	httpSrv := &http.Server{
		Addr:         cfg.HTTP.Addr,
		Handler:      protectedHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// Initialize OAuth map repo for in-memory stores migration.
	var mapRepo *oauthMapRepo
	if pool != nil {
		mapRepo = newOAuthMapRepo(pool)
		if err := mapRepo.EnsureSchema(ctx); err != nil {
			log.Printf("warning: oauth map repo schema error: %v", err)
		}
	}

	return &Server{cfg: cfg, httpSrv: httpSrv, oauthSvc: oauthSvc, pool: pool, stopTicker: stopTicker, rotatingKP: rotatingKP, auditPub: auditPub, mapRepo: mapRepo}, nil
}

// Run starts the server and blocks until ctx is cancelled.
func (s *Server) Run(ctx context.Context) error {
	lis, err := net.Listen("tcp", s.cfg.HTTP.Addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	errCh := make(chan error, 2)
	go func() {
		log.Printf("OAuth/OIDC server listening on %s", s.cfg.HTTP.Addr)
		if err := s.httpSrv.Serve(lis); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("http serve: %w", err)
		}
	}()

	// Start gRPC server if address is configured.
	if grpcAddr := os.Getenv("OAUTH_GRPC_ADDR"); grpcAddr != "" {
		_, _, gerr := s.startGRPCServer(grpcAddr)
		if gerr != nil {
			log.Printf("OAuth gRPC: failed to start on %s: %v (continuing HTTP-only)", grpcAddr, gerr)
		}
	}

	select {
	case <-ctx.Done():
		log.Println("OAuth server shutting down...")
		if s.stopTicker != nil {
			s.stopTicker()
		}
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
func buildHandler(oauthSvc *service.OAuthService, cfg *conf.Config, rotatingKP *service.RotatingKeyProvider, auditPub *audit.Publisher, trustValidator *TrustChainValidator) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	// OIDC Discovery (both prefixed and non-prefixed for gateway compatibility)
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		config := oauthSvc.GetDiscoveryConfig()
		overrideDiscoveryIssuer(config, r)
		writeJSON(w, http.StatusOK, config)
	})
	mux.HandleFunc("/api/v1/oauth/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		config := oauthSvc.GetDiscoveryConfig()
		overrideDiscoveryIssuer(config, r)
		writeJSON(w, http.StatusOK, config)
	})

	// JWKS (both prefixed and non-prefixed for gateway compatibility)
	mux.HandleFunc("/oauth/jwks", func(w http.ResponseWriter, r *http.Request) {
		jwks := oauthSvc.GetJWKS()
		writeJSON(w, http.StatusOK, jwks)
	})
	mux.HandleFunc("/api/v1/oauth/jwks", func(w http.ResponseWriter, r *http.Request) {
		jwks := oauthSvc.GetJWKS()
		writeJSON(w, http.StatusOK, jwks)
	})

	// Prefixed aliases for gateway: re-dispatch to non-prefixed handlers
	mux.HandleFunc("/api/v1/oauth/authorize", func(w http.ResponseWriter, r *http.Request) {
		r2 := r.Clone(r.Context())
		r2.URL.Path = "/oauth/authorize"
		mux.ServeHTTP(w, r2)
	})
	mux.HandleFunc("/api/v1/oauth/token", func(w http.ResponseWriter, r *http.Request) {
		r2 := r.Clone(r.Context())
		r2.URL.Path = "/oauth/token"
		mux.ServeHTTP(w, r2)
	})
	mux.HandleFunc("/api/v1/oauth/userinfo", func(w http.ResponseWriter, r *http.Request) {
		r2 := r.Clone(r.Context())
		r2.URL.Path = "/oauth/userinfo"
		mux.ServeHTTP(w, r2)
	})

	// Authorize endpoint (GET/POST — creates auth code, redirects)
	mux.HandleFunc("/oauth/authorize", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}

		clientID := r.URL.Query().Get("client_id")
		redirectURI := r.URL.Query().Get("redirect_uri")
		responseType := r.URL.Query().Get("response_type")
		state := r.URL.Query().Get("state")
		scopeParam := r.URL.Query().Get("scope")
		nonce := r.URL.Query().Get("nonce")
		codeChallenge := r.URL.Query().Get("code_challenge")
		codeChallengeMethod := r.URL.Query().Get("code_challenge_method")
		acrValues := r.URL.Query().Get("acr_values") // NIST 800-63-3 requested AAL level

		if clientID == "" || redirectURI == "" || responseType == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request", "error_description": "client_id, redirect_uri, and response_type are required"})
			return
		}

		if responseType != "code" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported_response_type"})
			return
		}

		// PKCE enforcement: if RequirePKCE is enabled, or the client is public
		// (no client_secret), code_challenge is mandatory.
		if cfg.RequirePKCE && codeChallenge == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error":             "invalid_request",
				"error_description": "PKCE is required: code_challenge parameter is mandatory",
			})
			return
		}
		// Validate S256 method is used when challenge is provided.
		if codeChallenge != "" && codeChallengeMethod != "" && codeChallengeMethod != "S256" && codeChallengeMethod != "plain" {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error":             "invalid_request",
				"error_description": "unsupported code_challenge_method: use S256 or plain",
			})
			return
		}

		// Inject tenant context from header or query param (public endpoint).
		tenantIDStr := r.Header.Get("X-Tenant-ID")
		if tenantIDStr == "" {
			tenantIDStr = r.URL.Query().Get("tenant_id")
		}
		tenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request", "error_description": "valid X-Tenant-ID header or tenant_id query param required"})
			return
		}

		// Federation TrustChainValidator: reject untrusted OIDC federation clients.
		if trustValidator != nil {
			if err := trustValidator.ValidateOIDCClient(r.Context(), tenantIDStr, clientID); err != nil {
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "untrusted_federation_client", "detail": err.Error()})
				return
			}
		}

		// The user must be authenticated (via JWT).
		userIDStr := r.URL.Query().Get("user_id")
		if userIDStr == "" {
			userIDStr = r.Header.Get("X-User-ID")
		}
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			// Show built-in login page (not the admin console)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `<!DOCTYPE html>
	<html lang="zh-CN">
	<head>
	<meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
	<title>GGID 登录</title>
	<style>
	*{margin:0;padding:0;box-sizing:border-box}
	body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;background:linear-gradient(135deg,#667eea 0%%,#764ba2 100%%);min-height:100vh;display:flex;align-items:center;justify-content:center}
	.card{background:#fff;border-radius:12px;padding:40px;width:400px;max-width:90vw;box-shadow:0 20px 60px rgba(0,0,0,.15)}
	h1{text-align:center;color:#1677ff;margin-bottom:8px;font-size:24px}
	.sub{text-align:center;color:#999;margin-bottom:24px;font-size:14px}
	label{display:block;margin-bottom:6px;font-weight:600;font-size:14px;color:#333}
	input{width:100%%;padding:12px 14px;border:1px solid #ddd;border-radius:8px;font-size:15px;margin-bottom:16px;transition:border .2s}
	input:focus{outline:none;border-color:#1677ff;box-shadow:0 0 0 2px rgba(22,119,255,.1)}
	button{width:100%%;padding:14px;background:#1677ff;color:#fff;border:none;border-radius:8px;font-size:16px;font-weight:600;cursor:pointer;transition:background .2s}
	button:hover{background:#0958d9}
	button:disabled{background:#ccc;cursor:not-allowed}
	.err{color:#ff4d4f;font-size:13px;margin-bottom:12px;display:none}
	.redirect-info{margin-top:16px;padding:10px;background:#f6f8fa;border-radius:6px;font-size:12px;color:#666;text-align:center}
	</style>
	</head>
	<body>
	<div class="card">
	<h1>GGID 登录</h1>
	<p class="sub">使用您的账户登录以继续</p>
	<form id="loginForm">
	<div id="err" class="err"></div>
	<label>用户名</label>
	<input id="username" type="text" required autocomplete="username" placeholder="输入用户名">
	<label>密码</label>
	<input id="password" type="password" required autocomplete="current-password" placeholder="输入密码">
	<button type="submit" id="btn">登录</button>
	</form>
	<div class="redirect-info">授权完成后将返回 <strong id="app-name">应用</strong></div>
	</div>
	<script>
	document.getElementById('loginForm').addEventListener('submit', async (e) => {
	  e.preventDefault();
	  const btn = document.getElementById('btn');
	  const errDiv = document.getElementById('err');
	  btn.disabled = true; btn.textContent = '登录中...';
	  errDiv.style.display = 'none';
	  try {
    const resp = await fetch('%s/api/v1/auth/login', {
      method: 'POST',
      headers: {'Content-Type': 'application/json', 'X-Tenant-ID': '%s'},
      body: JSON.stringify({username: document.getElementById('username').value, password: document.getElementById('password').value, tenant_id: '%s'})
    });
	    const data = await resp.json();
	    if (!resp.ok) { errDiv.textContent = data.error?.message || data.error || '登录失败'; errDiv.style.display = 'block'; btn.disabled = false; btn.textContent = '登录'; return; }
	    // Extract user_id from JWT and redirect back to authorize
	    const payload = JSON.parse(atob(data.access_token.split('.')[1]));
	    const url = new URL(window.location.href);
	    const authorizeURL = url.pathname + url.search;
	    const sep = authorizeURL.includes('?') ? '&' : '?';
	    window.location.href = authorizeURL + sep + 'user_id=' + payload.sub;
	  } catch (err) {
	    errDiv.textContent = '网络错误，请重试'; errDiv.style.display = 'block';
	    btn.disabled = false; btn.textContent = '登录';
	  }
	});
	</script>
	</body>
	</html>`, os.Getenv("GGID_URL"), tenantID.String(), tenantID.String())
			return
		}

		ctx := tenant.WithContext(r.Context(), &tenant.Context{
			TenantID:       tenantID,
			IsolationLevel: tenant.IsolationShared,
		})

		client, err := oauthSvc.GetClient(ctx, clientID)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_client", "error_description": "client not found"})
			return
		}
		if err := enforceFAPIAuthorize(client, r); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request", "error_description": err.Error()})
			return
		}

		scopes := []string{}
		if scopeParam != "" {
			scopes = strings.Split(scopeParam, " ")
		}

		// Consent screen: if client requests non-basic scopes and user hasn't
		// explicitly consented, return consent_required response.
		basicScopes := map[string]bool{"openid": true, "profile": true, "email": true, "offline_access": true}
		hasExtendedScope := false
		for _, s := range scopes {
			if !basicScopes[s] {
				hasExtendedScope = true
				break
			}
		}
		consentGiven := r.URL.Query().Get("consent") == "true"
		if hasExtendedScope && !consentGiven {
			writeJSON(w, http.StatusOK, map[string]any{
				"status":           "consent_required",
				"client_id":        clientID,
				"requested_scopes": scopes,
				"state":            state,
				"message":          "User consent is required for the requested scopes.",
				"consent_url":      "/oauth/authorize?consent=true&client_id=" + clientID + "&redirect_uri=" + redirectURI + "&response_type=code&scope=" + scopeParam + "&user_id=" + userIDStr,
			})
			return
		}

		// RAR: read authorization_details parameter (RFC 9396).
		authDetailsJSON := json.RawMessage(nil)
		if ad := r.URL.Query().Get("authorization_details"); ad != "" {
			// Validate it's valid JSON array.
			var parsed []any
			if err := json.Unmarshal([]byte(ad), &parsed); err == nil {
				authDetailsJSON = json.RawMessage(ad)
			}
		}
		// Also accept from form POST body.
		if len(authDetailsJSON) == 0 && r.Method == http.MethodPost {
			if ad := r.FormValue("authorization_details"); ad != "" {
				var parsed []any
				if err := json.Unmarshal([]byte(ad), &parsed); err == nil {
					authDetailsJSON = json.RawMessage(ad)
				}
			}
		}

		code, err := oauthSvc.CreateAuthorizationCode(ctx, &service.AuthorizeRequest{
			TenantID:             tenantID,
			ClientID:             clientID,
			RedirectURI:          redirectURI,
			ResponseType:         responseType,
			Scope:                scopes,
			State:                state,
			Nonce:                nonce,
			CodeChallenge:        codeChallenge,
			CodeChallengeMethod:  codeChallengeMethod,
			UserID:               userID,
			AuthorizationDetails: authDetailsJSON,
			RequestedACR:          acrValues,
		})
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request", "error_description": err.Error()})
			return
		}

		// Build redirect URL with code and state.
		redirectURL := redirectURI
		sep := "?"
		if strings.Contains(redirectURL, "?") {
			sep = "&"
		}
		redirectURL += sep + "code=" + code
		if state != "" {
			redirectURL += "&state=" + state
		}

		// HTTP 302 redirect to the client's redirect_uri with code and state
		http.Redirect(w, r, redirectURL, http.StatusFound)
	})

	// Token endpoint (POST)
	mux.HandleFunc("/oauth/token", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		_ = r.ParseForm()

		// Inject tenant context from header.
		tenantIDStr := r.Header.Get("X-Tenant-ID")
		tenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request", "error_description": "valid X-Tenant-ID header required"})
			return
		}

		ctx := tenant.WithContext(r.Context(), &tenant.Context{
			TenantID:       tenantID,
			IsolationLevel: tenant.IsolationShared,
		})

		grantType := r.FormValue("grant_type")
		clientID := r.FormValue("client_id")
		clientSecret := r.FormValue("client_secret")
		scopeParam := r.FormValue("scope")
		scopes := []string{}
		if scopeParam != "" {
			scopes = strings.Split(scopeParam, " ")
		}

		client, err := oauthSvc.GetClient(ctx, clientID)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_client", "error_description": "client not found"})
			return
		}
		if err := enforceFAPIToken(client, r); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request", "error_description": err.Error()})
			return
		}

		var resp *service.TokenResponse
		var tokenErr error

		switch grantType {
		case "authorization_code":
			resp, tokenErr = oauthSvc.ExchangeAuthorizationCode(ctx, &service.TokenExchangeRequest{
				TenantID:     tenantID,
				GrantType:    grantType,
				Code:         r.FormValue("code"),
				RedirectURI:  r.FormValue("redirect_uri"),
				ClientID:     clientID,
				ClientSecret: clientSecret,
				CodeVerifier: r.FormValue("code_verifier"),
			})
		case "refresh_token":
			resp, tokenErr = oauthSvc.RefreshToken(ctx, &service.RefreshTokenRequest{
				TenantID:     tenantID,
				RefreshToken: r.FormValue("refresh_token"),
				ClientID:     clientID,
				ClientSecret: clientSecret,
				Scope:        scopes,
			})
		case "client_credentials":
			resp, tokenErr = oauthSvc.ClientCredentials(ctx, &service.ClientCredentialsRequest{
				TenantID:     tenantID,
				ClientID:     clientID,
				ClientSecret: clientSecret,
				Scope:        scopes,
			})
		case "urn:ietf:params:oauth:grant-type:device_code":
			resp, tokenErr = oauthSvc.PollDeviceToken(ctx, r.FormValue("device_code"), clientID)
			if tokenErr != nil {
				// RFC 8628 uses specific error codes for polling.
				errMsg := tokenErr.Error()
				switch errMsg {
				case "authorization_pending":
					writeJSON(w, http.StatusBadRequest, map[string]string{"error": "authorization_pending"})
				case "slow_down":
					writeJSON(w, http.StatusBadRequest, map[string]string{"error": "slow_down"})
				case "expired_token":
					writeJSON(w, http.StatusBadRequest, map[string]string{"error": "expired_token"})
				case "access_denied":
					writeJSON(w, http.StatusBadRequest, map[string]string{"error": "access_denied"})
				default:
					writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_grant", "error_description": errMsg})
				}
				return
			}
		case "urn:ietf:params:oauth:grant-type:jwt-bearer":
			// RFC 7523: JWT bearer assertion grant.
			assertion := r.FormValue("assertion")
			if assertion == "" {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request", "error_description": "assertion parameter is required for jwt-bearer grant"})
				return
			}
			resp, tokenErr = oauthSvc.JWTBearerGrant(ctx, &service.JWTBearerRequest{
				TenantID:  tenantID,
				Assertion: assertion,
				Scope:     scopes,
				Issuer:    cfg.Issuer,
			})
		default:
			// RFC 8693 Token Exchange.
			if grantType == "urn:ietf:params:oauth:grant-type:token-exchange" {
				resp, tokenErr = oauthSvc.ExchangeTokenRFC8693(ctx, &service.RFC8693ExchangeRequest{
					TenantID:         tenantID,
					ClientID:         clientID,
					SubjectToken:     r.FormValue("subject_token"),
					SubjectTokenType: r.FormValue("subject_token_type"),
					ActorToken:       r.FormValue("actor_token"),
					ActorTokenType:   r.FormValue("actor_token_type"),
					Scope:            scopes,
					Resource:         r.FormValue("resource"),
				})
				if tokenErr == nil {
					// RFC 8693 requires issued_token_type in response.
					w.Header().Set("Cache-Control", "no-store")
					w.Header().Set("Pragma", "no-cache")
					writeJSON(w, http.StatusOK, map[string]any{
						"access_token":      resp.AccessToken,
						"issued_token_type": "urn:ietf:params:oauth:token-type:access_token",
						"token_type":        "Bearer",
						"expires_in":        resp.ExpiresIn,
						"scope":             resp.Scope,
					})
					return
				}
			} else {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported_grant_type"})
				return
			}
		}

		if tokenErr != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_grant", "error_description": tokenErr.Error()})
			return
		}

		// DPoP proof verification (RFC 9449): if the client sends a DPoP header,
		// validate the proof and bind the issued access token to the DPoP key.
		dpopHeader := r.Header.Get("DPoP")
		if dpopHeader != "" {
			proof, err := service.ParseDPoPHeader(dpopHeader, "POST", r.URL.String())
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{
					"error":             "invalid_dpop_proof",
					"error_description": "DPoP proof validation failed: " + err.Error(),
				})
				return
			}
			// Validate htm/htu match the token endpoint and check freshness (prevent replay).
			if proof.JTI == "" || time.Now().UTC().Sub(proof.IssuedAt) > 5*time.Minute {
				writeJSON(w, http.StatusBadRequest, map[string]string{
					"error":             "invalid_dpop_proof",
					"error_description": "DPoP proof expired or missing nonce",
				})
				return
			}
			// Bind the access token to the DPoP key thumbprint.
			jkt := computeKeyThumbprint(proof.PublicKey)
			if resp != nil && resp.AccessToken != "" {
				BindTokenToDPoP(resp.AccessToken, jkt)
				resp.TokenType = "DPoP"
			}
		}

		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Pragma", "no-cache")

		// Publish audit event for token issuance.
		if resp != nil && auditPub != nil {
			event := audit.NewEvent("token_issued", "success", tenantID, uuid.Nil)
			event.ResourceType = "oauth_token"
			auditPub.PublishAsync(event)
		}

		writeJSON(w, http.StatusOK, resp)
	})

	// UserInfo endpoint (GET)
	mux.HandleFunc("/oauth/userinfo", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}

		// Extract Bearer token from Authorization header.
		authHeader := r.Header.Get("Authorization")
		token := extractBearerToken(authHeader)
		if token == "" {
			w.Header().Set("WWW-Authenticate", "Bearer")
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_token", "error_description": "bearer token required"})
			return
		}

		userInfo, err := oauthSvc.GetUserInfo(token)
		if err != nil {
			w.Header().Set("WWW-Authenticate", "Bearer error=\"invalid_token\"")
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_token", "error_description": err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, userInfo)
	})

	// Back-channel logout (OIDC Session Management 1.0)
	mux.HandleFunc("/oauth/logout", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		_ = r.ParseForm()

		logoutToken := r.FormValue("logout_token")
		if logoutToken == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "logout_token is required"})
			return
		}

		// Parse the logout token to extract sub (user ID) and sid (session ID).
		claims, err := oauthSvc.ParseAccessToken(logoutToken)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid logout token"})
			return
		}

		// Revoke the token.
		_ = oauthSvc.RevokeToken(logoutToken)

		// Extract session ID for back-channel notification.
		sub := ""
		sid := ""
		if v, ok := claims["sub"]; ok {
			if s, ok := v.(string); ok {
				sub = s
			}
		}
		if v, ok := claims["sid"]; ok {
			if s, ok := v.(string); ok {
				sid = s
			}
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"status":  "logged_out",
			"sub":     sub,
			"sid":     sid,
		})
	})

	// Token revocation (RFC 7009)
	mux.HandleFunc("/oauth/revoke", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		_ = r.ParseForm()
		token := r.FormValue("token")
		tokenTypeHint := r.FormValue("token_type_hint")
		_ = oauthSvc.RevokeToken(token, tokenTypeHint)
		w.WriteHeader(http.StatusOK)
	})

	// Token revocation route alias (RFC 7009)
	mux.HandleFunc("/api/v1/oauth/revoke", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		_ = r.ParseForm()
		token := r.FormValue("token")
		tokenTypeHint := r.FormValue("token_type_hint")
		_ = oauthSvc.RevokeToken(token, tokenTypeHint)
		w.WriteHeader(http.StatusOK)
	})

	// Back-channel logout (OIDC Back-Channel Logout)
	mux.HandleFunc("/api/v1/oauth/backchannel-logout", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		_ = r.ParseForm()

		logoutToken := r.FormValue("logout_token")
		if logoutToken == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request", "error_description": "logout_token is required"})
			return
		}

		// Parse the logout token (JWT) to extract sub/sid for session cleanup.
		claims, err := oauthSvc.ParseBackchannelLogoutToken(logoutToken)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_logout_token", "error_description": err.Error()})
			return
		}

		// Revoke all tokens for this subject/session.
		sub, _ := claims["sub"].(string)
		if sub != "" {
			oauthSvc.BackchannelLogout(sub)
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "logged_out"})
	})

	// Dynamic client registration route alias (RFC 7591)
	mux.HandleFunc("/api/v1/oauth/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		var req service.DynamicRegistrationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}
		// Inject tenant context from header or query param.
		ctx, err := injectTenantContext(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request", "error_description": err.Error()})
			return
		}
		result, err := oauthSvc.DynamicClientRegister(ctx, &req)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, result)
	})

	// Token introspection — requires client authentication per RFC 7662 §2.1
	mux.HandleFunc("/oauth/introspect", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		_ = r.ParseForm()

		// RFC 7662 §2.1: introspection endpoint MUST require client authentication.
		// Supported methods: HTTP Basic, form-encoded client credentials,
		// or Bearer token (RFC 6750) as an alternative.
		if !isClientAuthenticated(r) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_client"})
			return
		}

		token := r.FormValue("token")
		if token == "" {
			writeJSON(w, http.StatusOK, map[string]bool{"active": false})
			return
		}

		result := oauthSvc.IntrospectToken(token)
		writeJSON(w, http.StatusOK, result)
	})
	mux.HandleFunc("/api/v1/oauth/introspect", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		_ = r.ParseForm()

		// RFC 7662 §2.1: introspection endpoint MUST require client authentication.
		// Supported methods: HTTP Basic, form-encoded client credentials,
		// or Bearer token (RFC 6750) as an alternative.
		if !isClientAuthenticated(r) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_client"})
			return
		}

		token := r.FormValue("token")
		if token == "" {
			writeJSON(w, http.StatusOK, map[string]bool{"active": false})
			return
		}
		result := oauthSvc.IntrospectToken(token)
		writeJSON(w, http.StatusOK, result)
	})

	// Dynamic Client Registration (RFC 7591)
	mux.HandleFunc("/oauth/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		var req service.DynamicRegistrationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}
		// Inject tenant context from header or query param.
		ctx, err := injectTenantContext(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request", "error_description": err.Error()})
			return
		}
		result, err := oauthSvc.DynamicClientRegister(ctx, &req)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, result)
	})

	// OAuth Consent Screen
	mux.HandleFunc("/oauth/consent", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}

		clientID := r.URL.Query().Get("client_id")
		scopeParam := r.URL.Query().Get("scope")
		redirectURI := r.URL.Query().Get("redirect_uri")
		state := r.URL.Query().Get("state")

		if r.Method == http.MethodPost {
			_ = r.ParseForm()
			decision := r.FormValue("decision")
			if decision == "approve" {
				authURL := "/oauth/authorize?consent=true&client_id=" + clientID + "&redirect_uri=" + redirectURI + "&response_type=code&scope=" + scopeParam + "&state=" + state
				writeJSON(w, http.StatusOK, map[string]string{
					"status":       "approved",
					"redirect_url": authURL,
				})
			} else {
				writeJSON(w, http.StatusOK, map[string]string{
					"status":       "denied",
					"redirect_url": redirectURI + "?error=access_denied&state=" + state,
				})
			}
			return
		}

		var scopes []string
		if scopeParam != "" {
			scopes = strings.Split(scopeParam, " ")
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"status":       "consent_required",
			"client_id":    clientID,
			"scopes":       scopes,
			"redirect_uri": redirectURI,
			"state":        state,
			"message":      "Review and approve the requested permissions",
		})
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
		_ = r.ParseForm()
		samlResponseB64 := r.FormValue("SAMLResponse")
		if samlResponseB64 == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing SAMLResponse"})
			return
		}

		rawXML, err := base64.StdEncoding.DecodeString(samlResponseB64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid base64 encoding"})
			return
		}

		// Federation TrustChainValidator: reject untrusted IdPs before parsing assertion.
		if trustValidator != nil {
			samlIssuer := extractSAMLIssuer(rawXML)
			if samlIssuer != "" {
				tenantIDStr := r.Header.Get("X-Tenant-ID")
				if err := trustValidator.ValidateSAMLIssuer(r.Context(), tenantIDStr, samlIssuer); err != nil {
					writeJSON(w, http.StatusForbidden, map[string]string{"error": "untrusted_saml_issuer", "detail": err.Error()})
					return
				}
			}
		}

		assertion, err := saml.ParseAssertion(rawXML)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "failed to parse SAML assertion", "detail": err.Error()})
			return
		}

		if err := assertion.ValidateConditions(); err != nil {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "assertion validation failed", "detail": err.Error()})
			return
		}

		attrs := saml.ExtractAttributes(assertion)
		nameID := assertion.Subject.NameID

		email := nameID
		if vals, ok := attrs["mail"]; ok && len(vals) > 0 {
			email = vals[0]
		}
		displayName := ""
		if vals, ok := attrs["displayName"]; ok && len(vals) > 0 {
			displayName = vals[0]
		}

		tenantIDStr := r.Header.Get("X-Tenant-ID")
		tenantID, _ := uuid.Parse(tenantIDStr)

		accessToken, expiresIn, err := oauthSvc.IssueSAMLToken(tenantID, nameID, email, displayName)
		if err != nil {
			writeInternalError(w, "IssueSAMLToken", err)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"access_token": accessToken,
			"token_type":   "Bearer",
			"expires_in":   expiresIn,
			"saml_subject": nameID,
			"email":        email,
			"name":         displayName,
		})
	})

	mux.HandleFunc("/saml/sso", func(w http.ResponseWriter, r *http.Request) {
		// SP-initiated SSO: generate a SAML AuthnRequest and redirect to IdP.
		entityID := cfg.Issuer + "/saml/metadata"
		acsURL := cfg.Issuer + "/saml/acs"
		idpSSOURL := r.URL.Query().Get("idp")
		if idpSSOURL == "" {
			idpSSOURL = cfg.Issuer + "/saml/idp/sso"
		}

		sp := &saml.ServiceProvider{
			EntityID:             entityID,
			ACSURL:               acsURL,
			WantAssertionsSigned: true,
		}

		authnReq := saml.BuildAuthnRequest(sp, idpSSOURL)
		encoded, err := authnReq.EncodeForRedirect()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to encode SAML AuthnRequest"})
			return
		}

		redirectURL := idpSSOURL + "?SAMLRequest=" + encoded
		http.Redirect(w, r, redirectURL, http.StatusFound)
	})

	mux.HandleFunc("/saml/slo", func(w http.ResponseWriter, r *http.Request) {
		// SLO: process LogoutRequest or LogoutResponse
		_ = r.ParseForm()
		samlRequest := r.FormValue("SAMLRequest")
		samlResponse := r.FormValue("SAMLResponse")

		if samlRequest != "" {
			// This is a LogoutRequest from an SP — return LogoutResponse
			writeJSON(w, http.StatusOK, map[string]string{
				"status":  "success",
				"message": "logout processed",
			})
			return
		}
		if samlResponse != "" {
			// This is a LogoutResponse from an SP
			writeJSON(w, http.StatusOK, map[string]string{
				"status":  "success",
				"message": "logout confirmed",
			})
			return
		}
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing SAMLRequest or SAMLResponse"})
	})

	// --- SAML 2.0 IdP endpoints (GGID as Identity Provider) ---

	mux.HandleFunc("/saml/idp/metadata", func(w http.ResponseWriter, r *http.Request) {
		idp := buildIdP(cfg)
		meta, err := idp.GenerateIdPMetadata()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to generate IdP metadata"})
			return
		}
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(meta)
	})

	mux.HandleFunc("/saml/idp/sso", func(w http.ResponseWriter, r *http.Request) {
		// IdP SSO: receive SAMLRequest (AuthnRequest) from an SP, return signed SAML Response
		_ = r.ParseForm()
		samlRequestB64 := r.FormValue("SAMLRequest")
		relayState := r.FormValue("RelayState")
		authHeader := r.Header.Get("Authorization")
		bearerToken := ""
		if strings.HasPrefix(authHeader, "Bearer ") {
			bearerToken = strings.TrimPrefix(authHeader, "Bearer ")
		}
		if bearerToken == "" {
			bearerToken = r.URL.Query().Get("token")
		}

		if samlRequestB64 == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing SAMLRequest"})
			return
		}

		// Decode the AuthnRequest (base64 + deflate)
		rawCompressed, err := base64.StdEncoding.DecodeString(samlRequestB64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid base64"})
			return
		}
		rawReader := bytes.NewReader(rawCompressed)
		flateReader := flate.NewReader(rawReader)
		rawXML, err := io.ReadAll(flateReader)
		flateReader.Close()
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "failed to decompress SAMLRequest"})
			return
		}

		// Parse AuthnRequest to extract SP info
		spEntityID, spACSURL, requestID := parseAuthnRequest(rawXML)

		// If no bearer token, redirect to login with callback
		if bearerToken == "" {
			loginURL := cfg.Issuer + "/api/v1/auth/login?redirect=/saml/idp/sso&SAMLRequest=" + samlRequestB64
			if relayState != "" {
				loginURL += "&RelayState=" + relayState
			}
			http.Redirect(w, r, loginURL, http.StatusFound)
			return
		}

		// Verify the bearer token to get user info
		claims, err := oauthSvc.ParseAccessToken(bearerToken)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid token", "detail": err.Error()})
			return
		}

		// Extract user identity from claims
		nameID := ""
		if sub, ok := claims["sub"].(string); ok {
			nameID = sub
		}
		email := nameID
		if e, ok := claims["email"].(string); ok && e != "" {
			email = e
		}
		displayName := ""
		if dn, ok := claims["name"].(string); ok {
			displayName = dn
		}

		// Build IdP and create signed SAML Response
		idp := buildIdP(cfg)
		respXML, err := idp.BuildSAMLResponse(&saml.SAMLResponseRequest{
			Destination:  spACSURL,
			Audience:     spEntityID,
			NameID:       email,
			NameIDFormat: saml.NameIDFormatEmailAddress,
			Attributes: map[string][]string{
				"email":       {email},
				"displayName": {displayName},
			},
			InResponseTo: requestID,
			RelayState:   relayState,
		})
		if err != nil {
			writeInternalError(w, "BuildSAMLResponse", err)
			return
		}

		// Return SAML Response via HTTP-POST binding
		encoded := saml.EncodeResponseForPOST(respXML)
		html := fmt.Sprintf(`<!DOCTYPE html><html><body onload="document.forms[0].submit()"><form method="POST" action="%s"><input type="hidden" name="SAMLResponse" value="%s"/><input type="hidden" name="RelayState" value="%s"/></form></body></html>`, spACSURL, encoded, relayState)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(html))
	})

	mux.HandleFunc("/saml/idp/slo", func(w http.ResponseWriter, r *http.Request) {
		// IdP SLO: process LogoutRequest/LogoutResponse from SPs
		_ = r.ParseForm()
		samlRequest := r.FormValue("SAMLRequest")
		samlResponse := r.FormValue("SAMLResponse")

		if samlRequest != "" {
			// LogoutRequest from SP — invalidate session, return LogoutResponse
			writeJSON(w, http.StatusOK, map[string]string{
				"status":  "success",
				"message": "IdP logout processed",
			})
			return
		}
		if samlResponse != "" {
			writeJSON(w, http.StatusOK, map[string]string{
				"status":  "success",
				"message": "IdP logout confirmed",
			})
			return
		}
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing SAMLRequest or SAMLResponse"})
	})

	// --- OAuth Client Management REST API ---

	mux.HandleFunc("/api/v1/oauth/clients", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/oauth/clients/scope-matrix" {
			handleScopeMatrix(w, r)
			return
		}
		// Inject tenant context.
		tenantIDStr := r.Header.Get("X-Tenant-ID")
		tenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "valid X-Tenant-ID header required"})
			return
		}

		ctx := tenant.WithContext(r.Context(), &tenant.Context{
			TenantID:       tenantID,
			IsolationLevel: tenant.IsolationShared,
		})

		switch r.Method {
		case http.MethodPost:
			// Register a new OAuth client.
			var body struct {
				Name          string   `json:"name"`
				Type          string   `json:"type"`
				GrantTypes    []string `json:"grant_types"`
				ResponseTypes []string `json:"response_types"`
				RedirectURIs  []string `json:"redirect_uris"`
				Scopes        []string `json:"scopes"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
				return
			}

			clientType := domain.ClientType(body.Type)
			if clientType == "" {
				clientType = domain.ClientTypeConfidential
			}
			if !clientType.IsValid() {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid client type"})
				return
			}

			result, err := oauthSvc.CreateClient(ctx, &service.CreateClientInput{
				TenantID:      tenantID,
				Name:          body.Name,
				Type:          clientType,
				GrantTypes:    body.GrantTypes,
				ResponseTypes: body.ResponseTypes,
				RedirectURIs:  body.RedirectURIs,
				Scopes:        body.Scopes,
			})
			if err != nil {
				writeInternalError(w, "CreateClient", err)
				return
			}

			// Audit: client created
			if auditPub != nil {
				ev := audit.NewEvent("oauth_client.create", "success", tenantID, uuid.Nil)
				ev.ResourceType = "oauth_client"
				if result.Client != nil {
					ev.ResourceID = result.Client.ID
				}
				auditPub.PublishAsync(ev)
			}

			writeJSON(w, http.StatusCreated, result)

		case http.MethodGet:
			// List clients.
			clients, _, err := oauthSvc.ListClients(ctx, 20, 0)
			if err != nil {
				writeInternalError(w, "ListClients", err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"clients": clients})

		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		}
	})

	// Handle individual client operations (GET/DELETE).
	mux.HandleFunc("/api/v1/oauth/clients/", func(w http.ResponseWriter, r *http.Request) {
		tenantIDStr := r.Header.Get("X-Tenant-ID")
		tenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "valid X-Tenant-ID header required"})
			return
		}

		ctx := tenant.WithContext(r.Context(), &tenant.Context{
			TenantID:       tenantID,
			IsolationLevel: tenant.IsolationShared,
		})

		clientID := r.URL.Path[len("/api/v1/oauth/clients/"):]
		if clientID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "client_id is required"})
			return
		}

		// Sub-path: client scopes binding
		if strings.Contains(clientID, "/scopes") {
			handleClientScopes(w, r)
			return
		}
		// Sub-path: client branding
		if strings.HasSuffix(r.URL.Path, "/branding") {
			handleClientBranding(w, r)
			return
		}
		// Sub-path: client versioning
		if strings.HasSuffix(r.URL.Path, "/version") || strings.HasSuffix(r.URL.Path, "/versions") {
			handleClientVersioning(w, r)
			return
		}
		// Sub-path: client health
		if strings.HasSuffix(r.URL.Path, "/health") {
			handleClientHealth(w, r)
			return
		}
		// Sub-path: client lifecycle (suspend/reinstate)
		if strings.HasSuffix(r.URL.Path, "/suspend") || strings.HasSuffix(r.URL.Path, "/reinstate") {
			handleClientLifecycle(w, r)
			return
		}
		// Sub-path: usage policy
		if strings.HasSuffix(r.URL.Path, "/usage-policy") {
			handleUsagePolicy(w, r)
			return
		}
		// Sub-path: deprecation
		if strings.HasSuffix(r.URL.Path, "/deprecation") {
			handleClientDeprecation(w, r)
			return
		}
		// Sub-path: consent screen
		if strings.HasSuffix(r.URL.Path, "/consent-screen") {
			handleConsentScreen(w, r)
			return
		}
		// Sub-path: secret rotation
		if strings.HasSuffix(r.URL.Path, "/rotate-secret") || strings.HasSuffix(r.URL.Path, "/secret-status") {
			handleClientSecretRotation(w, r)
			return
		}
		// Sub-path: analytics
		if strings.HasSuffix(r.URL.Path, "/analytics") {
			handleClientAnalytics(w, r)
			return
		}
		// Sub-path: migrate
		if strings.HasSuffix(r.URL.Path, "/migrate") {
			handleClientMigration(w, r)
			return
		}

		switch r.Method {
		case http.MethodGet:
			client, err := oauthSvc.GetClient(ctx, clientID)
			if err != nil {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "client not found"})
				return
			}
			writeJSON(w, http.StatusOK, client)

		case http.MethodDelete:
			if err := oauthSvc.DeleteClient(ctx, clientID); err != nil {
				writeInternalError(w, "DeleteClient", err)
				return
			}
			// Audit: client deleted
			if auditPub != nil {
				ev := audit.NewEvent("oauth_client.delete", "success", tenantID, uuid.Nil)
				ev.ResourceType = "oauth_client"
				auditPub.PublishAsync(ev)
			}
			writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})

		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		}
	})

	// Device Authorization Flow (RFC 8628)
	// Device authorization (RFC 8628) — both /device and /device_authorization
	mux.HandleFunc("/api/v1/oauth/device", func(w http.ResponseWriter, r *http.Request) {
		r2 := r.Clone(r.Context())
		r2.URL.Path = "/api/v1/oauth/device_authorization"
		mux.ServeHTTP(w, r2)
	})
	mux.HandleFunc("/api/v1/oauth/device_authorization", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		_ = r.ParseForm()

		tenantIDStr := r.Header.Get("X-Tenant-ID")
		if tenantIDStr == "" {
			tenantIDStr = r.FormValue("tenant_id")
		}
		tenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "valid X-Tenant-ID header or tenant_id param required"})
			return
		}

		clientID := r.FormValue("client_id")
		if clientID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "client_id is required"})
			return
		}

		scopeParam := r.FormValue("scope")
		var scopes []string
		if scopeParam != "" {
			scopes = strings.Split(scopeParam, " ")
		}

		resp, err := oauthSvc.CreateDeviceAuthorization(&service.DeviceAuthorizationRequest{
			TenantID: tenantID,
			ClientID: clientID,
			Scope:    scopes,
			Issuer:   cfg.Issuer,
		})
		if err != nil {
			writeInternalError(w, "CreateDeviceAuthorization", err)
			return
		}
		writeJSON(w, http.StatusOK, resp)
	})

	// Device code approval endpoint (user visits verification_uri and enters user_code)
	mux.HandleFunc("/api/v1/oauth/device/approve", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		_ = r.ParseForm()

		userCode := r.FormValue("user_code")
		if userCode == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "user_code is required"})
			return
		}

		userIDStr := r.FormValue("user_id")
		if userIDStr == "" {
			userIDStr = r.Header.Get("X-User-ID")
		}
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "valid user_id or X-User-ID header required"})
			return
		}

		if err := oauthSvc.ApproveDeviceCode(userCode, userID); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "approved"})
	})

	// PAR (RFC 9126) — Pushed Authorization Request
	mux.HandleFunc("/oauth/par", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		if err := r.ParseForm(); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid form data"})
			return
		}
		req := &service.PushedAuthorizationRequest{
			ClientID:            r.Form.Get("client_id"),
			ClientSecret:        r.Form.Get("client_secret"),
			RedirectURI:         r.Form.Get("redirect_uri"),
			ResponseType:        r.Form.Get("response_type"),
			Scope:               r.Form.Get("scope"),
			State:               r.Form.Get("state"),
			Nonce:               r.Form.Get("nonce"),
			CodeChallenge:       r.Form.Get("code_challenge"),
			CodeChallengeMethod: r.Form.Get("code_challenge_method"),
		}
		resp, err := oauthSvc.PushAuthorizationRequest(r.Context(), req)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, resp)
	})

	// Consent management
	mux.HandleFunc("/api/v1/oauth/consent/list", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"consents": []any{}})
	})
	mux.HandleFunc("/api/v1/oauth/consent/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/receipt") {
			handleConsentReceipt(w, r)
			return
		}
		if r.Method != http.MethodDelete {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
	})

	// Introspection cache config
	mux.HandleFunc("/api/v1/oauth/introspection/stats", handleIntrospectionStats)
	mux.HandleFunc("/api/v1/oauth/stats/token-binding", handleTokenBindingStats)
	mux.HandleFunc("/api/v1/oauth/scope-lifecycle", handleScopeLifecycle)
	mux.HandleFunc("/api/v1/oauth/stats/authorize-flow", handleAuthorizeFlowStats)
	mux.HandleFunc("/api/v1/oauth/stats/backchannel-logout", handleBackchannelLogoutStats)
	mux.HandleFunc("/api/v1/oauth/stats/oauth-2-1-audit", handleOAuth21Audit(oauthSvc))
	mux.HandleFunc("/api/v1/oauth/clients/onboarding", handleClientOnboarding)
	mux.HandleFunc("/api/v1/oauth/consents/dashboard", handleConsentDashboard)
	mux.HandleFunc("/api/v1/oauth/stats/token-revocation", handleTokenRevocationStats)
	mux.HandleFunc("/api/v1/oauth/dpop/config", handleDPoPConfig)
	mux.HandleFunc("/api/v1/oauth/redirect-uri-validation/config", handleRedirectURIValidationConfig)
	mux.HandleFunc("/api/v1/oauth/oidc/claim-mapping", handleOIDCClaimMapping)
	mux.HandleFunc("/api/v1/oauth/issuer/metadata", handleIssuerMetadataConfig)
	mux.HandleFunc("/api/v1/oauth/ciba/config", handleCIBAConfig)
	mux.HandleFunc("/api/v1/oauth/backchannel", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		_ = r.ParseForm()
		tenantIDStr := r.Header.Get("X-Tenant-ID")
		if tenantIDStr == "" {
			tenantIDStr = r.FormValue("tenant_id")
		}
		tenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request", "error_description": "valid X-Tenant-ID header or tenant_id param required"})
			return
		}
		req := &service.BackchannelAuthRequest{
			TenantID:       tenantID,
			ClientID:       r.FormValue("client_id"),
			ClientSecret:   r.FormValue("client_secret"),
			Scope:          r.FormValue("scope"),
			ACRValues:      r.FormValue("acr_values"),
			LoginHint:      r.FormValue("login_hint"),
			LoginHintToken: r.FormValue("login_hint_token"),
			IDTokenHint:    r.FormValue("id_token_hint"),
			BindingMessage: r.FormValue("binding_message"),
			UserCode:       r.FormValue("user_code"),
			Context:        r.FormValue("context"),
		}
		if expiryStr := r.FormValue("requested_expiry"); expiryStr != "" {
			if n, err := strconv.Atoi(expiryStr); err == nil {
				req.RequestedExpiry = n
			}
		}
		resp, err := oauthSvc.BackchannelAuthentication(r.Context(), req)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request", "error_description": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, resp)
	})
	mux.HandleFunc("/api/v1/oauth/jar/config", handleJARConfig)
	mux.HandleFunc("/api/v1/oauth/oidc-federation/config", handleOIDCFederationConfig)
	mux.HandleFunc("/api/v1/oauth/par/config", handlePARConfig)
	mux.HandleFunc("/api/v1/oauth/fapi-config", handleFAPIConfig(oauthSvc))
	mux.HandleFunc("/api/v1/oauth/token-rotation/config", handleTokenRotationConfig)
	mux.HandleFunc("/api/v1/oauth/client-lifecycle/config", handleClientLifecycleConfig)
	mux.HandleFunc("/api/v1/oauth/consent/config", handleConsentConfig)
	mux.HandleFunc("/api/v1/oauth/dynamic-registration/config", handleDynamicRegistrationConfig)
	mux.HandleFunc("/api/v1/agents/reviews", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			handleAgentReviewCreate(w, r)
		} else {
			handleAgentReviewList(w, r)
		}
	})
	mux.HandleFunc("/api/v1/agents/reviews/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			handleAgentReviewUpdate(w, r)
		} else {
			handleAgentReviewGet(w, r)
		}
	})
	mux.HandleFunc("/api/v1/agents/shadows", handleAgentShadows)
	mux.HandleFunc("/api/v1/agents/drift/report", handleAgentDriftReport)
	mux.HandleFunc("/api/v1/agents/", func(w http.ResponseWriter, r *http.Request) {
		// Agent status update: POST /api/v1/agents/{id}/status
		if strings.HasSuffix(r.URL.Path, "/status") && r.Method == http.MethodPost {
			agentIDStr := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/v1/agents/"), "/status")
			agentID, err := uuid.Parse(agentIDStr)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid agent_id in path"})
				return
			}
			var body struct {
				Status string `json:"status"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
				return
			}
			if err := oauthSvc.UpdateAgentStatus(r.Context(), agentID, service.AgentStatus(body.Status)); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"status": "updated", "agent_id": agentIDStr})
			return
		}
		// Agent drift detection: GET /api/v1/agents/{id}/drift
		if strings.HasSuffix(r.URL.Path, "/drift") {
			handleAgentDriftDetect(w, r)
			return
		}
		// Default: treat as drift detection for backward compat
		handleAgentDriftDetect(w, r)
	}) // /api/v1/agents/{id}/drift
	mux.HandleFunc("/api/v1/oauth/agents/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasSuffix(path, "/lifecycle") {
			handleAgentLifecycle(w, r)
		} else if strings.HasSuffix(path, "/consent") {
			handleAgentConsent(w, r)
		} else if strings.HasSuffix(path, "/behavior") {
			handleAgentBehavior(w, r)
		} else {
			http.Error(w, "not found", http.StatusNotFound)
		}
	})
	mux.HandleFunc("/api/v1/oauth/introspection/config", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, map[string]any{"ttl_seconds": 30, "enabled": true})
		case http.MethodPut:
			var req struct {
				TTLSeconds int  `json:"ttl_seconds"`
				Enabled    *bool `json:"enabled"`
			}
			json.NewDecoder(r.Body).Decode(&req)
			writeJSON(w, http.StatusOK, map[string]any{
				"ttl_seconds": req.TTLSeconds,
				"enabled":     true,
				"updated":     true,
			})
		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		}
	})

	// Front-channel logout
	mux.HandleFunc("/api/v1/oauth/frontchannel-logout", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		var req struct {
			SessionID string `json:"session_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
			return
		}
		uris, err := service.FrontChannelLogout(req.SessionID)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"logout_uris": uris})
	})

	// --- AI Agent Identity (MCP Auth) ---

	// Register a new AI agent
	mux.HandleFunc("/api/v1/agents/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		var req service.AgentRegistration
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}
		agent, err := oauthSvc.RegisterAgent(r.Context(), &req)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, agent)
	})

	// List agents for a tenant
	mux.HandleFunc("/api/v1/agents", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		tenantIDStr := r.Header.Get("X-Tenant-ID")
		if tenantIDStr == "" {
			tenantIDStr = r.URL.Query().Get("tenant_id")
		}
		tenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "valid X-Tenant-ID header or tenant_id query param required"})
			return
		}
		agents, err := oauthSvc.ListAgents(r.Context(), tenantID)
		if err != nil {
			writeInternalError(w, "ListAgents", err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"agents": agents, "total": len(agents)})
	})

	// Agent token exchange (RFC 8693 with agent claims)
	mux.HandleFunc("/api/v1/agents/token", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		tenantIDStr := r.Header.Get("X-Tenant-ID")
		if tenantIDStr == "" {
			tenantIDStr = r.URL.Query().Get("tenant_id")
		}
		tenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "valid X-Tenant-ID header required"})
			return
		}
		var body struct {
			SubjectToken   string   `json:"subject_token"`
			AgentID        string   `json:"agent_id"`
			Scope          []string `json:"scope"`
			MCPServers     []string `json:"mcp_servers"`
			Audience       string   `json:"audience"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}
		agentID, err := uuid.Parse(body.AgentID)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid agent_id"})
			return
		}
		resp, err := oauthSvc.ExchangeAgentToken(r.Context(), &service.AgentTokenExchangeRequest{
			TenantID:       tenantID,
			SubjectToken:   body.SubjectToken,
			AgentID:        agentID,
			RequestedScope: body.Scope,
			MCPServers:     body.MCPServers,
			Audience:       body.Audience,
		})
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, resp)
	})

	// Verify an agent token
	mux.HandleFunc("/api/v1/agents/verify", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		var body struct {
			Token string `json:"token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}
		claims, err := oauthSvc.VerifyAgentToken(r.Context(), body.Token)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error(), "active": "false"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"active":               true,
			"agent_id":             claims.AgentID,
			"agent_type":           claims.AgentType,
			"is_agent_token":       claims.IsAgentToken,
			"max_delegation_depth": claims.MaxDelegationDepth,
			"delegation_chain":     claims.DelegationChain,
			"mcp_servers":          claims.MCPServers,
			"sub":                  claims.Subject,
			"exp":                  claims.ExpiresAt,
		})
	})

	// OAuth scope management
	mux.HandleFunc("/api/v1/oauth/token-lifetime", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/analytics") || r.URL.Path == "/api/v1/oauth/token-lifetime/analytics" {
			handleTokenLifetimeAnalytics(w, r)
			return
		}
	})
	mux.HandleFunc("/api/v1/oauth/token-lifetime/analytics", handleTokenLifetimeAnalytics)
	mux.HandleFunc("/api/v1/oauth/revoke-cascade", handleRevokeCascade)
	mux.HandleFunc("/api/v1/oauth/clients/dependency-graph", handleDependencyGraph)
	mux.HandleFunc("/api/v1/oauth/audience-mismatches", handleAudienceMismatches)
	mux.HandleFunc("/api/v1/oauth/token-scope-diff", handleTokenScopeDiff)
	mux.HandleFunc("/api/v1/oauth/consents/history", handleConsentsHistory)
	mux.HandleFunc("/api/v1/oauth/stats/grant-types", handleGrantTypeStats)
	mux.HandleFunc("/api/v1/oauth/scopes/deprecations", handleScopeDeprecation)
	mux.HandleFunc("/api/v1/oauth/scopes/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/description") {
			handleScopeDescription(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/deprecate") {
			handleScopeDeprecation(w, r)
			return
		}
		if r.URL.Path == "/api/v1/oauth/scopes/deprecations" {
			handleScopeDeprecation(w, r)
			return
		}
		handleScopes(w, r)
	})

	// DPoP proof verification
	mux.HandleFunc("/api/v1/oauth/dpop/verify", handleDPoPVerify)
	mux.HandleFunc("/api/v1/oauth/token/dpop-bind", handleDPoPTokenBind)
	mux.HandleFunc("/api/v1/oauth/token/dpop-verify", handleDPoPTokenVerify)
	mux.HandleFunc("/api/v1/oauth/token-exchange-delegation", handleTokenExchangeDelegation)

	// RAR (RFC 9396): consent preview for authorization_details.
	mux.HandleFunc("/api/v1/oauth/rar/consent-preview", RARConsentPreviewHandler)

	// Previously unreachable handlers — now registered.
	mux.HandleFunc("/api/v1/oauth/client-cert", handleClientCert)
	mux.HandleFunc("/api/v1/oauth/client-events", handleClientEvents)
	mux.HandleFunc("/api/v1/oauth/client-rate-limits", handleClientRateLimits)
	mux.HandleFunc("/api/v1/oauth/onboarding-checklist", handleOnboardingChecklist)
	mux.HandleFunc("/api/v1/oauth/rotation-policy", handleRotationPolicy)
	mux.HandleFunc("/api/v1/oauth/scope-drift", handleScopeDrift)
	mux.HandleFunc("/api/v1/oauth/secret-compare", handleSecretCompare)
	mux.HandleFunc("/api/v1/oauth/secret-history", handleSecretHistory)
	mux.HandleFunc("/api/v1/oauth/validate-client-secret", handleValidateClientSecret)
	mux.HandleFunc("/api/v1/oauth/resource-indicator", handleResourceIndicator)
	mux.HandleFunc("/api/v1/oauth/resource-allowed", handleResourceAllowed)
	mux.HandleFunc("/api/v1/oauth/token-events/stream", handleTokenEventStream)
	// /api/v1/oauth/consent/ is already registered at line ~1025 with sub-path routing
	mux.HandleFunc("/api/v1/oauth/token-families/", handleTokenFamily)
	mux.HandleFunc("/api/v1/oauth/par", handlePAR)
	mux.HandleFunc("/api/v1/oauth/par/", handlePAR)
	mux.HandleFunc("/api/v1/oauth/scopes", handleScopesI18n)
	mux.HandleFunc("/api/v1/oauth/scopes/hierarchy", handleScopeHierarchy)
	mux.HandleFunc("/api/v1/oauth/introspect/batch", handleBatchIntrospect)
	mux.HandleFunc("/api/v1/oauth/consent/analytics", handleConsentAnalytics)
	mux.HandleFunc("/api/v1/oauth/consent/admin-override", handleConsentAdminOverride)
	mux.HandleFunc("/api/v1/oauth/tokens/validate-audience", handleValidateAudience)
	mux.HandleFunc("/api/v1/oauth/token/downscope", handleTokenDownscope)
	// Note: /api/v1/oauth/clients/ sub-paths (branding, suspend, etc.) are handled
	// by the first /api/v1/oauth/clients/ handler registered above (line ~831).
	// Additional sub-paths not covered there are registered with distinct prefixes:
	mux.HandleFunc("/api/v1/oauth/token/claims", handleTokenClaims)
	mux.HandleFunc("/api/v1/oauth/scope-delegation", handleScopeDelegation)
	mux.HandleFunc("/api/v1/oauth/analytics/summary", handleAnalyticsSummary)
	mux.HandleFunc("/api/v1/oauth/grant-flows", handleGrantFlows)
	mux.HandleFunc("/api/v1/oauth/scopes/resolve-dependencies", handleResolveDependencies)
	mux.HandleFunc("/api/v1/oauth/token-entropy", handleTokenEntropy)

	// JWKS key rotation
	mux.HandleFunc("/api/v1/oauth/jwks/rotate", func(w http.ResponseWriter, r *http.Request) {
		handleJWKSRotateWithKP(w, r, rotatingKP)
	})
	mux.HandleFunc("/api/v1/oauth/jwks/rotation-status", func(w http.ResponseWriter, r *http.Request) {
		handleJWKSRotationStatusWithKP(w, r, rotatingKP)
	})
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

// injectTenantContext extracts the tenant ID from the X-Tenant-ID header
// (or tenant_id query param) and returns a context with the tenant attached.
func injectTenantContext(r *http.Request) (context.Context, error) {
	tenantIDStr := r.Header.Get("X-Tenant-ID")
	if tenantIDStr == "" {
		tenantIDStr = r.URL.Query().Get("tenant_id")
	}
	if tenantIDStr == "" {
		return nil, fmt.Errorf("valid X-Tenant-ID header or tenant_id query param required")
	}
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid tenant_id: %s", tenantIDStr)
	}
	return tenant.WithContext(r.Context(), &tenant.Context{TenantID: tenantID}), nil
}

// isClientAuthenticated checks whether the request provides valid client
// authentication per RFC 7662 §2.1. Supported methods:
//  1. HTTP Basic auth (client_id:client_secret)
//  2. Form-encoded client_id + client_secret in the POST body
//  3. Bearer token in the Authorization header (RFC 6750)
//
// If any of these methods provides credentials, the request is considered
// authenticated. This allows resource servers to introspect tokens using
// their own access token without needing to register a client.
func isClientAuthenticated(r *http.Request) bool {
	// Method 1: HTTP Basic auth
	clientID, clientSecret, ok := r.BasicAuth()
	if ok && clientID != "" && clientSecret != "" {
		return true
	}

	// Method 2: Form-encoded client credentials
	formClientID := r.FormValue("client_id")
	formClientSecret := r.FormValue("client_secret")
	if formClientID != "" && formClientSecret != "" {
		return true
	}

	// Method 3: Bearer token (RFC 6750)
	// The presence of a valid Bearer token in the Authorization header
	// satisfies the client authentication requirement.
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		if token != "" {
			return true
		}
	}

	return false
}

// overrideDiscoveryIssuer replaces the internal issuer URL in the OIDC discovery
// config with the public-facing URL derived from the request's forwarded headers.
// This ensures clients see the correct public endpoints when the service runs
// behind a reverse proxy or gateway.
func overrideDiscoveryIssuer(config *domain.OIDCDiscoveryConfig, r *http.Request) {
	// Determine the public scheme.
	scheme := r.Header.Get("X-Forwarded-Proto")
	if scheme == "" {
		scheme = "http"
		if r.TLS != nil {
			scheme = "https"
		}
	}

	// Determine the public host.
	host := r.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = r.Host
	}
	if host == "" {
		return // nothing to override
	}

	publicURL := scheme + "://" + host
	originalIssuer := config.Issuer

	// Only override if the current issuer looks like an internal address.
	// (localhost, private IP ranges, or raw port numbers without a domain)
	if !shouldOverrideIssuer(originalIssuer) {
		return
	}

	config.Issuer = publicURL
	config.AuthorizationEndpoint = strings.Replace(config.AuthorizationEndpoint, originalIssuer, publicURL, 1)
	config.TokenEndpoint = strings.Replace(config.TokenEndpoint, originalIssuer, publicURL, 1)
	config.UserInfoEndpoint = strings.Replace(config.UserInfoEndpoint, originalIssuer, publicURL, 1)
	config.JwksURI = strings.Replace(config.JwksURI, originalIssuer, publicURL, 1)
	config.RevocationEndpoint = strings.Replace(config.RevocationEndpoint, originalIssuer, publicURL, 1)
	config.IntrospectionEndpoint = strings.Replace(config.IntrospectionEndpoint, originalIssuer, publicURL, 1)
	if config.CheckSessionIFrame != "" {
		config.CheckSessionIFrame = strings.Replace(config.CheckSessionIFrame, originalIssuer, publicURL, 1)
	}
	if config.EndSessionEndpoint != "" {
		config.EndSessionEndpoint = strings.Replace(config.EndSessionEndpoint, originalIssuer, publicURL, 1)
	}
}

// shouldOverrideIssuer returns true when the issuer URL looks like an internal
// address (localhost, 127.x, 10.x, 172.16-31.x, 192.168.x, or a bare :port).
func shouldOverrideIssuer(issuer string) bool {
	if issuer == "" {
		return true
	}
	lower := strings.ToLower(issuer)
	if strings.Contains(lower, "localhost") {
		return true
	}
	if strings.Contains(lower, "127.0.0.1") {
		return true
	}
	if strings.Contains(lower, "192.168.") {
		return true
	}
	if strings.Contains(lower, "10.") {
		return true
	}
	// 172.16.0.0 – 172.31.255.255
	if strings.Contains(lower, "172.") {
		return true
	}
	// Bare port like :9005 without a hostname
	if strings.HasPrefix(lower, "http://:") || strings.HasPrefix(lower, "https://:") {
		return true
	}
	return false
}

// redisAdapter wraps *redis.Client to satisfy the service.RedisCmdable interface.
type redisAdapter struct {
	rdb *redis.Client
}

func (a *redisAdapter) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	return a.rdb.Set(ctx, key, value, ttl).Err()
}

func (a *redisAdapter) Get(ctx context.Context, key string) (string, error) {
	return a.rdb.Get(ctx, key).Result()
}

func (a *redisAdapter) GetDel(ctx context.Context, key string) (string, error) {
	return a.rdb.GetDel(ctx, key).Result()
}

func (a *redisAdapter) Del(ctx context.Context, key string) error {
	return a.rdb.Del(ctx, key).Err()
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// writeInternalError logs the actual error and returns a sanitized 500 response.
// Never expose internal error details to the HTTP client.
func writeInternalError(w http.ResponseWriter, op string, err error) {
	slog.Error("internal error", "operation", op, "error", err)
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
}

// extractBearerToken extracts the token from an "Authorization: Bearer <token>" header.
func extractBearerToken(authHeader string) string {
	if authHeader == "" {
		return ""
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return strings.TrimSpace(parts[1])
	}
	return ""
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

// buildIdP constructs a SAML IdentityProvider from the OAuth server config.
// It uses the same RSA key pair used for JWT signing to sign SAML assertions.
func buildIdP(cfg *conf.Config) *saml.IdentityProvider {
	// The IdP uses the OAuth signing key for SAML assertion signing.
	// The actual key is injected via the OAuthService's keyProvider,
	// but for the IdP we construct it from the server's key file.
	privKey, certPEM := loadSAMLSigningKey(cfg)
	certDER := []byte{}
	if certPEM != nil {
		if block, _ := pem.Decode(certPEM); block != nil {
			certDER = block.Bytes
		}
	}

	return &saml.IdentityProvider{
		EntityID:    cfg.Issuer + "/saml/idp/metadata",
		SSOURL:      cfg.Issuer + "/saml/idp/sso",
		SLOURL:      cfg.Issuer + "/saml/idp/slo",
		PrivateKey:  privKey,
		Certificate: certDER,
	}
}

// loadSAMLSigningKey loads the RSA private key and certificate from env/config.
// Falls back to generating an ephemeral key if not configured (for development).
func loadSAMLSigningKey(cfg *conf.Config) (*rsa.PrivateKey, []byte) {
	// Try to load from the same key file used by the OAuth service
	keyPath := os.Getenv("OAUTH_SIGNING_KEY_PATH")
	if keyPath != "" {
		if keyData, err := os.ReadFile(keyPath); err == nil {
			if key, err := jwt.ParseRSAPrivateKeyFromPEM(keyData); err == nil {
				return key, keyData
			}
		}
	}

	// Ephemeral key for development (not for production)
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	certDER := generateSelfSignedCert(key)
	return key, certDER
}

// generateSelfSignedCert creates a DER-encoded self-signed certificate for development.
func generateSelfSignedCert(key *rsa.PrivateKey) []byte {
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		IsCA:         true,
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
	}
	certDER, _ := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	return certDER
}

// parseAuthnRequest extracts the SP EntityID, ACS URL, and request ID from a SAML AuthnRequest XML.
func parseAuthnRequest(rawXML []byte) (entityID, acsURL, requestID string) {
	str := string(rawXML)

	// Extract ID attribute
	if idx := strings.Index(str, `ID="`); idx >= 0 {
		start := idx + 4
		if end := strings.Index(str[start:], `"`); end >= 0 {
			requestID = str[start : start+end]
		}
	}

	// Extract Issuer element (SP EntityID)
	if start := strings.Index(str, "<saml:Issuer>"); start >= 0 {
		start += 14
		if end := strings.Index(str[start:], "</saml:Issuer>"); end >= 0 {
			entityID = str[start : start+end]
		}
	} else if start := strings.Index(str, "<Issuer>"); start >= 0 {
		start += 8
		if end := strings.Index(str[start:], "</Issuer>"); end >= 0 {
			entityID = str[start : start+end]
		}
	}

	// Extract AssertionConsumerServiceURL
	if idx := strings.Index(str, `AssertionConsumerServiceURL="`); idx >= 0 {
		start := idx + 29
		if end := strings.Index(str[start:], `"`); end >= 0 {
			acsURL = str[start : start+end]
		}
	}

	return entityID, acsURL, requestID
}