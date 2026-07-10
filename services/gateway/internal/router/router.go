// Package router implements the HTTP reverse proxy router for the API Gateway.
package router

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/ggid/ggid/services/gateway/internal/config"
	"github.com/ggid/ggid/services/gateway/internal/middleware"
)

// publicPaths are paths that skip JWT verification.
var publicPaths = []string{
	"/api/v1/auth/login",
	"/api/v1/auth/register",
	"/api/v1/auth/refresh",
	"/api/v1/auth/password/forgot",
	"/api/v1/auth/password/reset",
	"/api/v1/auth/social/",
	"/oauth/",
	"/saml/",
	"/.well-known/",
	"/docs",
	"/api-docs",
	"/login",
	"/register",
	"/forgot-password",
}

// Gateway is the API Gateway HTTP handler.
type Gateway struct {
	cfg      *config.Config
	jwks     *middleware.JWKSClient
	proxies  map[string]*httputil.ReverseProxy
	mu       sync.RWMutex
}

// New creates a new API Gateway handler.
func New(cfg *config.Config, jwks *middleware.JWKSClient) *Gateway {
	gw := &Gateway{
		cfg:     cfg,
		jwks:    jwks,
		proxies: make(map[string]*httputil.ReverseProxy),
	}
	gw.buildProxies()
	return gw
}

func (gw *Gateway) buildProxies() {
	for prefix, backendURL := range gw.cfg.Routes {
		parsed, err := url.Parse(backendURL)
		if err != nil {
			log.Printf("invalid backend URL %s: %v", backendURL, err)
			continue
		}
		proxy := httputil.NewSingleHostReverseProxy(parsed)
		originalDirector := proxy.Director
		proxy.Director = func(req *http.Request) {
			originalDirector(req)
			// Forward resolved identity headers to the backend service
			if requestID, ok := req.Context().Value(middleware.RequestIDKey).(string); ok {
				req.Header.Set("X-Request-ID", requestID)
			}
			if userID, ok := middleware.UserIDFromRequest(req); ok {
				req.Header.Set("X-User-ID", userID.String())
			}
			if tenantID, ok := middleware.TenantIDFromRequest(req); ok {
				req.Header.Set("X-Tenant-ID", tenantID)
				// Inject as query param for GET requests
				q := req.URL.Query()
				if q.Get("tenant_id") == "" {
					q.Set("tenant_id", tenantID)
					req.URL.RawQuery = q.Encode()
				}
				// Inject into JSON body for POST/PUT/PATCH requests
				injectTenantIntoBody(req, tenantID)
			}
		}
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("proxy error for %s%s: %v", parsed.Host, r.URL.Path, err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadGateway)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "backend service unavailable"})
		}
		gw.proxies[prefix] = proxy
	}
}

// ServeHTTP routes the request to the appropriate backend service.
func (gw *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Health check
	if r.URL.Path == "/healthz" {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		return
	}

	// Prometheus metrics
	if r.URL.Path == "/metrics" {
		middleware.MetricsHandler().ServeHTTP(w, r)
		return
	}

	// JWKS endpoint
	if r.URL.Path == "/.well-known/jwks.json" {
		gw.jwks.JWKSHandler()(w, r)
		return
	}

	// API documentation (Swagger UI)
	if r.URL.Path == "/docs" || r.URL.Path == "/docs/" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(swaggerHTML))
		return
	}

	// Hosted login page (served by Gateway — any app can redirect here)
	if r.URL.Path == "/login" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(hostedLoginHTML))
		return
	}

	// Hosted registration page
	if r.URL.Path == "/register" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(hostedRegisterHTML))
		return
	}

	// Password reset page
	if r.URL.Path == "/forgot-password" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(hostedForgotPasswordHTML))
		return
	}

	// OpenAPI JSON spec
	if r.URL.Path == "/api-docs" {
		serveOpenAPISpec(w, r)
		return
	}

	// Find matching backend by longest prefix
	backend := gw.matchBackend(r.URL.Path)
	if backend == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "no route for this path"})
		return
	}

	backend.ServeHTTP(w, r)
}

func (gw *Gateway) matchBackend(path string) *httputil.ReverseProxy {
	gw.mu.RLock()
	defer gw.mu.RUnlock()

	var bestMatch string
	for prefix := range gw.proxies {
		if strings.HasPrefix(path, prefix) {
			if len(prefix) > len(bestMatch) {
				bestMatch = prefix
			}
		}
	}
	if bestMatch == "" {
		return nil
	}
	return gw.proxies[bestMatch]
}

