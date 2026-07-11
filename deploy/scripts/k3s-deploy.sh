#!/bin/bash
# GGID K3s deployment — lightweight, uses infra already running in ggid namespace
set -e
export KUBECONFIG=~/.kube/config.k3s
NS=ggid

# RSA keys as secret
kubectl create secret generic ggid-rsa-keys -n $NS \
  --from-file=rsa_public.pem=/tmp/rsa_public.pem \
  --from-file=rsa_private.pem=/tmp/rsa_private.pem \
  --dry-run=client -o yaml | kubectl apply -f -

# JWT secret
kubectl create secret generic ggid-jwt-secret -n $NS \
  --from-literal=secret=k3s-jwt-dev-secret \
  --dry-run=client -o yaml | kubectl apply -f -

# DB password secret
kubectl create secret generic ggid-db-secret -n $NS \
  --from-literal=password=ggid-k3s \
  --dry-run=client -o yaml | kubectl apply -f -

# Shared env for all Go services
COMMON_ENV=$(cat <<'EOF'
        - {name: DB_HOST, value: "ggid-postgresql"}
        - {name: DB_PORT, value: "5432"}
        - {name: DB_USER, value: "ggid"}
        - {name: DB_PASSWORD, value: "ggid-k3s"}
        - {name: DB_NAME, value: "ggid"}
        - {name: REDIS_URL, value: "redis://ggid-redis:6379"}
        - {name: NATS_URL, value: "nats://ggid-nats:4222"}
        - {name: JWT_SECRET, value: "k3s-jwt-dev-secret"}
        - {name: JWT_PUBLIC_KEY_PATH, value: "/configs/rsa_public.pem"}
        - {name: LOG_LEVEL, value: "info"}
        - {name: DEFAULT_TENANT_ID, value: "00000000-0000-0000-0000-000000000001"}
EOF
)

RSA_VOLUME='- name: rsa-keys
          secret:
            secretName: ggid-rsa-keys'
RSA_MOUNT='- name: rsa-keys
            mountPath: /configs
            readOnly: true'

RESOURCES='resources:
          limits:
            cpu: 200m
            memory: 128Mi
          requests:
            cpu: 50m
            memory: 64Mi'

PROBE='livenessProbe:
          httpGet:
            path: /healthz
            port: http
          initialDelaySeconds: 15
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /healthz
            port: http
          initialDelaySeconds: 5
          periodSeconds: 5'

# Function to generate deployment + service YAML
gen_service() {
  local name=$1 port=$2 grpc_port=$3
  local svc_port_name="http"

  cat <<EOF
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ggid-${name}
  namespace: ggid
spec:
  replicas: 1
  selector:
    matchLabels: {app: ggid-${name}}
  template:
    metadata:
      labels: {app: ggid-${name}}
    spec:
      volumes:
${RSA_VOLUME}
      containers:
      - name: ${name}
        image: registry.iot2.win/ggid/${name}:latest
        imagePullPolicy: Always
        ports:
        - name: http
          containerPort: ${port}
$(if [ -n "$grpc_port" ]; then echo "        - name: grpc
          containerPort: ${grpc_port}"; fi)
        env:
${COMMON_ENV}
        volumeMounts:
${RSA_MOUNT}
        ${PROBE}
        ${RESOURCES}
---
apiVersion: v1
kind: Service
metadata:
  name: ggid-${name}
  namespace: ggid
spec:
  selector: {app: ggid-${name}}
  ports:
  - {name: http, port: ${port}, targetPort: ${port}}
$(if [ -n "$grpc_port" ]; then echo "  - {name: grpc, port: ${grpc_port}, targetPort: ${grpc_port}}"; fi)
EOF
}

# Generate all services
echo "# GGID K3s manifests" > /tmp/ggid-k8s.yaml

gen_service gateway 8080 >> /tmp/ggid-k8s.yaml
gen_service identity 8080 50051 >> /tmp/ggid-k8s.yaml
gen_service auth 9001 50052 >> /tmp/ggid-k8s.yaml
gen_service oauth 9005 >> /tmp/ggid-k8s.yaml
gen_service policy 8070 9070 >> /tmp/ggid-k8s.yaml
gen_service org 8071 9071 >> /tmp/ggid-k8s.yaml
gen_service audit 8072 9072 >> /tmp/ggid-k8s.yaml

# Patch gateway service to NodePort
cat <<EOF >> /tmp/ggid-k8s.yaml
---
apiVersion: v1
kind: Service
metadata:
  name: ggid-gateway
  namespace: ggid
spec:
  type: NodePort
  selector: {app: ggid-gateway}
  ports:
  - {name: http, port: 8080, targetPort: 8080, nodePort: 30080}
EOF

echo "Generated $(grep -c 'kind: Deployment' /tmp/ggid-k8s.yaml) deployments"
echo "Generated $(grep -c 'kind: Service' /tmp/ggid-k8s.yaml) services"
