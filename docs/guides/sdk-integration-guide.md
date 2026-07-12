# SDK Integration Guide

Setup and usage for Go, Node, Java, and React SDKs — token verification, permission checking, error handling, and best practices.

## Installation

### Go

```bash
go get github.com/ggid-dev/ggid-go@latest
```

### Node

```bash
npm install @ggid/node-sdk
# or
yarn add @ggid/node-sdk
```

### Java

```xml
<!-- Maven -->
<dependency>
  <groupId>dev.ggid</groupId>
  <artifactId>ggid-java-sdk</artifactId>
  <version>latest</version>
</dependency>
```

### React

```bash
npm install @ggid/react-sdk
```

## Initialization

### Go

```go
import "github.com/ggid-dev/ggid-go"

client, err := ggid.NewClient(ggid.Config{
    GatewayURL:  "https://gateway.ggid.dev",
    ClientID:    "your-client-id",
    ClientSecret: "your-client-secret",
    TenantID:    "00000000-0000-0000-0000-000000000001",
})
if err != nil {
    log.Fatal(err)
}
```

### Node

```javascript
const { GGIDClient } = require('@ggid/node-sdk');

const client = new GGIDClient({
  gatewayURL: 'https://gateway.ggid.dev',
  clientID: 'your-client-id',
  clientSecret: 'your-client-secret',
  tenantID: '00000000-0000-0000-0000-000000000001',
});
```

### Java

```java
import dev.ggid.sdk.GGIDClient;
import dev.ggid.sdk.GGIDConfig;

GGIDClient client = GGIDClient.builder()
    .gatewayURL("https://gateway.ggid.dev")
    .clientID("your-client-id")
    .clientSecret("your-client-secret")
    .tenantID("00000000-0000-0000-0000-000000000001")
    .build();
```

### React (Provider)

```tsx
import { GGIDProvider } from '@ggid/react-sdk';

function App() {
  return (
    <GGIDProvider
      gatewayURL="https://gateway.ggid.dev"
      clientID="your-client-id"
      tenantID="00000000-0000-0000-0000-000000000001"
    >
      <YourApp />
    </GGIDProvider>
  );
}
```

## User Management

### Go

```go
// Create user
user, err := client.Users.Create(ctx, &ggid.CreateUserRequest{
    Email:       "jane@corp.com",
    DisplayName: "Jane Doe",
    Password:    "secure-password",
})
if err != nil {
    handleErr(err)
}
fmt.Printf("Created: %s\n", user.ID)

// List users with pagination
users, err := client.Users.List(ctx, &ggid.ListOptions{
    Page: 1, PerPage: 50,
    Filter: ggid.Filter{Department: "engineering"},
})

// Update user
_, err = client.Users.Update(ctx, user.ID, &ggid.UpdateUserRequest{
    DisplayName: "Jane Smith",
})
```

### Node

```javascript
// Create user
const user = await client.users.create({
  email: 'jane@corp.com',
  displayName: 'Jane Doe',
  password: 'secure-password',
});

// List with pagination
const { users, total } = await client.users.list({
  page: 1, perPage: 50,
  filter: { department: 'engineering' },
});
```

### React (Hooks)

```tsx
import { useUsers, useCreateUser } from '@ggid/react-sdk';

function UserList() {
  const { users, loading, error, mutate } = useUsers({ perPage: 50 });
  const { createUser } = useCreateUser();

  if (loading) return <Spinner />;
  if (error) return <ErrorView error={error} />;

  return (
    <ul>
      {users.map(u => <li key={u.id}>{u.displayName}</li>)}
    </ul>
  );
}
```

## Authentication

### Login

```go
// Go
auth, err := client.Auth.Login(ctx, &ggid.LoginRequest{
    Username: "jane@corp.com",
    Password: "secure-password",
})
// auth.AccessToken, auth.RefreshToken
```

```javascript
// Node
const auth = await client.auth.login({
  username: 'jane@corp.com',
  password: 'secure-password',
});
```

### Token Verification

```go
// Verify JWT from incoming request
claims, err := client.Auth.VerifyToken(ctx, accessToken)
if err != nil {
    http.Error(w, "invalid token", 401)
    return
}
fmt.Printf("User: %s, Scopes: %s\n", claims.Subject, claims.Scope)
```

