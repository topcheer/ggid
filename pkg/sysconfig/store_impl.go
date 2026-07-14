package sysconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// PubSubChannel is the Redis channel name for config change notifications.
const PubSubChannel = "ggid:config:changed"

// Store provides hot-reloadable system configuration.
// Priority: DB > env > default. If DB/Redis is down, uses last-known + defaults.
type Store interface {
	// Get returns the current config for a tenant.
	Get(tenantID string) SystemConfig
	// Set updates a single config key for a tenant and broadcasts the change.
	Set(ctx context.Context, tenantID, key, value string) error
	// Reset removes a single config key, reverting to default.
	Reset(ctx context.Context, tenantID, key string) error
	// GetAll returns metadata for all keys (for API responses).
	GetAll(tenantID string) []ConfigKey
	// Close releases resources.
	Close() error
}

// pgStore is the production Store backed by PostgreSQL + Redis Pub/Sub.
type pgStore struct {
	pool   *pgxpool.Pool
	rdb    *redis.Client
	cache  sync.Map // tenantID → *SystemConfig
}

// NewStore creates a new config store.
// pool: PostgreSQL connection for persistence.
// rdb: Redis client for caching + Pub/Sub. May be nil (degrades to DB-only).
func NewStore(pool *pgxpool.Pool, rdb *redis.Client) Store {
	s := &pgStore{pool: pool, rdb: rdb}
	if rdb != nil {
		go s.subscribe()
	}
	return s
}

// Get returns the cached config, loading from DB if needed.
func (s *pgStore) Get(tenantID string) SystemConfig {
	if v, ok := s.cache.Load(tenantID); ok {
		return *v.(*SystemConfig)
	}
	// Load from DB
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cfg := s.loadFromDB(ctx, tenantID)
	s.cache.Store(tenantID, &cfg)
	return cfg
}

// loadFromDB reads all overrides for a tenant from the DB and applies them
// on top of the default config.
func (s *pgStore) loadFromDB(ctx context.Context, tenantID string) SystemConfig {
	cfg := DefaultSystemConfig()
	if s.pool == nil {
		return cfg
	}
	rows, err := s.pool.Query(ctx,
		"SELECT key, value, value_type FROM system_config WHERE tenant_id = $1", tenantID)
	if err != nil {
		log.Printf("sysconfig: failed to load from DB for tenant %s: %v (using defaults)", tenantID, err)
		return cfg
	}
	defer rows.Close()
	for rows.Next() {
		var key, value, valueType string
		if err := rows.Scan(&key, &value, &valueType); err != nil {
			continue
		}
		applyValue(&cfg, key, value, valueType)
	}
	return cfg
}

// Set updates a config key in the DB, refreshes the cache, and broadcasts via Pub/Sub.
func (s *pgStore) Set(ctx context.Context, tenantID, key, value string) error {
	valueType := inferType(key, value)
	_, err := s.pool.Exec(ctx,
		`INSERT INTO system_config (tenant_id, key, value, value_type, updated_at)
		 VALUES ($1, $2, $3, $4, NOW())
		 ON CONFLICT (tenant_id, key) DO UPDATE SET value = $3, value_type = $4, updated_at = NOW()`,
		tenantID, key, value, valueType)
	if err != nil {
		return fmt.Errorf("failed to update config: %w", err)
	}
	// Refresh cache
	cfg := s.loadFromDB(ctx, tenantID)
	s.cache.Store(tenantID, &cfg)
	// Broadcast change
	s.broadcast(ctx, tenantID, key)
	return nil
}

// Reset removes a config key, reverting to default.
func (s *pgStore) Reset(ctx context.Context, tenantID, key string) error {
	_, err := s.pool.Exec(ctx,
		"DELETE FROM system_config WHERE tenant_id = $1 AND key = $2", tenantID, key)
	if err != nil {
		return fmt.Errorf("failed to reset config: %w", err)
	}
	cfg := s.loadFromDB(ctx, tenantID)
	s.cache.Store(tenantID, &cfg)
	s.broadcast(ctx, tenantID, key)
	return nil
}

// GetAll returns all config keys with metadata.
func (s *pgStore) GetAll(tenantID string) []ConfigKey {
	cfg := s.Get(tenantID)
	return AllKeys(cfg)
}

// Close releases resources.
func (s *pgStore) Close() error {
	return nil
}

// subscribe listens for Pub/Sub messages and refreshes the affected tenant's cache.
func (s *pgStore) subscribe() {
	sub := s.rdb.Subscribe(context.Background(), PubSubChannel)
	defer sub.Close()
	ch := sub.Channel()
	for msg := range ch {
		var payload struct {
			TenantID string `json:"tenant_id"`
			Key      string `json:"key"`
		}
		if err := json.Unmarshal([]byte(msg.Payload), &payload); err != nil {
			continue
		}
		// Reload this tenant's config from DB
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		cfg := s.loadFromDB(ctx, payload.TenantID)
		s.cache.Store(payload.TenantID, &cfg)
		cancel()
		log.Printf("sysconfig: hot-reloaded config for tenant %s (key=%s)", payload.TenantID, payload.Key)
	}
}

