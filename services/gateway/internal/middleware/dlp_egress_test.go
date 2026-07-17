package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRedactSSN(t *testing.T) {
	cfg := DefaultDLPEgressConfig()
	input := []byte(`{"name":"John","ssn":"123-45-6789"}`)
	redacted, matches, err := ScanResponseBody(input, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	if matches[0].Type != "ssn" {
		t.Fatalf("expected type ssn, got %s", matches[0].Type)
	}
	var obj map[string]any
	json.Unmarshal(redacted, &obj)
	ssn, _ := obj["ssn"].(string)
	if ssn == "123-45-6789" {
		t.Fatal("SSN was not redacted")
	}
}

func TestRedactCreditCard_LuhnValidation(t *testing.T) {
	cfg := DefaultDLPEgressConfig()
	// Valid Visa test number.
	input := []byte(`{"card":"4111111111111111"}`)
	_, matches, _ := ScanResponseBody(input, cfg)
	found := false
	for _, m := range matches {
		if m.Type == "credit_card" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected credit_card match for valid Luhn number")
	}

	// Invalid number should NOT be flagged.
	input2 := []byte(`{"card":"1234567890123456"}`)
	_, matches2, _ := ScanResponseBody(input2, cfg)
	for _, m := range matches2 {
		if m.Type == "credit_card" {
			t.Fatal("invalid credit card should not match (Luhn fails)")
		}
	}
}

func TestRedactEmail_MaskStrategy(t *testing.T) {
	cfg := DefaultDLPEgressConfig()
	input := []byte(`{"email":"john.doe@example.com"}`)
	redacted, matches, _ := ScanResponseBody(input, cfg)
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	var obj map[string]any
	json.Unmarshal(redacted, &obj)
	masked, _ := obj["email"].(string)
	if masked == "john.doe@example.com" {
		t.Fatal("email was not masked")
	}
	if !containsStr(masked, "@example.com") {
		t.Fatalf("email mask should preserve domain, got %s", masked)
	}
}

func TestNestedObjectRedaction(t *testing.T) {
	cfg := DefaultDLPEgressConfig()
	input := []byte(`{"user":{"name":"Jane","email":"jane@test.com"},"items":[{"ssn":"987-65-4321"}]}`)
	redacted, matches, _ := ScanResponseBody(input, cfg)
	if len(matches) < 2 {
		t.Fatalf("expected at least 2 matches, got %d", len(matches))
	}
	var obj map[string]any
	json.Unmarshal(redacted, &obj)
	// Verify nested email is redacted.
	user, _ := obj["user"].(map[string]any)
	email, _ := user["email"].(string)
	if email == "jane@test.com" {
		t.Fatal("nested email was not redacted")
	}
}

func TestPasswordFullMask(t *testing.T) {
	cfg := DefaultDLPEgressConfig()
	input := []byte(`{"password":"supersecret123","token":"sk_live_abcdef1234567890"}`)
	redacted, _, _ := ScanResponseBody(input, cfg)
	var obj map[string]any
	json.Unmarshal(redacted, &obj)
	pw, _ := obj["password"].(string)
	if pw != "**************" {
		t.Fatalf("expected full mask for password, got %s", pw)
	}
}

func TestDLPEgressMiddleware(t *testing.T) {
	cfg := DefaultDLPEgressConfig()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ssn":"111-22-3333","name":"test"}`))
	})

	var auditMatches []PIIMatch
	wrapped := DLPEgressMiddleware(cfg, func(m PIIMatch) {
		auditMatches = append(auditMatches, m)
	})(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if containsStr(body, "111-22-3333") {
		t.Fatal("SSN should be redacted in response")
	}
	if len(auditMatches) == 0 {
		t.Fatal("expected audit matches to be recorded")
	}
}

func TestNonJSONResponsePassthrough(t *testing.T) {
	cfg := DefaultDLPEgressConfig()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("hello 123-45-6789"))
	})

	wrapped := DLPEgressMiddleware(cfg, nil)(handler)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if !containsStr(rec.Body.String(), "123-45-6789") {
		t.Fatal("plain text should not be redacted")
	}
}

func containsStr(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}
