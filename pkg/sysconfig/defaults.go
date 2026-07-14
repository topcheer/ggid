// Package sysconfig provides a hot-reloadable configuration store backed by
// PostgreSQL (source of truth) and Redis (cache + Pub/Sub for instant updates).
//
// Priority: DB value > env var > hardcoded default.
//
// All config keys have defaults baked into code, so services start correctly
// even with an empty system_config table or no Redis connection.
package sysconfig

import (
	"time"
)

// SystemConfig holds all runtime-tunable parameters for the GGID platform.
// Every field has a sensible default — see DefaultSystemConfig().
type SystemConfig struct {
	// Auth — account lockout
	AuthMaxAttempts          int           `json:"auth.max_attempts"`
	AuthLockDuration         time.Duration `json:"auth.lock_duration"`
	AuthRateLimitPerMinute   int           `json:"auth.rate_limit_per_minute"`

	// Auth — session policy
	AuthSessionIdleTimeout     time.Duration `json:"auth.session_idle_timeout"`
	AuthSessionAbsoluteTimeout time.Duration `json:"auth.session_absolute_timeout"`
	AuthSessionMaxConcurrent   int           `json:"auth.session_max_concurrent"`

	// Auth — password policy
	AuthPasswordMinLength       int    `json:"auth.password_min_length"`
	AuthPasswordRequireUpper    bool   `json:"auth.password_require_upper"`
	AuthPasswordRequireLower    bool   `json:"auth.password_require_lower"`
	AuthPasswordRequireDigit    bool   `json:"auth.password_require_digit"`
	AuthPasswordRequireSpecial  bool   `json:"auth.password_require_special"`
	AuthPasswordMaxAgeDays      int    `json:"auth.password_max_age_days"`
	AuthPasswordHistoryCount    int    `json:"auth.password_history_count"`

	// Gateway — rate limiting
	GatewayRateLimitTokens       float64 `json:"gateway.rate_limit_tokens"`
	GatewayRateLimitRefillPerSec float64 `json:"gateway.rate_limit_refill_per_sec"`

	// Gateway — request limits (global)
	GatewayUpstreamTimeout time.Duration `json:"gateway.upstream_timeout"`
	GatewayBodySizeLimit   int64          `json:"gateway.body_size_limit"`
}

// DefaultSystemConfig returns production-safe defaults.
// These values are used when no DB override or env var exists.
func DefaultSystemConfig() SystemConfig {
	return SystemConfig{
		// Auth lockout
		AuthMaxAttempts:        5,
		AuthLockDuration:       30 * time.Minute,
		AuthRateLimitPerMinute: 5,

		// Session policy
		AuthSessionIdleTimeout:     30 * time.Minute,
		AuthSessionAbsoluteTimeout: 8 * time.Hour,
		AuthSessionMaxConcurrent:   0, // 0 = unlimited

		// Password policy
		AuthPasswordMinLength:      12,
		AuthPasswordRequireUpper:   true,
		AuthPasswordRequireLower:   true,
		AuthPasswordRequireDigit:   true,
		AuthPasswordRequireSpecial: true,
		AuthPasswordMaxAgeDays:     90,
		AuthPasswordHistoryCount:   5,

		// Gateway rate limit
		GatewayRateLimitTokens:       100,
		GatewayRateLimitRefillPerSec: 10, // 600/min sustained

		// Gateway request limits
		GatewayUpstreamTimeout: 30 * time.Second,
		GatewayBodySizeLimit:   10 * 1024 * 1024, // 10MB
	}
}

// ConfigKey describes a single configuration key for the API.
type ConfigKey struct {
	Key          string      `json:"key"`
	Value        interface{} `json:"value"`
	Default      interface{} `json:"default"`
	Type         string      `json:"type"` // int, bool, duration, float, string
	Description  string      `json:"description"`
	IsDefault    bool        `json:"is_default"` // true if value == default
}

