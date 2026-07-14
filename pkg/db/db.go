// Package db provides a database abstraction layer supporting PostgreSQL,
// MySQL, and SQLite. It offers a unified Pool interface and factory function
// so services can switch databases via DB_DRIVER env var.
//
// Supported drivers:
//   - postgres (default, recommended for production) — uses pgx/v5
//   - mysql (enterprise compatibility) — uses go-sql-driver/mysql + database/sql
//   - sqlite (development and testing) — uses modernc.org/sqlite + database/sql
//
// Usage:
//
//	pool, err := db.NewPool("postgres", "postgresql://user:pass@host:5432/dbname")
//	// or via env: DB_DRIVER=mysql DATABASE_URL=user:pass@tcp(host:3306)/dbname
//	pool.QueryRow(ctx, "SELECT 1").Scan(&n)
package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "modernc.org/sqlite" // register sqlite driver
	_ "github.com/go-sql-driver/mysql" // register mysql driver
)

// DBDriver is the type alias for database driver names.
type DBDriver string

const (
	DriverPostgres DBDriver = "postgres"
	DriverMySQL    DBDriver = "mysql"
	DriverSQLite   DBDriver = "sqlite"
)

// Pool is the unified database interface that works across all drivers.
// For PostgreSQL, it wraps *pgxpool.Pool. For MySQL/SQLite, it wraps *sql.DB.
type Pool interface {
	// Query executes a query that returns rows.
	Query(ctx context.Context, sql string, args ...any) (Rows, error)
	// QueryRow executes a query that returns at most one row.
	QueryRow(ctx context.Context, sql string, args ...any) Row
	// Exec executes a query that doesn't return rows.
	Exec(ctx context.Context, sql string, args ...any) (CommandTag, error)
	// Close closes the database connection.
	Close()
	// Driver returns the active driver name.
	Driver() DBDriver
}

// Rows is the unified interface for query results.
type Rows interface {
	Next() bool
	Scan(dest ...any) error
	Close() error
	Err() error
}

// Row is the unified interface for single-row queries.
type Row interface {
	Scan(dest ...any) error
}

// CommandTag contains result metadata from Exec.
type CommandTag struct {
	RowsAffected int64
}

// pgxPool wraps pgxpool.Pool to implement the Pool interface.
type pgxPool struct {
	pool *pgxpool.Pool
}

func (p *pgxPool) Query(ctx context.Context, sql string, args ...any) (Rows, error) {
	rows, err := p.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	return &pgxRows{rows: rows}, nil
}

func (p *pgxPool) QueryRow(ctx context.Context, sql string, args ...any) Row {
	return &pgxRow{row: p.pool.QueryRow(ctx, sql, args...)}
}

func (p *pgxPool) Exec(ctx context.Context, sql string, args ...any) (CommandTag, error) {
	tag, err := p.pool.Exec(ctx, sql, args...)
	if err != nil {
		return CommandTag{}, err
	}
	return CommandTag{RowsAffected: tag.RowsAffected()}, nil
}

func (p *pgxPool) Close() { p.pool.Close() }
func (p *pgxPool) Driver() DBDriver { return DriverPostgres }

type pgxRows struct{ rows pgx.Rows }

func (r *pgxRows) Next() bool        { return r.rows.Next() }
func (r *pgxRows) Scan(d ...any) error { return r.rows.Scan(d...) }
func (r *pgxRows) Close() error      { r.rows.Close(); return nil }
func (r *pgxRows) Err() error        { return r.rows.Err() }

type pgxRow struct{ row pgx.Row }

func (r *pgxRow) Scan(d ...any) error { return r.row.Scan(d...) }

// sqlPool wraps database/sql.DB for MySQL and SQLite.
type sqlPool struct {
	db     *sql.DB
	driver DBDriver
}

func (p *sqlPool) Query(ctx context.Context, query string, args ...any) (Rows, error) {
	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &sqlRows{rows: rows}, nil
}

func (p *sqlPool) QueryRow(ctx context.Context, query string, args ...any) Row {
	return &sqlRow{row: p.db.QueryRowContext(ctx, query, args...)}
}

func (p *sqlPool) Exec(ctx context.Context, query string, args ...any) (CommandTag, error) {
	res, err := p.db.ExecContext(ctx, query, args...)
	if err != nil {
		return CommandTag{}, err
	}
	n, _ := res.RowsAffected()
	return CommandTag{RowsAffected: n}, nil
}

func (p *sqlPool) Close() { p.db.Close() }
func (p *sqlPool) Driver() DBDriver { return p.driver }

type sqlRows struct{ rows *sql.Rows }

func (r *sqlRows) Next() bool        { return r.rows.Next() }
func (r *sqlRows) Scan(d ...any) error { return r.rows.Scan(d...) }
func (r *sqlRows) Close() error      { return r.rows.Close() }
func (r *sqlRows) Err() error        { return r.rows.Err() }

type sqlRow struct{ row *sql.Row }

func (r *sqlRow) Scan(d ...any) error { return r.row.Scan(d...) }

// NewPool creates a new database Pool based on the driver name.
// If driver is empty, it reads DB_DRIVER from the environment (defaults to postgres).
// If url is empty, it reads DATABASE_URL from the environment.
func NewPool(driver, url string) (Pool, error) {
	if driver == "" {
		driver = os.Getenv("DB_DRIVER")
		if driver == "" {
			driver = string(DriverPostgres)
		}
	}
	if url == "" {
		url = os.Getenv("DATABASE_URL")
	}
	if url == "" {
		return nil, fmt.Errorf("database URL is required (set DATABASE_URL or pass url)")
	}

	switch DBDriver(driver) {
	case DriverPostgres:
		return newPostgresPool(url)
	case DriverMySQL:
		return newMySQLPool(url)
	case DriverSQLite:
		return newSQLitePool(url)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s (use postgres, mysql, or sqlite)", driver)
	}
}

var (
	pgInitOnce sync.Once
	pgInitErr  error
)

func newPostgresPool(url string) (Pool, error) {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("postgres connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("postgres ping: %w", err)
	}
	return &pgxPool{pool: pool}, nil
}

func newMySQLPool(dsn string) (Pool, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("mysql open: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("mysql ping: %w", err)
	}
	return &sqlPool{db: db, driver: DriverMySQL}, nil
}

func newSQLitePool(path string) (Pool, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("sqlite open: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("sqlite ping: %w", err)
	}
	return &sqlPool{db: db, driver: DriverSQLite}, nil
}

// FromPgxPool wraps an existing *pgxpool.Pool into a Pool interface.
// This is useful for services that already have a pgxpool and want to
// use the unified interface without reconnecting.
func FromPgxPool(pool *pgxpool.Pool) Pool {
	return &pgxPool{pool: pool}
}

// IsPostgres returns true if the pool is backed by PostgreSQL.
func IsPostgres(p Pool) bool { return p.Driver() == DriverPostgres }

// IsMySQL returns true if the pool is backed by MySQL.
func IsMySQL(p Pool) bool { return p.Driver() == DriverMySQL }

// IsSQLite returns true if the pool is backed by SQLite.
func IsSQLite(p Pool) bool { return p.Driver() == DriverSQLite }
