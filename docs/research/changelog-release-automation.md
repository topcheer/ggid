# Changelog & Release Notes Automation for GGID

> **Focus**: Automated changelog generation from conventional commits, semantic versioning strategy, release notes templates, GitHub Releases integration, migration guides, and deprecation policy.
>
> **Author**: ggcxf (researcher) | **Date**: 2026-07-17 | **Status**: Research Complete

---

## 1. Executive Summary

GGID has **no changelog, no version tags, no release automation**. Commits are free-form messages. This makes it impossible for users to track what changed between versions or plan upgrades.

**Recommendation**: Adopt conventional commits + git-cliff for auto-changelog generation + GitHub Releases via CI on tag push.

---

## 2. Conventional Commits Strategy

### Commit Message Format

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Types

| Type | Meaning | Changelog Section |
|------|---------|-------------------|
| `feat` | New feature | ✨ Features |
| `fix` | Bug fix | 🐛 Bug Fixes |
| `security` | Security fix | 🔒 Security |
| `perf` | Performance improvement | ⚡ Performance |
| `refactor` | Code refactoring | (hidden) |
| `docs` | Documentation | 📚 Documentation |
| `test` | Tests | (hidden) |
| `chore` | Maintenance | (hidden) |
| `ci` | CI/CD | (hidden) |
| `breaking` | Breaking change | ⚠️ Breaking Changes |

### Scope Examples

```
feat(auth): add passkey self-enrollment
fix(oauth): token exchange delegation chain order
security(audit): fix hash chain verification bypass
perf(gateway): add response compression
feat(policy): unified PDP with risk overlay
docs(research): add multi-region active-active doc
```

### Migration from Current Style

Current: `research: Risk-Based Adaptive Authentication Engine + 6 backlog items`
Target: `feat(research): risk-based adaptive authentication engine + 6 backlog items`

Add `.commitlintrc` or rely on git-cliff parsing flexibility.

---

## 3. Semantic Versioning Strategy

### GGID Version Phases

| Phase | Version | Criteria |
|-------|---------|----------|
| **Development** | `0.x.y` | Breaking changes allowed freely |
| **Beta** | `0.9.x` | Feature-complete, API stabilizing |
| **v1.0** | `1.0.0` | API frozen, production-tested, full docs |
| **Post-1.0** | `1.x.0` (minor) / `1.0.x` (patch) | SemVer strictly enforced |

### v1.0 Release Criteria (Checklist)

- [ ] All P0 backlog items completed
- [ ] 786+ endpoints documented (OpenAPI 3.1)
- [ ] Production hardening checklist (50+ items) verified
- [ ] Load testing baseline established
- [ ] DR backup system operational
- [ ] PostgreSQL RLS enforced on all tables
- [ ] 99.9% uptime demonstrated (30-day soak)
- [ ] Security audit (external) completed
- [ ] README + CONTRIBUTING + LICENSE published
- [ ] 11 SDKs at production-ready tier

### Version Bump Rules

| Change Type | Version Bump | Example |
|-------------|-------------|---------|
| Breaking API change | MAJOR | `2.0.0` |
| New feature (backward-compatible) | MINOR | `1.1.0` |
| Bug fix (backward-compatible) | PATCH | `1.0.1` |
| Security fix | PATCH (or MINOR if new config) | `1.0.2` |

---

## 4. git-cliff Configuration

### `cliff.toml`

```toml
[changelog]
header = """
# Changelog\n
All notable changes to GGID are documented here.\n
The format is based on [Conventional Commits](https://conventionalcommits.org).
"""
body = """
{% if version %}\
    ## [{{ version | trim_start_matches(pat="v") }}] - {{ timestamp | date(format="%Y-%m-%d") }}
{% else %}\
    ## [unreleased]
{% endif %}\
{% for group, commits in commits | group_by(attribute="group") %}
    ### {{ group | upper_first }}
    {% for commit in commits %}
        - {{ commit.message | upper_first }}\
    {% endfor %}
{% endfor %}\n
"""
trim = true

[git]
conventional_commits = true
filter_unconventional = false

commit_parsers = [
    { message = "^feat", group = "Features" },
    { message = "^fix", group = "Bug Fixes" },
    { message = "^security", group = "Security" },
    { message = "^perf", group = "Performance" },
    { message = "^docs", group = "Documentation" },
    { message = "^refactor", group = "Refactoring" },
    { message = "^test", skip = true },
    { message = "^chore", skip = true },
    { message = "^ci", skip = true },
]
```

### Generation Command

```bash
# Generate CHANGELOG.md from commits since last tag
git cliff --tag v0.9.0 -o CHANGELOG.md

# Generate for specific range
git cliff v0.8.0..v0.9.0

# Output to stdout for GitHub Release body
git cliff --latest --strip header
```

---

## 5. GitHub Releases Integration

### CI Workflow: Auto-Release on Tag

```yaml
# .github/workflows/release-automated.yml
name: Release
on:
  push:
    tags: ['v*']

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with: { fetch-depth: 0 }  # Full history for changelog

      - name: Generate changelog
        run: |
          cargo install git-cliff
          git cliff --latest --strip header > RELEASE_NOTES.md

      - name: Create GitHub Release
        uses: softprops/action-gh-release@v2
        with:
          body_path: RELEASE_NOTES.md
          generate_release_notes: false
          draft: false
          prerelease: ${{ contains(github.ref, '-rc') }}

      - name: Build and attach binaries
        run: |
          go build -ldflags='-s -w' -o ggid-linux-amd64 ./cmd/ggid
          # ... per-service binaries or single binary
```

