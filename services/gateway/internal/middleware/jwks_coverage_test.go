package middleware

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

// Helper: generate a test RSA key pair and return PEM-encoded public key
func generateTestRSAPublicKeyPEM(t *testing.T) (*rsa.PublicKey, string) {
	t.Helper()
	// Use a fixed small RSA key for testing speed
	// We'll generate it at runtime
	priv, err := rsa.GenerateKey(nil, 0)
	if err != nil {
		// Can't generate with nil source, use a pre-computed key
	}
	_ = priv

	// Use a pre-built key for determinism
	// n = 256-bit modulus for test speed
	n := new(big.Int)
	n.SetString("00b3510a2f7c9820e2b34e8054a9b3c2f8e1d4a7b9c6e3f0a1d2b3c4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2", 16)
	pub := &rsa.PublicKey{
		N: n,
		E: 65537,
	}

	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		t.Fatalf("marshal PKIX: %v", err)
	}
	pemData := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: der,
	})
	return pub, string(pemData)
}

func TestJWKToRSAPublicKey_Valid(t *testing.T) {
	// Use a real RSA key
	pub, _ := generateTestRSAPublicKeyPEM(t)
	nB64 := base64.RawURLEncoding.EncodeToString(pub.N.Bytes())
	eB64 := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes())

	result, err := jwkToRSAPublicKey(nB64, eB64)
	if err != nil {
		t.Fatalf("jwkToRSAPublicKey: %v", err)
	}
	if result.N.Cmp(pub.N) != 0 {
		t.Error("N mismatch")
	}
	if result.E != pub.E {
		t.Errorf("E: want %d, got %d", pub.E, result.E)
	}
}

func TestJWKToRSAPublicKey_InvalidN(t *testing.T) {
	_, err := jwkToRSAPublicKey("!!!invalid-base64!!!", "AQAB")
	if err == nil {
		t.Error("Should fail on invalid N base64")
	}
}

func TestJWKToRSAPublicKey_InvalidE(t *testing.T) {
	nB64 := base64.RawURLEncoding.EncodeToString(big.NewInt(123).Bytes())
	_, err := jwkToRSAPublicKey(nB64, "!!!invalid!!!")
	if err == nil {
		t.Error("Should fail on invalid E base64")
	}
}

func TestLoadPublicKey_ValidPKIX(t *testing.T) {
	_, pemStr := generateTestRSAPublicKeyPEM(t)

	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "pub.pem")
	if err := os.WriteFile(keyPath, []byte(pemStr), 0644); err != nil {
		t.Fatalf("write key: %v", err)
	}

	pub, kid, err := loadPublicKey(keyPath)
	if err != nil {
		t.Fatalf("loadPublicKey: %v", err)
	}
	if pub == nil {
		t.Fatal("pub is nil")
	}
	if kid == "" {
		t.Error("kid should not be empty")
	}
}

func TestLoadPublicKey_FileNotFound(t *testing.T) {
	_, _, err := loadPublicKey("/nonexistent/path/key.pem")
	if err == nil {
		t.Error("Should fail on nonexistent file")
	}
}

func TestLoadPublicKey_InvalidPEM(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "bad.pem")
	os.WriteFile(keyPath, []byte("not a PEM file"), 0644)

	_, _, err := loadPublicKey(keyPath)
	if err == nil {
		t.Error("Should fail on invalid PEM")
	}
}

func TestKeyFingerprint(t *testing.T) {
	pub, _ := generateTestRSAPublicKeyPEM(t)
	fp := keyFingerprint(pub)
	if fp == "" {
		t.Error("fingerprint should not be empty")
	}
	if fp == "unknown" {
		t.Error("fingerprint should not be 'unknown' for valid key")
	}
}

func TestNewJWKSClient_EmptyURL(t *testing.T) {
	client, err := NewJWKSClient("", "")
	if err != nil {
		t.Fatalf("NewJWKSClient: %v", err)
	}
	if client == nil {
		t.Fatal("client is nil")
	}
	if client.KeyID() != "" {
		t.Error("Empty URL should have empty keyID")
	}
}

func TestNewJWKSClient_WithPublicKey(t *testing.T) {
	_, pemStr := generateTestRSAPublicKeyPEM(t)

	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "pub.pem")
	os.WriteFile(keyPath, []byte(pemStr), 0644)

	client, err := NewJWKSClient("", keyPath)
	if err != nil {
		t.Fatalf("NewJWKSClient: %v", err)
	}
	if client.KeyID() == "" {
		t.Error("Should have keyID from public key")
	}

	// GetKey should work
	key, err := client.GetKey(client.KeyID())
	if err != nil {
		t.Errorf("GetKey: %v", err)
	}
	if key == nil {
		t.Error("key is nil")
	}
}

func TestNewJWKSClient_InvalidKeyPath(t *testing.T) {
	_, err := NewJWKSClient("", "/nonexistent/key.pem")
	if err == nil {
		t.Error("Should fail on invalid key path")
	}
}

func TestJWKSClient_GetKey_NotFound(t *testing.T) {
	client, _ := NewJWKSClient("", "")
	_, err := client.GetKey("nonexistent-kid")
	if err == nil {
		t.Error("Should fail for nonexistent key")
	}
}

