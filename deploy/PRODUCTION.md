# GGID Production Deployment Guide

This guide covers production deployment of the GGID IAM platform using Docker Compose
with hardened security settings.

## Quick Start

```bash
# 1. Copy environment template
cp .env.example .env

# 2. Edit .env with your secrets
# IMPORTANT: Change ALL passwords and secrets!
vi .env

# 3. Start production stack
docker compose -f docker-compose.prod.yaml --env-file .env up -d

# 4. Verify all services are healthy
docker compose -f docker-compose.prod.yaml ps

# 5. Run E2E tests
bash e2e-docker-test.sh
```

## Architecture

```
                    Internet
                       |
                   [Nginx TLS]
                   /         \
          [Gateway:8080]  [Console:3000]
              |
         [backend-net] (internal)
         /  |  |  |  |  \
    [Auth] [Identity] [Policy] [Org] [Audit] [OAuth]
              |
         [data-net] (internal)
         /  |  \
    [PostgreSQL] [Redis] [NATS]
```

### Network Isolation

| Network | Scope | Services |
|---------|-------|----------|
| `frontend-net` | External | Gateway, Console, Nginx |
| `backend-net` | Internal only | All microservices |
| `data-net` | Internal only | PostgreSQL, Redis, NATS |

Microservices **cannot** be accessed directly from the host. All traffic must
flow through the Gateway.

## Security Features

### 1. Authentication & Secrets

- **No default passwords** — `.env` file requires explicit configuration
- **Redis authentication** — password required (`--requirepass`)
- **JWT RSA 4096-bit keys** — generated on first start, stored in named volume
- **LDAP admin password** — configurable via environment

### 2. Network Security

- **3-tier network isolation** — frontend, backend, data
- **No host port exposure** for internal services
- Only Gateway (8080) and Console (3000) expose ports

### 3. Container Hardening

- **Read-only root filesystem** (`read_only: true`) for microservices
- **Resource limits** — memory and CPU caps per container
- **Log rotation** — 10MB max size, 3 files retained
- **Non-root execution** — containers run as non-root user where possible

### 4. TLS Termination (Optional)

Use the included Nginx configuration for TLS termination:

```bash
# 1. Obtain TLS certificates (Let's Encrypt or commercial CA)
cp fullchain.pem deploy/nginx/certs/
cp privkey.pem deploy/nginx/certs/

# 2. Add nginx service to your compose override
docker compose -f docker-compose.prod.yaml -f docker-compose.tls.yaml up -d
```

See `nginx/nginx.conf` for the full configuration with:
- HTTP → HTTPS redirect
- HSTS, CSP, X-Frame-Options headers
- Per-endpoint rate limiting (API: 10r/s, Auth: 5r/s, Token: 3r/s)
- Request body size limiting (10MB)

### 5. Data Persistence

| Volume | Purpose |
|--------|---------|
| `ggid-pgdata` | PostgreSQL data |
| `ggid-redis-data` | Redis append-only file |
| `ggid-nats-data` | NATS JetStream data |
| `ggid-configs` | RSA keys (JWT signing) |

### 6. Health Checks

All services have health checks with:
- 10s interval
- 5s timeout
- 3 retries
- 10-30s start period

## Resource Planning

### Minimum Requirements

| Resource | Minimum | Recommended |
|----------|---------|-------------|
| CPU | 2 cores | 4+ cores |
| RAM | 4 GB | 8+ GB |
| Disk | 20 GB | 50+ GB SSD |
| Network | 100 Mbps | 1 Gbps |

### Per-Service Resource Limits

| Service | Memory | CPU |
|---------|--------|-----|
| Gateway | 256 MB | 1.0 |
| Auth | 256 MB | 0.5 |
| Identity | 256 MB | 0.5 |
| OAuth | 256 MB | 0.5 |
| Policy | 256 MB | 0.5 |
| Org | 256 MB | 0.5 |
| Audit | 256 MB | 0.5 |
| Console | 512 MB | 0.5 |
| PostgreSQL | 512 MB | 1.0 |
| Redis | 256 MB | 0.5 |
| NATS | 128 MB | 0.5 |
| **Total** | **~3.3 GB** | **~6.5 cores** |

## Backup & Recovery

### Database Backup

```bash
# Backup
docker exec ggid-postgres pg_dump -U ggid ggid > backup_$(date +%Y%m%d).sql

# Restore
cat backup_20240101.sql | docker exec -i ggid-postgres psql -U ggid ggid
```

### Volume Backup

```bash
# Backup all volumes
docker run --rm -v ggid-pgdata:/data -v $(pwd):/backup alpine \
  tar czf /backup/ggid-pgdata.tar.gz -C /data .

# Restore
docker run --rm -v ggid-pgdata:/data -v $(pwd):/backup alpine \
  tar xzf /backup/ggid-pgdata.tar.gz -C /data
```

### RSA Key Rotation

```bash
# 1. Stop auth + oauth services
docker compose -f docker-compose.prod.yaml stop auth oauth

# 2. Remove old keys
docker run --rm -v ggid-configs:/configs alpine rm /configs/rsa_*.pem

# 3. Restart keygen
docker compose -f docker-compose.prod.yaml up keygen

# 4. Restart services
docker compose -f docker-compose.prod.yaml up -d auth oauth
```

## Monitoring

### Prometheus Metrics

```bash
# Gateway metrics endpoint
curl http://localhost:8080/metrics
```

### Grafana Dashboard

Import `deploy/grafana/dashboard-overview.json` into Grafana for:
- Request rate per service
- Error rate
- Latency P50/P95/P99
- Active sessions
- JWT verification rate

### Log Aggregation

All services use JSON-formatted logging. Forward to your log aggregator:

```bash
# Example: Filebeat configuration
filebeat.inputs:
  - type: container
    paths:
      - /var/lib/docker/containers/*/*.log
```

## Scaling

### Horizontal Scaling

```bash
# Scale policy service to 3 replicas
docker compose -f docker-compose.prod.yaml up -d --scale policy=3
```

Note: Session state is stored in Redis, so services are stateless and horizontally scalable.

### Vertical Scaling

Adjust resource limits in `docker-compose.prod.yaml`:

```yaml
deploy:
  resources:
    limits:
      memory: 512m  # Increase as needed
      cpus: "1.0"
```

## Troubleshooting

### Service Won't Start

```bash
# Check logs
docker compose -f docker-compose.prod.yaml logs <service>

# Check health
docker compose -f docker-compose.prod.yaml ps
```

### Database Connection Issues

```bash
# Verify database is running
docker exec ggid-postgres pg_isready -U ggid

# Check connection from a service
docker exec ggid-auth wget -qO- http://localhost:9001/healthz
```

### Reset Everything

```bash
# Stop and remove all containers + volumes
docker compose -f docker-compose.prod.yaml down -v

# Start fresh
docker compose -f docker-compose.prod.yaml --env-file .env up -d
```

## Production Checklist

- [ ] All passwords in `.env` changed from defaults
- [ ] TLS certificates configured (nginx or load balancer)
- [ ] Firewall configured (only expose 80/443)
- [ ] Database backup scheduled (cron or external tool)
- [ ] Log aggregation configured
- [ ] Monitoring alerts configured
- [ ] Resource limits tuned for your workload
- [ ] RSA keys backed up securely
- [ ] Rate limits configured appropriately
- [ ] Health check endpoints monitored externally
