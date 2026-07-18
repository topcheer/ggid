# Plugin System & WASM Enhancement: Lifecycle, Hooks, SDK, Marketplace, and Hardening

> **Focus**: Enhancing GGID's existing WASM plugin host (`wasm_plugin.go`, 434 lines, wazero runtime) with full lifecycle management, hook system, plugin SDK API, marketplace, hot reload, and security hardening. Builds on `wasm-plugin-architecture.md` and `plugin-system.md`.
>
> **Author**: ggcxf (researcher) | **Date**: 2026-07-17 | **Status**: Research Complete
>
> **Checklist Compliance**: DoD per backlog item (§8).

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [GGID Current State: WASM Plugin Host](#2-ggid-current-state-wasm-plugin-host)
3. [Gap Analysis](#3-gap-analysis)
4. [Plugin Lifecycle Management](#4-plugin-lifecycle-management)
5. [Hook System](#5-hook-system)
6. [Plugin SDK API](#6-plugin-sdk-api)
7. [WASM Runtime Hardening](#7-wasm-runtime-hardening)
8. [Hot Reload](#8-hot-reload)
9. [Plugin Marketplace](#9-plugin-marketplace)
10. [Implementation Backlog with DoD](#10-implementation-backlog-with-dod)
11. [Competitive Differentiation](#11-competitive-differentiation)

---

## 1. Executive Summary

GGID has a **functional WASM plugin host** (`wasm_plugin.go:69`, 434 lines) using wazero (Go-native WASM runtime). It supports:
- Plugin loading from `.wasm` files ✅
- WASI preview1 imports ✅
- HMAC-SHA256 signature verification ✅
- Memory limit (16MB) + timeout (100ms) ✅
- DB-backed plugin store (`plugin_repo.go`) ✅
- Plugin CRUD API ✅

**What's missing:**
1. **No hook system** — Plugins can't hook into auth/policy/audit events
2. **No plugin SDK** — Plugins have no structured API to read context
3. **No lifecycle management** — No enable/disable/validate flow
4. **No hot reload** — Plugin update requires restart
5. **No marketplace** — No discovery/distribution
6. **No capability-based security** — All plugins have same permissions

**Recommendation**: Build a **Plugin Hook System** (6 lifecycle hooks), **Plugin SDK** (host functions for context/data/HTTP), **Lifecycle Manager** (upload→validate→sandbox→enable→invoke→disable→delete), and **Hot Reload** (atomic swap).

---

## 2. GGID Current State

### Existing WASM Host

| Component | File:Line | Status |
|-----------|-----------|--------|
| WasmPluginHost | `wasm_plugin.go:69` | ✅ wazero runtime |
| WasmPluginConfig | `wasm_plugin.go:22` | ✅ Path, hooks, signature |
| LoadPlugin | `wasm_plugin.go:142` | ✅ Compile + instantiate |
| Signature verification | `wasm_plugin.go:408` | ✅ HMAC-SHA256 + sidecar .sig |
| Runtime config | `wasm_plugin.go:118` | ✅ Memory limit configurable |
| WASI imports | `wasm_plugin.go:19` | ✅ wasi_snapshot_preview1 |
| Plugin repo (DB) | `plugin_repo.go` | ✅ CRUD + versioning |
| Plugin config | `plugin_repo.go:18` | ✅ hooks, enabled, max_memory, timeout |

### Current Limitations

```go
// wasm_plugin.go:22 — Config exists but hooks field is just strings
type WasmPluginConfig struct {
    WasmPath   string            // path to .wasm file
    Config     map[string]any    // plugin-specific config
    Hooks      []string          // ["pre_auth", "post_auth", ...] — NOT WIRED
    Signature  string            // HMAC signature
}
```

The `Hooks` field exists in config but **is not wired to actual lifecycle events** — plugins are loaded but never invoked at specific points.

---

## 3. Gap Analysis

| # | Gap | Impact |
|---|-----|--------|
| 1 | Hooks field not wired | Plugins never invoked |
| 2 | No plugin SDK (host functions) | Plugins can't read request context |
| 3 | No lifecycle management | No enable/disable/validate |
| 4 | No hot reload | Update = restart |
| 5 | No capability model | All plugins = same access |
| 6 | No HTTP from plugins | Can't call external APIs |
| 7 | No marketplace | No discovery |

---

## 4. Plugin Lifecycle Management

### Lifecycle States

```
upload → validate → sandbox → enable → invoke → disable → delete
  │         │          │         │         │         │
  │    checksum    memory test  activated  running  deactivated
  │    signature   timeout test            (hooks)   (stops)
  │    parse WASM  security scan
```

### Lifecycle API

```bash
# Upload
POST /api/v1/gateway/plugins/upload
  Body: multipart/form-data (wasm file + config)

# Validate (dry-run)
POST /api/v1/gateway/plugins/{id}/validate

# Enable
POST /api/v1/gateway/plugins/{id}/enable

# Disable (stops invocations, keeps loaded)
POST /api/v1/gateway/plugins/{id}/disable

# Hot reload (atomic swap to new version)
POST /api/v1/gateway/plugins/{id}/reload

# Delete (unload + remove)
DELETE /api/v1/gateway/plugins/{id}
```

---

## 5. Hook System

### 6 Lifecycle Hooks

| Hook | When | Can Modify | Use Cases |
|------|------|-----------|-----------|
| `pre_auth` | Before credential check | Request (add headers) | Bot detection, IP reputation |
| `post_auth` | After successful auth | Response (add claims) | Custom token enrichment |
| `pre_policy` | Before PDP decision | Request context | Risk score injection |
| `post_policy` | After decision | Decision (override) | Custom authz logic |
| `pre_response` | Before response sent | Response body | DLP, response transformation |
| `on_audit` | Before audit write | Event data | PII redaction in audit |

### Hook Registration

```go
// Plugin exports a function per hook it wants:
// export fn pre_auth(ctx_ptr: i32, ctx_len: i32) -> i32
// export fn post_auth(ctx_ptr: i32, ctx_len: i32) -> i32

// Host checks which hooks the plugin registered:
func (h *WasmPluginHost) GetHooks() []string {
    hooks := []string{}
    for _, hook := range allHooks {
        if h.module.ExportedFunction(hook) != nil {
            hooks = append(hooks, hook)
        }
    }
    return hooks
}
```

### Hook Invocation

```go
func (h *WasmPluginHost) InvokeHook(ctx context.Context, hook string, reqCtx *PluginContext) (*PluginResult, error) {
    fn := h.module.ExportedFunction(hook)
    if fn == nil {
        return nil, nil  // Plugin doesn't hook this event
    }

    // Serialize context to WASM memory
    ctxJSON, _ := json.Marshal(reqCtx)
    ctxPtr, err := h.allocate(ctxJSON)

    // Call plugin function
    results, err := fn(ctx, ctxPtr, len(ctxJSON))

    // Read result from WASM memory
    result := h.readResult(results[0])

    return result, nil
}
```

---

## 6. Plugin SDK API

### Host Functions (imported by WASM plugin)

| Function | Purpose | Capability Required |
|----------|---------|---------------------|
| `ggid_log(level, msg_ptr, msg_len)` | Structured logging | `log` |
| `ggid_get_header(name_ptr, name_len) → val_ptr` | Read request header | `read_request` |
| `ggid_set_header(name_ptr, name_len, val_ptr, val_len)` | Set response header | `modify_response` |
| `ggid_get_claim(key_ptr, key_len) → val_ptr` | Read JWT claim | `read_token` |
| `ggid_http_get(url_ptr, url_len) → resp_ptr` | HTTP GET to external API | `http_outbound` |
| `ggid_redis_get(key_ptr, key_len) → val_ptr` | Redis lookup | `redis_read` |
| `ggid_metric(name_ptr, name_len, value)` | Record metric | `metrics` |

### Capability-Based Security

```go
type PluginCapabilities struct {
    ReadRequest    bool   `json:"read_request"`
    ModifyResponse bool   `json:"modify_response"`
    ReadToken      bool   `json:"read_token"`
    HTTPOutbound   bool   `json:"http_outbound"`
    RedisRead      bool   `json:"redis_read"`
    RedisWrite     bool   `json:"redis_write"`
    Log            bool   `json:"log"`
    Metrics        bool   `json:"metrics"`
}
```

### Example Plugin (Rust → WASM)

```rust
// plugins/ip-reputation/src/lib.rs
use ggid_sdk::{pre_auth, PluginContext, Response};

#[no_mangle]
pub extern "C" fn pre_auth(ctx_ptr: i32, ctx_len: i32) -> i32 {
    let ctx = PluginContext::from_host(ctx_ptr, ctx_len);

    // Check IP reputation via HTTP
    let ip = ctx.get_header("X-Forwarded-For");
    let reputation = ggid_http_get(&format!("https://reputation.api/check/{}", ip));

    if reputation.contains("malicious") {
        return Response::deny("ip_reputation_blocked").encode();
    }

    Response::allow().encode()
}
```

---

## 7. WASM Runtime Hardening

### Resource Limits

| Resource | Default | Configurable |
|----------|---------|-------------|
| Max memory | 16 MB | Per-plugin |
| Max execution time | 100ms | Per-hook |
| Max HTTP calls | 3 per invocation | Per-plugin |
| Max Redis calls | 5 per invocation | Per-plugin |
| Filesystem | None (no FS access) | Fixed |
| Network | HTTP outbound only (if capability) | Per-plugin |

### Timeout Enforcement

```go
func (h *WasmPluginHost) InvokeHook(ctx context.Context, hook string, reqCtx *PluginContext) (*PluginResult, error) {
    timeoutCtx, cancel := context.WithTimeout(ctx, h.config.TimeoutMs*time.Millisecond)
    defer cancel()

    done := make(chan *PluginResult, 1)
    go func() {
        result, err := h.invokeInternal(ctx, hook, reqCtx)
        done <- result
    }()

    select {
    case result := <-done:
        return result, nil
    case <-timeoutCtx.Done():
        h.metrics.TimeoutCount.Inc()
        return nil, ErrPluginTimeout
    }
}
```

---

## 8. Hot Reload

### Atomic Swap Pattern

```go
func (h *WasmPluginHost) HotReload(ctx context.Context, newWasmPath string) error {
    // 1. Load new module (compile + validate)
    newModule, err := h.runtime.CompileModule(ctx, wasmBytes)
    if err != nil {
        return err
    }

    // 2. Instantiate new module
    newHost := h.instantiate(newModule)

    // 3. Atomic swap (pointer swap under lock)
    h.mu.Lock()
    oldModule := h.module
    h.module = newModule
    h.mu.Unlock()

    // 4. Wait for in-flight invocations to complete
    h.wg.Wait()

    // 5. Close old module
    oldModule.Close(ctx)

    return nil
}
```

---

## 9. Plugin Marketplace

### Marketplace Flow

```
Developer:
  1. Write plugin in Rust/C/AssemblyScript
  2. Compile to .wasm
  3. Sign with developer key
  4. Publish to marketplace

Admin:
  1. Browse marketplace (list plugins)
  2. Read reviews + ratings
  3. Install: download + verify signature + load
  4. Configure: set hooks + capabilities + limits
  5. Enable
```

### Marketplace Metadata

```json
{
  "name": "ip-reputation-checker",
  "version": "1.2.0",
  "author": "ggid-community",
  "description": "Checks client IP against threat intelligence feeds",
  "hooks": ["pre_auth"],
  "capabilities": ["read_request", "http_outbound", "log"],
  "wasm_hash": "sha256:abc123...",
  "signature": "ed25519:def456...",
  "config_schema": {
    "api_key": { "type": "string", "required": true }
  }
}
```

---

## 10. Implementation Backlog with DoD

### P0 — Hook System + SDK (2 sprints)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 1 | Wire hooks to lifecycle events (6 hooks) | ✅ pre_auth/post_auth/pre_policy/post_policy/pre_response/on_audit ✅ ≥3 tests | 4d |
| 2 | Plugin SDK host functions (7 functions) | ✅ ggid_log/get_header/set_header/get_claim/http_get/redis_get/metric ✅ ≥3 tests | 4d |
| 3 | Capability-based security model | ✅ Per-plugin capabilities ✅ Denied without capability ✅ ≥3 tests | 2d |
| 4 | Plugin lifecycle API (upload/enable/disable/delete) | ✅ Full lifecycle ✅ DB-backed ✅ ≥3 tests | 3d |

### P1 — Hot Reload + Marketplace (1 sprint)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 5 | Hot reload (atomic swap) | ✅ Zero-downtime update ✅ In-flight completion ✅ ≥3 tests | 3d |
| 6 | Plugin validation pipeline | ✅ Checksum + signature + WASM parse ✅ Security scan ✅ ≥3 tests | 2d |
| 7 | Plugin marketplace API | ✅ List/install/publish ✅ Metadata ✅ ≥3 tests | 3d |

### P2 — Advanced (Future)

| # | Task | DoD |
|---|------|-----|
| 8 | Plugin SDK for Rust | Full Rust crate + docs |
| 9 | Plugin SDK for AssemblyScript | TypeScript-like plugin dev |
| 10 | Per-tenant plugin isolation | Resource quotas per tenant |
| 11 | Plugin metrics dashboard | Invocation count + latency + errors |
| 12 | Community marketplace | Public registry + signing chain |

---

## 11. Competitive Differentiation

| Feature | GGID (target) | Envoy WASM | Cloudflare Workers | Kong Plugins | Auth0 Actions |
|---------|---------------|-----------|-------------------|-------------|---------------|
| **Runtime** | wazero (Go) | Wasmtime | V8 isolates | Lua/WASM | Node.js |
| **Hook system** | 6 lifecycle hooks | 5 phases | Fetch event | 6 phases | 5 actions |
| **Plugin SDK** | 7 host functions | ABI | Web APIs | PDK | Node SDK |
| **Hot reload** | Atomic swap | ✅ | ✅ | ❌ | N/A |
| **Capabilities** | Per-plugin | Per-policy | Per-worker | Per-plugin | Sandboxed |
| **Marketplace** | Built-in | External | ✅ | ✅ | ✅ |
| **Open source** | Yes | Yes | No | Partially | No |

---

## References

- [wazero](https://wazero.io/) — Go-native WASM runtime
- [Extism](https://extism.org/) — Universal plugin framework
- [Wasmtime](https://wasmtime.dev/) — Standalone WASM runtime
- [Envoy WASM Filters](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/wasm_filter) — Proxy plugins
- [Cloudflare Workers](https://workers.cloudflare.com/) — Edge WASM runtime
- [Kong Plugin Development Kit](https://docs.konghq.com/gateway/latest/plugin-development/) — PDK
- [Auth0 Actions](https://auth0.com/docs/customize/actions) — Extensibility
- [GGID WASM Plugin Host](../services/gateway/internal/middleware/wasm_plugin.go) — At line 69
- [GGID Plugin Repo](../services/gateway/internal/middleware/plugin_repo.go) — DB-backed at line 18
- [GGID WASM Plugin Architecture](./wasm-plugin-architecture.md) — Previous research
- [GGID Plugin System](./plugin-system.md) — Previous research
- [GGID API Gateway Hardening](./api-gateway-hardening.md) — Gateway middleware stack