// AllKeys returns metadata for every config key, useful for API responses.
func AllKeys(cfg SystemConfig) []ConfigKey {
	def := DefaultSystemConfig()
	return []ConfigKey{
		{Key: "auth.max_attempts", Value: cfg.AuthMaxAttempts, Default: def.AuthMaxAttempts, Type: "int", Description: "Max failed login attempts before account lockout", IsDefault: cfg.AuthMaxAttempts == def.AuthMaxAttempts},
		{Key: "auth.lock_duration", Value: cfg.AuthLockDuration.String(), Default: def.AuthLockDuration.String(), Type: "duration", Description: "How long to lock an account after max attempts", IsDefault: cfg.AuthLockDuration == def.AuthLockDuration},
		{Key: "auth.rate_limit_per_minute", Value: cfg.AuthRateLimitPerMinute, Default: def.AuthRateLimitPerMinute, Type: "int", Description: "Max login attempts per minute per IP", IsDefault: cfg.AuthRateLimitPerMinute == def.AuthRateLimitPerMinute},
		{Key: "auth.session_idle_timeout", Value: cfg.AuthSessionIdleTimeout.String(), Default: def.AuthSessionIdleTimeout.String(), Type: "duration", Description: "Inactivity session timeout", IsDefault: cfg.AuthSessionIdleTimeout == def.AuthSessionIdleTimeout},
		{Key: "auth.session_absolute_timeout", Value: cfg.AuthSessionAbsoluteTimeout.String(), Default: def.AuthSessionAbsoluteTimeout.String(), Type: "duration", Description: "Maximum session lifetime", IsDefault: cfg.AuthSessionAbsoluteTimeout == def.AuthSessionAbsoluteTimeout},
		{Key: "auth.session_max_concurrent", Value: cfg.AuthSessionMaxConcurrent, Default: def.AuthSessionMaxConcurrent, Type: "int", Description: "Max concurrent sessions per user (0 = unlimited)", IsDefault: cfg.AuthSessionMaxConcurrent == def.AuthSessionMaxConcurrent},
		{Key: "auth.password_min_length", Value: cfg.AuthPasswordMinLength, Default: def.AuthPasswordMinLength, Type: "int", Description: "Minimum password length", IsDefault: cfg.AuthPasswordMinLength == def.AuthPasswordMinLength},
		{Key: "auth.password_require_upper", Value: cfg.AuthPasswordRequireUpper, Default: def.AuthPasswordRequireUpper, Type: "bool", Description: "Require uppercase letters in password", IsDefault: cfg.AuthPasswordRequireUpper == def.AuthPasswordRequireUpper},
		{Key: "auth.password_require_lower", Value: cfg.AuthPasswordRequireLower, Default: def.AuthPasswordRequireLower, Type: "bool", Description: "Require lowercase letters in password", IsDefault: cfg.AuthPasswordRequireLower == def.AuthPasswordRequireLower},
		{Key: "auth.password_require_digit", Value: cfg.AuthPasswordRequireDigit, Default: def.AuthPasswordRequireDigit, Type: "bool", Description: "Require digits in password", IsDefault: cfg.AuthPasswordRequireDigit == def.AuthPasswordRequireDigit},
		{Key: "auth.password_require_special", Value: cfg.AuthPasswordRequireSpecial, Default: def.AuthPasswordRequireSpecial, Type: "bool", Description: "Require special characters in password", IsDefault: cfg.AuthPasswordRequireSpecial == def.AuthPasswordRequireSpecial},
		{Key: "auth.password_max_age_days", Value: cfg.AuthPasswordMaxAgeDays, Default: def.AuthPasswordMaxAgeDays, Type: "int", Description: "Password max age in days before forced reset", IsDefault: cfg.AuthPasswordMaxAgeDays == def.AuthPasswordMaxAgeDays},
		{Key: "auth.password_history_count", Value: cfg.AuthPasswordHistoryCount, Default: def.AuthPasswordHistoryCount, Type: "int", Description: "Number of old passwords to check against", IsDefault: cfg.AuthPasswordHistoryCount == def.AuthPasswordHistoryCount},
		{Key: "gateway.rate_limit_tokens", Value: cfg.GatewayRateLimitTokens, Default: def.GatewayRateLimitTokens, Type: "float", Description: "Max token bucket size for rate limiting", IsDefault: cfg.GatewayRateLimitTokens == def.GatewayRateLimitTokens},
		{Key: "gateway.rate_limit_refill_per_sec", Value: cfg.GatewayRateLimitRefillPerSec, Default: def.GatewayRateLimitRefillPerSec, Type: "float", Description: "Token bucket refill rate per second", IsDefault: cfg.GatewayRateLimitRefillPerSec == def.GatewayRateLimitRefillPerSec},
		{Key: "gateway.upstream_timeout", Value: cfg.GatewayUpstreamTimeout.String(), Default: def.GatewayUpstreamTimeout.String(), Type: "duration", Description: "Upstream proxy timeout", IsDefault: cfg.GatewayUpstreamTimeout == def.GatewayUpstreamTimeout},
		{Key: "gateway.body_size_limit", Value: cfg.GatewayBodySizeLimit, Default: def.GatewayBodySizeLimit, Type: "int", Description: "Max request body size in bytes", IsDefault: cfg.GatewayBodySizeLimit == def.GatewayBodySizeLimit},
	}
}
