package db

import (
	"context"
	"testing"
)

func TestNewPool_UnsupportedDriver(t *testing.T) {
	_, err := NewPool("oracle", "oracle://localhost:1521/db")
	if err == nil {
		t.Fatal("expected error for unsupported driver")
	}
}

func TestNewPool_EmptyURL(t *testing.T) {
	// Clear env vars
	t.Setenv("DATABASE_URL", "")
	t.Setenv("DB_DRIVER", "")
	_, err := NewPool("", "")
	if err == nil {
		t.Fatal("expected error for empty URL")
	}
}

func TestNewPool_DefaultDriver(t *testing.T) {
	// When DB_DRIVER is not set, should default to postgres
	t.Setenv("DB_DRIVER", "")
	t.Setenv("DATABASE_URL", "")
	_, err := NewPool("", "")
	// Should fail because URL is empty, not because driver is unknown
	if err == nil {
		t.Fatal("expected error for empty URL")
	}
	// Error should not mention unsupported driver
	if err != nil && err.Error() == "unsupported database driver: postgres" {
		t.Fatal("postgres should be the default driver, not unsupported")
	}
}

func TestDriver_Constants(t *testing.T) {
	if DriverPostgres != "postgres" {
		t.Errorf("DriverPostgres = %s, want postgres", DriverPostgres)
	}
	if DriverMySQL != "mysql" {
		t.Errorf("DriverMySQL = %s, want mysql", DriverMySQL)
	}
	if DriverSQLite != "sqlite" {
		t.Errorf("DriverSQLite = %s, want sqlite", DriverSQLite)
	}
}

func TestIsPostgres(t *testing.T) {
	if !IsPostgres(&pgxPool{}) {
		t.Error("pgxPool should be postgres")
	}
	if IsPostgres(&sqlPool{driver: DriverMySQL}) {
		t.Error("mysql pool should not be postgres")
	}
}

func TestIsMySQL(t *testing.T) {
	if !IsMySQL(&sqlPool{driver: DriverMySQL}) {
		t.Error("mysql pool should be mysql")
	}
	if IsMySQL(&pgxPool{}) {
		t.Error("pgxPool should not be mysql")
	}
}

func TestIsSQLite(t *testing.T) {
	if !IsSQLite(&sqlPool{driver: DriverSQLite}) {
		t.Error("sqlite pool should be sqlite")
	}
	if IsSQLite(&pgxPool{}) {
		t.Error("pgxPool should not be sqlite")
	}
}

func TestSQLiteInMemory(t *testing.T) {
	// Test SQLite in-memory mode works
	pool, err := newSQLitePool(":memory:")
	if err != nil {
		t.Fatalf("sqlite open: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()
	// Create a test table
	_, err = pool.Exec(ctx, "CREATE TABLE test (id INTEGER PRIMARY KEY, name TEXT)")
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	// Insert a row
	_, err = pool.Exec(ctx, "INSERT INTO test (id, name) VALUES (?, ?)", 1, "hello")
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	// Query it back
	row := pool.QueryRow(ctx, "SELECT name FROM test WHERE id = ?", 1)
	var name string
	if err := row.Scan(&name); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if name != "hello" {
		t.Errorf("got %s, want hello", name)
	}

	// Verify driver
	if !IsSQLite(pool) {
		t.Error("pool should be SQLite")
	}
}

func TestSQLiteQueryMultipleRows(t *testing.T) {
	pool, err := newSQLitePool(":memory:")
	if err != nil {
		t.Fatalf("sqlite open: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()
	pool.Exec(ctx, "CREATE TABLE items (id INTEGER PRIMARY KEY, val TEXT)")
	pool.Exec(ctx, "INSERT INTO items (val) VALUES (?), (?), (?)", "a", "b", "c")

	rows, err := pool.Query(ctx, "SELECT val FROM items ORDER BY id")
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer rows.Close()

	var results []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			t.Fatalf("scan: %v", err)
		}
		results = append(results, v)
	}
	if len(results) != 3 {
		t.Errorf("got %d rows, want 3", len(results))
	}
	if results[0] != "a" || results[1] != "b" || results[2] != "c" {
		t.Errorf("results = %v, want [a b c]", results)
	}
}

func TestPoolInterface(t *testing.T) {
	// Verify all pool types implement the Pool interface
	var _ Pool = (*pgxPool)(nil)
	var _ Pool = (*sqlPool)(nil)
	var _ Rows = (*pgxRows)(nil)
	var _ Rows = (*sqlRows)(nil)
	var _ Row = (*pgxRow)(nil)
	var _ Row = (*sqlRow)(nil)
}
