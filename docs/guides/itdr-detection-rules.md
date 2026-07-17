# ITDR Detection Rules — Technical Guide

> Feature: Identity Threat Detection & Response (ITDR)
> Location: `services/audit/internal/detection/rule_kb192.go`
> Console: `/security/itdr-dashboard` and `/security/itdr-rules`

## What It Does

GGID's ITDR engine evaluates every audit event against 8 specialized detection rules mapped to MITRE ATT&CK techniques. When a rule's trigger conditions are met, a detection is created with severity, recommended response actions, and MITRE technique references.

## Detection Rules

### 1. Consent Phishing — T1098 (High)

**Trigger:** User grants risky OAuth scopes (e.g., `write:all`, `admin:read`) to an untrusted or newly registered client.

**Configurable thresholds:**
- `risky_scopes`: List of scope patterns (default: `admin:*`, `write:all`, `delete:*`)
- `untrusted_client_age_hours`: Client age threshold for "untrusted" (default: 24h)

**Response actions:** Review consent grant, revoke OAuth token, notify user, flag client for investigation.

### 2. MFA Fatigue — T1621 (High)

**Trigger:** Excessive MFA push notifications sent to a user in a short time window, indicating possible MFA bombing attack.

**Configurable thresholds:**
- `max_pushes`: Maximum pushes before alert (default: 5)
- `window_minutes`: Time window for counting (default: 10 minutes)

**Response actions:** Temporarily disable MFA push, force password reset, lock account, notify user.

### 3. Token Theft — T1528 (Critical)

**Trigger:** Access token used from an IP address or device fingerprint that differs significantly from the token issuance context, indicating possible token theft.

**Detection logic:** Compares current request IP/device fingerprint against the fingerprint recorded at token issuance. Uses a 10-minute window for previous fingerprints.

**Response actions:** Revoke token immediately, force re-authentication, flag for investigation.

### 4. Session Hijacking — T1539 (Critical)

**Trigger:** Session used from a different IP address or user agent than the original session establishment, mid-session.

**Detection logic:** Tracks IP and User-Agent at session creation, then compares against subsequent requests.

**Response actions:** Revoke session, require step-up authentication, alert user of possible compromise.

### 5. Mass Account Creation — T1136 (High)

**Trigger:** Abnormal volume of new account creation within a time window, indicating automated account creation.

**Configurable thresholds:**
- `max_accounts`: Maximum new accounts before alert (default: 10)
- `window_minutes`: Time window (default: 60 minutes)

**Response actions:** Rate limit registration, enable CAPTCHA, review new accounts, block source IP.

### 6. Federation Anomaly — T1606 (High)

**Trigger:** SAML/OIDC assertion received from an IdP that is not in the tenant's trusted federation entities list, or assertion claims differ from expected patterns.

**Detection logic:** Cross-references incoming federation assertions against `federation_entities` table. Flags entities with trust_level "pending" or not registered.

**Response actions:** Block the assertion, disable federation entity, investigate IdP compromise.

### 7. MFA Bypass — T1098.001 (Critical)

**Trigger:** User successfully authenticates without completing required MFA, despite MFA being enforced for their role or tenant.

**Detection logic:** Checks if authentication event completed without MFA when policy requires it (admin users, high-privilege roles).

**Response actions:** Revoke session, enforce immediate MFA enrollment, investigate policy bypass.

### 8. Mass Data Export — T1005 (High)

**Trigger:** Bulk data export exceeding 5x the user's historical baseline, indicating possible data exfiltration.

**Configurable thresholds:**
- `baseline_multiplier`: Multiplier over baseline (default: 5x)
- `min_records`: Minimum records to trigger (default: 100)

**Response actions:** Block export, flag account, review audit trail, notify data owner.

## Additional Detection Rules (Baseline)

Beyond the 8 KB-192 rules, GGID includes:

| Rule | MITRE | Severity | Trigger |
|------|-------|----------|---------|
| Brute Force Login | T1110 | High | >5 failed logins in 5 minutes |
| Credential Stuffing | T1110.004 | Critical | Distributed failed logins across IPs |
| Threat Intel Match | T1589 | High | Login from IP/domain in threat intel feeds |
| Impossible Travel | T1027 | Medium | Logins from impossible geographic distance |

## Rule Evaluation Pipeline

```
Audit Event (login, token_use, consent_grant, export, ...)
         ↓
   Rule Engine (evaluate all enabled rules)
         ↓
   State Store (Redis: sliding window counters)
         ↓
   Detection Created (if threshold met)
         ↓
   Response Actions (revoke, lock, alert, notify)
         ↓
   Audit Trail (logged with MITRE reference)
```

## API Endpoints

| Endpoint | Method | Purpose |
|----------|--------|--------|
| `/api/v1/audit/itdr/detections` | GET | List detections (filterable) |
| `/api/v1/audit/itdr/detections/:id` | GET | Get detection details |
| `/api/v1/audit/itdr/stats` | GET | ITDR statistics |
| `/api/v1/audit/itdr/rules` | GET | List configured rules |
| `/api/v1/audit/itdr/rules/:id` | PUT | Update rule configuration (thresholds, enabled) |

### curl Examples

```bash
TOKEN="your-jwt-token"
TENANT="00000000-0000-0000-0000-000000000001"

# List recent detections
curl -k -H 'Accept-Encoding: identity' \
  "https://ggid.iot2.win/api/v1/audit/itdr/detections?limit=20" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT"

# Get ITDR stats
curl -k -H 'Accept-Encoding: identity' \
  "https://ggid.iot2.win/api/v1/audit/itdr/stats" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT"

# List configured rules
curl -k -H 'Accept-Encoding: identity' \
  "https://ggid.iot2.win/api/v1/audit/itdr/rules" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT"

# Update MFA Fatigue rule threshold
curl -k -H 'Accept-Encoding: identity' \
  -X PUT "https://ggid.iot2.win/api/v1/audit/itdr/rules/mfa_fatigue" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"enabled":true,"threshold":{"max_pushes":3,"window_minutes":5}}'
```

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|--------|
| No detections firing | Rules disabled or thresholds too high | Check rule enabled status; lower thresholds |
| Too many false positives | Thresholds too sensitive | Increase window or count thresholds |
| Redis state errors | Redis unavailable | Check `ggid-redis` pod; rules degrade gracefully but lose state |
| Detection actions not executing | Response action handler not configured | Verify action handlers in audit service config |

## Best Practices

- **Start with defaults**: Enable all rules with default thresholds, then tune based on false positive rate.
- **Monitor severity mix**: A healthy system has mostly High detections, few Critical. Many Criticals may indicate active attack.
- **Tune per environment**: Development environments may need higher thresholds to avoid noise.
- **Correlate with threat intel**: Cross-reference ITDR detections with the Threat Intel Hub for enriched context.
- **Review weekly**: Audit the detection dashboard weekly to identify tuning opportunities.
