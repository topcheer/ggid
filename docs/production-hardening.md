# GGID Production Hardening Guide

> Security checklist and production deployment best practices.

---

## 1. Docker Compose Production Hardening

### 1.1 Secrets Management

Replace plaintext passwords with Docker secrets or environment files:

```bash
# Create a .env file (NEVER commit this)
cat > deploy/.env << 'EOF'
POSTGRES_PASSWORD=$(openssl rand -base64 32)
REDIS_PASSWORD=$(openssl rand -base64 32)
JWT_SIGNING_KEY=$(openssl rand -base64 48)
NATS_PASSWORD=$(openssl rand -base64 32)
LDAP_BIND_PASSWORD=$(openssl rand -base64 32)
WEBHOOK_SECRET=$(openssl rand -base64 32)
EOF

# Load in docker-compose
# docker-compose --env-file deploy/.env up -d
```

### 1.2 TLS Configuration

Add a reverse proxy (Caddy/Traefik) for automatic TLS:

```yaml
# Add to docker-compose.yaml
  caddy:
    image: caddy:2-alpine
    ports:
      - "443:443"
      - "80:80"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile
      - caddy_data:/data
      - caddy_config:/config
    depends_on:
      - gateway
```

Caddyfile:
```
iam.example.com {
    reverse_proxy gateway:8080
    encode gzip zstd
    header {
        Strict-Transport-Security "max-age=31536000; includeSubDomains; preload"
        X-Content-Type-Options nosniff
        X-Frame-Options DENY
        Referrer-Policy strict-origin-when-cross-origin
    }
}
```

### 1.3 Resource Limits

Add resource constraints to every service:

```yaml
services:
  gateway:
    # ... existing config ...
    deploy:
      resources:
        limits:
          cpus: '2.0'
          memory: 512M
        reservations:
          cpus: '0.5'
          memory: 128M
```

### 1.4 Persistent Volumes

Ensure data survives container restarts:

```yaml
volumes:
  ggid-pgdata:
    driver: local
    driver_opts:
      type: none
      o: bind
      device: /data/ggid/postgres  # Mount to fast SSD
  ggid-redis-data:
    driver: local
  ggid-nats-data:
    driver: local
```

### 1.5 Network Isolation

```yaml
networks:
  frontend:
    # Gateway, Console — exposed to host
  backend:
    # Services — internal only
    internal: true
  infrastructure:
    # DB, Redis, NATS — internal only
    internal: true
```

Assign services:
- `gateway`: both `frontend` + `backend`
- `console`: `frontend`
- All other services: `backend` + `infrastructure`
- DB/Redis/NATS: `infrastructure` only

### 1.6 Health Check Tuning

```yaml
healthcheck:
  test: ["CMD", "wget", "--spider", "-q", "http://localhost:8080/healthz"]
  interval: 10s
  timeout: 5s
  retries: 3
  start_period: 30s  # Give time for DB migration
```

---

## 2. Kubernetes Production Checklist

### 2.1 Resource Management

```yaml
# values.yaml production overrides
gateway:
  replicaCount: 3  # Minimum 3 for HA
  resources:
    limits:
      cpu: 1000m
      memory: 512Mi
    requests:
      cpu: 250m
      memory: 128Mi

# Enable HPA
hpa:
  enabled: true
  minReplicas: 3
  maxReplicas: 20
  targetCPUUtilizationPercentage: 70
```

### 2.2 Pod Security

```yaml
podSecurityContext:
  runAsNonRoot: true
  runAsUser: 65532  # nonroot user from distroleless image
  fsGroup: 65532

securityContext:
  allowPrivilegeEscalation: false
  readOnlyRootFilesystem: true
  capabilities:
    drop:
      - ALL
```

### 2.3 Network Policies

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: ggid-deny-all-default
spec:
  podSelector: {}
  policyTypes:
    - Ingress
    - Egress
---
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: ggid-allow-gateway-ingress
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/name: ggid
  ingress:
    - from:
        - namespaceSelector:
            matchLabels:
              name: ingress-nginx
      ports:
        - port: 8080
```

### 2.4 Backup Strategy

```bash
# PostgreSQL backup (cron job)
kubectl create cronjob pg-backup \
  --schedule="0 2 * * *" \
  --image=postgres:16-alpine \
  --env="PGPASSWORD=$DB_PASSWORD" \
  -- pg_dump -h ggid-postgresql -U ggid -Fc ggid > /backup/ggid-$(date +%Y%m%d).dump

# Redis: Enable AOF persistence
redis:
  master:
    persistence:
      enabled: true
      size: 10Gi
    configDatabase: |
      appendonly yes
      appendfsync everysec
```

---

## 3. Monitoring & Alerting

### 3.1 Prometheus Alerts

```yaml
groups:
  - name: ggid-alerts
    rules:
      - alert: HighErrorRate
        expr: rate(ggid_http_requests_total{status=~"5.."}[5m]) > 0.05
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "GGID error rate > 5%"

      - alert: HighLoginFailures
        expr: rate(ggid_auth_login_failures_total[5m]) > 10
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "High login failure rate — possible brute force"

      - alert: DatabaseConnectionsHigh
        expr: pg_stat_database_numbackends > 150
        for: 5m
        labels:
          severity: warning

      - alert: NATSConsumerLag
        expr: nats_consumer_num_ack_pending > 1000
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Audit consumer lag — events may be delayed"
```

### 3.2 Grafana Dashboard

Import the GGID dashboard JSON from `deploy/grafana/dashboard.json`:
- Request rate + latency (p50/p95/p99)
- Error rate by endpoint
- Login success/failure ratio
- Active sessions
- Database connection pool
- NATS consumer lag
- JWT verification latency

---

## 4. Secret Rotation

| Secret | Rotation Frequency | Method |
|--------|-------------------|--------|
| JWT signing key | Quarterly | Generate new key → add to JWKS → remove old after 24h |
| Database password | Bi-annually | Update Secret → rolling restart |
| Redis password | Bi-annually | Update Secret → rolling restart |
| OAuth client secrets | On demand | Rotate via Admin Console |
| LDAP bind password | Annually | Update env var → restart auth service |
| TLS certificates | Auto (cert-manager/Let's Encrypt) | 90 days, auto-renewed |

---

## 5. Production Checklist

- [ ] All secrets in env files or K8s Secrets (not in git)
- [ ] TLS enabled (Caddy/Traefik/cert-manager)
- [ ] Resource limits set on all containers
- [ ] Persistent volumes mounted for DB/Redis/NATS
- [ ] Network isolation (internal networks for backend services)
- [ ] Health checks configured for all services
- [ ] Backup automation (daily PostgreSQL dump)
- [ ] Monitoring + alerting (Prometheus + Grafana)
- [ ] Log aggregation (Loki/ELK/Datadog)
- [ ] HPA configured (min 3 replicas for HA)
- [ ] Pod Disruption Budget set
- [ ] Rate limits configured per tenant
- [ ] MFA enforced for admin accounts
- [ ] Audit log retention configured
- [ ] Regular dependency scanning (Trivy/govulncheck)
