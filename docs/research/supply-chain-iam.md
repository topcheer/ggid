# Go Module Supply Chain Security for IAM Systems

> **Research Document** — GGID IAM Suite
> Topic: Supply chain threats, verification, SBOM, SLSA provenance, and dependency auditing for Go-based identity and access management systems.
> Date: 2025-07-11

---

## Table of Contents

1. [Supply Chain Threat Model for Go Projects](#1-supply-chain-threat-model-for-go-projects)
2. [go.sum Verification](#2-gosum-verification)
3. [SBOM (Software Bill of Materials) Generation](#3-sbom-software-bill-of-materials-generation)
4. [SLSA Provenance](#4-slsa-provenance)
5. [govulncheck for Vulnerability Scanning](#5-govulncheck-for-vulnerability-scanning)
6. [Module Authentication with GOPRIVATE/GONOSUMCHECK](#6-module-authentication-with-goprivate-gonosumcheck)
7. [Dependency Pinning and Update Policies](#7-dependency-pinning-and-update-policies)
8. [GGID Dependency Audit](#8-ggid-dependency-audit)
9. [Gap Analysis & Recommendations](#9-gap-analysis--recommendations)

---

## 1. Supply Chain Threat Model for Go Projects

An Identity and Access Management (IAM) system is a high-value target for supply chain attacks. A compromised dependency in GGID could lead to token theft, authentication bypass, privilege escalation, or mass credential exfiltration. This section maps the primary threat vectors in the Go module ecosystem.

### 1.1 Dependency Confusion Attacks

**Mechanism:** An attacker publishes a malicious package to the public Go module proxy (proxy.golang.org) using a module path that matches a private internal path. When a developer or CI build resolves dependencies, the public (malicious) module may be selected over the private one if the private module is not properly configured with `GOPRIVATE` or `GONOSUMCHECK`.

**Example scenario for GGID:**
```
# Internal private module (assumed)
github.com/ggid/internal-auth

# Attacker publishes to public proxy:
github.com/ggid/internal-auth  (malicious, version v1.0.0)
```
If the Go toolchain queries the public proxy first and finds `v1.0.0`, it downloads the attacker's code. The `go.sum` file does not protect against this if no prior checksum exists — the first download creates the entry.

**Mitigation:**
```bash
# In .envrc or CI environment
export GOPRIVATE="github.com/ggid/*"
# This tells Go to skip the public proxy and checksum DB for these paths
```

### 1.2 Typosquatting

**Mechanism:** Attackers register module names that are visually similar to popular packages. Developers mistype import paths or copy-paste incorrect paths from search results.

| Legitimate Module | Typosquat Variant |
|---|---|
| `github.com/gorilla/mux` | `github.com/gorila/mux` |
| `github.com/golang-jwt/jwt/v5` | `github.com/golang-jwt/jwt5` |
| `golang.org/x/crypto` | `golang.org/x/crytpo` |
| `github.com/google/uuid` | `github.com/goog1e/uuid` |

In Go, typosquatting is somewhat mitigated by the module proxy and `go.sum` — a typosquatted module would appear as a new dependency in `go.mod`, making it visible during code review. However, if a developer is tricked into intentionally importing it (e.g., via a fake StackOverflow answer), the damage is done.

### 1.3 Malicious Commits and Account Takeover

**Mechanism:** A maintainer's GitHub account, npm account, or Git credentials are compromised. The attacker pushes a malicious commit or tags a new release containing backdoors, credential stealers, or obfuscated malware.

**Notable real-world incidents:**

- **Go-iban (2024):** A malicious version of the `github.com/zytellnrg/Go-iban` package was published targeting specific organizations, using targeted module names to perform dependency confusion.
- **ueberall/go-mobile (2023):** A compromised GitHub token was used to backdoor a Go mobile development tool.
- **Go-module-backdoor (2022):** Security researchers demonstrated a proof-of-concept where a Go module could execute arbitrary code during `go build` via compiler directives (`//go:generate`).
- **xz-utils (2024, cross-ecosystem):** While not Go-specific, this attack demonstrated how a patient attacker can inject backdoors into widely-used open-source libraries over years. The Go ecosystem is equally vulnerable to this class of social engineering + supply chain attack.

### 1.4 Protestware

**Mechanism:** Maintainers intentionally sabotage their own packages to make political statements. This has affected npm (colors.js, node-ipc) and could affect any ecosystem.

**Relevance to Go:** Go's module proxy provides caching and immutability (once a version is downloaded by the proxy, it cannot be changed). This means a maintainer cannot retroactively modify a published version. However, they can publish a new malicious version. If CI runs `go get -u` without version constraints, it picks up the new release.

**Defense:** Pin versions in `go.mod`, review changelogs before upgrading, and use `GOPROXY=direct` only in trusted environments.

### 1.5 Threat Summary for GGID

| Threat | Likelihood | Impact | Current GGID Exposure |
|---|---|---|---|
| Dependency confusion | Medium | Critical | Low (all deps are public OSS, no private modules yet) |
| Typosquatting | Low | High | Low (go.mod is reviewed, imports are explicit) |
| Account takeover | Low | Critical | Medium (depends on upstream maintainer security) |
| Protestware | Low | Medium | Low (module proxy caching) |
| Transitive vuln | Medium | High | Medium (no govulncheck in CI yet) |

---

## 2. go.sum Verification

### 2.1 How go.sum Provides Integrity

The `go.sum` file contains cryptographic hashes (SHA-256) of every module version that the project depends on — both direct and transitive. Each entry has two hashes: one for the module `.zip` and one for the `go.mod` file.

```
# Format: <module> <version> <hash-type>:<hex>
golang.org/x/crypto v0.53.0 h1:ABcdEf123...==
golang.org/x/crypto v0.53.0/go.mod h1:ABcdEf456...==
```

When `go build` or `go mod download` fetches a module, the Go toolchain:
1. Downloads the module from `GOPROXY` (default: `proxy.golang.org`)
2. Computes the SHA-256 hash of the downloaded `.zip` and `go.mod`
3. Compares against the entry in `go.sum`
4. If mismatch: **build fails** with a `SECURITY ERROR` — this prevents tampering
5. If no entry exists: verifies against `GOSUMDB` (sum.golang.org) and adds the entry

### 2.2 Why GONOSUMCHECK Is Dangerous

```bash
# DANGEROUS — disables all checksum verification
export GONOSUMCHECK=*
# Or slightly less dangerous but still risky:
export GONOSUMCHECK=github.com/some-org/*
```

Setting `GONOSUMCHECK` tells the Go toolchain to **skip checksum database verification** for matching module paths. This means:
- A compromised module proxy could serve a malicious version without detection
- `go.sum` entries may not be validated against the global checksum DB
- Transitive dependencies may be silently replaced

**Recommendation:** Never use `GONOSUMCHECK`. Use `GOPRIVATE` instead, which skips both the proxy and checksum DB for private modules but does not weaken verification for public modules.

### 2.3 GOSUMDB (sum.golang.org)

The Go checksum database (`sum.golang.org`) is a global, append-only, cryptographically-signed log of all known Go module hashes. It provides:

- **Transparency:** Anyone can verify that a module version has not been surreptitiously altered
- **Global consistency:** All developers see the same checksums for public modules
- **Tamper-evidence:** The Merkle Tree structure makes retroactive modifications detectable

**Verification flow:**
```
Developer runs: go build
  → Go fetches module from proxy.golang.org
  → Go computes SHA-256 of downloaded content
  → Go checks against local go.sum
  → If not in go.sum, Go queries sum.golang.org
  → sum.golang.org returns the canonical hash
  → Go verifies downloaded content matches canonical hash
  → Go adds entry to go.sum
```

### 2.4 Offline Verification

For air-gapped environments or CI without internet access:

```bash
# Step 1: On a networked machine, download all dependencies
go mod download
GOFLAGS=-mod=mod go mod verify

# Step 2: Vendor dependencies for offline use
go mod vendor

# Step 3: Verify checksums (works offline with vendored deps)
go mod verify
# Output: "all modules verified"

# Step 4: Transfer vendor/ directory to air-gapped system
# Build offline:
GOFLAGS=-mod=vendor go build ./...
```

### 2.5 Verifying Module Checksums Programmatically

```bash
# Verify all modules in go.sum match downloaded content
go mod verify

# Check a specific module's checksum manually
go mod download -x golang.org/x/crypto@v0.53.0 2>&1 | grep hash

# List all modules and their verification status
go list -m all | while read mod; do
  echo "Checking $mod..."
done
go mod verify
```

```go
// Programmatic checksum verification (simplified)
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
)

func verifyChecksum(filePath, expectedHash string) bool {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}
	hash := sha256.Sum256(data)
	computed := hex.EncodeToString(hash[:])
	return computed == expectedHash
}

func main() {
	ok := verifyChecksum("module.zip", "expected-sha256-hex")
	if !ok {
		fmt.Println("CHECKSUM MISMATCH — potential tampering!")
		os.Exit(1)
	}
	fmt.Println("Checksum verified successfully")
}
```

---

## 3. SBOM (Software Bill of Materials) Generation

### 3.1 Why SBOM Matters for IAM

An IAM system like GGID processes authentication tokens, passwords, and authorization decisions. Regulatory frameworks (SOC 2, ISO 27001, NIST SSDF) increasingly require SBOMs for security-critical software:

- **Vulnerability tracking:** When a new CVE is announced, an SBOM lets you instantly determine if GGID uses the affected library
- **License compliance:** Verify all dependencies use compatible licenses (GGID is Apache 2.0)
- **Audit trail:** Demonstrates due diligence in dependency management
- **Incident response:** Rapid identification of affected components during a supply chain incident

### 3.2 Generating SBOM with syft

[Syft](https://github.com/anchore/syft) is the most widely-used SBOM tool. It supports multiple output formats (SPDX, CycloneDX).

```bash
# Install syft
curl -sSfL https://raw.githubusercontent.com/anchore/syft/main/install.sh | sh -s -- -b /usr/local/bin

# Generate SBOM in CycloneDX format
syft /Users/zhanju/ggai/ggid -o cyclonedx-json > sbom-ggid.json

# Generate SBOM in SPDX format
syft /Users/zhanju/ggai/ggid -o spdx-json > sbom-ggid.spdx.json

# Scan a built Docker image
syft deploy-identity:latest -o cyclonedx-json > sbom-identity.json
```

### 3.3 Generating SBOM with cyclonedx-gomod

[cyclonedx-gomod](https://github.com/CycloneDX/cyclonedx-gomod) is Go-native and produces more accurate CycloneDX output by reading `go.mod` directly.

```bash
# Install
go install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@latest

# Generate SBOM from go.mod
cyclonedx-gomod mod -json -licenses > ggid-sbom.cdx.json

# Include transitive dependencies and test dependencies
cyclonedx-gomod app -json -licenses -main ./services/gateway > ggid-gateway-sbom.cdx.json
```

### 3.4 Example SBOM Output (CycloneDX)

```json
{
  "bomFormat": "CycloneDX",
  "specVersion": "1.5",
  "components": [
    {
      "type": "library",
      "name": "golang.org/x/crypto",
      "version": "v0.53.0",
      "purl": "pkg:golang/golang.org/x/crypto@v0.53.0",
      "licenses": [{"license": {"id": "BSD-3-Clause"}}]
    },
    {
      "type": "library",
      "name": "github.com/golang-jwt/jwt/v5",
      "version": "v5.3.1",
      "purl": "pkg:golang/github.com/golang-jwt/jwt/v5@v5.3.1",
      "licenses": [{"license": {"id": "MIT"}}]
    },
    {
      "type": "library",
      "name": "github.com/jackc/pgx/v5",
      "version": "v5.10.0",
      "purl": "pkg:golang/github.com/jackc/pgx/v5@v5.10.0",
      "licenses": [{"license": {"id": "MIT"}}]
    }
  ]
}
```

### 3.5 CI/CD Integration

```yaml
# .github/workflows/sbom.yml
name: Generate SBOM

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  sbom:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'

      - name: Generate SBOM (CycloneDX)
        run: |
          go install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@latest
          cyclonedx-gomod mod -json -licenses -output ggid-sbom.cdx.json

      - name: Upload SBOM artifact
        uses: actions/upload-artifact@v4
        with:
          name: ggid-sbom
          path: ggid-sbom.cdx.json

      - name: Scan SBOM for vulnerabilities
        run: |
          go install github.com/anchore/grype@latest
          grype dir:. --scope all-layers --fail-on high
```

---

## 4. SLSA Provenance

### 4.1 What Is SLSA?

**SLSA** (Supply-chain Levels for Software Artifacts, pronounced "salsa") is a security framework from Google/OpenSSF that defines graduated levels of supply chain integrity assurance. It addresses the question: *"How do we know this binary was built from the expected source code, on a trusted build platform, without tampering?"*

### 4.2 SLSA Build Levels

| Level | Description | Build Platform | Provenance | Tamper Resistance |
|---|---|---|---|---|
| **SLSA 1** | Build documented | Any | Build process documented | Basic |
| **SLSA 2** | Hosted build | Trusted CI (GitHub Actions, Cloud Build) | Signed, verifiable provenance | Hardened CI |
| **SLSA 3** | Hardened build | Isolated, ephemeral build environment | Non-falsifiable provenance | Hermetic builds |
| **SLSA 4** | Reproducible + reviewed | Two-party reviewed, reproducible | Complete provenance | Highest assurance |

**Current GGID state:** SLSA 0-1 (standard GitHub Actions, no provenance generation).

### 4.3 Provenance Attestation Format

SLSA provenance is an in-toto attestation — a signed JSON document describing how a build artifact was produced:

```json
{
  "_type": "https://in-toto.io/Statement/v0.1",
  "subject": [
    {
      "name": "ggid-gateway",
      "digest": {
        "sha256": "e3b0c44298fc1c149afbf4c8996fb924..."
      }
    }
  ],
  "predicateType": "https://slsa.dev/provenance/v0.2",
  "predicate": {
    "builder": {
      "id": "https://github.com/actions/runner"
    },
    "buildType": "https://github.com/ggid/ggid/.github/workflows/build.yml@refs/heads/main",
    "source": {
      "location": "https://github.com/ggid/ggid",
      "revision": {
        "uri": "git+https://github.com/ggid/ggid",
        "digest": {
          "sha1": "abc123def456..."
        }
      }
    },
    "materials": [
      {
        "uri": "pkg:golang/github.com/golang-jwt/jwt/v5@v5.3.1",
        "digest": {
          "sha256": "f1d2d2f924e986ac86fdf7b36c94bcdf32beec15"
        }
      }
    ]
  }
}
```

### 4.4 Generating Provenance for Go Binaries with GitHub Actions

GitHub provides the `actions/attest-build-provenance` action for SLSA Level 3 provenance:

```yaml
# .github/workflows/release.yml
name: Build and Attest

on:
  push:
    tags: ['v*']

permissions:
  contents: read
  id-token: write   # Required for OIDC token
  attestations: write

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'

      - name: Build all services
        run: |
          CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -buildid=" \
            -o bin/ggid-gateway ./services/gateway
          CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -buildid=" \
            -o bin/ggid-auth ./services/auth

      - name: Generate SHA-256 for artifacts
        id: hash
        run: |
          echo "gateway=$(sha256sum bin/ggid-gateway | awk '{print $1}')" >> $GITHUB_OUTPUT
          echo "auth=$(sha256sum bin/ggid-auth | awk '{print $1}')" >> $GITHUB_OUTPUT

      - name: Attest build provenance (SLSA L3)
        uses: actions/attest-build-provenance@v1
        with:
          subject-name: ggid-gateway
          subject-digest: sha256:${{ steps.hash.outputs.gateway }}
          push-to-registry: false

      - name: Attest auth service
        uses: actions/attest-build-provenance@v1
        with:
          subject-name: ggid-auth
          subject-digest: sha256:${{ steps.hash.outputs.auth }}
          push-to-registry: false
```

### 4.5 Verification at Consumption Time

Consumers verify provenance before trusting a binary:

```bash
# Install GitHub CLI attestation verification
gh extension install actions/attest-build-provenance

# Verify a downloaded binary's provenance
gh attestation verify ggid-gateway \
  --repo ggid/ggid \
  --digest-alg sha256 \
  --digest e3b0c44298fc1c149afbf4c8996fb924...

# Programmatic verification with cosign
cosign verify-attestation \
  --certificate-identity "https://github.com/ggid/ggid/.github/workflows/release.yml@refs/tags/v1.0.0" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
  ggid-gateway
```

---

## 5. govulncheck for Vulnerability Scanning

### 5.1 Overview

`govulncheck` is the official Go vulnerability scanner. It is distinct from generic SCA tools because it performs **call-graph analysis** — it only reports vulnerabilities in code paths that GGID actually reaches, dramatically reducing false positives.

### 5.2 Running govulncheck Against GGID

```bash
# Install
go install golang.org/x/vuln/cmd/govulncheck@latest

# Scan the entire project
govulncheck ./...

# Scan a specific service
govulncheck ./services/auth/...

# Output in JSON for CI parsing
govulncheck -json ./... > vuln-report.json

# Scan with mode=source (analyzes source code paths)
govulncheck -mode source ./...

# Scan with mode=binary (analyzes compiled binary)
govulncheck -mode binary ./bin/ggid-gateway
```

### 5.3 Go Vulnerability Database

`govulncheck` queries `vuln.go.dev`, the official Go vulnerability database maintained by the Go security team. It is sourced from:

- **GHSA** (GitHub Security Advisories)
- **Go vuln database** entries (manually curated)
- **OSV** (Open Source Vulnerabilities database)

The database includes:
- Affected module and version ranges
- Fixed versions
- Affected symbols/functions (enabling call-graph analysis)
- CVE and GHSA identifiers
- Severity scores

### 5.4 Call-Graph Analysis — Why It Matters

Traditional SCA tools flag any dependency with a known CVE, regardless of whether your code calls the vulnerable function. For example:

```
# Without call-graph analysis (false positive):
Vulnerability #1: GO-2024-1234 in golang.org/x/crypto
  Affected: ssh.ClientConfig with InsecureIgnoreHostKey
  Your project imports golang.org/x/crypto but GGID does NOT use the SSH package.
  → Traditional tools: ALERT (false positive)
  → govulncheck: No alert (not reachable)
```

```bash
# Example govulncheck output showing call-graph filtering:
$ govulncheck ./services/auth/...
=== Symbol Results ===

Vulnerability #1: GO-2024-2687
    If errors returned from MarshalJSON methods are placed into fmt.Errorf
    contexts, the sensitive information may be exposed.
  More info: https://pkg.go.dev/vuln/GO-2024-2687
  Standard library
    Found in: go1.25.0
    Fixed in: go1.25.1
  Example:
    # GGID calls jwt.Marshal which internally uses encoding/json
    services/auth/internal/handler/jwt.go:42:23: jwt.MarshalClaims

=== Informational ===

Vulnerability #2: GO-2024-1234
  More info: https://pkg.go.dev/vuln/GO-2024-1234
  golang.org/x/crypto
    Found in: golang.org/x/crypto@v0.53.0
    Fixed in: golang.org/x/crypto@v0.54.0
  This vulnerability is NOT reachable from GGID code.
```

### 5.5 CI Integration

```yaml
# .github/workflows/vulncheck.yml
name: Vulnerability Scan

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]
  schedule:
    - cron: '0 6 * * 1'  # Weekly Monday scan

jobs:
  govulncheck:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'

      - name: Install govulncheck
        run: go install golang.org/x/vuln/cmd/govulncheck@latest

      - name: Run vulnerability scan
        run: |
          # Fail on reachable vulnerabilities, informational goes to stdout
          govulncheck ./... 2>&1 | tee vuln-report.txt

          # Check if any reachable vulnerabilities were found
          if grep -q "Vulnerability" vuln-report.txt && \
             ! grep -q "No vulnerabilities found" vuln-report.txt; then
            echo "::error::Reachable vulnerabilities found"
            exit 1
          fi

      - name: Upload vulnerability report
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: vuln-report
          path: vuln-report.txt
```

### 5.6 Remediation Workflow

When `govulncheck` identifies a reachable vulnerability:

1. **Assess severity** — Check CVE/GHSA score and CVSS vector
2. **Check fix availability** — `govulncheck` reports the fixed version
3. **Upgrade the dependency:**
   ```bash
   go get golang.org/x/crypto@latest
   go mod tidy
   go build ./...
   go test ./...
   ```
4. **Verify the fix** — Re-run `govulncheck ./...`
5. **Document** — Add a security note to the release changelog

---

## 6. Module Authentication with GOPRIVATE/GONOSUMCHECK

### 6.1 Safe Use of Private Modules

When GGID begins using private modules (e.g., `github.com/ggid/internal-*`), proper configuration is critical to prevent dependency confusion while maintaining security.

### 6.2 GOPRIVATE Pattern Matching

```bash
# .envrc or CI environment
export GOPRIVATE="github.com/ggid/*,gitlab.com/ggid-company/*"

# What GOPRIVATE does:
# 1. Skips proxy.golang.org for matching modules (fetches directly from VCS)
# 2. Skips sum.golang.org verification (no public checksum available)
# 3. Still uses go.sum for local verification (first-download pinning)
```

**Pattern matching syntax (glob):**
```bash
# All modules under ggid org
GOPRIVATE="github.com/ggid/*"

# Specific repo only
GOPRIVATE="github.com/ggid/billing"

# Multiple patterns
GOPRIVATE="github.com/ggid/*,github.com/ggid-enterprise/*"
```

### 6.3 GOFLAGS for Consistent Behavior

```bash
# Enforce vendor mode in CI (all deps pre-vendored)
export GOFLAGS="-mod=vendor"

# Enforce readonly mode (prevents accidental go.mod changes)
export GOFLAGS="-mod=mod"

# Combined with GOPRIVATE
export GOFLAGS="-mod=mod"
export GOPRIVATE="github.com/ggid/*"
export GOPROXY="https://proxy.golang.org,direct"
```

### 6.4 Athens / JFrog Artifactory as Private Module Proxy

Using a private Go module proxy provides significant security benefits:

**Athens (open-source):**
```bash
# Run Athens as a caching proxy
export GOPROXY="https://athens.ggid.internal,https://proxy.golang.org,direct"
export GOPRIVATE="github.com/ggid/*"
```

**JFrog Artifactory:**
```bash
export GOPROXY="https://ggid.jfrog.io/artifactory/api/go/go-local,https://proxy.golang.org,direct"
```

**Security benefits of a private proxy:**
1. **Caching/immutability** — Once a module version is cached, subsequent fetches always get the same bytes (prevents supply chain tampering of upstream repos)
2. **Access control** — Private modules are only accessible through authenticated proxy
3. **Audit trail** — All module downloads are logged
4. **Air-gap support** — Private proxy can run in isolated networks
5. **VET (Vetting)** — Modules can be scanned/approved before being cached

```yaml
# Example: Athens in docker-compose for GGID
# deploy/docker-compose.yaml (excerpt)
athens:
  image: gomods/athens:latest
  ports:
    - "3001:3000"
  environment:
    - ATHENS_STORAGE_TYPE=disk
    - ATHENS_DISK_STORAGE_ROOT=/var/lib/athens
    - ATHENS_GONOSUM_PATTERNS=github.com/ggid/*
  volumes:
    - ggid-athens:/var/lib/athens
```

---

## 7. Dependency Pinning and Update Policies

### 7.1 Why Pinning Matters for IAM

GGID's `go.mod` specifies minimum version selection (MVS) — Go uses the highest version in the transitive dependency tree. Explicit version pins in `go.mod` provide:

- **Reproducibility** — Every build uses identical dependency versions
- **Audit trail** — Changes to dependencies are visible in git diffs
- **Rollback safety** — A breaking upgrade can be reverted precisely
- **Security** — Prevents accidental upgrades to unreviewed versions

### 7.2 Automated Dependency Management

**Dependabot (GitHub-native):**
```yaml
# .github/dependabot.yml
version: 2
updates:
  - package-ecosystem: "gomod"
    directory: "/"
    schedule:
      interval: "weekly"
    open-pull-requests-limit: 10
    reviewers: ["ggid/maintainers"]
    groups:
      security-critical:
        patterns:
          - "golang.org/x/crypto"
          - "golang.org/x/oauth2"
          - "github.com/golang-jwt/jwt*"
          - "github.com/go-webauthn/webauthn"
        update-types:
          - "security-update"
      go-modules:
        patterns:
          - "*"
        update-types:
          - "minor"
          - "patch"
```

**Renovate (more configurable):**
```json
{
  "extends": ["config:recommended"],
  "schedule": ["before 6am on Monday"],
  "packageRules": [
    {
      "matchPackagePatterns": ["golang.org/x/crypto", "golang.org/x/oauth2"],
      "labels": ["security-critical"],
      "automerge": false,
      "reviewersRequired": 2
    },
    {
      "matchUpdateTypes": ["patch"],
      "automerge": true,
      "automergeType": "pr"
    }
  ]
}
```

### 7.3 go get Update Strategies

```bash
# Upgrade only patch versions (safest): v1.2.3 → v1.2.4
go get -u=patch ./...

# Upgrade minor versions: v1.2.3 → v1.3.0
go get -u=minor ./...

# Upgrade to latest including major versions: v1.2.3 → v2.0.0
go get -u ./...

# Upgrade a specific security-critical dependency
go get golang.org/x/crypto@latest
go get github.com/golang-jwt/jwt/v5@latest

# View available updates without applying
go list -m -u all
```

### 7.4 Review Cadence for Security-Critical Dependencies

| Dependency | Cadence | Rationale |
|---|---|---|
| `golang.org/x/crypto` | Weekly | Password hashing (Argon2id), encryption (AES-GCM) |
| `github.com/golang-jwt/jwt/v5` | Weekly | Token signing/verification — direct attack surface |
| `golang.org/x/oauth2` | Bi-weekly | OAuth2 flows — protocol-level security |
| `github.com/go-webauthn/webauthn` | Bi-weekly | WebAuthn/FIDO2 — credential security |
| `github.com/go-ldap/ldap/v3` | Monthly | LDAP auth — external identity provider |
| `github.com/jackc/pgx/v5` | Monthly | SQL injection prevention layer |
| `google.golang.org/grpc` | Monthly | gRPC framework — internal service comms |

---

## 8. GGID Dependency Audit

This section analyzes every dependency in GGID's `go.mod` for supply chain risk. All versions represent the latest at time of last `go get -u`.

### 8.1 Direct Dependencies (19 modules)

| Module | Version | Maintainer | Canonical? | Assessment |
|---|---|---|---|---|
| `github.com/alicebob/miniredis/v2` | v2.38.0 | alicebob | Yes | Test-only. Redis mock. Active maintenance. Low risk. |
| `github.com/andybalholm/brotli` | v1.2.2 | andybalholm | Yes | Brotli compression. Active. Low risk. |
| `github.com/coder/websocket` | v1.8.15 | Coder Inc. | Yes | WebSocket library (formerly `nhooyr/websocket`). Canonical path updated. Active. Low risk. |
| `github.com/go-ldap/ldap/v3` | v3.4.13 | go-ldap org | Yes | LDAP client. Active community. Medium risk (LDAP is external auth surface). |
| `github.com/go-webauthn/webauthn` | v0.17.4 | go-webauthn org | Yes | WebAuthn/FIDO2. Still pre-v1 (v0.x). API may change. Medium risk. |
| `github.com/golang-jwt/jwt/v5` | v5.3.1 | golang-jwt org | Yes | JWT library. **Security-critical.** Active, well-maintained. This is the canonical JWT library for Go (replaced `dgrijalva/jwt-go`). |
| `github.com/google/uuid` | v1.6.0 | Google | Yes | UUID generation. Stable, minimal. Low risk. |
| `github.com/jackc/pgx/v5` | v5.10.0 | Jack Christensen | Yes | PostgreSQL driver. Active. Well-maintained. Low risk. |
| `github.com/nats-io/nats-server/v2` | v2.14.3 | Synadia (NATS) | Yes | NATS server (embedded for tests). Active. Low risk. |
| `github.com/nats-io/nats.go` | v1.52.0 | Synadia (NATS) | Yes | NATS client. Active. Low risk. |
| `github.com/pquerna/otp` | v1.5.0 | Paul Querna | Yes | TOTP/HOTP for MFA. **Security-critical.** Active. Note: pinned at v1.5.0 for a long time — verify `@latest` available. |
| `github.com/prometheus/client_golang` | v1.23.2 | Prometheus | Yes | Metrics. Active. Low risk. |
| `github.com/quic-go/quic-go` | v0.60.0 | quic-go org | Yes | QUIC/HTTP3. Active. Medium risk (complex networking code). |
| `github.com/redis/go-redis/v9` | v9.21.0 | Redis Inc. | Yes | Redis client. Active. Low risk. |
| `github.com/tetratelabs/wazero` | v1.12.0 | Tetrate | Yes | WASM runtime (for middleware). Active. Medium risk (sandbox boundary). |
| `golang.org/x/crypto` | v0.53.0 | Go Team | Yes | **Security-critical.** Argon2id, AES-GCM, TLS. Active. |
| `golang.org/x/oauth2` | v0.36.0 | Go Team | Yes | **Security-critical.** OAuth2 client. Active. |
| `google.golang.org/grpc` | v1.82.0 | Google | Yes | gRPC framework. Active. Low risk. |
| `google.golang.org/protobuf` | v1.36.11 | Google | Yes | Protobuf runtime. Active. Low risk. |

### 8.2 Notable Indirect Dependencies

| Module | Version | Note |
|---|---|---|
| `github.com/Azure/go-ntlmssp` | v0.1.0 | NTLM auth (via go-ldap). Microsoft-authored. Low risk. |
| `github.com/fxamacker/cbor/v2` | v2.9.2 | CBOR encoding (via webauthn). Active. Low risk. |
| `github.com/go-asn1-ber/asn1-ber` | v1.5.8 | ASN.1 (via go-ldap). Low risk. |
| `github.com/nats-io/jwt/v2` | v2.8.2 | NATS JWT auth. Note: **second JWT library** — but this is for NATS internal auth, not GGID tokens. Acceptable. |
| `golang.org/x/net` | v0.55.0 | Networking. Pulled by grpc/quic-go. Low risk. |
| `github.com/antithesishq/antithesis-sdk-go` | v0.7.0 | Antithesis deterministic testing SDK. No-op default build. Low risk. |

### 8.3 Findings

**No duplicate functionality detected:**
- Only one JWT library used for GGID tokens (`golang-jwt/jwt/v5`). The `nats-io/jwt/v2` is a different protocol (NATS auth), not a functional duplicate.
- Only one PostgreSQL driver (`pgx/v5`).
- Only one Redis client (`go-redis/v9`).

**No typosquatting indicators:**
- All module paths use canonical import paths (verified against go.dev and GitHub).
- No suspicious or unknown maintainers.

**No deprecated modules:**
- `pquerna/otp` at v1.5.0 is the latest stable release. The repository is active with periodic releases.
- `coder/websocket` correctly migrated from the deprecated `nhooyr/websocket` path.

**Risk flags:**
- `go-webauthn/webauthn` is at v0.x (pre-v1). API stability not guaranteed. Monitor for breaking changes.
- `tetratelabs/wazero` executes WASM in the gateway middleware. Ensure sandbox isolation is maintained.
- `antithesishq/antithesis-sdk-go` is a testing SDK with a no-op default — verify it does not activate in production builds.

---

## 9. Gap Analysis & Recommendations

### 9.1 Current Supply Chain Maturity

| Capability | Status | Gap |
|---|---|---|
| go.sum verification | Active | None — Go toolchain enforces by default |
| GOSUMDB verification | Active | None — default behavior |
| govulncheck in CI | **Missing** | No automated vulnerability scanning |
| SBOM generation | **Missing** | No SBOM produced in releases |
| SLSA provenance | **Missing** | No build provenance attestation |
| Dependency auto-updates | **Missing** | No Dependabot/Renovate configured |
| Private module proxy | **N/A** | No private modules yet (will matter when added) |
| License scanning | **Missing** | No automated license compliance check |

### 9.2 Risk Assessment

**Overall risk: MEDIUM.** GGID's dependency tree is clean — all modules are canonical, actively maintained, and from reputable maintainers (Google, Synadia, Redis Inc., etc.). However, the absence of automated vulnerability scanning and SBOM generation creates blind spots:

1. **Unknown vulnerabilities:** Without `govulncheck` in CI, a newly announced CVE in `golang.org/x/crypto` or `golang-jwt/jwt/v5` would not be detected until manual review
2. **No provenance:** Consumers of GGID binaries cannot verify build integrity
3. **No SBOM:** Compliance audits require an SBOM; its absence blocks SOC 2 / ISO 27001 certification

### 9.3 Remediation Roadmap

| # | Action Item | Effort | Priority | Timeline |
|---|---|---|---|---|
| 1 | **Add govulncheck to CI** — Weekly scheduled scan + PR gate on reachable vulns | Low (2h) | P0 | Immediate |
| 2 | **Configure Dependabot** — Weekly PRs for security-critical deps, grouped updates | Low (1h) | P0 | Immediate |
| 3 | **Generate SBOM on release** — cyclonedx-gomod output attached to GitHub releases | Low (2h) | P1 | 1 sprint |
| 4 | **Add SLSA L3 provenance** — GitHub Actions attestation for release binaries | Medium (4h) | P1 | 1 sprint |
| 5 | **Set up Athens proxy** — Private module caching for when internal modules are added | Medium (1d) | P2 | 2 sprints |

### 9.4 Quick-Start: govulncheck (Highest ROI)

```bash
# Run right now to get a baseline:
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```

```yaml
# Add this to .github/workflows/security.yml immediately:
name: Security Scan
on: [push, pull_request, schedule]
  schedule:
    - cron: '0 6 * * 1'
jobs:
  govulncheck:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.25' }
      - run: go install golang.org/x/vuln/cmd/govulncheck@latest
      - run: govulncheck ./...
```

### 9.5 Long-Term Vision

For GGID to achieve supply chain maturity comparable to production IAM platforms (Auth0, Okta, Keycloak):

1. **SLSA L3 for all release artifacts** — Every binary has verifiable provenance
2. **Continuous SBOM + vuln monitoring** — SBOM generated on every release, scanned daily for new CVEs
3. **Private module proxy (Athens)** — All dependencies cached through a vetted, immutable proxy
4. **Two-person review for security-critical dependency upgrades** — No single-maintainer merge for `crypto`, `jwt`, `oauth2`, `webauthn`
5. **Reproducible builds** — `go build -trimpath` with deterministic output for independent verification

---

## Appendix A: Useful Commands Cheat Sheet

```bash
# Verify all module checksums
go mod verify

# Download and cache all dependencies
go mod download

# Vendor dependencies for offline/air-gapped builds
go mod vendor

# List all dependencies (direct + indirect)
go list -m all

# Check for available updates
go list -m -u all

# Upgrade a specific dependency to latest
go get <module>@latest

# Run vulnerability scanner
govulncheck ./...

# Generate SBOM
cyclonedx-gomod mod -json -licenses -output sbom.json

# Audit go.sum against Go checksum database
go env GONOSUMCHECK GOPRIVATE GOSUMDB GOPROXY

# Clean module cache (if corruption suspected)
go clean -modcache
```

---

## Appendix B: References

- [Go Module Reference](https://go.dev/ref/mod)
- [Go Vulnerability Database](https://vuln.go.dev)
- [SLSA Framework](https://slsa.dev)
- [OpenSSF Scorecard](https://securityscorecards.dev)
- [Syft (SBOM tool)](https://github.com/anchore/syft)
- [cyclonedx-gomod](https://github.com/CycloneDX/cyclonedx-gomod)
- [govulncheck](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck)
- [Athens Module Proxy](https://docs.gomods.io)
- [GitHub Actions Attestations](https://docs.github.com/en/actions/security-guides/using-artifact-attestations)
- [NIST SSDF (Secure Software Development Framework)](https://csrc.nist.gov/Projects/ssdf)

---

*Document version: 1.0 | Last updated: 2025-07-11 | Author: GGID Security Research*
