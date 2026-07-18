# Self-Hosting Guide

Complete guide for deploying GGID on your own infrastructure. Covers development, small-scale (<1000 users), and production deployments.

---

## Hardware Requirements

### Development / Testing

| Component | CPU | RAM | Disk |
|-----------|-----|-----|------|
| All-in-one container | 2 cores | 4 GB | 10 GB SSD |
| PostgreSQL (embedded) | shared | shared | 5 GB |

### Small Scale (<1,000 users)

| Component | CPU | RAM | Disk | Notes |
|-----------|-----|-----|------|-------|
| GGID (all services) | 4 cores | 8 GB | 20 GB SSD | Single node |
| PostgreSQL | 2 cores | 4 GB | 50 GB SSD | Separate or embedded |
| Redis | 1 core | 2 GB | 1 GB | Cache + sessions |
| NATS | 1 core | 512 MB | 1 GB | Audit event bus |

### Medium Scale (1,000-10,000 users)

| Component | CPU | RAM | Disk | Notes |
|-----------|-----|-----|------|-------|
| GGID Gateway | 2 cores | 2 GB | 5 GB | 2 replicas |
| GGID Auth | 2 cores | 2 GB | 5 GB | 2 replicas |
| GGID Identity | 2 cores | 2 GB | 5 GB | 2 replicas |
| GGID OAuth | 2 cores | 1 GB | 5 GB | 2 replicas |
| GGID Policy | 2 cores | 1 GB | 5 GB | 2 replicas |
| GGID Audit | 2 cores | 2 GB | 50 GB | 1-2 replicas |
| PostgreSQL | 4 cores | 16 GB | 200 GB SSD | Managed recommended |
| Redis | 2 cores | 4 GB | 5 GB | Managed recommended |
| NATS | 2 cores | 1 GB | 10 GB | Cluster of 3 |

### Large Scale (10,000+ users)

Contact us for architecture review. Key considerations:
- Horizontal autoscaling for stateless services
- PostgreSQL read replicas + connection pooling (PgBouncer)
- Redis cluster mode
- NATS cluster (3+ nodes) with JetStream persistence
- CDN for Console static assets
- Dedicated audit storage (hot/cold tiering)

---

## Quick Start: Docker Compose (Development / Small Scale)

### Prerequisites

- Docker 24+ and Docker Compose v2+
- 4 GB free RAM
- Ports 8080, 3000, 5432 available

### Steps

```bash
# 1. Clone the repository
git clone https://github.com/topcheer/ggid.git
cd ggid

# 2. Copy environment configuration
cp .env.example .env
# Edit .env: set DB_PASSWORD, AES_KEY, JWT_SECRET

# 3. Start all services
docker compose -f deploy/docker-compose.yml up -d

# 4. Wait for services to initialize (30-60 seconds)
docker compose -f deploy/docker-compose.yml logs -f ggid

# 5. Verify health
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz

# 6. Access Console
open http://localhost:3000
# Default admin: admin@ggid.local / Admin123!
```

### Configuration via .env

Key environment variables (see `.env.example` for full list):

```bash
# Database
DB_PASSWORD=change-me          # REQUIRED — strong password
DATABASE_URL=postgres://ggid:change-me@localhost:5432/ggid?sslmode=disable

# Security — generate with: openssl rand -hex 32
AES_KEY=0000000000000000000000000000000000000000000000000000000000000000
JWT_SECRET=change-me-jwt-secret

# SMTP (for password resets, notifications)
SMTP_HOST=smtp.yourprovider.com
SMTP_PORT=587
SMTP_USER=apikey
SMTP_PASSWORD=your-smtp-key
SMTP_FROM=noreply@yourcompany.com

# External URLs (for OAuth redirects)
GATEWAY_URL=https://auth.yourcompany.com
CONSOLE_URL=https://console.yourcompany.com
```

---

## Production Deployment: Kubernetes (K3s / EKS / AKS / GKE)

### Option A: K3s (Lightweight — On-Premise / Edge)

```bash
# 1. Install K3s
curl -sfL https://get.k3s.io | sh -

# 2. Apply GGID manifests
kubectl apply -f deploy/k8s/namespace.yaml
kubectl apply -f deploy/k8s/secrets.yaml    # Edit first: set secrets
kubectl apply -f deploy/k8s/postgres.yaml
kubectl apply -f deploy/k8s/redis.yaml
kubectl apply -f deploy/k8s/ggid-services.yaml
kubectl apply -f deploy/k8s/ingress.yaml

# 3. Verify
kubectl get pods -n ggid
kubectl get ingress -n ggid
```

