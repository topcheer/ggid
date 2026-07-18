# China GM (SM2/SM3/SM4) Compliance Guide

## Overview

GGID supports Chinese national cryptographic standards (国密/GM) for organizations requiring compliance with PRC regulations. This includes SM2 (asymmetric), SM3 (hash), and SM4 (symmetric) algorithms for JWT signing, certificate generation, and data encryption.

## Supported Algorithms

| Standard | Algorithm | RSA/SHA Equivalent | Use Case |
|----------|-----------|---------------------|----------|
| SM2 | Elliptic Curve | RSA-2048 | Digital signatures, key exchange |
| SM3 | Merkle-Damgård | SHA-256 | Hashing, integrity |
| SM4 | Block Cipher | AES-128 | Data encryption |

## Configuration

### Enable GM Signing for JWT

Set the JWT signing algorithm to SM2 in your environment:

```bash
# .env
JWT_SIGNING_ALGORITHM=SM2
JWT_SIGNING_KEY_PATH=/app/configs/sm2_private.pem
JWT_PUBLIC_KEY_PATH=/app/configs/sm2_public.pem
```

### Generate SM2 Key Pair

```bash
# Using GmSSL or crypto library
gmssl genpkey -algorithm EC -pkeyopt ec_paramgen_curve:sm2p256v1 \
  -out sm2_private.pem
gmssl pkey -in sm2_private.pem -pubout -out sm2_public.pem
```

### OAuth Token Signing

```bash
# .env
OAUTH_SIGNING_ALGORITHM=SM2
OAUTH_PRIVATE_KEY_PATH=/app/configs/sm2_private.pem
OAUTH_PUBLIC_KEY_PATH=/app/configs/sm2_public.pem
```

### SAML Assertion Signing

Configure SAML IdP to use SM2/SM3 for assertion signatures:

```json
{
  "signing_algorithm": "http://www.gov.cn/gm/SM2",
  "digest_algorithm": "http://www.gov.cn/gm/SM3"
}
```

## Data Encryption

### SM4 for Data at Rest

Enable SM4 for database column encryption:

```bash
# .env
DATA_ENCRYPTION_ALGORITHM=SM4
DATA_ENCRYPTION_KEY_PATH=/app/configs/sm4_key.bin
```

### Generate SM4 Key

```bash
openssl rand -out sm4_key.bin 16  # 128-bit key
```

## API Usage

### Register User with SM2-signed Credentials

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{"email":"user@corp.cn","password":"Secure@123","username":"user"}'
```

The JWT returned will be signed with SM2 if configured.

### Verify SM2-signed JWT

```python
# Python example using gmssl library
from gmssl import sm2, func
from jose import jwt

public_key = load_sm2_public_key("sm2_public.pem")
payload = jwt.decode(token, public_key, algorithms=["SM2"])
print(payload)
```

## Compliance Framework Mapping

| Regulation | Requirement | GGID Coverage |
|-----------|-------------|---------------|
| 《密码法》 | Use certified GM algorithms | SM2/SM3/SM4 |
| 等保2.0 三级 | Identity-based access control | RBAC + ABAC + audit chain |
| GB/T 39786 | Information security tech requirements | GM signing + encryption |
| PIPL | Personal information protection | PII scanning + data residency |

## Hybrid Mode

GGID supports hybrid mode — using GM algorithms for domestic operations and RSA/SHA for international federation:

```bash
# Domestic tenant: SM2/SM3/SM4
JWT_SIGNING_ALGORITHM=SM2

# International tenant: RSA/SHA-256
# (configure per-tenant via console)
```

## Best Practices

1. Use **certified** GM crypto libraries (e.g., GmSSL, SJCL)
2. Store SM2 private keys in HSM or KMS when available
3. Rotate SM4 keys annually
4. Test interop with downstream systems before production deployment
5. Document which tenants use GM vs RSA in your deployment config

## Troubleshooting

| Issue | Solution |
|-------|----------|
| JWT verification fails with foreign SDKs | SDK must support SM2; use RSA for cross-border |
| SM2 key format mismatch | Ensure PKCS#8 encoding with SM2 curve OID |
| SM4 performance | Use hardware acceleration (Intel AES-NI equivalent for SM4) |
