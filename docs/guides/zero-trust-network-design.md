# Zero Trust Network Design

Identity-aware proxy, microsegmentation, continuous verification, device trust signals, network access control, and BeyondCorp patterns.

## Core Principles

1. **Never trust, always verify** — Every request authenticated, regardless of network location
2. **Least privilege** — Minimal access, just-in-time elevation
3. **Assume breach** — Design as if attacker is already inside
4. **Verify explicitly** — Identity + device + context on every request

## Architecture

```
User/Service
    │
    ▼
┌──────────────────┐
│ Identity-Aware    │  Verify: identity, device, context
│ Proxy (GGID)      │  Enforce: policy, rate limit, logging
└────────┬─────────┘
         │ mTLS
         ▼
┌──────────────────┐
│ Microsegmented    │  Each service isolated
│ Services          │  Per-service auth + policy
└──────────────────┘
```

## Identity-Aware Proxy (IAP)

### Every Request

```go
func IAPMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // 1. Authenticate (JWT)
        claims, err := verifyJWT(r)
        if err != nil {
            http.Error(w, "unauthorized", 401)
            return
        }

        // 2. Check device trust
        if !isDeviceTrusted(claims, r) {
            if requireTrustedDevice(r.URL.Path) {
                http.Error(w, "untrusted device", 403)
                return
            }
        }

        // 3. Evaluate context
        risk := evaluateRisk(claims, r)
        if risk > denyThreshold {
            http.Error(w, "access denied", 403)
            return
        }
        if risk > stepUpThreshold {
            // Require step-up MFA
            w.Header().Set("X-Step-Up-Required", "true")
            return
        }

        // 4. Authorize (policy engine)
        allowed := policyEngine.Evaluate(claims, r.URL.Path, r.Method)
        if !allowed {
            http.Error(w, "forbidden", 403)
            return
        }

        next.ServeHTTP(w, r)
    })
}
```

## Microsegmentation

### Service-Level Isolation

| Segment | Allowed Inbound | mTLS Required |
|---------|----------------|---------------|
| Identity | Gateway only | ✅ |
| Auth | Gateway only | ✅ |
| Policy | Gateway, Identity, Auth | ✅ |
| Audit | All services (write) | ✅ |
| PostgreSQL | App subnet only | ✅ |
| Redis | App subnet only | ✅ |

### Network Policy (Kubernetes)

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: identity-svc-isolation
spec:
  podSelector:
    matchLabels:
      app: identity
  policyTypes: [Ingress, Egress]
  ingress:
    - from:
        - podSelector:
            matchLabels:
              app: gateway
      ports:
        - port: 8080
  egress:
    - to:
        - podSelector:
            matchLabels:
              app: postgres
      ports:
        - port: 5432
```

## Continuous Verification

### Per-Request Verification

| Check | Frequency | Latency |
|-------|-----------|---------|
| JWT signature | Every request | 0.05ms |
| Token not revoked (jti) | Every request | 0.1ms (Redis) |
| Policy evaluation | Every request | 0.3ms (cached) |
| Device trust | Every request | 0.1ms (cached) |
| Risk score | Every request | 0.5ms |

### Session Re-evaluation

```go
// Every 5 minutes, re-evaluate active sessions
func reevaluateSessions() {
    for _, session := range activeSessions {
        // Re-check risk
        if newRisk := calculateRisk(session); newRisk > session.RiskScore+20 {
            // Risk increased significantly
            requireStepUp(session)
        }
        // Re-check policy
        if !policyStillValid(session) {
            revokeSession(session)
        }
    }
}
```

## Device Trust Signals

| Signal | Source | Weight |
|--------|--------|--------|
| Device managed (MDM) | Device cert | High |
| OS up to date | Device attestation | Medium |
| Screen lock enabled | Device policy | Low |
| Disk encryption | Device attestation | Medium |
| Not jailbroken/rooted | Device attestation | High |
| Known device (seen before) | Session history | Medium |

### Device Certificate

```bash
# Device enrollment via MDM
POST /api/v1/devices/enroll
{
  "device_id": "uuid",
  "user_id": "user-uuid",
  "platform": "macOS",
  "certificate": "-----BEGIN CERTIFICATE-----...",
  "attestation": {...}
}
# → Device registered, cert used for mTLS
```

## Network Access Control (NAC)

### Access Tiers

| Tier | Trust Level | What's Accessible |
|------|------------|-------------------|
| Unauthenticated | None | Login page only |
| Authenticated (unmanaged device) | Low | Email, calendar (web only) |
| Authenticated (managed device) | Medium | Internal apps, code repos |
| Authenticated + mTLS + step-up | High | Production systems, admin |
| Break-glass (dual approval) | Max | Emergency access |

## BeyondCorp Pattern

```
                    ┌─────────────┐
                    │   User       │
                    │  + Device    │
                    └──────┬──────┘
                           │
                    ┌──────▼──────┐
                    │ Access Proxy │ ← GGID Gateway
                    │ (auth+policy)│
                    └──────┬──────┘
                           │
              ┌────────────┼────────────┐
              │            │            │
          ┌───▼───┐   ┌───▼───┐   ┌───▼───┐
          │ App A  │   │ App B  │   │ App C  │
          │(low)   │   │(medium)│   │(high)  │
          └────────┘   └────────┘   └────────┘
```

No VPN. All access through IAP. Policy decides what each user+device can reach.

## Monitoring

| Metric | Alert |
|--------|-------|
| Untrusted device access attempts | Spike → policy enforcement issue |
| Policy denials | >5% → investigate |
| Risk score elevation mid-session | Any → step-up required |
| mTLS failures | >1% → cert management issue |
| Lateral movement attempts | Any → possible breach |

## See Also

- [Conditional Access](conditional-access.md)
- [Adaptive Authentication Design](adaptive-authentication-design.md)
- [Gateway Architecture](gateway-architecture.md)
- [Identity Threat Detection](identity-threat-detection.md)
- [Service Mesh Integration](service-mesh-integration.md)
