# Okta & Cisco Duo 2026 Feature Analysis

> How Okta and Cisco Duo's latest features compare to GGID.

---

## Okta 2026 Highlights

### Identity Security Posture Management (ISPM)
- Continuous assessment of identity configuration
- Detects over-privileged accounts, inactive users
- Recommendations engine for privilege right-sizing

**GGID status**: Audit hash chain + IGA workflows cover parts. ISPM dashboard not yet available.

### Okta AI
- Natural language identity queries ("show inactive users")
- AI-assisted policy recommendations
- Anomaly explanation in plain language

**GGID status**: AI Agent Identity (commit 55ffd6f) enables AI agents to interact with GGID APIs, but no natural-language query UI yet.

### Device Bound SSO (GA)
- Promoted from preview to GA in 2026
- Platform authenticator required for SSO sessions

**GGID status**: Building blocks exist. 4.5-day implementation estimate.

### Okta Workforce Identity Cloud
- Unified governance lifecycle
- Access certifications (manager reviews)
- Separation of duties (SoD) policies

**GGID status**: IGA workflows (access requests) implemented. Access certifications and SoD are future P2.

---

## Cisco Duo 2026 Highlights

### Device Health Policies
- Check OS version, disk encryption, firewall status
- Block access from non-compliant devices

**GGID status**: Not implemented. Would require device posture agent (significant effort).

### Adaptive Access Policies
- Risk-based authentication (low/medium/high)
- Step-up authentication based on context
- Geographic anomaly detection

**GGID status**: ABAC engine + rate limiting provide basic adaptive access. Risk scoring and step-up auth are P2.

### Duo Network Gateway
- Zero-trust access to internal apps
- TCP/SSH/RDP proxying

**GGID status**: Out of scope (IAM, not ZTNA). GGID integrates with ZTNA providers.

---

## Gap Summary

| Feature | Okta/Duo | GGID | Priority |
|---------|----------|------|----------|
| Device health checks | Duo | No | P3 |
| Risk scoring | Okta | Rate limit only | P2 |
| Access certifications | Okta | No | P2 |
| SoD policies | Okta | No | P2 |
| NL identity queries | Okta AI | No | P3 |
| ISPM dashboard | Okta | No | P3 |
| Device-bound SSO | Okta GA | Planned (4.5d) | P1 |
| Step-up auth | Duo | Via ABAC | P2 |
| Multi-tenant RLS | No | Yes (superior) | GGID wins |
| SCIM 2.0 | Okta | Yes | GGID wins |
| ABAC | Okta (custom) | Yes (native) | GGID wins |
| Audit hash chain | No | Yes | GGID wins |

---

*See: [Competitive Analysis](competitive-analysis.md) | [Gap Closure Report](gap-closure-report.md)*

*Last updated: 2025-07-11*
