// Package main is the entry point for the Policy Engine service.
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

	"github.com/ggid/ggid/services/policy/internal/config"
	"github.com/ggid/ggid/services/policy/internal/data"
	"github.com/ggid/ggid/services/policy/internal/repository"
	"github.com/ggid/ggid/services/policy/internal/service"
)

func main() {
	var (
		migrateOnly bool
	)
	flag.BoolVar(&migrateOnly, "migrate-only", false, "Run database migrations and exit")
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
	evaluator := service.NewEvaluator(roleRepo, permRepo, policyRepo, userRoleRepo)

	_ = roleSvc
	_ = policySvc
	_ = evaluator

	if migrateOnly {
		log.Println("Policy Engine: migration mode, skipping server start")
		return
	}

	// Start HTTP health server (skeleton — gRPC server will be added after proto generation)
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	httpServer := &http.Server{Addr: cfg.HTTPAddr, Handler: mux}

	go func() {
		log.Printf("Policy Engine: HTTP listening on %s", cfg.HTTPAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Start gRPC listener (placeholder)
	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", cfg.GRPCAddr, err)
	}
	_ = lis // gRPC server registration will happen after proto generation
	log.Printf("Policy Engine: gRPC listening on %s", cfg.GRPCAddr)

	// Wait for shutdown signal
	// Keep gRPC listener open until shutdown
	go func() {
		for {
			conn, err := lis.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Policy Engine: shutting down...")
	cancel()
	httpServer.Shutdown(context.Background())
	log.Println("Policy Engine: stopped")
}
