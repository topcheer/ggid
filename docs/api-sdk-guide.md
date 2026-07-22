# GGID SDK Guide

Complete guide for integrating GGID using the official SDKs in Go, Node.js, and Java.

---

## Table of Contents

- [Installation](#installation)
- [Initialization](#initialization)
- [Authentication Flows](#authentication-flows)
- [Token Refresh](#token-refresh)
- [Error Handling](#error-handling)
- [Pagination](#pagination)
- [Go SDK](#go-sdk)
- [Node.js SDK](#nodejs-sdk)
- [Java SDK](#java-sdk)

---

## Installation

### Go

```bash
go get github.com/ggid/sdk-go@latest
```

### Node.js

```bash
npm install @ggid/sdk
# or
yarn add @ggid/sdk
# or
pnpm add @ggid/sdk
```

### Java (Maven)

```xml
<dependency>
    <groupId>dev.ggid</groupId>
    <artifactId>ggid-sdk</artifactId>
    <version>1.0.0</version>
</dependency>
```

### Java (Gradle)

```groovy
implementation 'dev.ggid:ggid-sdk:1.0.0'
```

---

## Initialization

All SDKs share the same configuration pattern:

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| `baseURL` | Yes | — | GGID Gateway URL (e.g., `https://iam.example.com`) |
| `tenantID` | Yes | — | Tenant UUID |
| `apiKey` | No | — | API key for server-to-server calls |
| `timeout` | No | 30s | HTTP client timeout |

### Go

```go
import "github.com/ggid/sdk-go/client"

client := client.New(
    client.WithBaseURL("https://iam.example.com"),
    client.WithTenantID("28d6fe98-adeb-4c0c-b49b-20c6695bbca6"),
    client.WithAPIKey("ggid-key-..."), // for service accounts
    client.WithTimeout(30*time.Second),
)
```

### Node.js

```typescript
import { GGIDClient } from '@ggid/sdk';

const client = new GGIDClient({
    baseURL: 'https://iam.example.com',
    tenantID: '28d6fe98-adeb-4c0c-b49b-20c6695bbca6',
    apiKey: process.env.GGID_API_KEY, // for server-side
    timeout: 30000,
});
```

### Java

```java
import dev.ggid.sdk.GGIDClient;

GGIDClient client = GGIDClient.builder()
    .baseURL("https://iam.example.com")
    .tenantID("28d6fe98-adeb-4c0c-b49b-20c6695bbca6")
    .apiKey(System.getenv("GGID_API_KEY"))
    .timeout(Duration.ofSeconds(30))
    .build();
```

---

## Authentication Flows

### Password Login

#### Go

```go
resp, err := client.Auth.Login(ctx, &client.LoginRequest{
    Username: "john.doe",
    Password: "SecurePass123!",
})
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Access token: %s\n", resp.AccessToken)
fmt.Printf("Expires in: %d seconds\n", resp.ExpiresIn)
```

#### Node.js

```typescript
const result = await client.auth.login({
    username: 'john.doe',
    password: 'SecurePass123!',
});
console.log('Access token:', result.access_token);
console.log('Expires in:', result.expires_in);
```

#### Java

```java
LoginResponse resp = client.auth().login(LoginRequest.builder()
    .username("john.doe")
    .password("SecurePass123!")
    .build());
System.out.println("Access token: " + resp.getAccessToken());
```

### Social Login (OAuth)

```go
// Step 1: Get authorization URL
authURL, err := client.Auth.GetSocialAuthURL(ctx, "google", "https://app.example.com/callback")
// Step 2: Redirect user to authURL
// Step 3: Exchange callback code for tokens
tokens, err := client.Auth.HandleSocialCallback(ctx, "google", code, state)
```

### WebAuthn (Passkey) Login

```typescript
// Step 1: Begin authentication
const options = await client.auth.webauthn.beginLogin({ username: 'john.doe@example.com' });

// Step 2: Browser calls navigator.credentials.get()
const assertion = await navigator.credentials.get({ publicKey: decodeOptions(options) });

// Step 3: Finish authentication
const tokens = await client.auth.webauthn.finishLogin(encodeAssertion(assertion));
```

### Client Credentials (Service Account)

```go
// Machine-to-machine authentication (no user involved)
resp, err := client.Auth.ClientCredentials(ctx, &client.ClientCredentialsRequest{
    ClientID:     "my-service",
    ClientSecret: os.Getenv("CLIENT_SECRET"),
    Scope:        "users:read users:write",
})
```

---

## Token Refresh

All SDKs automatically refresh expired access tokens when a refresh token is available.

### Go (Automatic)

```go
// The Go SDK automatically refreshes tokens when they expire.
// Just keep using the client — it handles refresh internally.
resp, err := client.Users.List(ctx, nil)
// SDK checks token expiry → refreshes if needed → retries request
```

### Go (Manual)

```go
// Manual refresh
tokens, err := client.Auth.Refresh(ctx, resp.RefreshToken)
if err != nil {
    // Refresh failed — user must re-authenticate
    log.Fatal(err)
}
client.SetAccessToken(tokens.AccessToken)
```

### Node.js (Automatic with Interceptor)

```typescript
// The SDK automatically refreshes using Axios interceptors
const client = new GGIDClient({
    baseURL: 'https://iam.example.com',
    tenantID: process.env.GGID_TENANT_ID,
    refreshToken: storedRefreshToken, // from previous login
    onTokenRefresh: (tokens) => {
        // Called automatically when tokens are refreshed
        // Persist new tokens
        localStorage.setItem('access_token', tokens.access_token);
        localStorage.setItem('refresh_token', tokens.refresh_token);
    },
});

// All requests use the latest token automatically
const users = await client.users.list();
```

### Java (Automatic)

```java
// Token manager handles refresh automatically
GGIDClient client = GGIDClient.builder()
    .baseURL("https://iam.example.com")
    .tokenManager(new AutoRefreshTokenManager(refreshToken))
    .build();

// All calls use valid tokens automatically
UserList users = client.users().list();
```

---

## Error Handling

All SDKs provide typed errors for different failure modes:

| Error Type | HTTP Status | Cause |
|-----------|-------------|-------|
| `AuthenticationError` | 401 | Invalid credentials or expired token |
| `AuthorizationError` | 403 | Insufficient permissions |
| `NotFoundError` | 404 | Resource doesn't exist |
| `ValidationError` | 400 | Invalid input |
| `ConflictError` | 409 | Duplicate resource |
| `RateLimitError` | 429 | Rate limit exceeded |
| `ServerError` | 500-599 | Internal server error |

### Go

```go
users, err := client.Users.List(ctx, &client.ListOptions{Limit: 10})
if err != nil {
    var authErr *client.AuthenticationError
    if errors.As(err, &authErr) {
        // Token expired — redirect to login
        log.Println("Authentication required")
        return
    }

    var rateErr *client.RateLimitError
    if errors.As(err, &rateErr) {
        // Wait and retry
        time.Sleep(rateErr.RetryAfter)
        return
    }

    var notFoundErr *client.NotFoundError
    if errors.As(err, &notFoundErr) {
        log.Printf("Resource not found: %s", notFoundErr.Resource)
        return
    }

    // Generic error
    log.Printf("Unexpected error: %v", err)
}
```

### Node.js

```typescript
try {
    const users = await client.users.list({ limit: 10 });
} catch (err) {
    if (err instanceof AuthenticationError) {
        // Redirect to login
        window.location.href = '/login';
    } else if (err instanceof RateLimitError) {
        // Wait and retry
        await sleep(err.retryAfter);
        return retry();
    } else if (err instanceof NotFoundError) {
        console.log('Resource not found:', err.resource);
    } else if (err instanceof ValidationError) {
        // Show validation errors to user
        err.fields.forEach(f => {
            console.error(`${f.field}: ${f.message}`);
        });
    } else {
        console.error('Unexpected error:', err.message);
    }
}
```

### Java

```java
try {
    UserList users = client.users().list(ListParams.builder().limit(10).build());
} catch (GGIDException e) {
    if (e instanceof AuthenticationException) {
        // Redirect to login
    } else if (e instanceof RateLimitException) {
        Thread.sleep(((RateLimitException) e).getRetryAfterMs());
    } else if (e instanceof NotFoundException) {
        log.warn("Resource not found: {}", e.getResource());
    } else if (e instanceof ValidationException) {
        ((ValidationException) e).getFields().forEach(f ->
            log.error("{}: {}", f.getField(), f.getMessage()));
    } else {
        log.error("Unexpected error", e);
    }
}
```

---

## Pagination

GGID uses cursor-based pagination for all list endpoints.

### Request Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `limit` | int | 20 | Items per page (max 100) |
| `cursor` | string | — | Cursor from previous response |
| `sort` | string | `created_at` | Sort field |
| `order` | string | `desc` | `asc` or `desc` |

### Response Fields

| Field | Type | Description |
|-------|------|-------------|
| `data` | array | Current page items |
| `cursor` | string \| null | Cursor for next page (null if no more) |
| `has_more` | bool | Whether more pages exist |
| `total` | int | Total item count (optional) |

### Go

```go
// Paginate through all users
var allUsers []client.User
cursor := ""

for {
    resp, err := client.Users.List(ctx, &client.ListOptions{
        Limit:  50,
        Cursor: cursor,
    })
    if err != nil {
        log.Fatal(err)
    }

    allUsers = append(allUsers, resp.Data...)

    if !resp.HasMore {
        break
    }
    cursor = resp.Cursor
}

fmt.Printf("Total users: %d\n", len(allUsers))
```

### Node.js

```typescript
// Async iterator for automatic pagination
for await (const user of client.users.listAll({ limit: 50 })) {
    console.log(user.email);
}

// Or manual pagination
let cursor: string | undefined;
do {
    const page = await client.users.list({ limit: 50, cursor });
    page.data.forEach(u => console.log(u.email));
    cursor = page.cursor;
} while (cursor);
```

### Java

```java
// Iterator-based pagination
Iterator<User> iterator = client.users().listIterator(ListParams.builder()
    .limit(50)
    .build());

while (iterator.hasNext()) {
    User user = iterator.next();
    System.out.println(user.getEmail());
}
```

---

## Go SDK

### User Management

```go
// Create user
user, err := client.Users.Create(ctx, &client.CreateUserRequest{
    Username: "jane.doe",
    Email:    "jane@example.com",
    Password: "TempPass123!",
    Name:     "Jane Doe",
})

// Get user by ID
user, err := client.Users.Get(ctx, userID)

// Update user
user.Email = "jane.new@example.com"
updated, err := client.Users.Update(ctx, user)

// Delete user
err := client.Users.Delete(ctx, userID)

// Assign role
err := client.Users.AssignRole(ctx, userID, roleID)

// List users with filter
resp, err := client.Users.List(ctx, &client.ListOptions{
    Limit: 20,
    Filter: "status eq 'active'",
    Sort:   "email",
    Order:  "asc",
})
```

### Policy Check

```go
allowed, err := client.Policy.Check(ctx, &client.PolicyCheckRequest{
    Subject:  userID.String(),
    Resource: "document:123",
    Action:   "read",
})
if allowed {
    // Grant access
}
```

---

## Node.js SDK

### User Management

```typescript
// Create user
const user = await client.users.create({
    username: 'jane.doe',
    email: 'jane@example.com',
    password: 'TempPass123!',
    name: 'Jane Doe',
});

// Batch create
const result = await client.users.batchCreate([
    { username: 'user1', email: 'u1@example.com', password: 'Pass123!' },
    { username: 'user2', email: 'u2@example.com', password: 'Pass123!' },
]);

// Search users
const results = await client.users.search({
    filter: 'emails.value eq "jane@example.com"',
});
```

### Audit Events

```typescript
// Query audit events
const events = await client.audit.list({
    startTime: '2024-01-01T00:00:00Z',
    endTime: '2024-01-31T23:59:59Z',
    eventType: 'auth.login',
    limit: 50,
});

// Subscribe to live audit stream (SSE)
const stream = client.audit.stream({
    eventType: 'user.*',
});

for await (const event of stream) {
    console.log(`[${event.event_type}] ${event.timestamp} user=${event.data.user_id}`);
}
```

---

## Java SDK

### User Management

```java
// Create user
User user = client.users().create(CreateUserRequest.builder()
    .username("jane.doe")
    .email("jane@example.com")
    .password("TempPass123!")
    .name("Jane Doe")
    .build());

// Assign role
client.users().assignRole(user.getId(), roleID.toString());

// Check permissions
boolean allowed = client.policy().check(PolicyCheckRequest.builder()
    .subject(user.getId())
    .resource("document:123")
    .action("read")
    .build());
```

### OAuth Integration (Spring Boot)

```java
@Bean
public GGIDClient ggidClient() {
    return GGIDClient.builder()
        .baseURL(env.getProperty("ggid.url"))
        .tenantID(env.getProperty("ggid.tenant-id"))
        .apiKey(env.getProperty("ggid.api-key"))
        .build();
}

// Use in a controller
@GetMapping("/api/users/{id}")
public ResponseEntity<User> getUser(@PathVariable String id) {
    try {
        User user = ggidClient.users().get(UUID.fromString(id));
        return ResponseEntity.ok(user);
    } catch (NotFoundException e) {
        return ResponseEntity.notFound().build();
    }
}
```

---

## Configuration Reference

### Environment Variables

| Variable | SDK | Default | Description |
|----------|-----|---------|-------------|
| `GGID_BASE_URL` | All | — | Gateway URL |
| `GGID_TENANT_ID` | All | — | Tenant UUID |
| `GGID_API_KEY` | All | — | API key for service auth |
| `GGID_TIMEOUT` | Go/Java | 30s | Request timeout |
| `GGID_DEBUG` | All | false | Enable debug logging |
| `GGID_RETRY_COUNT` | Go/Java | 3 | Retry attempts for 5xx |
| `GGID_RETRY_BASE_DELAY` | Node.js | 1s | Base delay for exponential backoff |

### Retry Configuration

All SDKs retry on 5xx errors with exponential backoff:

```go
// Go
client := client.New(
    client.WithBaseURL("https://iam.example.com"),
    client.WithTenantID(tenantID),
    client.WithRetry(3, 1*time.Second), // 3 retries, 1s base delay
)
```

```typescript
// Node.js
const client = new GGIDClient({
    baseURL: 'https://iam.example.com',
    tenantID,
    retry: { count: 3, baseDelay: 1000, maxDelay: 10000 },
});
```

---

## References

- [SDK Cookbook](./sdk-cookbook.md) — Real-world integration recipes
- [API Reference](./api-reference.md) — REST endpoint documentation
- [Integration Examples](./integration-examples.md) — Full code samples
- [Error Codes](./error-codes.md) — Complete error reference