// Handler returns an http.Handler with all middleware applied in the correct order.
// Public paths (login, register, healthz, .well-known) skip JWT verification.
// All other paths require a valid JWT Bearer token.
func (gw *Gateway) Handler() http.Handler {
	// Inner handler: JWT enforcement + gateway routing
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if this is a public path
		isPublic := false
		for _, pp := range publicPaths {
			if strings.HasPrefix(r.URL.Path, pp) {
				isPublic = true
				break
		}
		}
		// Health check and JWKS are always public
		if r.URL.Path == "/healthz" || r.URL.Path == "/.well-known/jwks.json" || r.URL.Path == "/metrics" {
			isPublic = true
		}

		if isPublic {
			// Public path: no JWT required, but still validate if token present
			jwtMW := middleware.JWTAuth(gw.jwks, false, gw.cfg.JWTIssuer, gw.cfg.JWTAudience)
			jwtMW(gw).ServeHTTP(w, r)
		} else {
			// Protected path: JWT required
			jwtMW := middleware.JWTAuth(gw.jwks, true, gw.cfg.JWTIssuer, gw.cfg.JWTAudience)
			jwtMW(gw).ServeHTTP(w, r)
		}
	})

	// Apply outer middleware: CORS → RequestID → Logging → TenantResolver → inner
	handler := middleware.TenantResolver(gw.cfg.DomainSuffix)(inner)
	handler = middleware.Logging(handler)
	handler = middleware.RequestID(handler)
	handler = middleware.CORS(handler)

	return handler
}

// injectTenantIntoBody injects tenant_id into the JSON body of POST/PUT/PATCH requests.
// It only modifies flat JSON objects and preserves the original body if it's not JSON
// or already contains a tenant_id field.
func injectTenantIntoBody(req *http.Request, tenantID string) {
	if req.Body == nil || tenantID == "" {
		return
	}
	// Only modify JSON bodies for write methods
	if req.Method != http.MethodPost && req.Method != http.MethodPut && req.Method != http.MethodPatch {
		return
	}
	ct := req.Header.Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		return
	}

	bodyBytes, err := io.ReadAll(req.Body)
	req.Body.Close()
	if err != nil {
		return
	}
	// Restore body if anything fails
	restore := func() {
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	var bodyMap map[string]any
	if json.Unmarshal(bodyBytes, &bodyMap) != nil {
		restore()
		return
	}
	// Skip if tenant_id already present
	if _, exists := bodyMap["tenant_id"]; exists {
		restore()
		return
	}

	bodyMap["tenant_id"] = tenantID
	newBody, err := json.Marshal(bodyMap)
	if err != nil {
		restore()
		return
	}
	req.Body = io.NopCloser(bytes.NewReader(newBody))
	req.ContentLength = int64(len(newBody))
	req.Header.Set("Content-Length", strconv.Itoa(len(newBody)))
}

// PrintRoutes logs the configured routes at startup.
func (gw *Gateway) PrintRoutes() {
	log.Println("API Gateway routes:")
	for prefix, backend := range gw.cfg.Routes {
		log.Printf("  %s -> %s", prefix, backend)
	}
	log.Println("  /docs -> Swagger UI")
	log.Println("  /api-docs -> OpenAPI JSON spec")
}

// serveSwaggerUI writes the Swagger UI HTML page.
func serveSwaggerUI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(swaggerHTML))
}

// serveOpenAPISpec writes the OpenAPI 3.0 JSON spec.
func serveOpenAPISpec(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(openAPISpec))
}

const swaggerHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>GGID API Documentation</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
  <style>body{margin:0}</style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.onload=function(){
      SwaggerUIBundle({
        url:'/api-docs',
        dom_id:'#swagger-ui',
        deepLinking:true,
        presets:[SwaggerUIBundle.presets.apis],
        layout:'BaseLayout',
        requestInterceptor:function(req){
          req.headers['X-Tenant-ID']='00000000-0000-0000-0000-000000000001';
          return req;
        }
      })
    }
  </script>
