package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/users" {
			t.Errorf("expected /api/v1/users, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("expected Bearer test-token, got %s", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"users": []any{}})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	var result map[string]any
	if err := c.Get(context.Background(), "/api/v1/users", &result); err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if _, ok := result["users"]; !ok {
		t.Error("expected 'users' key in response")
	}
}

func TestClientPost(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["name"] != "test-user" {
			t.Errorf("expected name=test-user, got %v", body["name"])
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"id": "123"})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	var result map[string]any
	err := c.Post(context.Background(), "/api/v1/users", map[string]any{
		"name": "test-user",
	}, &result)
	if err != nil {
		t.Fatalf("Post failed: %v", err)
	}
	if result["id"] != "123" {
		t.Errorf("expected id=123, got %v", result["id"])
	}
}

func TestClientPut(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"updated": true})
	}))
	defer srv.Close()

	c := New(srv.URL, "")
	var result map[string]any
	err := c.Put(context.Background(), "/api/v1/users/123", map[string]any{"name": "updated"}, &result)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}
	if result["updated"] != true {
		t.Errorf("expected updated=true, got %v", result["updated"])
	}
}

func TestClientDelete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.Delete(context.Background(), "/api/v1/users/123", nil)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
}

func TestClientErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("not found"))
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.Get(context.Background(), "/api/v1/nonexistent", nil)
	if err == nil {
		t.Fatal("expected error for 404")
	}
}

func TestClientNoToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); auth != "" {
			t.Errorf("expected no Authorization header, got %s", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer srv.Close()

	c := New(srv.URL, "")
	var result map[string]any
	if err := c.Get(context.Background(), "/test", &result); err != nil {
		t.Fatalf("Get failed: %v", err)
	}
}

func TestClientContextCanceled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Slow response
		select {}
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	c := New(srv.URL, "test-token")
	err := c.Get(ctx, "/api/v1/test", nil)
	if err == nil {
		t.Fatal("expected error for canceled context")
	}
}
