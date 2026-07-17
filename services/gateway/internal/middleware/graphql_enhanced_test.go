package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAnalyzeComplexity_Simple(t *testing.T) {
	query := `{ users { id username } }`
	depth, complexity, err := AnalyzeComplexity(query)
	if err != nil { t.Fatalf("unexpected error: %v", err) }
	if depth != 2 { t.Errorf("depth=%d want 2", depth) }
	if complexity < 10 { t.Errorf("complexity=%d should include users cost", complexity) }
}

func TestAnalyzeComplexity_Deep(t *testing.T) {
	query := `{ users { groups { members { roles { permissions { key } } } } } }`
	depth, _, err := AnalyzeComplexity(query)
	if err != nil { t.Fatalf("unexpected error: %v", err) }
	if depth != 6 { t.Errorf("depth=%d want 6", depth) }
}

func TestValidateQuery_TooDeep(t *testing.T) {
	// Build a query deeper than MaxQueryDepth (10).
	query := `{ users { groups { members { roles { permissions { a { b { c { d { e { f { g { h { i { j } } } } } } } } } } } } } } }`
	err := ValidateQuery(query)
	if err == nil { t.Error("should reject deep query") }
	if !strings.Contains(err.Error(), "depth") { t.Errorf("error should mention depth: %v", err) }
}

func TestValidateQuery_Empty(t *testing.T) {
	err := ValidateQuery("")
	if err == nil { t.Error("should reject empty query") }
}

func TestPersistedQuery_RegisterLookup(t *testing.T) {
	query := `{ users { id } }`
	hash := RegisterPersistedQuery(query)
	if hash == "" { t.Fatal("hash should not be empty") }
	found, ok := LookupPersistedQuery(hash)
	if !ok { t.Fatal("should find registered query") }
	if found != query { t.Error("query mismatch") }
}

func TestPersistedQuery_NotFound(t *testing.T) {
	_, ok := LookupPersistedQuery("nonexistent-hash")
	if ok { t.Error("should not find unregistered hash") }
}

func TestGraphQLMiddleware_ComplexityReject(t *testing.T) {
	// Build an expensive query.
	query := strings.Repeat(" users", 200) // many fields
	body := `{"query":"{ ` + query + ` { id } }"}`

	req := httptest.NewRequest("POST", "/graphql", io.NopCloser(strings.NewReader(body)))
	w := httptest.NewRecorder()

	called := false
	GraphQLMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})).ServeHTTP(w, req)

	if called { t.Error("handler should not be called for complex query") }
	if w.Code != http.StatusOK { t.Errorf("expected 200 with error, got %d", w.Code) }
}

func TestGraphQLMiddleware_ValidQuery(t *testing.T) {
	body := `{"query":"{ users { id username } }"}`
	req := httptest.NewRequest("POST", "/graphql", io.NopCloser(strings.NewReader(body)))
	w := httptest.NewRecorder()

	called := false
	GraphQLMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})).ServeHTTP(w, req)

	if !called { t.Error("handler should be called for valid query") }
}

func TestSetPersistedOnlyMode(t *testing.T) {
	SetPersistedOnlyMode(true)
	if !IsPersistedOnly() { t.Error("should be in persisted-only mode") }
	SetPersistedOnlyMode(false)
	if IsPersistedOnly() { t.Error("should not be in persisted-only mode") }
}
