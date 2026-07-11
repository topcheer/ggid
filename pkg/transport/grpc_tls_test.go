package tlsconfig

import (
	"os"
	"testing"
)

func TestIsTLSEnabled_DefaultFalse(t *testing.T) {
	os.Unsetenv("GRPC_TLS_ENABLED")
	if IsTLSEnabled() {
		t.Error("expected IsTLSEnabled() to return false when GRPC_TLS_ENABLED is unset")
	}
}

func TestIsTLSEnabled_True(t *testing.T) {
	os.Setenv("GRPC_TLS_ENABLED", "true")
	defer os.Unsetenv("GRPC_TLS_ENABLED")
	if !IsTLSEnabled() {
		t.Error("expected IsTLSEnabled() to return true when GRPC_TLS_ENABLED=true")
	}
}

func TestNewGRPCClientDialer_Insecure(t *testing.T) {
	os.Unsetenv("GRPC_TLS_ENABLED")
	creds, err := NewGRPCClientDialer("")
	if err != nil {
		t.Fatalf("expected nil error for insecure mode, got %v", err)
	}
	if creds != nil {
		t.Error("expected nil credentials for insecure mode")
	}
}

func TestNewGRPCClientDialer_MissingCAFile(t *testing.T) {
	os.Setenv("GRPC_TLS_ENABLED", "true")
	defer os.Unsetenv("GRPC_TLS_ENABLED")
	creds, err := NewGRPCClientDialer("/nonexistent/ca.crt")
	if err == nil {
		t.Fatal("expected error for missing CA file")
	}
	if creds != nil {
		t.Error("expected nil credentials on error")
	}
}

func TestTLSServerConfig_MissingCert(t *testing.T) {
	_, err := TLSServerConfig("/nonexistent/cert.pem", "/nonexistent/key.pem")
	if err == nil {
		t.Fatal("expected error for missing cert/key files")
	}
}

func TestNewGRPCServer_Plaintext(t *testing.T) {
	os.Unsetenv("GRPC_TLS_ENABLED")
	os.Setenv("GRPC_LISTEN_ADDR", ":0")
	defer os.Unsetenv("GRPC_LISTEN_ADDR")

	srv, lis, err := NewGRPCServer("", "")
	if err != nil {
		t.Fatalf("expected no error creating plaintext server, got %v", err)
	}
	if srv == nil {
		t.Fatal("expected non-nil grpc.Server")
	}
	if lis == nil {
		t.Fatal("expected non-nil listener")
	}
	lis.Close()
	srv.Stop()
}
