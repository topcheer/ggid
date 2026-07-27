package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNormalizePath_NoIDs(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/api/v1/users", "/api/v1/users"},
		{"/api/v1/oauth/token", "/api/v1/oauth/token"},
		{"/", "/"},
		{"/healthz", "/healthz"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := normalizePath(tt.input); got != tt.want {
				t.Errorf("normalizePath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizePath_WithUUID(t *testing.T) {
	got := normalizePath("/api/v1/users/550e8400-e29b-41d4-a716-446655440000")
	want := "/api/v1/users/:id"
	if got != want {
		t.Errorf("normalizePath(UUID) = %q, want %q", got, want)
	}
}

func TestNormalizePath_MultipleUUIDs(t *testing.T) {
	got := normalizePath("/api/v1/orgs/550e8400-e29b-41d4-a716-446655440000/teams/660e8400-e29b-41d4-a716-446655440000")
	want := "/api/v1/orgs/:id/teams/:id"
	if got != want {
		t.Errorf("normalizePath(multi-UUID) = %q, want %q", got, want)
	}
}

func TestNormalizePath_NumericID(t *testing.T) {
	got := normalizePath("/api/v1/audit/12345/events")
	want := "/api/v1/audit/:id/events"
	if got != want {
		t.Errorf("normalizePath(numeric) = %q, want %q", got, want)
	}
}

func TestNormalizePath_ShortNumericNotID(t *testing.T) {
	// Numbers ≤3 chars are NOT IDs (e.g., v1, v2)
	got := normalizePath("/api/v1/users")
	want := "/api/v1/users" // "v1" and "1" should NOT be replaced
	if got != want {
		t.Errorf("normalizePath(short numeric) = %q, want %q", got, want)
	}
}

func TestLooksLikeID_UUID(t *testing.T) {
	if !looksLikeID("550e8400-e29b-41d4-a716-446655440000") {
		t.Error("expected UUID to look like ID")
	}
}

func TestLooksLikeID_LongNumeric(t *testing.T) {
	if !looksLikeID("12345") {
		t.Error("expected 5-digit number to look like ID")
	}
}

func TestLooksLikeID_ShortString(t *testing.T) {
	if looksLikeID("abc") {
		t.Error("expected short string to NOT look like ID")
	}
}

func TestLooksLikeID_ShortNumeric(t *testing.T) {
	if looksLikeID("123") {
		t.Error("expected 3-digit number to NOT look like ID")
	}
}

func TestLooksLikeID_PathSegment(t *testing.T) {
	if looksLikeID("users") {
		t.Error("expected 'users' to NOT look like ID")
	}
	if looksLikeID("v1") {
		t.Error("expected 'v1' to NOT look like ID")
	}
}

func TestSplitPath(t *testing.T) {
	parts := splitPath("/api/v1/users")
	if len(parts) != 3 || parts[0] != "api" || parts[1] != "v1" || parts[2] != "users" {
		t.Errorf("splitPath = %v, want [api v1 users]", parts)
	}
}

func TestSplitPath_Empty(t *testing.T) {
	parts := splitPath("/")
	if len(parts) != 0 {
		t.Errorf("splitPath('/') = %v, want empty", parts)
	}
}

func TestJoinPath(t *testing.T) {
	got := joinPath([]string{"api", "v1", "users"})
	if got != "/api/v1/users" {
		t.Errorf("joinPath = %q, want %q", got, "/api/v1/users")
	}
}

func TestJoinPath_Empty(t *testing.T) {
	got := joinPath([]string{})
	if got != "/" {
		t.Errorf("joinPath(empty) = %q, want %q", got, "/")
	}
}

func TestMiddleware_BasicRequest(t *testing.T) {
	handler := Middleware("test-service", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/users", nil))

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestMiddleware_SkipsMetricsEndpoint(t *testing.T) {
	called := false
	handler := Middleware("test-service", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/metrics", nil))

	if !called {
		t.Error("expected handler to be called even for /metrics")
	}
}

func TestMiddleware_Captures500(t *testing.T) {
	handler := Middleware("test-service", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/test", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

func TestStatusWriter_WriteHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	sw := &statusWriter{ResponseWriter: rec, status: 0}
	sw.WriteHeader(http.StatusTeapot)

	if sw.status != http.StatusTeapot {
		t.Errorf("expected status 418, got %d", sw.status)
	}
	if rec.Code != http.StatusTeapot {
		t.Errorf("expected recorded status 418, got %d", rec.Code)
	}
}
