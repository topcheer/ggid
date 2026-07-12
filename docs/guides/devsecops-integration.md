# DevSecOps Integration Guide

## Overview

DevSecOps integrates security practices into every stage of the software development lifecycle (SDLC), embedding security as a shared responsibility across development, security, and operations teams. This guide covers shift-left security, CI/CD security gates, security as code, automated remediation, security metrics, toolchain integration, and how GGID supports a DevSecOps pipeline.

## Shift-Left Security

### Principles

Shift-left security moves security testing and verification to the earliest stages of development, reducing the cost and effort of fixing vulnerabilities.

- **Early detection**: Identify vulnerabilities during design and coding, not after deployment
- **Developer empowerment**: Provide tools and feedback directly in the IDE and CI pipeline
- **Continuous feedback**: Security findings surface as code is written, not during periodic audits
- **Risk-based prioritization**: Focus on exploitable vulnerabilities, not theoretical issues

### Implementation Stages

| Stage | Security Activity | Owner |
|-------|-------------------|-------|
| Design | Threat modeling, architecture review | Security + Dev |
| Develop | IDE security plugins, pre-commit hooks | Developer |
| Build | SAST, SCA, secret scanning | CI pipeline |
| Test | DAST, fuzzing, penetration testing | QA + Security |
| Deploy | Container scanning, IaC scanning, policy enforcement | DevOps |
| Runtime | Runtime monitoring, anomaly detection | Operations |
| Post-release | Incident response, lessons learned | All teams |

### Threat Modeling

Threat modeling should be performed during the design phase for new features and significant changes.

- **Methodology**: STRIDE (Spoofing, Tampering, Repudiation, Information Disclosure, Denial of Service, Elevation of Privilege)
- **Data flow diagrams**: Map data flows to identify trust boundaries
- **Attack surface analysis**: Document entry points, authentication boundaries, and data stores
- **Mitigation tracking**: Record identified threats, mitigations, and residual risk
- **Review cadence**: Revisit threat models on architecture changes or new integrations

## CI/CD Security Gates

### Gate Architecture

Security gates are automated checkpoints in the CI/CD pipeline that block progression when security criteria are not met.

```
Commit -> [Pre-commit hooks] -> PR -> [SAST + SCA + Secret Scan] -> Build -> [Container Scan + IaC Scan] -> Test -> [DAST + Fuzz] -> Deploy -> [Policy Check + Config Audit] -> Runtime [Monitoring]
```

### SAST (Static Application Security Testing)

Analyzes source code for security vulnerabilities without executing the code.

- **Coverage**: Injection flaws (SQLi, XSS, command injection), hardcoded secrets, insecure crypto, path traversal, deserialization issues
- **Tools**: Semgrep, CodeQL, SonarQube, Snyk Code, GitHub CodeQL
- **Configuration**:
  - Run on every pull request and merge to main
  - Block merge on critical/high findings
  - Warn on medium/low findings with tracking issues
  - Custom rules for project-specific patterns (e.g., unsafe JWT handling)
- **Tuning**: Reduce false positives with rule suppression, baseline files, and severity calibration
- **Baseline**: Maintain a `.sast-baseline.json` to track accepted findings

### DAST (Dynamic Application Security Testing)

Tests running applications for security vulnerabilities by simulating attacks.

- **Coverage**: OWASP Top 10, injection, authentication bypass, session management, access control
- **Tools**: OWASP ZAP, Burp Suite, Nuclei, Nikto
- **Configuration**:
  - Run against staging environment after deployment
  - Authenticated scans using test credentials with limited scope
  - Schedule full scans nightly or weekly
  - Block production deploy on critical findings
- **Integration**: Trigger from CI pipeline after staging deployment

### SCA (Software Composition Analysis)

Identifies vulnerabilities in third-party dependencies and libraries.

- **Coverage**: CVE scanning, license compliance, transitive dependencies
- **Tools**: Snyk, Dependabot, OWASP Dependency-Check, Trivy, GoSec
- **Configuration**:
  - Scan `go.mod`, `package.json`, `pom.xml` on every build
  - Block on critical CVEs with available fixes
  - Auto-create PRs for dependency updates (Dependabot)
  - License policy: reject GPL-3.0, AGPL-3.0 in non-compliant projects
- **Renovate/Dependabot**: Automated dependency update PRs with security advisory priority