</body>
</html>`

const openAPISpec = `{
  "openapi": "3.0.3",
  "info": {"title": "GGID IAM API", "version": "1.0.0", "license": {"name": "Apache 2.0"}},
  "servers": [{"url": "http://localhost:8080"}],
  "components": {
    "securitySchemes": {
      "BearerAuth": {"type": "http", "scheme": "bearer"},
      "TenantID": {"type": "apiKey", "in": "header", "name": "X-Tenant-ID"}
    }
  },
  "security": [{"BearerAuth": [], "TenantID": []}],
  "tags": [
    {"name": "Auth", "description": "Authentication endpoints"},
    {"name": "Users", "description": "User management"},
    {"name": "Roles", "description": "Roles and permissions"},
    {"name": "Organizations", "description": "Organization management"},
    {"name": "Audit", "description": "Audit logs"},
    {"name": "OAuth2", "description": "OAuth2/OIDC endpoints"}
  ],
  "paths": {
    "/api/v1/auth/register": {
      "post": {"tags":["Auth"],"summary":"Register new user","security":[],"requestBody":{"content":{"application/json":{"schema":{"type":"object","properties":{"username":{"type":"string"},"email":{"type":"string"},"password":{"type":"string"}}}}}},"responses":{"201":{"description":"Created"},"409":{"description":"Conflict"}}}
    },
    "/api/v1/auth/login": {
      "post": {"tags":["Auth"],"summary":"Login","security":[],"requestBody":{"content":{"application/json":{"schema":{"type":"object","properties":{"username":{"type":"string"},"password":{"type":"string"}}}}}},"responses":{"200":{"description":"OK"},"401":{"description":"Unauthorized"}}}
    },
    "/api/v1/auth/social/{provider}": {
      "get": {"tags":["Auth"],"summary":"Begin social login","security":[],"parameters":[{"name":"provider","in":"path","required":true,"schema":{"type":"string","enum":["google","github","oidc"]}}],"responses":{"200":{"description":"Auth URL"}}}
    },
    "/api/v1/users": {
      "get": {"tags":["Users"],"summary":"List users","responses":{"200":{"description":"OK"}}}
    },
    "/api/v1/users/{id}": {
      "get": {"tags":["Users"],"summary":"Get user by ID","parameters":[{"name":"id","in":"path","required":true,"schema":{"type":"string","format":"uuid"}}],"responses":{"200":{"description":"OK"}}},
      "delete": {"tags":["Users"],"summary":"Delete user","parameters":[{"name":"id","in":"path","required":true,"schema":{"type":"string","format":"uuid"}}],"responses":{"204":{"description":"Deleted"}}}
    },
    "/api/v1/roles": {
      "get": {"tags":["Roles"],"summary":"List roles","responses":{"200":{"description":"OK"}}},
      "post": {"tags":["Roles"],"summary":"Create role","requestBody":{"content":{"application/json":{"schema":{"type":"object","properties":{"name":{"type":"string"},"key":{"type":"string"},"description":{"type":"string"}}}}}},"responses":{"201":{"description":"Created"}}}
    },
    "/api/v1/orgs": {
      "get": {"tags":["Organizations"],"summary":"List organizations","responses":{"200":{"description":"OK"}}},
      "post": {"tags":["Organizations"],"summary":"Create organization","requestBody":{"content":{"application/json":{"schema":{"type":"object","properties":{"name":{"type":"string"},"slug":{"type":"string"}}}}}},"responses":{"201":{"description":"Created"}}}
    },
    "/api/v1/orgs/{id}/tree": {
      "get": {"tags":["Organizations"],"summary":"Get org tree","parameters":[{"name":"id","in":"path","required":true,"schema":{"type":"string","format":"uuid"}}],"responses":{"200":{"description":"OK"}}}
    },
    "/api/v1/audit": {
      "get": {"tags":["Audit"],"summary":"Query audit events","parameters":[{"name":"limit","in":"query","schema":{"type":"integer"}},{"name":"offset","in":"query","schema":{"type":"integer"}}],"responses":{"200":{"description":"OK"}}}
    },
    "/oauth/authorize": {
      "get": {"tags":["OAuth2"],"summary":"OAuth2 authorize","security":[],"parameters":[{"name":"client_id","in":"query","required":true,"schema":{"type":"string"}},{"name":"redirect_uri","in":"query","required":true,"schema":{"type":"string"}},{"name":"response_type","in":"query","required":true,"schema":{"type":"string"}}],"responses":{"302":{"description":"Redirect"}}}
    },
    "/oauth/token": {
      "post": {"tags":["OAuth2"],"summary":"OAuth2 token exchange","security":[],"requestBody":{"content":{"application/x-www-form-urlencoded":{"schema":{"type":"object","properties":{"grant_type":{"type":"string"},"code":{"type":"string"},"client_id":{"type":"string"},"client_secret":{"type":"string"}}}}}},"responses":{"200":{"description":"Token"}}}
    },
    "/oauth/.well-known/openid-configuration": {
      "get": {"tags":["OAuth2"],"summary":"OIDC discovery","security":[],"responses":{"200":{"description":"Discovery document"}}}
    },
    "/healthz": {
      "get": {"tags":["Auth"],"summary":"Health check","security":[],"responses":{"200":{"description":"OK"}}}
    }
  }
}`

// hostedLoginHTML is the GGID Universal Login page.
// Any application can redirect users here via OAuth2 /authorize flow.
// It supports: username/password, social login (Google/GitHub/SSO), and MFA.
const hostedLoginHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Sign In — GGID</title>
  <style>
    *{margin:0;padding:0;box-sizing:border-box}
    body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;background:#f0f2f5;display:flex;align-items:center;justify-content:center;min-height:100vh}
    .card{background:#fff;border-radius:12px;box-shadow:0 2px 10px rgba(0,0,0,.08);width:400px;max-width:90vw;padding:36px}
    .logo{font-size:28px;font-weight:800;text-align:center;margin-bottom:6px;color:#1a1a1a}
    .logo span{color:#4f46e5}
    .subtitle{text-align:center;color:#6b7280;font-size:14px;margin-bottom:24px}
    .field{margin-bottom:16px}
    label{display:block;font-size:13px;font-weight:600;color:#374151;margin-bottom:5px}
    input{width:100%;padding:10px 12px;border:1px solid #d1d5db;border-radius:8px;font-size:14px;outline:none;transition:border-color .2s}
    input:focus{border-color:#4f46e5;box-shadow:0 0 0 3px rgba(79,70,229,.1)}
    .btn{width:100%;padding:11px;border:none;border-radius:8px;font-size:14px;font-weight:600;cursor:pointer;transition:background .2s}
    .btn-primary{background:#4f46e5;color:#fff}
    .btn-primary:hover{background:#4338ca}
    .btn-primary:disabled{opacity:.5;cursor:not-allowed}
    .divider{display:flex;align-items:center;margin:20px 0;color:#9ca3af;font-size:12px}
    .divider::before,.divider::after{content:'';flex:1;height:1px;background:#e5e7eb}
    .divider span{padding:0 12px}
    .social{display:grid;grid-template-columns:1fr 1fr 1fr;gap:8px}
    .btn-social{padding:10px;border:1px solid #d1d5db;border-radius:8px;background:#fff;font-size:13px;font-weight:500;cursor:pointer;text-align:center;transition:background .2s}
    .btn-social:hover{background:#f9fafb}
    .error{background:#fef2f2;border:1px solid #fecaca;color:#dc2626;padding:10px 12px;border-radius:8px;font-size:13px;margin-bottom:16px;display:none}
    .success{background:#f0fdf4;border:1px solid #bbf7d0;color:#16a34a;padding:10px 12px;border-radius:8px;font-size:13px;margin-bottom:16px;display:none}
    .footer{text-align:center;margin-top:20px;font-size:12px;color:#9ca3af}
    .footer a{color:#4f46e5;text-decoration:none}
    .mfa{display:none}
    .hint{background:#eff6ff;border:1px solid #bfdbfe;color:#1e40af;padding:8px 12px;border-radius:8px;font-size:12px;margin-top:16px;text-align:center}
  </style>
</head>
<body>
<div class="card">
  <div class="logo">G<span>GID</span></div>
  <div class="subtitle" id="subtitle">Sign in to your account</div>

  <div class="error" id="err"></div>
  <div class="success" id="ok"></div>

  <!-- Login Form -->
  <div id="login-form">
    <div class="field">
      <label>Username</label>
      <input type="text" id="username" placeholder="admin" autocomplete="username">
    </div>
    <div class="field">
      <label>Password</label>
      <input type="password" id="password" placeholder="Your password" autocomplete="current-password">
    </div>
    <button class="btn btn-primary" id="login-btn" onclick="doLogin()">Sign In</button>

    <div class="divider"><span>or continue with</span></div>
    <div class="social">
      <button class="btn-social" onclick="socialLogin('google')">Google</button>
      <button class="btn-social" onclick="socialLogin('github')">GitHub</button>
      <button class="btn-social" onclick="socialLogin('oidc')">SSO</button>
    </div>

    <div class="hint">Default: admin / Admin@123456</div>
    <div class="footer">
      <a href="/forgot-password">Forgot password?</a> ·
      <a href="/register">Create account</a>
    </div>
  </div>

  <!-- MFA Form -->
  <div id="mfa-form" style="display:none">
    <div class="field">
      <label>Authentication Code</label>
      <input type="text" id="mfa-code" placeholder="123456" maxlength="6" onkeyup="if(event.keyCode===13)doMfa()">
    </div>
    <button class="btn btn-primary" onclick="doMfa()">Verify</button>
  </div>
</div>

<script>
const T="00000000-0000-0000-0000-000000000001";
const params=new URLSearchParams(location.search);
const redirectUri=params.get("redirect_uri")||"/";

function showErr(m){const e=document.getElementById("err");e.textContent=m;e.style.display="block";setTimeout(()=>e.style.display="none",5000)}
function showOk(m){const e=document.getElementById("ok");e.textContent=m;e.style.display="block"}

async function doLogin(){
  const u=document.getElementById("username").value;
  const p=document.getElementById("password").value;
  if(!u||!p){showErr("Please enter username and password");return}
  const btn=document.getElementById("login-btn");
  btn.disabled=true;btn.textContent="Signing in...";
  try{
    const r=await fetch("/api/v1/auth/login",{method:"POST",headers:{"Content-Type":"application/json","X-Tenant-ID":T},body:JSON.stringify({username:u,password:p})});
    const d=await r.json();
    if(!r.ok){showErr(d.error||"Login failed");return}
    if(d.mfa_required){document.getElementById("login-form").style.display="none";document.getElementById("mfa-form").style.display="block";document.getElementById("subtitle").textContent="Enter your verification code";window._session=d.session_id;return}
    localStorage.setItem("ggid_access_token",d.access_token);
    localStorage.setItem("ggid_refresh_token",d.refresh_token);
    showOk("Success! Redirecting...");
    setTimeout(()=>window.location.href=redirectUri,500);
  }catch(e){showErr("Network error — is the API running?")}
  finally{btn.disabled=false;btn.textContent="Sign In"}
}

async function doMfa(){
  const code=document.getElementById("mfa-code").value;
  try{
    const r=await fetch("/api/v1/auth/mfa/login",{method:"POST",headers:{"Content-Type":"application/json","X-Tenant-ID":T},body:JSON.stringify({session_id:window._session,code:code})});
    const d=await r.json();
    if(!r.ok){showErr(d.error||"Invalid code");return}
    localStorage.setItem("ggid_access_token",d.access_token);
    showOk("Verified! Redirecting...");
    setTimeout(()=>window.location.href=redirectUri,500);
  }catch(e){showErr("Network error")}
}

function socialLogin(p){
  fetch("/api/v1/auth/social/"+p+"?redirect_uri="+encodeURIComponent(redirectUri),{headers:{"X-Tenant-ID":T}})
    .then(r=>r.json()).then(d=>{if(d.auth_url)window.location.href=d.auth_url}).catch(()=>showErr(p+" login not configured"));
}
</script>
</body>
</html>`

