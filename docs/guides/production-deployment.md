# GGID Production Deployment Guide

This guide covers 4 deployment methods and cloud-specific configurations.

## Table of Contents

1. [All-in-One Binary](#1-all-in-one-binary)
2. [Docker Compose](#2-docker-compose)
3. [Helm (Kubernetes)](#3-helm-kubernetes)
4. [Terraform (Infrastructure as Code)](#4-terraform)
5. [Cloud Platforms](#5-cloud-platforms)
6. [TLS / DNS Configuration](#6-tls--dns-configuration)
7. [External Services](#7-external-services)
8. [Upgrade Flow](#8-upgrade-flow)

---

## 1. All-in-One Binary

Simplest deployment for testing or small deployments.

```bash
# Build
make build-all-in-one

# Run (requires PostgreSQL + Redis + NATS)
./ggid-all-in-one \
  --db-host=localhost \
  --db-port=5432 \
  --redis-host=localhost \
  --nats-url=nats://localhost:4222
```

**Best for**: Local development, CI/CD testing, single-server deployments.

---

## 2. Docker Compose

Bundled deployment with all services + PostgreSQL + Redis + NATS.

```bash
# Start all services
docker compose -f deploy/docker-compose.yml up -d

# Production config (with TLS, persistent storage)
docker compose -f deploy/docker-compose.prod.yaml up -d

# Check status
docker compose ps

# View logs
docker compose logs -f gateway
```

**Configuration**: Edit `deploy/docker-compose.yml` environment variables.

**Best for**: Single-host production, staging, demos.

---

## 3. Helm (Kubernetes)

Production-grade deployment on Kubernetes.

```bash
# Install
helm install ggid deploy/helm/ggid/ -n ggid --create-namespace

# Custom values
helm install ggid deploy/helm/ggid/ -n ggid -f my-values.yaml

# Upgrade
helm upgrade ggid deploy/helm/ggid/ -n ggid -f my-values.yaml

# Uninstall
helm uninstall ggid -n ggid
```

### Key Configurations

```yaml
# my-values.yaml
postgresql:
  enabled: true
  auth:
    password: "CHANGE_ME"

redis:
  enabled: true
  auth:
    password: "CHANGE_ME"

gateway:
  replicaCount: 3
  service:
    type: LoadBalancer

auth:
  replicaCount: 2

networkPolicy:
  enabled: true

jwt:
  secret: "your-jwt-secret"
```

See: `deploy/helm/README.md` for full configuration reference.

**Best for**: Production, scalable deployments.

---

## 4. Terraform

Provision infrastructure using Terraform.

```bash
cd deploy/terraform

# Initialize
terraform init

# Plan
terraform plan -var-file=production.tfvars

# Apply
terraform apply -var-file=production.tfvars
```

**Best for**: Multi-environment management, GitOps workflows.

---

## 5. Cloud Platforms

### AWS EKS

```bash
./deploy/aws/deploy-eks.sh
```

- Uses `eksctl` to create EKS cluster
- AWS Load Balancer Controller for ALB ingress
- Recommended: RDS for PostgreSQL, ElastiCache for Redis

### Azure AKS

```bash
./deploy/azure/deploy-aks.sh
```

- Uses `az aks create` for managed Kubernetes
- Optional: Application Gateway for ingress
- Recommended: Azure Database for PostgreSQL, Azure Cache for Redis

### GCP GKE

```bash
./deploy/gcp/deploy-gke.sh
```

- Uses `gcloud container clusters create`
- Optional: Cloud SQL for PostgreSQL
- Recommended: Memorystore for Redis

---

## 6. TLS / DNS Configuration

### Using cert-manager + Let's Encrypt

```bash
# Install cert-manager
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.15.0/cert-manager.yaml

# Create ClusterIssuer
cat <<EOF | kubectl apply -f -
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: admin@example.com
    privateKeySecretRef:
      name: letsencrypt-prod
    solvers:
      - http01:
          ingress:
            class: nginx
EOF
```

### DNS

Point your domain to the LoadBalancer external IP:
```
ggid.example.com  A  <LOAD_BALANCER_IP>
```

---

## 7. External Services

### External PostgreSQL

```yaml
postgresql:
  enabled: false
externalDatabase:
  host: "db.internal.example.com"
  port: 5432
  username: "ggid"
  password: "secure-password"
  database: "ggid"
```

### External Redis

```yaml
redis:
  enabled: false
externalRedis:
  host: "redis.internal.example.com"
  port: 6379
  password: "secure-password"
```

### External NATS

```yaml
nats:
  enabled: false
externalNats:
  url: "nats://nats.internal.example.com:4222"
```

---

## 8. Upgrade Flow

### Docker Compose

```bash
git pull
docker compose pull
docker compose up -d
```

### Helm

```bash
helm repo update
helm upgrade ggid deploy/helm/ggid/ -f my-values.yaml
```

### Database Migrations

Always run migrations after upgrading:

```bash
kubectl exec -it deploy/ggid-identity -n ggid -- ./migrate up
```

Or using the migration script:
```bash
./deploy/migrate.sh
```