### Secret Scanning

Detects hardcoded secrets, API keys, tokens, and credentials in source code.

- **Tools**: GitGuardian, TruffleHog, GitHub Secret Scanning, Gitleaks
- **Configuration**:
  - Pre-commit hook for local detection
  - CI gate scanning full repository history on push
  - Block merge on any verified secret
  - Auto-revoke detected secrets via integration with vault/secret manager
- **Custom patterns**: Add regex patterns for internal token formats (e.g., GGID JWT signing keys)

### Container Scanning

Scans container images for vulnerabilities, misconfigurations, and embedded secrets.

- **Coverage**: OS package CVEs, application dependency CVEs, Dockerfile best practices, embedded secrets
- **Tools**: Trivy, Grype, Snyk Container, Aqua, Clair
- **Configuration**:
  - Scan every built image before pushing to registry
  - Block on critical CVEs with available fixes
  - Fail on Dockerfile issues (root user, no HEALTHCHECK, large image)
  - Sign images with Cosign/Sigstore
- **SBOM**: Generate Software Bill of Materials (SPDX/CycloneDX) for each image

### IaC Scanning

Scans infrastructure-as-code files for misconfigurations and security issues.

- **Coverage**: Terraform, Helm charts, Docker Compose, Kubernetes manifests
- **Tools**: Checkov, Terrascan, KICS, tfsec, kube-score
- **Configuration**:
  - Scan on every IaC change in PR
  - Block on critical misconfigurations (exposed databases, missing encryption, overly permissive IAM)
  - Enforce CIS benchmarks for Kubernetes and cloud providers
  - Custom policies for organizational standards

## Security as Code

### Policy as Code

Define and enforce security policies programmatically using policy engines.

#### OPA (Open Policy Agent)

```rego
# Example: Enforce JWT authentication for all API routes
package ggid.api

default allow = false

allow {
    input.request.method == "GET"
    input.request.path == "/healthz"
}

allow {
    input.token.valid == true
    input.token.tenant_id == input.request.headers["X-Tenant-ID"]
    input.token.scopes[_] == required_scope(input.request.path)
}

required_scope(path) = scope {
    some i
    route := routes[i]
    startswith(path, route.path)
    scope := route.scope
}

routes = [
    {"path": "/api/v1/users", "scope": "users:read"},
    {"path": "/api/v1/admin", "scope": "admin:all"},
    {"path": "/api/v1/audit", "scope": "audit:read"},
]
```

#### Cedar (AWS Policy Language)

```cedar
// Example: RBAC policy for user management
permit (
    principal in GGID::Role::"admin",
    action in [GGID::Action::"CreateUser", GGID::Action::"DeleteUser"],
    resource in GGID::Tenant::"tenant-001"
);

forbid (
    principal,
    action == GGID::Action::"DeleteUser",
    resource
) unless {
    principal != resource  // cannot delete self
};
```

### Pipeline Policy Enforcement

- **Gate policies**: OPA/Cedar policies evaluated at each CI/CD gate
- **Admission control**: Kubernetes admission webhooks enforce policies at deploy time
- **Runtime policies**: Continuous policy evaluation against runtime state
- **Policy versioning**: Policies stored in Git, reviewed via PR, versioned with semver

## Automated Remediation

### Vulnerability Auto-Fix

| Finding Type | Auto-Fix Strategy |
|-------------|-------------------|
| Dependency CVE | Dependabot/Renovate auto-PR with patched version |
| SAST finding | Semgrep autofix or CodeQL suggested fix |
| Container CVE | Rebuild with updated base image |
| IaC misconfig | Checkov auto-fix or suggested remediation |
| Secret leak | Auto-revoke + rotate via vault integration |
| License violation | Auto-replace with compliant alternative |

### Remediation Workflow

```
Detect -> Classify -> Assign -> Remediate -> Verify -> Close
  |         |         |         |         |
Scanner  Severity  Owner    Auto/Manual  Re-scan
         + CVSS   + SLA    + PR         + Gate
```

### SLA Targets

| Severity | Detection to Fix | Auto-Fix Eligible |
|----------|------------------|-------------------|
| Critical | 24 hours | Yes (dependency updates, container rebuilds) |
| High | 7 days | Yes (auto-PR for SAST findings) |
| Medium | 30 days | Partial (IaC fixes) |
| Low | 90 days | Manual |

