# OAuth Introspection Caching Strategy

Cache key design, TTL tuning, invalidation on revocation, per-client TTL override, stampede prevention, and benchmarks.

## Cache Key Design

```go
func cacheKey(token string) string {
    // Hash the token — never store raw tokens as keys
    h := sha256.Sum256([]byte(token))
    return "introspect:" + hex.EncodeToString(h[:])
}
```

### Key Namespacing

```
introspect:{token_hash}                    → IntrospectionResult
introspect:rs:{resource_server}:{hash}     → Per-resource-server filtered result
introspect:user:{user_id}                  → Set of active token hashes (for invalidation)
```

## TTL Tuning

### Default TTL

```go
const DefaultIntrospectionTTL = 60 * time.Second

func cacheTTL(result *IntrospectionResult) time.Duration {
    // Never cache longer than token expiry
    remaining := time.Until(result.Expiry)
    return min(remaining, DefaultIntrospectionTTL)
}
```

### Per-Client TTL Override

```yaml
client_cache_config:
  client-123:                   # High-volume API
    introspection_ttl: 120s     # Longer cache (stable tokens)
  client-456:                   # Financial app
    introspection_ttl: 10s      # Short cache (strict revocation)
  client-789:                   # Internal service
    introspection_ttl: 300s     # Very long (trusted, low risk)
```

### TTL Guidelines

| Token Risk | TTL | Use Case |
|-----------|-----|----------|
| High (financial/admin) | 10-15s | Strict revocation needed |
| Medium (standard) | 30-60s | Default |
| Low (internal) | 120-300s | Trusted services |

## Invalidation on Revocation

### Token Revocation

```go
func OnTokenRevoked(jti string, userID string) {
    // Remove specific token from cache
    tokenHash := redis.Get("jti:" + jti)
    if tokenHash != "" {
        cache.Del("introspect:" + tokenHash)
        redis.Del("jti:" + jti)
    }
}
```

### User Suspension

```go
func OnUserSuspended(userID string) {
    // Remove ALL of user's introspection cache entries
    tokenHashes := redis.SMembers("introspect:user:" + userID)
    for _, hash := range tokenHashes {
        cache.Del("introspect:" + hash)
    }
    redis.Del("introspect:user:" + userID)
}
```

### Scope Change

```go
func OnScopeChanged(userID string, oldScopes, newScopes []string) {
    // Tokens with old scopes are now invalid
    OnUserSuspended(userID) // Flush all, tokens will re-introspect
}
```

## Cache Stampede Prevention

When cache expires, multiple concurrent requests trigger simultaneous introspection:

### Singleflight

```go
import "golang.org/x/sync/singleflight"

type IntrospectionCache struct {
    sf singleflight.Group
}

func (c *IntrospectionCache) Introspect(token string) (*Result, error) {
    key := cacheKey(token)
    
    // Singleflight ensures only one introspection per token
    result, err, _ := c.sf.Do(key, func() (interface{}, error) {
        // Check cache again (might have been populated by another goroutine)
        if cached, ok := c.cache.Get(key); ok {
            return cached, nil
        }
        
        // Only one request hits the server
        result, err := serverIntrospect(token)
        if err != nil { return nil, err }
        
        c.cache.Set(key, result, cacheTTL(result))
        return result, nil
    })
    
    return result.(*Result), err
}
```

### Probabilistic Early Expiration

```go
func (c *IntrospectionCache) getWithJitter(key string) (*Result, bool) {
    result, expiry, ok := c.cache.GetWithExpiry(key)
    if !ok { return nil, false }
    
    // If within last 10% of TTL, probabilistically refresh early
    remaining := time.Until(expiry)
    if remaining < c.ttl/10 {
        if rand.Float64() < 0.1 { // 10% chance
            return nil, false // Force refresh
        }
    }
    return result, true
}
```

## Cache Implementation

```go
type IntrospectionCache struct {
    cache  *ristretto.Cache
    redis  *redis.Client  // For distributed invalidation
    sf     singleflight.Group
    config map[string]ClientCacheConfig
}

func (c *IntrospectionCache) Get(token, clientID string) (*IntrospectionResult, error) {
    key := cacheKey(token)
    ttl := c.getClientTTL(clientID)
    
    // 1. Check ristretto (local, fast)
    if result, ok := c.cache.Get(key); ok {
        return result.(*IntrospectionResult), nil
    }
    
    // 2. Singleflight (prevent stampede)
    result, err, _ := c.sf.Do(key, func() (interface{}, error) {
        // 3. Check Redis (distributed)
        if data, err := c.redis.Get(key).Bytes(); err == nil {
            r := &IntrospectionResult{}
            json.Unmarshal(data, r)
            c.cache.Set(key, r, ttl)
            return r, nil
        }
        
        // 4. Introspect at server
        r, err := serverIntrospect(token)
        if err != nil { return nil, err }
        
        // 5. Cache in both layers
        data, _ := json.Marshal(r)
        c.cache.Set(key, r, ttl)
        c.redis.Set(key, data, ttl)
        c.redis.SAdd("introspect:user:"+r.UserID, hashToken(token))
        
        return r, nil
    })
    
    return result.(*IntrospectionResult), err
}
```

## Benchmark Results

| Scenario | Latency (p50) | Latency (p99) | Cache Hit Rate |
|----------|-------------|-------------|----------------|
| L1 hit (ristretto) | 0.01ms | 0.05ms | 92% |
| L2 hit (Redis) | 0.3ms | 0.8ms | 6% |
| Server introspection | 15ms | 40ms | 2% |
| With singleflight | 15ms (1 req) | 40ms | Stampede prevented |
| 1000 concurrent (same token) | 15ms total | 40ms | 1 server call |

### Configuration

- Ristretto: 500K capacity, 10min max TTL
- Redis: 2GB maxmemory, volatile-ttl eviction
- Singleflight: Prevents stampede for identical keys

## Monitoring

| Metric | Target | Alert |
|--------|--------|-------|
| L1 cache hit rate | >90% | <80% → increase capacity |
| L2 cache hit rate | >5% | — |
| Server introspection rate | <5% of total | >10% → check invalidation storm |
| Stampede prevention | Track | High singleflight → cache TTL too short |
| Cache memory usage | <70% | >85% → scale |

## See Also

- [Token Introspection Design](token-introspection-design.md)
- [JWT Security Best Practices](jwt-security-best-practices.md)
- [Session Clustering](session-clustering.md)
- [Policy Evaluation Engine](policy-evaluation-engine.md)
