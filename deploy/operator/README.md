# GGID Kubernetes Operator

A Kubernetes operator for managing GGID instances via Custom Resource Definitions (CRDs).

## Architecture

- **CRD**: `GGIDInstance` custom resource
- **Controller**: Provisions and reconciles GGID deployments
- **Webhook**: Validates GGIDInstance specs

## Building

```bash
cd deploy/operator
go build -o ggid-operator ./cmd/
docker build -t ggid/operator:latest .
```

## Deployment

```bash
# Install CRD
kubectl apply -f config/crd/bases/

# Deploy the operator
kubectl apply -f config/deployment/

# Create a GGID instance
cat <<EOF | kubectl apply -f -
apiVersion: ggid.io/v1alpha1
kind: GGIDInstance
metadata:
  name: my-ggid
spec:
  version: latest
  replicas: 2
  database:
    enabled: true
EOF
```

## Development

```bash
# Run locally against a cluster
make install  # Install CRD
make run      # Run controller locally
```
