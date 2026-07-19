#!/usr/bin/env bash
# Generate a shared RSA key pair and create a Kubernetes Secret.
# All GGID pods (auth, gateway, oauth) must share the same key pair
# to prevent JWT sign/verify mismatch.
#
# Usage: bash deploy/scripts/generate-rsa-keys.sh [namespace]
# Default namespace: ggid
set -euo pipefail

NS="${1:-ggid}"
KEY_DIR=$(mktemp -d)
trap "rm -rf $KEY_DIR" EXIT

echo "Generating RSA-2048 key pair..."

openssl genrsa -out "$KEY_DIR/rsa_private.pem" 2048 2>/dev/null
openssl rsa -in "$KEY_DIR/rsa_private.pem" -pubout -out "$KEY_DIR/rsa_public.pem" 2>/dev/null

echo "Creating k8s secret 'ggid-rsa-keys' in namespace '$NS'..."

kubectl create secret generic ggid-rsa-keys \
  --from-file=rsa_private.pem="$KEY_DIR/rsa_private.pem" \
  --from-file=rsa_public.pem="$KEY_DIR/rsa_public.pem" \
  -n "$NS" \
  --dry-run=client -o yaml | kubectl apply -f -

echo ""
echo "Secret created. Verifying..."
kubectl get secret ggid-rsa-keys -n "$NS" -o jsonpath='{.data}' | head -c 100
echo ""
echo ""
echo "IMPORTANT: Restart all pods to pick up the new keys:"
echo "  kubectl rollout restart deployment -n $NS"
echo ""
echo "NOTE: Existing JWTs signed with old keys will be invalid after restart."
echo "      Users will need to re-authenticate."
