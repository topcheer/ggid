// Command ggid-migrate is a CLI tool for database migration, backup, and restore.
//
// Usage:
//   ggid-migrate up                  — Execute pending migrations
//   ggid-migrate down [N]            — Roll back N migrations (default: 1)
//   ggid-migrate status              — Show migration status
//   ggid-migrate backup [file]       — Backup database to JSON file
//   ggid-migrate restore [file]      — Restore database from JSON file
//   ggid-migrate export [table] [file] — Export single table
//   ggid-migrate import [table] [file] — Import single table
//
// Environment:
//   DB_DRIVER   — postgres (default), mysql, or sqlite
//   DATABASE_URL — connection string
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ggid/ggid/pkg/db"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "up":
		runMigrateUp(args)
	case "down":
		runMigrateDown(args)
	case "status":
		runMigrateStatus(args)
	case "backup":
		runBackup(args)
	case "restore":
		runRestore(args)
	case "export":
		runExport(args)
	case "import":
		runImport(args)
	case "version":
		runVersion(args)
	case "-h", "--help", "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`ggid-migrate — Database migration and backup tool

Usage:
  ggid-migrate up                   Execute pending migrations
  ggid-migrate down [N]             Roll back N migrations (default: 1)
  ggid-migrate status               Show migration status
  ggid-migrate backup [file]        Backup database to JSON file
  ggid-migrate restore [file]       Restore database from JSON file
  ggid-migrate export [table] [file]  Export single table
  ggid-migrate import [table] [file]  Import single table
  ggid-migrate version              Show current migration version

Environment:
  DB_DRIVER    postgres (default), mysql, or sqlite
  DATABASE_URL Connection string
  MIGRATIONS_DIR  Path to migration files (default: migrations)`)
}

func getPool() db.Pool {
	p, err := db.NewPool("", "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Database connection failed: %v\n", err)
		os.Exit(1)
	}
	return p
}

func getMigrationsDir() string {
	dir := os.Getenv("MIGRATIONS_DIR")
	if dir == "" {
		dir = "migrations"
	}
	return dir
}

func runMigrateUp(_ []string) {
	p := getPool()
	defer p.Close()
	dir := getMigrationsDir()

	fmt.Printf("Running migrations from %s...\n", dir)
	ctx := context.Background()
	applied, err := db.MigrateUp(ctx, p, dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Migration failed: %v\n", err)
		os.Exit(1)
	}
	if applied == 0 {
		fmt.Println("No pending migrations — database is up to date.")
	} else {
		fmt.Printf("Successfully applied %d migration(s).\n", applied)
	}
}

func runMigrateDown(args []string) {
	steps := 1
	if len(args) > 0 {
		fmt.Sscanf(args[0], "%d", &steps)
	}

	p := getPool()
	defer p.Close()
	dir := getMigrationsDir()

	ctx := context.Background()
	rolledBack, err := db.MigrateDown(ctx, p, dir, steps)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Rollback failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Rolled back %d migration(s).\n", rolledBack)
}

func runMigrateStatus(_ []string) {
	p := getPool()
	defer p.Close()
	dir := getMigrationsDir()

	ctx := context.Background()
	statuses, err := db.MigrateStatus(ctx, p, dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Status check failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%-8s %-30s %-8s %s\n", "Version", "Name", "Status", "Applied At")
	fmt.Println(strings.Repeat("-", 80))
	for _, s := range statuses {
		status := "pending"
		appliedAt := ""
		if s.Applied {
			status = "applied"
			appliedAt = s.AppliedAt.Format(time.RFC3339)
		}
		fmt.Printf("%04d     %-30s %-8s %s\n", s.Version, s.Name, status, appliedAt)
	}
}

func runBackup(args []string) {
	outputFile := fmt.Sprintf("backup-%s.json", time.Now().Format("20060102-150405"))
	if len(args) > 0 {
		outputFile = args[0]
	}

	p := getPool()
	defer p.Close()

	f, err := os.Create(outputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Create file failed: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	ctx := context.Background()
	if err := db.ExportDatabase(ctx, p, db.FormatJSON, f); err != nil {
		fmt.Fprintf(os.Stderr, "Export failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Database backed up to %s\n", outputFile)
}

func runRestore(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: ggid-migrate restore <file>")
		os.Exit(1)
	}
	inputFile := args[0]

	data, err := os.ReadFile(inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Read file failed: %v\n", err)
		os.Exit(1)
	}

	var export db.DatabaseExport
	if err := json.Unmarshal(data, &export); err != nil {
		fmt.Fprintf(os.Stderr, "Parse JSON failed: %v\n", err)
		os.Exit(1)
	}

	p := getPool()
	defer p.Close()

	ctx := context.Background()
	if err := db.ImportDatabase(ctx, p, &export); err != nil {
		fmt.Fprintf(os.Stderr, "Import failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Database restored from %s (%d tables)\n", inputFile, len(export.Tables))
}

func runExport(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: ggid-migrate export <table> <file>")
		os.Exit(1)
	}
	tableName := args[0]
	outputFile := args[1]

	p := getPool()
	defer p.Close()

	ctx := context.Background()
	td, err := db.BackupTable(ctx, p, tableName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Export table failed: %v\n", err)
		os.Exit(1)
	}

	f, err := os.Create(outputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Create file failed: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	encoder.Encode(td)
	fmt.Printf("Table '%s' exported to %s (%d rows)\n", tableName, outputFile, td.Count)
}

func runImport(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: ggid-migrate import <table> <file>")
		os.Exit(1)
	}
	tableName := args[0]
	inputFile := args[1]

	data, err := os.ReadFile(inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Read file failed: %v\n", err)
		os.Exit(1)
	}

	var td db.TableData
	if err := json.Unmarshal(data, &td); err != nil {
		fmt.Fprintf(os.Stderr, "Parse JSON failed: %v\n", err)
		os.Exit(1)
	}

	p := getPool()
	defer p.Close()

	ctx := context.Background()
	if err := db.RestoreTable(ctx, p, tableName, &td); err != nil {
		fmt.Fprintf(os.Stderr, "Import table failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Table '%s' imported from %s (%d rows)\n", tableName, inputFile, td.Count)
}

func runVersion(_ []string) {
	p := getPool()
	defer p.Close()

	ctx := context.Background()
	version, err := db.GetMigrationVersion(ctx, p)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Get version failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Current migration version: %d\n", version)
}
