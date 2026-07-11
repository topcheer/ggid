# Kubernetes Deployment Tutorial

This tutorial covers deploying the full GGID stack (7 microservices + infrastructure) to Kubernetes.

## Prerequisites

- Kubernetes 1.28+ cluster
- `kubectl` configured
- Container registry with GGID images
- TLS certificate for ingress

## Step 1: Namespace

```bash
kubectl create namespace ggid
```

```yaml
# namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: ggid
  labels:
    app.kubernetes.io/part-of: ggid
```

## Step 2: Secrets

```bash
# Create database password secret
kubectl create secret generic ggid-db-secret \
  --namespace=ggid \
  --from-literal=password='STRONG_DB_PASSWORD'

# Create JWT signing key secret
kubectl create secret generic ggid-jwt-secret \
  --namespace=ggid \
  --from-file=jwt-signing-key=/path/to/private.key

# Create Redis password secret
kubectl create secret generic ggid-redis-secret \
  --namespace=ggid \
  --from-literal=password='STRONG_REDIS_PASSWORD'
```

## Step 3: ConfigMap

```yaml
# configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: ggid-config
  namespace: ggid
data:
  TENANT_ID: "00000000-0000-0000-0000-000000000001"
  DB_HOST: "postgres.ggid.svc.cluster.local"
  DB_PORT: "5432"
  DB_NAME: "ggid"
  DB_USER: "ggid"
  DB_SSLMODE: "require"
  REDIS_HOST: "redis.ggid.svc.cluster.local"
  REDIS_PORT: "6379"
  NATS_URL: "nats://nats.ggid.svc.cluster.local:4222"
  GATEWAY_ADDR: ":8080"
  AUTH_ADDR: ":9001"
  OAUTH_ADDR: ":9005"
  IDENTITY_ADDR: ":8080"
  POLICY_HTTP_ADDR: ":8070"
  POLICY_GRPC_ADDR: ":9070"
  ORG_HTTP_ADDR: ":8071"
  ORG_GRPC_ADDR: ":9071"
  AUDIT_HTTP_ADDR: ":8072"
  AUDIT_GRPC_ADDR: ":9072"
  LOG_LEVEL: "info"
```

## Step 4: PostgreSQL

```yaml
# postgres.yaml
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
        image: postgres:16-alpine
        ports:
        - containerPort: 5432
        env:
        - name: POSTGRES_DB
          value: ggid
        - name: POSTGRES_USER
          value: ggid
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: ggid-db-secret
              key: password
        volumeMounts:
        - name: pgdata
          mountPath: /var/lib/postgresql/data
        resources:
          requests:
            cpu: 250m
            memory: 512Mi
          limits:
            cpu: 1000m
            memory: 2Gi
        livenessProbe:
          exec:
            command: ["pg_isready", "-U", "ggid"]
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          exec:
            command: ["pg_isready", "-U", "ggid"]
          initialDelaySeconds: 5
          periodSeconds: 5
  volumeClaimTemplates:
  - metadata:
      name: pgdata
    spec:
      accessModes: ["ReadWriteOnce"]
      resources:
        requests:
          storage: 20Gi
---
apiVersion: v1
kind: Service
metadata:
  name: postgres
  namespace: ggid
spec:
  selector:
    app: postgres
  ports:
  - port: 5432
    targetPort: 5432
```

## Step 5: Redis

```yaml
# redis.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: redis
  namespace: ggid
spec:
  replicas: 1
  selector:
    matchLabels:
      app: redis
  template:
    metadata:
      labels:
        app: redis
    spec:
      containers:
      - name: redis
        image: redis:7-alpine
        command: ["redis-server", "--appendonly", "yes", "--requirepass", "$(REDIS_PASSWORD)"]
        env:
        - name: REDIS_PASSWORD
          valueFrom:
            secretKeyRef:
              name: ggid-redis-secret
              key: password
        ports:
        - containerPort: 6379
        volumeMounts:
        - name: redis-data
          mountPath: /data
        resources:
          requests:
            cpu: 100m
            memory: 256Mi
          limits:
            cpu: 500m
            memory: 512Mi
      volumes:
      - name: redis-data
        emptyDir: {}
---
apiVersion: v1
kind: Service
metadata:
  name: redis
  namespace: ggid
spec:
  selector:
    app: redis
  ports:
  - port: 6379
    targetPort: 6379
```

## Step 6: NATS

```yaml
# nats.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nats
  namespace: ggid
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nats
  template:
    metadata:
      labels:
        app: nats
    spec:
      containers:
      - name: nats
        image: nats:2.10-alpine
        command: ["nats-server", "--jetstream", "--store_dir", "/data", "-m", "8222"]
        ports:
        - containerPort: 4222
        - containerPort: 8222
        volumeMounts:
        - name: nats-data
          mountPath: /data
        resources:
          requests:
            cpu: 100m
            memory: 256Mi
          limits:
            cpu: 500m
            memory: 512Mi
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8222
          initialDelaySeconds: 10
          periodSeconds: 10
      volumes:
      - name: nats-data
        emptyDir: {}
---
apiVersion: v1
kind: Service
metadata:
  name: nats
  namespace: ggid
spec:
  selector:
    app: nats
  ports:
  - name: client
    port: 4222
    targetPort: 4222
  - name: monitor
    port: 8222
    targetPort: 8222
```

## Step 7: Microservices

