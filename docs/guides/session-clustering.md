# Session Clustering Guide

Redis session store, cluster topology, failover, serialization, partition by tenant, eviction policy, and benchmark results.

## Architecture

```
┌──────────┐    ┌──────────┐    ┌──────────┐
│ Gateway  │    │  Auth    │    │ Policy   │
└────┬─────┘    └────┬─────┘    └────┬─────┘
     │               │               │
     └───────────────┼───────────────┘
                     │
              ┌──────┴──────┐
              │ Redis Cluster│
              │ (6 nodes:    │
              │  3 primary + │
              │  3 replica)  │
              └─────────────┘
```

## Redis Cluster Topology

### 3 Primary + 3 Replica

```
Primary 1 (slots 0-5460)  ← Replica 1
Primary 2 (slots 5461-10922) ← Replica 2
Primary 3 (slots 10923-16383) ← Replica 3
```

### Failover

| Event | Action | Time |
|-------|--------|------|
| Primary down | Replica promoted | <10s |
| Node rejoins | Sync as replica | Background |
| Split brain | Majority quorum | Redis Sentinel |

## Session Serialization

```go
type Session struct {
    ID             string    `json:"id"`
    UserID         string    `json:"user_id"`
    TenantID       string    `json:"tenant_id"`
    IP             string    `json:"ip"`
    UserAgent      string    `json:"user_agent"`
    CreatedAt      time.Time `json:"created_at"`
    LastActivity   time.Time `json:"last_activity"`
    ExpiresAt      time.Time `json:"expires_at"`
    Scopes         []string  `json:"scopes"`
    MFAVerified    bool      `json:"mfa_verified"`
    DeviceFingerprint string `json:"device_fp"`
}

func (s *Session) Serialize() ([]byte, error) {
    return json.Marshal(s)  // Compact JSON for storage
}
```

### Serialization Options

| Format | Size | Speed | Use Case |
|--------|------|-------|----------|
| JSON | ~400 bytes | Fast | Default (human-debuggable) |
| MsgPack | ~250 bytes | Faster | Production (40% smaller) |
| Gob | ~200 bytes | Fastest | Internal only (Go-specific) |

## Session Store Operations

### Create Session

```go
func (s *SessionStore) Create(session *Session) error {
    key := "session:" + session.ID
    data, _ := json.Marshal(session)
    
    pipe := s.redis.Pipeline()
    pipe.Set(key, data, time.Until(session.ExpiresAt))
    pipe.SAdd("user:"+session.UserID+":sessions", session.ID)
    pipe.Expire("user:"+session.UserID+":sessions", time.Until(session.ExpiresAt))
    _, err := pipe.Exec()
    return err
}
```

### Get Session

```go
func (s *SessionStore) Get(sessionID string) (*Session, error) {
    data, err := s.redis.Get("session:" + sessionID).Bytes()
    if err == redis.Nil { return nil, ErrSessionNotFound }
    if err != nil { return nil, err }
    
    session := &Session{}
    return session, json.Unmarshal(data, session)
}
```

### Update Activity

```go
func (s *SessionStore) Touch(sessionID string) error {
    // Get current session
    session, err := s.Get(sessionID)
    if err != nil { return err }
    
    // Update last activity
    session.LastActivity = time.Now()
    return s.Save(session)
}
```

### Delete Session

```go
func (s *SessionStore) Delete(sessionID string) error {
    session, _ := s.Get(sessionID)
    
    pipe := s.redis.Pipeline()
    pipe.Del("session:" + sessionID)
    if session != nil {
        pipe.SRem("user:"+session.UserID+":sessions", sessionID)
    }
    _, err := pipe.Exec()
    return err
}
```

### List User Sessions

```go
func (s *SessionStore) ListForUser(userID string) ([]*Session, error) {
    sessionIDs, err := s.redis.SMembers("user:" + userID + ":sessions").Result()
    if err != nil { return nil, err }
    
    sessions := []*Session{}
    for _, id := range sessionIDs {
        if sess, err := s.Get(id); err == nil {
            sessions = append(sessions, sess)
        }
    }
    return sessions, nil
}
```

