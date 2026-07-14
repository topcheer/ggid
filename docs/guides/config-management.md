# Config Management

Env vars, K8s ConfigMap/Secret, feature flags vs env, dynamic vs static, per-tenant overrides, validation, and hot reload.

## Config Hierarchy (Precedence: high to low)

```
1. Per-tenant override (DB)
2. Feature flag (flag engine)
3. Environment variable (K8s Secret/ConfigMap)
4. Default value (code)
```

## Static Config (Environment Variables)

```yaml
# K8s ConfigMap
apiVersion: v1
kind: ConfigMap
metadata:
  name: auth-config
data:
  JWT_TTL: "900"
  MFA_ENABLED: "true"
  RATE_LIMIT_PER_MIN: "100"

---
# K8s Secret (sensitive)
apiVersion: v1
kind: Secret
metadata:
  name: auth-secrets
type: Opaque
stringData:
  JWT_SIGNING_KEY: "..."
  DB_PASSWORD: "..."
```

### Loading

```go
type Config struct {
    JWTTTL          int    `env:"JWT_TTL" default:"900"`
    MFAEnabled      bool   `env:"MFA_ENABLED" default:"true"`
    RateLimitPerMin int    `env:"RATE_LIMIT_PER_MIN" default:"100"`
}

func LoadConfig() (*Config, error) {
    cfg := &Config{}
    return cfg, env.Parse(cfg)
}
```

## Dynamic Config (Feature Flags)

| Aspect | Static (Env Var) | Dynamic (Flag) |
|--------|-----------------|----------------|
| Change requires | Redeploy | API call (instant) |
| Scope | Service-wide | Per-tenant, per-user |
| Use case | DB host, TLS port | New features, kill switches |
| Validation | At startup | At evaluation |

## Per-Tenant Overrides

```bash
# Override config for specific tenant
PUT /api/v1/admin/tenants/{tenant_id}/config
{
  "jwt_ttl": 1800,
  "rate_limit_per_min": 200,
  "mfa_required": true,
  "custom_branding": {"primary_color": "#0052CC"}
}
```

### Evaluation

```go
func getConfig(tenantID, key string) interface{} {
    // 1. Check tenant override
    if val, ok := tenantConfig.Get(tenantID, key); ok {
        return val
    }
    // 2. Fall back to global config
    return globalConfig[key]
}
```

## Validation

```go
func ValidateConfig(cfg *Config) error {
    if cfg.JWTTTL < 60 || cfg.JWTTTL > 86400 {
        return fmt.Errorf("JWT_TTL must be 60-86400")
    }
    if cfg.RateLimitPerMin < 1 {
        return fmt.Errorf("RATE_LIMIT_PER_MIN must be positive")
    }
    return nil
}
```

### Startup Validation

```go
func main() {
    cfg, err := LoadConfig()
    if err != nil { log.Fatal("config load failed:", err) }
    if err := ValidateConfig(cfg); err != nil {
        log.Fatal("config validation failed:", err)
    }
    // Service won't start with invalid config
}
```

## Hot Reload

```go
// Watch ConfigMap for changes (via fsnotify or K8s API)
func watchConfig() {
    watcher, _ := fsnotify.NewWatcher()
    watcher.Add("/etc/ggid/config/")

    for {
        select {
        case event := <-watcher.Events:
            if event.Op&fsnotify.Write != 0 {
                newCfg := loadConfig()
                if err := ValidateConfig(newCfg); err == nil {
                    atomic.StorePointer(&currentConfig, unsafe.Pointer(newCfg))
                    log.Info("config reloaded")
                }
            }
        }
    }
}
```

## Config Audit Trail

```bash
# Track who changed what config when
GET /api/v1/admin/config/history
# → [
#   {"key": "jwt_ttl", "old": "900", "new": "1800", "changed_by": "admin", "timestamp": "..."}
# ]
```

## See Also

- [Feature Flag Architecture](feature-flag-architecture.md)
- [Policy Hot Reload](policy-hot-reload.md)
- [Infrastructure as Code](infrastructure-as-code.md)
- [Secret Sprawl Prevention](secret-sprawl-prevention.md)
