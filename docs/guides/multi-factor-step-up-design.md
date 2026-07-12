# Multi-Factor Step-Up Design Guide

Adaptive triggers, per-risk-level factor selection, session elevation patterns, UX, accessibility, and fallback design.

## Design Principles

1. **Friction proportional to risk** — Low risk = no step-up; high risk = hardware key
2. **Never block without alternative** — Always offer fallback factor chain
3. **Preserve user context** — Don't force full re-auth; elevate session scope
4. **Accessible by default** — All factors have screen reader and keyboard support

## Adaptive Trigger Model

### Signal Collection

```
Request → Gateway middleware collects:
  ├── User signals: account age, recent activity, privilege level
  ├── Device signals: known device, compliance status, attestation
  ├── Location signals: geo-IP, VPN, impossible travel
  ├── Network signals: IP reputation, ASN, rate of requests
  └── Behavioral signals: typing pattern, session duration anomaly
```

### Risk Score → Factor Matrix

| Risk Score | Risk Level | Required Factor | UX Impact |
|-----------|-----------|----------------|-----------|
| 0-19 | Minimal | None | Seamless |
| 20-39 | Low | TOTP (remember 30 days) | 10 sec |
| 40-59 | Medium | TOTP (no remember) | 10 sec |
| 60-79 | High | WebAuthn | 3 sec (biometric) |
| 80-100 | Critical | WebAuthn + approval | 30 sec + wait |

### Dynamic Trigger Examples

```go
func evaluateStepUp(signals RiskSignals) StepUpDecision {
    score := calculateRiskScore(signals)
    
    switch {
    case score < 20:
        return StepUpDecision{Required: false}
    
    case score < 60:
        if signals.DeviceTrusted && signals.RememberStepUp {
            return StepUpDecision{Required: false}  // Trusted device
        }
        return StepUpDecision{Factor: "totp", TTL: 600}
    
    case score < 80:
        return StepUpDecision{Factor: "webauthn", TTL: 300}
    
    default:
        return StepUpDecision{
            Factor: "webauthn", 
            RequireApproval: true,
            TTL: 300,
            Notify: "security@corp.com",
        }
    }
}
```

## Session Elevation vs Re-Authentication

### Session Elevation (Preferred)

```
User has active session (scope: users:read)
  → Clicks "Delete User" (needs scope: users:delete)
  → Gateway: scope mismatch → step-up challenge
  → User completes TOTP
  → New JWT issued with expanded scope (users:read + users:delete)
  → Original session preserved, no re-login needed
```

```bash
# Step 1: Gateway detects missing scope
GET /api/v1/users/{id}/delete-check
# → 403 {"error":"step_up_required","factor":"totp","challenge":"temp-token"}

# Step 2: Client presents step-up challenge
POST /api/v1/auth/step-up
{"challenge":"temp-token","factor":"totp","code":"123456"}
# → 200 {"elevated_token":"eyJ...","expires_in":600,"added_scopes":["users:delete"]}

# Step 3: Client uses elevated token for the operation
DELETE /api/v1/users/{id}
Authorization: Bearer <elevated-token>
```

### Re-Authentication (Fallback)

When elevation fails or factor is unavailable:
- User redirected to full login flow
- After login, redirected back to the action they were attempting
- Session is fresh (no inherited context)

### When to Use Which

| Scenario | Approach |
|----------|----------|
| Missing scope for action | Session elevation |
| Session expired | Re-authentication |
| Risk score spiked | Re-authentication + risk assessment |
| Device changed | Re-authentication + device registration |
| Admin operation | Elevation + approval workflow |

## Elevation Token Lifecycle

```
Base session (8h TTL)
  └── Elevation token (10 min TTL, scope: users:delete)
       └── Auto-expire after 10 min
       └── Re-elevation requires fresh step-up
```

Elevation is always time-bounded. Even on a trusted device, elevated scopes expire.

## Per-Risk-Level Factor Selection

| Action Category | Example Actions | Required Factor |
|----------------|----------------|----------------|
| Read (self) | View profile, dashboard | None |
| Read (others) | List users, view roles | None |
| Write (self) | Change own password, email | TOTP |
| Write (others) | Create user, assign role | TOTP |
| Delete | Remove user, delete org | WebAuthn |
| Security config | Rotate keys, change MFA policy | WebAuthn + approval |
| Break-glass | Force access, override policy | WebAuthn + dual approval |

## UX Patterns

### Inline Challenge (Preferred)

```typescript
// User clicks delete → inline modal appears
async function deleteUser(userId: string) {
  try {
    await api.delete(`/users/${userId}`);
  } catch (err) {
    if (err.requiresStepUp) {
      const code = await showTOTPModal(); // Inline, no page reload
      const { elevatedToken } = await api.stepUp(err.challenge, 'totp', code);
      await api.delete(`/users/${userId}`, { token: elevatedToken });
    }
  }
}
```

### Progressive Disclosure

1. Don't mention MFA until step-up is actually needed
2. Show inline challenge only when required
3. "Remember this device for 30 days" checkbox (for low-risk step-ups)

### Visual Risk Indicators

| Level | Indicator |
|-------|-----------|
| Normal | No special UI |
| Step-up pending | Amber banner: "Additional verification required" |
| High risk | Red banner: "Security verification required" |
| Critical | Full-screen modal: "Admin approval needed" |

## Accessibility

| Requirement | Implementation |
|------------|---------------|
| Screen reader | ARIA labels on all MFA inputs |
| Keyboard nav | Tab through code inputs, Enter to submit |
| Voice input | All factors operable via voice control |
| Color blindness | Icons + text (not color alone) for risk levels |
| Cognitive | Clear instructions, no timeout pressure (extend on activity) |
| Motor | Large tap targets (min 44x44px), no precise drag required |

## Fallback Chain Design

```
Preferred factor available?
  ├── Yes → Use it
  └── No → Fall back:
        ├── WebAuthn not available → TOTP
        ├── TOTP not enrolled → Email OTP (6-digit, 5 min TTL)
        ├── Email unavailable → SMS OTP (last resort)
        └── All exhausted → Admin-assisted recovery
```

### Fallback Rules

- Never fall back to password-only for high-risk operations
- Each fallback step logs the factor used + reason
- SMS fallback triggers additional rate limiting
- Admin-assisted recovery requires identity verification

## Monitoring

| Metric | Target | Alert |
|--------|--------|-------|
| Step-up completion rate | >90% | <80% → UX issue |
| Step-up abandonment | <10% | >20% → investigate friction |
| Fallback usage | <5% | >15% → device enrollment campaign |
| Time to complete step-up | <15s median | >30s → simplify UX |
| Elevation token reuse | 0 | Any → possible token theft |

## See Also

- [Multi-Factor Step-Up](multi-factor-step-up.md)
- [MFA Architecture](mfa-architecture.md)
- [Conditional Access](conditional-access.md)
- [Session Security](session-security.md)
- [Passwordless Auth Architecture](passwordless-auth-architecture.md)
