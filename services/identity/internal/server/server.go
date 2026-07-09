// Package server wires up and runs the Identity Service gRPC server.
package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/ggid/ggid/services/identity/internal/conf"
	"github.com/ggid/ggid/services/identity/internal/data"
	"github.com/ggid/ggid/services/identity/internal/repository"
	"github.com/ggid/ggid/services/identity/internal/service"
	"google.golang.org/grpc"
)

// Server encapsulates the gRPC and HTTP servers for the Identity Service.
type Server struct {
	cfg      *conf.Config
	grpcSrv  *grpc.Server
	httpSrv  *http.Server
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
	_ = identitySvc // used by gRPC handler (generated from proto)

	grpcSrv := grpc.NewServer()
	// TODO: register generated identity_pb.IdentityServiceServer once proto is compiled.

	httpSrv := &http.Server{
		Addr:         cfg.HTTP.Addr,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return &Server{
		cfg:     cfg,
		grpcSrv: grpcSrv,
		httpSrv: httpSrv,
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
