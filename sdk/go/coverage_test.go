package ggid

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_Put(t *testing.T) {
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/test" {
			t.Errorf("expected /api/v1/test, got %s", r.URL.Path)
		}
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["name"] != "updated" {
			t.Errorf("expected name=updated, got %s", body["name"])
		}
		writeJSON(w, http.StatusOK, map[string]string{"name": "updated"})
	}))

	var result map[string]string
	err := c.put(context.Background(), "/api/v1/test", map[string]string{"name": "updated"}, &result)
	if err != nil {
		t.Fatalf("put failed: %v", err)
	}
	if result["name"] != "updated" {
		t.Errorf("expected name=updated, got %s", result["name"])
	}
}

func TestClient_Put_Error(t *testing.T) {
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad request"})
	}))

	err := c.put(context.Background(), "/api/v1/test", map[string]string{}, nil)
	if err == nil {
		t.Fatal("expected error from 400 response")
	}
}

func TestRequirePermission_NoUser(t *testing.T) {
	c, _ := newTestClient(t, nil)

	handler := c.RequirePermission("users", "read", func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called without user")
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestRequirePermission_Allowed(t *testing.T) {
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"allowed": true})
	}))

	called := false
	handler := c.RequirePermission("users", "read", func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req = req.WithContext(context.WithValue(req.Context(), ContextKeyUser, &UserInfo{UserID: "user-1"}))
	handler.ServeHTTP(rr, req)

	if !called {
		t.Error("handler should have been called")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestRequirePermission_Denied(t *testing.T) {
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"allowed": false})
	}))

	handler := c.RequirePermission("users", "read", func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called when denied")
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req = req.WithContext(context.WithValue(req.Context(), ContextKeyUser, &UserInfo{UserID: "user-1"}))
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}

func TestRequirePermission_ServerError(t *testing.T) {
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "server error"})
	}))

	handler := c.RequirePermission("users", "read", func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called on server error")
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req = req.WithContext(context.WithValue(req.Context(), ContextKeyUser, &UserInfo{UserID: "user-1"}))
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}
