// Package main is the entry point for the Identity Service.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ggid/ggid/services/identity/internal/conf"
	"github.com/ggid/ggid/services/identity/internal/server"
)

func main() {
	var (
		grpcAddr = flag.String("grpc-addr", ":9090", "gRPC listen address")
		httpAddr = flag.String("http-addr", ":8080", "HTTP listen address")
		dbURL    = flag.String("db-url", os.Getenv("DATABASE_URL"), "PostgreSQL connection URL")
	)
	flag.Parse()

	if *dbURL == "" {
		*dbURL = "postgres://ggid:ggid@localhost:5432/ggid?sslmode=disable"
	}

	cfg := &conf.Config{}
	cfg.GRPC.Addr = *grpcAddr
	cfg.HTTP.Addr = *httpAddr
	cfg.Database.URL = *dbURL
	cfg.Database.MaxConns = 20
	cfg.Database.MinConns = 2
	cfg.Database.MaxConnLifetime = time.Hour
	cfg.Database.MaxConnIdleTime = 30 * time.Minute

	srv, err := server.New(cfg)
	if err != nil {
		log.Fatalf("failed to create identity server: %v", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := srv.Run(ctx); err != nil {
		log.Fatalf("identity server error: %v", err)
	}

	log.Println("Identity service stopped")
}
