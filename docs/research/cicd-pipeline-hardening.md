# CI/CD Pipeline & GitOps Hardening: Production-Grade Delivery for GGID

> **Focus**: Upgrading GGID's existing GitHub Actions CI to production-grade — pre-commit hooks, branch protection, container security, GitOps deployment, migration testing, contract testing, and performance regression detection.
>
> **Author**: ggcxf (researcher) | **Date**: 2026-07-17 | **Status**: Research Complete
>
> **Checklist Compliance**: DoD per backlog item (§7).

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [GGID Current State: CI/CD](#2-ggid-current-state-cicd)
3. [Gap Analysis](#3-gap-analysis)
4. [Pre-Commit Hooks](#4-pre-commit-hooks)
5. [GitHub Actions Optimization](#5-github-actions-optimization)
6. [Container Security](#6-container-security)
7. [GitOps Deployment](#7-gitops-deployment)
8. [Database Migration CI](#8-database-migration-ci)
9. [Contract & Performance Testing](#9-contract--performance-testing)
10. [Implementation Backlog with DoD](#10-implementation-backlog-with-dod)
11. [Competitive Differentiation](#11-competitive-differentiation)

---

## 1. Executive Summary

GGID has a **functional CI pipeline** with 4 GitHub Actions workflows:
- `ci.yml` — Build + test + coverage ✅
- `coverage.yml` — Coverage reporting ✅
- `publish-node-sdk.yml` — npm SDK publish ✅
- `release.yml` — Release pipeline ✅

Makefile targets: proto, build, test, test-short, test-race, coverage, lint, migrate-up/down, docker-run/stop ✅

**What's working well:**
- Go module caching in CI ✅
- Build + test + coverage in single job ✅
- Timeout enforcement (15 min) ✅
- Coverage generation ✅

**Key gaps:**
1. **No lint in CI** — `make lint` exists but not in workflow
2. **No pre-commit hooks** — No golangci-lint/tsc before commit
3. **No branch protection** — Direct push to main possible
4. **No container scanning** — No Trivy/Snyk image scan
5. **No migration testing** — Migrations not tested before merge
6. **No contract testing** — OpenAPI not validated
7. **No performance regression** — No benchmark CI job
8. **No GitOps** — Manual kubectl deploy
9. **No parallel jobs** — Single job runs everything sequentially
10. **No npm caching** — Console build slow

---

## 2. GGID Current State: CI/CD

### Existing Workflows

| Workflow | File | Purpose | Status |
|----------|------|---------|--------|
| CI | `.github/workflows/ci.yml` | Build + test + coverage | ✅ Works |
| Coverage | `coverage.yml` | Coverage reporting | ✅ Works |
| Node SDK | `publish-node-sdk.yml` | npm publish | ✅ Works |
| Release | `release.yml` | Release pipeline | ✅ Works |

### CI Pipeline (Current)

```yaml
# ci.yml — single job, sequential
jobs:
  build-and-test:
    steps:
      - checkout
      - setup-go (with cache)
      - go mod download
      - go build ./...
      - make test (timeout 15min)
      - generate coverage
```

---

## 3. Gap Analysis

| # | Gap | Impact |
|---|-----|--------|
| 1 | No lint in CI | Code quality issues merged |
| 2 | No pre-commit hooks | Bad commits pushed |
| 3 | No branch protection | Main branch can be corrupted |
| 4 | No container scanning | Vulnerable images deployed |
| 5 | No migration testing | Bad migrations break production |
| 6 | No contract testing | SDK/API drift |
| 7 | No perf regression | Silent performance degradation |
| 8 | No GitOps | Manual deploy error-prone |
| 9 | No parallel jobs | CI slow (sequential) |
| 10 | No npm cache | Console build 5+ min |

---

## 4. Pre-Commit Hooks

### `.pre-commit-config.yaml`

```yaml
repos:
  - repo: https://github.com/golangci/golangci-lint
    rev: v1.62.0
    hooks:
      - id: golangci-lint
        args: [--config=.golangci.yml]

  - repo: https://github.com/dnephin/pre-commit-golang
    rev: v0.0.33
    hooks:
      - id: go-fmt
      - id: go-vet
      - id: go-imports

  - repo: https://github.com/pre-commit/pre-commit-hooks
    rev: v4.6.0
    hooks:
      - id: trailing-whitespace
      - id: end-of-file-fixer
      - id: check-yaml
      - id: check-added-large-files
      - id: detect-private-key

  - repo: local
    hooks:
      - id: tsc
        name: TypeScript compile check
        entry: bash -c 'cd console && npx tsc --noEmit'
        language: system
        files: \.(ts|tsx)$
```

---

## 5. GitHub Actions Optimization

### Parallel Job Matrix

```yaml
jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.25', cache: true }
      - run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
      - run: golangci-lint run ./...

  test-go:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:16
        env: { POSTGRES_PASSWORD: test }
      redis:
        image: redis:7
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.25', cache: true }
      - run: make test

  build-console:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with: { node-version: '20', cache: 'npm', cache-dependency-path: console/package-lock.json }
      - run: cd console && npm ci && npm run build

  scan:
    needs: [test-go, build-console]
    runs-on: ubuntu-latest
    steps:
      - uses: aquasecurity/trivy-action@master
        with: { severity: 'CRITICAL,HIGH' }

  migration-test:
    runs-on: ubuntu-latest
    services:
      postgres: { image: postgres:16 }
    steps:
      - run: make migrate-up  # Test all migrations apply cleanly
```

### Optimizations

| Optimization | Impact | Effort |
|-------------|--------|--------|
| Go module cache | -2 min | Already ✅ |
| npm cache | -3 min | Add setup-node cache |
| Parallel jobs (lint+test+console) | -5 min | Split into jobs |
| Test -short by default | -3 min | CI-specific flag |
| Build cache (act/docker) | -1 min | Cache Docker layers |

---

## 6. Container Security

### Multi-Stage Dockerfile

```dockerfile
# Build stage
FROM golang:1.25-alpine AS builder
RUN apk add --no-cache git ca-certificates
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags='-s -w' -o /svc ./services/gateway/cmd

# Runtime stage (distroless)
FROM gcr.io/distroless/static-debian12
COPY --from=builder /svc /svc
USER nonroot:nonroot
ENTRYPOINT ["/svc"]
```

### Image Scanning (Trivy)

```yaml
  scan:
    steps:
      - uses: aquasecurity/trivy-action@master
        with:
          image-ref: 'ghcr.io/ggid/gateway:latest'
          severity: 'CRITICAL,HIGH'
          fail-on-vulnerability: true
```

---

## 7. GitOps Deployment

### ArgoCD for k3s

```yaml
# deploy/argocd/application.yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: ggid
spec:
  source:
    repoURL: https://github.com/topcheer/ggid
    path: deploy/k8s
    targetRevision: HEAD
  destination:
    server: https://kubernetes.default.svc
    namespace: ggid
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
```

### GitOps Flow

```
Developer pushes to main
  → CI runs (build + test + scan)
  → CI builds container image → pushes to registry
  → CI updates k8s manifest with new image tag
  → ArgoCD detects manifest change
  → ArgoCD syncs to k3s cluster
  → Rolling update deployed automatically
```

---

## 8. Database Migration CI

```yaml
  migration-test:
    services:
      postgres:
        image: postgres:16
        env: { POSTGRES_DB: ggid_test, POSTGRES_PASSWORD: test }
    steps:
      - run: |
          # Apply all migrations to fresh DB
          make migrate-up DB_URL=postgres://test:test@localhost/ggid_test

          # Verify schema
          psql -c "\dt" $DB_URL

          # Test rollback last migration
          make migrate-down DB_URL=$DB_URL
          make migrate-up DB_URL=$DB_URL  # Re-apply
```

---

## 9. Contract & Performance Testing

### OpenAPI Contract Test

```yaml
  contract:
    steps:
      - run: |
          # Generate OpenAPI spec from code
          go run ./cmd/openapi-gen > openapi.json

          # Validate spec
          npx @redocly/cli lint openapi.json

          # Check no breaking changes
          npx oasdiff breaking-change old.json openapi.json
```

### Performance Regression

```yaml
  bench:
    steps:
      - run: go test -bench=. -benchmem -count=3 ./... | tee bench-results.txt
      - uses: benchmark-action/github-action-benchmark@v1
        with:
          tool: 'go'
          output-file-path: bench-results.txt
          alert-threshold: '110%'  # Alert if >10% slower
          comment-on-alert: true
```

---

## 10. Implementation Backlog with DoD

### P0 — CI Hardening (1 sprint)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 1 | Pre-commit hooks (golangci-lint + go-fmt + tsc + secret detection) | ✅ .pre-commit-config.yaml ✅ All hooks pass ✅ ≥3 local test | 2d |
| 2 | Parallel CI jobs (lint + test + console build) | ✅ 3 parallel jobs ✅ npm cache ✅ CI time < 5min | 2d |
| 3 | Branch protection rules | ✅ Require PR review ✅ Require CI green ✅ No direct push to main | 1d |
| 4 | golangci-lint in CI | ✅ Lint job ✅ Fail on issues ✅ ≥3 verified | 1d |

### P1 — Security + GitOps (1 sprint)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 5 | Container scanning (Trivy) | ✅ Scan on every build ✅ Fail on CRITICAL ✅ ≥3 verified | 2d |
| 6 | Multi-stage Dockerfile (distroless) | ✅ Image < 50MB ✅ Non-root user ✅ ≥3 verified | 2d |
| 7 | Migration testing in CI | ✅ Fresh PG + all migrations ✅ Rollback test ✅ ≥3 verified | 2d |
| 8 | ArgoCD GitOps for k3s | ✅ Auto-deploy from main ✅ Self-heal ✅ ≥3 verified | 3d |

### P2 — Advanced (Future)

| # | Task | DoD |
|---|------|-----|
| 9 | OpenAPI contract testing | Breaking change detection |
| 10 | Performance regression CI | Benchmark + alert on >10% regression |
| 11 | Semantic release | Auto-version from conventional commits |
| 12 | Canary deployment via GitOps | Route % traffic to new version |

---

## 11. Competitive Differentiation

| Feature | GGID (target) | Okta | Auth0 | Keycloak |
|---------|---------------|------|-------|----------|
| **CI pipeline** | GitHub Actions | Internal | GitHub Actions | GitHub Actions |
| **Lint** | golangci-lint | Internal | ESLint+TS | Checkstyle |
| **Container scan** | Trivy | Internal | Snyk | None |
| **GitOps** | ArgoCD | Internal | Spinnaker | None |
| **Migration CI** | Fresh PG test | Internal | Internal | Manual |
| **Contract test** | OpenAPI diff | Internal | Internal | None |
| **Open source** | Yes | No | No | Yes |

---

## References

- [GitHub Actions Best Practices](https://docs.github.com/en/actions/learn-github-actions/usage-limits-billing-and-administration)
- [golangci-lint](https://golangci-lint.run/) — Go linter
- [Trivy](https://aquasecurity.github.io/trivy/) — Container scanner
- [ArgoCD](https://argoproj.github.io/argo-cd/) — GitOps for K8s
- [pre-commit](https://pre-commit.com/) — Git hook framework
- [oasdiff](https://github.com/Tufin/oasdiff) — OpenAPI breaking change detection
- [benchmark-action](https://github.com/benchmark-action/github-action-benchmark) — Performance tracking
- [GGID CI Workflow](../.github/workflows/ci.yml) — Current pipeline
- [GGID Makefile](../Makefile) — Build targets
