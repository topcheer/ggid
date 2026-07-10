# Production Security Checklist

> Pre-deployment security audit checklist for GGID production environments.

---

## TLS / Network Security

- [ ] **TLS 1.3 enforced** — Disable TLS 1.0/1.1/1.2 at load balancer level
- [ ] **HSTS header** — `Strict-Transport-Security: max-age=31536000; includeSubDomains; preload`
- [ ] **Certificate management** — Use cert-manager (Let's Encrypt) or internal CA
- [ ] **mTLS between services** — Optional via Istio/Linkerd service mesh
- [ ] **Database TLS** — `sslmode=verify-full` with CA certificate
- [ ] **Redis TLS** — Enable `rediss://` protocol for encrypted connections
- [ ] **NATS TLS** — Configure server certificate verification
- [ ] **LDAP STARTTLS/LDAPS** — Never use plaintext LDAP in production

```nginx
# Nginx Ingress TLS configuration
ssl_protocols TLSv1.3;
ssl_prefer_server_ciphers off;
ssl_session_cache shared:SSL:10m;
ssl_session_timeout 10m;
add_header Strict-Transport-Security "max-age=31536000; includeSubDomains; preload" always;
```

---

## HTTP Security Headers

- [ ] **Content-Security-Policy** — Restrict script/style/img sources
- [ ] **X-Frame-Options** — `DENY` to prevent clickjacking
- [ ] **X-Content-Type-Options** — `nosniff`
- [ ] **Referrer-Policy** — `strict-origin-when-cross-origin`
- [ ] **Permissions-Policy** — Disable unused browser features

```yaml
# Gateway middleware headers
security_headers:
  Content-Security-Policy: "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; font-src 'self';"
  X-Frame-Options: "DENY"
  X-Content-Type-Options: "nosniff"
  Referrer-Policy: "strict-origin-when-cross-origin"
  Permissions-Policy: "geolocation=(), microphone=(), camera=()"
```

---

## CORS Configuration

- [ ] **Restrictive origins** — Never use `*` in production
- [ ] **Explicit methods** — Only allow needed HTTP methods
- [ ] **Credential support** — `AllowCredentials: true` with specific origins
- [ ] **Preflight caching** — Set `MaxAge: 86400`

```go
corsConfig := cors.Config{
    AllowOrigins:     []string{"https://console.example.com"},
    AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
    AllowHeaders:     []string{"Authorization", "Content-Type", "X-Tenant-ID"},
    ExposeHeaders:    []string{"X-Request-ID"},
    AllowCredentials: true,
    MaxAge:           24 * time.Hour,
}
```

---

## Secret Management

- [ ] **No secrets in env files** — Use Vault, AWS Secrets Manager, or Sealed Secrets
- [ ] **Database password** — Stored in external secret, rotated quarterly
- [ ] **JWT signing keys** — RS256 private key in Vault/KMS, never in image
- [ ] **Redis password** — Required in production (`requirepass`)
- [ ] **NATS credentials** — Auth enabled with user/password or JWT
- [ ] **LDAP bind password** — Stored in sealed secret
- [ ] **Webhook signing secrets** — Per-tenant, stored in DB encrypted
- [ ] **API keys** — Hashed at rest, never logged

```bash
# External Secrets Operator with Vault
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: ggid-db-credentials
spec:
  secretStoreRef:
    name: vault-backend
  target:
    name: postgres-credentials
  data:
    - secretKey: password
      remoteRef:
        key: ggid/prod/postgres
        property: password
```

### Key Rotation Schedule

| Secret | Rotation Period | Method |
|--------|----------------|--------|
| JWT signing keys | 90 days | JWKS key rotation with overlap window |
| Database password | Quarterly | Vault rotation + service restart |
| Redis password | Annually | Config + rolling restart |
| API keys | Per-policy (default 365d) | Admin rotation endpoint |
| TLS certificates | 90 days (Let's Encrypt) | cert-manager auto-renewal |
| SAML certificates | Annually | Manual rotation |

---

## RBAC & Least Privilege

- [ ] **No superuser for application** — Create dedicated `ggid_app` role
- [ ] **Database grants** — Only `SELECT, INSERT, UPDATE, DELETE` on needed tables
- [ ] **Schema access** — Restrict to `public` schema only
- [ ] **Service accounts** — Each service has its own DB user with minimal grants

```sql
-- Create restricted application role
CREATE ROLE ggid_app WITH LOGIN PASSWORD 'secure-password';
GRANT CONNECT ON DATABASE ggid TO ggid_app;
GRANT USAGE ON SCHEMA public TO ggid_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO ggid_app;

-- Do NOT grant: CREATE, DROP, ALTER, TRUNCATE, REFERENCES, TRIGGER
-- RLS applies to ggid_app (not superuser)
```

- [ ] **Admin role** — Separate `ggid_admin` with BYPASSRLS for migrations only
- [ ] **API scopes** — Token scopes match exact permission needs

---

## Audit & Logging

- [ ] **Audit logging enabled** — NATS JetStream consumer running
- [ ] **Audit table append-only** — No UPDATE/DELETE grants on `audit_events`
- [ ] **PII redaction** — Logs filter email, phone, SSN, credit card
- [ ] **Structured JSON logs** — Machine-parseable for SIEM ingestion
- [ ] **Request ID tracing** — Every request has unique `X-Request-ID`
- [ ] **Log retention** — 90 days hot, 1 year cold storage
- [ ] **SIEM forwarding** — Splunk/Sentinel integration active

```bash
# Verify audit is working
curl $API/api/v1/audit/events?limit=5 \
  -H "Authorization: Bearer $TOKEN"

# Should return recent events
# Check NATS consumer health
nats consumer info AUDIT_EVENTS audit-consumer
```

---

## Password Policy

- [ ] **Minimum length** — 12+ characters enforced
- [ ] **Complexity rules** — Upper, lower, digit, special required
- [ ] **Password history** — Last 5+ passwords checked
- [ ] **Max age** — 90 days (configurable per NIST guidance)
- [ ] **Breach detection** — HIBP k-anonymity API enabled
- [ ] **bcrypt cost** — 12 or higher

```bash
# Verify password policy
curl $API/api/v1/settings/password-policy \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

---

## Rate Limiting & Brute Force Protection

- [ ] **Per-IP rate limiting** — 60 req/min default for API endpoints
- [ ] **Login rate limiting** — 10 attempts/min per IP + username
- [ ] **Account lockout** — Lock after 5 failed attempts (15-min TTL)
- [ ] **Register rate limiting** — 5/min per IP
- [ ] **Refresh token rate limiting** — 30/min per token
- [ ] **SCIM rate limiting** — 100/min per API key
- [ ] **Progressive delays** — Exponential backoff on repeated failures

```bash
# Verify rate limiting is active
for i in $(seq 1 15); do
  curl -s -o /dev/null -w "%{http_code} " \
    -X POST $API/api/v1/auth/login \
    -H "Content-Type: application/json" \
    -d '{"username":"test","password":"wrong"}'
done
# Expected: 401 401 401 401 401 429 429 429 ...
```

---

## Session Security

- [ ] **Short-lived access tokens** — 15-minute expiry
- [ ] **Refresh token rotation** — One-time use, rotated on each refresh
- [ ] **Session revocation** — Redis-backed, immediate invalidation
- [ ] **Concurrent session limits** — Max 5 active sessions per user
- [ ] **Idle timeout** — 30-minute inactivity → session revoked
- [ ] **Secure cookies** — `HttpOnly`, `Secure`, `SameSite=Strict`

---

## Container & Infrastructure

- [ ] **Non-root containers** — `runAsNonRoot: true`, `runAsUser: 65532`
- [ ] **Read-only root filesystem** — `readOnlyRootFilesystem: true`
- [ ] **Resource limits** — CPU and memory limits on all containers
- [ ] **Network policies** — Default deny, explicit allow rules
- [ ] **Pod Security Standards** — `restricted` profile enforced
- [ ] **Image scanning** — Trivy/Grype in CI pipeline
- [ ] **Image signing** — Cosign verification before deployment

```yaml
podSecurityContext:
  runAsNonRoot: true
  runAsUser: 65532
  fsGroup: 65532
  seccompProfile:
    type: RuntimeDefault

containerSecurityContext:
  allowPrivilegeEscalation: false
  readOnlyRootFilesystem: true
  capabilities:
    drop: ["ALL"]
```

---

## Dependency Security

- [ ] **Dependency scanning** — `govulncheck` in CI
- [ ] **License audit** — Only Apache 2.0 / MIT compatible licenses
- [ ] **Go version** — Latest stable (1.25+)
- [ ] **Base images** — Distroless or Alpine, not full Debian for production
- [ ] **Regular updates** — Monthly dependency review cycle

```bash
# Run vulnerability check
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...

# Scan container image
trivy image ghcr.io/ggid/gateway:latest
```

---

## Incident Readiness

- [ ] **On-call rotation** — Defined and documented
- [ ] **Runbooks** — Available for common incidents (login outage, DB failover)
- [ ] **Alerting rules** — Configured in Prometheus/Grafana
- [ ] **Backup verification** — Quarterly restore drill
- [ ] **Penetration testing** — Annual third-party assessment

---

## Final Sign-off

| Category | Checked By | Date | Notes |
|----------|-----------|------|-------|
| TLS/Network | __________ | ______ | ______ |
| HTTP Headers | __________ | ______ | ______ |
| Secret Management | __________ | ______ | ______ |
| RBAC | __________ | ______ | ______ |
| Audit/Logging | __________ | ______ | ______ |
| Password Policy | __________ | ______ | ______ |
| Rate Limiting | __________ | ______ | ______ |
| Container Security | __________ | ______ | ______ |
| Dependency Security | __________ | ______ | ______ |

---

## References

- [Security Whitepaper](./security-whitepaper.md) — Threat model (STRIDE)
- [Security Hardening](./security-hardening.md) — Hardening guide
- [Password Policy](./password-policy.md) — Policy configuration
- [Rate Limiting](./rate-limiting.md) — Rate limit details
- [Compliance Checklist](./compliance-checklist.md) — GDPR/SOC2/ISO27001
