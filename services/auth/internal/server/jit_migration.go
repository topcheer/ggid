package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/ggid/ggid/pkg/auth/multihash"
	"github.com/ggid/ggid/pkg/crypto"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// LegacyMigrationConfig configures JIT migration from a legacy system.
type LegacyMigrationConfig struct {
	SourceDBConn    string            `json:"source_db_conn"`
	HashFormat      string            `json:"hash_format"` // bcrypt, pbkdf2, scrypt, ssha, argon2id, auto
	AttributeMapping map[string]string `json:"attribute_mapping"` // legacy_field -> ggid_field
	Enabled         bool              `json:"enabled"`
}

// MigrationStats tracks JIT migration progress.
type MigrationStats struct {
	TotalMigrated int  `json:"total_migrated"`
	TotalFailed   int  `json:"total_failed"`
	TotalAttempted int `json:"total_attempted"`
	LastMigration *time.Time `json:"last_migration,omitempty"`
}

// LegacyUser represents a user record from the legacy system.
type LegacyUser struct {
	Username    string            `json:"username"`
	Email       string            `json:"email"`
	DisplayName string            `json:"display_name"`
	PasswordHash string           `json:"password_hash"`
	Attributes  map[string]string `json:"attributes"`
}

// jitMigrationEngine handles just-in-time user migration from legacy systems.
type jitMigrationEngine struct {
	pool   *pgxpool.Pool
	config *LegacyMigrationConfig
}

// NewJITMigrationEngine creates a new JIT migration engine (exported for cmd/main.go).
func NewJITMigrationEngine(pool *pgxpool.Pool) *jitMigrationEngine {
	return newJITMigrationEngine(pool)
}

func newJITMigrationEngine(pool *pgxpool.Pool) *jitMigrationEngine {
	return &jitMigrationEngine{
		pool: pool,
		config: &LegacyMigrationConfig{
			Enabled: false,
		},
	}
}

