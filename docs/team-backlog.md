# GGID Team Backlog

*Last updated: 2026-07-15 (Round 7 Focus A PM cycle)*

## Current Stats

- **Docs**: 757 markdown files
- **Console pages**: 629 page.tsx
- **React hooks**: 492 use*.ts
- **Go SDK**: 27 files, 154+ test functions
- **Go services**: 271+ source files, 293+ test files
- **Build**: `go build ./...` = CLEAN
- **Tests**: 40/40 packages PASS, 0 FAIL
- **Real productization gaps**: 3 (from platform-completeness-report.md)

## Gap Closure Priority Queue

### P1 — Real productization gaps (from platform-completeness-report.md)

| # | Feature | Owner | Location | Status | Next Action |
|---|---------|-------|----------|--------|-------------|
| 1 | Client Branding persistence | backend | services/oauth | [NEW] | Wire `handleClientBranding` to `brandingAdapterVar` (PG-first with mem fallback) |
| 2 | CIBA backchannel route verification | arch | services/oauth/server | [FIXED] | Write functional test that exercises `/api/v1/oauth/backchannel` endpoint |
| 3 | GeoIP MaxMind integration | arch | services/gateway/middleware | [PARTIAL] | Add optional GeoLite2/MMDB lookup with private-IP fallback |

### P2 — Round 7 Focus A 新增 stub (已分配)

| # | Feature | Owner | Location | Status | Next Action |
|---|---------|-------|----------|--------|-------------|
| 4 | Gateway sysconfigStore wiring | backend | services/gateway/cmd/main.go:39 | [NEW] | Wire sysconfigStore into gateway middleware for hot-reload |
| 5 | OAuth placeholder JWT | backend | services/oauth/internal/service/oauth_service.go:383 | [NEW] | Replace placeholder self-contained JWT with real token issuance |
| 6 | security-center mock data | frontend | console/src/app/security-center/page.tsx:88 | [NEW] | Replace mockData() with real API fetch + loading/error |
| 7 | security page mock data | frontend | console/src/app/security/page.tsx:62 | [NEW] | Replace mockData() with real API fetch + loading/error |

### P3 — Research-driven competitive/compliance gaps

| # | Feature | Owner | Driver | Notes |
|---|---------|-------|--------|-------|
| 8 | OAuth 2.1 enforcement mode | backend | RFC 9700 / OAuth 2.1 | Mandatory PKCE, reject implicit/ROPC, exact redirect URI matching |
| 9 | FAPI 2.0 profile | backend | OpenID FAPI | JAR/JARM, PAR, sender-constrained tokens, DPoP |
| 10 | Passkey health dashboard | frontend | passkey adoption | Console page showing passkey enrollment status, recovery risk |
| 11 | PQC migration (ML-DSA / ML-KEM) | arch | NIST PQC | Hybrid TLS + JWT signing in pkg/crypto |
| 12 | NIS2 / CRA compliance dashboard | frontend | EU regulation | Security incident reporting, SBOM, vulnerability tracking |
| 13 | AI agent identity lifecycle | backend | agentic AI | Persistent registry, consent flow, credential rotation, drift detection |
| 14 | Fraud: TOR/VPN/proxy detection | backend | ITDR/fraud | IP intelligence integration, geo-velocity anomaly |

### P4 — Quality/infrastructure improvements

| # | Feature | Owner | Notes |
|---|---------|-------|-------|
| 15 | Docker E2E infra | devops | docker compose up 未启动，E2E 测试 0/11 FAIL |
| 16 | Gateway middleware coverage | arch | BehavioralBotDetect, PII obfuscation wiring |
| 17 | i18n extraction | frontend | 1051 hardcoded strings -> messages/en.json, zh.json |
| 18 | Console loading/error states | frontend | Remaining 5 pages: ip-allowlist, tenant-config, branding-custom, settings/page, notifications/templates |
| 19 | SDK parity completion | arch | Node SDK admin extensions, React hooks for risk/SOD/PAR |

## Currently Dispatched (Next 24h)

### Backend
1. Client Branding persistence
2. Gateway sysconfigStore wiring
3. OAuth placeholder JWT

### Arch
1. CIBA backchannel functional test
2. GeoIP MaxMind integration design
3. Research: OAuth 2.1 / FAPI 2.0 gap analysis

### Frontend
1. security-center mockData replacement
2. security page mockData replacement
3. Console loading/error states for remaining 5 pages

### Docs/Research
1. Docker E2E infra gap analysis
2. Console mock-pages audit

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
- Docker E2E test infrastructure gaps
- Console mock data audit

See docs/research/ for full research docs.
