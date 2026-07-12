# OAuth Backpressure Strategy

Token endpoint throttling, queue management, per-client fair queuing, graceful degradation, circuit breaker integration, and rate limit headers.

## Overview

Under high load (traffic spikes, login storms), the OAuth token endpoint can become a bottleneck. Backpressure strategies protect the system while maximizing throughput.

## Token Endpoint Throttling

### Per-Client Rate Limiting

```go
type TokenEndpointThrottle struct {
    limiters map[string]*rate.Limiter // client_id → limiter
}

func (t *TokenEndpointThrottle) Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        clientID := extractClientID(r)
        limiter := t.getOrCreate(clientID)
        
        if !limiter.Allow() {
            w.Header().Set("Retry-After", strconv.Itoa(int(limiter.Tokens())))
            http.Error(w, `{"error":"too_many_requests"}`, 429)
            return
        }
        
        next.ServeHTTP(w, r)
    })
}
```

### Default Limits

| Client Tier | Requests/min | Burst |
|------------|-------------|-------|
| Free | 30 | 60 |
| Standard | 300 | 600 |
| Enterprise | 3000 | 6000 |
| Internal service | 10000 | 20000 |

## Queue Management

### Request Queue

```
Incoming requests
    │
    ▼
┌──────────────┐
│  Token Queue  │  Max: 5000 queued
│  (bounded)    │
└──────┬───────┘
       │ Dequeue (FIFO per client)
       ▼
┌──────────────┐
│ Token Workers │  Pool: 50 goroutines
│ (N workers)  │
└──────────────┘
```

### Implementation

```go
type TokenQueue struct {
    queue   chan TokenRequest
    workers int
}

func NewTokenQueue(workers, queueSize int) *TokenQueue {
    q := &TokenQueue{
        queue:   make(chan TokenRequest, queueSize),
        workers: workers,
    }
    for i := 0; i < workers; i++ {
        go q.worker()
    }
    return q
}

func (q *TokenQueue) Submit(req TokenRequest) error {
    select {
    case q.queue <- req:
        return nil
    default:
        return ErrQueueFull // Backpressure: reject
    }
}

func (q *TokenQueue) worker() {
    for req := range q.queue {
        processTokenRequest(req)
    }
}
```

## Per-Client Fair Queuing

Prevent one noisy client from monopolizing workers:

```go
type FairQueue struct {
    queues   map[string]chan TokenRequest // Per-client queue
    roundRobin []string                   // Client IDs in rotation
    current  int
}

func (fq *FairQueue) dispatch() {
    for {
        // Round-robin across clients
        for i := 0; i < len(fq.roundRobin); i++ {
            idx := (fq.current + i) % len(fq.roundRobin)
            clientID := fq.roundRobin[idx]
            
            select {
            case req := <-fq.queues[clientID]:
                processTokenRequest(req)
                fq.current = (idx + 1) % len(fq.roundRobin)
                break
            default:
                continue // This client's queue is empty
            }
        }
    }
}
```

### Fair Queue Weight

```yaml
fair_queue_weights:
  client-enterprise-1: 5    # Gets 5 slots per round
  client-standard-1: 2     # Gets 2 slots per round
  client-free-1: 1         # Gets 1 slot per round
```

## Graceful Degradation

When system is overwhelmed, degrade gracefully:

| Load Level | Strategy |
|-----------|----------|
| <70% capacity | Normal operation |
| 70-90% | Skip non-critical work (logging, metrics) |
| 90-95% | Only auth-critical endpoints, queue others |
| >95% | Reject low-priority clients, serve enterprise only |
| 100% | Return 503 with Retry-After header |

```go
func TokenHandler(w http.ResponseWriter, r *http.Request) {
    load := getSystemLoad()
    
    switch {
    case load < 0.7:
        processToken(w, r) // Normal
        
    case load < 0.95:
        clientTier := getClientTier(r)
        if clientTier == "free" {
            w.Header().Set("Retry-After", "30")
            http.Error(w, `{"error":"server_busy"}`, 503)
            return
        }
        processToken(w, r) // Paid clients served
        
    default:
        w.Header().Set("Retry-After", "60")
        http.Error(w, `{"error":"server_overloaded"}`, 503)
    }
}
```

## Circuit Breaker Integration

```go
type CircuitBreaker struct {
    failures   int
    threshold  int
    state      string // "closed", "open", "half-open"
    openUntil  time.Time
}

func (cb *CircuitBreaker) Execute(fn func() error) error {
    if cb.state == "open" {
        if time.Now().Before(cb.openUntil) {
            return ErrCircuitOpen // Fast fail
        }
        cb.state = "half-open" // Try again
    }
    
    err := fn()
    if err != nil {
        cb.failures++
        if cb.failures >= cb.threshold {
            cb.state = "open"
            cb.openUntil = time.Now().Add(30 * time.Second)
        }
        return err
    }
    
    cb.failures = 0
    cb.state = "closed"
    return nil
}
```

### Circuit Configuration

| Endpoint | Failure Threshold | Open Duration |
|----------|------------------|---------------|
| Token (DB write) | 10 failures | 30s |
| Token (Redis) | 20 failures | 10s |
| Authorize | 5 failures | 60s |
| Introspect | 15 failures | 15s |

## Rate Limit Headers

```http
HTTP/1.1 200 OK
X-RateLimit-Limit: 300
X-RateLimit-Remaining: 287
X-RateLimit-Reset: 1700000600
X-RateLimit-Policy: 300;w=60

HTTP/1.1 429 Too Many Requests
Retry-After: 30
X-RateLimit-Limit: 300
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1700000600
Content-Type: application/json

{"error": "too_many_requests", "error_description": "Rate limit exceeded. Retry after 30s."}
```

## Monitoring

| Metric | Target | Alert |
|--------|--------|-------|
| Queue depth | <100 | >1000 → scale workers |
| Queue rejection rate | <1% | >5% → increase queue or scale |
| Token endpoint latency p99 | <500ms | >2s → backpressure |
| 429 rate | <1% | >5% → clients need tuning |
| Circuit open events | 0 | Any → investigate |
| Worker utilization | <80% | >95% → add workers |

## See Also

- [OAuth Client Lifecycle](oauth-client-lifecycle.md)
- [Token Introspection Caching](oauth-introspection-caching.md)
- [Gateway Architecture](gateway-architecture.md)
- [Rate Limiting Strategy](rate-limiting-strategy.md)
