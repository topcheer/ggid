package db

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Migration represents a single database migration.
type Migration struct {
	Version  int
	Name     string
	UpFile   string
	DownFile string
}

// MigrationStatus represents the status of a migration.
type MigrationStatus struct {
	Version   int       `json:"version"`
	Name      string    `json:"name"`
	Applied   bool      `json:"applied"`
	AppliedAt time.Time `json:"applied_at,omitempty"`
}

// EnsureMigrationsTable creates the schema_migrations table if it doesn't exist.
func EnsureMigrationsTable(ctx context.Context, p Pool) error {
	var query string
	switch p.Driver() {
	case DriverPostgres:
		query = `CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`
	case DriverMySQL:
		query = `CREATE TABLE IF NOT EXISTS schema_migrations (
			version INT PRIMARY KEY,
			applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`
	case DriverSQLite:
		query = `CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`
	default:
		return fmt.Errorf("unsupported driver: %s", p.Driver())
	}
	_, err := p.Exec(ctx, query)
	return err
}

// GetMigrationVersion returns the highest applied migration version.
// Returns 0 if no migrations have been applied.
func GetMigrationVersion(ctx context.Context, p Pool) (int, error) {
	if err := EnsureMigrationsTable(ctx, p); err != nil {
		return 0, err
	}
	var version int
	err := p.QueryRow(ctx, `SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&version)
	return version, err
}

// LoadMigrations reads migration files from the given directory.
// Files should be named: 0001_name.up.sql / 0001_name.down.sql
func LoadMigrations(migrationsDir string) ([]Migration, error) {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return nil, fmt.Errorf("read migrations dir: %w", err)
	}

	migrationMap := make(map[int]*Migration)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		name := entry.Name()
		var version int
		var migName string
		var isUp bool

		if strings.HasSuffix(name, ".up.sql") {
			isUp = true
			base := strings.TrimSuffix(name, ".up.sql")
			parts := strings.SplitN(base, "_", 2)
			if len(parts) < 2 {
				continue
			}
			fmt.Sscanf(parts[0], "%d", &version)
			migName = parts[1]
		} else if strings.HasSuffix(name, ".down.sql") {
			base := strings.TrimSuffix(name, ".down.sql")
			parts := strings.SplitN(base, "_", 2)
			if len(parts) < 2 {
				continue
			}
			fmt.Sscanf(parts[0], "%d", &version)
			migName = parts[1]
		} else {
			continue
		}

		mig, ok := migrationMap[version]
		if !ok {
			mig = &Migration{Version: version, Name: migName}
			migrationMap[version] = mig
		}
		fullPath := filepath.Join(migrationsDir, name)
		if isUp {
			mig.UpFile = fullPath
		} else {
			mig.DownFile = fullPath
		}
	}

	var migrations []Migration
	for _, m := range migrationMap {
		migrations = append(migrations, *m)
	}
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})
	return migrations, nil
}

// MigrateUp executes all pending migrations.
// Returns the number of migrations applied.
func MigrateUp(ctx context.Context, p Pool, migrationsDir string) (int, error) {
	if err := EnsureMigrationsTable(ctx, p); err != nil {
		return 0, err
	}

	currentVersion, err := GetMigrationVersion(ctx, p)
	if err != nil {
		return 0, err
	}

	migrations, err := LoadMigrations(migrationsDir)
	if err != nil {
		return 0, err
	}

	applied := 0
	for _, mig := range migrations {
		if mig.Version <= currentVersion {
			continue
		}
		if mig.UpFile == "" {
			continue
		}

		sql, err := os.ReadFile(mig.UpFile)
		if err != nil {
			return applied, fmt.Errorf("read migration %d: %w", mig.Version, err)
		}

		// Execute the migration SQL
		_, err = p.Exec(ctx, string(sql))
		if err != nil {
			return applied, fmt.Errorf("execute migration %d (%s): %w", mig.Version, mig.Name, err)
		}

		// Record the migration
		_, err = p.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, mig.Version)
		if err != nil {
			return applied, fmt.Errorf("record migration %d: %w", mig.Version, err)
		}

		applied++
		currentVersion = mig.Version
		fmt.Printf("  ✓ migration %04d_%s applied\n", mig.Version, mig.Name)
	}

	return applied, nil
}

// MigrateDown rolls back the last N migrations.
// If steps is 0, rolls back all migrations.
func MigrateDown(ctx context.Context, p Pool, migrationsDir string, steps int) (int, error) {
	if err := EnsureMigrationsTable(ctx, p); err != nil {
		return 0, err
	}

	currentVersion, err := GetMigrationVersion(ctx, p)
	if err != nil {
		return 0, err
	}
	if currentVersion == 0 {
		return 0, nil
	}

	migrations, err := LoadMigrations(migrationsDir)
	if err != nil {
		return 0, err
	}

	// Sort descending for rollback
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version > migrations[j].Version
	})

	rolledBack := 0
	for _, mig := range migrations {
		if mig.Version > currentVersion {
			continue
		}
		if steps > 0 && rolledBack >= steps {
			break
		}
		if mig.DownFile == "" {
			continue
		}

		sql, err := os.ReadFile(mig.DownFile)
		if err != nil {
			return rolledBack, fmt.Errorf("read down migration %d: %w", mig.Version, err)
		}

		_, err = p.Exec(ctx, string(sql))
		if err != nil {
			return rolledBack, fmt.Errorf("execute down migration %d: %w", mig.Version, err)
		}

		_, err = p.Exec(ctx, `DELETE FROM schema_migrations WHERE version = $1`, mig.Version)
		if err != nil {
			return rolledBack, fmt.Errorf("remove migration record %d: %w", mig.Version, err)
		}

		rolledBack++
		fmt.Printf("  ✓ migration %04d_%s rolled back\n", mig.Version, mig.Name)
	}

	return rolledBack, nil
}

// MigrateStatus returns the status of all migrations.
func MigrateStatus(ctx context.Context, p Pool, migrationsDir string) ([]MigrationStatus, error) {
	if err := EnsureMigrationsTable(ctx, p); err != nil {
		return nil, err
	}

	migrations, err := LoadMigrations(migrationsDir)
	if err != nil {
		return nil, err
	}

	// Get applied versions
	appliedMap := make(map[int]time.Time)
	rows, err := p.Query(ctx, `SELECT version, applied_at FROM schema_migrations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var version int
		var appliedAt time.Time
		if err := rows.Scan(&version, &appliedAt); err != nil {
			return nil, err
		}
		appliedMap[version] = appliedAt
	}

	var statuses []MigrationStatus
	for _, mig := range migrations {
		status := MigrationStatus{
			Version: mig.Version,
			Name:    mig.Name,
		}
		if appliedAt, ok := appliedMap[mig.Version]; ok {
			status.Applied = true
			status.AppliedAt = appliedAt
		}
		statuses = append(statuses, status)
	}
	return statuses, nil
}
