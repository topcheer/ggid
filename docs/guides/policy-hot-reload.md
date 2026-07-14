# Policy Hot Reload Guide

Watch-based reload, atomic swap, cache invalidation, zero-downtime policy updates, version checking, and rollback on error.

## Overview

Hot reload updates policy rules without restarting the service. This enables security teams to respond to threats in real-time without downtime.

## Architecture

```
Policy DB Change → Notify Channel (PostgreSQL LISTEN) → Policy Engine
    │
    ├── Reload policies into memory
    ├── Atomically swap active policy set
    ├── Invalidate decision cache
    ├── Verify new policies compile
    └── If error → Rollback to previous version
```

## Watch-Based Reload

### PostgreSQL LISTEN/NOTIFY

```sql
-- Trigger on policy changes
CREATE OR REPLACE FUNCTION notify_policy_change() RETURNS trigger AS $$
BEGIN
  PERFORM pg_notify('policy_changed', json_build_object(
    'action', TG_OP,
    'policy_id', COALESCE(NEW.id, OLD.id),
    'version', COALESCE(NEW.version, OLD.version)
  )::text);
  RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER policy_change_trigger
  AFTER INSERT OR UPDATE OR DELETE ON policies
  FOR EACH ROW EXECUTE FUNCTION notify_policy_change();
```

### Listener

```go
func (e *Engine) StartPolicyWatcher(ctx context.Context) {
    conn, _ := pgx.Connect(ctx, dbURL)
    conn.Exec(ctx, "LISTEN policy_changed")

    for {
        notification, err := conn.WaitForNotification(ctx)
        if err != nil { continue }

        var event PolicyChangeEvent
        json.Unmarshal([]byte(notification.Payload), &event)

        e.ReloadPolicies()
    }
}
```

## Atomic Swap

```go
type Engine struct {
    policies atomic.Value // *PolicySet
}

func (e *Engine) ReloadPolicies() error {
    // 1. Load all policies from DB
    newSet, err := loadPoliciesFromDB()
    if err != nil { return err }

    // 2. Compile all CEL conditions
    for _, p := range newSet.Policies {
        if p.Condition != "" {
            prog, err := cel.Compile(p.Condition)
            if err != nil {
                return fmt.Errorf("policy %s compile error: %w", p.Name, err)
            }
            newSet.compiled[p.Name] = prog
        }
    }

    // 3. Atomic swap (lock-free, all new requests use new policies)
    old := e.policies.Swap(newSet).(*PolicySet)

    // 4. Invalidate cache
    e.cache.Clear()

    // 5. Log
    audit.Log("policy.reloaded", map[string]interface{}{
        "old_version": old.Version,
        "new_version": newSet.Version,
        "policy_count": len(newSet.Policies),
    })

    return nil
}
```

## Version Checking

```go
type PolicySet struct {
    Version   int64
    Policies  map[string]*Policy
    compiled  map[string]cel.Program
    LoadedAt  time.Time
}

func (e *Engine) ReloadPolicies() error {
    // Check version before full reload
    currentVersion := e.policies.Load().(*PolicySet).Version
    dbVersion := getLatestPolicyVersion()

    if dbVersion <= currentVersion {
        return nil // Already up to date
    }

    // Version changed → full reload
    return e.doReload(dbVersion)
}
```

## Rollback on Error

```go
func (e *Engine) ReloadPolicies() error {
    oldSet := e.policies.Load().(*PolicySet)

    newSet, err := loadAndCompile()
    if err != nil {
        // Compilation failed — keep old policies
        log.Error("policy reload failed, keeping old version", err)
        alert.Send("policy_reload_failed", err)
        return err
    }

    // Test new policies with sample requests
    if !e.validatePolicies(newSet) {
        log.Error("policy validation failed, rolling back")
        alert.Send("policy_validation_failed")
        return ErrPolicyValidationFailed
    }

    // Swap
    e.policies.Store(newSet)
    e.cache.Clear()

    // Health check after swap
    time.AfterFunc(5*time.Second, func() {
        if e.errorRate() > 0.1 { // >10% error rate
            // Auto-rollback!
            log.Error("high error rate after policy reload, rolling back")
            e.policies.Store(oldSet)
            e.cache.Clear()
            alert.Send("policy_auto_rollback", "error rate exceeded 10%")
        }
    })

    return nil
}
```

## Cache Invalidation

```go
func (e *Engine) invalidateCache() {
    // Full flush (simplest, safe)
    e.cache.Clear()

    // Or selective invalidation (more efficient)
    // Flush only affected resource types
    for _, resource := range changedResources {
        e.cache.DelPrefix("rbac:*:" + resource + ":")
    }
}
```

## Zero-Downtime Update

```
T=0:    Old policies active, serving requests
T=0.1ms: Atomic swap (pointer update)
T=0.1ms: All new requests use new policies
T=0.1ms: Cache cleared
T=0.2ms: New requests repopulate cache
        Old requests in-flight complete with old policies (already evaluated)

No dropped requests. No downtime. No race conditions.
```

## Monitoring

| Metric | Alert |
|--------|-------|
| Reload failures | Any → investigate |
| Reload latency | >100ms → optimize query |
| Error rate post-reload | >5% → auto-rollback |
| Version mismatch (nodes) | Any → some nodes haven't reloaded |
| Cache miss rate post-reload | Spike expected, normalizes in minutes |

## See Also

- [Policy Evaluation Engine](policy-evaluation-engine.md)
- [Policy Engine Internals](policy-engine-internals.md)
- [RBAC Design Patterns](rbac-design-patterns.md)
- [Conditional Access](conditional-access.md)
