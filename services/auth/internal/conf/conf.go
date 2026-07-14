// Package conf defines the configuration structures for the Auth Service.
package conf

import (
	"fmt"
	"os"
	"time"
)

// Config is the root configuration for the Auth Service.
type Config struct {
	Server          ServerConfig       `yaml:"server"`
	Database        DatabaseConfig     `yaml:"database"`
	Redis           RedisConfig        `yaml:"redis"`
	JWT             JWTConfig          `yaml:"jwt"`
	Password        PasswordPolicy     `yaml:"password_policy"`
	RateLimit       RateLimitConfig    `yaml:"rate_limit"`
	SessionTimeout  SessionTimeoutConfig `yaml:"session_timeout"`
}

type ServerConfig struct {
	HTTP HTTPConfig `yaml:"http"`
}

type HTTPConfig struct {
	Addr         string        `yaml:"addr"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
}

type DatabaseConfig struct {
	URL          string `yaml:"url"`
	MaxOpenConns int32  `yaml:"max_open_conns"`
	MaxIdleConns int32  `yaml:"max_idle_conns"`
}

type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type JWTConfig struct {
	PrivateKeyPath string        `yaml:"private_key_path"`
	PublicKeyPath  string        `yaml:"public_key_path"`
	Issuer         string        `yaml:"issuer"`
	Audience       string        `yaml:"audience"`
	AccessTokenTTL time.Duration `yaml:"access_token_ttl"`
}

type PasswordPolicy struct {
	MinLength      int           `yaml:"min_length"`
	RequireUpper   bool          `yaml:"require_upper"`
	RequireLower   bool          `yaml:"require_lower"`
	RequireDigit   bool          `yaml:"require_digit"`
	RequireSpecial bool          `yaml:"require_special"`
	Blacklist      []string      `yaml:"blacklist"`
	HistoryCount   int           `yaml:"history_count"`
	MaxAttempts    int           `yaml:"max_attempts"`
	LockDuration   time.Duration `yaml:"lock_duration"`
	MaxAgeDays     int           `yaml:"max_age_days"`
}

type RateLimitConfig struct {
	LoginPerMinute int `yaml:"login_per_minute"`
}

// SessionTimeoutConfig controls session expiration and concurrency policy.
type SessionTimeoutConfig struct {
	AbsoluteTimeout time.Duration `yaml:"absolute_timeout"`   // max session lifetime (e.g. 8h)
	IdleTimeout     time.Duration `yaml:"idle_timeout"`       // inactivity timeout (e.g. 30m)
	MaxSessions     int           `yaml:"max_sessions"`       // max concurrent sessions per user (0 = unlimited)
}

// Default returns the default configuration with sensible production values.
func Default() *Config {
	return &Config{
		Server: ServerConfig{
			HTTP: HTTPConfig{
				Addr:         ":9001",
				ReadTimeout:  10 * time.Second,
				WriteTimeout: 10 * time.Second,
			},
		},
		Database: DatabaseConfig{
			URL:          "postgres://ggid:ggid@localhost:5432/ggid?sslmode=disable",
			MaxOpenConns: 25,
			MaxIdleConns: 5,
		},
		Redis: RedisConfig{
			Addr: "localhost:6379",
			DB:   0,
		},
		JWT: JWTConfig{
			PrivateKeyPath: "configs/rsa_private.pem",
			PublicKeyPath:  "configs/rsa_public.pem",
			Issuer:         "ggid-auth",
			Audience:       "ggid",
			AccessTokenTTL: 15 * time.Minute,
		},
		Password: PasswordPolicy{
			MinLength:      12,
			RequireUpper:   true,
			RequireLower:   true,
			RequireDigit:   true,
			RequireSpecial: false,
			HistoryCount:   5,
			MaxAttempts:    5,
			LockDuration:   30 * time.Minute,
		},
		RateLimit: RateLimitConfig{
			LoginPerMinute: 5,
		},
		SessionTimeout: SessionTimeoutConfig{
			AbsoluteTimeout: 8 * time.Hour,
			IdleTimeout:     30 * time.Minute,
		},
	}
}

// LoadFromEnv overrides config values from environment variables.
func LoadFromEnv(cfg *Config) *Config {
	if v := os.Getenv("AUTH_HTTP_ADDR"); v != "" {
		cfg.Server.HTTP.Addr = v
	}
	if v := os.Getenv("DATABASE_URL"); v != "" {
		cfg.Database.URL = v
	}
	if v := os.Getenv("REDIS_ADDR"); v != "" {
		cfg.Redis.Addr = v
	}
	if v := os.Getenv("JWT_PRIVATE_KEY_PATH"); v != "" {
		cfg.JWT.PrivateKeyPath = v
	}
	if v := os.Getenv("JWT_PUBLIC_KEY_PATH"); v != "" {
		cfg.JWT.PublicKeyPath = v
	}
	// Account lockout configuration
	if v := os.Getenv("AUTH_MAX_ATTEMPTS"); v != "" {
		if n, err := parseIntDefault(v, cfg.Password.MaxAttempts); err == nil {
			cfg.Password.MaxAttempts = n
		}
	}
	if v := os.Getenv("AUTH_LOCK_DURATION"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Password.LockDuration = d
		}
	}
	// Rate limit configuration
	if v := os.Getenv("AUTH_RATE_LIMIT_PER_MINUTE"); v != "" {
		if n, err := parseIntDefault(v, cfg.RateLimit.LoginPerMinute); err == nil {
			cfg.RateLimit.LoginPerMinute = n
		}
	}
	return cfg
}

// parseIntDefault converts s to int, returning default on error.
func parseIntDefault(s string, def int) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	if err != nil {
		return def, err
	}
	return n, nil
}
