# Identity Threat Detection and Response (ITDR)

This guide covers the identity threat landscape, detection architecture, response automation, MITRE ATT&CK mapping, and GGID's ITDR implementation.

## Threat Landscape for Identity

| Threat | Description | Attack Vector |
|---|---|---|
| Credential theft | Stealing passwords/tokens | Phishing, breach, malware |
| Token hijacking | Intercepting active tokens | XSS, network MITM, log leakage |
| Session fixation | Forcing session ID | URL injection, cookie forcing |
| MFA bypass | Circumventing MFA | Fatigue attack, SIM swap, downgrade |
| SSO abuse | Exploiting federation | Token replay, IdP compromise |
| Account takeover | Gaining control of account | Credential stuffing, password reset abuse |
| Privilege escalation | Gaining higher access | Role abuse, policy bypass |
| Identity spoofing | Impersonating another user | JWT forgery, SAML assertion manipulation |

## Detection Architecture

### Signals + Correlation + Scoring

```
Signals → Correlation Engine → Risk Score → Response
```

### Detection Signals

| Signal | Source | Threat Indicator |
|---|---|---|
| Failed login spike | Audit log | Credential stuffing |
| Impossible travel | IP geolocation | Account compromise |
| New device | Device fingerprint | Unauthorized access |
| MFA denial streak | Auth service | MFA fatigue attack |
| Token reuse | OAuth service | Token theft |
| Off-hours admin | Audit log | Insider threat |
| Mass data export | Audit log | Data exfiltration |
| Role self-assignment | Policy service | Privilege escalation |
| JWT algorithm change | Gateway | Token forgery attempt |
| Session IP change | Session store | Session hijacking |

### Correlation Engine

```go
func correlateSignals(signals []Signal) float64 {
    score := 0.0
    // Multiple signals from same user = higher confidence
    if countSignals(signals, "user") >= 2 { score += 0.2 }
    // Impossible travel + new device = very high
    if hasSignal(signals, "impossible_travel") && hasSignal(signals, "new_device") { score += 0.4 }
    // Failed logins + MFA denial = credential stuffing + fatigue
    if hasSignal(signals, "failed_login_spike") && hasSignal(signals, "mfa_denial_streak") { score += 0.3 }
    return min(score, 1.0)
}
```

## Response Automation

| Risk Score | Response | Action |
|---|---|---|
| >0.85 | Critical | Lock account + revoke sessions + alert CISO |
| 0.7-0.85 | High | Step-up auth + alert security team |
| 0.5-0.7 | Medium | Challenge MFA + log + monitor |
| 0.3-0.5 | Low | Log only |
| <0.3 | Normal | No action |

## MITRE ATT&CK Identity Techniques Mapping

| MITRE Technique | ID | GGID Detection | GGID Mitigation |
|---|---|---|---|
| Credential Stuffing | T1110.004 | Failed login spike detection | Rate limiting + HIBP + adaptive MFA |
| Token Impersonation/Theft | T1528 | Token reuse detection | DPoP binding + refresh rotation |
| MFA Fatigue | T1621 | Push velocity detection | Number matching + rate limit pushes |
| Session Cookie Theft | T1539 | Session IP change detection | Session binding + short lifetime |
| Adversary-in-the-Middle | T1557 | TLS fingerprint anomaly | mTLS + cert pinning + HSTS |
| Social Engineering | T1566 | Impossible travel + new device | Adaptive MFA + step-up auth |
| Modify Authentication Process | T1556 | Config change detection | Config change audit + approval |
| OS Credential Dumping | T1003 | Impossible travel after credential use | Breached password check + password pepper |
| Replay Attacks | T1550 | Token jti tracking + nonce | One-time tokens + jti blacklist |
| Brute Force | T1110 | Failed login velocity | Account lockout + rate limiting |

## GGID ITDR Implementation

```yaml
itdr:
  enabled: true
  detection:
    real_time: true
    signals:
      - failed_login_spike
      - impossible_travel
      - new_device
      - mfa_fatigue
      - token_reuse
      - off_hours_admin
      - mass_data_export
      - session_hijack
    correlation: true
    scoring: true
  response:
    auto_lock: true
    auto_step_up: true
    auto_revoke: true
    alert_security: true
    alert_ciso: true
  mitre_mapping: true
  audit: true
  siem_forward: true
```

## Best Practices

1. **Correlate multiple signals** — Single signal may be false positive
2. **Automate critical response** — Lock first, investigate second
3. **Map to MITRE ATT&CK** — Use standard threat taxonomy
4. **Monitor in real-time** — Identity threats move fast
5. **Integrate with SIEM** — Correlate with broader security events
6. **Test detection rules** — Verify with simulated attacks
7. **Update threat intelligence** — New attack techniques emerge
8. **Balance security and UX** — Don't lock legitimate users
9. **Audit all responses** — Track what was detected and what action was taken
10. **Conduct purple team exercises** — Red + blue team testing