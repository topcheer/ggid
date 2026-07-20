package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Go SDK Demo — OAuth + SAML + Permission-aware application.
//
// Features:
//   - Login via OAuth authorization code flow
//   - Dashboard showing user info + role badges + permissions
//   - Inventory page (requires inventory:read)
//   - Orders page (requires orders:read, write button needs orders:write)
//   - Admin page (requires admin scope)
//   - 403 page for unauthorized access
//
// Run:
//   GGID_URL=http://localhost:8080 CLIENT_ID=gcid_xxx CLIENT_SECRET=gcs_xxx go run main.go

var ggidURL, clientID, clientSecret, redirectURI, port string

func main() {
	ggidURL = getenv("GGID_URL", "http://localhost:8080")
	clientID = getenv("CLIENT_ID", "")
	clientSecret = getenv("CLIENT_SECRET", "")
	redirectURI = getenv("REDIRECT_URI", "http://localhost:3001/auth/callback")
	port = getenv("PORT", "3001")

	if clientID == "" {
		fmt.Println("Warning: CLIENT_ID not set. Using demo mode.")
		clientID = "demo-client"
		clientSecret = "demo-secret"
	}

	// Routes
	http.HandleFunc("/", handleHome)
	http.HandleFunc("/auth/login", handleLogin)
	http.HandleFunc("/auth/callback", handleCallback)
	http.HandleFunc("/auth/logout", handleLogout)
	http.HandleFunc("/dashboard", withAuth(handleDashboard))
	http.HandleFunc("/inventory", withAuth(handleInventory))
	http.HandleFunc("/orders", withAuth(handleOrders))
	http.HandleFunc("/admin", withAuth(handleAdmin))

	fmt.Printf("Go SDK Demo running on :%s\nVisit http://localhost:%s/auth/login\n", port, port)
	http.ListenAndServe(":"+port, nil)
}

// --- Session Management ---

type UserSession struct {
	AccessToken string   `json:"access_token"`
	Username    string   `json:"username"`
	Email       string   `json:"email"`
	DisplayName string   `json:"display_name"`
	Scopes      []string `json:"scopes"`
	Roles       []string `json:"roles"`
}

func getSession(r *http.Request) *UserSession {
	cookie, err := r.Cookie("ggid_session")
	if err != nil {
		return nil
	}
	var sess UserSession
	if err := json.Unmarshal([]byte(cookie.Value), &sess); err != nil {
		return nil
	}
	return &sess
}

func setSession(w http.ResponseWriter, sess *UserSession) {
	data, _ := json.Marshal(sess)
	http.SetCookie(w, &http.Cookie{
		Name:    "ggid_session",
		Value:   string(data),
		Path:    "/",
		MaxAge:  3600,
	})
}

func clearSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:   "ggid_session",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
}

// --- Permission Helpers ---

func (s *UserSession) hasScope(scope string) bool {
	for _, sc := range s.Scopes {
		if sc == scope || sc == "platform:admin" || sc == "admin" {
			return true
		}
	}
	return false
}

func (s *UserSession) hasRole(role string) bool {
	for _, r := range s.Roles {
		if strings.EqualFold(r, role) {
			return true
		}
	}
	return false
}

func (s *UserSession) hasPermission(perm string) bool {
	// Admin has all permissions
	if s.hasScope("platform:admin") || s.hasScope("admin") || s.hasScope("tenant:admin") {
		return true
	}
	// Check role-based permissions
	switch perm {
	case "inventory:read":
		return s.hasRole("warehouse_manager") || s.hasRole("sales_manager") || s.hasRole("erp_admin")
	case "inventory:write":
		return s.hasRole("warehouse_manager") || s.hasRole("erp_admin")
	case "orders:read":
		return true // All authenticated users can read orders
	case "orders:write":
		return s.hasRole("sales_manager") || s.hasRole("warehouse_manager") || s.hasRole("erp_admin")
	case "orders:approve":
		return s.hasRole("sales_manager") || s.hasRole("erp_admin")
	case "reports:read":
		return s.hasRole("sales_manager") || s.hasRole("finance_officer") || s.hasRole("erp_admin")
	case "admin":
		return s.hasScope("platform:admin") || s.hasScope("admin") || s.hasScope("tenant:admin")
	}
	return false
}

