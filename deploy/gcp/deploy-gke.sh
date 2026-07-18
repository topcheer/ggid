#!/bin/bash
set -euo pipefail

# GGID GCP GKE Quick Deploy
# Prerequisites: gcloud CLI, kubectl, helm

CLUSTER_NAME="${CLUSTER_NAME:-ggid-prod}"
ZONE="${GCP_ZONE:-us-central1-a}"
PROJECT="${GCP_PROJECT_ID:?Set GCP_PROJECT_ID env var}"
NAMESPACE="ggid"

echo "=== GGID GKE Deployment ==="

# 1. Set project
gcloud config set project "$PROJECT"

# 2. Enable APIs
gcloud services enable container.googleapis.com sqladmin.googleapis.com

# 3. Create GKE cluster
echo "Creating GKE cluster: $CLUSTER_NAME..."
gcloud container clusters create "$CLUSTER_NAME" \
  --zone "$ZONE" \
  --num-nodes 3 \
  --machine-type e2-standard-2 \
  --enable-autoscaling \
  --min-nodes 3 \
  --max-nodes 6

# 4. Get credentials
gcloud container clusters get-credentials "$CLUSTER_NAME" --zone "$ZONE"

# 5. Create namespace
kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -

# 6. (Optional) Create Cloud SQL instance
echo "Note: For managed PostgreSQL, run:"
echo "  gcloud sql instances create ggid-db --database-version=POSTGRES_16 --zone=$ZONE"
echo "  gcloud sql databases create ggid --instance=ggid-db"

# 7. Deploy GGID via Helm
echo "Deploying GGID..."
helm install ggid ../../helm/ggid/ \
  -n "$NAMESPACE" \
  --set gateway.service.type=LoadBalancer

# 8. Get external IP
echo "Waiting for LoadBalancer..."
sleep 60
EXTERNAL_IP=$(kubectl get svc -n "$NAMESPACE" ggid-gateway -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "pending")

echo ""
echo "=== Deployment Complete ==="
echo "Gateway URL: http://$EXTERNAL_IP"
echo ""
