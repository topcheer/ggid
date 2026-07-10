# GGID Helm Chart

## Quick Start

```bash
# Install with defaults
helm install ggid ./deploy/helm/ggid

# Install with custom values
helm install ggid ./deploy/helm/ggid \
  --set global.imageRegistry=registry.example.com \
  --set gateway.replicaCount=3 \
  --set ingress.enabled=true \
  --set ingress.hosts[0].host=iam.example.com

# Use an external database
helm install ggid ./deploy/helm/ggid \
  --set postgresql.enabled=false \
  --set externalDatabase.host=db.internal \
  --set externalDatabase.password=$DB_PASSWORD
```

## Production Checklist

- [ ] Set unique `postgresql.auth.password` and `redis.auth.password`
- [ ] Configure `ingress.enabled=true` with TLS
- [ ] Set `gateway.replicaCount >= 2` for HA
- [ ] Configure `global.storageClass` for persistent volumes
- [ ] Set `imagePullSecrets` for private registries
- [ ] Configure HPA (requires metrics-server)

## Values

| Key | Default | Description |
|-----|---------|-------------|
| `global.imageRegistry` | `""` | Docker registry override |
| `gateway.replicaCount` | `2` | Gateway pod replicas |
| `gateway.service.type` | `ClusterIP` | Service type |
| `ingress.enabled` | `false` | Enable ingress controller |
| `ingress.hosts[0].host` | `iam.example.com` | Ingress hostname |
| `postgresql.enabled` | `true` | Deploy bundled PostgreSQL |
