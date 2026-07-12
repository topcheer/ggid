# Health Check Design

Liveness, readiness, and startup probes — dependency checks, graceful degradation, circuit breaker integration, and auto-healing.

## Probe Types

| Probe | Purpose | Fail Action | Frequency |
|-------|---------|-------------|-----------|
| Liveness | Is the process alive? | Restart pod | Every 10s |
| Readiness | Can I handle requests? | Remove from LB | Every 5s |
| Startup | Has initialization completed? | Block liveness until done | Until first success |

## Endpoints

### Liveness

```go
func LivenessHandler(w http.ResponseWriter, r *http.Request) {
    // Just check the process is responsive
    w.WriteHeader(200)
    json.NewEncoder(w).Encode(map[string]string{"status": "alive"})
}
```

### Readiness

```go
func ReadinessHandler(hc *HealthChecker) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        results := hc.CheckAll()
        allReady := true
        
        for _, result := range results {
            if !result.Healthy {
                allReady = false
            }
        }
        
        if allReady {
            w.WriteHeader(200)
        } else {
            w.WriteHeader(503) // LB removes from rotation
        }
        
        json.NewEncoder(w).Encode(map[string]interface{}{
            "status": allReady ? "ready" : "degraded",
            "checks": results,
        })
    }
}
```

### Response

```json
{
  "status": "ready",
  "checks": {
    "database": {"status": "healthy", "latency_ms": 2},
    "redis": {"status": "healthy", "latency_ms": 1},
    "nats": {"status": "healthy", "latency_ms": 3},
    "ldap": {"status": "degraded", "latency_ms": 500}
  }
}
```

## Dependency Checks

### Database

```go
func (hc *HealthChecker) checkDB() CheckResult {
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()
    
    start := time.Now()
    err := hc.db.Ping(ctx)
    latency := time.Since(start)
    
    if err != nil {
        return CheckResult{Healthy: false, Error: err.Error()}
    }
    return CheckResult{Healthy: true, Latency: latency}
}
```

### Redis

```go
func (hc *HealthChecker) checkRedis() CheckResult {
    ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
    defer cancel()
    
    start := time.Now()
    err := hc.redis.Ping(ctx).Err()
    
    return CheckResult{Healthy: err == nil, Latency: time.Since(start)}
}
```

### NATS

```go
func (hc *HealthChecker) checkNATS() CheckResult {
    if !hc.nats.Conn().IsConnected() {
        return CheckResult{Healthy: false, Error: "not connected"}
    }
    
    start := time.Now()
    _, err := hc.nats.JetStream().StreamInfo("AUDIT_EVENTS")
    return CheckResult{Healthy: err == nil, Latency: time.Since(start)}
}
```

### LDAP

```go
func (hc *HealthChecker) checkLDAP() CheckResult {
    if hc.ldapURL == "" {
        return CheckResult{Healthy: true, Skipped: true, Reason: "LDAP not configured"}
    }
    
    conn, err := ldap.DialURL(hc.ldapURL, ldap.DialWithTimeout(2*time.Second))
    if err != nil {
        return CheckResult{Healthy: false, Error: err.Error()}
    }
    defer conn.Close()
    
    return CheckResult{Healthy: true}
}
```

## Kubernetes Probes

```yaml
livenessProbe:
  httpGet:
    path: /healthz/live
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 10
  failureThreshold: 3      # Restart after 3 failures

readinessProbe:
  httpGet:
    path: /healthz/ready
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
  failureThreshold: 2      # Remove from LB after 2 failures

startupProbe:
  httpGet:
    path: /healthz/startup
    port: 8080
  initialDelaySeconds: 0
  periodSeconds: 5
  failureThreshold: 30     # Allow up to 150s for startup
```

## Graceful Degradation

When a non-critical dependency is down, degrade rather than fail:

```go
func (hc *HealthChecker) CheckAll() map[string]CheckResult {
    results := map[string]CheckResult{}
    
    // Critical: DB must be healthy
    dbCheck := hc.checkDB()
    results["database"] = dbCheck
    if !dbCheck.Healthy {
        results["status"] = "not_ready"
        return results // Can't operate without DB
    }
    
    // Non-critical: Redis (degrade to stateless)
    redisCheck := hc.checkRedis()
    results["redis"] = redisCheck
    if !redisCheck.Healthy {
        results["status"] = "degraded"
        // Still serve requests, but without cache
        return results
    }
    
    // Non-critical: LDAP (degrade to local auth)
    ldapCheck := hc.checkLDAP()
    results["ldap"] = ldapCheck
    if !ldapCheck.Healthy && !ldapCheck.Skipped {
        results["status"] = "degraded"
        // Still serve, LDAP auth will fail but local auth works
    }
    
    results["status"] = "ready"
    return results
}
```

## Auto-Healing

```yaml
auto_healing:
  pod_restart:
    trigger: "liveness_fail > 3"
    action: "restart pod"
    cooldown: "60s"
    
  db_connection_pool_reset:
    trigger: "db_check fail > 5"
    action: "reset connection pool"
    cooldown: "30s"
    
  redis_reconnect:
    trigger: "redis_check fail"
    action: "reconnect Redis client"
    cooldown: "10s"
```

## Monitoring

| Metric | Alert |
|--------|-------|
| Readiness failures | >1% → dependency issue |
| Check latency | DB >100ms → slow queries |
| Dependency down | Any critical → page |
| Pod restarts | >2/hour → investigate |
| Degraded state duration | >5min → escalate |

## See Also

- [Gateway Architecture](gateway-architecture.md)
- [gRPC Interceptor Patterns](grpc-interceptor-patterns.md)
- [Observability Guide](observability-guide.md)
- [Monitoring and Alerting](monitoring-and-alerting.md)
