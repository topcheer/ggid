# Database Setup Guide

This guide covers installation and configuration for the three databases supported by GGID: PostgreSQL (production), MySQL (enterprise compatibility), and SQLite (development/testing).

## Overview

| Database | Use Case | Status | Driver |
|----------|----------|--------|--------|
| PostgreSQL 16+ | Production (recommended) | Full support | pgx/v5 |
| MySQL 8.0+ | Enterprise compatibility | Experimental | go-sql-driver/mysql |
| SQLite 3.40+ | Development & testing | Experimental | modernc.org/sqlite |

## PostgreSQL (Production Recommended)

PostgreSQL is the primary and fully supported database for GGID. All features including Row-Level Security (RLS), JSONB columns, and full-text search are PostgreSQL-specific.

### Installation

#### Docker (Recommended)

```bash
docker run -d \
  --name ggid-postgres \
  -e POSTGRES_USER=ggid \
  -e POSTGRES_PASSWORD=secure-password \
  -e POSTGRES_DB=ggid \
  -p 5432:5432 \
  -v ggid-pgdata:/var/lib/postgresql/data \
  postgres:16
```

#### Native Installation

```bash
# macOS
brew install postgresql@16
brew services start postgresql@16

# Ubuntu/Debian
sudo apt install -y postgresql-16
sudo systemctl enable postgresql
sudo systemctl start postgresql

# Create database and user
sudo -u postgres createuser -s ggid
sudo -u postgres psql -c "ALTER USER ggid PASSWORD 'secure-password';"
sudo -u postgres createdb -O ggid ggid
```

### Configuration

GGID services use two patterns for PostgreSQL configuration:

#### Pattern 1: DATABASE_URL (Auth, Identity, OAuth services)

```bash
# .env
DATABASE_URL=postgres://ggid:secure-password@localhost:5432/ggid?sslmode=disable
```

With SSL (production):

```bash
DATABASE_URL=postgres://ggid:secure-password@db.example.com:5432/ggid?sslmode=require&sslrootcert=/etc/ssl/certs/ca-cert.pem
```

#### Pattern 2: Individual DB vars (Policy, Org, Audit services)

```bash
# .env
DB_HOST=localhost
DB_PORT=5432
DB_USER=ggid
DB_PASSWORD=secure-password
DB_DATABASE=ggid
DB_SSL_MODE=disable
DB_MAX_CONNS=20
DB_MIN_CONNS=2
DB_CONN_LIFETIME=300
```

### SSL Modes

| Value | Description |
|-------|-------------|
| `disable` | No SSL (development only) |
| `prefer` | SSL if available, otherwise plaintext |
| `require` | SSL required, no certificate verification |
| `verify-ca` | SSL required, verify CA certificate |
| `verify-full` | SSL required, verify CA and hostname (recommended for production) |

### Docker Compose

```yaml
services:
  postgres:
    image: postgres:16
    environment:
      POSTGRES_USER: ggid
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
      POSTGRES_DB: ggid
    ports:
      - "5432:5432"
    volumes:
      - ggid-pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ggid"]
      interval: 5s
      timeout: 5s
      retries: 5

  auth:
    environment:
      DATABASE_URL: "postgres://ggid:${POSTGRES_PASSWORD}@postgres:5432/ggid?sslmode=disable"

  policy:
    environment:
      DB_HOST: "postgres"
      DB_PORT: "5432"
      DB_USER: "ggid"
      DB_PASSWORD: "${POSTGRES_PASSWORD}"
      DB_DATABASE: "ggid"
```

### Migrations

```bash
# Run all migrations
psql "$DATABASE_URL" -f migrations/001_init.sql
psql "$DATABASE_URL" -f migrations/002_rls.sql
psql "$DATABASE_URL" -f migrations/003_audit.sql

# Or via Docker
docker exec -i ggid-postgres psql -U ggid -d ggid < migrations/001_init.sql
```

### Production Tuning

See [Multi-Database Guide](multi-database.md) for PostgreSQL tuning, RLS configuration, and read replica setup.

## MySQL (Enterprise Compatibility)

MySQL 8.0+ is supported for enterprise environments where MySQL is the standard.

### Installation

#### Docker

```bash
docker run -d \
  --name ggid-mysql \
  -e MYSQL_ROOT_PASSWORD=root-password \
  -e MYSQL_DATABASE=ggid \
  -e MYSQL_USER=ggid \
  -e MYSQL_PASSWORD=secure-password \
  -p 3306:3306 \
  -v ggid-mysqldata:/var/lib/mysql \
  mysql:8.0
```

#### Native

```bash
# macOS
brew install mysql
brew services start mysql

# Ubuntu/Debian
sudo apt install -y mysql-server
sudo systemctl enable mysql
sudo systemctl start mysql

# Create database and user
mysql -u root -p -e "
  CREATE DATABASE ggid CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
  CREATE USER 'ggid'@'%' IDENTIFIED BY 'secure-password';
  GRANT ALL PRIVILEGES ON ggid.* TO 'ggid'@'%';
  FLUSH PRIVILEGES;
"
```

### Configuration

```bash
# .env
DB_DRIVER=mysql
DB_HOST=localhost
DB_PORT=3306
DB_USER=ggid
DB_PASSWORD=secure-password
DB_DATABASE=ggid

# Or via connection string
DATABASE_URL="ggid:secure-password@tcp(localhost:3306)/ggid?charset=utf8mb4&parseTime=true&loc=Local"
```

### Docker Compose

