// Package config holds configuration for the Org Service.
package config

import (
	"time"

	"github.com/ggid/ggid/pkg/httputil"
	"github.com/ggid/ggid/services/org/internal/data"
)

// Config holds all configuration for the Org Service.
type Config struct {
	GRPCAddr string
	HTTPAddr string
	DB       data.Config
	NATSURL  string
}

func FromEnv() *Config {
	return &Config{
		GRPCAddr: getEnv("ORG_GRPC_ADDR", ":9071"),
		HTTPAddr: getEnv("ORG_HTTP_ADDR", ":8071"),
		DB: data.Config{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnvInt("DB_PORT", 5432),
			User:            getEnv("DB_USER", "ggid"),
			Password:        getEnv("DB_PASSWORD", "ggid"),
			Database:        getEnv("DB_DATABASE", "ggid"),
			SSLMode:         getEnv("DB_SSL_MODE", "disable"),
			MaxConns:        int32(getEnvInt("DB_MAX_CONNS", 20)), //nolint:gosec // G115: safe, values are small
			MinConns:        int32(getEnvInt("DB_MIN_CONNS", 2)),  //nolint:gosec // G115: safe, values are small
			MaxConnLifetime: time.Duration(getEnvInt("DB_CONN_LIFETIME", 300)) * time.Second,
		},
		NATSURL: getEnv("NATS_URL", "nats://localhost:4222"),
	}
}

func getEnv(key, def string) string {
	return httputil.GetEnv(key, def)
}

func getEnvInt(key string, def int) int {
	return httputil.GetEnvInt(key, def)
}