## Security Metrics

### Key Metrics

| Metric | Description | Target |
|--------|-------------|--------|
| MTTR (Mean Time to Remediate) | Time from detection to fix | Critical < 24h, High < 7d |
| Security test coverage | % of code covered by SAST/DAST | > 90% |
| Dependency freshness | % dependencies on latest patch | > 95% |
| Secret leak incidents | Verified secrets in Git | 0 |
| Container vulnerability density | Critical CVEs per image | < 1 |
| Policy compliance rate | % resources passing policy checks | > 99% |
| Pipeline gate pass rate | % builds passing all security gates | > 95% |
| False positive rate | % findings that are false positives | < 10% |

### Dashboards

- **Pipeline Security Dashboard**: Gate pass rates, finding trends, MTTR by severity
- **Vulnerability Dashboard**: Open vulnerabilities by severity, age, component
- **Compliance Dashboard**: Policy compliance, drift detection, exception tracking
- **Dependency Health Dashboard**: Outdated dependencies, CVE exposure, update PRs

### Reporting Cadence

| Report | Frequency | Audience |
|--------|-----------|---------|
| Pipeline security summary | Daily | Dev + Sec teams |
| Vulnerability report | Weekly | Eng leadership |
| Compliance posture | Monthly | CISO + Compliance |
| Executive security summary | Quarterly | C-suite + Board |

## Toolchain Integration

### GitHub Actions

```yaml
# .github/workflows/security.yml
name: Security Pipeline

on: [pull_request, push]

jobs:
  sast:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Semgrep Scan
        uses: returntocorp/semgrep-action@v1
        with:
          config: >-
            p/owasp-top-ten
            p/golang
            p/security-audit
      - name: CodeQL Analysis
        uses: github/codeql-action/init@v3
        with:
          languages: go, javascript
      - name: CodeQL Autobuild
        uses: github/codeql-action/autobuild@v3
      - name: CodeQL Analyze
        uses: github/codeql-action/analyze@v3

  sca:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Go Dependency Scan
        run: |
          go install github.com/securego/gosec/v2/cmd/gosec@latest
          gosec -severity high ./...
      - name: Trivy FS Scan
        run: trivy fs --severity CRITICAL,HIGH .

  secrets:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - name: Gitleaks
        uses: gitleaks/gitleaks-action@v2

  container:
    runs-on: ubuntu-latest
    if: github.event_name == 'push'
    steps:
      - uses: actions/checkout@v4
      - name: Build Image
        run: docker build -t ggid:${{ github.sha }} .
      - name: Trivy Container Scan
        run: trivy image --severity CRITICAL,HIGH ggid:${{ github.sha }}
      - name: Generate SBOM
        run: trivy image --format spdx-json -o sbom.json ggid:${{ github.sha }}

  iac:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Checkov Scan
        run: checkov -d deploy/ --framework terraform,dockerfile

  dast:
    runs-on: ubuntu-latest
    needs: container
    steps:
      - name: Deploy to Staging
        run: kubectl apply -f deploy/staging/
      - name: OWASP ZAP Scan
        uses: zaproxy/action-baseline@v0.12.0
        with:
          target: 'https://staging.ggid.example.com'
```

### GitLab CI

```yaml
# .gitlab-ci.yml
include:
  - template: Security/SAST.gitlab-ci.yml
  - template: Security/Secret-Detection.gitlab-ci.yml
  - template: Security/Dependency-Scanning.gitlab-ci.yml
  - template: Security/Container-Scanning.gitlab-ci.yml

stages:
  - build
  - test
  - security
  - deploy

semgrep:
  stage: security
  image: returntocorp/semgrep
  script:
    - semgrep ci --config p/owasp-top-ten --config p/golang
  rules:
    - if: $CI_PIPELINE_SOURCE == "merge_request_event"

iac-scan:
  stage: security
  image: bridgecrew/checkov
  script:
    - checkov -d deploy/ --framework terraform
  rules:
    - changes:
        - deploy/**/*
```

### Jenkins

