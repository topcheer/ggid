package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPHandlerHealthEndpoints(t *testing.T) {
	h := NewHTTPHandler(nil)

	t.Run("healthz", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}
		if body := rec.Body.String(); body == "" {
			t.Fatal("expected non-empty healthz body")
		}
	})

	t.Run("readyz", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}
		if body := rec.Body.String(); body == "" {
			t.Fatal("expected non-empty readyz body")
		}
	})
}

func TestHTTPHandlerTenantValidation(t *testing.T) {
	h := NewHTTPHandler(nil)

	cases := []struct {
		name   string
		method string
		path   string
		want   int
	}{
		{"users missing tenant", http.MethodGet, "/api/v1/users", http.StatusBadRequest},
		{"user by id missing tenant", http.MethodGet, "/api/v1/users/00000000-0000-0000-0000-000000000001", http.StatusBadRequest},
		{"import csv invalid method", http.MethodGet, "/api/v1/users/import", http.StatusMethodNotAllowed},
		{"export invalid method", http.MethodPost, "/api/v1/users/export", http.StatusMethodNotAllowed},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != tc.want {
				t.Fatalf("expected %d, got %d", tc.want, rec.Code)
			}
		})
	}
}
