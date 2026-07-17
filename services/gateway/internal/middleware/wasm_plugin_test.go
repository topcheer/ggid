package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"testing"
)

func TestDefaultWasmResourceLimits(t *testing.T) {
	limits := DefaultWasmResourceLimits()
	if limits.MaxMemoryBytes != 16*1024*1024 {
		t.Errorf("expected 16MB, got %d", limits.MaxMemoryBytes)
	}
	if limits.ExecutionFuel != 10000 {
		t.Errorf("expected fuel 10000, got %d", limits.ExecutionFuel)
	}
	if limits.Timeout.Milliseconds() != 100 {
		t.Errorf("expected 100ms timeout, got %v", limits.Timeout)
	}
	if !limits.VerifySignature {
		t.Error("signature verification should be enabled by default")
	}
}

func TestVerifyPluginSignature_NoSecret(t *testing.T) {
	os.Unsetenv("GGID_INTERNAL_SECRET")
	host := NewWasmPluginHost()
	err := host.verifyPluginSignature([]byte("wasm"), "", "/tmp/test.wasm")
	if err != nil {
		t.Errorf("should skip verification when no secret set: %v", err)
	}
}

func TestVerifyPluginSignature_Valid(t *testing.T) {
	secret := "test-secret-123"
	os.Setenv("GGID_INTERNAL_SECRET", secret)
	defer os.Unsetenv("GGID_INTERNAL_SECRET")

	wasmBytes := []byte("fake wasm binary")
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(wasmBytes)
	sig := hex.EncodeToString(mac.Sum(nil))

	host := NewWasmPluginHost()
	if err := host.verifyPluginSignature(wasmBytes, sig, "/tmp/test.wasm"); err != nil {
		t.Errorf("valid signature should pass: %v", err)
	}
}

func TestVerifyPluginSignature_Invalid(t *testing.T) {
	secret := "test-secret-123"
	os.Setenv("GGID_INTERNAL_SECRET", secret)
	defer os.Unsetenv("GGID_INTERNAL_SECRET")

	host := NewWasmPluginHost()
	err := host.verifyPluginSignature([]byte("tampered wasm"), "badsignature", "/tmp/test.wasm")
	if err == nil {
		t.Error("invalid signature should fail")
	}
}

func TestVerifyPluginSignature_MissingSig(t *testing.T) {
	secret := "test-secret-123"
	os.Setenv("GGID_INTERNAL_SECRET", secret)
	defer os.Unsetenv("GGID_INTERNAL_SECRET")

	host := NewWasmPluginHost()
	err := host.verifyPluginSignature([]byte("wasm"), "", "/nonexistent/path.wasm")
	if err == nil {
		t.Error("missing signature should fail when secret is set")
	}
}

func TestNewWasmPluginHostWithLimits(t *testing.T) {
	limits := WasmResourceLimits{
		MaxMemoryBytes:  8 * 1024 * 1024, // 8MB = 128 pages
		ExecutionFuel:   5000,
		Timeout:         50 * 1000 * 1000, // 50ms
		VerifySignature: false,
	}
	host := NewWasmPluginHostWithLimits(limits)
	if host.limits.MaxMemoryBytes != 8*1024*1024 {
		t.Error("custom limits not applied")
	}
	host.Close(nil)
}