### Option B: Managed Kubernetes (EKS/AKS/GKE)

1. **Create cluster** (3 nodes minimum, 4 vCPU / 16 GB each)

2. **Configure storage classes** for PostgreSQL and audit:
```yaml
# deploy/k8s/storage-class.yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: ggid-ssd
provisioner: kubernetes.io/aws-ebs-storage  # Adjust for your cloud
parameters:
  type: gp3
  fsType: ext4
volumeBindingMode: WaitForFirstConsumer
```

3. **Deploy GGID**:
```bash
kubectl create namespace ggid
kubectl apply -f deploy/k8s/ -n ggid
```

4. **Configure Ingress with TLS** (see TLS section below)

---

## External PostgreSQL Configuration

### AWS RDS

```bash
# .env
DB_HOST=my-ggid-db.xxxxxx.us-east-1.rds.amazonaws.com
DB_PORT=5432
DB_USER=ggid
DB_PASSWORD=secure-password
DB_NAME=ggid
DATABASE_URL=postgres://ggid:secure-password@my-ggid-db.xxxxxx.us-east-1.rds.amazonaws.com:5432/ggid?sslmode=require
```

RDS settings:
- Engine: PostgreSQL 16+
- Instance: `db.t3.medium` minimum (production: `db.r6g.large`)
- Storage: 100 GB GP3, autoscale enabled
- Multi-AZ: yes (production)
- Automated backups: 7-day retention

### Google Cloud SQL

```bash
# .env
DB_HOST=10.x.x.x  # Private IP from Cloud SQL
DB_PORT=5432
DATABASE_URL=postgres://ggid:password@10.x.x.x:5432/ggid?sslmode=require
```

Cloud SQL settings:
- PostgreSQL 16+
- `db-custom-2-7680` minimum
- Private IP + VPC peering
- Automated backups: 7-day

### Azure Database for PostgreSQL

```bash
# .env
DB_HOST=ggid-pg.postgres.database.azure.com
DB_PORT=5432
DATABASE_URL=postgres://ggid@pgadmin:password@ggid-pg.postgres.database.azure.com:5432/ggid?sslmode=require
```

---

## External Redis Configuration

### AWS ElastiCache

```bash
# .env
REDIS_URL=rediss://master.ggid-cache.xxxxxx.use1.cache.amazonaws.com:6379
```

Settings:
- Engine: Redis 7.x
- Node type: `cache.t3.small` minimum
- Multi-AZ with automatic failover
- Encryption in transit (`rediss://`) and at rest
- Parameter group: `maxmemory-policy=allkeys-lru`

### Google Memorystore

```bash
# .env
REDIS_URL=rediss://10.x.x.x:6379
```

---

## TLS Certificate Configuration

### Let's Encrypt (Recommended — Free, Automated)

Using cert-manager on Kubernetes:

```bash
# 1. Install cert-manager
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.14.0/cert-manager.yaml

# 2. Create ClusterIssuer
cat <<EOF | kubectl apply -f -
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: admin@yourcompany.com
    privateKeySecretRef:
      name: letsencrypt-prod
    solvers:
    - http01:
        ingress:
          class: nginx
EOF

# 3. Configure Ingress with TLS
# In deploy/k8s/ingress.yaml, add:
# annotations:
#   cert-manager.io/cluster-issuer: letsencrypt-prod
# tls:
#   - hosts: [auth.yourcompany.com]
#     secretName: ggid-tls
```

### Self-Signed (Development Only)

```bash
# Generate self-signed certificate
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem \
  -days 365 -nodes -subj "/CN=localhost" \
  -addext "subjectAltName=DNS:localhost,DNS:auth.yourcompany.com"

# Mount in docker-compose:
# volumes:
#   - ./certs:/etc/ggid/certs
# environment:
#   - TLS_CERT=/etc/ggid/certs/cert.pem
#   - TLS_KEY=/etc/ggid/certs/key.pem
```

### Enterprise CA

Mount your CA-signed certificate pair and configure:

```bash
TLS_CERT=/etc/ggid/certs/fullchain.pem
TLS_KEY=/etc/ggid/certs/privkey.pem
TLS_CA=/etc/ggid/certs/ca-bundle.crt  # For mTLS with internal services
```

---

## Mail (SMTP) Configuration

### Generic SMTP

```bash
SMTP_HOST=mail.yourcompany.com
SMTP_PORT=587
SMTP_USER=ggid
SMTP_PASSWORD=secure
SMTP_FROM=noreply@yourcompany.com
SMTP_TLS=true
```

### SendGrid

```bash
SMTP_HOST=smtp.sendgrid.net
SMTP_PORT=587
SMTP_USER=apikey
SMTP_PASSWORD=SG.xxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
SMTP_FROM=noreply@yourcompany.com
```

### AWS SES

```bash
SMTP_HOST=email-smtp.us-east-1.amazonaws.com
SMTP_PORT=587
SMTP_USER=AKIAXXXXXXXXXXXX
SMTP_PASSWORD=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
SMTP_FROM=noreply@yourcompany.com
```

---

## Backup Strategy

### Database Backups

```bash
# Daily automated backup (cron)
pg_dump $DATABASE_URL | gzip > /backups/ggid-$(date +%Y%m%d).sql.gz

# Retention: keep 30 days
find /backups -name "ggid-*.sql.gz" -mtime +30 -delete

# Restore
gunzip -c /backups/ggid-20250115.sql.gz | psql $DATABASE_URL
```

For managed databases (RDS/Cloud SQL): enable automated snapshots with 7-day retention + weekly manual snapshot.

### Audit Data (Long-Term)

```bash
# Export audit events quarterly for cold storage
curl -H "Authorization: Bearer $TOKEN" \
  "$GATEWAY_URL/api/v1/audit/export?format=json&start=2025-01-01T00:00:00Z&end=2025-03-31T23:59:59Z" \
  -o audit-Q1-2025.json

# Upload to S3/GCS for archival
aws s3 cp audit-Q1-2025.json s3://ggid-audit-archive/$(date +%Y)/
```

---

## Upgrade Process

### Docker Compose

```bash
# 1. Pull latest image
docker compose -f deploy/docker-compose.yml pull

# 2. Backup database
pg_dump $DATABASE_URL > backup-$(date +%Y%m%d).sql

# 3. Stop services
docker compose -f deploy/docker-compose.yml down

# 4. Start with new images (auto-runs migrations)
docker compose -f deploy/docker-compose.yml up -d

# 5. Verify
curl http://localhost:8080/healthz
docker compose -f deploy/docker-compose.yml logs --tail 50 ggid
```

### Kubernetes

```bash
# 1. Update image tag
kubectl set image deployment/ggid-gateway gateway=ggid/gateway:v1.1.0 -n ggid
kubectl set image deployment/ggid-auth auth=ggid/auth:v1.1.0 -n ggid
# ... repeat for each service

# 2. Wait for rollout
kubectl rollout status deployment/ggid-gateway -n ggid

# 3. Rollback if needed
kubectl rollout undo deployment/ggid-gateway -n ggid
```

---

## Performance Tuning

### PostgreSQL

```sql
-- Connection pooling (PgBouncer)
-- max_connections = 100 in PgBouncer (GGID opens ~20 connections per service)

-- For 10,000+ users:
ALTER SYSTEM SET shared_buffers = '4GB';
ALTER SYSTEM SET effective_cache_size = '12GB';
ALTER SYSTEM SET maintenance_work_mem = '512MB';
ALTER SYSTEM SET random_page_cost = 1.1;  -- SSD
ALTER SYSTEM SET checkpoint_completion_target = 0.9;
```

### Redis

```bash
# For session-heavy deployments:
maxmemory 2gb
maxmemory-policy allkeys-lru
timeout 300
tcp-keepalive 60
```

### GGID Service Tuning

```bash
# Increase per-service connection pool
DB_MAX_CONNS=50            # Per service instance

# NATS streaming for high-volume audit
NATS_MAX_PAYLOAD=1048576   # 1MB

# Gateway tuning
GATEWAY_READ_TIMEOUT=30s
GATEWAY_WRITE_TIMEOUT=30s
GATEWAY_IDLE_TIMEOUT=120s

# Redis pool
REDIS_POOL_SIZE=50
REDIS_MIN_IDLE=10
```
