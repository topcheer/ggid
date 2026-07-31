package router

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ggid/ggid/services/gateway/internal/middleware"
	"github.com/golang-jwt/jwt/v5"
)

// testAdminKey is a throwaway RSA key for signing admin test JWTs, with the
// matching public key written to testAdminPubPath for the JWKS client.
// Signed tokens are required since R226 P0 removed unsigned JWT parsing.
var (
	testAdminKey, testAdminPubPath = func() (*rsa.PrivateKey, string) {
		privKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			panic("generate admin test key: " + err.Error())
		}
		pubDER, err := x509.MarshalPKIXPublicKey(&privKey.PublicKey)
		if err != nil {
			panic("marshal admin test pubkey: " + err.Error())
		}
		pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})
		path := filepath.Join(os.TempDir(), "ggid_test_admin_pub.pem")
		if err := os.WriteFile(path, pubPEM, 0o600); err != nil {
			panic("write admin test pubkey: " + err.Error())
		}
		return privKey, path
	}()

	// adminAuthHeader is a signature-verified Bearer token with admin scopes.
	adminAuthHeader = func() string {
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
			"sub":   "admin-user",
			"scope": "platform:admin tenant:admin",
		})
		signed, err := token.SignedString(testAdminKey)
		if err != nil {
			panic("sign admin test JWT: " + err.Error())
		}
		return "Bearer " + signed
	}()
)

// adminRequest creates a request with admin JWT auth header.
func adminRequest(method, url string) *http.Request {
	r := httptest.NewRequest(method, url, nil)
	r.Header.Set("Authorization", adminAuthHeader)
	return r
}

// mustTestJWKS builds a JWKS client wired to the shared test public key,
// so signed test JWTs verify (R226 P0 removed unsigned JWT parsing).
func mustTestJWKS(t *testing.T) *middleware.JWKSClient {
	t.Helper()
	jwks, err := middleware.NewJWKSClient("", testAdminPubPath)
	if err != nil {
		t.Fatalf("failed to create JWKS client: %v", err)
	}
	return jwks
}

func TestAdminRoutes_ListRoutes(t *testing.T) {
	gw := newTestGateway(t)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, adminRequest("GET", "/api/v1/admin/routes"))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	routes, ok := resp["routes"].([]any)
	if !ok {
		t.Fatal("expected routes array")
	}
	if len(routes) == 0 {
		t.Error("expected at least 1 route")
	}
}

func TestAdminStats_ReturnsBackends(t *testing.T) {
	gw := newTestGateway(t)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, adminRequest("GET", "/api/v1/admin/stats"))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp AdminStatsResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp.Backends) == 0 {
		t.Error("expected at least 1 backend")
	}
}

func TestAdminToggleRoute_Disable(t *testing.T) {
	gw := newTestGateway(t)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, adminRequest("POST", "/api/v1/admin/routes//api/v1/users/toggle"))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["enabled"] != false {
		t.Errorf("expected enabled=false, got %v", resp["enabled"])
	}
}

func TestAdminToggleRoute_Enable(t *testing.T) {
	gw := newTestGateway(t)
	w1 := httptest.NewRecorder()
	gw.ServeHTTP(w1, adminRequest("POST", "/api/v1/admin/routes//api/v1/users/toggle"))

	w2 := httptest.NewRecorder()
	gw.ServeHTTP(w2, adminRequest("POST", "/api/v1/admin/routes//api/v1/users/toggle"))

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w2.Code)
	}
	var resp map[string]any
	json.NewDecoder(w2.Body).Decode(&resp)
	if resp["enabled"] != true {
		t.Errorf("expected enabled=true, got %v", resp["enabled"])
	}
}

func TestAdminToggleRoute_NotFound(t *testing.T) {
	gw := newTestGateway(t)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, adminRequest("POST", "/api/v1/admin/routes//nonexistent/toggle"))

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestAdminRoutes_ForbiddenWithoutAdminScope(t *testing.T) {
	gw := newTestGateway(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/admin/routes", nil)
	gw.ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 without admin scope, got %d", w.Code)
	}
}