// --- Middleware ---

func withAuth(handler func(http.ResponseWriter, *http.Request, *UserSession)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess := getSession(r)
		if sess == nil {
			http.Redirect(w, r, "/auth/login", http.StatusFound)
			return
		}
		handler(w, r, sess)
	}
}

// --- Handlers ---

func handleHome(w http.ResponseWriter, r *http.Request) {
	sess := getSession(r)
	if sess != nil {
		http.Redirect(w, r, "/dashboard", http.StatusFound)
		return
	}
	renderTemplate(w, "home", map[string]any{
		"GGIDURL":  ggidURL,
		"ClientID": clientID,
	})
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	url := fmt.Sprintf("%s/oauth/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=openid%%20profile%%20email",
		ggidURL, clientID, redirectURI)
	http.Redirect(w, r, url, http.StatusFound)
}

func handleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing authorization code", http.StatusBadRequest)
		return
	}

	// Exchange code for token
	tokenURL := fmt.Sprintf("%s/api/v1/oauth/token", ggidURL)
	resp, err := http.PostForm(tokenURL, map[string][]string{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"redirect_uri":  {redirectURI},
	})
	if err != nil {
		http.Error(w, "token exchange failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	var tokenResp map[string]any
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		http.Error(w, "failed to parse token response", http.StatusInternalServerError)
		return
	}

	accessToken, _ := tokenResp["access_token"].(string)
	if accessToken == "" {
		http.Error(w, "no access token in response", http.StatusUnauthorized)
		return
	}

	// Fetch user info
	sess := &UserSession{
		AccessToken: accessToken,
		Scopes:      []string{},
		Roles:       []string{},
	}

	// Decode JWT to get user info (simplified — in production use SDK.VerifyToken)
	if parts := strings.Split(accessToken, "."); len(parts) >= 2 {
		// Decode payload
		payload := parts[1]
		// Add padding
		for len(payload)%4 != 0 {
			payload += "="
		}
		if claims, err := decodeJWTPayload(payload); err == nil {
			sess.Username, _ = claims["sub"].(string)
			if scopes, ok := claims["scopes"].([]any); ok {
				for _, s := range scopes {
					if str, ok := s.(string); ok {
						sess.Scopes = append(sess.Scopes, str)
						sess.Roles = append(sess.Roles, str)
					}
				}
			}
		}
	}

	// Try to get user info from API
	userURL := fmt.Sprintf("%s/api/v1/users/me", ggidURL)
	if req, err := http.NewRequest("GET", userURL, nil); err == nil {
		req.Header.Set("Authorization", "Bearer "+accessToken)
		if resp, err := http.DefaultClient.Do(req); err == nil {
			var user map[string]any
			json.NewDecoder(resp.Body).Decode(&user)
			resp.Body.Close()
			if dn, ok := user["display_name"].(string); ok && dn != "" {
				sess.DisplayName = dn
			}
			if email, ok := user["email"].(string); ok {
				sess.Email = email
			}
			if un, ok := user["username"].(string); ok && un != "" {
				sess.Username = un
			}
		}
	}

	if sess.DisplayName == "" {
		sess.DisplayName = sess.Username
	}

	setSession(w, sess)
	http.Redirect(w, r, "/dashboard", http.StatusFound)
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	clearSession(w)
	http.Redirect(w, r, "/", http.StatusFound)
}

