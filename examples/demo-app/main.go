// GGID OAuth 2.0 / OIDC Demo Application
//
// This is a minimal Go HTTP server that demonstrates how to integrate
// with GGID using OAuth 2.0 Authorization Code flow with PKCE.
//
// Prerequisites:
//   1. Create an OAuth client in GGID:
//      POST /api/v1/oauth/clients
//      {"client_name":"Demo App","redirect_uris":["http://localhost:9099/callback"],"grant_types":["authorization_code"],"response_types":["code"],"scopes":["openid profile email"]}
//
//   2. Set env vars and run:
//      GGID_ISSUER=https://ggid.iot2.win \
//      CLIENT_ID=<your_client_id> \
//      CLIENT_SECRET=<your_client_secret> \
//      TENANT_ID=00000000-0000-0000-0000-000000000001 \
//      go run examples/demo-app/main.go
//
//   3. Open http://localhost:9099
package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

var (
	ggidIssuer = getenv("GGID_ISSUER", "http://localhost:8080")
	clientID   = getenv("CLIENT_ID", "")
	clientSec  = getenv("CLIENT_SECRET", "")
	tenantID   = getenv("TENANT_ID", "00000000-0000-0000-0000-000000000001")
	listenAddr = getenv("LISTEN_ADDR", ":9099")
	redirectURI = getenv("REDIRECT_URI", "http://localhost:9099/callback")
)

// In-memory session store (PKCE verifiers + states)
var (
	sessions   = sync.Map{}
)

type session struct {
	State       string
	CodeVerifier string
	CodeChallenge string
}

func main() {
	if clientID == "" {
		log.Fatal("CLIENT_ID env var is required")
	}

	http.HandleFunc("/", handleHome)
	http.HandleFunc("/login", handleLogin)
	http.HandleFunc("/callback", handleCallback)
	http.HandleFunc("/logout", handleLogout)

	log.Printf("Demo app listening on %s", listenAddr)
	log.Printf("GGID Issuer: %s", ggidIssuer)
	log.Printf("Client ID: %s", clientID)
	log.Printf("Redirect URI: %s", redirectURI)
	log.Fatal(http.ListenAndServe(listenAddr, nil))
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	// Check if user is logged in (cookie-based)
	cookie, err := r.Cookie("demo_user")
	if err == nil && cookie.Value != "" {
		// Show logged-in page
		userData := sessions.Load(cookie.Value)
		if userData != nil {
			showWelcome(w, userData.(map[string]any))
			return
		}
	}

	// Show login button
	tmpl := template.Must(template.New("login").Parse(loginHTML))
	tmpl.Execute(w, map[string]string{
		"Issuer":   ggidIssuer,
		"ClientID": clientID,
	})
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	state := randomString(16)
	verifier := randomString(64)
	challenge := pkceChallenge(verifier)

	// Store session
	sessions.Store(state, session{
		State:        state,
		CodeVerifier: verifier,
		CodeChallenge: challenge,
	})

	// Build authorize URL
	params := url.Values{
		"response_type":         {"code"},
		"client_id":             {clientID},
		"redirect_uri":          {redirectURI},
		"scope":                 {"openid profile email"},
		"state":                 {state},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
		"tenant_id":             {tenantID},
	}

	authURL := ggidIssuer + "/oauth/authorize?" + params.Encode()
	http.Redirect(w, r, authURL, http.StatusFound)
}

func handleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	errParam := r.URL.Query().Get("error")

	if errParam != "" {
		showError(w, "Authorization error: "+errParam+" - "+r.URL.Query().Get("error_description"))
		return
	}
	if code == "" {
		showError(w, "No authorization code received")
		return
	}

	// Verify state
	sessVal, ok := sessions.Load(state)
	if !ok {
		showError(w, "Invalid or expired state")
		return
	}
	sess := sessVal.(session)
	sessions.Delete(state) // Use once

	// Exchange code for token
	tokenResp, err := exchangeToken(code, sess.CodeVerifier)
	if err != nil {
		showError(w, "Token exchange failed: "+err.Error())
		return
	}

	accessToken, _ := tokenResp["access_token"].(string)
	if accessToken == "" {
		showError(w, "No access_token in response")
		return
	}

	// Fetch userinfo
	userInfo, err := fetchUserInfo(accessToken)
	if err != nil {
		// Show token without userinfo if userinfo fails
		userInfo = map[string]any{
			"access_token": accessToken[:20] + "...",
			"note":         "userinfo endpoint not available, showing token info",
		}
	}

	// Store user session via cookie
	sessionID := randomString(32)
	sessions.Store(sessionID, userInfo)
	http.SetCookie(w, &http.Cookie{
		Name:     "demo_user",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   3600,
	})

	showWelcome(w, userInfo)
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("demo_user")
	if err == nil {
		sessions.Delete(cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:   "demo_user",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
	http.Redirect(w, r, "/", http.StatusFound)
}

// exchangeToken exchanges the authorization code for an access token.
func exchangeToken(code, codeVerifier string) (map[string]any, error) {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"client_id":     {clientID},
		"client_secret": {clientSec},
		"code_verifier": {codeVerifier},
	}

	resp, err := http.PostForm(ggidIssuer+"/oauth/token", data)
	if err != nil {
		return nil, fmt.Errorf("POST /oauth/token: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse token response: %w", err)
	}
	return result, nil
}

