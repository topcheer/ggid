# SDK Error Handling Guide

Cross-language guide to handling GGID API errors in Go, Node.js, and Java.

---

## Error Types

All three SDKs classify errors into the same categories:

| Error Type | HTTP Status | When It Happens | Retryable? |
|-----------|:-----------:|-----------------|:----------:|
| `BadRequest` | 400 | Invalid input, missing field | No |
| `Unauthorized` | 401 | Missing/expired JWT, invalid credentials | Refresh token |
| `Forbidden` | 403 | Insufficient permissions (RBAC/ABAC denied) | No |
| `NotFound` | 404 | Resource doesn't exist | No |
| `MethodNotAllowed` | 405 | Wrong HTTP method | No |
| `Conflict` | 409 | Duplicate username/email/role key | No |
| `RateLimited` | 429 | Too many requests | Yes (with backoff) |
| `ServerError` | 500 | Internal server error | Yes (with backoff) |
| `BadGateway` | 502 | Backend service down | Yes (with backoff) |
| `ServiceUnavailable` | 503 | Backend unhealthy / circuit open | Yes (with backoff) |
| `Timeout` | — | Request exceeded timeout | Yes (with backoff) |
| `NetworkError` | — | Connection refused, DNS failure | Yes (with backoff) |

---

## Go SDK

### Error Type

```go
// ggid.APIError wraps all API errors
type APIError struct {
    StatusCode int
    Message    string
    Code       string  // GGID error code (e.g. "NOT_FOUND")
}
```

### Detection

```go
user, err := client.GetUser(ctx, userID)
if err != nil {
    var apiErr *ggid.APIError
    if errors.As(err, &apiErr) {
        switch {
        case apiErr.IsNotFound():
            // 404 — handle missing user
        case apiErr.IsUnauthorized():
            // 401 — refresh token, then retry
        case apiErr.IsForbidden():
            // 403 — permission denied
        case apiErr.IsConflict():
            // 409 — already exists
        case apiErr.IsRateLimited():
            // 429 — wait and retry
        case apiErr.IsServerError():
            // 500/502/503 — retry with backoff
        }
    }
    // Non-API errors: network, timeout, DNS
}
```

### Helper Methods

```go
apiErr.IsBadRequest()      // 400
apiErr.IsUnauthorized()    // 401
apiErr.IsForbidden()       // 403
apiErr.IsNotFound()        // 404
apiErr.IsConflict()        // 409
apiErr.IsRateLimited()     // 429
apiErr.IsServerError()     // 500-503
apiErr.IsRetryable()       // 429, 500, 502, 503, timeout
```

---

## Node.js SDK

### Error Detection

```typescript
try {
  const user = await client.getUser(accessToken, userId);
} catch (err: any) {
  const status = err.message.match(/GGID API (\d+)/)?.[1];

  switch (status) {
    case '400': // bad request
    case '401': // unauthorized — refresh token
    case '403': // forbidden
    case '404': // not found
    case '409': // conflict
    case '429': // rate limited — retry
    case '500': // server error — retry
    case '502': // bad gateway — retry
    case '503': // unavailable — retry
  }
}
```

---

## Java SDK

### Error Type

```java
public class GGIDException extends Exception {
    private final int statusCode;
    private final String errorCode;

    public boolean isBadRequest()    { return statusCode == 400; }
    public boolean isUnauthorized()  { return statusCode == 401; }
    public boolean isForbidden()     { return statusCode == 403; }
    public boolean isNotFound()      { return statusCode == 404; }
    public boolean isConflict()      { return statusCode == 409; }
    public boolean isRateLimited()   { return statusCode == 429; }
    public boolean isServerError()   { return statusCode >= 500; }
    public boolean isRetryable()     { return isRateLimited() || isServerError(); }
}
```

### Detection

```java
try {
    client.getUser(userId);
} catch (GGIDException e) {
    if (e.isNotFound()) {
        // handle missing
    } else if (e.isUnauthorized()) {
        // refresh and retry
    } else if (e.isRetryable()) {
        // retry with backoff
    }
    System.out.println(e.getStatusCode() + ": " + e.getMessage());
}
```

---

## Retry Strategy (Exponential Backoff)

Retry only on idempotent operations (GET) or when you have an idempotency key.

### Algorithm

```
attempt 1: wait 1s
attempt 2: wait 2s  + jitter
attempt 3: wait 4s  + jitter
attempt 4: wait 8s  + jitter
attempt 5: wait 16s + jitter (max)
```

### Go

