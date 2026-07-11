# Docker Compose Override Guide

> Use `docker-compose.override.yml` to customize GGID for local development without modifying the base compose file.

---

## How Overrides Work

Docker Compose automatically merges `docker-compose.override.yml` with `docker-compose.yaml`. The override file is NOT committed — it's gitignored for personal customizations.

```bash
# Base file:       deploy/docker-compose.yaml     (committed)
# Override file:   deploy/docker-compose.override.yml (gitignored, personal)
# Production:      deploy/docker-compose.prod.yaml (committed, production)
```

Merge priority: `override` > `prod` > `base`

---

## Common Use Cases

### 1. Enable Debug Logging

```yaml
# docker-compose.override.yml
services:
  gateway:
    environment:
      LOG_LEVEL: debug

  auth:
    environment:
      LOG_LEVEL: debug
```

### 2. Add Delve Debug Port (Go)

```yaml
services:
  auth:
    ports:
      - "40000:40000"
    environment:
      CGO_ENABLED: 0
      DELVE_PORT: 40000
```

### 3. Use Local Images (Skip Registry)

```yaml
services:
  gateway:
    image: ggid-gateway:local
    build:
      context: ..
      dockerfile: services/gateway/Dockerfile

  auth:
    image: ggid-auth:local
    build:
      context: ..
      dockerfile: services/auth/Dockerfile
```

### 4. Mount Source for Hot Reload

```yaml
services:
  gateway:
    volumes:
      - ../:/app
    working_dir: /app
    command: go run ./services/gateway/cmd/main.go

  auth:
    volumes:
      - ../:/app
    working_dir: /app
    command: go run ./services/auth/cmd/main.go
```

### 5. Disable LDAP (Not Needed for Testing)

```yaml
services:
  ldap:
    profiles:
      - ldap  # Only starts with: docker compose --profile ldap up
  ldap-seed:
    profiles:
      - ldap
  auth:
    environment:
      LDAP_URL: ""  # Disable LDAP provider
```

### 6. Use External Database

```yaml
services:
  postgres:
    image: postgres:16-alpine
    ports: []  # Don't expose port

  gateway:
    environment:
      DATABASE_URL: "postgres://ggid:ggid@external-db.internal:5432/ggid?sslmode=disable"
```

### 7. Custom JWT Secret

```yaml
services:
  gateway:
    environment:
      JWT_SECRET: "my-dev-secret"
  auth:
    environment:
      JWT_SECRET: "my-dev-secret"
  oauth:
    environment:
      JWT_SECRET: "my-dev-secret"
```

### 8. Console with Custom API URL

```yaml
services:
  console:
    environment:
      NEXT_PUBLIC_GGID_URL: "http://localhost:8080"
      NODE_ENV: development
    volumes:
      - ../console:/app
      - /app/node_modules
    command: npm run dev
```

---

## Using Multiple Override Files

```bash
# Explicit file specification
docker compose \
  -f docker-compose.yaml \
  -f docker-compose.override.yml \
  -f docker-compose.debug.yml \
  up -d
```

---

## Best Practices

1. **Never commit secrets** — Override file is gitignored
2. **Keep base file clean** — All customizations go in override
3. **Document your overrides** — Add comments explaining why
4. **Share team overrides** via a committed `docker-compose.team.yml`

---

*Last updated: 2025-07-11*