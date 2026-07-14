# Webhook Delivery Guarantees

Guide for reliable webhook delivery: retry, signatures, idempotency, and dead letter queues.

## Delivery Semantics

GGID guarantees **at-least-once delivery**. Consumers must handle duplicates via idempotency.

```
Event published
    │
    ▼
┌──────────────┐     ┌─────────┐
│ NATS          │────▶│ Webhook │────▶ Consumer
│ JetStream     │     │ Worker  │      (HTTP POST)
└──────────────┘     └─────────┘
    │                     │
    │ persist             │ retry on failure
    │ (7 day retention)   │ (exponential backoff)
```

## Retry Strategy

| Attempt | Delay | Total Elapsed |
|---------|-------|---------------|
| 1 | Immediate | 0s |
| 2 | 30s | 30s |
| 3 | 2min | 2.5min |
| 4 | 10min | 12.5min |
| 5 | 1h | 1h12m |
| 6 | 6h | 7h12m |
| 7 | 24h | 31h12m |
| 8 | Final DLQ | 31h12m |

After 8 failed attempts, the event is moved to the dead letter queue and an alert is triggered.

```go
func backoff(attempt int) time.Duration {
    delays := []time.Duration{
        0, 30 * time.Second, 2 * time.Minute, 10 * time.Minute,
        time.Hour, 6 * time.Hour, 24 * time.Hour,
    }
    if attempt >= len(delays) {
        return 0 // Move to DLQ
    }
    // Add jitter (±20%) to avoid thundering herd
    jitter := time.Duration(rand.Int63n(int64(delays[attempt]) / 5))
    return delays[attempt] + jitter - jitter/2
}
```

## Signature Verification

Every webhook includes an HMAC-SHA256 signature:

```http
POST /webhooks/ggid HTTP/1.1
Content-Type: application/json
X-GGID-Event: user.created
X-GGID-Delivery: dpl-abc123
X-GGID-Signature: t=1700000000,v1=8b1a9953c461...
X-GGID-Event-Id: evt-xyz789

{ "event_id": "evt-xyz789", ... }
```

### Verification (Consumer Side)

```go
func verifySignature(payload []byte, sigHeader, secret string) error {
    parts := parseSigHeader(sigHeader) // {t: timestamp, v1: hash}

    // Check freshness (prevent replay)
    if time.Since(parts.Timestamp) > 5*time.Minute {
        return ErrStaleSignature
    }

    // Recompute HMAC
    signedPayload := fmt.Sprintf("%d.%s", parts.Timestamp.Unix(), payload)
    expected := hmac.New(sha256.New, []byte(secret))
    expected.Write([]byte(signedPayload))

    if !hmac.Equal([]byte(parts.V1), expected.Sum(nil)) {
        return ErrInvalidSignature
    }
    return nil
}
```

## Idempotency

Consumers MUST deduplicate using `event_id`:

```go
func handleWebhook(event Event) error {
    // Idempotency check
    if seen := redis.SetNX(ctx, "event:"+event.ID, 1, 24*time.Hour); !seen {
        return nil // Already processed, skip
    }

    processEvent(event)
    return nil
}
```

Even if at-least-once delivery sends the same event twice, the consumer processes it once.

## Ordering

GGID provides **per-entity ordering** but NOT global ordering:

- Events for the same `user_id` arrive in order
- Events for different users may interleave

```go
// NATS partitioning by entity key ensures per-key ordering
js.Publish("events.user", payload, nats.MsgId(event.ID))
// Key = user_id → same consumer → ordered delivery
```

For global ordering, consumers must sort by `event.timestamp`.

## Dead Letter Queue

After max retries exhausted:

```bash
# List DLQ events
GET /api/v1/audit/events?channel=dlq

# Replay a DLQ event
POST /api/v1/webhooks/dlq/replay
{"event_id": "evt-xyz789"}

# Purge old DLQ events (after investigation)
DELETE /api/v1/webhooks/dlq?older_than=30d
```

## Health Monitoring

| Metric | Alert Threshold |
|--------|----------------|
| Delivery success rate | <98% |
| Average delivery latency | >2s |
| DLQ depth | >10 events |
| Consumer 4xx errors | >5% (misconfigured endpoint) |
| Consumer 5xx errors | >1% (consumer down) |
| Retry exhaustion | Any (DLQ entry) |

### Consumer Health Check

```bash
# GGID polls consumer health endpoint
GET https://consumer.example.com/webhooks/health
# Expected: 200 {"status":"ok"}
# Failure → GGID pauses delivery, retries after 60s
```

## Webhook Configuration

```bash
# Register webhook endpoint
POST /api/v1/webhooks/endpoints
{
  "url": "https://app.example.com/webhooks/ggid",
  "events": ["user.created", "user.updated", "user.deleted"],
  "secret": "whsec_...",
  "active": true
}
```

| Setting | Default | Description |
|---------|---------|-------------|
| Max retries | 8 | Before DLQ |
| Timeout | 10s | Per-request HTTP timeout |
| Batch size | 1 | Events per request (future: batching) |
| Secret rotation | 90 days | Webhook signing secret |

## See Also

- [Audit Events API](../api/audit-api.md)
- [OAuth Error Handling](oauth-error-handling.md)
- [API Versioning Strategy](api-versioning-strategy.md)
