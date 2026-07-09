// Package main is the entry point for the Org Service.
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

	pb "github.com/ggid/ggid/api/gen/org/v1"
	"github.com/ggid/ggid/services/org/internal/config"
	"github.com/ggid/ggid/services/org/internal/data"
	"github.com/ggid/ggid/services/org/internal/handler"
	"github.com/ggid/ggid/services/org/internal/repository"
	"github.com/ggid/ggid/services/org/internal/service"
	"google.golang.org/grpc"
)

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
	tenantHandler := handler.NewTenantHandler(tenantSvc, nil)
	orgHandler := handler.NewOrgHandler(orgSvc)
	deptHandler := handler.NewDeptHandler(deptSvc)
	teamHandler := handler.NewTeamHandler(teamSvc)
	memberHandler := handler.NewMembershipHandler(memberSvc, nil)

	// Start gRPC server
	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", cfg.GRPCAddr, err)
	}
	grpcServer := grpc.NewServer()
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

	// HTTP health server
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	httpServer := &http.Server{Addr: cfg.HTTPAddr, Handler: mux}

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