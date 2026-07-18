# Multi-Hash Password Verifier (KB-058)

## Overview

GGID's multi-hash password verifier supports transparent migration between hashing algorithms without requiring users to reset passwords. It validates against multiple hash formats simultaneously and automatically re-hashes on successful login.

## Supported Algorithms

| Algorithm | Prefix | Use Case |
|-----------|--------|----------|
| bcrypt (cost 12) | `$2b$` | Current default |
| argon2id | `$argon2id$` | Recommended for new deployments |
| scrypt | `$scrypt$` | Legacy compatibility |
| PBKDF2 | `$pbkdf2$` | FIPS-compliant environments |
| SHA-512 (deprecated) | `$6$` | Migration-only, auto-rehashes |

## How It Works

```
User Login → Extract hash prefix → Try matching verifier → 
  Success? → Re-hash with default algo (if different) → Update DB
  Fail?    → Try next verifier → ... → Deny if all fail
```

### Transparent Re-hashing

When a user authenticates with a legacy hash (e.g., SHA-512), the verifier:
1. Validates the password against the legacy hash
2. Immediately re-hashes with the current default (argon2id)
3. Updates the database in a single transaction
4. Returns success to the user (zero latency impact)

## Configuration

```yaml
password:
  default_algorithm: argon2id
  argon2id:
    memory: 65536      # 64MB
    iterations: 3
    parallelism: 4
  bcrypt:
    cost: 12
  rehash_on_login: true
```

## API Usage

### Verify Password
```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "user-password"
}
```

Response includes `rehashed: true` if the hash was upgraded:
```json
{
  "token": "eyJ...",
  "user": { "id": "...", "email": "..." },
  "rehashed": true,
  "new_algorithm": "argon2id"
}
```

### Bulk Re-hash Check
```http
GET /api/v1/admin/passwords/hash-stats
```

Returns distribution of hash algorithms across all users:
```json
{
  "total_users": 1247,
  "by_algorithm": {
    "argon2id": 892,
    "bcrypt": 341,
    "sha512": 14
  },
  "needs_rehash": 14
}
```

## Migration Guide

1. **Deploy with multi-hash support** — no config change needed, all algorithms supported by default
2. **Set default algorithm** — configure `argon2id` as default in config
3. **Monitor rehash progress** — check `/api/v1/admin/passwords/hash-stats` periodically
4. **Users auto-migrate** — each login transparently upgrades the hash
5. **Force migration** (optional) — for dormant accounts, trigger password reset

## Security Notes

- Never store plaintext passwords
- Re-hashing happens in a goroutine after token issuance (non-blocking)
- All hash comparisons use constant-time comparison
- PBKDF2 minimum 600,000 iterations per OWASP 2024
