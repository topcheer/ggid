// Package conf holds configuration types for the OAuth/OIDC Service.
package conf

import (
	"time"
)

// Config holds all configuration for the OAuth Service.
type Config struct {
	HTTP struct {
		Addr    string        `yaml:"addr"`
		Timeout time.Duration `yaml:"timeout"`
	} `yaml:"http"`

	GRPC struct {
		Addr    string        `yaml:"addr"`
		Timeout time.Duration `yaml:"timeout"`
	} `yaml:"grpc"`

	Database DBConfig `yaml:"database"`

	// OAuth/OIDC configuration
	Issuer         string `yaml:"issuer"`           // e.g. https://auth.ggid.dev
	PrivateKeyPath string `yaml:"private_key_path"` // RSA private key for signing
	PublicKeyPath  string `yaml:"public_key_path"`  // RSA public key for JWKS
	CodeTTL        time.Duration `yaml:"code_ttl"`   // authorization code lifetime
}

// DBConfig holds database connection parameters.
type DBConfig struct {
	URL             string        `yaml:"url"`
	MaxConns        int32         `yaml:"max_conns"`
	MinConns        int32         `yaml:"min_conns"`
	MaxConnLifetime time.Duration `yaml:"max_conn_lifetime"`
	MaxConnIdleTime time.Duration `yaml:"max_conn_idle_time"`
}

// Default returns sensible default configuration.
func Default() *Config {
	cfg := &Config{}
	cfg.HTTP.Addr = ":9005"
	cfg.GRPC.Addr = ":9006"
	cfg.Issuer = "http://localhost:9005"
	cfg.PrivateKeyPath = "configs/rsa_private.pem"
	cfg.PublicKeyPath = "configs/rsa_public.pem"
	cfg.CodeTTL = 10 * time.Minute
	cfg.Database.URL = "postgres://ggid:ggid@localhost:5432/ggid?sslmode=disable"
	cfg.Database.MaxConns = 20
	cfg.Database.MinConns = 2
	cfg.Database.MaxConnLifetime = time.Hour
	cfg.Database.MaxConnIdleTime = 30 * time.Minute
	return cfg
}
