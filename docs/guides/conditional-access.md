# Conditional Access Policies

Guide for risk-based, signal-driven access control in GGID.

## Overview

Conditional access evaluates real-time signals at authentication time to decide whether to grant, deny, or require step-up. It bridges authentication and authorization with context.

## Signal Types

| Signal | Source | Examples |
|--------|--------|---------|
| User | Directory | Group membership, risk score, employment type |
| Device | MDM/EDR | Managed, compliant, encryption, OS version |
| Location | Geo-IP | Country, IP range, impossible travel |
| Application | Client metadata | OAuth client, app risk tier |
| Network | Gateway | VPN, corporate IP, TOR exit |
| Time | Request | Business hours, weekend, maintenance window |
| Risk | Threat intel | Known bad IP, credential leak, brute force |

## Policy Evaluation Flow

```
Request arrives
    │
    ▼
┌─────────────────────────┐
│ Collect signals         │ ← user, device, location, app, risk
│ (parallel, <50ms)       │
└───────────┬─────────────┘
            ▼
┌─────────────────────────┐
│ Evaluate policies       │ ← match conditions against signals
│ (CEL expressions)       │
└───────────┬─────────────┘
            ▼
┌─────────────────────────┐
│ Decision: Allow / Step-up / Deny │
└───────────┬─────────────┘
            ▼
┌─────────────────────────┐
│ Apply session controls  │ ← token TTL, scope, persistent vs ephemeral
└─────────────────────────┘
```

## Policy Definition

```yaml
policies:
  - name: "block-high-risk-countries"
    priority: 100
    condition: |
      location.country in ["XX", "YY"]
    decision: deny
    log: true

  - name: "require-mfa-from-new-device"
    priority: 80
    condition: |
      device.is_new == true && user.risk_score < 50
    decision: step_up
    step_up_factor: totp
    session_control:
      token_ttl: 300   # 5 min only

  - name: "corporate-vpn-full-access"
    priority: 50
    condition: |
      network.is_vpn == true && device.managed == true
    decision: allow
    session_control:
      token_ttl: 28800  # 8 hours

  - name: "default-require-totp"
    priority: 1
    condition: "true"
    decision: step_up
    step_up_factor: totp
```

### Condition Language

GGID uses CEL (Common Expression Language) for policy conditions:

```cel
// Block impossible travel
location.country != last_login.country &&
time_diff(last_login.timestamp, now) < flight_time(last_login.country, location.country)

// Require WebAuthn for admin access
app.scope.contains("admin:*") && user.risk_score > 20

// Business hours only for sensitive ops
time.hour >= 9 && time.hour < 17 && time.day_of_week in [1,2,3,4,5]
```

## Grant Controls

| Control | Effect |
|---------|--------|
| Allow | Proceed with normal session |
| Step-up: TOTP | Require TOTP code before proceeding |
| Step-up: WebAuthn | Require hardware key or platform authenticator |
| Step-up: Approval | Require manager/admin approval |
| Deny | Block access, log alert |

## Session Controls

| Control | Effect |
|---------|--------|
| Token TTL | Override default token lifetime |
| Scope reduction | Limit scopes granted this session |
| Session type | Ephemeral (no refresh token) vs persistent |
| Device binding | Bind session to device fingerprint |
| IP binding | Bind session to source IP |

## ABAC Integration

Conditional access feeds into ABAC evaluation:

```go
func evaluateAccess(ctx context.Context, req AccessRequest) Decision {
    signals := collectSignals(ctx, req)
    policy := matchPolicy(signals)

    if policy.Decision == StepUp {
        return Decision{Allow: false, Challenge: policy.StepUpFactor}
    }
    if policy.Decision == Deny {
        audit.Log("conditional_access_denied", req, signals)
        return Decision{Allow: false}
    }

    // Merge session controls into ABAC context
    return abac.Evaluate(req.WithSessionControls(policy.SessionControl))
}
```

## Real-Time Enforcement

Conditional access runs on **every** request, not just login:

```go
// Gateway middleware
func ConditionalAccessMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        signals := collectSignals(r.Context(), r)
        decision := policyEngine.Evaluate(signals)

        switch decision {
        case Deny:
            http.Error(w, "access denied", 403)
            return
        case StepUp:
            // Return 403 with step-up challenge header
            w.Header().Set("X-Step-Up-Required", decision.Factor)
            http.Error(w, "step-up required", 403)
            return
        case Allow:
            next.ServeHTTP(w, r)
        }
    })
}
```

This means a session can be terminated mid-use if conditions change (e.g., user's risk score spikes).

## Risk Score Calculation

```
risk_score =
    device_risk * 0.25 +      // managed/unmanaged, last seen
    location_risk * 0.20 +    // known country, TOR
    behavior_risk * 0.20 +    // login pattern deviation
    threat_risk * 0.20 +      // threat intel feeds
    app_risk * 0.15           // app sensitivity tier
```

| Score Range | Default Policy |
|-------------|---------------|
| 0-19 | Allow |
| 20-49 | Require TOTP |
| 50-79 | Require WebAuthn |
| 80-100 | Deny + alert |

## Monitoring

| Metric | Alert |
|--------|-------|
| Policy evaluation time | >100ms p99 |
| Deny rate | Sudden spike → possible attack or misconfigured policy |
| Step-up failure rate | >30% → UX issue or credential problem |
| Policy changes | Audit log + notify security team |

## See Also

- [MFA Architecture](mfa-architecture.md)
- [Delegated Administration](delegated-administration.md)
- [Session Security](session-security.md)
- [Adaptive Authentication](../research/adaptive-authentication.md)
