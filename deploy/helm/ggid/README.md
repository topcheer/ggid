# GGID - Identity & Access Management Suite

Full-stack IAM platform: Authentication, Authorization, SSO, Multi-Tenancy, Audit.

## Quick Deploy

```bash
# Add Helm repo (when published)
helm install ggid ./deploy/helm/ggid

# Or with custom values
helm install ggid ./deploy/helm/ggid \
  --set global.imageRegistry=your-registry.com \
  --set postgresql.auth.password=your-password
```

## Configuration

See [values.yaml](./values.yaml) for all configuration options.

Key parameters:

| Parameter | Description | Default |
|-----------|-------------|---------|
| `postgresql.enabled` | Deploy bundled PostgreSQL | `true` |
| `redis.enabled` | Deploy bundled Redis | `true` |
| `nats.enabled` | Deploy bundled NATS | `true` |
| `gateway.replicaCount` | Gateway replicas | `2` |
| `ingress.enabled` | Enable ingress | `false` |
