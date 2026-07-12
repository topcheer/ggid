# ITDR + Fraud Detection + AI Agent Lifecycle Gap Analysis

**Date**: 2026-07-26
**Researcher**: arch (research-driven backlog cycle)

## Summary

Research into Okta/Auth0 September 2025 announcements and 2025-2026 IAM trends reveals three critical competitive gaps in GGID:

## Gap 1: Identity Threat Detection and Response (ITDR) — P1

**What competitors have**:
- Okta: Identity Threat Orchestrator with automated response playbooks
- Auth0: Open-source detection catalog with 50+ threat detection rules
- CrowdStrike + Okta integration for lateral movement detection

**What GGID has**:
- Risk scoring engine (weighted_sum) ✅
- Session hijack timeline ✅
- Credential stuffing stats ✅
- Anomaly scoring config ✅

**What GGID is MISSING**:
- [ ] ITDR detection rules catalog (curated threat signatures like Auth0's open-source catalog)
- [ ] Automated response playbooks (block IP → revoke sessions → notify admin → create ticket)
- [ ] Lateral movement detection (user accessing unusual resources in rapid succession)
- [ ] Privilege escalation detection (user gaining new permissions without normal workflow)
- [ ] Golden ticket / Kerberoasting equivalent detection for JWT (token forgery patterns)
- [ ] MITRE ATT&CK mapping for identity-based attacks
- [ ] Threat intelligence feed integration (known-bad IPs, compromised credentials)

**Implementation priorities**:
1. Backend: `services/auth/internal/server/itdr_handler.go` — detection rules engine + response playbooks
2. Frontend: `console/src/app/settings/itdr-dashboard/` — threat detection dashboard
3. Docs: ITDR implementation guide

## Gap 2: Fraud Detection Engine — P1

**What competitors have**:
- Okta: Risk scoring with device fingerprinting, velocity checks, bot detection
- Auth0: Attack Protection suite (credential stuffing, brute force, suspicious IP)
- Cloudflare: Bot management integration

**What GGID has**:
- Credential stuffing stats ✅
- Brute force config ✅
- Bot detection middleware (gateway) ✅
- IP reputation page ✅

**What GGID is MISSING**:
- [ ] Device fingerprinting service (canvas fingerprint, WebGL fingerprint, TLS fingerprint)
- [ ] Velocity rules engine (max registrations per IP per hour, max logins per device)
- [ ] Account takeover (ATO) prevention with ML-based scoring
- [ ] Synthetic identity detection (newly created emails, disposable domains)
- [ ] Disposable email domain blocklist
- [ ] TOR exit node detection
- [ ] VPN/proxy detection
- [ ] Fraud score aggregation across signals (device + velocity + reputation + behavioral)

**Implementation priorities**:
1. Backend: `services/auth/internal/server/fraud_detection_handler.go`
2. Frontend: `console/src/app/settings/fraud-detection-dashboard/`
3. Shared: `pkg/fraud/` — reusable fraud scoring package

## Gap 3: AI Agent Identity Lifecycle — P1

**What competitors have**:
- Okta: "Standards-first AI agents" with identity security fabric for end-to-end lifecycle
- Auth0: AI agent authentication with OAuth + delegation chains
- Microsoft: Copilot identity via Entra ID

**What GGID has**:
- Agent identity registration + token exchange ✅ (commit 55ffd6f)
- Agent token claims with delegation chain ✅
- Agent registry (in-memory) ✅
- Console AI Agents page ✅

**What GGID is MISSING**:
- [ ] Agent lifecycle management (onboard → provision → monitor → revoke → audit)
- [ ] Agent-to-agent delegation (multi-hop delegation chain validation)
- [ ] Agent permission scoping (fine-grained per-resource, per-action)
- [ ] Agent behavioral monitoring (unusual API patterns, excessive requests)
- [ ] Agent rate limiting per tenant
- [ ] Agent credential rotation automation
- [ ] Agent consent flow (user approves agent access scope)
- [ ] Persistent agent registry (database-backed, not in-memory)
- [ ] MCP (Model Context Protocol) server authentication integration
- [ ] Agent federation (cross-tenant agent trust)

**Implementation priorities**:
1. Backend: `services/oauth/internal/service/agent_lifecycle.go` — lifecycle management
2. Frontend: `console/src/app/settings/agent-lifecycle-dashboard/`
3. SDK: Go SDK `agent_lifecycle.go` methods
4. Docs: AI Agent Identity Lifecycle guide

## Gap 4: Conditional UI Passkey Best Practices — P2

**Status**: Partially implemented (conditional_ui.go exists)

**What's MISSING**:
- [ ] Passkey autofill mediation UI component in console login flow
- [ ] Cross-device passkey sync status indicator
- [ ] Passkey health dashboard (registered passkeys per user, last used, device type)
- [ ] Conditional UI fallback strategy documentation

## Gap 5: PIPL (China Personal Information Protection Law) Compliance — P2

**What GGID has**:
- GDPR requests page ✅
- Data sovereignty ✅

**What's MISSING**:
- [ ] PIPL-specific data handling rules
- [ ] Cross-border data transfer assessment for China
- [ ] Separate consent management for Chinese users
- [ ] Data protection officer (DPO) assignment workflow

## Gap 6: OAuth 2.1 Compliance Audit — P2

**What GGID has**:
- PKCE enforcement ✅
- DPoP ✅
- JAR ✅
- PAR ✅
- Token exchange (RFC 8693) ✅

**What's MISSING**:
- [ ] OAuth 2.1 compliance audit report (formal checklist)
- [ ] Deprecation of implicit grant (should be disabled/removed)
- [ ] Deprecation of password grant (should be disabled/removed)
- [ ] Exact redirect URI matching enforcement (no wildcards)
- [ ] State parameter mandatory enforcement

## Recommendations

**Immediate (P1)**:
- ITDR detection rules + response playbooks (backend + frontend)
- Fraud detection engine with device fingerprinting (shared pkg)
- AI agent lifecycle (persistent registry + monitoring)

**Next quarter (P2)**:
- PIPL compliance module
- OAuth 2.1 compliance audit tool
- Passkey health dashboard
