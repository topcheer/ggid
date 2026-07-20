package main

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/ggid/ggid/pkg/saml"
)

var (
	ggidBaseURL string
	listenAddr  string
	spEntityID  string
	spACSURL    string
	idpCertPEM  = ""
	idpCert     *x509.Certificate
)

type PageData struct {
	Title   string
	Message string
	User    *UserInfo
}

type UserInfo struct {
	NameID  string
	Email   string
	Name    string
	Groups  []string
}

func main() {
	// Read configuration from environment variables (no hardcoded URLs)
	ggidBaseURL = os.Getenv("GGID_URL")
	if ggidBaseURL == "" {
		log.Fatal("GGID_URL environment variable is required (e.g. https://ggid.example.com)")
	}
	listenAddr = os.Getenv("LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = ":9090"
	}
	spEntityID = os.Getenv("SP_ENTITY_ID")
	if spEntityID == "" {
		spEntityID = ggidBaseURL + "/saml"
	}
	spACSURL = os.Getenv("SP_ACS_URL")
	if spACSURL == "" {
		spACSURL = "http://localhost" + listenAddr + "/acs"
	}

	// Try to load IdP certificate from GGID metadata
	loadIdPCert()

	http.HandleFunc("/", handleHome)
	http.HandleFunc("/login", handleLogin)
	http.HandleFunc("/acs", handleACS)
	http.HandleFunc("/logout", handleLogout)

	fmt.Printf("SAML Demo App running on http://localhost%s\n", listenAddr)
	fmt.Printf("GGID SAML SSO URL: %s/saml/sso\n", ggidBaseURL)
	fmt.Printf("ACS URL: %s\n", spACSURL)
	fmt.Printf("SP Entity ID: %s\n", spEntityID)
	log.Fatal(http.ListenAndServe(listenAddr, nil))
}

func loadIdPCert() {
	// Fetch IdP metadata from GGID
	resp, err := http.Get(ggidBaseURL + "/saml/idp/metadata")
	if err != nil {
		log.Printf("Warning: cannot fetch IdP metadata: %v", err)
		return
	}
	defer resp.Body.Close()

	var md struct {
		XMLName xml.Name `xml:"EntityDescriptor"`
		IDPSSODescriptor struct {
			KeyDescriptor []struct {
				Use string `xml:"use,attr"`
				KeyInfo struct {
					X509Data struct {
						X509Certificate string `xml:"X509Certificate"`
					} `xml:"X509Data"`
				} `xml:"KeyInfo"`
			} `xml:"KeyDescriptor"`
		} `xml:"IDPSSODescriptor"`
	}

	if err := xml.NewDecoder(resp.Body).Decode(&md); err != nil {
		log.Printf("Warning: cannot parse IdP metadata: %v", err)
		return
	}

	for _, kd := range md.IDPSSODescriptor.KeyDescriptor {
		if kd.Use == "signing" && kd.KeyInfo.X509Data.X509Certificate != "" {
			certPEM := "-----BEGIN CERTIFICATE-----\n" + kd.KeyInfo.X509Data.X509Certificate + "\n-----END CERTIFICATE-----"
			idpCertPEM = certPEM
			if cert, err := parseCert(certPEM); err == nil {
				idpCert = cert
				log.Printf("IdP certificate loaded successfully")
			}
		}
	}
}

func parseCert(pem string) (*x509.Certificate, error) {
	// Remove PEM headers
	b64 := strings.ReplaceAll(strings.ReplaceAll(pem, "-----BEGIN CERTIFICATE-----", ""), "-----END CERTIFICATE-----", "")
	b64 = strings.TrimSpace(b64)
	der, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(der)
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	tmpl := template.Must(template.New("page").Parse(homeTemplate))
	tmpl.Execute(w, PageData{Title: "SAML Demo App", User: user})
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	// Redirect to GGID SAML SSO
	ssoURL := fmt.Sprintf("%s/saml/sso?relay_state=%s", ggidBaseURL, spACSURL)
	http.Redirect(w, r, ssoURL, http.StatusFound)
}