// hostedRegisterHTML is the GGID registration page.
const hostedRegisterHTML = `<!DOCTYPE html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Create Account — GGID</title>
<style>*{margin:0;padding:0;box-sizing:border-box}body{font-family:-apple-system,BlinkMacSystemFont,sans-serif;background:#f0f2f5;display:flex;align-items:center;justify-content:center;min-height:100vh}.card{background:#fff;border-radius:12px;box-shadow:0 2px 10px rgba(0,0,0,.08);width:400px;max-width:90vw;padding:36px}.logo{font-size:28px;font-weight:800;text-align:center;margin-bottom:6px}.logo span{color:#4f46e5}.sub{text-align:center;color:#6b7280;font-size:14px;margin-bottom:24px}.field{margin-bottom:16px}label{display:block;font-size:13px;font-weight:600;margin-bottom:5px}input{width:100%;padding:10px 12px;border:1px solid #d1d5db;border-radius:8px;font-size:14px}.btn{width:100%;padding:11px;border:none;border-radius:8px;background:#4f46e5;color:#fff;font-size:14px;font-weight:600;cursor:pointer}.btn:hover{background:#4338ca}.err{background:#fef2f2;border:1px solid #fecaca;color:#dc2626;padding:10px;border-radius:8px;font-size:13px;margin-bottom:16px;display:none}.ok{background:#f0fdf4;border:1px solid #bbf7d0;color:#16a34a;padding:10px;border-radius:8px;font-size:13px;margin-bottom:16px;display:none}.footer{text-align:center;margin-top:20px;font-size:12px;color:#9ca3af}.footer a{color:#4f46e5;text-decoration:none}</style>
</head><body><div class="card">
<div class="logo">G<span>GID</span></div><div class="sub">Create your account</div>
<div class="err" id="err"></div><div class="ok" id="ok"></div>
<div class="field"><label>Username</label><input type="text" id="username" placeholder="Choose a username"></div>
<div class="field"><label>Email</label><input type="email" id="email" placeholder="you@example.com"></div>
<div class="field"><label>Password</label><input type="password" id="password" placeholder="At least 8 characters"></div>
<div class="field"><label>Full Name</label><input type="text" id="name" placeholder="Your name"></div>
<button class="btn" id="btn" onclick="doRegister()">Create Account</button>
<div class="footer">Already have an account? <a href="/login">Sign in</a></div>
</div><script>
const T="00000000-0000-0000-0000-000000000001";
function showErr(m){const e=document.getElementById("err");e.textContent=m;e.style.display="block";setTimeout(()=>e.style.display="none",5000)}
function showOk(m){const e=document.getElementById("ok");e.textContent=m;e.style.display="block"}
async function doRegister(){
  const u=document.getElementById("username").value,p=document.getElementById("password").value,e=document.getElementById("email").value,n=document.getElementById("name").value;
  if(!u||!p||!e){showErr("Username, email, and password are required");return}
  const btn=document.getElementById("btn");btn.disabled=true;btn.textContent="Creating...";
  try{
    const r=await fetch("/api/v1/auth/register",{method:"POST",headers:{"Content-Type":"application/json","X-Tenant-ID":T},body:JSON.stringify({username:u,email:e,password:p,name:n})});
    const d=await r.json();
    if(!r.ok){showErr(d.error||"Registration failed");return}
    showOk("Account created! Redirecting to login...");
    setTimeout(()=>window.location.href="/login",1500);
  }catch(err){showErr("Network error")}
  finally{btn.disabled=false;btn.textContent="Create Account"}
}
</script></body></html>`

