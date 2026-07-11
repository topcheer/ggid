# Kubernetes Deployment Guide

> Deploy GGID to Kubernetes using Helm or raw manifests.

---

## Prerequisites

- Kubernetes 1.28+
- kubectl configured
- helm 3.8+
- Container registry with GGID images

---

## Option A: Helm Chart (Recommended)

### Install

```bash
kubectl create namespace ggid

helm install ggid deploy/helm/ggid \
  --namespace ggid \
  --set global.imageRegistry=ghcr.io/ggid \
  --set global.domain=iam.example.com
```

### With External Database

```bash
helm install ggid deploy/helm/ggid \
  --namespace ggid \
  --set postgresql.enabled=false \
  --set redis.enabled=false \
  --set nats.enabled=false \
  --set externalDatabase.host=prod-db.internal \
  --set externalDatabase.password=secret \
  --set externalRedis.host=prod-redis.internal \
  --set externalNats.url=nats://prod-nats:4222
```

### TLS via cert-manager

```yaml
# values-tls.yaml
ingress:
  enabled: true
  tls: true
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod

# Apply
helm upgrade ggid deploy/helm/ggid -f values-tls.yaml -n ggid
```

---

## Option B: Raw Manifests

### PostgreSQL StatefulSet

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: postgres
  namespace: ggid
spec:
  serviceName: postgres
  replicas: 1
  selector:
    matchLabels:
      app: postgres
  template:
    metadata:
      labels:
        app: postgres
    spec:
      containers:
      - name: postgres
        image: postgres:16
        env:
        - name: POSTGRES_DB
          value: ggid
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: db-secret
              key: password
        ports:
        - containerPort: 5432
        volumeMounts:
        - name: data
          mountPath: /var/lib/postgresql/data
  volumeClaimTemplates:
  - metadata:
      name: data
    spec:
      accessModes: ["ReadWriteOnce"]
      resources:
        requests:
          storage: 20Gi
```

### NATS JetStream

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: nats
  namespace: ggid
spec:
  serviceName: nats
  replicas: 1
  template:
    spec:
      containers:
      - name: nats
        image: nats:2.10-alpine
        args: ["-js", "-m", "8222", "--store_dir", "/data"]
        ports:
        - containerPort: 4222
        - containerPort: 8222
        volumeMounts:
        - name: data
          mountPath: /data
  volumeClaimTemplates:
  - metadata:
      name: data
    spec:
      accessModes: ["ReadWriteOnce"]
      resources:
        requests:
          storage: 10Gi
```

### GGID Gateway Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ggid-gateway
  namespace: ggid
spec:
  replicas: 2
  template:
    spec:
      containers:
      - name: gateway
        image: ghcr.io/ggid/gateway:latest
        ports:
        - containerPort: 8080
        env:
        - name: NATS_URL
          value: "nats://nats:4222"
        - name: REDIS_HOST
          value: "redis:6379"
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8080
        readinessProbe:
          httpGet:
            path: /healthz
            port: 8080
```

---

## HPA Auto-Scaling

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: ggid-gateway
  namespace: ggid
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: ggid-gateway
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
```

---

## Verification

```bash
kubectl -n ggid get pods
kubectl -n ggid port-forward svc/ggid-gateway 8080:8080
curl http://localhost:8080/healthz
```

---

*See: [Helm Chart Guide](../deploy/helm-chart-guide.md) | [Helm 5-Minute](../quickstart/helm-5-min.md) | [Production Checklist](../deploy/production-checklist.md)*

*Last updated: 2025-07-11*
