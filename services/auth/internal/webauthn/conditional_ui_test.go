package webauthn

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestConditionalUI_BeginBasic(t *testing.T) {
	req := &ConditionalUIRequest{
		Challenge: "test-challenge",
		RPID:      "example.com",
		UserID:    "user-123",
	}
	resp := BeginConditionalUI(req, [][]byte{[]byte("cred-1"), []byte("cred-2")})

	if resp.Mediation != "conditional" {
		t.Error("should use conditional mediation")
	}
	if resp.PublicKey.RPID != "example.com" {
		t.Error("RPID mismatch")
	}
	if len(resp.PublicKey.AllowCredentials) != 2 {
		t.Errorf("expected 2 allowed creds, got %d", len(resp.PublicKey.AllowCredentials))
	}
}

func TestConditionalUI_DefaultUserVerification(t *testing.T) {
	resp := BeginConditionalUI(&ConditionalUIRequest{}, nil)
	if resp.PublicKey.UserVerification != "preferred" {
		t.Errorf("expected 'preferred', got '%s'", resp.PublicKey.UserVerification)
	}
}

func TestConditionalUI_NoCredentials(t *testing.T) {
	resp := BeginConditionalUI(&ConditionalUIRequest{}, nil)
	if len(resp.PublicKey.AllowCredentials) != 0 {
		t.Error("should have empty allowCredentials when no creds")
	}
}

func TestConditionalUI_HTTPHandler(t *testing.T) {
	creds := map[string][][]byte{
		"user-1": {[]byte("cred-a")},
	}
	handler := HandleConditionalUIBegin(creds)

	req := httptest.NewRequest(http.MethodGet, "/webauthn/conditional-ui/begin?user_id=user-1&rp_id=example.com&challenge=abc", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp ConditionalUIResponse
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp.Mediation != "conditional" {
		t.Error("response should have conditional mediation")
	}
}

func TestConditionalUI_HTTPMissingUserID(t *testing.T) {
	handler := HandleConditionalUIBegin(map[string][][]byte{})

	req := httptest.NewRequest(http.MethodGet, "/webauthn/conditional-ui/begin", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestConditionalUI_HTTPMethodNotAllowed(t *testing.T) {
	handler := HandleConditionalUIBegin(map[string][][]byte{})

	req := httptest.NewRequest(http.MethodPost, "/webauthn/conditional-ui/begin?user_id=u", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

func TestConditionalUI_BrowserSupport(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Sec-WebAuthn-Conditional-Mediation", "true")
	if !IsConditionalMediationSupported(req) {
		t.Error("should detect conditional mediation support")
	}

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	if IsConditionalMediationSupported(req2) {
		t.Error("should detect lack of support")
	}
}
