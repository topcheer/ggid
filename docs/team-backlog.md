# GGID Team Backlog

*Last updated: 2026-07-14 (arch/PM gap audit + research)*

## Current Stats

- **Docs**: 757 markdown files
- **Console pages**: 629 page.tsx
- **React hooks**: 492 use*.ts
- **Go SDK**: 27 files, 154+ test functions
- **Go services**: 271+ source files, 293+ test files
- **Build**: `go build ./...` = CLEAN
- **Tests**: 40/40 packages PASS, 0 FAIL
- **Real productization gaps**: 3 (1 MEDIUM, 1 LOW, 1 verification pending)

## Gap Closure Priority Queue

### P1 — Real productization gaps (from platform-completeness-report.md)

| # | Feature | Owner | Location | Status | Next Action |
|---|---------|-------|----------|--------|-------------|
| 1 | Client Branding persistence | backend | services/oauth | [NEW] | Replace `brandingStore` map with PostgreSQL repo; add migration |
| 2 | CIBA backchannel route verification | arch | services/oauth/server | [FIXED] | Write functional test that exercises backchannel endpoint |
| 3 | GeoIP MaxMind integration | arch | services/gateway/middleware | [PARTIAL] | Add optional GeoLite2/MMDB lookup with fallback |

### P2 — Research-driven competitive/compliance gaps

| # | Feature | Owner | Driver | Notes |
|---|---------|-------|--------|-------|
| 4 | OAuth 2.1 enforcement mode | backend | RFC 9700 / OAuth 2.1 | Mandatory PKCE, reject implicit/ROPC, exact redirect URI matching |
| 5 | FAPI 2.0 profile | backend | OpenID FAPI | JAR/JARM, PAR, sender-constrained tokens, DPoP |
| 6 | Passkey health dashboard | frontend | passkey adoption | Console page showing passkey enrollment status, recovery risk |
| 7 | PQC migration (ML-DSA / ML-KEM) | arch | NIST PQC | Hybrid TLS + JWT signing in pkg/crypto |
| 8 | NIS2 / CRA compliance dashboard | frontend | EU regulation | Security incident reporting, SBOM, vulnerability tracking |
| 9 | AI agent identity lifecycle | backend | agentic AI | Persistent registry, consent flow, credential rotation, drift detection |
| 10 | Fraud: TOR/VPN/proxy detection | backend | ITDR/fraud | IP intelligence integration, geo-velocity anomaly |

### P3 — Quality/infrastructure improvements

| # | Feature | Owner | Notes |
|---|---------|-------|-------|
| 11 | Gateway middleware coverage | arch | BehavioralBotDetect, PII obfuscation wiring |
| 12 | i18n extraction | frontend | 1051 hardcoded strings -> messages/en.json, zh.json |
| 13 | Console loading/error states | frontend | Remaining 5 pages: ip-allowlist, tenant-config, branding-custom, settings/page, notifications/templates |
| 14 | SDK parity completion | arch | Node SDK admin extensions, React hooks for risk/SOD/PAR |

## Currently Dispatched (Next 24h)

### Backend
1. Client Branding persistence (services/oauth/internal/service/branding.go)

### Arch
1. CIBA backchannel functional test
2. Research: OAuth 2.1 enforcement gap analysis
3. GeoIP MaxMind integration design

### Frontend
1. Passkey health dashboard (console/src/app/settings/passkey-health/)
2. Console loading/error states for remaining 5 pages

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

See docs/research/ for full research docs.