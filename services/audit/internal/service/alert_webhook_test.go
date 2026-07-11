package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAlertWebhook_Post(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sig := r.Header.Get("X-GGID-Signature")
		if sig == "" {
			t.Error("should have signature header")
		}
		if !strings.HasPrefix(sig, "sha256=") {
			t.Error("signature should be sha256= prefixed")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	w := NewAlertWebhook(srv.URL, "test-secret")
	err := w.Post(context.Background(), []byte(`{"alert":"test"}`))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if w.GetSent() != 1 {
		t.Error("should have sent 1 webhook")
	}
}

func TestAlertWebhook_NoEndpoint(t *testing.T) {
	w := NewAlertWebhook("", "secret")
	err := w.Post(context.Background(), []byte("{}"))
	if err == nil {
		t.Error("should error with empty endpoint")
	}
}

func TestAlertWebhook_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	w := NewAlertWebhook(srv.URL, "secret")
	err := w.Post(context.Background(), []byte("{}"))
	if err == nil {
		t.Error("should error on 500")
	}
}

func TestAlertWebhook_SignConsistent(t *testing.T) {
	w := NewAlertWebhook("http://x", "secret")
	payload := []byte(`{"a":1}`)
	sig1 := w.Sign(payload)
	sig2 := w.Sign(payload)
	if sig1 != sig2 {
		t.Error("same payload should produce same signature")
	}
}
