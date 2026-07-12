# Webhook Retry Design

Retry strategy, exponential backoff, jitter, max attempts, dead letter queue, idempotency, manual replay, and monitoring.

## Retry Strategy

### Schedule

| Attempt | Delay | Cumulative |
|---------|-------|-----------|
| 1 | Immediate | 0s |
| 2 | 30s | 30s |
| 3 | 2min | 2.5m |
| 4 | 10min | 12.5m |
| 5 | 1h | 1h12m |
| 6 | 6h | 7h12m |
| 7 | 24h | 31h |
| 8 | → DLQ | 31h |

### Backoff with Jitter

```go
func retryDelay(attempt int) time.Duration {
    base := []time.Duration{
        0, 30 * time.Second, 2 * time.Minute, 10 * time.Minute,
        time.Hour, 6 * time.Hour, 24 * time.Hour,
    }
    if attempt >= len(base) {
        return 0 // Move to DLQ
    }
    // Add ±20% jitter to prevent thundering herd
    delay := base[attempt]
    jitter := time.Duration(rand.Int63n(int64(delay) / 5))
    return delay + jitter - jitter/2
}
```

## Response Handling

| HTTP Response | Action |
|-------------|--------|
| 2xx | Success, remove from queue |
| 3xx | Follow redirect (max 3) |
| 4xx (except 408/429) | Client error, move to DLQ (don't retry) |
| 408 / 429 | Rate limited, retry with backoff |
| 5xx | Server error, retry |
| Timeout | Network error, retry |

## Idempotency

```go
func deliver(webhook *Webhook, endpoint string) error {
    // Include event_id for consumer dedup
    payload, _ := json.Marshal(webhook)
    
    resp, err := http.Post(endpoint, "application/json", payload)
    if err != nil { return err }
    
    if resp.StatusCode >= 200 && resp.StatusCode < 300 {
        markDelivered(webhook.ID)
    }
    
    // Consumer must use event_id for idempotency:
    // if seen(event_id) { return ok } else { process + mark seen }
}
```

## Dead Letter Queue

```bash
# List DLQ events
GET /api/v1/audit/events?channel=webhook_dlq

# Replay single event
POST /api/v1/webhooks/dlq/{event_id}/replay

# Replay all DLQ events for a webhook endpoint
POST /api/v1/webhooks/dlq/replay-all?endpoint_id=ep-123

# Purge old DLQ entries
DELETE /api/v1/webhooks/dlq?older_than=30d
```

## Monitoring

| Metric | Alert |
|--------|-------|
| Delivery success rate | <98% |
| Average attempts per delivery | >1.5 → consumer slow |
| DLQ depth | >10 |
| 4xx from consumer | >5% → misconfigured endpoint |
| Consumer response time | >5s → timeout risk |

## See Also

- [Webhook Delivery Guarantees](webhook-delivery-guarantees.md)
- [Webhook Event Catalog](webhook-event-catalog.md)
- [Event-Driven Audit](event-driven-audit.md)
- [OAuth Backpressure](oauth-backpressure.md)
