# GGID Team Backlog

*Last updated: 2026-07-15 (Round 17 Focus G — SDK Alignment)*

## Current Stats

- **Docs**: 757 markdown files
- **Console pages**: 629 page.tsx
- **React hooks**: 492 use*.ts
- **Go SDK**: 27 files, 154+ test functions
- **Go services**: 271+ source files, 293+ test files
- **Build**: `go build ./...` = CLEAN
- **Tests**: 40/40 packages PASS, 0 FAIL
- **Real productization gaps**: 2 (GeoIP PARTIAL + SDK alignment NEW)

## Gap Closure Priority Queue

### P1 — Real productization gaps (from platform-completeness-report.md)

| # | Feature | Owner | Location | Status | Next Action |
|---|---------|-------|----------|--------|-------------|
| 1 | GeoIP MaxMind integration | arch | services/gateway/middleware | [PARTIAL] | Add optional GeoLite2/MMDB lookup with private-IP fallback |
| 2 | SDK alignment for Agent Identity / IGA | backend/arch | sdk/python, sdk/java, sdk/rust, sdk/ruby, sdk/csharp, sdk/dart, sdk/php | [NEW] | Add RegisterAgent, ExchangeAgentToken, AccessRequest endpoints to 7 SDKs |

### P2 — Research-driven competitive/compliance gaps

| # | Feature | Owner | Driver | Notes |
|---|---------|-------|--------|-------|
| 3 | OAuth 2.1 enforcement mode | backend | RFC 9700 / OAuth 2.1 | Mandatory PKCE, reject implicit/ROPC, exact redirect URI matching |
| 4 | FAPI 2.0 profile | backend | OpenID FAPI | JAR/JARM, PAR, sender-constrained tokens, DPoP |
| 5 | Passkey health dashboard | frontend | passkey adoption | Console page showing passkey enrollment status, recovery risk |
| 6 | PQC migration (ML-DSA / ML-KEM) | arch | NIST PQC | Hybrid TLS + JWT signing in pkg/crypto |
| 7 | NIS2 / CRA compliance dashboard | frontend | EU regulation | Security incident reporting, SBOM, vulnerability tracking |
| 8 | AI agent identity lifecycle | backend | agentic AI | Persistent registry, consent flow, credential rotation, drift detection |
| 9 | Fraud: TOR/VPN/proxy detection | backend | ITDR/fraud | IP intelligence integration, geo-velocity anomaly |

### P3 — Quality/infrastructure improvements

| # | Feature | Owner | Notes |
|---|---------|-------|-------|
| 10 | Console loading/error states | frontend | Remaining pages: ip-allowlist, tenant-config, branding-custom, settings/page, notifications/templates |
| 11 | i18n extraction | frontend | 1051 hardcoded strings -> messages/en.json, zh.json |
| 12 | SDK parity completion | arch | Node SDK admin extensions, React hooks for risk/SOD/PAR |

## Currently Dispatched (Next 24h)

### Backend
1. SDK alignment: Agent Identity / Access Request methods for 7 SDKs

### Arch
1. GeoIP MaxMind design
2. OAuth 2.1 / FAPI 2.0 gap analysis

### Frontend
1. Console loading/error states for remaining pages
2. Passkey health dashboard

### Docs/Research
1. Console mock-pages audit
2. OAuth 2.1 / FAPI 2.0 research

## Rules

- Each task owner must report: commit hash + make test result
- No modification of other teammates' directories
- Gap status changes require verification (see docs/gap-maintenance-rules.md)
- Research findings go to docs/research/*.md before entering backlog
- All dependencies use @latest

## Research Pipeline

Active research topics:
- OAuth 2.1 / FAPI 2.0 compliance gap analysis
- PQC migration for IAM systems
- AI agent identity governance patterns
- NIS2 / CRA compliance for IAM vendors
- Console mock data audit

See docs/research/ for full research docs.