func handleDashboard(w http.ResponseWriter, r *http.Request, sess *UserSession) {
	// Build permission list for display
	perms := []string{}
	for _, p := range []string{"inventory:read", "inventory:write", "orders:read", "orders:write", "orders:approve", "reports:read", "admin"} {
		if sess.hasPermission(p) {
			perms = append(perms, "✓ "+p)
		} else {
			perms = append(perms, "✗ "+p)
		}
	}

	// Build menu items based on permissions
	menu := []map[string]any{}
	menu = append(menu, map[string]any{"url": "/dashboard", "label": "Dashboard", "visible": true})
	menu = append(menu, map[string]any{"url": "/orders", "label": "Orders", "visible": sess.hasPermission("orders:read")})
	menu = append(menu, map[string]any{"url": "/inventory", "label": "Inventory", "visible": sess.hasPermission("inventory:read")})
	menu = append(menu, map[string]any{"url": "/admin", "label": "Admin", "visible": sess.hasPermission("admin")})

	renderTemplate(w, "dashboard", map[string]any{
		"Session":     sess,
		"Permissions": perms,
		"Menu":        menu,
	})
}

func handleInventory(w http.ResponseWriter, r *http.Request, sess *UserSession) {
	if !sess.hasPermission("inventory:read") {
		renderTemplate(w, "403", map[string]any{
			"Permission": "inventory:read",
			"Session":    sess,
		})
		return
	}

	canWrite := sess.hasPermission("inventory:write")
	renderTemplate(w, "inventory", map[string]any{
		"Session":  sess,
		"CanWrite": canWrite,
	})
}

func handleOrders(w http.ResponseWriter, r *http.Request, sess *UserSession) {
	if !sess.hasPermission("orders:read") {
		renderTemplate(w, "403", map[string]any{
			"Permission": "orders:read",
			"Session":    sess,
		})
		return
	}

	canWrite := sess.hasPermission("orders:write")
	canApprove := sess.hasPermission("orders:approve")
	renderTemplate(w, "orders", map[string]any{
		"Session":    sess,
		"CanWrite":   canWrite,
		"CanApprove": canApprove,
	})
}

func handleAdmin(w http.ResponseWriter, r *http.Request, sess *UserSession) {
	if !sess.hasPermission("admin") {
		renderTemplate(w, "403", map[string]any{
			"Permission": "admin",
			"Session":    sess,
		})
		return
	}

	renderTemplate(w, "admin", map[string]any{
		"Session": sess,
	})
}

// --- Helpers ---

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func decodeJWTPayload(payload string) (map[string]any, error) {
	decoded := make([]byte, len(payload))
	n, err := base64Decode([]byte(payload), decoded)
	if err != nil {
		return nil, err
	}
	var claims map[string]any
	if err := json.Unmarshal(decoded[:n], &claims); err != nil {
		return nil, err
	}
	return claims, nil
}

// Simple base64url decoder
func base64Decode(src, dst []byte) (int, error) {
	// Replace URL-safe chars
	s := strings.ReplaceAll(string(src), "-", "+")
	s = strings.ReplaceAll(s, "_", "/")
	// Pad
	for len(s)%4 != 0 {
		s += "="
	}
	n := copy(dst, s)
	_ = n
	idx := 0
	for i := 0; i < len(s); i += 4 {
		if i+4 > len(s) {
			break
		}
		var val int
		for j := 0; j < 4; j++ {
			val <<= 6
			c := s[i+j]
			switch {
			case c >= 'A' && c <= 'Z':
				val += int(c - 'A')
			case c >= 'a' && c <= 'z':
				val += int(c - 'a' + 26)
			case c >= '0' && c <= '9':
				val += int(c - '0' + 52)
			case c == '+':
				val += 62
			case c == '/':
				val += 63
			case c == '=':
				val <<= 6
				continue
			}
		}
		if i+4 <= len(s) {
			dst[idx] = byte(val >> 16)
			dst[idx+1] = byte(val >> 8)
			dst[idx+2] = byte(val)
			idx += 3
		}
	}
	_ = n
	return idx, nil
}

