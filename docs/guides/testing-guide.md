# Testing Guide

## Running Tests

### Full Suite
```bash
make test
```

### Specific Package
```bash
go test ./services/auth/... -count=1
go test ./pkg/auth/multihash/... -v
```

### With Coverage
```bash
go test ./... -cover -coverprofile=coverage.out
go tool cover -html=coverage.out  # opens browser
go tool cover -func=coverage.out  # text summary
```

### Race Detection
```bash
go test -race ./services/auth/...
```

## Test Standards

Every new feature must include:

1. **≥3 tests**: happy path, error case, edge case
2. **Nil-pool safety**: repositories must handle `pool == nil` without panic
```go
func (r *repo) Get(ctx context.Context, id uuid.UUID) (*Entity, error) {
    if r.pool == nil { return nil, nil }  // ← required
    ...
}
```
3. **Edge cases**: empty input, nil pointers, invalid UUIDs, concurrent access

## Checklist

- [ ] Tests pass: `go test ./path/to/pkg/... -v`
- [ ] No panics on nil pool/repo
- [ ] Build clean: `go build ./...`
- [ ] At least 3 test cases per new function
- [ ] Table-driven tests preferred
- [ ] No `time.Sleep` in tests (use channels or polling)
