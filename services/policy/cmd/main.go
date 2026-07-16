// Package main is the entry point for the Policy Engine service.
package main

import (
	"context"
	"crypto/tls"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	pb "github.com/ggid/ggid/api/gen/policy/v1"
	"github.com/ggid/ggid/pkg/audit"
	"github.com/ggid/ggid/services/policy/internal/config"
	"github.com/ggid/ggid/services/policy/internal/data"
	"github.com/ggid/ggid/services/policy/internal/handler"
	"github.com/ggid/ggid/services/policy/internal/repository"
	httpserver "github.com/ggid/ggid/services/policy/internal/server"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/ggid/ggid/services/policy/internal/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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
			if os.Getenv("GRPC_TLS_ALLOW_PLAINTEXT_FALLBACK") != "true" {
				log.Fatalf("GRPC_TLS_ENABLED but cert/key invalid: %v; refusing to start with plaintext fallback. Set GRPC_TLS_ALLOW_PLAINTEXT_FALLBACK=true only in dev.", err)
			}
			log.Printf("Warning: GRPC_TLS_ENABLED but cert/key invalid: %v, falling back to plaintext because GRPC_TLS_ALLOW_PLAINTEXT_FALLBACK=true", err)
		}
	}
	return grpc.NewServer()
}

func main() {
	migrateOnly := flag.Bool("migrate-only", false, "Run database migrations and exit")
	flag.Parse()

	cfg := config.FromEnv()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Connect to PostgreSQL
	db, err := data.New(ctx, cfg.DB)
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}
	defer db.Close()
	log.Println("Policy Engine: database connected")

	// Initialize repositories
	roleRepo := repository.NewRoleRepository(db)
	permRepo := repository.NewPermissionRepository(db)
	policyRepo := repository.NewPolicyRepository(db)
	userRoleRepo := repository.NewUserRoleRepository(db)

	// Initialize services
	roleSvc := service.NewRoleService(roleRepo, permRepo, userRoleRepo)
	policySvc := service.NewPolicyService(policyRepo)
	evaluator := service.NewEvaluator(roleRepo, userRoleRepo, policyRepo)

	if *migrateOnly {
		log.Println("Policy Engine: migration mode, skipping server start")
		return
	}

	// Initialize NATS audit publisher (best-effort — service runs without NATS)
	var auditor *audit.Publisher
	if pub, err := audit.NewPublisher(ctx, cfg.NATSURL); err != nil {
		log.Printf("Policy Engine: NATS unavailable, audit events disabled: %v", err)
	} else {
		auditor = pub
		defer auditor.Close()
		log.Println("Policy Engine: NATS audit publisher connected")
	}

	// Initialize gRPC handlers
	roleHandler := handler.NewRoleHandler(roleSvc, auditor)
	permHandler := handler.NewPermissionHandler(roleSvc)
	policyHandler := handler.NewPolicyHandler(policySvc, evaluator)

	// Start gRPC server
	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", cfg.GRPCAddr, err)
	}
	grpcServer := newGRPCServer()
	pb.RegisterRoleServiceServer(grpcServer, roleHandler)
	pb.RegisterPermissionServiceServer(grpcServer, permHandler)
	pb.RegisterPolicyServiceServer(grpcServer, policyHandler)

	go func() {
		log.Printf("Policy Engine: gRPC listening on %s", cfg.GRPCAddr)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("gRPC serve: %v", err)
		}
	}()

	// Start HTTP server (health + REST API for Console)
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
	httpAPI := httpserver.NewHTTPServer(roleSvc, policySvc, evaluator)
	// Wire DB-backed campaign store for SOX-compliant access review persistence.
	httpAPI.SetCampaignRepo(httpserver.NewCampaignRepo(db))
	httpAPI.RegisterRoutes(mux)

	httpServer := &http.Server{
		Addr: cfg.HTTPAddr,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rvr := recover(); rvr != nil {
					log.Printf("PANIC recovered in policy handler: %v", rvr)
					http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
				}
			}()
			mux.ServeHTTP(w, r)
		}),
	}

	go func() {
		log.Printf("Policy Engine: HTTP listening on %s", cfg.HTTPAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Policy Engine: shutting down...")
	grpcServer.GracefulStop()
	httpServer.Shutdown(context.Background())
	log.Println("Policy Engine: stopped")
}