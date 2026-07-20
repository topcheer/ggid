// GGID SDK Demo — Full-featured app with OAuth login + permission-based UI.
//
// Features: Login → Dashboard (role badge + permissions) → Inventory (needs inventory:read) → Orders (needs orders:read, write shows button) → Admin (needs admin scope)
//
// Run: GGID_URL=https://ggid.iot2.win CLIENT_ID=xxx CLIENT_SECRET=xxx go run sdk/go/examples/demo/demo.go
package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"encoding/base64"
	"strings"
)

var (
	ggidURL      = envOr("GGID_URL", "http://localhost:8080")
	clientID     = os.Getenv("CLIENT_ID")
	clientSecret = os.Getenv("CLIENT_SECRET")
	redirectURI  = envOr("REDIRECT_URI", "http://localhost:9090/callback")
	tenantID     = envOr("TENANT_ID", "00000000-0000-0000-0000-000000000001")
	port         = envOr("PORT", "9090")
)

func envOr(k, d string) string { if v := os.Getenv(k); v != "" { return v }; return d }

// Session stores user info after OAuth login
type Session struct {
	AccessToken string
	Scopes      []string
	Permissions []string
	Roles       []string
	UserInfo    map[string]any
}

var sessions = map[string]*Session{}

func main() {
	http.HandleFunc("/", handleHome)
	http.HandleFunc("/login", handleLogin)
	http.HandleFunc("/callback", handleCallback)
	http.HandleFunc("/logout", handleLogout)
	http.HandleFunc("/inventory", handleInventory)
	http.HandleFunc("/orders", handleOrders)
	http.HandleFunc("/admin", handleAdmin)
	log.Printf("GGID SDK Demo on :%s (GGID: %s)", port, ggidURL)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func httpClient() *http.Client {
	return &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
}

func getSession(r *http.Request) *Session {
	c, err := r.Cookie("session_id")
	if err != nil { return nil }
	return sessions[c.Value]
}

func hasScope(s *Session, scope string) bool {
	if s == nil { return false }
	for _, sc := range s.Scopes {
		if strings.EqualFold(sc, scope) || strings.EqualFold(sc, "admin") ||
			strings.EqualFold(sc, "platform:admin") || strings.EqualFold(sc, "platform administrator") {
			return true
		}
	}
	return false
}

func hasPermission(s *Session, resource, action string) bool {
	if s == nil { return false }
	// Admin has all permissions
	if hasScope(s, "admin") { return true }
	permKey := resource + ":" + action
	// Check fine-grained permissions claim first (new JWT structure)
	for _, p := range s.Permissions {
		if strings.EqualFold(p, permKey) { return true }
	}
	// Fallback: check scopes for backward compatibility (old JWT structure)
	for _, sc := range s.Scopes {
		if strings.EqualFold(sc, permKey) { return true }
	}
	return false
}

var tmpl = template.Must(template.New("").Parse(`<!DOCTYPE html><html><head><title>GGID SDK Demo</title>
<style>body{font-family:sans-serif;max-width:800px;margin:0 auto;padding:20px}
.nav a{margin-right:15px;text-decoration:none;color:#4f46e5}
.card{background:#f8fafc;border:1px solid #e2e8f0;border-radius:8px;padding:16px;margin:10px 0}
.badge{display:inline-block;padding:2px 8px;border-radius:12px;font-size:12px;background:#4f46e5;color:#fff;margin:2px}
.deny{text-align:center;padding:40px;color:#ef4444}
</style></head><body>
<h1>GGID SDK Demo</h1>
<nav class="nav">
<a href="/">Dashboard</a>
{{if .HasInventoryRead}}<a href="/inventory">Inventory</a>{{end}}
{{if .HasOrdersRead}}<a href="/orders">Orders</a>{{end}}
{{if .IsAdmin}}<a href="/admin">Admin</a>{{end}}
<a href="/logout" style="float:right">Logout</a>
</nav>
{{.Content}}
</body></html>`))

type pageData struct {
	IsAdmin         bool
	HasInventoryRead bool
	HasOrdersRead   bool
	Content         template.HTML
}

func renderPage(w http.ResponseWriter, s *Session, content string) {
	data := pageData{
		IsAdmin:          hasScope(s, "admin"),
		HasInventoryRead: hasPermission(s, "inventory", "read") || hasScope(s, "admin"),
		HasOrdersRead:    hasPermission(s, "orders", "read") || hasScope(s, "admin"),
		Content:          template.HTML(content),
	}
	tmpl.Execute(w, data)
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	s := getSession(r)
	if s == nil {
		renderPage(w, nil, `<div class="card"><h2>Welcome</h2><p><a href="/login" class="badge">Login with GGID</a></p></div>`)
		return
	}
	scopes := strings.Join(s.Scopes, `, `)
	email, _ := s.UserInfo["email"].(string)
	name, _ := s.UserInfo["name"].(string)
	if name == "" { name = email }
	content := fmt.Sprintf(`<div class="card"><h2>Welcome, %s!</h2><p>Email: %s</p><p>Scopes: %s</p>`, name, email, scopes)
	for _, sc := range s.Scopes {
		content += fmt.Sprintf(`<span class="badge">%s</span>`, sc)
	}
	content += `</div><div class="card"><h3>Permission Status</h3>`
	content += fmt.Sprintf(`<p>Inventory Read: %v | Inventory Write: %v</p>`, hasPermission(s, "inventory", "read"), hasPermission(s, "inventory", "write"))
	content += fmt.Sprintf(`<p>Orders Read: %v | Orders Write: %v</p>`, hasPermission(s, "orders", "read"), hasPermission(s, "orders", "write"))
	content += fmt.Sprintf(`<p>Admin: %v</p>`, hasScope(s, "admin"))
	content += `</div>`
	renderPage(w, s, content)
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	state := fmt.Sprintf("state_%d", os.Getpid())
	authURL := fmt.Sprintf("%s/api/v1/oauth/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=openid+profile+email&state=%s&tenant_id=%s",
		ggidURL, url.QueryEscape(clientID), url.QueryEscape(redirectURI), state, tenantID)
	http.Redirect(w, r, authURL, http.StatusFound)
}

func handleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" { http.Error(w, "no code", http.StatusBadRequest); return }

	form := url.Values{"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {redirectURI}, "client_id": {clientID}, "client_secret": {clientSecret}}
	req, _ := http.NewRequest("POST", ggidURL+"/api/v1/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Tenant-ID", tenantID)
	resp, err := httpClient().Do(req)
	if err != nil { http.Error(w, "token exchange failed", http.StatusInternalServerError); return }
	defer resp.Body.Close()
	var tok map[string]any; json.NewDecoder(resp.Body).Decode(&tok)

	accessToken, _ := tok["access_token"].(string)
	if accessToken == "" { http.Error(w, "no token", http.StatusUnauthorized); return }

	// Get user info + scopes/permissions/roles from JWT
	scopes := extractScopes(accessToken)
	permissions := extractPermissions(accessToken)
	roles := extractRoles(accessToken)
	uiReq, _ := http.NewRequest("GET", ggidURL+"/api/v1/oauth/userinfo", nil)
	uiReq.Header.Set("Authorization", "Bearer "+accessToken)
	uiResp, err := httpClient().Do(uiReq)
	if err != nil { http.Error(w, "userinfo failed", http.StatusInternalServerError); return }
	defer uiResp.Body.Close()
	body, _ := io.ReadAll(uiResp.Body)
	var userInfo map[string]any; json.Unmarshal(body, &userInfo)

	sessionID := fmt.Sprintf("sess_%d", os.Getpid())
	sessions[sessionID] = &Session{AccessToken: accessToken, Scopes: scopes, Permissions: permissions, Roles: roles, UserInfo: userInfo}
	http.SetCookie(w, &http.Cookie{Name: "session_id", Value: sessionID, Path: "/"})
	http.Redirect(w, r, "/", http.StatusFound)
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("session_id")
	if err == nil { delete(sessions, c.Value) }
	http.SetCookie(w, &http.Cookie{Name: "session_id", Value: "", Path: "/", MaxAge: -1})
	http.Redirect(w, r, "/", http.StatusFound)
}

func handleInventory(w http.ResponseWriter, r *http.Request) {
	s := getSession(r)
	if !hasPermission(s, "inventory", "read") && !hasScope(s, "admin") {
		renderPage(w, s, `<div class="deny"><h2>403 Forbidden</h2><p>You need inventory:read permission.</p></div>`)
		return
	}
	content := `<div class="card"><h2>Inventory</h2><p>Items list (read access granted).</p>`
	if hasPermission(s, "inventory", "write") || hasScope(s, "admin") {
		content += `<button class="badge">Create Item (inventory:write)</button>`
	}
	content += `</div>`
	renderPage(w, s, content)
}

func handleOrders(w http.ResponseWriter, r *http.Request) {
	s := getSession(r)
	if !hasPermission(s, "orders", "read") && !hasScope(s, "admin") {
		renderPage(w, s, `<div class="deny"><h2>403 Forbidden</h2><p>You need orders:read permission.</p></div>`)
		return
	}
	content := `<div class="card"><h2>Orders</h2><p>Order list (read access granted).</p>`
	if hasPermission(s, "orders", "write") || hasScope(s, "admin") {
		content += `<button class="badge">Create Order (orders:write)</button>`
	}
	content += `</div>`
	renderPage(w, s, content)
}

func handleAdmin(w http.ResponseWriter, r *http.Request) {
	s := getSession(r)
	if !hasScope(s, "admin") {
		renderPage(w, s, `<div class="deny"><h2>403 Forbidden</h2><p>Admin scope required.</p></div>`)
		return
	}
	renderPage(w, s, `<div class="card"><h2>Admin Panel</h2><p>Welcome, administrator. Full access granted.</p></div>`)
}

// extractClaimsFromToken parses JWT claims from the access token (no signature verification in demo)
func extractClaimsFromToken(token string) map[string]any {
	parts := strings.Split(token, ".")
	if len(parts) < 2 { return nil }
	payload, err := base64UrlDecode(parts[1])
	if err != nil { return nil }
	var claims map[string]any
	if json.Unmarshal(payload, &claims) != nil { return nil }
	return claims
}

// extractScopes parses OAuth scopes from the access token.
func extractScopes(token string) []string {
	claims := extractClaimsFromToken(token)
	if claims == nil { return []string{} }
	if raw, ok := claims["scopes"]; ok {
		if arr, ok := raw.([]any); ok {
			result := make([]string, 0, len(arr))
			for _, v := range arr { result = append(result, fmt.Sprintf("%v", v)) }
			return result
		}
	}
	if s, ok := claims["scope"].(string); ok && s != "" {
		return strings.Fields(s)
	}
	return []string{}
}

// extractPermissions parses fine-grained permissions from the JWT permissions claim.
func extractPermissions(token string) []string {
	claims := extractClaimsFromToken(token)
	if claims == nil { return []string{} }
	if raw, ok := claims["permissions"]; ok {
		if arr, ok := raw.([]any); ok {
			result := make([]string, 0, len(arr))
			for _, v := range arr { result = append(result, fmt.Sprintf("%v", v)) }
			return result
		}
	}
	return []string{}
}

// extractRoles parses role names from the JWT roles claim.
func extractRoles(token string) []string {
	claims := extractClaimsFromToken(token)
	if claims == nil { return []string{} }
	if raw, ok := claims["roles"]; ok {
		if arr, ok := raw.([]any); ok {
			result := make([]string, 0, len(arr))
			for _, v := range arr { result = append(result, fmt.Sprintf("%v", v)) }
			return result
		}
	}
	return []string{}
}

func base64UrlDecode(s string) ([]byte, error) {
	s = strings.ReplaceAll(s, "-", "+")
	s = strings.ReplaceAll(s, "_", "/")
	for len(s)%4 != 0 { s += "=" }
	return base64Decode(s)
}

func base64Decode(s string) ([]byte, error) { return base64.StdEncoding.DecodeString(s) }

func init() { _ = context.Background }
