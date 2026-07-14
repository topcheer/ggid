# Docker E2E Infrastructure Gap Analysis

*Research document: P2/P3 backlog input for GGID DevOps/QA team.*

## Executive Summary

| Item | Status |
|------|--------|
| Target test suite | `deploy/e2e-docker-test.sh` (11 tests) |
| Root cause | Docker Compose production config requires secrets that are not provided by default, causing immediate startup failure. |
| Impact | New E2E regression cannot run in CI without a configured `.env` file; local reproduction is blocked. |
| Recommended priority | P2 (blocks Docker-based E2E regression) |

## Failure Symptom

Running `docker compose -f deploy/docker-compose.prod.yaml up` immediately fails with:

```text
time="..." level=warning msg="The \"POSTGRES_PASSWORD\" variable is not set."
time="..." level=warning msg="The \"REDIS_PASSWORD\" variable is not set."
error while interpolating services.console.environment.NEXTAUTH_SECRET:
required variable NEXTAUTH_SECRET is missing a value:
Set NEXTAUTH_SECRET in .env
```

The same set of missing variables would fail `deploy/e2e-docker-test.sh` if it were to start the stack, because the test script itself does not provision a `.env` file.

## Required Variables

The production compose file declares the following variables as required (`${VAR:?error}`) or uses them without defaults:

| Variable | Used By | Sensitivity | Recommendation |
|----------|---------|-------------|----------------|
| `POSTGRES_PASSWORD` | postgres, keygen, migrate, auth, identity, oauth | High | Generate random 32-char secret |
| `REDIS_PASSWORD` | redis, gateway, auth, oauth | High | Generate random 32-char secret |
| `NEXTAUTH_SECRET` | console | High | Generate random 32-char secret |
| `JWT_PRIVATE_KEY` / `JWT_PUBLIC_KEY` | auth, gateway, oauth | Very High | Use `keygen` service or mounted files |

Optional but recommended variables missing in a typical local/CI run:

| Variable | Default / Current | Note |
|----------|-------------------|------|
| `POSTGRES_USER` | `ggid` | OK |
| `POSTGRES_DB` | `ggid` | OK |
| `APP_URL` | `http://localhost:3000` | OK for local |
| `API_URL` | `http://gateway:8080` | OK for local |

## Root Cause Breakdown

### 1. Missing `.env` Provisioning

`deploy/docker-compose.prod.yaml` explicitly instructs users to `cp .env.example .env` in its header comment, but the repository does not ship an `.env.example` file that satisfies all required variables. Users must manually discover the required secrets.

### 2. NEXTAUTH_SECRET is Required Without Default

`services.console.environment.NEXTAUTH_SECRET` uses the `?` modifier:

```yaml
services:
  console:
    environment:
      NEXTAUTH_SECRET: ${NEXTAUTH_SECRET:?Set NEXTAUTH_SECRET in .env}
```

This causes a hard failure before any container can be created. For a local E2E run, this secret can be auto-generated.

### 3. JWT Key Pair Dependency

The `auth`, `oauth`, and `gateway` services expect mounted RSA key files at `/etc/ggid/jwt.pem` and `/etc/ggid/jwt.pub`. The `keygen` service generates these once, but the volume mount and ordering dependency must be present.

### 4. E2E Script Does Not Manage Lifecycle

`deploy/e2e-docker-test.sh` only sends HTTP requests. It assumes the operator has already started the stack. There is no single command that:

1. Generates secrets
2. Starts the stack
3. Waits for healthchecks
4. Runs the E2E tests
5. Tears down the stack

## Recommended Fixes

### Immediate (P2)

1. **Create `deploy/.env.example`** with all required variables and sensible local defaults.
2. **Create a `deploy/start-e2e.sh` wrapper** that:
   - Generates `POSTGRES_PASSWORD`, `REDIS_PASSWORD`, `NEXTAUTH_SECRET` if not set
   - Runs `docker compose -f deploy/docker-compose.prod.yaml up -d`
   - Polls `http://localhost:8080/healthz` until ready
   - Invokes `deploy/e2e-docker-test.sh`
   - Exits with the E2E script's status code

