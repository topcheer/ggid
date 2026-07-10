# Contributing Quick Start

5-minute guide to making your first contribution to GGID.

## 1. Fork & Clone

```bash
git clone https://github.com/YOUR_USERNAME/ggid.git
cd ggid
git remote add upstream https://github.com/ggid/ggid.git
```

## 2. Create Branch

```bash
git checkout -b feat/my-feature
```

Naming: `feat/`, `fix/`, `docs/`, `test/`, `chore/`

## 3. Code

Follow existing patterns:
- Services depend on **repo interfaces** (not concrete types)
- Use `pkg/errors.GGIDError` for domain errors
- Pass `context.Context` as first param
- `tenant_id` required on all multi-tenant queries
- Dependencies must use `@latest`

## 4. Test

```bash
go build ./...          # must pass first
make test               # all unit tests
go test -v ./services/YOUR_SVC/... -run TestYourFunc
```

## 5. PR

```bash
git add <specific files>
git commit -m "feat: add awesome feature

Brief description of what and why.

Co-Authored-By: ggcode <noreply@ggcode.dev>"
git push origin feat/my-feature
```

Open PR on GitHub. Ensure:
- [ ] `go build ./...` passes
- [ ] `make test` passes (0 FAIL)
- [ ] `gofmt -l .` returns nothing
- [ ] No new dependencies without justification
- [ ] Tests added for new logic

That's it. Happy hacking!
