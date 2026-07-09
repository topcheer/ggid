// Package main is the entry point for the Audit Service.
package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/ggid/ggid/services/audit/internal/config"
	"github.com/ggid/ggid/services/audit/internal/consumer"
	"github.com/ggid/ggid/services/audit/internal/data"
	"github.com/ggid/ggid/services/audit/internal/repository"
	"github.com/ggid/ggid/services/audit/internal/service"
)

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

	// Initialize repository and service
	repo := repository.NewAuditRepository(db)
	auditSvc := service.NewAuditService(repo)

	_ = auditSvc // gRPC handler wiring will come after proto generation

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

	// HTTP health server
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

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
	cancel()
	if natsConsumer != nil {
		natsConsumer.Close()
	}
	httpServer.Shutdown(context.Background())
	log.Println("Audit Service: stopped")
}
