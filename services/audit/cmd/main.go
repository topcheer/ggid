// Package main is the entry point for the Audit Service.
package main

import (
	"context"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	pb "github.com/ggid/ggid/api/gen/audit/v1"

	"github.com/ggid/ggid/services/audit/internal/config"
	"github.com/ggid/ggid/services/audit/internal/domain"
	"github.com/ggid/ggid/services/audit/internal/consumer"
	"github.com/ggid/ggid/services/audit/internal/data"
	"github.com/ggid/ggid/services/audit/internal/handler"
	"github.com/ggid/ggid/services/audit/internal/repository"
	"github.com/ggid/ggid/pkg/audit"
	httpserver "github.com/ggid/ggid/services/audit/internal/server"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/ggid/ggid/services/audit/internal/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"crypto/tls"
)

// newGRPCServer creates a gRPC server with optional TLS based on GRPC_TLS_ENABLED env var.
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
			log.Printf("Warning: GRPC_TLS_ENABLED but cert/key invalid: %v, falling back to plaintext", err)
		}
	}
	return grpc.NewServer()
}

func main() {
	migrateOnly := flag.Bool("migrate-only", false, "Run migrations only and exit")
	noConsumer := flag.Bool("no-consumer", false, "Disable NATS consumer (query-only mode)")
	flag.Parse()

	cfg := config.FromEnv()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := data.New(ctx, cfg.DB)
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}
	defer db.Close()
	log.Println("Audit Service: database connected")

	// Initialize hash chain for tamper-evident audit events
	if cfg.HashChainSecret != "" {
		domain.SetHashChainSecret([]byte(cfg.HashChainSecret))
		log.Println("Audit Service: hash chain enabled")
	} else {
		log.Println("Audit Service: WARNING — hash chain disabled (set AUDIT_HASH_CHAIN_SECRET to enable)")
	}

	// Initialize repository and service
	repo := repository.NewAuditRepository(db)
	auditSvc := service.NewAuditService(repo)

	if *migrateOnly {
		log.Println("Audit Service: migration mode, skipping server start")
		return
	}

	// Start NATS consumer (unless disabled)
	var natsConsumer *consumer.EventConsumer
	if !*noConsumer {
		nc, err := consumer.New(ctx, cfg.NATS, repo)
		if err != nil {
			log.Fatalf("failed to create NATS consumer: %v", err)
		}
		if err := nc.Start(); err != nil {
			log.Fatalf("failed to start NATS consumer: %v", err)
		}
		natsConsumer = nc
		log.Println("Audit Service: NATS consumer started")
	}

	// Initialize gRPC handler
	auditHandler := handler.NewAuditHandler(auditSvc)

	// Start gRPC server (TLS-aware via GRPC_TLS_ENABLED env var)
	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", cfg.GRPCAddr, err)
	}
	grpcServer := newGRPCServer()
	pb.RegisterAuditServiceServer(grpcServer, auditHandler)

	go func() {
		log.Printf("Audit Service: gRPC listening on %s", cfg.GRPCAddr)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("gRPC serve: %v", err)
		}
	}()

	// HTTP server (health + REST API for Console)
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ready"))
	})
	// REST API endpoints
	httpAPI := httpserver.NewHTTPServer(auditSvc)
	httpAPI.RegisterRoutes(mux)

	httpServer := &http.Server{Addr: cfg.HTTPAddr, Handler: mux}

	go func() {
		log.Printf("Audit Service: HTTP listening on %s", cfg.HTTPAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Audit Service: shutting down...")
	grpcServer.GracefulStop()
	if natsConsumer != nil {
		natsConsumer.Close()
	}
	httpServer.Shutdown(context.Background())
	log.Println("Audit Service: stopped")
}