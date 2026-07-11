# Security Hardening Guide

> Practical guide to hardening GGID for production: password policy, MFA enforcement, rate limiting, CORS, SSRF protection, and audit retention.

---

## 1. Password Policy

### Configuration

```bash
# Auth service environment
PASSWORD_MIN_LENGTH=12         # Minimum password length (default: 8)
PASSWORD_BCRYPT_COST=14        # bcrypt cost factor (default: 12, higher = slower)
PASSWORD_PEPPER=random-32-char-string  # Server-side pepper (recommended)
```

### Policy Enforcement

Passwords must meet ALL of:
- Minimum 12 characters
- At least 1 uppercase letter
- At least 1 lowercase letter
- At least 1 digit
- At least 1 special character
- Not in breach database (planned — HaveIBeenPwned API)

### Enforce Per-Role

```json
{
  "rule_name": "admin_strong_password",
  "condition": "user.role = 'admin'",
  "action": "REQUIRE_PASSWORD_STRENGTH",
  "params": { "min_length": 16, "require_special": true }
}
```

---

## 2. MFA Enforcement

### Per-Role MFA Requirement

Use ABAC rules to require MFA for privileged roles:

```bash
curl -X POST http://localhost:8080/api/v1/policies \
  -H "Authorization: Bearer $ADMIN_JWT" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{
    "name": "admin_mfa_required",
    "effect": "deny",
    "conditions": "user.role = admin AND user.mfa_enrolled = false"
  }'
```

### Recommended MFA Methods (by security level)

| Method | Security | User Friction | Recommendation |
|--------|----------|--------------|----------------|
| WebAuthn/Passkey | Highest | Low | Primary for admins |
| TOTP (Google Authenticator) | High | Low | Primary for users |
| Email OTP | Medium | Medium | Fallback |
| SMS OTP | Low | Medium | Not recommended (SIM swap) |

---

## 3. Rate Limiting Tuning

### Production Tiers

```bash
# Gateway environment variables
RATE_LIMIT_ENABLED=true
RATE_LIMIT_RPS=100              # Requests per second per IP
RATE_LIMIT_BURST=200            # Burst capacity
```

### Per-Endpoint Tuning

| Endpoint Category | RPS | Burst | Rationale |
|-------------------|-----|-------|----------|
| Auth login | 5 | 10 | Brute force protection |
| Auth register | 3 | 5 | Account creation abuse |
| Password reset | 2 | 5 | Reset flood protection |
| API read (GET) | 200 | 500 | Normal usage |
| API write (POST/PUT) | 50 | 100 | Moderate |
| API delete (DELETE) | 10 | 20 | Destructive operations |

---

## 4. CORS Policy

### Production Configuration

```bash
# Only allow your known frontend origins
CORS_ALLOWED_ORIGINS=https://console.example.com,https://admin.example.com
```

### Never Use `*` in Production

```bash
# BAD — allows any origin
CORS_ALLOWED_ORIGINS=*

# GOOD — explicit origins
CORS_ALLOWED_ORIGINS=https://console.example.com
```

### Preflight Cache

```
Access-Control-Max-Age: 600  # Cache preflight for 10 minutes
```

---

## 5. IP Allowlisting

### Admin API IP Restriction

Restrict admin endpoints to known office/VPN IPs:

```bash
# Nginx
location /api/v1/admin/ {
    allow 10.0.0.0/8;          # Corporate network
    allow 203.0.113.50/32;     # VPN exit IP
    deny all;
    proxy_pass http://ggid_gateway;
}
```

### K8s NetworkPolicy

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: admin-api-restrict
spec:
  podSelector:
    matchLabels:
      app: ggid-gateway
  ingress:
  - from:
    - ipBlock:
        cidr: 10.0.0.0/8
    ports:
    - protocol: TCP
      port: 8080
```

---

## 6. Audit Log Retention

### Configuration

```bash
# Audit service
AUDIT_RETENTION_DAYS=90    # Default retention
```

### By Compliance Requirement

| Regulation | Minimum Retention | Setting |
|-------------|-------------------|--------|
| SOX (US) | 7 years | `AUDIT_RETENTION_DAYS=2555` |
| HIPAA (US) | 6 years | `AUDIT_RETENTION_DAYS=2190` |
| GDPR (EU) | Minimal (purpose limitation) | `AUDIT_RETENTION_DAYS=90` |
| PCI-DSS | 1 year | `AUDIT_RETENTION_DAYS=365` |
| ISO 27001 | 12 months recommended | `AUDIT_RETENTION_DAYS=365` |

### SIEM Integration

Forward audit events to your SIEM for long-term storage:

```bash
curl -X POST http://localhost:8080/api/v1/webhooks \
  -d '{"url":"https://siem.example.com/events","events":["audit.*"]}'
```

---

## 7. Webhook SSRF Protection

### Built-in Protection

GGID blocks webhook delivery to private IP ranges by default:

| Range | Blocked |
|-------|--------|
| `10.0.0.0/8` | Yes |
| `172.16.0.0/12` | Yes |
| `192.168.0.0/16` | Yes |
| `127.0.0.0/8` (loopback) | Yes |
| `169.254.0.0/16` (link-local) | Yes |
| `::1/128` (IPv6 loopback) | Yes |

### Verify Protection

```bash
# Should be blocked
curl -X POST http://localhost:8080/api/v1/webhooks \
  -d '{"url":"http://127.0.0.1:9090/webhook","events":["user.created"]}'
# → 400 Bad Request: webhook URL resolves to private IP
```

---

## 8. Additional Hardening

### Disable Unused Features

```bash
LDAP_URL=               # Empty = LDAP disabled
SCIM_ENABLED=false      # If not using SCIM provisioning
```

### gRPC TLS (Planned)

Internal service communication should use mTLS. Current: plaintext (acceptable on isolated networks).

### Cookie Security

```
Set-Cookie: session=...; HttpOnly; Secure; SameSite=Strict
```

---

*Last updated: 2025-07-11*