# K8s Manifests

Raw Kubernetes manifests for deploying GGID without Helm.

## Files

- `console-deployment.yaml` — Admin console deployment + service

## Usage

```bash
# Apply the manifest
kubectl apply -f deploy/k8s/console-deployment.yaml

# Check status
kubectl get pods -l app=ggid-console
kubectl get svc ggid-console
```

## Production Notes

- Set resource limits in the manifest before production use
- Configure persistent storage for PostgreSQL and Redis
- Use real secrets (not plaintext) via `kubectl create secret`
- For full deployment, use the Helm chart instead: `deploy/helm/ggid/`

## Port Forwarding (Debug)

```bash
kubectl port-forward svc/ggid-console 3000:3000
```
