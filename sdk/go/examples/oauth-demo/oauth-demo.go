// OAuth 2.0 Authorization Code flow demo using GGID IAM.
// Run: GGID_URL=https://ggid.iot2.win CLIENT_ID=xxx CLIENT_SECRET=xxx go run oauth-demo.go
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
	"strings"
)

var (
	ggidURL       = envOrDefault("GGID_URL", "http://localhost:8080")
	clientID      = os.Getenv("CLIENT_ID")
	clientSecret  = os.Getenv("CLIENT_SECRET")
	redirectURI   = envOrDefault("REDIRECT_URI", "http://localhost:9099/callback")
	tenantID      = envOrDefault("TENANT_ID", "00000000-0000-0000-0000-000000000001")
	port          = envOrDefault("PORT", "9099")
)

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	http.HandleFunc("/", handleHome)
	http.HandleFunc("/login", handleLogin)
	http.HandleFunc("/callback", handleCallback)
	http.HandleFunc("/logout", handleLogout)
	log.Printf("OAuth demo on :%s (GGID: %s)", port, ggidURL)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	userInfo := r.URL.Query().Get("user")
	tmpl := template.Must(template.New("").Parse(`<!DOCTYPE html><html><head><title>GGID OAuth Demo</title><style>body{font-family:sans-serif;max-width:600px;margin:40px auto;padding:20px}a.btn{display:inline-block;padding:10px 20px;background:#4f46e5;color:#fff;text-decoration:none;border-radius:6px}</style></head><body>
<h1>GGID OAuth Demo</h1>
{{if .}}<div><h3>Welcome!</h3><pre>{{.}}</pre><br><a href="/logout">Logout</a></div>{{else}}<a class="btn" href="/login">Login with GGID</a>{{end}}
</body></html>`))
	tmpl.Execute(w, userInfo)
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	state := "demo-state-123"
	authURL := fmt.Sprintf("%s/api/v1/oauth/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=openid+profile&state=%s",
		ggidURL, url.QueryEscape(clientID), url.QueryEscape(redirectURI), state)
	http.Redirect(w, r, authURL, http.StatusFound)
}

func handleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "no code", http.StatusBadRequest)
		return
	}

	// Exchange code for token
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
	}

	client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
	resp, err := client.PostForm(ggidURL+"/api/v1/oauth/token", form)
	if err != nil {
		http.Error(w, fmt.Sprintf("token exchange failed: %v", err), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var tokenResp map[string]interface{}
	body, _ := io.ReadAll(resp.Body)
	json.Unmarshal(body, &tokenResp)

	accessToken, _ := tokenResp["access_token"].(string)
	if accessToken == "" {
		http.Error(w, fmt.Sprintf("no access_token: %s", string(body)), http.StatusInternalServerError)
		return
	}

	// Get user info
	req, _ := http.NewRequest("GET", ggidURL+"/api/v1/oauth/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("X-Tenant-ID", tenantID)
	resp2, err := client.Do(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("userinfo failed: %v", err), http.StatusInternalServerError)
		return
	}
	defer resp2.Body.Close()
	userBody, _ := io.ReadAll(resp2.Body)

	http.Redirect(w, r, "/?user="+url.QueryEscape(string(userBody)), http.StatusFound)
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/", http.StatusFound)
}

func init() {
	if clientID == "" {
		log.Println("Warning: CLIENT_ID not set")
	}
	_ = context.Background
	_ = strings.TrimSpace
}