## Partition by Tenant

### Redis Key Namespacing

```
session:{tenant_id}:{session_id}           # Session data
user:{tenant_id}:{user_id}:sessions        # User session set
tenant:{tenant_id}:active_sessions         # Tenant session count
```

### Benefits

| Benefit | Detail |
|---------|--------|
| Isolation | Scan one tenant without touching others |
| Eviction | Evict per-tenant independently |
| Quota | Enforce max sessions per tenant |
| Debug | Easy to find all sessions for a tenant |

### Cross-Tenant Operations

```go
// Admin can list sessions across tenants (requires admin scope)
func (s *SessionStore) ListAll(tenantID string) ([]*Session, error) {
    pattern := "session:" + tenantID + ":*"
    var sessions []*Session
    
    iter := s.redis.Scan(ctx, 0, pattern, 1000).Iterator()
    for iter.Next(ctx) {
        key := iter.Val()
        data, _ := s.redis.Get(key).Bytes()
        sess := &Session{}
        json.Unmarshal(data, sess)
        sessions = append(sessions, sess)
    }
    return sessions, nil
}
```

## Eviction Policy

### TTL-Based (Primary)

```go
// Session auto-expires via Redis TTL
pipe.Set(key, data, time.Until(session.ExpiresAt))
```

### Idle Timeout

```go
// Cron checks for idle sessions every 5 minutes
func evictIdleSessions() {
    cutoff := time.Now().Add(-30 * time.Minute) // 30 min idle
    pattern := "session:*"
    
    iter := redis.Scan(ctx, 0, pattern, 1000).Iterator()
    for iter.Next(ctx) {
        data, _ := redis.Get(iter.Val()).Bytes()
        sess := &Session{}
        json.Unmarshal(data, sess)
        
        if sess.LastActivity.Before(cutoff) {
            redis.Del(iter.Val())
        }
    }
}
```

### Max Memory Policy

```ini
# redis.conf
maxmemory 2gb
maxmemory-policy volatile-ttl
# Evict keys with nearest TTL first when memory is full
```

## Concurrent Session Limits

```go
func enforceSessionLimit(userID string, maxSessions int) error {
    sessions := store.ListForUser(userID)
    
    if len(sessions) >= maxSessions {
        // Evict oldest session (FIFO)
        oldest := findOldest(sessions)
        store.Delete(oldest.ID)
        audit.Log("session_evicted", map[string]interface{}{
            "user_id": userID,
            "evicted_session": oldest.ID,
            "reason": "concurrent_limit",
        })
    }
    return nil
}
```

| User Type | Max Sessions |
|-----------|-------------|
| Standard | 5 |
| Admin | 2 |
| Service account | 1 |

## Benchmark Results

| Operation | Latency (p50) | Latency (p99) | Throughput |
|-----------|-------------|-------------|-----------|
| Create session | 0.5ms | 1.2ms | 50K/s |
| Get session | 0.2ms | 0.5ms | 100K/s |
| Touch session | 0.3ms | 0.8ms | 80K/s |
| Delete session | 0.3ms | 0.7ms | 90K/s |
| List user sessions | 1.0ms | 3.0ms | 20K/s |
| Evict idle (batch 1000) | 50ms | 100ms | — |

### Test Environment

- Redis 7.2 cluster (3P+3R)
- 1M active sessions
- 100 concurrent connections
- Network: same datacenter (0.5ms RTT)

## Monitoring

| Metric | Target | Alert |
|--------|--------|-------|
| Redis memory usage | <70% | >85% → scale or evict |
| Session creation latency | <1ms | >5ms → Redis overloaded |
| Active session count | Track | Spike → possible bot |
| Eviction rate | Low | High → idle timeout too long |
| Failover events | 0 | Any → investigate |
| Replica lag | <1s | >5s → sync issue |

## See Also

- [Session Security](session-security.md)
- [Gateway Architecture](gateway-architecture.md)
- [Multi-Region Deployment](multi-region-deployment.md)
- [Monitoring and Alerting](monitoring-and-alerting.md)
