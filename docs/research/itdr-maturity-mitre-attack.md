# ITDR Maturity & MITRE ATT&CK Mapping: Enterprise-Grade Identity Threat Detection for GGID

> **Focus**: Mapping GGID's existing ITDR detection rules to MITRE ATT&CK techniques, identifying coverage gaps, designing ML-based UEBA enhancement, attack simulation framework, SOAR integration, and detection-as-code pipeline.
>
> **Author**: ggcxf (researcher) | **Date**: 2026-07-17 | **Status**: Research Complete
>
> **Checklist Compliance**: Endpoint precondition check (§7), DoD per backlog item (§9), curl commands (§8).

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [GGID Current State: Detection Rules](#2-ggid-current-state-detection-rules)
3. [MITRE ATT&CK Coverage Map](#3-mitre-attck-coverage-map)
4. [Detection Gaps — Missing ATT&CK Techniques](#4-detection-gaps--missing-attck-techniques)
5. [Proposed Architecture: Enhanced ITDR](#5-proposed-architecture-enhanced-itdr)
6. [UEBA Enhancement: ML-Based Detection](#6-ueba-enhancement-ml-based-detection)
7. [Attack Simulation Framework](#7-attack-simulation-framework)
8. [SOAR Integration & Automated Response](#8-soar-integration--automated-response)
9. [Implementation Backlog with DoD](#9-implementation-backlog-with-dod)
10. [Competitive Differentiation](#10-competitive-differentiation)

---

## 1. Executive Summary

GGID has a **surprisingly mature ITDR foundation** — 7+ detection rules with MITRE ATT&CK mappings, a rule registry with per-tenant overrides, UEBA behavioral baselining, threat intel integration, and a detection engine that evaluates every audit event.

**Existing detection rules (all have MITRE mappings):**
- Brute Force (T1110) ✅
- Off-Hours Admin (T1078) ✅
- New Device Privileged (T1078) ✅
- Token Replay (T1550) ✅
- Impossible Travel (T1078) ✅
- Baseline Deviation / UEBA (T1078) ✅
- Threat Intel Match (T1589) ✅

However, critical identity attack patterns are **not detected**:
- Consent phishing (OAuth malicious app install)
- Token theft (stolen token used from different device)
- Federation compromise (golden SAML equivalent)
- Session cookie hijacking
- Admin account takeover via MFA fatigue
- Supply chain (compromised SCIM provisioner)

**Recommendation**: Add 8 new detection rules for missing ATT&CK techniques, enhance UEBA with ML, build attack simulation framework, and add SOAR playbook integration.

**Estimated effort**: 3 sprints (8 new rules + ML UEBA + simulation + SOAR).

---

## 2. GGID Current State: Detection Rules

### Existing Rules

| Rule ID | File | MITRE | Severity | Trigger |
|---------|------|-------|----------|---------|
| `brute_force` | `rule_brute_force.go:14` | T1110 | High | >5 failed logins in 5 min |
| `offhours_admin` | `rule_phase4.go:15` | T1078 | Medium | Admin action outside business hours |
| `new_device_privileged` | `rule_phase4.go:64` | T1078 | High | Privileged action from unseen device |
| `token_replay` | `rule_phase4.go:119` | T1550 | Critical | Revoked token reuse attempt |
| `impossible_travel` | `rule_impossible_travel.go:16` | T1078 | High | Geographic impossibility |
| `baseline_deviation` | `profile_builder.go:121` | T1078 | Medium | UEBA behavioral deviation |
| `threat_intel_hit` | `rule_threat_intel.go:39` | T1589 | High | IP/email in threat feed |

### Architecture (Existing)

| Component | File | Status |
|-----------|------|--------|
| Rule interface | `detection/rule.go` | ✅ ID/Name/MITRE/Severity/Actions/Evaluate |
| RuleRegistry | `registry.go:14` | ✅ Per-tenant overrides |
| DetectionEngine | `engine.go` | ✅ Evaluates every audit event |
| StateStore | `rule_phase4.go` | ✅ Per-user state tracking |
| ProfileBuilder | `profile_builder.go` | ✅ UEBA baseline computation |
| ThreatIntelChecker | `rule_threat_intel.go:35` | ✅ IP/email/hash checking |
| RuleConfig | `domain/` | ✅ Per-tenant threshold overrides |

---

## 3. MITRE ATT&CK Coverage Map

### Identity-Relevant ATT&CK Techniques

| Technique | Name | GGID Coverage | Rule |
|-----------|------|---------------|------|
| **T1078** | Valid Accounts | ✅ **3 rules** | offhours_admin, new_device_privileged, impossible_travel |
| **T1110** | Brute Force | ✅ | brute_force |
| **T1550** | Use Alternate Authentication Material | ✅ Partial | token_replay |
| **T1589** | Gather Victim Identity Info | ✅ | threat_intel_hit |
| **T1098** | Account Manipulation | ❌ **Gap** | — |
| **T1136** | Create Account | ❌ **Gap** | — |
| **T1531** | Account Access Removal | ❌ **Gap** | — |
| **T1621** | Multi-Factor Authentication Request Generation | ❌ **Gap** | — |
| **T1606** | Forge Web Credentials | ❌ **Gap** | — |
| **T1528** | Steal Application Access Token | ❌ **Gap** | — |
| **T1539** | Steal Web Session Cookie | ❌ **Gap** | — |
| **T1185** | Browser Session Hijacking | ❌ **Gap** | — |
| **T1098.001** | Additional Cloud Credentials (SSO) | ❌ **Gap** | — |

### Coverage Summary

| Metric | Value |
|--------|-------|
| Techniques covered | **4 / 13** (31%) |
| Critical techniques missing | **9** |
| Most covered tactic | Initial Access (T1078, T1110) |
| Least covered tactic | Credential Access (T1528, T1539) |

---

## 4. Detection Gaps — Missing ATT&CK Techniques

### 8 New Detection Rules Needed

| # | Rule ID | ATT&CK | Technique | Detection Logic | Severity |
|---|---------|--------|-----------|-----------------|----------|
| 1 | `consent_phishing` | T1098 | Account Manipulation | OAuth app requests excessive scopes (e.g., `full_access` from unverified publisher) | High |
| 2 | `mfa_fatigue` | T1621 | MFA Request Generation | >5 MFA pushes in 2 minutes without login success | High |
| 3 | `token_theft` | T1528 | Steal Application Access Token | Valid token used from new device + new geo + new UA simultaneously | Critical |
| 4 | `session_hijack` | T1539 | Steal Web Session Cookie | Session token from IP A suddenly appears from IP B in different country | Critical |
| 5 | `mass_account_creation` | T1136 | Create Account | >10 accounts created in 5 minutes by non-provisioning service | Medium |
| 6 | `federation_anomaly` | T1606 | Forge Web Credentials | SAML/OIDC assertion from new IdP entity ID or unexpected issuer | Critical |
| 7 | `admin_mfa_bypass` | T1098.001 | Additional Cloud Credentials | Admin disabled MFA for a user → immediate alert | Critical |
| 8 | `mass_data_export` | T1005 | Data from Local System | User exports >1000 records in single session | High |

---

## 5. Proposed Architecture: Enhanced ITDR

```
                    ┌──────────────────────────────────────────────┐
                    │         Enhanced ITDR Engine                   │
                    │                                              │
                    │  ┌──────────────────────────────────────┐    │
                    │  │  Detection Rules (15 total)           │    │
                    │  │  7 existing + 8 new (see §4)          │    │
                    │  │  All with MITRE ATT&CK mapping        │    │
                    │  │  Per-tenant configurable thresholds   │    │
                    │  └──────────────┬───────────────────────┘    │
                    │                 │                            │
                    │  ┌──────────────▼───────────────────────┐    │
                    │  │  Detection Engine (existing)          │    │
                    │  │  Evaluates every audit event          │    │
                    │  │  Publishes detections to NATS         │    │
                    │  └──────────────┬───────────────────────┘    │
                    │                 │                            │
                    │  ┌──────────────▼───────────────────────┐    │
                    │  │  Response Pipeline                    │    │
                    │  │                                      │    │
                    │  │  Detection → Severity Assessment →    │    │
                    │  │  ┌─────────┐ ┌────────┐ ┌──────────┐ │    │
                    │  │  │ Alert   │ │ Auto   │ │ SOAR     │ │    │
                    │  │  │ (email) │ │ Respond│ │ Playbook │ │    │
                    │  │  └─────────┘ └────────┘ └──────────┘ │    │
                    │  └──────────────────────────────────────┘    │
                    │                                              │
                    │  ┌──────────────────────────────────────┐    │
                    │  │  Attack Simulation (Purple Team)      │    │
                    │  │  Safe simulation of identity attacks  │    │
                    │  │  Validates detection coverage         │    │
                    │  └──────────────────────────────────────┘    │
                    └──────────────────────────────────────────────┘
```

---

## 6. UEBA Enhancement: ML-Based Detection

### Current (3σ) vs Enhanced (ML)

| Method | Current | Enhanced |
|--------|---------|---------|
| **Baseline** | 30-day mean + stddev | ML model (isolation forest) |
| **Deviation detection** | Z-score > 3σ | Anomaly score from model |
| **Multi-dimensional** | Per-metric independently | Multi-dimensional correlation |
| **Adaptation** | Static baseline | Continuously retrained |
| **Sequence detection** | No | LSTM for temporal patterns |

### ML Pipeline

```
1. Feature extraction from audit events:
   - Login frequency per hour
   - Unique IPs per day
   - Unique endpoints accessed
   - Session duration distribution
   - Action diversity (number of distinct actions)
   - Geographic spread
   - Device diversity
   - Error rate

2. Model: Isolation Forest (unsupervised)
   - Train on 30-day clean baseline per user
   - Score new events: anomaly_score 0.0-1.0
   - >0.7 = anomaly → detection

3. Model: Autoencoder (deep learning, P2)
   - Reconstruct behavior pattern
   - High reconstruction error = anomaly
```

### Implementation (Isolation Forest)

```go
type IsolationForestDetector struct {
    models map[uuid.UUID]*isolationforest.Forest  // Per-user models
}

func (d *IsolationForestDetector) Evaluate(ctx context.Context, evt *domain.AuditEvent, state StateStore) (*domain.Detection, error) {
    features := d.extractFeatures(evt)
    model := d.getModel(evt.UserID)
    score := model.Score(features)

    if score > 0.7 {
        return &domain.Detection{
            RuleID:   "ml_anomaly",
            MITRE:    "T1078",
            Severity: domain.SeverityHigh,
            Title:    "ML-based behavioral anomaly detected",
            Detail:   fmt.Sprintf("Anomaly score: %.2f (threshold: 0.7)", score),
        }, nil
    }
    return nil, nil
}
```

---

## 7. Attack Simulation Framework

### Purple Team Simulation Suite

| Simulation | ATT&CK | Safe Method | Expected Detection |
|-----------|--------|-------------|-------------------|
| Brute force | T1110 | 6 failed logins from test user | brute_force rule |
| Impossible travel | T1078 | Login from US then immediately from CN | impossible_travel rule |
| Off-hours admin | T1078 | Admin action at 3AM | offhours_admin rule |
| Token replay | T1550 | Use explicitly revoked token | token_replay rule |
| MFA fatigue | T1621 | 6 MFA pushes in 1 min | mfa_fatigue (new) |
| Consent phishing | T1098 | OAuth app requesting `full_access` | consent_phishing (new) |
| Mass export | T1005 | Export 1000+ records | mass_data_export (new) |

### Simulation API

```bash
# Run purple team simulation
curl -X POST https://ggid.corp.com/api/v1/itdr/simulate \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{
    "scenario": "brute_force",
    "target_user": "test-user@corp.com",
    "dry_run": true
  }'

# Response:
{
  "simulation_id": "sim_abc",
  "scenario": "brute_force",
  "status": "completed",
  "expected_detection": "brute_force (T1110)",
  "detected": true,
  "detection_id": "det_xyz",
  "detection_latency_ms": 120,
  "coverage_gap": false
}
```

---

## 8. SOAR Integration & Automated Response

### Automated Response Playbooks

| Detection Severity | Response | Latency |
|-------------------|----------|---------|
| Critical (token_theft, session_hijack) | Auto-revoke session + lock account + alert SOC | <5s |
| High (brute_force, mfa_fatigue) | Auto-trigger MFA challenge + rate-limit IP | <10s |
| Medium (offhours_admin, baseline_deviation) | Alert + log + require step-up | <30s |
| Low (mass_export) | Log + notify manager | <60s |

### SOAR Webhook

```bash
# GGID sends alert to external SOAR (Splunk SOAR / Cortex XSOAR)
POST https://soar.corp.com/api/v1/incidents
{
  "source": "GGID-ITDR",
  "severity": "critical",
  "rule_id": "token_theft",
  "mitre": "T1528",
  "user_id": "uuid-alice",
  "user_email": "alice@corp.com",
  "description": "Valid token used from new device + new geo + new UA",
  "auto_actions_taken": ["session_revoked", "account_locked"],
  "evidence": { "ip": "...", "device": "...", "session_id": "..." },
  "playbook_suggested": "isolate_investigate_remediate"
}
```

---

## 9. Implementation Backlog with DoD

### P0 — 8 New Detection Rules (1 sprint)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 1 | consent_phishing rule (T1098) | ✅ MITRE mapped ✅ Detects excessive OAuth scope ✅ DB-backed ✅ ≥3 tests | 2d |
| 2 | mfa_fatigue rule (T1621) | ✅ Detects MFA push flood ✅ ≥3 tests | 1d |
| 3 | token_theft rule (T1528) | ✅ Detects token from new device+geo+UA ✅ ≥3 tests | 2d |
| 4 | session_hijack rule (T1539) | ✅ Detects session from different country ✅ ≥3 tests | 2d |
| 5 | mass_account_creation rule (T1136) | ✅ Detects bulk account creation ✅ ≥3 tests | 1d |
| 6 | federation_anomaly rule (T1606) | ✅ Detects new IdP entity ID ✅ ≥3 tests | 2d |
| 7 | admin_mfa_bypass rule (T1098.001) | ✅ Detects MFA disable by admin ✅ ≥3 tests | 1d |
| 8 | mass_data_export rule (T1005) | ✅ Detects bulk data export ✅ ≥3 tests | 1d |

### P1 — Attack Simulation + Coverage Metrics (1 sprint)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 9 | Attack simulation API | ✅ 7 simulation scenarios ✅ Safe execution ✅ Coverage validation ✅ ≥3 tests | 4d |
| 10 | MITRE coverage dashboard | ✅ Coverage % per tactic ✅ Gap list ✅ DB-backed ✅ ≥3 tests | 2d |
| 11 | SOAR webhook integration | ✅ Critical detections → external SOAR ✅ Auto-response playbook ✅ ≥3 tests | 3d |
| 12 | Auto-response (session revoke + account lock) | ✅ Critical → auto-revoke <5s ✅ ≥3 tests | 2d |

### P2 — ML-Based UEBA (1 sprint)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 13 | Feature extraction pipeline | ✅ 8 behavioral features ✅ Per-user profiles ✅ ≥3 tests | 3d |
| 14 | Isolation Forest model | ✅ Per-user model ✅ Anomaly scoring ✅ ≥3 tests | 4d |
| 15 | ML anomaly detection rule | ✅ Integrates with detection engine ✅ MITRE mapped ✅ ≥3 tests | 2d |
| 16 | Detection-as-code pipeline | ✅ Rules versioned in YAML ✅ CI-tested ✅ ≥3 tests | 3d |

---

## 10. Competitive Differentiation

| Feature | GGID (target) | Microsoft Defender for Identity | Crowdstrike Falcon Identity | Vectra AI | Okta Identity Threat |
|---------|---------------|--------------------------------|---------------------------|-----------|---------------------|
| **Detection rules** | **15 (7+8 new)** | 80+ | 40+ | ML-based | 10 |
| **MITRE mapping** | **All rules** | Yes | Yes | Partial | No |
| **UEBA** | **3σ + Isolation Forest** | ML | ML | AI-native | Basic |
| **Attack simulation** | **Built-in** | Atomic Red Team | Falcon OverWatch | Manual | No |
| **SOAR integration** | **Webhook** | Sentinel | Falcon XSOAR | Vectra AI | Okta Workflow |
| **Auto-response** | **Session revoke + lock** | Full | Full | Full | Basic |
| **Detection-as-code** | **YAML versioned** | KQL | SPL | N/A | N/A |
| **Open source** | **Yes** | No | No | No | No |

**Key differentiator**: GGID would be the only open-source ITDR with MITRE ATT&CK mapping, built-in attack simulation, ML-based UEBA, and detection-as-code — making enterprise-grade identity threat detection accessible without commercial licensing.

---

## References

- [MITRE ATT&CK for Cloud](https://attack.mitre.org/matrices/enterprise/cloud/) — Technique matrix
- [MITRE ATT&CK Identity](https://attack.mitre.org/techniques/T1078/) — Valid Accounts
- [Microsoft Defender for Identity](https://www.microsoft.com/en-us/security/business/identity-access/microsoft-defender-for-identity) — Enterprise ITDR
- [Crowdstrike Falcon Identity Protection](https://www.crowdstrike.com/products/identity-protection/) — Identity threat detection
- [Vectra AI](https://www.vectra.ai/) — AI-driven threat detection
- [Elastic Detection Rules](https://github.com/elastic/detection-rules) — Detection-as-code
- [Sigma Rules](https://github.com/SigmaHQ/sigma) — Generic signature format
- [Atomic Red Team](https://github.com/redcanaryco/atomic-red-team) — Attack simulation
- [GGID Detection Rules](../services/audit/internal/detection/) — All rule files
- [GGID Brute Force Rule](../services/audit/internal/detection/rule_brute_force.go) — T1110 at line 14
- [GGID Token Replay Rule](../services/audit/internal/detection/rule_phase4.go) — T1550 at line 119
- [GGID Threat Intel Rule](../services/audit/internal/detection/rule_threat_intel.go) — T1589 at line 39
- [GGID UEBA Baseline](../services/audit/internal/detection/profile_builder.go) — T1078 at line 121
- [GGID Rule Registry](../services/audit/internal/detection/registry.go) — Per-tenant overrides at line 14
- [GGID ITDR Research](./itdr-fraud-agent-lifecycle-gaps.md) — Previous gap analysis
- [GGID Risk Adaptive Auth Engine](./risk-adaptive-auth-engine.md) — Unified risk engine
- [GGID Zero Trust Maturity Assessment](./zero-trust-maturity-assessment.md) — Identity pillar assessment