func renderTemplate(w http.ResponseWriter, name string, data map[string]any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl, ok := templates[name]
	if !ok {
		http.Error(w, "template not found: "+name, http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// --- Templates ---

var templates = map[string]*template.Template{
	"home": template.Must(template.New("").Parse(`
<!DOCTYPE html><html><head><title>GGID SDK Demo</title>
<style>body{font-family:sans-serif;margin:40px;max-width:600px}
a{color:#3b82f6}</style></head><body>
<h1>🔐 GGID SDK Demo (Go)</h1>
<p>GGID: {{.GGIDURL}}</p>
<p><a href="/auth/login">Login with GGID →</a></p>
</body></html>
`)),
	"dashboard": template.Must(template.New("").Funcs(template.FuncMap{
		"hasAttr": func(m map[string]any, key string) bool { _, ok := m[key]; return ok },
	}).Parse(`
<!DOCTYPE html><html><head><title>Dashboard — GGID Demo</title>
<style>
body{font-family:sans-serif;margin:0}
.nav{background:#1f2937;padding:12px 24px;display:flex;gap:16px}
.nav a{color:#d1d5db;text-decoration:none}
.nav a:hover{color:#fff}
.badge{background:#3b82f6;color:#fff;padding:2px 8px;border-radius:4px;font-size:12px;margin:2px}
.content{padding:24px;max-width:800px}
table{border-collapse:collapse;width:100%}
td,th{border:1px solid #e5e7eb;padding:8px;text-align:left}
.perm-yes{color:#16a34a}.perm-no{color:#dc2626}
</style></head><body>
<div class="nav">
{{range .Menu}}{{if .visible}}<a href="{{.url}}">{{.label}}</a>{{end}}{{end}}
<a href="/auth/logout" style="margin-left:auto;color:#f87171">Logout</a>
</div>
<div class="content">
<h1>📊 Dashboard</h1>
<p>Welcome, <strong>{{.Session.DisplayName}}</strong></p>
<p>Email: {{.Session.Email}}</p>
<p>Scopes: {{range .Session.Scopes}}<span class="badge">{{.}}</span>{{end}}</p>
<h3>Permissions</h3>
<table>
{{range .Permissions}}<tr><td class="{{if eq (slice . 0 1) "✓"}}perm-yes{{else}}perm-no{{end}}">{{.}}</td></tr>{{end}}
</table>
</div>
</body></html>
`)),
	"inventory": template.Must(template.New("").Parse(`
<!DOCTYPE html><html><head><title>Inventory — GGID Demo</title>
<style>
body{font-family:sans-serif;margin:0}
.nav{background:#1f2937;padding:12px 24px;display:flex;gap:16px}
.nav a{color:#d1d5db;text-decoration:none}
.content{padding:24px}
.btn{background:#3b82f6;color:#fff;padding:8px 16px;border:none;border-radius:4px;cursor:pointer}
</style></head><body>
<div class="nav">
<a href="/dashboard">Dashboard</a>
<a href="/orders">Orders</a>
<a href="/inventory"><strong>Inventory</strong></a>
{{if .Session.hasPermission "admin"}}<a href="/admin">Admin</a>{{end}}
<a href="/auth/logout" style="margin-left:auto;color:#f87171">Logout</a>
</div>
<div class="content">
<h1>📦 Inventory</h1>
<p>Logged in as: {{.Session.DisplayName}}</p>
{{if .CanWrite}}
<button class="btn" onclick="alert('Create item (demo)')">+ New Item</button>
{{else}}
<p><em>You have read-only access to inventory.</em></p>
{{end}}
<table border="1" style="border-collapse:collapse;margin-top:12px">
<tr><th>SKU</th><th>Name</th><th>Stock</th>{{if .CanWrite}}<th>Actions</th>{{end}}</tr>
<tr><td>SKU-001</td><td>Widget A</td><td>150</td>{{if .CanWrite}}<td><button>Edit</button> <button>Delete</button></td>{{end}}</tr>
<tr><td>SKU-002</td><td>Widget B</td><td>75</td>{{if .CanWrite}}<td><button>Edit</button> <button>Delete</button></td>{{end}}</tr>
</table>
</div>
</body></html>
`)),
	"orders": template.Must(template.New("").Parse(`
<!DOCTYPE html><html><head><title>Orders — GGID Demo</title>
<style>
body{font-family:sans-serif;margin:0}
.nav{background:#1f2937;padding:12px 24px;display:flex;gap:16px}
.nav a{color:#d1d5db;text-decoration:none}
.content{padding:24px}
.btn{background:#3b82f6;color:#fff;padding:8px 16px;border:none;border-radius:4px;cursor:pointer;margin:2px}
.btn-approve{background:#16a34a}
</style></head><body>
<div class="nav">
<a href="/dashboard">Dashboard</a>
<a href="/orders"><strong>Orders</strong></a>
<a href="/inventory">Inventory</a>
{{if .Session.hasPermission "admin"}}<a href="/admin">Admin</a>{{end}}
<a href="/auth/logout" style="margin-left:auto;color:#f87171">Logout</a>
</div>
<div class="content">
<h1>📋 Orders</h1>
{{if .CanWrite}}<button class="btn" onclick="alert('Create order (demo)')">+ New Order</button>{{end}}
<table border="1" style="border-collapse:collapse;margin-top:12px">
<tr><th>Order #</th><th>Customer</th><th>Total</th><th>Status</th>{{if .CanApprove}}<th>Actions</th>{{end}}</tr>
<tr><td>ORD-001</td><td>Acme Corp</td><td>$1,200</td><td>Pending</td>{{if .CanApprove}}<td><button class="btn btn-approve">Approve</button> <button class="btn">Ship</button></td>{{end}}</tr>
<tr><td>ORD-002</td><td>Globex</td><td>$850</td><td>Shipped</td>{{if .CanApprove}}<td>—</td>{{end}}</tr>
</table>
{{if not .CanWrite}}<p><em>Read-only access.</em></p>{{end}}
</div>
</body></html>
`)),
	"admin": template.Must(template.New("").Parse(`
<!DOCTYPE html><html><head><title>Admin — GGID Demo</title>
<style>
body{font-family:sans-serif;margin:0}
.nav{background:#1f2937;padding:12px 24px;display:flex;gap:16px}
.nav a{color:#d1d5db;text-decoration:none}
.content{padding:24px}
</style></head><body>
<div class="nav">
<a href="/dashboard">Dashboard</a>
<a href="/orders">Orders</a>
<a href="/inventory">Inventory</a>
<a href="/admin"><strong>Admin</strong></a>
<a href="/auth/logout" style="margin-left:auto;color:#f87171">Logout</a>
</div>
<div class="content">
<h1>⚙️ Admin Panel</h1>
<p>Welcome, administrator {{.Session.DisplayName}}</p>
<p>This page is only visible to users with admin scope.</p>
<ul>
<li>User Management</li>
<li>System Settings</li>
<li>Audit Logs</li>
</ul>
</div>
</body></html>
`)),
	"403": template.Must(template.New("").Parse(`
<!DOCTYPE html><html><head><title>403 — Access Denied</title>
<style>
body{font-family:sans-serif;display:flex;justify-content:center;align-items:center;min-height:90vh;margin:0}
.card{text-align:center;padding:40px;border:1px solid #fecaca;border-radius:8px;background:#fef2f2}
</style></head><body>
<div class="card">
<h1 style="color:#dc2626">🚫 403 — Access Denied</h1>
<p>You do not have permission to access this page.</p>
<p>Required permission: <code>{{.Permission}}</code></p>
<p>Logged in as: {{.Session.DisplayName}}</p>
<p><a href="/dashboard">← Back to Dashboard</a></p>
</div>
</body></html>
`)),
}

// Unused but needed for compilation
var _ = time.Now