// fetchUserInfo calls the GGID userinfo endpoint with the access token.
func fetchUserInfo(accessToken string) (map[string]any, error) {
	req, _ := http.NewRequest("GET", ggidIssuer+"/oauth/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET /oauth/userinfo: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("userinfo returned %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse userinfo: %w", err)
	}
	return result, nil
}

// --- Helpers ---

func randomString(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func pkceChallenge(verifier string) string {
	// S256: BASE64URL(SHA256(verifier))
	// Using crypto/sha256
	h := sha256Bytes([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h)
}

// Avoid importing crypto/sha254 at top level for clarity
func sha256Bytes(b []byte) []byte {
	// Inline to keep imports minimal
	h := newSHA256()
	h.Write(b)
	return h.Sum(nil)
}

func getenv(key, dflt string) string {
	v := strings.TrimSpace(getenvRaw(key))
	if v == "" {
		return dflt
	}
	return v
}

// showWelcome renders the logged-in page.
func showWelcome(w http.ResponseWriter, user map[string]any) {
	username := getStr(user, "preferred_username")
	if username == "" {
		username = getStr(user, "sub")
	}
	if username == "" {
		username = "User"
	}

	name := getStr(user, "name")
	if name == "" {
		name = username
	}

	email := getStr(user, "email")

	// Pretty-print all claims
	claims, _ := json.MarshalIndent(user, "", "  ")

	tmpl := template.Must(template.New("welcome").Parse(welcomeHTML))
	tmpl.Execute(w, map[string]any{
		"Name":    name,
		"Username": username,
		"Email":   email,
		"Claims":  template.HTML(string(claims)),
	})
}

func showError(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<html><body><h2>Error</h2><p>%s</p><p><a href="/">← Back to Home</a></p></body></html>`, template.HTMLEscapeString(msg))
}

func getStr(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

const loginHTML = `<!DOCTYPE html>
<html>
<head><title>GGID Demo App</title>
<style>
  body { font-family: sans-serif; display: flex; justify-content: center; align-items: center; height: 100vh; margin: 0; background: #f5f5f5; }
  .card { background: white; padding: 3rem; border-radius: 12px; box-shadow: 0 2px 8px rgba(0,0,0,0.1); text-align: center; }
  h1 { color: #333; margin-bottom: 0.5rem; }
  p { color: #666; margin-bottom: 2rem; }
  .btn { display: inline-block; background: #4F46E5; color: white; padding: 0.75rem 2rem; border-radius: 8px; text-decoration: none; font-weight: 600; }
  .btn:hover { background: #4338CA; }
  .info { margin-top: 1rem; font-size: 0.8rem; color: #999; }
</style>
</head>
<body>
<div class="card">
  <h1>GGID Demo App</h1>
  <p>Sign in with your GGID account to continue</p>
  <a href="/login" class="btn">Login with GGID</a>
  <div class="info">Issuer: {{.Issuer}} | Client: {{.ClientID}}</div>
</div>
</body>
</html>`

const welcomeHTML = `<!DOCTYPE html>
<html>
<head><title>Welcome - GGID Demo</title>
<style>
  body { font-family: sans-serif; display: flex; justify-content: center; align-items: center; min-height: 100vh; margin: 0; background: #f5f5f5; }
  .card { background: white; padding: 3rem; border-radius: 12px; box-shadow: 0 2px 8px rgba(0,0,0,0.1); max-width: 600px; width: 90%; }
  h1 { color: #333; }
  .info { background: #f9fafb; padding: 1rem; border-radius: 8px; margin: 1rem 0; }
  .claims { background: #1e1e1e; color: #d4d4d4; padding: 1rem; border-radius: 8px; font-family: monospace; font-size: 0.85rem; overflow-x: auto; white-space: pre-wrap; }
  .btn { display: inline-block; background: #dc2626; color: white; padding: 0.5rem 1.5rem; border-radius: 8px; text-decoration: none; font-weight: 600; margin-top: 1rem; }
</style>
</head>
<body>
<div class="card">
  <h1>Welcome, {{.Name}}!</h1>
  <div class="info">
    <p><strong>Username:</strong> {{.Username}}</p>
    <p><strong>Email:</strong> {{.Email}}</p>
  </div>
  <h3>Token Claims</h3>
  <div class="claims">{{.Claims}}</div>
  <br>
  <a href="/logout" class="btn">Logout</a>
</div>
</body>
</html>`
