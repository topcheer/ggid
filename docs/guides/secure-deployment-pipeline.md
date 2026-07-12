# Secure Deployment Pipeline

This guide covers CI/CD security, build hardening, deployment gates, container security, supply chain security (SLSA), and GGID's CI/CD security gates.

## CI/CD Security Pipeline

```
Code Commit -> SAST -> SCA -> Secret Scan -> Build -> DAST -> Container Scan -> IaC Scan -> Deploy Gate -> Production
```

### SAST (Static Application Security Testing)

| Tool | Language | Purpose |
|---|---|---|
| gosec | Go | Go security analysis |
| staticcheck | Go | Go static analysis |
| Semgrep | Multi | Pattern-based analysis |

```yaml
ci_cd:
  sast:
    - tool: "gosec"
      command: "gosec -severity high ./..."
      fail_on: "high"
    - tool: "staticcheck"
      command: "staticcheck ./..."
      fail_on: "error"
```

### SCA (Software Composition Analysis)

| Tool | Purpose |
|---|---|
| govulncheck | Go vulnerability check |
| nancy | Go dependency scanner |
| trivy | Multi-language dependency scan |

### Secret Scanning

```yaml
ci_cd:
  secret_scan:
    - tool: "gitleaks"
      command: "gitleaks detect --source ."
      fail_on: "any"
```

### IaC Scanning

```yaml
ci_cd:
  iac_scan:
    - tool: "trivy"
      command: "trivy config deploy/"
      fail_on: "high"
```

## Build Hardening

### Reproducible Builds

```yaml
build:
  reproducible:
    flags:
      - "-trimpath"
      - "-ldflags='-s -w'"
    go_env:
      CGO_ENABLED: "0"
```

### SBOM (Software Bill of Materials)

```bash
syft . -o cyclonedx-json > sbom.json
trivy fs --format cyclonedx . > sbom.json
```

### Binary Signing

```bash
cosign sign-blob --key cosign.key ggid-auth
cosign verify-blob --key cosign.pub --signature ggid-auth.sig ggid-auth
```

## Deployment Gates

### Security Approval

```yaml
deployment:
  gates:
    security_approval:
      required: true
      approvers: ["security-team"]
      auto_approve_if:
        - "only_docs_changed"
        - "only_tests_changed"
      block_if:
        - "sast_high_findings"
        - "secret_leak_detected"
        - "container_scan_critical"
```

### Segregation of Duties

```yaml
deployment:
  segregation:
    rules:
      - "committer_cannot_approve"
      - "approver_cannot_deploy"
    minimum_approvers: 2
    require_security_review: true
```

### Environment Progression

```
Feature Branch -> CI Tests -> Staging -> Security Review -> Production
                     ^          ^           ^              ^
                Unit Tests   DAST     Approval Gate   Canary Deploy
```

## Container Security

### Image Scanning

```yaml
container:
  scan:
    enabled: true
    tool: "trivy"
    fail_on: "critical"
    scan_registry: true
    scan_on_build: true
    scan_on_deploy: true
```

### Container Hardening

```dockerfile
FROM gcr.io/distroless/static-debian12
USER nonroot:nonroot
# Read-only filesystem at runtime: --read-only
# No shell, no package manager (distroless)
```

### Runtime Protection

```yaml
container:
  runtime:
    security_context:
      run_as_non_root: true
      run_as_user: 1000
      read_only_root_filesystem: true
      allow_privilege_escalation: false
      capabilities:
        drop: ["ALL"]
    seccomp_profile: "runtime/default"
```

## Supply Chain Security (SLSA)

### SLSA Levels

| Level | Description | Requirements |
|---|---|---|
| L1 | Build process documented | Provenance exists |
| L2 | Hosted build service | Tamper-resistant provenance |
| L3 | Hardened build platform | Isolated builds, verified provenance |
| L4 | Two-party reviewed | Reproducible builds, verified |

### GGID SLSA Configuration

```yaml
slsa:
  level: 3
  provenance:
    enabled: true
    generator: "slsa-github-generator"
    sign: true
    verify_on_deploy: true
  requirements:
    isolated_builds: true
    verified_provenance: true
```

## GGID CI/CD Security Gates

### Full Pipeline

```yaml
name: Security Pipeline
on: [push, pull_request]

jobs:
  sast:
    steps:
      - uses: securego/gosec@master
        with: {args: "-severity high ./..."}
      - run: go vet ./...
      - run: staticcheck ./...

  sca:
    needs: sast
    steps:
      - run: govulncheck ./...
      - run: trivy fs --severity HIGH,CRITICAL .

  secret-scan:
    steps:
      - uses: gitleaks/gitleaks-action@v2

  build:
    needs: [sast, sca, secret-scan]
    steps:
      - run: go build -trimpath -ldflags='-s -w' ./...
      - run: syft . -o cyclonedx-json > sbom.json
      - run: cosign sign-blob --key ${{ secrets.COSIGN_KEY }} ggid-auth

  container-scan:
    needs: build
    steps:
      - run: trivy image gcr.io/ggid/auth:latest --severity CRITICAL

  dast:
    needs: container-scan
    steps:
      - run: zap-baseline.py -t https://staging.ggid.example.com

  deploy-gate:
    needs: [dast]
    environment: production
    steps:
      - run: echo "Security gates passed"
```

### Gate Requirements

| Gate | Required | Block On |
|---|---|---|
| SAST (gosec) | Yes | High severity |
| SCA (govulncheck) | Yes | High severity |
| Secret scan (gitleaks) | Yes | Any finding |
| Build (reproducible) | Yes | Build failure |
| SBOM generation | Yes | Generation failure |
| Binary signing | Yes | Signing failure |
| Container scan (trivy) | Yes | Critical findings |
| DAST (ZAP) | Staging only | High findings |
| Security approval | Production | Manual approval |

## Best Practices

1. **Fail fast** — Run SAST early, before build
2. **Block on critical** — Never deploy with critical findings
3. **Generate SBOM** — Always produce SBOM for every build
4. **Sign everything** — Sign binaries, containers, and SBOMs
5. **Verify on deploy** — Check signatures before deploying
6. **Scan at every stage** — Build, registry, and runtime scanning
7. **Enforce segregation** — Committers can't approve, approvers can't deploy
8. **Use SLSA L3+** — Hardened build platform with verified provenance
9. **Pin dependencies** — Use go.sum and vendor directory
10. **Monitor runtime** — Container runtime protection in production