### Short-term (P3)

1. **Add `make docker-e2e` target** in `Makefile` that runs the wrapper.
2. **Generate deterministic secrets for CI** from a hash of the commit SHA + CI run ID so that CI runs are reproducible but unique.
3. **Document the E2E setup** in `docs/guides/docker-e2e-testing.md` for new contributors.

### Long-term (P3)

1. **Replace required-secret `?` modifiers with generated defaults** in a non-production compose file (e.g., `deploy/docker-compose.e2e.yaml`) so that `docker compose up` works without any `.env`.
2. **Add a GitHub Actions workflow** that runs `make docker-e2e` on every PR.

## Example `deploy/.env.example`

```bash
# PostgreSQL
POSTGRES_USER=ggid
POSTGRES_PASSWORD=change-me-in-production
POSTGRES_DB=ggid

# Redis
REDIS_PASSWORD=change-me-in-production

# Console (NextAuth)
NEXTAUTH_SECRET=change-me-in-production

# URLs (Docker internal hostnames)
APP_URL=http://localhost:3000
API_URL=http://gateway:8080
AUTH_SECRET=change-me-in-production
```

## Example `deploy/start-e2e.sh`

```bash
#!/bin/bash
set -euo pipefail

cd "$(dirname "$0")"

export POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-$(openssl rand -hex 16)}"
export REDIS_PASSWORD="${REDIS_PASSWORD:-$(openssl rand -hex 16)}"
export NEXTAUTH_SECRET="${NEXTAUTH_SECRET:-$(openssl rand -hex 16)}"

COMPOSE="docker compose -f docker-compose.prod.yaml --env-file .env"

# Use a generated .env if none exists
if [ ! -f .env ]; then
  cat > .env <<EOF
POSTGRES_PASSWORD=$POSTGRES_PASSWORD
REDIS_PASSWORD=$REDIS_PASSWORD
NEXTAUTH_SECRET=$NEXTAUTH_SECRET
EOF
fi

$COMPOSE up -d

# Wait for gateway health
for i in {1..60}; do
  if curl -fs "http://localhost:8080/healthz" > /dev/null; then
    break
  fi
  sleep 2
done

# Run E2E tests
bash e2e-docker-test.sh
status=$?

$COMPOSE down
exit $status
```

## Impact Without Fix

| Scenario | Result |
|----------|--------|
| New contributor runs `docker compose up` | Immediate failure due to missing secrets |
| CI runs `e2e-docker-test.sh` | All 11 tests fail because the stack never started |
| Release verification | Cannot validate Docker deployment end-to-end |

## Verification After Fix

1. `cd deploy && rm -f .env && bash start-e2e.sh` should pass all 11 tests.
2. `make docker-e2e` should produce `11 PASS / 0 FAIL`.
3. `go test ./...` should remain unaffected (no Go code changes required).

## Related Files

- `deploy/docker-compose.prod.yaml` — production compose
- `deploy/docker-compose.yaml` — development compose (may not have the same secret requirements)
- `deploy/e2e-docker-test.sh` — current E2E script
- `deploy/e2e-k3s-test.sh` — K3s equivalent (does not share this issue)
- `deploy/README.md` — deployment documentation (does not mention E2E setup)

## Recommended Gap Status

This is a **real infrastructure gap**, not a false positive. It should be tracked in `docs/platform-completeness-report.md` as:

| # | Feature | Location | Issue | Status |
|---|---------|----------|-------|--------|
| — | Docker E2E environment | `deploy/` | Docker Compose E2E cannot start due to missing secrets and no wrapper script | [NEW] |

## Suggested Assignee

DevOps / backend integration teammate (owns `deploy/`).

## Status Update (2026-07-15)

- Root cause fixed: `deploy/docker-compose.yaml` migrate service had duplicated `sh` in command list.
- After fix: `docker compose up -d --build` starts all 12 containers healthy.
- E2E test: `bash deploy/e2e-docker-test.sh` → **11/11 PASS**.
- commit: 6f7d68e0
- This infra gap can be marked as [DONE] in the next platform scan.
