# Webhook Event Catalog

All event types, payload schema per event, delivery guarantees, retry policy, signature verification, and sample handlers.

## Event Categories

| Category | Prefix | Events |
|----------|--------|--------|
| User | `user.*` | created, updated, deleted, login, logout, suspended |
| Role | `role.*` | assigned, revoked |
| Group | `group.*` | added, removed |
| Session | `session.*` | created, revoked, expired |
| OAuth | `oauth.*` | consent_granted, consent_revoked, token_issued |
| Policy | `policy.*` | access_allowed, access_denied |
| Provisioning | `provisioning.*` | queued, completed, failed |
| Access Request | `access_request.*` | submitted, approved, denied, expired |
| Threat | `threat.*` | detected, mitigated |

## User Events

### user.created

```json
{
  "event_id": "evt-uuid",
  "event_type": "user.created",
  "timestamp": "2025-01-15T10:00:00Z",
  "tenant_id": "uuid",
  "data": {
    "user_id": "uuid",
    "email": "jane@corp.com",
    "display_name": "Jane Doe",
    "status": "active",
    "department": "Engineering",
    "source": "scim"
  }
}
```

### user.updated

```json
{
  "event_id": "evt-uuid",
  "event_type": "user.updated",
  "timestamp": "2025-01-15T11:00:00Z",
  "data": {
    "user_id": "uuid",
    "changes": {
      "department": {"old": "Engineering", "new": "Product"},
      "title": {"old": "Engineer", "new": "Senior Engineer"}
    }
  }
}
```

### user.deleted

```json
{
  "event_id": "evt-uuid",
  "event_type": "user.deleted",
  "data": {
    "user_id": "uuid",
    "reason": "offboarding",
    "deleted_by": "admin-uuid",
    "sessions_revoked": true
  }
}
```

### user.login

```json
{
  "event_type": "user.login",
  "data": {
    "user_id": "uuid",
    "method": "password+mfa",
    "ip": "10.0.1.5",
    "user_agent": "Mozilla/5.0...",
    "session_id": "sess-uuid",
    "risk_score": 12
  }
}
```

## Role & Group Events

### role.assigned

```json
{
  "event_type": "role.assigned",
  "data": {
    "user_id": "uuid",
    "role_id": "role-uuid",
    "role_name": "Engineering Admin",
    "assigned_by": "admin-uuid",
    "expires_at": "2025-01-16T00:00:00Z"
  }
}
```

### group.added

```json
{
  "event_type": "group.added",
  "data": {
    "user_id": "uuid",
    "group_id": "grp-uuid",
    "group_name": "on_call",
    "added_by": "system"
  }
}
```

## Session Events

### session.revoked

```json
{
  "event_type": "session.revoked",
  "data": {
    "session_id": "sess-uuid",
    "user_id": "uuid",
    "revoked_by": "admin-uuid",
    "reason": "security_incident"
  }
}
```

## Threat Events

### threat.detected

```json
{
  "event_type": "threat.detected",
  "data": {
    "threat_type": "credential_stuffing",
    "user_id": "uuid",
    "ip": "192.168.1.50",
    "severity": "high",
    "mitre_technique": "T1110",
    "action_taken": "account_locked",
    "details": {"failed_attempts": 15, "timeframe": "60s"}
  }
}
```

## Delivery Guarantees

| Guarantee | GGID Implementation |
|-----------|-------------------|
| At-least-once | NATS JetStream durable delivery |
| Ordering | Per-entity (same user_id events in order) |
| Retry | 8 attempts, exponential backoff (30s → 24h) |
| Dead letter | After 8 failures, moved to DLQ |
| Signature | HMAC-SHA256 per delivery |

## Signature Verification

### Header

```http
X-GGID-Event: user.created
X-GGID-Delivery: dpl-abc123
X-GGID-Signature: t=1700000000,v1=8b1a9953c461...
X-GGID-Event-Id: evt-xyz789
```

### Verification (Go)

```go
func verifySignature(payload []byte, header, secret string) error {
    parts := parseSigHeader(header)
    if time.Since(parts.Timestamp) > 5*time.Minute {
        return ErrStaleSignature
    }
    signedPayload := fmt.Sprintf("%d.%s", parts.Timestamp.Unix(), payload)
    expected := hmac.New(sha256.New, []byte(secret))
    expected.Write([]byte(signedPayload))
    if !hmac.Equal([]byte(parts.V1), expected.Sum(nil)) {
        return ErrInvalidSignature
    }
    return nil
}
```

### Verification (Node)

```javascript
const crypto = require('crypto');

function verifyWebhook(rawBody, signature, secret) {
  const { t, v1 } = parseSigHeader(signature);
  
  const signedPayload = `${t}.${rawBody}`;
  const expected = crypto
    .createHmac('sha256', secret)
    .update(signedPayload)
    .digest('hex');
  
  if (expected !== v1) throw new Error('Invalid signature');
  if (Date.now() / 1000 - t > 300) throw new Error('Stale');
}
```

### Verification (Python)

```python
import hmac, hashlib, time

def verify_webhook(payload: bytes, signature: str, secret: str):
    parts = dict(p.split('=') for p in signature.split(','))
    t, v1 = parts['t'], parts['v1']
    
    if time.time() - int(t) > 300:
        raise ValueError('Stale signature')
    
    signed = f"{t}.".encode() + payload
    expected = hmac.new(secret.encode(), signed, hashlib.sha256).hexdigest()
    
    if not hmac.compare_digest(expected, v1):
        raise ValueError('Invalid signature')
```

## Sample Handlers

### Go

```go
func webhookHandler(w http.ResponseWriter, r *http.Request) {
    payload, _ := io.ReadAll(r.Body)
    sigHeader := r.Header.Get("X-GGID-Signature")
    
    if err := verifySignature(payload, sigHeader, webhookSecret); err != nil {
        http.Error(w, "invalid signature", 401)
        return
    }
    
    var event WebhookEvent
    json.Unmarshal(payload, &event)
    
    // Idempotency check
    if seen := redis.SetNX(ctx, "event:"+event.EventID, 1, 24*time.Hour); !seen {
        w.WriteHeader(200) // Already processed
        return
    }
    
    // Route to handler
    switch event.EventType {
    case "user.created":
        handleUserCreated(event.Data)
    case "user.deleted":
        handleUserDeleted(event.Data)
    }
    
    w.WriteHeader(200)
}
```

### Node (Express)

```javascript
app.post('/webhooks/ggid', express.raw({ type: 'application/json' }), (req, res) => {
  try {
    verifyWebhook(req.body, req.get('X-GGID-Signature'), SECRET);
    
    const event = JSON.parse(req.body);
    
    // Idempotency
    if (await cache.has(`event:${event.event_id}`)) {
      return res.status(200).send();
    }
    
    await handleEvent(event);
    await cache.set(`event:${event.event_id}`, 1, 86400);
    
    res.status(200).send();
  } catch (err) {
    res.status(401).send('Verification failed');
  }
});
```

## Monitoring

| Metric | Target | Alert |
|--------|--------|-------|
| Delivery success rate | >98% | <95% → endpoint issue |
| Delivery latency | <2s | >5s → slow endpoint |
| DLQ depth | 0 | >10 → investigate |
| 4xx from consumer | <1% | High → misconfigured endpoint |
| 5xx from consumer | <0.5% | High → consumer down |

## See Also

- [Webhook Delivery Guarantees](webhook-delivery-guarantees.md)
- [Audit Log Architecture](audit-log-architecture.md)
- [Event-Driven Audit](event-driven-audit.md)
- [User Provisioning Pipeline](user-provisioning-pipeline.md)