// hostedForgotPasswordHTML is the GGID password reset request page.
const hostedForgotPasswordHTML = `<!DOCTYPE html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Reset Password — GGID</title>
<style>*{margin:0;padding:0;box-sizing:border-box}body{font-family:-apple-system,BlinkMacSystemFont,sans-serif;background:#f0f2f5;display:flex;align-items:center;justify-content:center;min-height:100vh}.card{background:#fff;border-radius:12px;box-shadow:0 2px 10px rgba(0,0,0,.08);width:400px;max-width:90vw;padding:36px}.logo{font-size:28px;font-weight:800;text-align:center;margin-bottom:6px}.logo span{color:#4f46e5}.sub{text-align:center;color:#6b7280;font-size:14px;margin-bottom:24px}.field{margin-bottom:16px}label{display:block;font-size:13px;font-weight:600;margin-bottom:5px}input{width:100%;padding:10px 12px;border:1px solid #d1d5db;border-radius:8px;font-size:14px}.btn{width:100%;padding:11px;border:none;border-radius:8px;background:#4f46e5;color:#fff;font-size:14px;font-weight:600;cursor:pointer}.ok{background:#f0fdf4;border:1px solid #bbf7d0;color:#16a34a;padding:10px;border-radius:8px;font-size:13px;margin-bottom:16px;display:none}.footer{text-align:center;margin-top:20px;font-size:12px;color:#9ca3af}.footer a{color:#4f46e5;text-decoration:none}</style>
</head><body><div class="card">
<div class="logo">G<span>GID</span></div><div class="sub">Reset your password</div>
<div class="ok" id="ok"></div>
<div class="field"><label>Email or Username</label><input type="text" id="email" placeholder="you@example.com"></div>
<button class="btn" onclick="doReset()">Send Reset Link</button>
<div class="footer"><a href="/login">Back to sign in</a></div>
</div><script>
const T="00000000-0000-0000-0000-000000000001";
function showOk(m){const e=document.getElementById("ok");e.textContent=m;e.style.display="block"}
async function doReset(){
  const e=document.getElementById("email").value;
  if(!e)return;
  try{
    await fetch("/api/v1/auth/password/forgot",{method:"POST",headers:{"Content-Type":"application/json","X-Tenant-ID":T},body:JSON.stringify({email:e})});
  }catch(err){}
  showOk("If that email exists, a reset link has been sent.");
}
</script></body></html>`
