# GGID Docker Compose — Production Hardening Guide

This guide describes how to harden the Docker Compose deployment for production use.

---

## 1. Secrets Management

### 1.1 Use Docker Secrets (not plaintext env vars)

Create a secrets file:

```yaml
# deploy/docker-compose.prod.yaml (overlay)
services:
  postgres:
    environment:
      POSTGRES_PASSWORD_FILE: /run/secrets/db_password
    secrets:
      - db_password

  redis:
    command: ["redis-server", "--requirepass", "$(cat /run/secrets/redis_password)"]
    secrets:
      - redis_password

  gateway:
    environment:
      JWT_SIGNING_KEY_FILE: /run/secrets/jwt_signing_key
    secrets:
      - jwt_signing_key

secrets:
  db_password:
    file: ./secrets/db_password.txt
  redis_password:
    file: ./secrets/redis_password.txt
  jwt_signing_key:
    file: ./secrets/jwt_signing_key.pem
```

### 1.2 Generate Strong Secrets

```bash
# Database password
openssl rand -base64 32 > deploy/secrets/db_password.txt

# Redis password
openssl rand -base64 32 > deploy/secrets/redis_password.txt

# JWT signing key (RSA 2048)
openssl genrsa -out deploy/secrets/jwt_signing_key.pem 2048
openssl rsa -in deploy/secrets/jwt_signing_key.pem -pubout -out deploy/secrets/jwt_public_key.pem
```

---

## 2. Network Isolation

### 2.1 Separate Networks

```yaml
networks:
  frontend:
    driver: bridge
  backend:
    driver: bridge
    internal: true  # No external access

services:
  gateway:
    networks:
      - frontend
      - backend

  identity:
    networks:
      - backend  # Not accessible from outside

  auth:
    networks:
      - backend

  postgres:
    networks:
      - backend

  redis:
    networks:
      - backend
```

### 2.2 Remove Exposed Ports for Internal Services

In production, only the gateway should expose ports:

```yaml
# Production: only expose gateway + console
services:
  gateway:
    ports:
      - "443:8080"  # TLS terminates at load balancer

  console:
    ports:
      - "3000:3000"

  # All other services: NO ports exposed
  postgres:
    # ports: REMOVED — only accessible within backend network

  redis:
    # ports: REMOVED

  nats:
    # ports: REMOVED
```

---

## 3. TLS Configuration

### 3.1 Terminate TLS at Gateway

```yaml
services:
  gateway:
    environment:
      TLS_ENABLED: "true"
      TLS_CERT_FILE: /certs/fullchain.pem
      TLS_KEY_FILE: /certs/privkey.pem
    volumes:
      - ./certs:/certs:ro
    ports:
      - "443:8080"
      - "80:8080"  # Redirect to HTTPS
```

### 3.2 Using Let's Encrypt with Caddy

```yaml
services:
  caddy:
    image: caddy:2-alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile
      - caddy_data:/data
    networks:
      - frontend

  gateway:
    # No ports exposed — only accessible via Caddy
    networks:
      - frontend
      - backend
```

Caddyfile:
```
iam.example.com {
    reverse_proxy gateway:8080
    encode gzip zstd
    header {
        Strict-Transport-Security "max-age=63072000"
        X-Content-Type-Options nosniff
        X-Frame-Options DENY
    }
}
```

---

## 4. Resource Limits

```yaml
services:
  gateway:
    deploy:
      resources:
        limits:
          cpus: "2.0"
          memory: 512M
        reservations:
          cpus: "0.5"
          memory: 128M

  postgres:
    deploy:
      resources:
        limits:
          cpus: "4.0"
          memory: 2G
        reservations:
          cpus: "1.0"
          memory: 512M

  redis:
    deploy:
      resources:
        limits:
          cpus: "1.0"
          memory: 512M
```

---

## 5. Persistent Volumes

```yaml
volumes:
  ggid-pgdata:
    driver: local
    driver_opts:
      type: none
      o: bind
      device: /data/ggid/postgres  # Mounted from dedicated disk

  ggid-nats:
    driver: local
    driver_opts:
      type: none
      o: bind
      device: /data/ggid/nats
```

---

## 6. Logging

```yaml
services:
  gateway:
    logging:
      driver: json-file
      options:
        max-size: "10m"
        max-file: "5"

  postgres:
    logging:
      driver: json-file
      options:
        max-size: "10m"
        max-file: "10"
```

Or use a central log collector:

```yaml
services:
  gateway:
    logging:
      driver: fluentd
      options:
        fluentd-address: localhost:24224
        tag: ggid.gateway
```

---

## 7. Health Check Tuning

```yaml
services:
  gateway:
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost:8080/healthz"]
      interval: 10s
      timeout: 5s
      retries: 3
      start_period: 30s

  postgres:
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ggid -d ggid"]
      interval: 5s
      timeout: 3s
      retries: 5
      start_period: 10s
```

---

## 8. Backup Strategy

### 8.1 Database Backup Cron

```yaml
services:
  db-backup:
    image: postgres:16-alpine
    environment:
      PGHOST: postgres
      PGUSER: ggid
      PGDATABASE: ggid
      PGPASSWORD_FILE: /run/secrets/db_password
    volumes:
      - ./backups:/backups
    secrets:
      - db_password
    entrypoint: |
      sh -c '
        while true; do
          pg_dump | gzip > /backups/ggid-$$(date +%Y%m%d-%H%M%S).sql.gz
          find /backups -name "*.sql.gz" -mtime +30 -delete
          sleep 86400
        done
      '
    networks:
      - backend
```

### 8.2 Redis Persistence

```yaml
services:
  redis:
    command: ["redis-server", "--appendonly", "yes", "--save", "60", "1000"]
    volumes:
      - ggid-redis-data:/data
```

---

## 9. Production Deployment Commands

```bash
# Deploy with production overlay
docker compose -f docker-compose.yaml -f docker-compose.prod.yaml up -d

# View logs
docker compose logs -f gateway

# Scale gateway
docker compose up -d --scale gateway=3

# Rolling restart
docker compose up -d --no-deps --build gateway

# Backup database
docker compose exec postgres pg_dump -U ggid ggid | gzip > backup.sql.gz

# Restore database
gunzip -c backup.sql.gz | docker compose exec -T postgres psql -U ggid ggid
```

---

## 10. Production Checklist

- [ ] All passwords replaced with Docker secrets
- [ ] Internal services have no exposed ports
- [ ] Networks separated (frontend + backend)
- [ ] TLS configured (certificates + HSTS)
- [ ] Resource limits set for all services
- [ ] Persistent volumes on dedicated disk
- [ ] Log rotation configured
- [ ] Health check start_period tuned
- [ ] Database backup cron running
- [ ] Redis persistence enabled
- [ ] Docker images scanned (Trivy)
- [ ] No debug environment variables in production
