// Package main is the entry point for the Org Service.
package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/ggid/ggid/services/org/internal/config"
	"github.com/ggid/ggid/services/org/internal/data"
	"github.com/ggid/ggid/services/org/internal/repository"
	"github.com/ggid/ggid/services/org/internal/service"
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

	_ = tenantSvc
	_ = orgSvc
	_ = deptSvc
	_ = teamSvc
	_ = memberSvc

	if *migrateOnly {
		log.Println("Org Service: migration mode, skipping server start")
		return
	}

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
	cancel()
	httpServer.Shutdown(context.Background())
	log.Println("Org Service: stopped")
}
