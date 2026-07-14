# Testing Strategy Guide

> How to test GGID integration: unit tests, integration tests, and E2E patterns.

---

## Testing Pyramid

```
        /\
       /E2E\          ← 5-10 tests (full flow)
      /------\
     /Integration\     ← 20-50 tests (API calls)
    /--------------\
   /    Unit Tests   \  ← 100+ tests (pure logic)
  /--------------------\
```

---

## Unit Tests (Go)

Test JWT verification logic without network:

```go
func TestVerifyToken_ValidJWT(t *testing.T) {
    // Generate test key
    key, _ := rsa.GenerateKey(rand.Reader, 2048)

    // Create test JWT
    claims := jwt.MapClaims{
        "sub": "usr_123",
        "tenant_id": "00000000-0000-0000-0000-000000000001",
        "exp": time.Now().Add(15 * time.Minute).Unix(),
        "scope": "read:users",
    }
    token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
    tokenString, _ := token.SignedString(key)

    // Verify
    verifier := NewTestVerifier(key)
    user, err := verifier.Verify(tokenString)

    assert.NoError(t, err)
    assert.Equal(t, "usr_123", user.UserID)
}
```

---

## Integration Tests (Go)

Test against a running GGID instance:

```go
func TestIntegration_LoginAndListUsers(t *testing.T) {
    client := ggid.New(os.Getenv("GGID_URL"), ggid.WithJWKS(5*time.Minute))

    // Register
    user, err := client.CreateUser(ctx, &ggid.CreateUserRequest{
        Username: "testuser",
        Email:    "test@test.com",
        Password: "Test1234!",
    })
    assert.NoError(t, err)

    // Login
    token, err := client.Login(ctx, "testuser", "Test1234!")
    assert.NoError(t, err)
    assert.NotEmpty(t, token.AccessToken)

    // List users with JWT
    authCtx := ggid.WithToken(ctx, token.AccessToken)
    users, err := client.ListUsers(authCtx)
    assert.NoError(t, err)
    assert.True(t, len(users) > 0)
}
```

Run with build tag:
```bash
go test -tags=integration ./test/integration/...
```

---

## E2E Tests (curl/bats)

```bash
#!/usr/bin/env bats

@test "register user" {
    run curl -s -X POST $GGID_URL/api/v1/auth/register \
      -H "Content-Type: application/json" \
      -H "X-Tenant-ID: $TENANT" \
      -d '{"username":"e2e_user","email":"e2e@test.com","password":"Test1234!"}'
    [ "$status" -eq 0 ]
    echo "$output" | jq -e '.id'
}

@test "login returns JWT" {
    run curl -s -X POST $GGID_URL/api/v1/auth/login \
      -H "Content-Type: application/json" \
      -H "X-Tenant-ID: $TENANT" \
      -d '{"username":"e2e_user","password":"Test1234!"}'
    JWT=$(echo "$output" | jq -r .access_token)
    [ ${#JWT} -gt 100 ]
}
```

---

## React Component Tests

```typescript
import { render, screen } from '@testing-library/react';
import { GGIDProvider, useAuth } from '@ggid/react';

function TestComponent() {
  const { isAuthenticated } = useAuth();
  return <div>{isAuthenticated ? 'Logged in' : 'Not logged in'}</div>;
}

test('shows not logged in by default', () => {
  render(
    <GGIDProvider domain="localhost:8080" tenantId="test">
      <TestComponent />
    </GGIDProvider>
  );
  expect(screen.getByText('Not logged in')).toBeInTheDocument();
});
```

---

## Test Matrix

| Layer | Tool | What to Test |
|-------|------|-------------|
| Unit | Go testing | JWT verify, tenant resolution, role check |
| Integration | Go testing + httptest | API CRUD with mock DB |
| E2E | bats/curl | Full register→login→API flow |
| Frontend | Testing Library | Component render, auth state |
| Load | k6 | 1000 req/s sustained |

---

*See: [Developer Onboarding](../quickstart/developer-onboarding.md) | [5-Minute Quickstart](5-minute-quickstart.md)*

*Last updated: 2025-07-11*
