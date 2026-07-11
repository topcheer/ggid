// Package tlsconfig provides helpers for loading TLS credentials
// for gRPC connections between GGID microservices.
package tlsconfig

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

// LoadServerTLS loads a TLS key pair for a gRPC server.
// certFile and keyFile are PEM-encoded.
// Returns transport credentials suitable for grpc.NewServer.
//
// Usage:
//
//	creds, err := tlsconfig.LoadServerTLS("cert.pem", "key.pem")
//	if err != nil { log.Fatal(err) }
//	srv := grpc.NewServer(grpc.Creds(creds))
func LoadServerTLS(certFile, keyFile string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("load server key pair: %w", err)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
	}, nil
}

// LoadClientTLS loads CA certificate for a gRPC client to verify
// the server's certificate. caFile is PEM-encoded.
// Returns transport credentials suitable for grpc.Dial.
//
// Usage:
//
//	creds, err := tlsconfig.LoadClientTLS("ca.pem")
//	if err != nil { log.Fatal(err) }
//	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(creds))
func LoadClientTLS(caFile string) (*tls.Config, error) {
	caBytes, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("read CA file: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caBytes) {
		return nil, fmt.Errorf("failed to parse CA certificate from %s", caFile)
	}
	return &tls.Config{
		RootCAs:    pool,
		MinVersion: tls.VersionTLS13,
	}, nil
}

// LoadMutualTLS loads both client identity and CA for mTLS.
// The client presents its own certificate and verifies the server.
//
// Usage:
//
//	creds, err := tlsconfig.LoadMutualTLS("client-cert.pem", "client-key.pem", "ca.pem")
//	if err != nil { log.Fatal(err) }
//	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(credentials.NewTLS(creds)))
func LoadMutualTLS(certFile, keyFile, caFile string) (*tls.Config, error) {
	creds, err := LoadServerTLS(certFile, keyFile)
	if err != nil {
		return nil, err
	}
	caBytes, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("read CA file: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caBytes) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}
	creds.RootCAs = pool
	creds.ClientCAs = pool
	creds.ClientAuth = tls.RequireAndVerifyClientCert
	return creds, nil
}