```yaml
# services.yaml
# --- Gateway ---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gateway
  namespace: ggid
spec:
  replicas: 2
  selector:
    matchLabels:
      app: gateway
  template:
    metadata:
      labels:
        app: gateway
    spec:
      containers:
      - name: gateway
        image: ggid/gateway:latest
        ports:
        - containerPort: 8080
        envFrom:
        - configMapRef:
            name: ggid-config
        - secretRef:
            name: ggid-jwt-secret
        env:
        - name: IDENTITY_URL
          value: "http://identity.ggid.svc.cluster.local:8080"
        - name: AUTH_URL
          value: "http://auth.ggid.svc.cluster.local:9001"
        - name: OAUTH_URL
          value: "http://oauth.ggid.svc.cluster.local:9005"
        - name: POLICY_URL
          value: "http://policy.ggid.svc.cluster.local:8070"
        - name: ORG_URL
          value: "http://org.ggid.svc.cluster.local:8071"
        - name: AUDIT_URL
          value: "http://audit.ggid.svc.cluster.local:8072"
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8080
          initialDelaySeconds: 15
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /healthz
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
        resources:
          requests:
            cpu: 100m
            memory: 128Mi
          limits:
            cpu: 500m
            memory: 256Mi
---
apiVersion: v1
kind: Service
metadata:
  name: gateway
  namespace: ggid
spec:
  selector:
    app: gateway
  ports:
  - port: 8080
    targetPort: 8080

---
# --- Identity ---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: identity
  namespace: ggid
spec:
  replicas: 2
  selector:
    matchLabels:
      app: identity
  template:
    metadata:
      labels:
        app: identity
    spec:
      containers:
      - name: identity
        image: ggid/identity:latest
        ports:
        - containerPort: 8080
        - containerPort: 50051
        envFrom:
        - configMapRef:
            name: ggid-config
        env:
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: ggid-db-secret
              key: password
        - name: DATABASE_URL
          value: "postgres://ggid:$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)"
        livenessProbe:
          httpGet: { path: /healthz, port: 8080 }
          initialDelaySeconds: 15
          periodSeconds: 10
        readinessProbe:
          httpGet: { path: /healthz, port: 8080 }
          initialDelaySeconds: 5
          periodSeconds: 5
        resources:
          requests: { cpu: 100m, memory: 128Mi }
          limits: { cpu: 500m, memory: 256Mi }
---
apiVersion: v1
kind: Service
metadata:
  name: identity
  namespace: ggid
spec:
  selector: { app: identity }
  ports:
  - name: http
    port: 8080
    targetPort: 8080
  - name: grpc
    port: 50051
    targetPort: 50051

---
# --- Auth / OAuth / Policy / Org / Audit ---
# (Same pattern as Identity — change name, image, ports)
# Apply the same Deployment+Service pattern for each:

# auth:    ports 9001
# oauth:   ports 9005
# policy:  ports 8070 (http), 9070 (grpc)
# org:     ports 8071 (http), 9071 (grpc)
# audit:   ports 8072 (http), 9072 (grpc)
```

## Step 8: Ingress

```yaml
# ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: ggid-ingress
  namespace: ggid
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/proxy-body-size: 10m
    nginx.ingress.kubernetes.io/proxy-read-timeout: "60"
spec:
  tls:
  - hosts:
    - api.ggid.example.com
    secretName: ggid-tls
  rules:
  - host: api.ggid.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: gateway
            port:
              number: 8080
```

## Step 9: HPA (Horizontal Pod Autoscaler)

```yaml
# hpa.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: gateway-hpa
  namespace: ggid
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: gateway
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

## Step 10: Deploy Everything

```bash
# Apply all resources
kubectl apply -f namespace.yaml
kubectl apply -f configmap.yaml
kubectl apply -f postgres.yaml
kubectl apply -f redis.yaml
kubectl apply -f nats.yaml
kubectl apply -f services.yaml
kubectl apply -f ingress.yaml
kubectl apply -f hpa.yaml

# Wait for pods to be ready
kubectl wait --for=condition=ready pod -l app -n ggid --timeout=300s

# Check status
kubectl get pods -n ggid
kubectl get svc -n ggid
```

## Step 11: Verify

```bash
# Port-forward gateway for testing
kubectl port-forward svc/gateway 8080:8080 -n ggid

# Test healthz
curl http://localhost:8080/healthz
# Expected: {"status":"ok"}

# Test register
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{"username":"testuser","email":"test@example.com","password":"Test123!"}'

# Test login
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{"username":"testuser","password":"Test123!"}'
```

## Troubleshooting

| Issue | Command | Fix |
|-------|---------|-----|
| Pod CrashLoopBackOff | `kubectl logs <pod> -n ggid` | Check env vars / DB connection |
| DB connection refused | `kubectl exec -it postgres-0 -- psql -U ggid -c '\l'` | Check DB_HOST, password |
| Redis auth failed | `kubectl logs redis-xxx -n ggid` | Verify REDIS_PASSWORD matches |
| NATS unhealthy | `curl http://nats:8222/healthz` | Check store_dir writable |
| Ingress 502 | `kubectl describe ingress ggid-ingress -n ggid` | Gateway not ready |

## See Also

- [Docker Deployment](../guides/docker-deployment.md)
- [Production Checklist](../guides/production-checklist.md)
- [Multi-Region Deployment](../research/multi-region.md)
- [Performance Tuning](../guides/performance-tuning.md)
