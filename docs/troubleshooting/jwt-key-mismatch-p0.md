# P0: JWT Key Mismatch — Root Cause & Fix

**Date**: 2026-07-19  
**Severity**: P0 (login broken — tokens invalid within seconds)  

---

## Root Cause

Both the **auth service** and **gateway service** call `ensureLocalKeyPair()` at startup (`auth/cmd/main.go:494`, `gateway/cmd/main.go:51`). This function:

1. Checks if `configs/rsa_private.pem` exists on the filesystem
2. If **missing**: generates a **new random RSA-2048 key pair** and writes both PEM files

In K8s without a shared volume/secret, each pod has its own ephemeral filesystem:
- Auth pod generates key pair **A** → signs JWTs with private key **A**
- Gateway pod generates key pair **B** → verifies JWTs with public key **B**
- **Result**: every JWT fails signature verification → 401 → token appears "invalid"

## Evidence

```go
// auth/cmd/main.go:508-514
func ensureLocalKeyPair(privateKeyPath, publicKeyPath string) error {
    if _, err := os.Stat(privateKeyPath); err == nil {
        return nil  // key exists, skip
    }
    key, err := rsa.GenerateKey(rand.Reader, 2048)  // ← GENERATES NEW RANDOM KEY
    ...
}
```

The Helm chart (`deploy/helm/ggid/templates/deployments.yaml:27-30`) already mounts:
```yaml
volumes:
  - name: rsa-keys
    secret:
      secretName: ggid-rsa-keys
```

But if the Secret `ggid-rsa-keys` was never created, the mount is empty → `ensureLocalKeyPair()` generates different keys per pod.

## Also: Key Rotation Impact

KB-329 added `StartAutoRotation()` to the OAuth service's `RotatingKeyProvider`. This rotates the signing key in-memory every 90 days. The gateway's JWKS refresh runs every 15 minutes. If rotation occurs between refreshes, there's a window where gateway has stale keys.

However, the gateway also has a local fallback public key AND JWKS refresh. The primary issue is the per-pod key generation, not rotation.

## Fix

### Immediate: Create Shared Secret

```bash
bash deploy/scripts/generate-rsa-keys.sh ggid
kubectl rollout restart deployment -n ggid
```

This script:
1. Generates a single RSA-2048 key pair using OpenSSL
2. Creates/updates K8s Secret `ggid-rsa-keys` with both PEM files
3. All pods mount the same Secret → same key pair → JWT sign/verify matches

### Permanent: Helm Chart Already Correct

The Helm chart at `deploy/helm/ggid/templates/deployments.yaml` already:
- Defines volume `rsa-keys` from Secret `ggid-rsa-keys`
- Mounts at `/configs` (readOnly)
- Sets `JWT_PUBLIC_KEY_PATH=/configs/rsa_public.pem`

The fix is simply ensuring the Secret exists before deployment.

### Verification

```bash
# Check both pods use the same public key
kubectl exec deployment/ggid-auth -n ggid -- cat /configs/rsa_public.pem | openssl rsa -pubin -modulus -noout | md5
kubectl exec deployment/ggid-gateway -n ggid -- cat /configs/rsa_public.pem | openssl rsa -pubin -modulus -noout | md5

# MD5 hashes MUST match

# Test login
TOKEN=$(curl -s -X POST https://ggid.iot2.win/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{"username":"admin","password":"Admin@123456"}' | jq -r '.access_token')

# Verify gateway accepts the token (should return 200, not 401)
curl -s -o /dev/null -w '%{http_code}' \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  https://ggid.iot2.win/api/v1/users
```

## Prevention

1. **Helm pre-install hook**: Add a Helm pre-install hook to generate keys if Secret doesn't exist
2. **Health check**: Add startup probe that verifies JWT sign→verify roundtrip
3. **Log warning**: Log when `ensureLocalKeyPair()` generates a new key (not from file)