```javascript
// Express middleware
const { verifyToken } = client.auth;

app.get('/protected', async (req, res) => {
  try {
    const claims = await verifyToken(req.headers.authorization);
    req.user = claims;
    next();
  } catch (err) {
    res.status(401).json({ error: 'invalid token' });
  }
});
```

### Token Refresh

```go
auth, err := client.Auth.Refresh(ctx, refreshToken)
// New access + refresh tokens (old refresh invalidated)
```

## Permission Checking

### Go

```go
// Check if user has permission
allowed, err := client.Policy.Check(ctx, &ggid.CheckRequest{
    UserID:   user.ID,
    Resource: "users",
    Action:   "delete",
})
if !allowed {
    http.Error(w, "forbidden", 403)
}
```

### Node

```javascript
const allowed = await client.policy.check({
  userID: user.id,
  resource: 'users',
  action: 'delete',
});
if (!allowed) {
  return res.status(403).json({ error: 'forbidden' });
}
```

### React

```tsx
import { usePermission } from '@ggid/react-sdk';

function DeleteButton({ user }) {
  const { allowed } = usePermission({ resource: 'users', action: 'delete' });
  
  if (!allowed) return null;
  return <button onClick={() => deleteUser(user.id)}>Delete</button>;
}
```

## OAuth Operations

```go
// Build authorize URL
authURL := client.OAuth.AuthorizeURL(&ggid.AuthorizeRequest{
    RedirectURI:  "https://app.example.com/callback",
    Scope:        "openid profile email",
    State:        "random-state",
    CodeChallenge: pkceChallenge,  // PKCE
})

// Exchange code for tokens
tokens, err := client.OAuth.ExchangeCode(ctx, code, codeVerifier)
```

## Error Handling

### Go

```go
user, err := client.Users.Get(ctx, "uuid")
if err != nil {
    var apiErr *ggid.APIError
    if errors.As(err, &apiErr) {
        switch apiErr.StatusCode {
        case 404:
            fmt.Println("user not found")
        case 401:
            fmt.Println("token expired — refresh needed")
        case 403:
            fmt.Println("insufficient scope")
        case 429:
            fmt.Printf("rate limited, retry after %s\n", apiErr.RetryAfter)
        default:
            fmt.Printf("server error: %s\n", apiErr.Message)
        }
    }
}
```

### Node

```javascript
try {
  const user = await client.users.get('uuid');
} catch (err) {
  if (err.status === 404) console.log('not found');
  if (err.status === 429) console.log(`retry after ${err.retryAfter}s`);
  if (err.code === 'TOKEN_EXPIRED') await refreshAndRetry();
}
```

## Best Practices

### 1. Token Management

```go
// Cache tokens — don't fetch per request
tokenCache := cache.New(14 * time.Minute) // Access tokens expire in 15 min

func getToken(ctx context.Context) (string, error) {
    if t := tokenCache.Get("access"); t != nil { return t.(string), nil }
    auth, _ := client.Auth.ClientCredentials(ctx)
    tokenCache.Set("access", auth.AccessToken, 14*time.Minute)
    return auth.AccessToken, nil
}
```

### 2. Automatic Retry (with backoff)

```go
client.SetRetryPolicy(ggid.RetryPolicy{
    MaxRetries:  3,
    BaseDelay:   time.Second,
    MaxDelay:    10 * time.Second,
    RetryOn:     []int{502, 503, 504},
})
```

### 3. Context with Timeout

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
user, err := client.Users.Get(ctx, "uuid") // Cancels after 5s
```

### 4. React — SWR Cache

```tsx
// useUsers hook auto-caches via SWR
const { users, mutate } = useUsers();
// mutate() revalidates on focus, interval, manual
```

## See Also

- [Gateway Architecture](gateway-architecture.md)
- [OAuth Client Lifecycle](oauth-client-lifecycle.md)
- [Policy Engine Internals](policy-engine-internals.md)
- [OAuth Client Scoped Permissions](oauth-client-scoped-permissions.md)