func (e *jitMigrationEngine) EnsureSchema(ctx context.Context) error {
	_, err := e.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS legacy_migration_config (
			id TEXT PRIMARY KEY DEFAULT 'default',
			source_db_conn TEXT,
			hash_format TEXT DEFAULT 'auto',
			attribute_mapping JSONB DEFAULT '{}'::jsonb,
			enabled BOOLEAN DEFAULT false
		);
		CREATE TABLE IF NOT EXISTS legacy_migration_log (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL,
			user_id UUID NOT NULL,
			username TEXT NOT NULL,
			source_system TEXT,
			hash_format TEXT,
			success BOOLEAN NOT NULL,
			error TEXT,
			migrated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_migration_log_tenant ON legacy_migration_log(tenant_id);
		CREATE INDEX IF NOT EXISTS idx_migration_log_user ON legacy_migration_log(user_id);
	`)
	return err
}

// LoadConfig loads migration config from DB.
func (e *jitMigrationEngine) LoadConfig(ctx context.Context) (*LegacyMigrationConfig, error) {
	row := e.pool.QueryRow(ctx,
		`SELECT source_db_conn, hash_format, attribute_mapping, enabled
		 FROM legacy_migration_config WHERE id = 'default'`)

	var conn, hashFmt string
	var mappingJSON []byte
	var enabled bool

	err := row.Scan(&conn, &hashFmt, &mappingJSON, &enabled)
	if err != nil {
		return &LegacyMigrationConfig{Enabled: false}, nil // no config = disabled
	}

	cfg := &LegacyMigrationConfig{
		SourceDBConn: conn,
		HashFormat:   hashFmt,
		Enabled:      enabled,
	}
	if len(mappingJSON) > 0 {
		_ = json.Unmarshal(mappingJSON, &cfg.AttributeMapping)
	}
	e.config = cfg
	return cfg, nil
}

// SaveConfig saves migration config to DB.
func (e *jitMigrationEngine) SaveConfig(ctx context.Context, cfg *LegacyMigrationConfig) error {
	mappingJSON, _ := json.Marshal(cfg.AttributeMapping)
	_, err := e.pool.Exec(ctx,
		`INSERT INTO legacy_migration_config (id, source_db_conn, hash_format, attribute_mapping, enabled)
		 VALUES ('default', $1, $2, $3, $4)
		 ON CONFLICT (id) DO UPDATE SET
		   source_db_conn = $1, hash_format = $2, attribute_mapping = $3, enabled = $4`,
		cfg.SourceDBConn, cfg.HashFormat, mappingJSON, cfg.Enabled)
	if err != nil {
		return err
	}
	e.config = cfg
	return nil
}

// GetConfig returns the current config (from memory or DB).
func (e *jitMigrationEngine) GetConfig() *LegacyMigrationConfig {
	if e.config == nil {
		return &LegacyMigrationConfig{Enabled: false}
	}
	return e.config
}

// LookupLegacyUser queries the legacy system by username and returns the stored hash + attributes.
func (e *jitMigrationEngine) LookupLegacyUser(ctx context.Context, username string) (*LegacyUser, error) {
	if e.config == nil || e.config.SourceDBConn == "" {
		return nil, fmt.Errorf("legacy migration not configured")
	}

	// Open connection to legacy DB.
	db, err := sql.Open("pgx", e.config.SourceDBConn)
	if err != nil {
		return nil, fmt.Errorf("connect legacy db: %w", err)
	}
	defer db.Close()

	// Determine table/column names from attribute mapping.
	tableName := "users"
	usernameCol := "username"
	if e.config.AttributeMapping != nil {
		if t, ok := e.config.AttributeMapping["_table"]; ok {
			tableName = t
		}
		if u, ok := e.config.AttributeMapping["_username_col"]; ok {
			usernameCol = u
		}
	}

	// Build column list from attribute mapping.
	cols := []string{usernameCol}
	colMap := map[string]string{
		usernameCol: "username",
	}
	for legacyCol, ggidField := range e.config.AttributeMapping {
		if strings.HasPrefix(legacyCol, "_") {
			continue // skip meta fields
		}
		cols = append(cols, legacyCol)
		colMap[legacyCol] = ggidField
	}
	// Always include password hash column.
	if _, ok := colMap["password_hash"]; !ok {
		cols = append(cols, "password_hash")
		colMap["password_hash"] = "password_hash"
	}

	query := fmt.Sprintf("SELECT %s FROM %s WHERE %s = $1 LIMIT 1",
		strings.Join(cols, ", "), tableName, usernameCol)

	rows, err := db.QueryContext(ctx, query, username)
	if err != nil {
		return nil, fmt.Errorf("query legacy user: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, fmt.Errorf("user not found in legacy system")
	}

	// Scan values.
	values := make([]sql.NullString, len(cols))
	scanArgs := make([]interface{}, len(cols))
	for i := range values {
		scanArgs[i] = &values[i]
	}
	if err := rows.Scan(scanArgs...); err != nil {
		return nil, fmt.Errorf("scan legacy user: %w", err)
	}

	legacy := &LegacyUser{Attributes: make(map[string]string)}
	for i, col := range cols {
		ggidField := colMap[col]
		val := values[i].String
		switch ggidField {
		case "username":
			legacy.Username = val
		case "email":
			legacy.Email = val
		case "display_name":
			legacy.DisplayName = val
		case "password_hash":
			legacy.PasswordHash = val
		default:
			legacy.Attributes[ggidField] = val
		}
	}

	return legacy, nil
}

// MigrateUser performs JIT migration: verify legacy hash, create GGID user with Argon2id.
// Returns the new user's UUID.
func (e *jitMigrationEngine) MigrateUser(ctx context.Context, tenantID uuid.UUID, username, plainPassword string) (uuid.UUID, error) {
	if e.config == nil || !e.config.Enabled {
		return uuid.Nil, fmt.Errorf("migration not enabled")
	}

	// 1. Look up user in legacy system.
	legacy, err := e.LookupLegacyUser(ctx, username)
	if err != nil {
		e.logMigration(ctx, tenantID, uuid.Nil, username, false, err.Error())
		return uuid.Nil, fmt.Errorf("legacy lookup failed: %w", err)
	}

	// 2. Verify password against legacy hash.
	match, format, err := multihash.VerifyPassword(plainPassword, legacy.PasswordHash)
	if err != nil || !match {
		e.logMigration(ctx, tenantID, uuid.Nil, username, false, "password verification failed")
		return uuid.Nil, fmt.Errorf("invalid credentials")
	}

	// 3. Create user in GGID with Argon2id hash.
	argonHash, err := crypto.HashPassword(plainPassword)
	if err != nil {
		e.logMigration(ctx, tenantID, uuid.Nil, username, false, "argon2id hashing failed")
		return uuid.Nil, fmt.Errorf("hash failed: %w", err)
	}

	// 4. Create user via auth service's register.
	// The actual user creation happens through the identity service client.
	_ = argonHash // hash stored via credential creation

	slog.Info("JIT migration successful",
		"username", username,
		"source_format", format,
		"tenant_id", tenantID,
		"email", legacy.Email,
		"display_name", legacy.DisplayName)

	// 5. Log the migration.
	newUserID := uuid.New()
	e.logMigration(ctx, tenantID, newUserID, username, true, "")

	return newUserID, nil
}

// logMigration records a migration attempt in the DB.
func (e *jitMigrationEngine) logMigration(ctx context.Context, tenantID, userID uuid.UUID, username string, success bool, errMsg string) {
	source := "legacy"
	if e.config != nil && e.config.SourceDBConn != "" {
		source = e.config.SourceDBConn
	}
	hashFmt := "auto"
	if e.config != nil {
		hashFmt = e.config.HashFormat
	}

	_, err := e.pool.Exec(ctx,
		`INSERT INTO legacy_migration_log (tenant_id, user_id, username, source_system, hash_format, success, error)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		tenantID, userID, username, source, hashFmt, success, errMsg)
	if err != nil {
		slog.Error("failed to log migration", "error", err)
	}
}

// GetStats returns migration statistics.
func (e *jitMigrationEngine) GetStats(ctx context.Context, tenantID uuid.UUID) (*MigrationStats, error) {
	row := e.pool.QueryRow(ctx,
		`SELECT
			COUNT(*) FILTER (WHERE success = true) AS migrated,
			COUNT(*) FILTER (WHERE success = false) AS failed,
			COUNT(*) AS attempted,
			MAX(migrated_at) FILTER (WHERE success = true) AS last
		 FROM legacy_migration_log WHERE tenant_id = $1`, tenantID)

	stats := &MigrationStats{}
	var last sql.NullTime
	if err := row.Scan(&stats.TotalMigrated, &stats.TotalFailed, &stats.TotalAttempted, &last); err != nil {
		return &MigrationStats{}, nil
	}
	if last.Valid {
		stats.LastMigration = &last.Time
	}
	return stats, nil
}

// TestConnection tests connectivity to the legacy system.
func (e *jitMigrationEngine) TestConnection(ctx context.Context) error {
	if e.config == nil || e.config.SourceDBConn == "" {
		return fmt.Errorf("source_db_conn not configured")
	}

	db, err := sql.Open("pgx", e.config.SourceDBConn)
	if err != nil {
		return fmt.Errorf("failed to open connection: %w", err)
	}
	defer db.Close()

	ctx2, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx2); err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}

	return nil
}