```groovy
// Jenkinsfile security pipeline
pipeline {
    agent any
    stages {
        stage('SAST') {
            steps {
                sh 'semgrep --config p/owasp-top-ten --config p/golang ./services/'
            }
        }
        stage('SCA') {
            steps {
                sh 'gosec -severity high ./...'
                sh 'trivy fs --severity CRITICAL,HIGH .'
            }
        }
        stage('Secret Scan') {
            steps {
                sh 'gitleaks detect --source . --report-format json --report-path leaks.json'
            }
        }
        stage('Container Scan') {
            when { branch 'main' }
            steps {
                sh 'trivy image --severity CRITICAL,HIGH ggid:${BUILD_NUMBER}'
            }
        }
        stage('Deploy Gate') {
            steps {
                sh 'opa eval -d policies/ -i deploy/manifest.json "data.ggid.allow"'
            }
        }
    }
    post {
        always {
            publishSecurityReport()
        }
    }
}
```

## GGID DevSecOps Pipeline

### Pipeline Architecture

GGID's DevSecOps pipeline integrates security into every stage of the monorepo CI/CD:

```
Developer -> IDE Security -> Pre-commit -> PR -> CI Gates -> Build -> Container Scan -> Staging Deploy -> DAST -> Policy Gate -> Production Deploy -> Runtime Monitoring
```

### GGID-Specific Security Checks

| Check | Tool | Gate |
|-------|------|------|
| JWT handling | Semgrep custom rule | Block on insecure JWT operations |
| SQL injection | Semgrep + GoSec | Block on raw SQL string concatenation |
| Tenant isolation | Custom Semgrep rule | Block on missing tenant_id in queries |
| Crypto usage | GoSec + custom rules | Block on MD5, SHA1, ECB mode |
| API auth | CodeQL | Block on routes without auth middleware |
| Secret in code | Gitleaks + GitGuardian | Block on any verified secret |
| Container root | Trivy + Hadolint | Block on USER root or no HEALTHCHECK |
| IaC exposure | Checkov | Block on publicly exposed DB/Redis |

### Pre-commit Hooks

```yaml
# .pre-commit-config.yaml
repos:
  - repo: https://github.com/semgrep/semgrep
    rev: v1.60.0
    hooks:
      - id: semgrep
        args: ['--config', 'p/owasp-top-ten', '--config', 'p/golang', '--error']
  - repo: https://github.com/gitleaks/gitleaks
    rev: v8.18.0
    hooks:
      - id: gitleaks
  - repo: https://github.com/dnephin/pre-commit-golang
    rev: v0.5.1
    hooks:
      - id: go-fmt
      - id: go-imports
      - id: go-vet
      - id: go-build
  - repo: https://github.com/hadolint/hadolint
    rev: v2.12.0
    hooks:
      - id: hadolint
```

### Runtime Security

- **SIEM integration**: Audit events forwarded to SIEM for correlation and alerting
- **Runtime anomaly detection**: Behavioral analysis of auth patterns via risk engine
- **Continuous compliance**: Automated policy checks against running infrastructure
- **Incident response**: Automated alerting with severity-based escalation

### Secrets Management

- **Vault**: HashiCorp Vault for secret storage and rotation
- **CI secrets**: GitHub Actions secrets / GitLab CI variables with masked output
- **Runtime secrets**: Vault sidecar injector or external secrets operator
- **Key rotation**: Automated rotation of JWT signing keys, database credentials, API keys

## Best Practices

1. **Fail fast, fail loud**: Security gates should block builds immediately with clear error messages
2. **Developer experience**: Security feedback should be actionable with suggested fixes
3. **False positive management**: Maintain baselines and suppression files to reduce noise
4. **Shift everything left**: Every security check that can run early should
5. **Automate everything**: Manual security reviews are a bottleneck - automate checks
6. **Measure and improve**: Track metrics, identify trends, continuously improve the pipeline
7. **Security champions**: Embed security advocates in each development team
8. **Threat-informed defense**: Prioritize security investments based on actual threat landscape
9. **Pipeline as code**: Store all pipeline configuration and security policies in Git
10. **Transparency**: Make security findings visible to all stakeholders

## See Also

- [OWASP Top 10](https://owasp.org/Top10/)
- [NIST Secure Software Development Framework (SSDF)](https://csrc.nist.gov/Projects/ssdf)
- [SLSA Framework](https://slsa.dev/)
- [Continuous Compliance Monitoring](./continuous-compliance-monitoring.md)
- [CI/CD Pipeline Security](./cicd-pipeline-security.md)
- [Security Monitoring Guide](./security-monitoring-guide.md)
