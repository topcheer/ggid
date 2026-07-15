// Package server wires up and runs the Identity Service gRPC server.
package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/ggid/ggid/services/identity/internal/conf"
	"github.com/ggid/ggid/services/identity/internal/data"
	"github.com/ggid/ggid/services/identity/internal/repository"
	"github.com/ggid/ggid/services/identity/internal/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Server encapsulates the gRPC and HTTP servers for the Identity Service.
type Server struct {
	cfg      *conf.Config
	grpcSrv  *grpc.Server
	httpSrv  *http.Server
	idSvc    *service.IdentityService
}

// newGRPCServer creates a gRPC server with optional TLS based on GRPC_TLS_ENABLED env var.
// When TLS is explicitly enabled but cert/key is invalid, the server fails secure by default.
// Set GRPC_TLS_ALLOW_PLAINTEXT_FALLBACK=true only in development environments to allow fallback.
func newGRPCServer() *grpc.Server {
	if os.Getenv("GRPC_TLS_ENABLED") == "true" {
		certFile := os.Getenv("GRPC_TLS_CERT")
		keyFile := os.Getenv("GRPC_TLS_KEY")
		if certFile != "" && keyFile != "" {
			tlsCfg, err := tls.LoadX509KeyPair(certFile, keyFile)
			if err == nil {
				return grpc.NewServer(grpc.Creds(credentials.NewTLS(&tls.Config{
					Certificates: []tls.Certificate{tlsCfg},
					MinVersion:   tls.VersionTLS12,
				})))
			}
			if os.Getenv("GRPC_TLS_ALLOW_PLAINTEXT_FALLBACK") != "true" {
				log.Fatalf("GRPC_TLS_ENABLED but cert/key invalid: %v; refusing to start with plaintext fallback. Set GRPC_TLS_ALLOW_PLAINTEXT_FALLBACK=true only in dev.", err)
			}
			log.Printf("Warning: GRPC_TLS_ENABLED but cert/key invalid: %v, falling back to plaintext because GRPC_TLS_ALLOW_PLAINTEXT_FALLBACK=true", err)
		}
	}
	return grpc.NewServer()
}

// New constructs a new Server with all dependencies wired up.
func New(cfg *conf.Config) (*Server, error) {
	ctx := context.Background()

	// Create database connection pool.
	dbCfg := data.DBConfig{
		URL:             cfg.Database.URL,
		MaxConns:        cfg.Database.MaxConns,
		MinConns:        cfg.Database.MinConns,
		MaxConnLifetime: cfg.Database.MaxConnLifetime,
		MaxConnIdleTime: cfg.Database.MaxConnIdleTime,
	}
	pool, err := data.NewDB(ctx, dbCfg)
	if err != nil {
		return nil, fmt.Errorf("init database: %w", err)
	}

	repo := repository.NewPGRepository(pool)
	identitySvc := service.NewIdentityService(repo)

	grpcSrv := newGRPCServer()
	// gRPC handlers will be registered once proto code is generated.

	httpHandler := NewHTTPHandler(identitySvc)
	httpSrv := &http.Server{
		Addr:         cfg.HTTP.Addr,
		Handler:      httpHandler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return &Server{
		cfg:     cfg,
		grpcSrv: grpcSrv,
		httpSrv: httpSrv,
		idSvc:   identitySvc,
	}, nil
}

// Run starts the gRPC and HTTP servers and blocks until ctx is cancelled.
func (s *Server) Run(ctx context.Context) error {
	// gRPC listener
	lis, err := net.Listen("tcp", s.cfg.GRPC.Addr)
	if err != nil {
		return fmt.Errorf("listen grpc: %w", err)
	}

	errCh := make(chan error, 2)

	go func() {
		log.Printf("Identity gRPC server listening on %s", s.cfg.GRPC.Addr)
		if err := s.grpcSrv.Serve(lis); err != nil {
			errCh <- fmt.Errorf("grpc serve: %w", err)
		}
	}()

	go func() {
		log.Printf("Identity HTTP server listening on %s", s.cfg.HTTP.Addr)
		if err := s.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("http serve: %w", err)
		}
	}()

	select {
	case <-ctx.Done():
		log.Println("Identity server shutting down...")
		s.grpcSrv.GracefulStop()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = s.httpSrv.Shutdown(shutdownCtx)
		return nil
	case err := <-errCh:
		return err
	}
}
