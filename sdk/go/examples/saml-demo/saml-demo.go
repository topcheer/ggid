
package main

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
)

var (
	ggidURL  = envOr("GGID_URL", "http://localhost:8080")
	port     = envOr("PORT", "9091")
)

func envOr(k, d string) string { if v := os.Getenv(k); v != "" { return v }; return d }

type session struct {
	Email string
	Name  string
	Role  string
}

var sessions = map[string]*session{}
var tmpl = template.Must(template.New("").Parse(`<!DOCTYPE html><html><head><title>SAML Demo</title>
<style>body{font-family:sans-serif;max-width:700px;margin:40px auto;padding:20px}
.badge{display:inline-block;padding:2px 8px;border-radius:12px;font-size:12px;background:#4f46e5;color:#fff;margin:2px}
.deny{text-align:center;padding:40px;color:#ef4444}</style></head><body>
<h1>SAML SSO Demo</h1>
{{if .Email}}
<p>Welcome, <strong>{{.Name}}</strong> <span class="badge">{{.Role}}</span></p>
<p>Email: {{.Email}}</p>
<h3>Permissions</h3>
{{range .Perms}}<span class="badge">{{.}}</span> {{end}}
<div style="margin-top:16px">
{{if .CanInventory}}<p><a href="/inventory">Inventory</a> {{if .CanInvWrite}}[Create OK]{{end}}</p>{{else}}<p style="color:#999">Inventory: no permission</p>{{end}}
{{if .CanOrders}}<p><a href="/orders">Orders</a> {{if .CanOrderApprove}}[Approve OK]{{end}}</p>{{else}}<p style="color:#999">Orders: no permission</p>{{end}}
</div>
<p><a href="/logout">Logout</a></p>
{{else}}<p><a href="/saml/sso">Login via SAML SSO</a></p>{{end}}
</body></html>`))

func main() {
	http.HandleFunc("/", handleHome)
	http.HandleFunc("/saml/metadata", handleMetadata)
	http.HandleFunc("/saml/sso", handleSSO)
	http.HandleFunc("/saml/acs", handleACS)
	http.HandleFunc("/inventory", handleInventory)
	http.HandleFunc("/orders", handleOrders)
	http.HandleFunc("/logout", handleLogout)
	log.Printf("SAML demo on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func getSession(r *http.Request) *session {
	c, err := r.Cookie("saml_session")
	if err != nil { return nil }
	return sessions[c.Value]
}

func rolePerms(role string) []string {
	switch strings.ToLower(role) {
	case "sales manager": return []string{"orders:read","orders:write","orders:approve","inventory:read","reports:read"}
	case "warehouse manager": return []string{"orders:read","inventory:read","inventory:write","inventory:delete","reports:read"}
	case "finance officer": return []string{"orders:read","reports:read","reports:write","audit:read"}
	case "administrator","admin": return []string{"*"}
	default: return []string{}
	}
}

func hasPerm(role, resource, action string) bool {
	if strings.EqualFold(role,"Administrator")||strings.EqualFold(role,"admin") { return true }
	perms := rolePerms(role)
	target := resource + ":" + action
	for _, p := range perms { if p == target || p == "*" { return true } }
	return false
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	s := getSession(r)
	if s == nil {
		tmpl.Execute(w, map[string]any{})
		return
	}
	perms := rolePerms(s.Role)
	tmpl.Execute(w, map[string]any{
		"Email": s.Email, "Name": s.Name, "Role": s.Role, "Perms": perms,
		"CanInventory": hasPerm(s.Role,"inventory","read"), "CanInvWrite": hasPerm(s.Role,"inventory","write"),
		"CanOrders": hasPerm(s.Role,"orders","read"), "CanOrderApprove": hasPerm(s.Role,"orders","approve"),
	})
}

func handleMetadata(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/xml")
	fmt.Fprintf(w, `<?xml version="1.0"?><EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata" entityID="saml-demo"><SPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol"><NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress</NameIDFormat><AssertionConsumerService index="0" isDefault="true" Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" Location="http://localhost:%s/saml/acs"/></SPSSODescriptor></EntityDescriptor>`, port)
}

func handleSSO(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, ggidURL+"/saml/sso?RelayState=http://localhost:"+port+"/", http.StatusFound)
}

func handleACS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost { http.Error(w, "POST required", 405); return }
	r.ParseForm()
	samlResp := r.FormValue("SAMLResponse")
	if samlResp == "" { http.Error(w, "missing SAMLResponse", 400); return }
	xmlBytes, _ := base64.StdEncoding.DecodeString(samlResp)
	// Extract email from assertion (simplified — in production verify signature)
	xmlStr := string(xmlBytes)
	email := extractXMLValue(xmlStr, "NameID")
	if email == "" { email = extractXMLValue(xmlStr, "email") }
	name := extractXMLValue(xmlStr, "name")
	if name == "" { name = email }
	role := extractXMLValue(xmlStr, "role")
	if role == "" { role = "Viewer" }

	sid := fmt.Sprintf("s_%d", os.Getpid())
	sessions[sid] = &session{Email: email, Name: name, Role: role}
	http.SetCookie(w, &http.Cookie{Name: "saml_session", Value: sid, Path: "/"})
	http.Redirect(w, r, "/", http.StatusFound)
}

func handleInventory(w http.ResponseWriter, r *http.Request) {
	s := getSession(r)
	if s == nil { http.Redirect(w, r, "/", 302); return }
	if !hasPerm(s.Role, "inventory", "read") {
		fmt.Fprintf(w, `<html><body><div class="deny"><h1>403 Forbidden</h1><p>Need inventory:read</p><p>Role: %s</p></div></body></html>`)
		return
	}
	write := ""
	if hasPerm(s.Role, "inventory", "write") { write = " [Create enabled]" }
	fmt.Fprintf(w, `<html><body><h1>Inventory</h1><p>Items list.%s</p><p>Role: %s</p><a href="/">Back</a></body></html>`, write, s.Role)
}

func handleOrders(w http.ResponseWriter, r *http.Request) {
	s := getSession(r)
	if s == nil { http.Redirect(w, r, "/", 302); return }
	if !hasPerm(s.Role, "orders", "read") {
		fmt.Fprintf(w, `<html><body><div class="deny"><h1>403 Forbidden</h1><p>Need orders:read</p></div></body></html>`)
		return
	}
	approve := ""
	if hasPerm(s.Role, "orders", "approve") { approve = " [Approve enabled]" }
	fmt.Fprintf(w, `<html><body><h1>Orders</h1><p>Order list.%s</p><p>Role: %s</p><a href="/">Back</a></body></html>`, approve, s.Role)
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	c, _ := r.Cookie("saml_session")
	if c != nil { delete(sessions, c.Value) }
	http.SetCookie(w, &http.Cookie{Name: "saml_session", Value: "", Path: "/", MaxAge: -1})
	http.Redirect(w, r, "/", 302)
}

func extractXMLValue(xml, tag string) string {
	open := "<" + tag; close := "</" + tag + ">"
	start := strings.Index(xml, open)
	if start < 0 { return "" }
	start = strings.Index(xml[start:], ">")
	if start < 0 { return "" }
	start += len(xml[:0]) + 1
	end := strings.Index(xml[start:], close)
	if end < 0 { return "" }
	return strings.TrimSpace(xml[start : start+end])
}

func init() { _ = tls.Config{}; _ = json.Marshal }
