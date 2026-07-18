package middleware

import (
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
	// Reset config to ensure no exemptions from other tests.
	SetInputValidationConfig(InputValidationConfig{
		Enabled: true, ExemptPaths: map[string]bool{}, MaxBodySize: 10 * 1024 * 1024,
	})
	body := `{"username":"' OR 1=1--","password":"test"}`
	// Use strings.NewReader directly so httptest.NewRequest can detect ContentLength.
	// io.NopCloser wrapping hides the length, causing ContentLength=-1 which skips validation.
	req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(body))
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
	req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(body))
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
	// Set config with the exempt path we're testing.
	SetInputValidationConfig(InputValidationConfig{
		Enabled: true,
		ExemptPaths: map[string]bool{
			"/api/v1/dlp/scan": true,
		},
		MaxBodySize: 10 * 1024 * 1024,
	})
	t.Cleanup(func() {
		SetInputValidationConfig(InputValidationConfig{
			Enabled: true,
			ExemptPaths: map[string]bool{
				"/api/v1/audit/pii-scan": true,
				"/api/v1/crypto/fields":  true,
				"/api/v1/dlp/scan":       true,
				"/graphql":               true,
			},
			MaxBodySize: 10 * 1024 * 1024,
		})
	})
	body := `{"data":"<script>alert(1)</script>"}`
	req := httptest.NewRequest("POST", "/api/v1/dlp/scan", strings.NewReader(body))
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