---

## 6. Release Notes Template

```markdown
## GGID v0.9.0 — 2026-07-17

### ✨ Features
- feat(auth): passkey self-enrollment without admin
- feat(policy): unified PDP with risk overlay (KB-161)
- feat(identity): PostgreSQL RLS on 27 tables (KB-212)

### 🔒 Security
- security(audit): 8 new ITDR detection rules (MITRE ATT&CK)
- security(gateway): CORS + security headers enforcement
- security(session): device fingerprint binding

### 🐛 Bug Fixes
- fix(oauth): token exchange delegation chain order
- fix(auth): session timeout not applied on refresh

### ⚠️ Breaking Changes
- BREAKING(identity): `/api/v1/users` response format changed
  - Migration: update clients to use `data` wrapper
  - See: docs/migrations/v0.9.0.md

### 📦 Upgrade Guide
1. `git pull && make build`
2. `make migrate-up` (applies migration 036)
3. Restart services
4. Verify: `curl /healthz`

### 📊 Stats
- 241 backlog items (KB-001 to KB-241)
- 43 research documents
- 61/61 test packages passing
- 786+ API endpoints
```

---

## 7. Deprecation Policy

| Stage | Duration | Action |
|-------|----------|--------|
| **Announce** | 2 minor versions before removal | Add `@deprecated` annotation + OpenAPI `deprecated: true` |
| **Warn** | 1 minor version before | API returns `Deprecation` + `Sunset` headers (RFC 8594) |
| **Remove** | Major version | Endpoint removed, changelog documents migration |

### RFC 8594 Deprecation Headers

```
HTTP/1.1 200 OK
Deprecation: Sun, 30 Sep 2026 00:00:00 GMT
Sunset: Wed, 31 Dec 2026 00:00:00 GMT
Link: </api/v2/users>; rel="successor-version"
```

---

## 8. Migration Guide Template

```markdown
# Migration Guide: v0.8.x → v0.9.0

## Database Migrations
Run `make migrate-up` — applies migrations 030-036.

## API Changes
| Old | New | Notes |
|-----|-----|-------|
| `/api/v1/users` (array response) | `/api/v1/users` (paginated `{data, total, cursor}`) | Use `cursor` param |

## Config Changes
```yaml
# New required config
security:
  cors:
    enabled: true
    allowed_origins: ["https://console.corp.com"]
```

## Rolling Upgrade Steps
1. Pull latest: `git pull origin main`
2. Backup DB: `pg_dump ggid > backup.sql`
3. Build: `make build`
4. Migrate: `make migrate-up`
5. Restart: `docker-compose restart`
6. Verify: `curl /healthz` returns 200
```

---

## 9. Implementation Backlog with DoD

### P0 — Release Infrastructure (1 sprint)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 1 | Adopt conventional commits + commitlint | ✅ .commitlintrc ✅ Pre-commit hook ✅ CI verifies | 1d |
| 2 | git-cliff setup + CHANGELOG.md | ✅ cliff.toml ✅ CHANGELOG.md generated ✅ CI job | 1d |
| 3 | GitHub Release workflow | ✅ Auto-create on tag ✅ Release notes from cliff ✅ Binary attachments | 2d |
| 4 | Tag v0.9.0 + first formal release | ✅ Release notes generated ✅ Migration guide ✅ Assets attached | 1d |

### P1 — Deprecation + Migration Framework

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 5 | Deprecation header middleware | ✅ RFC 8594 headers ✅ Configurable per-endpoint ✅ ≥3 tests | 2d |
| 6 | Migration guide template + docs/v0.9.0.md | ✅ Template ✅ First real guide ✅ Published | 1d |

### P2 — Advanced

| # | Task | DoD |
|---|------|-----|
| 7 | Release notes i18n (EN + CN) | Bilingual changelog |
| 8 | Release metrics dashboard | Track adoption |
| 9 | Pre-release channel (v0.x-rc) | Release candidate workflow |
| 10 | Automatic migration testing | CI tests upgrade path |

---

## 10. Competitive Differentiation

| Feature | GGID (target) | Auth0 | Okta | Keycloak |
|---------|---------------|-------|------|----------|
| **Conventional commits** | Target | Internal | Internal | No |
| **Auto changelog** | git-cliff | Manual | Manual | Manual |
| **Semantic versioning** | Target | Yes | Yes | Yes |
| **Migration guides** | Target | Yes | Yes | Minimal |
| **Deprecation policy** | RFC 8594 | Custom | Custom | No |
| **Release binaries** | Target | N/A | N/A | Yes |
| **Open source** | Yes | No | No | Yes |

---

## References

- [Conventional Commits](https://conventionalcommits.org/) — Commit message spec
- [git-cliff](https://git-cliff.org/) — Changelog generator
- [Semantic Versioning](https://semver.org/) — Version numbering
- [RFC 8594 (Deprecation)](https://datatracker.ietf.org/doc/html/rfc8594) — Sunset headers
- [softprops/action-gh-release](https://github.com/softprops/action-gh-release) — GitHub Release action
- [GGID CI Workflows](../.github/workflows/) — Existing GitHub Actions
- [GGID Makefile](../Makefile) — Build targets
- [GGID Kanban](../docs/kanban.md) — Backlog for v1.0 criteria