func handleACS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	samlResponse := r.FormValue("SAMLResponse")
	if samlResponse == "" {
		renderError(w, "No SAMLResponse received")
		return
	}

	// Decode SAMLResponse
	rawXML, err := base64.StdEncoding.DecodeString(samlResponse)
	if err != nil {
		renderError(w, "Failed to decode SAMLResponse: "+err.Error())
		return
	}

	// Parse assertion
	assertion, err := saml.ParseAssertion(rawXML)
	if err != nil {
		renderError(w, "Failed to parse SAML assertion: "+err.Error())
		return
	}

	// Verify signature if cert available
	if idpCert != nil {
		if err := saml.ValidateSignature(assertion, idpCert); err != nil {
			renderError(w, "Signature validation failed: "+err.Error())
			return
		}
	}

	// Extract attributes
	attrs := saml.ExtractAttributes(assertion)
	user := &UserInfo{
		NameID: assertion.Subject.NameID,
		Email: saml.GetAttribute(assertion, "email"),
		Name:  saml.GetAttribute(assertion, "name"),
	}
	if groups, ok := attrs["groups"]; ok {
		user.Groups = groups
	}

	// Set session cookie
	setSession(w, user.NameID)

	// Show success page
	tmpl := template.Must(template.New("page").Parse(successTemplate))
	tmpl.Execute(w, PageData{Title: "SAML Login Success", User: user, Message: "SAML authentication successful!"})
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	clearSession(w)
	http.Redirect(w, r, "/", http.StatusFound)
}

func getUserFromSession(r *http.Request) *UserInfo {
	cookie, err := r.Cookie("saml_session")
	if err != nil || cookie.Value == "" {
		return nil
	}
	// Simple session: cookie value is the NameID
	return &UserInfo{NameID: cookie.Value}
}

func setSession(w http.ResponseWriter, nameID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "saml_session",
		Value:    nameID,
		Path:     "/",
		MaxAge:   3600,
		HttpOnly: true,
	})
}

func clearSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:   "saml_session",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
}

func renderError(w http.ResponseWriter, msg string) {
	tmpl := template.Must(template.New("page").Parse(errorTemplate))
	tmpl.Execute(w, PageData{Title: "SAML Error", Message: msg})
}

const homeTemplate = `<!DOCTYPE html>
<html>
<head><title>{{.Title}}</title>
<style>body{font-family:system-ui;max-width:600px;margin:50px auto;padding:20px} .card{border:1px solid #ddd;border-radius:8px;padding:24px} .btn{background:#4f46e5;color:white;padding:10px 20px;border-radius:6px;text-decoration:none;display:inline-block;margin:8px 0} .info{background:#f0f9ff;padding:12px;border-radius:6px;margin:12px 0} .logout{background:#dc2626}</style>
</head>
<body>
<div class="card">
<h1>{{.Title}}</h1>
<p>Demonstrates SAML 2.0 SSO integration with GGID IAM.</p>
{{if .User}}
<div class="info">✅ Logged in as: <strong>{{.User.NameID}}</strong></div>
<a href="/logout" class="btn logout">Logout</a>
{{else}}
<div class="info">Not logged in. Click "Login with GGID SSO" to authenticate.</div>
<a href="/login" class="btn">Login with GGID SSO</a>
{{end}}
</div>
</body>
</html>`

const successTemplate = `<!DOCTYPE html>
<html>
<head><title>{{.Title}}</title>
<style>body{font-family:system-ui;max-width:600px;margin:50px auto;padding:20px} .card{border:1px solid #ddd;border-radius:8px;padding:24px} .success{color:#16a34a;font-size:18px} .info{background:#f0f9ff;padding:12px;border-radius:6px;margin:12px 0} .btn{background:#4f46e5;color:white;padding:10px 20px;border-radius:6px;text-decoration:none;display:inline-block}</style>
</head>
<body>
<div class="card">
<h1>{{.Title}}</h1>
<p class="success">✅ {{.Message}}</p>
<div class="info">
<p><strong>Name ID:</strong> {{.User.NameID}}</p>
{{if .User.Email}}<p><strong>Email:</strong> {{.User.Email}}</p>{{end}}
{{if .User.Name}}<p><strong>Name:</strong> {{.User.Name}}</p>{{end}}
{{if .User.Groups}}<p><strong>Groups:</strong> {{range .User.Groups}}<span style="background:#e0e7ff;padding:2px 6px;border-radius:4px;margin:2px">{{.}}</span> {{end}}</p>{{end}}
</div>
<a href="/" class="btn">Go to Dashboard</a>
</div>
</body>
</html>`

const errorTemplate = `<!DOCTYPE html>
<html>
<head><title>{{.Title}}</title>
<style>body{font-family:system-ui;max-width:600px;margin:50px auto;padding:20px} .card{border:1px solid #ddd;border-radius:8px;padding:24px} .error{color:#dc2626;font-size:18px} .btn{background:#4f46e5;color:white;padding:10px 20px;border-radius:6px;text-decoration:none;display:inline-block}</style>
</head>
<body>
<div class="card">
<h1>{{.Title}}</h1>
<p class="error">❌ {{.Message}}</p>
<a href="/" class="btn">Back to Home</a>
</div>
</body>
</html>`