```go
func withRetry(ctx context.Context, op func() error) error {
    maxRetries := 5
    baseDelay := 1 * time.Second

    for attempt := 0; attempt < maxRetries; attempt++ {
        err := op()
        if err == nil { return nil }

        var apiErr *ggid.APIError
        if !errors.As(err, &apiErr) || !apiErr.IsRetryable() {
            return err  // non-retryable
        }

        // Exponential backoff with jitter
        delay := baseDelay * time.Duration(1<<attempt) // 1s, 2s, 4s, 8s, 16s
        jitter := time.Duration(rand.Intn(int(delay / 2)))
        wait := delay + jitter

        // Respect Retry-After header for 429
        if apiErr.IsRateLimited() && apiErr.RetryAfter > 0 {
            wait = time.Duration(apiErr.RetryAfter) * time.Second
        }

        select {
        case <-time.After(wait):
        case <-ctx.Done():
            return ctx.Err()
        }
    }
    return fmt.Errorf("max retries (%d) exceeded", maxRetries)
}
```

### Node.js

```typescript
async function withRetry<T>(fn: () => Promise<T>, maxRetries = 5): Promise<T> {
  for (let attempt = 0; attempt < maxRetries; attempt++) {
    try {
      return await fn();
    } catch (err: any) {
      const status = err.message.match(/GGID API (\d+)/)?.[1];
      const retryable = ['429', '500', '502', '503'];
      if (!status || !retryable.includes(status)) throw err;

      const baseDelay = Math.pow(2, attempt) * 1000; // 1s, 2s, 4s, 8s, 16s
      const jitter = Math.random() * 500;
      await new Promise(r => setTimeout(r, baseDelay + jitter));
    }
  }
  throw new Error(`Max retries (${maxRetries}) exceeded`);
}
```

### Java

```java
public <T> T withRetry(Supplier<T> operation, int maxRetries) {
    long baseDelay = 1000; // 1 second

    for (int attempt = 0; attempt < maxRetries; attempt++) {
        try {
            return operation.get();
        } catch (GGIDException e) {
            if (!e.isRetryable()) throw e;

            long delay = baseDelay * (1L << attempt); // exponential
            long jitter = (long)(Math.random() * (delay / 2));
            try {
                Thread.sleep(delay + jitter);
            } catch (InterruptedException ie) {
                Thread.currentThread().interrupt();
                throw new RuntimeException("interrupted", ie);
            }
        }
    }
    throw new RuntimeException("Max retries exceeded");
}
```

---

## Circuit Breaker Pattern

Stop calling the API after repeated failures, then periodically probe.

### Go (sony/gobreaker)

```go
import "github.com/sony/gobreaker"

cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
    Name:        "ggid-api",
    MaxRequests: 5,                    // max half-open probes
    Interval:    60 * time.Second,     // reset window
    Timeout:     30 * time.Second,     // open → half-open cooldown
    ReadyToTrip: func(counts gobreaker.Counts) bool {
        return counts.ConsecutiveFailures > 5
    },
    OnStateChange: func(name string, from, to gobreaker.State) {
        log.Printf("CB %s: %s → %s", name, from, to)
    },
})

// Usage
result, err := cb.Execute(func() (interface{}, error) {
    return client.GetUser(ctx, userID)
})
```

### Node.js (cockatiel)

```typescript
import { CircuitBreakerPolicy, ConsecutiveBreaker } from 'cockatiel';

const breaker = new CircuitBreakerPolicy(new ConsecutiveBreaker(5), {
  halfOpenAfter: 30_000,
  cooldown: 60_000,
});

// Usage
try {
  const user = await breaker.execute(() => client.getUser(accessToken, userId));
} catch (err) {
  if (err.name === 'BrokenCircuitError') {
    // Circuit is open — fail fast
  }
}
```

---

## Idempotency Key

For POST/PUT requests that might be retried, send a unique key to ensure
the server processes the request only once:

### Go

```go
idempotencyKey := uuid.New().String()
req, _ := http.NewRequest("POST", url, body)
req.Header.Set("Idempotency-Key", idempotencyKey)
```

### Node.js

```typescript
const response = await fetch(url, {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Idempotency-Key': crypto.randomUUID(),
    'Authorization': `Bearer ${token}`,
  },
  body: JSON.stringify(data),
});
```

### Java

```java
Request request = new Request.Builder()
    .url(url)
    .post(body)
    .addHeader("Idempotency-Key", UUID.randomUUID().toString())
    .build();
```

### How It Works

1. Client sends `Idempotency-Key: <uuid>` header on POST/PUT
2. Server stores the response keyed by the idempotency key (Redis, 24h TTL)
3. If the same key is seen again (retry), the stored response is returned without re-executing
4. Prevents duplicate user creation, double charges, etc.

---

## Decision Flowchart

```
Error received
      │
      ├─► 400 BadRequest? ──► Fix request, do NOT retry
      ├─► 401 Unauthorized? ──► Refresh token, retry once
      ├─► 403 Forbidden? ──► Check permissions, do NOT retry
      ├─► 404 NotFound? ──► Handle gracefully, do NOT retry
      ├─► 409 Conflict? ──► Resource exists, do NOT retry
      │
      ├─► 429 RateLimited? ──► Wait Retry-After, retry
      ├─► 500/502/503? ──► Exponential backoff, retry up to 5
      ├─► Timeout? ──► Exponential backoff, retry up to 3
      └─► NetworkError? ──► Exponential backoff, retry up to 3
```
