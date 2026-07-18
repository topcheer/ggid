package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDetectMaliciousInput_SQLi(t *testing.T) {
	tests := []string{
		"' OR 1=1--",
		"' OR '1'='1",
		"1 UNION SELECT * FROM users",
		"; DROP TABLE users",
		"'; DELETE FROM accounts WHERE 1=1",
	}
	for _, input := range tests {
		typ, _ := DetectMaliciousInput(input)
		if typ != "sql_injection" {
			t.Errorf("should detect SQLi in: %s", input)
		}
	}
}

func TestDetectMaliciousInput_XSS(t *testing.T) {
	tests := []string{
		"<script>alert(1)</script>",
		"javascript:alert(1)",
		"<img onerror=alert(1)>",
		"<iframe src=evil.com>",
	}
	for _, input := range tests {
		typ, _ := DetectMaliciousInput(input)
		if typ != "xss" {
			t.Errorf("should detect XSS in: %s", input)
		}
	}
}

func TestDetectMaliciousInput_Clean(t *testing.T) {
	clean := []string{
		"normal user input",
		"email@example.com",
		"Password123!",
		"O'Brien",
		"SELECTED ITEMS", // not SQL SELECT
	}
	for _, input := range clean {
		typ, _ := DetectMaliciousInput(input)
		if typ != "" {
			t.Errorf("clean input flagged as %s: %s", typ, input)
		}
	}
}

func TestInputValidationMiddleware_RejectsSQLi(t *testing.T) {
	body := `{"username":"' OR 1=1--","password":"test"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/login", io.NopCloser(strings.NewReader(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	called := false
	InputValidationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})).ServeHTTP(w, req)

	if called {
		t.Error("should not call next handler on SQLi")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestInputValidationMiddleware_AllowsClean(t *testing.T) {
	body := `{"username":"testuser","password":"Password123"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/login", io.NopCloser(strings.NewReader(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	called := false
	InputValidationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(200)
	})).ServeHTTP(w, req)

	if !called {
		t.Error("should call next handler for clean input")
	}
}

func TestInputValidationMiddleware_ExemptPath(t *testing.T) {
	body := `{"data":"<script>alert(1)</script>"}`
	req := httptest.NewRequest("POST", "/api/v1/dlp/scan", io.NopCloser(strings.NewReader(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	called := false
	InputValidationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})).ServeHTTP(w, req)

	if !called {
		t.Error("exempt path should bypass validation")
	}
}
