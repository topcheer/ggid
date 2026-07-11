# Tutorial: Webhook Integration

> Complete tutorial: register a webhook, receive events, verify signatures, implement retry-safe processing with a sample Go server.

---

## Prerequisites

- GGID running (`docker compose up -d`)
- A JWT token with webhook permissions
- Go 1.25+ installed

---

## Step 1: Register a Webhook

```bash
export TOKEN="your-jwt-token"
export TENANT_ID="00000000-0000-0000-0000-000000000001"

curl -X POST http://localhost:8080/api/v1/webhooks \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "http://localhost:9090/webhook",
    "events": ["user.created", "user.deleted", "auth.login", "auth.login_failed"],
    "description": "My integration"
  }' | jq .
```

Save the `secret` from the response — you need it for signature verification.

```bash
export WEBHOOK_SECRET="whsec_abc123def456"
```

---

## Step 2: Build the Receiver (Go)

```go
// main.go
package main

import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "io"
    "log"
    "net/http"
    "os"
    "sync"
)

type WebhookEvent struct {
    EventID   string          `json:"event_id"`
    EventType string          `json:"event_type"`
    Timestamp string          `json:"timestamp"`
    TenantID  string          `json:"tenant_id"`
    Data      json.RawMessage `json:"data"`
}

var processed sync.Map // event_id -> bool (for idempotency)

func verifySignature(payload []byte, signature, secret string) bool {
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(payload)
    expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
    return hmac.Equal([]byte(expected), []byte(signature))
}

func handler(w http.ResponseWriter, r *http.Request) {
    // 1. Read raw body
    body, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "bad body", http.StatusBadRequest)
        return
    }

    // 2. Verify HMAC signature
    signature := r.Header.Get("X-GGID-Signature")
    secret := os.Getenv("WEBHOOK_SECRET")
    if !verifySignature(body, signature, secret) {
        log.Println("Signature verification failed")
        http.Error(w, "invalid signature", http.StatusUnauthorized)
        return
    }

    // 3. Parse event
    var event WebhookEvent
    if err := json.Unmarshal(body, &event); err != nil {
        http.Error(w, "bad json", http.StatusBadRequest)
        return
    }

    // 4. Idempotency check (dedup)
    if _, ok := processed.Load(event.EventID); ok {
        log.Printf("Duplicate event %s, skipping", event.EventID)
        w.WriteHeader(http.StatusOK) // Ack so GGID stops retrying
        return
    }
    processed.Store(event.EventID, true)

    // 5. Process event
    switch event.EventType {
    case "user.created":
        log.Printf("New user created: %s", event.Data)
        // ... provision in your system
    case "user.deleted":
        log.Printf("User deleted: %s", event.Data)
        // ... deprovision
    case "auth.login_failed":
        log.Printf("Login failed: %s", event.Data)
        // ... alert security team
    default:
        log.Printf("Unhandled event: %s", event.EventType)
    }

    // 6. Respond 200 so GGID stops retrying
    w.WriteHeader(http.StatusOK)
    fmt.Fprintf(w, `{"received":true}`)
}

func main() {
    http.HandleFunc("/webhook", handler)
    log.Println("Webhook receiver listening on :9090")
    log.Fatal(http.ListenAndServe(":9090", nil))
}
```

### Run It

```bash
export WEBHOOK_SECRET="whsec_abc123def456"
go run main.go
```

---

## Step 3: Test the Webhook

```bash
# Send a test event from GGID
curl -X POST http://localhost:8080/api/v1/webhooks/{webhook_id}/test \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"

# Check delivery status
curl http://localhost:8080/api/v1/webhooks/{webhook_id}/deliveries \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" | jq .
```

---

## Event Payload Format

```json
{
  "event_id": "evt_unique_id_123",
  "event_type": "user.created",
  "timestamp": "2025-07-11T12:00:00.123Z",
  "tenant_id": "00000000-0000-0000-0000-000000000001",
  "data": {
    "user_id": "usr_abc123",
    "email": "john@example.com",
    "username": "johndoe"
  },
  "metadata": {
    "delivery_attempt": 1,
    "source": "auth-service"
  }
}
```

### HTTP Headers

| Header | Purpose |
|--------|---------|
| `X-GGID-Signature` | `sha256=<hex>` HMAC |
| `X-GGID-Event` | Event type (e.g., `user.created`) |
| `X-GGID-Event-ID` | Unique event ID for idempotency |

---

## Retry Behavior

GGID retries with exponential backoff. Always respond **200** once you've safely persisted the event.

| Attempt | Delay |
|---------|-------|
| 1 | Immediate |
| 2 | 30 seconds |
| 3 | 2 minutes |
| 4 | 10 minutes |
| 5 | 1 hour |
| 6+ | Up to 24 hours |

After 8 failed attempts, events go to a dead letter queue.

---

## Idempotency Pattern

GGID uses **at-least-once delivery** — the same event may arrive more than once. Always deduplicate using `event_id`:

```go
// In-memory (for production use Redis/DB)
var processed sync.Map

if _, ok := processed.Load(event.EventID); ok {
    w.WriteHeader(http.StatusOK)
    return
}
processed.Store(event.EventID, true)
```

For production, use a database table:
```sql
CREATE TABLE processed_events (
    event_id VARCHAR(255) PRIMARY KEY,
    processed_at TIMESTAMP DEFAULT NOW()
);
```

---

## Summary

1. Register webhook via API (save the secret)
2. Build HTTP server that verifies HMAC-SHA256 signatures
3. Use `event_id` for idempotent processing
4. Respond 200 to stop retries
5. Handle duplicates gracefully

See: [Webhook Guide](../webhook-guide.md) for full event catalog and HMAC examples in Python/Node.js.

---

*Last updated: 2025-07-11*