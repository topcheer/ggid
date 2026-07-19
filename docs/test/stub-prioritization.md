# STUB Prioritization Report

## Classification Criteria

- **MUST-FIX**: Core security/compliance/IAM function. Customer cannot use GGID without it.
- **SHOULD-FIX**: Important but has workaround or partial functionality.
- **ACCEPTABLE-STUB**: Advisory/analytics/nice-to-have. Mock data acceptable for demo.

## MUST-FIX (7) — Core functionality

| # | Feature | File | Issue | Impact |
|---|---------|------|-------|--------|
| 1 | **Audit Events** | All services | Events NOT written to DB after operations | Compliance failure, no traceability |
| 2 | **JWT Key Sync** | auth ↔ gateway | Token invalidated seconds after issue | All E2E flows break after login |
| 3 | **Privilege Creep Diff** | `privilege_creep_handler.go:87` | Returns placeholder diff | Security audit incomplete |
| 4 | **Batch Introspection** | `batch_introspect_handler.go:41` | Returns sample data | OAuth token validation broken |
| 5 | **Password Verify** | `bulk_import.go:203` | Placeholder bcrypt check | Import accepts invalid passwords |
| 6 | **VPN Check** | `vpn_check_handler.go:26` | Sample VPN exit nodes | Risk scoring inaccurate |
| 7 | **Standing Access** | policy service | Returns empty list | PAM compliance gap |

## SHOULD-FIX (8) — Important, partial workaround

| # | Feature | File | Issue | Impact |
|---|---------|------|-------|--------|
| 8 | **Skill Matrix** | `skill_matrix_handler.go:110` | Pre-populated sample users | IGA dashboard misleading |
| 9 | **Policy Conflicts** | `policy_conflicts_handler.go:77` | Sample conflicts when no real policies | SoD analysis unreliable |
| 10 | **Session Tracking** | auth service | Sessions not persisting | Session management broken |
| 11 | **CCM Scan** | audit service | Never run, returns 0 controls | Compliance monitoring gap |
| 12 | **Conditional Access** | auth service | Returns empty policies | CAP not enforced |
| 13 | **Attribute Mapping Test** | `attribute_mapping_repo.go:182` | Simulated test only | Mapping validation incomplete |
| 14 | **Role Mining** | policy service | Returns stub data | IGA analysis unreliable |
| 15 | **Blast Radius** | policy service | Returns mock impact | Security assessment gap |

## ACCEPTABLE-STUB (14) — Advisory/analytics, mock OK

| # | Feature | Impact | Reason |
|---|---------|--------|--------|
| 16 | Risk Engine scores | Dashboard display | Mock scores acceptable for demo |
| 17 | SOAR Playbooks | 0 playbooks | User creates their own |
| 18 | UEBA baselines | Behavioral analytics | Needs historical data first |
| 19 | NHI risk scores | Non-human identity | Niche feature |
| 20 | Threat Intel feed | IOC data | External integration |
| 21 | Impact Preview | Policy simulation | Nice-to-have preview |
| 22 | Access Path Analytics | Graph analysis | Advanced feature |
| 23 | Coverage Matrix | Policy coverage viz | Reporting only |
| 24 | DLP policies | Return empty (honest) | User configures |
| 25 | Credential Stuffing stats | Return empty (honest) | Needs attack data |
| 26 | Session Fingerprint | Returns empty (by design) | "No fake data" — correct |
| 27 | PII Discovery sample | Masking demo | Sample acceptable |
| 28 | Import preview rows | Sample valid rows | Preview only |
| 29 | ReBAC sample tuples | Relationship examples | Demo tuples OK |

## Summary

| Priority | Count | Action |
|----------|-------|--------|
| **MUST-FIX** | 7 | Block v1.0-stable release |
| **SHOULD-FIX** | 8 | Fix before GA |
| **ACCEPTABLE** | 14 | Defer to v1.1 |

## Top 3 Immediate Actions

1. **Fix audit event pipeline** — NATS consumer or direct DB write from all services
2. **Fix JWT key sync** — ensure auth + gateway pods use same RSA key
3. **Fix privilege creep + standing access** — return real DB queries, not placeholders
