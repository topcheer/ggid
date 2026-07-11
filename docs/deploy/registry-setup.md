# Private Docker Registry Setup

> How to set up a private Docker registry for GGID service images with TLS and authentication.

---

## Overview

```
Developer ──push──▶ Private Registry ──pull──▶ K8s/Docker Host
                     registry.example.com
```

---

## Quick Start (Docker Distribution)

### 1. Deploy Registry

```bash
docker run -d \
  --name registry \
  -p 5000:5000 \
  --restart=always \
  -v registry-data:/var/lib/registry \
  registry:2
```

### 2. Build and Push

```bash
# Tag images
docker tag ggid-gateway:latest localhost:5000/ggid/gateway:latest
docker tag ggid-auth:latest localhost:5000/ggid/auth:latest

# Push
docker push localhost:5000/ggid/gateway:latest
docker push localhost:5000/ggid/auth:latest
```

### 3. Pull from Registry

```bash
docker pull localhost:5000/ggid/gateway:latest
```

---

## TLS with Let's Encrypt

### 1. Create TLS Certificates

```bash
# Get certificate
certbot certonly --standalone -d registry.example.com

# Copy certs
cp /etc/letsencrypt/live/registry.example.com/fullchain.pem certs/domain.crt
cp /etc/letsencrypt/live/registry.example.com/privkey.pem certs/domain.key
```

### 2. Deploy with TLS

```bash
mkdir -p certs auth

docker run -d \
  --name registry \
  -p 443:443 \
  --restart=always \
  -v $(pwd)/certs:/certs:ro \
  -v registry-data:/var/lib/registry \
  -e REGISTRY_HTTP_ADDR=0.0.0.0:443 \
  -e REGISTRY_HTTP_TLS_CERTIFICATE=/certs/domain.crt \
  -e REGISTRY_HTTP_TLS_KEY=/certs/domain.key \
  registry:2
```

### 3. Login and Push

```bash
docker login registry.example.com
docker tag ggid-gateway:latest registry.example.com/ggid/gateway:latest
docker push registry.example.com/ggid/gateway:latest
```

---

## Authentication (htpasswd)

### 1. Create htpasswd File

```bash
mkdir -p auth
htpasswd -Bbn ggid-user secure-password > auth/htpasswd
```

### 2. Deploy with Auth

```bash
docker run -d \
  --name registry \
  -p 443:443 \
  -v $(pwd)/auth:/auth:ro \
  -v $(pwd)/certs:/certs:ro \
  -v registry-data:/var/lib/registry \
  -e REGISTRY_AUTH=htpasswd \
  -e REGISTRY_AUTH_HTPASSWD_REALM="GGID Registry" \
  -e REGISTRY_AUTH_HTPASSWD_PATH=/auth/htpasswd \
  -e REGISTRY_HTTP_TLS_CERTIFICATE=/certs/domain.crt \
  -e REGISTRY_HTTP_TLS_KEY=/certs/domain.key \
  registry:2
```

### 3. Login

```bash
docker login registry.example.com
# Username: ggid-user
# Password: secure-password
```

---

## Kubernetes Image Pull Secret

```bash
# Create secret for pulling from private registry
kubectl create secret docker-registry ggid-registry-secret \
  --docker-server=registry.example.com \
  --docker-username=ggid-user \
  --docker-password=secure-password \
  --docker-email=admin@example.com \
  -n ggid

# Reference in Helm values
cat >> values.yaml <<'EOF'
global:
  imagePullSecrets:
    - name: ggid-registry-secret
EOF
```

---

## Insecure Registry (Development Only)

For local registries without TLS (e.g., `registry.iot2.win`):

### Docker Desktop / OrbStack

```json
// ~/.docker/daemon.json or OrbStack Settings → Docker → Insecure registries
{
  "insecure-registries": [
    "registry.iot2.win:5000",
    "localhost:5000"
  ]
}
```

Restart Docker after editing.

### Kubernetes (K3s)

```yaml
# /etc/rancher/k3s/registries.yaml
mirrors:
  registry.iot2.win:
    endpoint:
      - "http://registry.iot2.win:5000"
```

Restart K3s: `sudo systemctl restart k3s`

---

## Using an Existing Registry (registry.iot2.win)

```bash
# Login
docker login registry.iot2.win

# Build and push all GGID images
for svc in gateway identity auth oauth policy org audit; do
  docker build -f services/$svc/Dockerfile \
    -t registry.iot2.win/ggid/$svc:latest .
  docker push registry.iot2.win/ggid/$svc:latest
done

# Build and push console
docker build -f console/Dockerfile \
  -t registry.iot2.win/ggid/console:latest .
docker push registry.iot2.win/ggid/console:latest

# Use in Helm
helm install ggid deploy/helm/ggid \
  --set global.imageRegistry=registry.iot2.win/ggid \
  -n ggid --create-namespace
```

---

## Garbage Collection

```bash
# Stop registry
docker stop registry

# Run garbage collection (removes unreferenced blobs)
docker run -it --rm \
  -v registry-data:/var/lib/registry \
  registry:2 garbage-collect \
  /etc/docker/registry/config.yml

# Restart
docker start registry
```

---

## Registry Backup

```bash
# Backup registry data
docker run --rm \
  -v registry-data:/data \
  -v $(pwd):/backup \
  alpine tar czf /backup/registry-backup.tar.gz -C /data .

# Restore
docker run --rm \
  -v registry-data:/data \
  -v $(pwd):/backup \
  alpine sh -c "cd /data && tar xzf /backup/registry-backup.tar.gz"
```

---

*Last updated: 2025-07-11*