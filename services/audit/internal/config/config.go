// Package config provides configuration for the Audit Service.
package config

import (
	"os"
	"strconv"
	"time"

	"github.com/ggid/ggid/services/audit/internal/consumer"
	"github.com/ggid/ggid/services/audit/internal/data"
)

// Config holds all configuration for the Audit Service.
type Config struct {
	GRPCAddr        string
	HTTPAddr        string
	DB              data.Config
	NATS            consumer.Config
	HashChainSecret string
}

func FromEnv() *Config {
	return &Config{
		GRPCAddr: getEnv("AUDIT_GRPC_ADDR", ":9072"),
		HTTPAddr: getEnv("AUDIT_HTTP_ADDR", ":8072"),
		DB: data.Config{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnvInt("DB_PORT", 5432),
			User:            getEnv("DB_USER", "ggid"),
			Password:        getEnv("DB_PASSWORD", "ggid"),
			Database:        getEnv("DB_DATABASE", "ggid"),
			SSLMode:         getEnv("DB_SSL_MODE", "disable"),
			MaxConns:        int32(getEnvInt("DB_MAX_CONNS", 20)),
			MinConns:        int32(getEnvInt("DB_MIN_CONNS", 2)),
			MaxConnLifetime: time.Duration(getEnvInt("DB_CONN_LIFETIME", 300)) * time.Second,
		},
		HashChainSecret: getEnv("AUDIT_HASH_CHAIN_SECRET", ""),
		NATS: consumer.Config{
			URL:        getEnv("NATS_URL", "nats://localhost:4222"),
			StreamName: getEnv("NATS_STREAM", "AUDIT"),
			Subject:    getEnv("NATS_SUBJECT", "audit.events"),
			Consumer:   getEnv("NATS_CONSUMER", "audit-db-writer"),
			MaxDeliver: getEnvInt("NATS_MAX_DELIVER", 3),
			BatchSize:  getEnvInt("NATS_BATCH_SIZE", 10),
		},
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