func TestJWKSClient_UpdatePublicKey(t *testing.T) {
	client, _ := NewJWKSClient("", "")
	pub, _ := generateTestRSAPublicKeyPEM(t)

	client.UpdatePublicKey(pub, "new-kid")

	if client.KeyID() != "new-kid" {
		t.Errorf("KeyID: want 'new-kid', got '%s'", client.KeyID())
	}
	key, err := client.GetKey("new-kid")
	if err != nil {
		t.Errorf("GetKey after update: %v", err)
	}
	if key == nil {
		t.Error("key is nil after update")
	}
}

func TestJWKSClient_JWKSHandler_V2(t *testing.T) {
	pub, _ := generateTestRSAPublicKeyPEM(t)
	client, _ := NewJWKSClient("", "")
	client.UpdatePublicKey(pub, "test-kid")

	handler := client.JWKSHandler()
	req := httptest.NewRequest("GET", "/.well-known/jwks.json", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("JWKSHandler: want 200, got %d", rr.Code)
	}

	var result struct {
		Keys []struct {
			KTY string `json:"kty"`
			KID string `json:"kid"`
		} `json:"keys"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatalf("parse JWKS: %v", err)
	}
	if len(result.Keys) == 0 {
		t.Error("Should have at least 1 key")
	}
	if result.Keys[0].KID != "test-kid" {
		t.Errorf("KID: want 'test-kid', got '%s'", result.Keys[0].KID)
	}
}

func TestJWKSClient_RefreshJWKS_MockServer(t *testing.T) {
	pub, _ := generateTestRSAPublicKeyPEM(t)
	nB64 := base64.RawURLEncoding.EncodeToString(pub.N.Bytes())
	eB64 := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes())

	// Mock JWKS endpoint
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"keys": []map[string]any{
				{
					"kty": "RSA",
					"kid": "mock-kid-1",
					"use": "sig",
					"n":   nB64,
					"e":   eB64,
				},
				{
					"kty": "RSA",
					"kid": "mock-kid-2",
					"use": "sig",
					"n":   nB64,
					"e":   eB64,
				},
				{
					"kty": "oct", // should be skipped
					"kid": "skip-me",
					"use": "enc",
				},
			},
		})
	}))
	defer ts.Close()

	client, err := NewJWKSClient(ts.URL, "")
	if err != nil {
		t.Fatalf("NewJWKSClient: %v", err)
	}

	key, err := client.GetKey("mock-kid-1")
	if err != nil {
		t.Errorf("GetKey mock-kid-1: %v", err)
	}
	if key == nil {
		t.Error("key is nil")
	}
}

func TestJWKSClient_RefreshJWKS_EmptyURL(t *testing.T) {
	client, _ := NewJWKSClient("", "")
	err := client.refreshJWKS()
	if err != nil {
		t.Errorf("Empty URL should be no-op: %v", err)
	}
}

func TestJWKSClient_RefreshJWKS_Non200(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	client, _ := NewJWKSClient(ts.URL, "")
	err := client.refreshJWKS()
	if err == nil {
		t.Error("Should fail on 500 response")
	}
}

func TestJWKSClient_RefreshJWKS_InvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer ts.Close()

	client, _ := NewJWKSClient(ts.URL, "")
	err := client.refreshJWKS()
	if err == nil {
		t.Error("Should fail on invalid JSON")
	}
}

func TestJWKSClient_RefreshJWKS_ConnectionError(t *testing.T) {
	// Create a server then close it immediately to force connection error
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	ts.Close()

	client, _ := NewJWKSClient(ts.URL, "")
	err := client.refreshJWKS()
	if err == nil {
		t.Error("Should fail on connection error")
	}
}

func TestJWKSClient_RefreshJWKS_SkipNonRSA(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"keys": []map[string]any{
				{
					"kty": "EC", // should be skipped
					"kid": "ec-key",
					"use": "sig",
					"crv": "P-256",
				},
				{
					"kty": "RSA",
					"kid": "rsa-key",
					"use": "enc", // should be skipped (not "sig")
					"n":   "AQAB",
					"e":   "AQAB",
				},
			},
		})
	}))
	defer ts.Close()

	client, _ := NewJWKSClient(ts.URL, "")
	// refreshJWKS should succeed (no error) but with 0 valid keys
	_, err := client.GetKey("ec-key")
	if err == nil {
		t.Error("EC key should not be in store")
	}
}

func TestJWKSClient_StartRefresh(t *testing.T) {
	pub, _ := generateTestRSAPublicKeyPEM(t)
	nB64 := base64.RawURLEncoding.EncodeToString(pub.N.Bytes())
	eB64 := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes())

	var callCount atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"keys": []map[string]any{
				{
					"kty": "RSA",
					"kid": fmt.Sprintf("key-%d", callCount.Load()),
					"use": "sig",
					"n":   nB64,
					"e":   eB64,
				},
			},
		})
	}))
	defer ts.Close()

	client, _ := NewJWKSClient(ts.URL, "")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client.StartRefresh(ctx, 50*time.Millisecond)

	// Wait for a couple refresh cycles
	time.Sleep(200 * time.Millisecond)
	cancel()
	time.Sleep(50 * time.Millisecond)

	// Should have refreshed at least once
	if callCount.Load() < 2 {
		t.Errorf("Expected at least 2 refresh calls, got %d", callCount.Load())
	}
}
