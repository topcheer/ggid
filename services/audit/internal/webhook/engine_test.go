package webhook

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateAndListEndpoints(t *testing.T) {
	engine := NewEngine(nil)
	ep := engine.CreateEndpoint(&Endpoint{
		URL: "https://hooks.example.com/test",
		Events: []string{"user.created", "risk.high"},
		Secret: "whsec_test123",
		Enabled: true,
	})

	if ep.ID == "" {
		t.Fatal("expected non-empty ID")
	}
	if ep.MaxRetries != 5 {
		t.Fatalf("expected default max_retries 5, got %d", ep.MaxRetries)
	}

	eps := engine.ListEndpoints()
	if len(eps) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(eps))
	}
}

func TestIsSubscribed(t *testing.T) {
	engine := NewEngine(nil)
	ep := &Endpoint{
		Events:  []string{"user.created", "user.*", "itdr.detection"},
		Enabled: true,
	}

	if !engine.IsSubscribed(ep, "user.created") {
		t.Error("should subscribe to user.created")
	}
	if !engine.IsSubscribed(ep, "user.deleted") {
		t.Error("should subscribe via wildcard user.*")
	}
	if !engine.IsSubscribed(ep, "itdr.detection") {
		t.Error("should subscribe to itdr.detection")
	}
	if engine.IsSubscribed(ep, "system.health") {
		t.Error("should not subscribe to system.health")
	}

	// Disabled endpoint should not subscribe.
	ep.Enabled = false
	if engine.IsSubscribed(ep, "user.created") {
		t.Error("disabled endpoint should not subscribe")
	}
}

func TestSend_DeliverySuccess(t *testing.T) {
	received := false
	var receivedSig string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = true
		receivedSig = r.Header.Get("X-GGID-Signature")
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	engine := NewEngine(nil)
	engine.CreateEndpoint(&Endpoint{
		URL: ts.URL, Events: []string{"test.event"}, Secret: "secret123", Enabled: true,
	})

	deliveries := engine.Send(context.Background(), "test.event", map[string]any{"key": "value"})

	if len(deliveries) != 1 {
		t.Fatalf("expected 1 delivery, got %d", len(deliveries))
	}
	if !received {
		t.Fatal("webhook was not received by test server")
	}
	if deliveries[0].Status != "delivered" {
		t.Fatalf("expected delivered, got %s", deliveries[0].Status)
	}
	if receivedSig == "" {
		t.Fatal("expected X-GGID-Signature header")
	}
}

func TestSend_DeliveryFailure_DeadLetter(t *testing.T) {
	engine := NewEngine(nil)
	engine.CreateEndpoint(&Endpoint{
		URL: "http://localhost:59999/nonexistent", Events: []string{"test.fail"},
		Enabled: true, MaxRetries: 2,
	})

	deliveries := engine.Send(context.Background(), "test.fail", map[string]any{"k": "v"})

	if len(deliveries) != 1 {
		t.Fatalf("expected 1 delivery, got %d", len(deliveries))
	}
	d := deliveries[0]
	if d.Status != "dead_letter" {
		t.Fatalf("expected dead_letter after max retries, got %s", d.Status)
	}
	if d.Attempts != 3 {
		t.Fatalf("expected 3 attempts (1+2 retries), got %d", d.Attempts)
	}
}

func TestVerifySignature(t *testing.T) {
	payload := []byte(`{"event":"test"}`)
	secret := "mysecret"

	// Generate signature.
	mac := hmacSHA256(payload, secret)
	correct := hexEncode(mac)

	if !VerifySignature(payload, secret, correct) {
		t.Fatal("valid signature should verify")
	}
	if VerifySignature(payload, secret, "wrong-signature") {
		t.Fatal("invalid signature should not verify")
	}
	if VerifySignature(payload, "wrong-secret", correct) {
		t.Fatal("wrong secret should not verify")
	}
}

func TestSend_NoSubscribers(t *testing.T) {
	engine := NewEngine(nil)
	engine.CreateEndpoint(&Endpoint{
		URL: "https://example.com", Events: []string{"other.event"}, Enabled: true,
	})

	deliveries := engine.Send(context.Background(), "unrelated.event", map[string]any{})
	if len(deliveries) != 0 {
		t.Fatalf("expected 0 deliveries for non-matching event, got %d", len(deliveries))
	}
}

func TestDeleteEndpoint(t *testing.T) {
	engine := NewEngine(nil)
	ep := engine.CreateEndpoint(&Endpoint{
		URL: "https://example.com", Events: []string{"*"}, Enabled: true,
	})

	if engine.GetEndpoint(ep.ID) == nil {
		t.Fatal("endpoint should exist")
	}
	engine.DeleteEndpoint(ep.ID)
	if engine.GetEndpoint(ep.ID) != nil {
		t.Fatal("endpoint should be deleted")
	}
}

func TestEnsureSchema_NilPool(t *testing.T) {
	engine := NewEngine(nil)
	if err := engine.EnsureSchema(context.Background()); err != nil {
		t.Fatalf("nil pool should not error: %v", err)
	}
}

// Helper functions for test.
func hmacSHA256(data []byte, secret string) []byte {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(data)
	return mac.Sum(nil)
}

func hexEncode(b []byte) string {
	return hex.EncodeToString(b)
}
