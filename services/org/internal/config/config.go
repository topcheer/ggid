// Package config holds configuration for the Org Service.
package config

import (
	"os"
	"strconv"
	"time"

	"github.com/ggid/ggid/services/org/internal/data"
)

// Config holds all configuration for the Org Service.
type Config struct {
	GRPCAddr string
	HTTPAddr string
	DB       data.Config
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
			MaxConns:        int32(getEnvInt("DB_MAX_CONNS", 20)),
			MinConns:        int32(getEnvInt("DB_MIN_CONNS", 2)),
			MaxConnLifetime: time.Duration(getEnvInt("DB_CONN_LIFETIME", 300)) * time.Second,
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
