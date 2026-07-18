# Dependency Security Audit

**Date**: 2025-07-18  
**Tool**: govulncheck + go list -m -u  
**Go Version**: 1.26.4

---

## Vulnerability Scan Results

| ID | Package | Severity | Found In | Fixed In | Status |
|----|---------|----------|----------|----------|--------|
| GO-2026-5856 | crypto/tls (stdlib) | Medium | go1.26.4 | go1.26.5 | Pending Go upgrade |
| GO-2026-4970 | os (stdlib) | Low | go1.26.4 | go1.26.5 | Pending Go upgrade |
| GO-2026-5932 | golang.org/x/crypto | Info | v0.53.0 | N/A | Fixed: upgraded v0.54.0 |

## Upgraded Dependencies

| Package | From | To | Reason |
|---------|------|-----|--------|
| golang.org/x/crypto | v0.53.0 | v0.54.0 | Vuln fix (GO-2026-5932) |
| golang.org/x/sync | v0.21.0 | v0.22.0 | Minor version |
| golang.org/x/sys | v0.46.0 | v0.47.0 | Minor version |
| golang.org/x/text | v0.38.0 | v0.40.0 | Minor version |
| golang.org/x/net | v0.56.0 | v0.57.0 | Minor version |
| github.com/go-ldap/ldap/v3 | v3.4.13 | v3.4.14 | Patch: security fix |
| github.com/klauspost/compress | v1.18.6 | v1.19.0 | Minor: performance |
| github.com/Azure/go-ntlmssp | v0.1.0 | v0.1.1 | Patch |
| github.com/boombuler/barcode | v1.0.1-pre | v1.1.0 | Major: stable release |

## Available Upgrades (Not Applied)

These have newer versions but were not upgraded to avoid potential breaking changes:

| Package | Current | Available | Risk |
|---------|---------|-----------|------|
| prometheus/common | v0.66.1 | v0.70.0 | Medium: metrics API changes |
| prometheus/procfs | v0.16.1 | v0.21.0 | Low: read-only |
| klauspost/cpuid/v2 | v2.2.10 | v2.4.0 | Low |
| mattn/go-isatty | v0.0.20 | v0.0.23 | Low |

**Recommendation**: Upgrade prometheus/common and procfs after v1.0-stable (requires testing metrics output compatibility).

## High-Risk Dependencies Review

| Package | Version | Risk Assessment |
|---------|---------|----------------|
| golang.org/x/crypto | v0.54.0 | Safe: latest patched |
| golang.org/x/net | v0.57.0 | Safe: latest |
| pgx/v5 | v5.x | Safe: stable database driver |
| google/uuid | v1.x | Safe: no known CVEs |
| jwt/v5 | v5.x | Safe: audited JWT library |

## Action Items

1. Upgrade Go toolchain to 1.26.5 (fixes 2 stdlib vulns)
2. Upgrade prometheus/common to v0.70.0 (post-stable)
3. Run govulncheck as blocking CI step (currently advisory)

## Conclusion

No critical or high-severity vulnerabilities in dependencies. Two medium stdlib issues are fixed in Go 1.26.5. All third-party packages are at or near latest stable versions.
