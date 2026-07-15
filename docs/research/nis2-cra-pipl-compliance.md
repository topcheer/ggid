# NIS2 / CRA / PIPL Compliance Research — 2026-07-15

## Sources
- EU Spring 2026 Cybersecurity Update (NIS2, DORA, CRA)
- PIPL amended CSL (Jan 2026)
- GGID internal docs/research/supply-chain-iam.md

## Findings

### PIPL (China)
Requirements:
- Explicit consent + easy withdrawal
- Cross-border transfer assessment
- Data minimization + retention limits
- User rights: access, correction, deletion, portability
- Penalties up to CNY 10M

GGID status:
- DSR handler exists: services/audit/internal/server/dsr_handler.go
- GDPR forget: services/audit/internal/server/gdpr_forget_handler.go
- Data export: services/identity/internal/server/data_export_handler.go
- Consent management: services/oauth/internal/service/consent_management.go + console/src/app/settings/consent-management
- PIPL data inventory: services/identity/internal/server/pipl_data_inventory_handler.go
- PIPL compliance page: console/src/app/settings/pipl-compliance
- Result: **No new productization gap**

### NIS2 / CRA (EU)
Requirements for IAM products:
- SBOM generation and vulnerability tracking
- Secure-by-default configuration
- Coordinated vulnerability disclosure
- Incident reporting within 24h (NIS2)

GGID status:
- SBOM center: console/src/app/settings/sbom-center
- Vulnerability management: console/src/app/settings/vulnerability-management
- Vuln scan results: console/src/app/settings/vuln-scan-results
- SBOM handler: services/audit/internal/server/sbom_handler.go
- Supply-chain research: docs/research/supply-chain-iam.md
- Secure-by-default: gRPC TLS fail-secure, HostValidation, password pepper, rate limiting already wired
- Result: **No new productization gap**

## Conclusion

Current GGID feature set covers PIPL and NIS2/CRA table-stakes requirements. No new [NEW] gaps identified. Continue monitoring:
- EU Digital Identity Wallet (eIDAS 2.0) integration
- Post-quantum cryptography (PQC) migration timelines
- AI Act compliance for identity risk scoring
