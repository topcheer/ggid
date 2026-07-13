package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ggid/ggid/pkg/truststore"
)

func TestTrustStoreCA_CRUD(t *testing.T) {
	h := NewTrustStoreHandler()

	// Generate a test cert
	certPEM, _, _, err := truststore.GenerateSelfSignedCert("test-ca.example.com")
	if err != nil {
		t.Fatalf("GenerateSelfSignedCert: %v", err)
	}

	// Upload CA
	body := `{"name":"Test CA","pem_data":"` + strings.ReplaceAll(certPEM, "\n", "\\n") + `"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/trust-store/cas", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.HandleTrustStoreCAs(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var ca map[string]any
	json.Unmarshal(w.Body.Bytes(), &ca)
	caID, _ := ca["id"].(string)
	if caID == "" {
		t.Fatal("expected non-empty CA id")
	}

	// List CAs
	req = httptest.NewRequest("GET", "/api/v1/auth/trust-store/cas", nil)
	w = httptest.NewRecorder()
	h.HandleTrustStoreCAs(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var listResp map[string]any
	json.Unmarshal(w.Body.Bytes(), &listResp)
	cas, _ := listResp["trusted_cas"].([]any)
	if len(cas) != 1 {
		t.Errorf("expected 1 CA, got %d", len(cas))
	}

	// Remove CA
	req = httptest.NewRequest("DELETE", "/api/v1/auth/trust-store/cas/"+caID, nil)
	w = httptest.NewRecorder()
	h.HandleTrustStoreCAs(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 on delete, got %d", w.Code)
	}

	// Verify removed
	req = httptest.NewRequest("GET", "/api/v1/auth/trust-store/cas", nil)
	w = httptest.NewRecorder()
	h.HandleTrustStoreCAs(w, req)

	json.Unmarshal(w.Body.Bytes(), &listResp)
	cas, _ = listResp["trusted_cas"].([]any)
	if len(cas) != 0 {
		t.Errorf("expected 0 CAs after removal, got %d", len(cas))
	}
}

func TestTrustStoreCA_InvalidPEM(t *testing.T) {
	h := NewTrustStoreHandler()

	body := `{"name":"Bad CA","pem_data":"not a certificate"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/trust-store/cas", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.HandleTrustStoreCAs(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 on invalid PEM, got %d", w.Code)
	}
}

func TestCertificateManagement_CRUD(t *testing.T) {
	h := NewTrustStoreHandler()

	certPEM, _, _, err := truststore.GenerateSelfSignedCert("managed.example.com")
	if err != nil {
		t.Fatalf("GenerateSelfSignedCert: %v", err)
	}

	// Upload certificate
	body := `{"name":"Managed Cert","type":"TLS","pem_data":"` + strings.ReplaceAll(certPEM, "\n", "\\n") + `"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/certificates", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.HandleCertificates(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var cert map[string]any
	json.Unmarshal(w.Body.Bytes(), &cert)
	certID, _ := cert["id"].(string)
	if certID == "" {
		t.Fatal("expected non-empty cert id")
	}

	// List certificates
	req = httptest.NewRequest("GET", "/api/v1/auth/certificates", nil)
	w = httptest.NewRecorder()
	h.HandleCertificates(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	// Get expiry tracker
	req = httptest.NewRequest("GET", "/api/v1/auth/certificates/expiry", nil)
	w = httptest.NewRecorder()
	h.HandleCertificates(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 on expiry, got %d", w.Code)
	}

	var expiryResp map[string]any
	json.Unmarshal(w.Body.Bytes(), &expiryResp)
	summary, _ := expiryResp["summary"].(map[string]any)
	if summary == nil {
		t.Fatal("expected summary in expiry response")
	}
	total, _ := summary["total"].(float64)
	if total != 1 {
		t.Errorf("expected total=1, got %v", total)
	}

	// Remove certificate
	req = httptest.NewRequest("DELETE", "/api/v1/auth/certificates/"+certID, nil)
	w = httptest.NewRecorder()
	h.HandleCertificates(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 on delete, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCertificateManagement_CSRGeneration(t *testing.T) {
	h := NewTrustStoreHandler()

	body := `{"cn":"test.example.com","organization":"Test Org","key_type":"rsa"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/certificates/csr", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.HandleCertificates(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	csr, _ := resp["csr"].(string)
	if !strings.Contains(csr, "CERTIFICATE REQUEST") {
		t.Error("expected CSR in response")
	}
	key, _ := resp["key_pem"].(string)
	if !strings.Contains(key, "PRIVATE KEY") {
		t.Error("expected private key in response")
	}
}

func TestMTLSConfig_GetAndUpdate(t *testing.T) {
	h := NewTrustStoreHandler()

	// Get default config
	req := httptest.NewRequest("GET", "/api/v1/auth/mtls/config", nil)
	w := httptest.NewRecorder()
	h.HandleMTLSConfig(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var config map[string]any
	json.Unmarshal(w.Body.Bytes(), &config)
	if config["require_mtls"] != false {
		t.Error("expected require_mtls=false by default")
	}

	// Update config
	body := `{"require_mtls":true,"revocation_check":"OCSP","allow_self_signed":true}`
	req = httptest.NewRequest("PUT", "/api/v1/auth/mtls/config", strings.NewReader(body))
	w = httptest.NewRecorder()
	h.HandleMTLSConfig(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 on update, got %d", w.Code)
	}

	// Verify updated
	req = httptest.NewRequest("GET", "/api/v1/auth/mtls/config", nil)
	w = httptest.NewRecorder()
	h.HandleMTLSConfig(w, req)

	json.Unmarshal(w.Body.Bytes(), &config)
	if config["require_mtls"] != true {
		t.Error("expected require_mtls=true after update")
	}
	if config["revocation_check"] != "OCSP" {
		t.Error("expected revocation_check=OCSP after update")
	}
}

func TestVerifyCert(t *testing.T) {
	h := NewTrustStoreHandler()

	certPEM, _, _, err := truststore.GenerateSelfSignedCert("verify.example.com")
	if err != nil {
		t.Fatalf("GenerateSelfSignedCert: %v", err)
	}

	body := `{"pem_data":"` + strings.ReplaceAll(certPEM, "\n", "\\n") + `"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/trust-store/verify", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.HandleVerifyCert(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}
