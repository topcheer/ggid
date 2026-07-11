package tlsconfig

import (
	"crypto/tls"
	"log"
	"net"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// NewGRPCServer creates a gRPC server with optional TLS.
// When GRPC_TLS_ENABLED=true, loads server cert/key from the given paths.
// When false or unset, creates a plaintext server (for development).
func NewGRPCServer(certFile, keyFile string) (*grpc.Server, net.Listener, error) {
	tlsEnabled := os.Getenv("GRPC_TLS_ENABLED") == "true"

	addr := os.Getenv("GRPC_LISTEN_ADDR")
	if addr == "" {
		addr = ":50051"
	}

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, nil, err
	}

	var opts []grpc.ServerOption
	if tlsEnabled {
		tlsCfg, err := LoadServerTLS(certFile, keyFile)
		if err != nil {
			return nil, nil, err
		}
		opts = append(opts, grpc.Creds(credentials.NewTLS(tlsCfg)))
		log.Printf("gRPC server: TLS enabled on %s", addr)
	} else {
		log.Printf("gRPC server: plaintext on %s (set GRPC_TLS_ENABLED=true for TLS)", addr)
	}

	return grpc.NewServer(opts...), lis, nil
}

// NewGRPCClientDialer returns credentials for a gRPC client.
// When GRPC_TLS_ENABLED=true, loads CA cert for server verification.
// When false, returns nil (insecure, for development).
func NewGRPCClientDialer(caFile string) (credentials.TransportCredentials, error) {
	tlsEnabled := os.Getenv("GRPC_TLS_ENABLED") == "true"
	if !tlsEnabled {
		return nil, nil
	}

	tlsCfg, err := LoadClientTLS(caFile)
	if err != nil {
		return nil, err
	}
	return credentials.NewTLS(tlsCfg), nil
}

// IsTLSEnabled returns whether GRPC_TLS_ENABLED is set.
func IsTLSEnabled() bool {
	return os.Getenv("GRPC_TLS_ENABLED") == "true"
}

// TLSServerConfig returns a *tls.Config for non-gRPC TLS usage.
func TLSServerConfig(certFile, keyFile string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
	}, nil
}
