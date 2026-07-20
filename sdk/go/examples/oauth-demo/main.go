package main

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
)

// Go OAuth 2.0 demo application.
// Demonstrates: login → authorization code flow → get token → call API → show user info.
//
// Run:
//   GGID_URL=http://localhost:8080 CLIENT_ID=gcid_xxx CLIENT_SECRET=gcs_xxx go run main.go
func main() {
	ggidURL := getenv("GGID_URL", "http://localhost:8080")
	clientID := getenv("CLIENT_ID", "")
	clientSecret := getenv("CLIENT_SECRET", "")
	redirectURI := getenv("REDIRECT_URI", "http://localhost:3001/auth/callback")
	port := getenv("PORT", "3001")

	if clientID == "" || clientSecret == "" {
		fmt.Println("Warning: CLIENT_ID and CLIENT_SECRET not set. Using demo mode.")
		clientID = "demo-client"
		clientSecret = "demo-secret"
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t := template.Must(template.New("").Parse(homeTmpl))
		t.Execute(w, map[string]string{"GGIDURL": ggidURL, "ClientID": clientID})
	})

	http.HandleFunc("/auth/login", func(w http.ResponseWriter, r *http.Request) {
		url := fmt.Sprintf("%s/oauth/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=openid%%20profile%%20email",
			ggidURL, clientID, redirectURI)
		http.Redirect(w, r, url, http.StatusFound)
	})

	http.HandleFunc("/auth/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "missing code", http.StatusBadRequest)
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
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "<h1>OAuth Success</h1><p>Token response status: %d</p><p>Check server logs for token details.</p>", resp.StatusCode)
	})

	fmt.Printf("Go OAuth demo running on :%s\nVisit http://localhost:%s/auth/login\n", port, port)
	http.ListenAndServe(":"+port, nil)
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

const homeTmpl = `
<!DOCTYPE html><html><body>
<h1>Go OAuth 2.0 Demo</h1>
<p>GGID URL: {{.GGIDURL}}</p>
<p>Client ID: {{.ClientID}}</p>
<p><a href="/auth/login">Login with GGID</a></p>
</body></html>
`
