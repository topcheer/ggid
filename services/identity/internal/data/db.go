// Package data provides database connection management for the Identity Service.
package data

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DBConfig holds connection pool parameters.
type DBConfig struct {
	URL             string        // postgres://user:pass@host:5432/dbname?sslmode=disable
	MaxConns        int32         // default 20
	MinConns        int32         // default 2
	MaxConnLifetime time.Duration // default 1h
	MaxConnIdleTime time.Duration // default 30m
}

// NewDB creates and returns a configured pgxpool connection.
func NewDB(ctx context.Context, cfg DBConfig) (*pgxpool.Pool, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("database URL is required")
	}
	if cfg.MaxConns == 0 {
		cfg.MaxConns = 20
	}
	if cfg.MinConns == 0 {
		cfg.MinConns = 2
	}
	if cfg.MaxConnLifetime == 0 {
		cfg.MaxConnLifetime = time.Hour
	}
	if cfg.MaxConnIdleTime == 0 {
		cfg.MaxConnIdleTime = 30 * time.Minute
	}

	poolCfg, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse db config: %w", err)
	}

	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.MinConns = cfg.MinConns
	poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	poolCfg.MaxConnIdleTime = cfg.MaxConnIdleTime

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	// Verify connectivity.
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return pool, nil
}
