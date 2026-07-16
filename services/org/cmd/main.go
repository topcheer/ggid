// Package main is the entry point for the Org Service.
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

	pb "github.com/ggid/ggid/api/gen/org/v1"
	"github.com/ggid/ggid/pkg/audit"
	"github.com/ggid/ggid/services/org/internal/config"
	"github.com/ggid/ggid/services/org/internal/data"
	"github.com/ggid/ggid/services/org/internal/handler"
	"github.com/ggid/ggid/services/org/internal/repository"
	httpserver "github.com/ggid/ggid/services/org/internal/server"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/ggid/ggid/services/org/internal/service"
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
	migrateOnly := flag.Bool("migrate-only", false, "Run migrations only and exit")
	flag.Parse()

	cfg := config.FromEnv()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := data.New(ctx, cfg.DB)
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}
	defer db.Close()
	log.Println("Org Service: database connected")

	// Initialize repositories
	tenantRepo := repository.NewTenantRepository(db)
	orgRepo := repository.NewOrgRepository(db)
	deptRepo := repository.NewDeptRepository(db)
	teamRepo := repository.NewTeamRepository(db)
	memberRepo := repository.NewMembershipRepository(db)

	// Initialize services
	tenantSvc := service.NewTenantService(tenantRepo)
	orgSvc := service.NewOrgService(orgRepo)
	deptSvc := service.NewDeptService(deptRepo)
	teamSvc := service.NewTeamService(teamRepo)
	memberSvc := service.NewMembershipService(memberRepo)

	if *migrateOnly {
		log.Println("Org Service: migration mode, skipping server start")
		return
	}

	// Initialize gRPC handlers
	// Initialize NATS audit publisher (best-effort — service runs without NATS)
	var auditor *audit.Publisher
	if pub, err := audit.NewPublisher(ctx, cfg.NATSURL); err != nil {
		log.Printf("Org Service: NATS unavailable, audit events disabled: %v", err)
	} else {
		auditor = pub
		defer auditor.Close()
		log.Println("Org Service: NATS audit publisher connected")
	}

	tenantHandler := handler.NewTenantHandler(tenantSvc, auditor)
	orgHandler := handler.NewOrgHandler(orgSvc)
	deptHandler := handler.NewDeptHandler(deptSvc)
	teamHandler := handler.NewTeamHandler(teamSvc)
	memberHandler := handler.NewMembershipHandler(memberSvc, auditor)

	// Start gRPC server
	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", cfg.GRPCAddr, err)
	}
	grpcServer := newGRPCServer()
	pb.RegisterTenantServiceServer(grpcServer, tenantHandler)
	pb.RegisterOrganizationServiceServer(grpcServer, orgHandler)
	pb.RegisterDepartmentServiceServer(grpcServer, deptHandler)
	pb.RegisterTeamServiceServer(grpcServer, teamHandler)
	pb.RegisterMembershipServiceServer(grpcServer, memberHandler)

	go func() {
		log.Printf("Org Service: gRPC listening on %s", cfg.GRPCAddr)
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
	httpAPI := httpserver.NewHTTPServer(orgSvc, deptSvc, teamSvc, memberSvc)
	httpAPI.RegisterRoutes(mux)

	httpServer := &http.Server{
		Addr: cfg.HTTPAddr,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rvr := recover(); rvr != nil {
					log.Printf("PANIC recovered in org handler: %v", rvr)
					http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
				}
			}()
			mux.ServeHTTP(w, r)
		}),
	}

	go func() {
		log.Printf("Org Service: HTTP listening on %s", cfg.HTTPAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Org Service: shutting down...")
	grpcServer.GracefulStop()
	httpServer.Shutdown(context.Background())
	log.Println("Org Service: stopped")
}