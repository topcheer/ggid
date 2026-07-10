# GGID SDK Quickstart

Copy-paste examples for Go, Node.js, and Java SDKs. Covers initialization,
registration, login, MFA, OAuth flows, and token refresh.

---

## Go SDK

### Install

```bash
go get github.com/ggid/sdk-go@latest
```

### Init

```go
package main

import ggid "github.com/ggid/sdk-go"

client := ggid.NewClient(ggid.Config{
    GatewayURL: "https://iam.example.com",
    TenantID:   "00000000-0000-0000-0000-000000000001",
})
```

### Register

```go
user, err := client.Auth.Register(ctx, ggid.RegisterRequest{
    Username: "alice",
    Email:    "alice@example.com",
    Password: "SecurePass123!",
})
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Registered: %s\n", user.ID)
```

### Login

```go
tokens, err := client.Auth.Login(ctx, ggid.LoginRequest{
    Username: "alice",
    Password: "SecurePass123!",
})
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Access token: %s\n", tokens.AccessToken[:30]+"...")
```

### Token Refresh

```go
// Automatic — SDK refreshes when access token expires
client.SetRefreshToken(tokens.RefreshToken)

// Manual
newTokens, err := client.Auth.Refresh(ctx, tokens.RefreshToken)
```

### MFA (TOTP)

```go
// Enable MFA
challenge, err := client.MFA.EnableTOTP(ctx, tokens.AccessToken)
// -> Returns QR code URL + secret

// Verify TOTP code
err = client.MFA.VerifyTOTP(ctx, tokens.AccessToken, "123456")

// Login with MFA
tokens, err := client.Auth.LoginWithMFA(ctx, ggid.MFARequest{
    Username:    "alice",
    Password:    "SecurePass123!",
    MFACode:     "123456",
    MFAMethod:   "totp",
})
```

### RBAC Check

```go
allowed, err := client.Policy.Check(ctx, ggid.PolicyCheck{
    UserID:   user.ID,
    Action:   "users:read",
    Resource: "users:*",
})
if !allowed {
    fmt.Println("Access denied")
}
```

---

## Node.js SDK

### Install

```bash
npm install @ggid/sdk
```

### Init

```typescript
import { GGID } from '@ggid/sdk';

const ggid = new GGID({
    gatewayURL: 'https://iam.example.com',
    tenantID: '00000000-0000-0000-0000-000000000001',
});
```

### Register + Login

```typescript
// Register
const user = await ggid.auth.register({
    username: 'bob',
    email: 'bob@example.com',
    password: 'SecurePass123!',
});

// Login
const { access_token, refresh_token } = await ggid.auth.login({
    username: 'bob',
    password: 'SecurePass123!',
});

// Set token for subsequent calls
ggid.setToken(access_token);
```

### OAuth Social Login

```typescript
// Get authorization URL
const authURL = await ggid.oauth.getAuthURL({
    client_id: 'my-app',
    redirect_uri: 'https://app.example.com/callback',
    provider: 'google',
    scope: 'openid email profile',
    state: 'random-state-value',
});

// Redirect user to authURL, then exchange code
const tokens = await ggid.oauth.exchangeCode({
    code: 'auth_code_from_callback',
    redirect_uri: 'https://app.example.com/callback',
    client_id: 'my-app',
    client_secret: process.env.OAUTH_SECRET,
});
```

### Auto Token Refresh

```typescript
// SDK auto-refreshes when access token expires
ggid.setToken(access_token, refresh_token);

// Listen for token rotation
ggid.on('tokenRefreshed', ({ access_token, refresh_token }) => {
    // Persist new tokens
    saveTokens(access_token, refresh_token);
});
```

---

## Java SDK

### Install (Maven)

```xml
<dependency>
    <groupId>dev.ggid</groupId>
    <artifactId>ggid-sdk</artifactId>
    <version>1.0.0</version>
</dependency>
```

### Init

```java
import dev.ggid.sdk.GGIDClient;
import dev.ggid.sdk.Config;

GGIDClient client = GGIDClient.builder()
    .gatewayURL("https://iam.example.com")
    .tenantID("00000000-0000-0000-0000-000000000001")
    .build();
```

### Login + MFA

```java
// Login
TokenResponse tokens = client.auth().login(LoginRequest.builder()
    .username("charlie")
    .password("SecurePass123!")
    .build());

// If MFA required
if (tokens.requiresMFA()) {
    tokens = client.auth().verifyMFA(MFARequest.builder()
        .mfaToken(tokens.getMfaToken())
        .code("123456")
        .method("totp")
        .build());
}

String jwt = tokens.getAccessToken();
```

### List Users (with pagination)

```java
// First page
Page<User> page = client.users().list(PageRequest.of(1, 20));
for (User u : page.getData()) {
    System.out.println(u.getEmail());
}

// Next page
if (page.hasNext()) {
    page = client.users().list(page.nextPageRequest());
}
```

---

## Cross-Platform Patterns

### Token Storage

| Platform | Recommended Storage |
|----------|-------------------|
| Go (server) | Environment variable / Vault |
| Node.js (server) | Environment variable / Vault |
| Java (server) | Environment variable / Vault |
| Next.js (browser) | HttpOnly cookie (set by middleware) |
| SPA (browser) | Memory only (never localStorage) |

### Error Handling

```go
// Go
user, err := client.Auth.Login(ctx, req)
if errors.Is(err, ggid.ErrInvalidCredentials) {
    // Wrong password
} else if errors.Is(err, ggid.ErrRateLimited) {
    // 429 — wait before retry
    time.Sleep(60 * time.Second)
} else if err != nil {
    log.Fatal(err)
}
```

```typescript
// Node.js
try {
    const tokens = await ggid.auth.login(req);
} catch (err) {
    if (err.code === 'AUTH_INVALID_CREDENTIALS') {
        // Wrong password
    } else if (err.code === 'RATE_LIMITED') {
        await sleep(err.retryAfter * 1000);
    }
}
```

---

## References

- [SDK Guide](./sdk-guide.md) — Full SDK documentation
- [API Reference](./api-reference.md) — REST endpoints
- [Error Codes](./api-error-codes.md) — Error code reference
