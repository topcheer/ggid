package alerting

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestWebhookNotifier_HMACSignature(t *testing.T) {
	var receivedBody []byte
	var receivedSig string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		receivedSig = r.Header.Get("X-GGID-Signature")
		w.WriteHeader(200)
	}))
	defer srv.Close()

	secret := "test-secret-key"
	wn := &WebhookNotifier{URL: srv.URL, Secret: secret, client: srv.Client()}

	alert := &Alert{RuleID: "r1", RuleName: "test", TenantID: "t1", Trigger: "login.failed", Count: 5, FiredAt: time.Now()}
	if err := wn.Notify(context.Background(), alert, nil); err != nil {
		t.Fatalf("Notify: %v", err)
	}

	// Verify HMAC
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(receivedBody)
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	if receivedSig != expected {
		t.Errorf("signature mismatch: got %s, want %s", receivedSig, expected)
	}
	if len(receivedBody) == 0 {
		t.Error("body was empty")
	}
}

func TestWebhookNotifier_NoSecretNoSignature(t *testing.T) {
	var sig string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sig = r.Header.Get("X-GGID-Signature")
		w.WriteHeader(200)
	}))
	defer srv.Close()

	wn := &WebhookNotifier{URL: srv.URL, client: srv.Client()}
	alert := &Alert{RuleID: "r1", RuleName: "test"}
	if err := wn.Notify(context.Background(), alert, nil); err != nil {
		t.Fatalf("Notify: %v", err)
	}
	if sig != "" {
		t.Error("should not have signature without secret")
	}
}

func TestWebhookNotifier_ConnectionError(t *testing.T) {
	wn := &WebhookNotifier{URL: "http://127.0.0.1:1", client: &http.Client{Timeout: 1 * time.Second}}
	alert := &Alert{RuleID: "r1", RuleName: "test"}
	err := wn.Notify(context.Background(), alert, nil)
	if err == nil {
		t.Error("expected error for unreachable webhook, got nil")
	}
}

func TestWebhookNotifier_EmptyURL(t *testing.T) {
	wn := &WebhookNotifier{URL: ""}
	err := wn.Notify(context.Background(), &Alert{}, nil)
	if err != nil {
		t.Errorf("empty URL should be no-op, got: %v", err)
	}
}

// Silence unused import for bytes if not directly referenced
var _ = bytes.NewReader
