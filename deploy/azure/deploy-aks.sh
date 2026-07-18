#!/bin/bash
set -euo pipefail

# GGID Azure AKS Quick Deploy
# Prerequisites: az CLI, kubectl, helm

CLUSTER_NAME="${CLUSTER_NAME:-ggid-prod}"
RESOURCE_GROUP="${RESOURCE_GROUP:-ggid-rg}"
LOCATION="${AZURE_LOCATION:-eastus}"
NAMESPACE="ggid"

echo "=== GGID AKS Deployment ==="

# 1. Create resource group
echo "Creating resource group: $RESOURCE_GROUP..."
az group create --name "$RESOURCE_GROUP" --location "$LOCATION"

# 2. Create AKS cluster
echo "Creating AKS cluster: $CLUSTER_NAME..."
az aks create \
  --resource-group "$RESOURCE_GROUP" \
  --name "$CLUSTER_NAME" \
  --node-count 3 \
  --node-vm-size Standard_D2s_v5 \
  --enable-addons monitoring \
  --generate-ssh-keys

# 3. Get credentials
az aks get-credentials \
  --resource-group "$RESOURCE_GROUP" \
  --name "$CLUSTER_NAME"

# 4. Create namespace
kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -

# 5. Install Application Gateway ingress controller (optional)
echo "Note: For Application Gateway ingress, run:"
echo "  az aks enable-addons -g $RESOURCE_GROUP -n $CLUSTER_NAME -a ingress-appgw --appgw-name ggid-agw"

# 6. Deploy GGID via Helm
echo "Deploying GGID..."
helm install ggid ../../helm/ggid/ \
  -n "$NAMESPACE" \
  --set gateway.service.type=LoadBalancer

# 7. Get external IP
echo "Waiting for LoadBalancer..."
sleep 60
EXTERNAL_IP=$(kubectl get svc -n "$NAMESPACE" ggid-gateway -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "pending")

echo ""
echo "=== Deployment Complete ==="
echo "Gateway URL: http://$EXTERNAL_IP"
echo ""