```yaml
services:
  mysql:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: ${MYSQL_ROOT_PASSWORD}
      MYSQL_DATABASE: ggid
      MYSQL_USER: ggid
      MYSQL_PASSWORD: ${MYSQL_PASSWORD}
    ports:
      - "3306:3306"
    volumes:
      - ggid-mysqldata:/var/lib/mysql
    command: --character-set-server=utf8mb4 --collation-server=utf8mb4_unicode_ci
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost"]
      interval: 5s
      timeout: 5s
      retries: 10
```

### Migrations

```bash
mysql -u ggid -p ggid < migrations/001_init_mysql.sql
```

### Limitations

- Row-Level Security (RLS) is not available in MySQL. Tenant isolation is enforced at the application layer.
- JSONB columns use MySQL JSON type (less efficient indexing).
- Full-text search uses MySQL FULLTEXT indexes instead of PostgreSQL tsvector.
- Some migration scripts may need manual adjustment for MySQL syntax differences.

## SQLite (Development & Testing)

SQLite is ideal for local development, CI/CD pipelines, and unit tests. No server installation required.

### Installation

No installation needed — SQLite is embedded in the Go binary via `modernc.org/sqlite` (pure Go, no CGO).

### Configuration

```bash
# .env
DB_DRIVER=sqlite
DB_DATABASE=/var/lib/ggid/ggid.db

# Or via connection string
DATABASE_URL="file:/var/lib/ggid/ggid.db?cache=shared&_journal_mode=WAL&_busy_timeout=5000"
```

### Usage in Tests

```go
import _ "modernc.org/sqlite"

db, err := sql.Open("sqlite", ":memory:")
// or
db, err := sql.Open("sqlite", "file:test.db?cache=shared&_journal_mode=WAL")
```

### Docker Compose (Development)

```yaml
services:
  auth:
    environment:
      DB_DRIVER: sqlite
      DB_DATABASE: /data/ggid.db
    volumes:
      - ggid-sqlite:/data
```

### Migrations

```bash
sqlite3 /var/lib/ggid/ggid.db < migrations/001_init_sqlite.sql
```

### Limitations

- No concurrent writes (WAL mode helps but does not fully resolve).
- No RLS — tenant isolation enforced at application layer only.
- Limited full-text search capability (FTS5 extension).
- Not suitable for production multi-service deployments.
- No built-in replication or failover.

## Environment Variable Reference

### Common Variables (All Databases)

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_DRIVER` | `postgres` | Database driver: `postgres`, `mysql`, or `sqlite` |
| `DATABASE_URL` | — | Full connection URL (takes precedence over individual vars) |
| `DB_HOST` | `localhost` | Database host |
| `DB_PORT` | `5432` | Database port (3306 for MySQL) |
| `DB_USER` | `ggid` | Database username |
| `DB_PASSWORD` | — | Database password |
| `DB_DATABASE` | `ggid` | Database name |
| `DB_SSL_MODE` | `disable` | SSL mode (PostgreSQL only) |
| `DB_MAX_CONNS` | `20` | Maximum connections in pool |
| `DB_MIN_CONNS` | `2` | Minimum connections in pool |
| `DB_CONN_LIFETIME` | `300` | Connection lifetime in seconds |

### Per-Service Configuration

Each microservice can use independent database credentials:

```bash
# Auth service
DATABASE_URL=postgres://ggid_auth:auth_pass@db:5432/ggid_auth

# Identity service
DATABASE_URL=postgres://ggid_identity:identity_pass@db:5432/ggid_identity

# Policy service (uses individual vars)
DB_HOST=db
DB_USER=ggid_policy
DB_PASSWORD=policy_pass
DB_DATABASE=ggid_policy
```

## Migration Strategy

### From PostgreSQL to MySQL

1. Export schema: `pg_dump --schema-only $PG_URL > schema.sql`
2. Convert syntax: Replace `SERIAL` → `AUTO_INCREMENT`, `JSONB` → `JSON`, `BOOLEAN` → `TINYINT(1)`
3. Remove RLS policies (enforce tenant isolation in application layer)
4. Import: `mysql -u ggid -p ggid < schema_mysql.sql`

### From PostgreSQL to SQLite

1. Export schema: `pg_dump --schema-only $PG_URL > schema.sql`
2. Convert types: `SERIAL` → `INTEGER PRIMARY KEY AUTOINCREMENT`, `JSONB` → `TEXT`
3. Remove RLS policies and indexes not supported by SQLite
4. Import: `sqlite3 ggid.db < schema_sqlite.sql`

## Troubleshooting

### Connection Refused

```bash
# Check if database is running
docker ps | grep postgres
pg_isready -h localhost -p 5432

# Check credentials
psql "postgres://ggid:password@localhost:5432/ggid" -c "SELECT 1;"
```

### SSL Certificate Errors

```bash
# For development, use sslmode=disable
DATABASE_URL="postgres://ggid:pass@localhost:5432/ggid?sslmode=disable"

# For production with self-signed certs
DATABASE_URL="postgres://ggid:pass@db:5432/ggid?sslmode=require"
```

### Migration Failures

```bash
# Check current migration state
psql "$DATABASE_URL" -c "SELECT * FROM schema_migrations;"

# Reset (development only)
psql "$DATABASE_URL" -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
psql "$DATABASE_URL" -f migrations/001_init.sql
```

## See Also

- [Multi-Database Guide](multi-database.md) — PostgreSQL tuning and read replicas
- [Database Security](database-security.md) — Encryption, RLS, and audit
- [Database Migration Playbook](database-migration-playbook.md) — Zero-downtime migrations
- [Architecture Overview](../architecture/overview.md) — Data layer architecture
