#!/bin/bash
set -euo pipefail

# GGID AWS EKS Quick Deploy
# Prerequisites: eksctl, kubectl, helm, aws CLI configured

CLUSTER_NAME="${CLUSTER_NAME:-ggid-prod}"
REGION="${AWS_REGION:-us-east-1}"
NAMESPACE="ggid"

echo "=== GGID EKS Deployment ==="

# 1. Create EKS cluster
echo "Creating EKS cluster: $CLUSTER_NAME..."
eksctl create cluster \
  --name "$CLUSTER_NAME" \
  --region "$REGION" \
  --version 1.29 \
  --nodegroup-name standard \
  --node-type t3.large \
  --nodes 3 \
  --nodes-min 3 \
  --nodes-max 5 \
  --managed

# 2. Install AWS Load Balancer Controller
echo "Installing AWS Load Balancer Controller..."
eksctl utils associate-iam-oidc-provider \
  --cluster "$CLUSTER_NAME" --region "$REGION" --approve

# 3. Create namespace
kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -

# 4. Deploy GGID via Helm
echo "Deploying GGID..."
helm install ggid ../../helm/ggid/ \
  -n "$NAMESPACE" \
  --set global.imageRegistry=public.ecr.aws \
  --set gateway.service.type=LoadBalancer

# 5. Wait for LoadBalancer
echo "Waiting for LoadBalancer..."
kubectl get svc -n "$NAMESPACE" ggid-gateway -w &
LB_PID=$!
sleep 60
kill $LB_PID 2>/dev/null || true

# 6. Get external IP
EXTERNAL_IP=$(kubectl get svc -n "$NAMESPACE" ggid-gateway -o jsonpath='{.status.loadBalancer.ingress[0].hostname}' 2>/dev/null || echo "pending")

echo ""
echo "=== Deployment Complete ==="
echo "Gateway URL: http://$EXTERNAL_IP"
echo "Namespace: $NAMESPACE"
echo ""
echo "Next steps:"
echo "  1. Point your DNS to: $EXTERNAL_IP"
echo "  2. Configure TLS (cert-manager + Let's Encrypt)"
echo "  3. Run database migrations"