// broadcast publishes a config change notification to all service instances.
func (s *pgStore) broadcast(ctx context.Context, tenantID, key string) {
	if s.rdb == nil {
		return
	}
	payload, _ := json.Marshal(map[string]string{"tenant_id": tenantID, "key": key})
	if err := s.rdb.Publish(ctx, PubSubChannel, payload).Err(); err != nil {
		log.Printf("sysconfig: failed to broadcast change: %v", err)
	}
}

// applyValue sets a single field on SystemConfig based on the key name.
func applyValue(cfg *SystemConfig, key, value, valueType string) {
	switch key {
	case "auth.max_attempts":
		cfg.AuthMaxAttempts = atoiSafe(value, cfg.AuthMaxAttempts)
	case "auth.lock_duration":
		cfg.AuthLockDuration = parseDurationSafe(value, cfg.AuthLockDuration)
	case "auth.rate_limit_per_minute":
		cfg.AuthRateLimitPerMinute = atoiSafe(value, cfg.AuthRateLimitPerMinute)
	case "auth.session_idle_timeout":
		cfg.AuthSessionIdleTimeout = parseDurationSafe(value, cfg.AuthSessionIdleTimeout)
	case "auth.session_absolute_timeout":
		cfg.AuthSessionAbsoluteTimeout = parseDurationSafe(value, cfg.AuthSessionAbsoluteTimeout)
	case "auth.session_max_concurrent":
		cfg.AuthSessionMaxConcurrent = atoiSafe(value, cfg.AuthSessionMaxConcurrent)
	case "auth.password_min_length":
		cfg.AuthPasswordMinLength = atoiSafe(value, cfg.AuthPasswordMinLength)
	case "auth.password_require_upper":
		cfg.AuthPasswordRequireUpper = atobSafe(value, cfg.AuthPasswordRequireUpper)
	case "auth.password_require_lower":
		cfg.AuthPasswordRequireLower = atobSafe(value, cfg.AuthPasswordRequireLower)
	case "auth.password_require_digit":
		cfg.AuthPasswordRequireDigit = atobSafe(value, cfg.AuthPasswordRequireDigit)
	case "auth.password_require_special":
		cfg.AuthPasswordRequireSpecial = atobSafe(value, cfg.AuthPasswordRequireSpecial)
	case "auth.password_max_age_days":
		cfg.AuthPasswordMaxAgeDays = atoiSafe(value, cfg.AuthPasswordMaxAgeDays)
	case "auth.password_history_count":
		cfg.AuthPasswordHistoryCount = atoiSafe(value, cfg.AuthPasswordHistoryCount)
	case "gateway.rate_limit_tokens":
		cfg.GatewayRateLimitTokens = atofSafe(value, cfg.GatewayRateLimitTokens)
	case "gateway.rate_limit_refill_per_sec":
		cfg.GatewayRateLimitRefillPerSec = atofSafe(value, cfg.GatewayRateLimitRefillPerSec)
	case "gateway.upstream_timeout":
		cfg.GatewayUpstreamTimeout = parseDurationSafe(value, cfg.GatewayUpstreamTimeout)
	case "gateway.body_size_limit":
		cfg.GatewayBodySizeLimit = atoi64Safe(value, cfg.GatewayBodySizeLimit)
	}
}

// inferType determines the value_type from the key name.
func inferType(key, value string) string {
	switch key {
	case "auth.lock_duration", "auth.session_idle_timeout", "auth.session_absolute_timeout", "gateway.upstream_timeout":
		return "duration"
	case "auth.max_attempts", "auth.rate_limit_per_minute", "auth.session_max_concurrent",
		"auth.password_min_length", "auth.password_max_age_days", "auth.password_history_count",
		"gateway.body_size_limit":
		return "int"
	case "auth.password_require_upper", "auth.password_require_lower",
		"auth.password_require_digit", "auth.password_require_special":
		return "bool"
	case "gateway.rate_limit_tokens", "gateway.rate_limit_refill_per_sec":
		return "float"
	default:
		return "string"
	}
}

// --- parsing helpers ---

func atoiSafe(s string, def int) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

func atoi64Safe(s string, def int64) int64 {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return def
	}
	return n
}

func atofSafe(s string, def float64) float64 {
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return def
	}
	return n
}

func atobSafe(s string, def bool) bool {
	b, err := strconv.ParseBool(s)
	if err != nil {
		return def
	}
	return b
}

func parseDurationSafe(s string, def time.Duration) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return def
	}
	return d
}
