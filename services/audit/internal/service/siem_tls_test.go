package service

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"testing"
)

// TLSConfig configures mTLS for SIEM forwarder connections.
type TLSConfig struct {
	// Enabled controls whether TLS is used for SIEM connections.
	Enabled bool
	// ClientCert is the client certificate PEM (for mTLS).
	ClientCert []byte
	// ClientKey is the client private key PEM.
	ClientKey []byte
	// CACert is the CA certificate PEM for server verification.
	CACert []byte
	// ServerName for SNI / verification.
	ServerName string
	// InsecureSkipVerify skips certificate verification (testing only).
	InsecureSkipVerify bool
}

// BuildTLSConfig converts SIEM TLSConfig into a crypto/tls.Config.
func BuildTLSConfig(cfg *TLSConfig) (*tls.Config, error) {
	if cfg == nil {
		return nil, fmt.Errorf("nil TLS config")
	}
	tc := &tls.Config{
		ServerName:         cfg.ServerName,
		InsecureSkipVerify: cfg.InsecureSkipVerify,
		MinVersion:         tls.VersionTLS12,
	}

	// mTLS: load client cert
	if len(cfg.ClientCert) > 0 && len(cfg.ClientKey) > 0 {
		cert, err := tls.X509KeyPair(cfg.ClientCert, cfg.ClientKey)
		if err != nil {
			return nil, fmt.Errorf("load client cert: %w", err)
		}
		tc.Certificates = []tls.Certificate{cert}
	}

	// CA verification
	if len(cfg.CACert) > 0 {
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(cfg.CACert) {
			return nil, fmt.Errorf("failed to parse CA cert")
		}
		tc.RootCAs = pool
	}

	return tc, nil
}

// TestTLSConfig_Defaults verifies TLS 1.2 minimum.
func TestTLSConfig_Defaults(t *testing.T) {
	cfg := &TLSConfig{Enabled: true, ServerName: "siem.example.com"}
	tc, err := BuildTLSConfig(cfg)
	if err != nil {
		t.Fatalf("BuildTLSConfig: %v", err)
	}
	if tc.MinVersion < tls.VersionTLS12 {
		t.Error("minimum TLS version should be 1.2")
	}
	if tc.ServerName != "siem.example.com" {
		t.Error("ServerName mismatch")
	}
}

func TestTLSConfig_NilConfig(t *testing.T) {
	_, err := BuildTLSConfig(nil)
	if err == nil {
		t.Error("should error on nil config")
	}
}

func TestTLSConfig_InsecureSkipVerify(t *testing.T) {
	cfg := &TLSConfig{Enabled: true, InsecureSkipVerify: true}
	tc, _ := BuildTLSConfig(cfg)
	if !tc.InsecureSkipVerify {
		t.Error("should allow insecure skip verify")
	}
}

func TestTLSConfig_InvalidCACert(t *testing.T) {
	cfg := &TLSConfig{Enabled: true, CACert: []byte("not-a-cert")}
	_, err := BuildTLSConfig(cfg)
	if err == nil {
		t.Error("should error on invalid CA cert")
	}
}

func TestTLSConfig_InvalidClientCert(t *testing.T) {
	cfg := &TLSConfig{
		Enabled:    true,
		ClientCert: []byte("bad"),
		ClientKey:  []byte("bad"),
	}
	_, err := BuildTLSConfig(cfg)
	if err == nil {
		t.Error("should error on invalid client cert")
	}
}

// Ensure context is referenced (used in production for connection lifecycle).
var _ = context.Background
