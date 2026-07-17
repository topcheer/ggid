# WASM Plugin Architecture: Extensible Policy, Transformation, and Custom Logic for GGID

> **Focus**: A comprehensive WASM plugin system enabling GGID tenants to upload custom logic — claim transformations, JIT mappings, DLP rules, custom policy evaluations — running in a secure sandbox with resource limits, per-tenant isolation, and hooks into the auth/policy/gateway pipeline.
>
> **Author**: ggcxf (researcher) | **Date**: 2026-07-17 | **Status**: Research Complete
>
> **Checklist Compliance**: Includes endpoint precondition check (§7), DoD per backlog item (§17), curl verification commands (§11).

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [What is WASM and Why It Matters for IAM](#2-what-is-wasm-and-why-it-matters-for-iam)
3. [GGID Current State: Gateway WASM Host](#3-ggid-current-state-gateway-wasm-host)
4. [Gap Analysis](#4-gap-analysis)
5. [Proposed Architecture: Universal Plugin Engine](#5-proposed-architecture-universal-plugin-engine)
6. [Plugin Hook Points](#6-plugin-hook-points)
7. [Endpoint Precondition Check](#7-endpoint-precondition-check)
8. [Sandbox Security Model](#8-sandbox-security-model)
9. [Plugin SDK Design](#9-plugin-sdk-design)
10. [Database Schema](#10-database-schema)
11. [API Design + Curl Commands](#11-api-design--curl-commands)
12. [Performance Considerations](#12-performance-considerations)
13. [Console UI Design](#13-console-ui-design)
14. [Competitive Differentiation](#14-competitive-differentiation)
15. [Migration Strategy](#15-migration-strategy)
16. [Security Considerations](#16-security-considerations)
17. [Implementation Backlog with DoD](#17-implementation-backlog-with-dod)

---

## 1. Executive Summary

WebAssembly (WASM) is the ideal sandbox runtime for IAM plugin systems: near-native speed, language-agnostic (compile from Rust/Go/C/AssemblyScript), memory-safe by default, and trivially sandboxable. Envoy, Istio, and Cloudflare Workers all use WASM for extensible proxy logic.

GGID already implements a **Gateway WASM Plugin Host** using wazero (`services/gateway/internal/middleware/wasm_plugin.go:34` — `WasmPluginHost`). It supports:
- Plugin load/unload/execute lifecycle
- Request and response phase hooks
- PluginContext with method, path, headers, body, tenant_id, user_id
- PluginResult with block/modify capability
- wazero runtime with WASI support

However, this is **gateway-only and request/response-only**. The plugin system does not extend to:
1. **Auth pipeline hooks** — no hooks for pre-login, post-login, token issuance, MFA
2. **Policy evaluation hooks** — no custom ABAC/ReBAC evaluation plugins
3. **Claim transformation** — no JWT/OIDC claim enrichment via plugins
4. **JIT mapping** — no plugin hooks in the JIT provisioning pipeline
5. **DLP rules** — no plugin hooks for data loss prevention scanning
6. **Plugin management API** — no REST endpoints for upload/manage/enable/disable
7. **Plugin storage** — plugins stored in filesystem only, not DB
8. **Per-tenant isolation** — plugins not isolated by tenant_id
9. **Resource limits** — no memory/CPU/execution-time limits enforced

**Recommendation**: Extend the existing Gateway WASM host into a **Universal Plugin Engine** with hooks across the entire auth/policy/gateway pipeline, DB-backed plugin storage, per-tenant isolation, resource limits, and a management API.

**Estimated effort**: 4 sprints for MVP (auth hooks + policy hooks + claim transform + management API + Console UI).

---

## 2. What is WASM and Why It Matters for IAM

### WASM Advantages for IAM Plugins

| Property | Benefit for IAM |
|----------|----------------|
| **Sandboxed by design** | Plugins can't access host memory, filesystem, or network unless explicitly granted |
| **Language-agnostic** | Tenants write plugins in Rust, Go, AssemblyScript, C, or any WASM-targeting language |
| **Near-native speed** | 10-100x faster than JavaScript/Python plugins; suitable for per-request execution |
| **Deterministic** | No garbage collection pauses (in some runtimes); predictable latency |
| **Portable** | Same .wasm binary runs on any platform (x86, ARM, Linux, macOS) |
| **Versioned** | Each .wasm is immutable; easy rollback by swapping versions |
| **Resource-limitable** | Memory cap, fuel/gas metering, execution timeout via wazero's `ResourceLimiter` |

### Comparison with Alternative Plugin Approaches

| Approach | Security | Performance | Flexibility | Complexity |
|----------|---------|-------------|-------------|-----------|
| **WASM (proposed)** | **Highest** (sandbox) | **Near-native** | **Any language** | Medium |
| JavaScript (V8) | Medium (isolates) | Good | JS only | Medium |
| Lua (embedded) | Medium | Good | Lua only | Low |
| Go plugins (.so) | **Low** (shared process) | Native | Go only | Low |
| External HTTP webhook | High (separate process) | **Poor** (network hop) | Any | High |
| CEL expressions | High (no I/O) | Excellent | Limited (expressions only) | Low |

**Key insight**: WASM is the only approach that combines **sandbox-level security** with **near-native performance** and **language flexibility**. External webhooks add latency; CEL can't do complex logic; Go plugins compromise the host process.

---

## 3. GGID Current State: Gateway WASM Host

### Existing Implementation

| Component | File:Line | Status |
|-----------|-----------|--------|
| WasmPluginHost | `wasm_plugin.go:38` | **Implemented** — wazero runtime, plugin map |
| LoadPlugin | `wasm_plugin.go:105` | **Implemented** — compile + instantiate from file path |
| Execute | `wasm_plugin.go` (~line 250) | **Implemented** — call plugin with PluginContext |
| UnloadPlugin | `wasm_plugin.go` | **Implemented** — close module + remove from map |
| WasmMiddleware | `wasm_plugin.go` (~line 260) | **Implemented** — HTTP middleware iterating loaded plugins |
| PluginContext | `wasm_plugin.go:60` | **Implemented** — method, path, headers, body, tenant_id, user_id |
| PluginResult | `wasm_plugin.go:71` | **Implemented** — status, headers, body, should_block, modified_* |
| WASI support | `wasm_plugin.go:85` | **Implemented** — `wasi_snapshot_preview1.MustInstantiate` |
| PluginMetadata | `wasm_plugin.go:51` | **Implemented** — name, version, author, description, hooks |
| Close | `wasm_plugin.go:94` | **Implemented** — cleanup all plugins + runtime |
| Tests | `wasm_plugin_test.go` | **Comprehensive** — 10+ test cases for load/execute/unload/middleware |

### Runtime: wazero

GGID uses [wazero](https://github.com/tetratelabs/wazero) — a pure-Go WebAssembly runtime with zero dependencies. This is an excellent choice:
- No CGO required (pure Go)
- Embeds directly in GGID binary
- Supports WASI preview 1
- Has `ResourceLimiter` interface for memory/fuel limits
- Compiled modules are reusable across goroutines

### What's Missing

| # | Gap | Impact |
|---|-----|--------|
| 1 | **No auth pipeline hooks** | Can't run plugins at pre-login, post-login, token-issuance, MFA challenge |
| 2 | **No policy evaluation hooks** | Can't use custom logic in ABAC/ReBAC evaluation |
| 3 | **No claim transformation** | Can't enrich/modify JWT/OIDC claims via plugin |
| 4 | **No plugin management API** | No REST CRUD for upload/enable/disable/version |
| 5 | **No DB-backed plugin storage** | Plugins only on filesystem; lost on restart |
| 6 | **No per-tenant isolation** | All plugins share same namespace |
| 7 | **No resource limits** | No memory cap, execution timeout, or fuel metering |
| 8 | **No plugin SDK** | No documented API for plugin authors |
| 9 | **No hot reload** | Plugin updates require service restart |
| 10 | **No audit logging** | Plugin execution not logged to audit pipeline |

---

## 4. Gap Analysis

### Use Cases That Fail Today

| # | Use Case | Current | Expected |
|---|----------|---------|----------|
| 1 | "Add custom claim 'risk_score' to every JWT based on user's recent login history" | Cannot express | Post-login hook plugin computes risk_score, adds to JWT claims |
| 2 | "Block login if user's email domain matches a custom deny-list" | No hook | Pre-login hook plugin checks domain, returns block decision |
| 3 | "Transform SAML attributes: uppercase department, rename group→role" | No hook | JIT mapping hook plugin transforms attributes before provisioning |
| 4 | "Scan outgoing API responses for SSNs and redact them" | No hook | Response phase plugin with DLP scanning |
| 5 | "Custom policy: deny if user's manager is on vacation" | Can't express in ABAC | Policy evaluation hook plugin queries external HR system |
| 6 | "Tenant uploads their own compliance plugin" | No upload API | REST upload → DB storage → sandboxed execution per-tenant |
| 7 | "Limit plugin to 16MB RAM, 100ms execution time" | No limits | ResourceLimiter enforces memory + fuel cap |
| 8 | "Rollback plugin to previous version" | No versioning | Versioned plugin storage with instant rollback |

---

## 5. Proposed Architecture: Universal Plugin Engine

```
                    ┌──────────────────────────────────────────────┐
                    │         Universal Plugin Engine               │
                    │         (extends Gateway WASM Host)          │
                    │                                              │
                    │  ┌─────────────────────────────────────────┐ │
                    │  │  Plugin Registry (PostgreSQL)           │ │
                    │  │  - Plugin metadata (name, version, ...) │ │
                    │  │  - .wasm binary (bytea)                 │ │
                    │  │  - Per-tenant config                    │ │
                    │  │  - Hook subscriptions                   │ │
                    │  │  - Resource limits per plugin           │ │
                    │  └─────────────────────────────────────────┘ │
                    │                                              │
                    │  ┌─────────────────────────────────────────┐ │
                    │  │  Wazero Runtime Pool                    │ │
                    │  │  - Per-tenant isolated runtimes         │ │
                    │  │  - ResourceLimiter (memory, fuel)       │ │
                    │  │  - Compiled module cache                │ │
                    │  └─────────────────────────────────────────┘ │
                    │                                              │
                    │  ┌─────────────────────────────────────────┐ │
                    │  │  Hook Dispatcher                        │ │
                    │  │                                         │ │
                    │  │  Hook points (called at specific times):│ │
                    │  │  ├── auth.pre_login     (Gateway)       │ │
                    │  │  ├── auth.post_login    (Auth Svc)      │ │
                    │  │  ├── auth.pre_register  (Auth Svc)      │ │
                    │  │  ├── token.pre_issue    (OAuth Svc)     │ │
                    │  │  ├── policy.pre_check   (Policy Svc)    │ │
                    │  │  ├── policy.post_check  (Policy Svc)    │ │
                    │  │  ├── jit.pre_provision  (Identity Svc)  │ │
                    │  │  ├── gateway.pre_proxy  (existing ✅)   │ │
                    │  │  └── gateway.post_proxy (existing ✅)   │ │
                    │  └─────────────────────────────────────────┘ │
                    └──────────────────────────────────────────────┘
```

### Hook Execution Flow

```
HTTP Request arrives at Gateway
         │
         ▼
┌────────────────────────────────┐
│ 1. JWT Validation (existing)  │
└──────────────┬─────────────────┘
               │
         ┌─────▼─────┐
         │ HOOK:     │  Plugin can:
         │ gateway   │  - Block request (return 403)
         │ .pre_proxy│  - Modify headers (add X-Custom)
         │           │  - Transform body
         └─────┬─────┘
               │ (if not blocked)
         ┌─────▼─────┐
         │ HOOK:     │  Plugin can:
         │ policy    │  - Return custom allow/deny
         │ .pre_check│  - Add policy conditions
         │           │  - Query external system
         └─────┬─────┘
               │ (if allowed)
         ┌─────▼─────┐
         │ Proxy to  │
         │ backend   │
         └─────┬─────┘
               │
         ┌─────▼─────┐
         │ HOOK:     │  Plugin can:
         │ gateway   │  - Redact sensitive data
         │ .post_proxy│  - Log response
         │           │  - Modify response headers
         └─────┬─────┘
               │
         ▼
    Response to client
```

---

## 6. Plugin Hook Points

### Complete Hook Catalog

| Hook | Service | When | Plugin Can | Existing? |
|------|---------|------|-----------|-----------|
| `auth.pre_login` | Auth | Before credential check | Block login, inject context | **New** |
| `auth.post_login` | Auth | After successful auth, before token | Add claims, trigger MFA | **New** |
| `auth.pre_register` | Auth | Before user creation | Block registration, validate | **New** |
| `token.pre_issue` | OAuth | Before JWT signing | Add/modify claims, restrict scope | **New** |
| `token.post_issue` | OAuth | After JWT signed | Log, audit, push to external | **New** |
| `policy.pre_check` | Policy | Before RBAC/ABAC evaluation | Pre-deny, add context attributes | **New** |
| `policy.post_check` | Policy | After evaluation, before result | Override decision, log rationale | **New** |
| `jit.pre_provision` | Identity | Before JIT user creation | Transform attributes, block creation | **New** |
| `gateway.pre_proxy` | Gateway | Before proxying to backend | Block, modify headers/body | **Existing** ✅ |
| `gateway.post_proxy` | Gateway | After backend response | Redact, modify response | **Existing** ✅ |

### Hook Contract

Every hook follows the same contract:

```go
// HookInput is passed to the plugin at hook execution.
type HookInput struct {
    Hook       string                 `json:"hook"`        // "auth.post_login"
    TenantID   string                 `json:"tenant_id"`
    UserID     string                 `json:"user_id,omitempty"`
    Context    map[string]any         `json:"context"`     // Hook-specific data
    Request    *HTTPRequest           `json:"request,omitempty"`
}

// HookOutput is returned by the plugin.
type HookOutput struct {
    Action       HookAction           `json:"action"`        // "continue", "block", "modify"
    BlockReason  string               `json:"block_reason,omitempty"`
    ModifiedData map[string]any       `json:"modified_data,omitempty"`  // Overlays onto context
    AddedClaims  map[string]any       `json:"added_claims,omitempty"`   // For token hooks
    Error        string               `json:"error,omitempty"`
}

type HookAction string
const (
    ActionContinue HookAction = "continue"
    ActionBlock    HookAction = "block"
    ActionModify   HookAction = "modify"
)
```

---

## 7. Endpoint Precondition Check

### Existing Endpoints (Plugin Can Use Immediately)

| Endpoint | File:Line | Status |
|----------|-----------|--------|
| Gateway middleware | `wasm_plugin.go:260` | **WasmMiddleware** wired in gateway chain |
| Plugin load (file-based) | `wasm_plugin.go:105` | **LoadPlugin** from filesystem path |
| Plugin execute | `wasm_plugin.go` (~250) | **Execute** with PluginContext |
| Plugin unload | `wasm_plugin.go` | **UnloadPlugin** by name |
| Plugin list | `wasm_plugin.go` | **ListPlugins** returns metadata |

### New Endpoints Required

| Endpoint | Method | Purpose | Priority |
|----------|--------|---------|----------|
| `/api/v1/plugins` | POST | Upload .wasm binary | P0 |
| `/api/v1/plugins` | GET | List plugins per tenant | P0 |
| `/api/v1/plugins/{id}` | GET | Get plugin details | P0 |
| `/api/v1/plugins/{id}` | PUT | Update plugin config | P0 |
| `/api/v1/plugins/{id}` | DELETE | Remove plugin | P0 |
| `/api/v1/plugins/{id}/enable` | POST | Enable plugin | P0 |
| `/api/v1/plugins/{id}/disable` | POST | Disable plugin | P0 |
| `/api/v1/plugins/{id}/test` | POST | Dry-run plugin with sample input | P1 |
| `/api/v1/plugins/{id}/versions` | GET | List versions | P1 |
| `/api/v1/plugins/{id}/versions/{v}/activate` | POST | Rollback to version | P1 |
| `/api/v1/plugins/hooks` | GET | List available hook points | P0 |

---

## 8. Sandbox Security Model

### Wazero Sandbox Properties

| Property | Enforcement | Default |
|----------|------------|---------|
| **Memory limit** | `wazero.NewRuntimeConfig().WithCloseOnContextDone(true)` + ResourceLimiter | 16MB per plugin |
| **Execution timeout** | `context.WithTimeout` on Execute call | 100ms per hook |
| **No network access** | WASI only provides stdio; no socket imports | Default |
| **No filesystem access** | No `fs` host module imported; only `wasi_snapshot_preview1` | Default |
| **No host memory access** | WASM linear memory is isolated | By design |
| **Fuel/gas metering** | wazero's fuel API (`runtime.AddFuel`) | 10,000 fuel units |
| **Per-tenant isolation** | Separate runtime per tenant | Planned |

### ResourceLimiter Implementation

```go
// resourceLimitedRuntime creates a wazero runtime with resource limits.
func newTenantRuntime(tenantID uuid.UUID, limits PluginResourceLimits) wazero.Runtime {
    ctx := context.Background()
    config := wazero.NewRuntimeConfig().
        WithCloseOnContextDone(true).
        // wazero 1.x supports ResourceLimiter via host module configuration
        WithMemoryLimitPages(limits.MemoryPages())  // 16MB = 256 pages × 64KB
    
    runtime := wazero.NewRuntimeWithConfig(ctx, config)
    wasi_snapshot_preview1.MustInstantiate(ctx, runtime)
    return runtime
}
```

### Plugin Resource Limits Config

```yaml
# Per-tenant plugin resource limits
plugin_limits:
  max_memory_mb: 16              # Max linear memory per plugin instance
  max_execution_ms: 100          # Max wall-clock time per hook call
  max_fuel: 10000               # WASM fuel (instruction count)
  max_plugins_per_tenant: 20    # Max active plugins
  max_wasm_size_mb: 5           # Max .wasm binary size
```

---

## 9. Plugin SDK Design

### Minimal Plugin ABI (AssemblyScript Example)

```typescript
// claim-transform.as — AssemblyScript plugin for auth.post_login hook
import { HookInput, HookOutput } from "./ggid-plugin-sdk";

// Plugin must export "execute" function
export function execute(inputPtr: i32, inputLen: i32): i32 {
    const input = HostFunctions.readInput(inputPtr, inputLen);
    
    // Custom logic: compute risk score from login history
    const riskScore = computeRiskScore(input.context.login_count, input.context.failed_attempts);
    
    // Add custom claim
    const output = new HookOutput();
    output.action = Action.Modify;
    output.addedClaims = { "risk_score": riskScore.toString() };
    
    return HostFunctions.writeOutput(output);
}

function computeRiskScore(loginCount: i32, failedAttempts: i32): i32 {
    let score = 0;
    if (failedAttempts > 5) score += 40;
    if (loginCount < 3) score += 20;
    return score;
}
```

### Rust Plugin Example

```rust
// policy-plugin.rs — Custom policy evaluation plugin
use ggid_plugin_sdk::{HookInput, HookOutput, Action};

#[no_mangle]
pub extern "C" fn execute(input_ptr: u32, input_len: u32) -> u32 {
    let input: HookInput = ggid_plugin_sdk::read_input(input_ptr, input_len);
    
    // Custom: deny if user's department is "contractor" and resource is "admin-panel"
    if input.context.get("department") == Some(&"contractor") 
       && input.context.get("resource") == Some(&"admin-panel") 
    {
        return ggid_plugin_sdk::write_output(HookOutput {
            action: Action::Block,
            block_reason: "Contractors cannot access admin panel".to_string(),
            ..Default::default()
        });
    }
    
    ggid_plugin_sdk::write_output(HookOutput {
        action: Action::Continue,
        ..Default::default()
    })
}
```

### Host Functions Provided to Plugins

| Function | Purpose | Available |
|----------|---------|-----------|
| `readInput(ptr, len)` | Read HookInput from host memory | Yes |
| `writeOutput(output)` | Return HookOutput to host | Yes |
| `log(level, msg)` | Structured logging from plugin | Planned |
| `kvGet(key)` | Read from per-plugin KV store | Planned |
| `kvSet(key, val)` | Write to per-plugin KV store | Planned |
| `httpGet(url)` | HTTP GET (rate-limited, allow-listed) | P2 |

---

## 10. Database Schema

```sql
-- Plugin registry
CREATE TABLE plugins (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    name                VARCHAR(128) NOT NULL,
    description         TEXT,
    author              VARCHAR(256),
    
    -- WASM binary
    wasm_binary         BYTEA NOT NULL,
    wasm_hash           VARCHAR(64) NOT NULL,           -- SHA-256 of binary
    
    -- Metadata from plugin init
    plugin_version      VARCHAR(32),
    plugin_hooks        JSONB DEFAULT '[]',              -- ["auth.post_login", "gateway.pre_proxy"]
    
    -- Configuration
    config              JSONB DEFAULT '{}',              -- Plugin-specific config
    resource_limits     JSONB DEFAULT '{"max_memory_mb": 16, "max_execution_ms": 100}',
    
    -- State
    enabled             BOOLEAN DEFAULT false,
    
    -- Audit
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          UUID NOT NULL,
    
    UNIQUE(tenant_id, name)
);

-- Plugin versions (for rollback)
CREATE TABLE plugin_versions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plugin_id           UUID NOT NULL REFERENCES plugins(id) ON DELETE CASCADE,
    tenant_id           UUID NOT NULL,
    version             INT NOT NULL,
    wasm_binary         BYTEA NOT NULL,
    wasm_hash           VARCHAR(64) NOT NULL,
    config              JSONB DEFAULT '{}',
    activated_at        TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(plugin_id, version)
);

-- Plugin execution log (audit trail)
CREATE TABLE plugin_execution_log (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    plugin_id           UUID NOT NULL,
    plugin_name         VARCHAR(128) NOT NULL,
    hook                VARCHAR(64) NOT NULL,
    user_id             UUID,
    execution_ms        INT NOT NULL,                    -- Wall-clock execution time
    fuel_used           BIGINT,                          -- WASM fuel consumed
    result              VARCHAR(32) NOT NULL,            -- "continue", "block", "modify", "error"
    error_message       TEXT,
    request_path        VARCHAR(512),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Plugin KV store (per-plugin key-value storage)
CREATE TABLE plugin_kv_store (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    plugin_id           UUID NOT NULL,
    key                 VARCHAR(256) NOT NULL,
    value               JSONB,
    expires_at          TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(plugin_id, key)
);

-- Per-tenant plugin resource limits
CREATE TABLE plugin_tenant_limits (
    tenant_id           UUID PRIMARY KEY,
    max_memory_mb       INT DEFAULT 16,
    max_execution_ms    INT DEFAULT 100,
    max_fuel            BIGINT DEFAULT 10000,
    max_plugins         INT DEFAULT 20,
    max_wasm_size_mb    INT DEFAULT 5,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_plugins_tenant ON plugins (tenant_id, enabled);
CREATE INDEX idx_plugin_versions_plugin ON plugin_versions (plugin_id, version DESC);
CREATE INDEX idx_plugin_exec_tenant ON plugin_execution_log (tenant_id, created_at DESC);
CREATE INDEX idx_plugin_exec_plugin ON plugin_execution_log (plugin_id, created_at DESC);
CREATE INDEX idx_plugin_kv_plugin ON plugin_kv_store (plugin_id, key);
```

---

## 11. API Design + Curl Commands

### Plugin Management

```bash
# Upload a new plugin
curl -X POST https://ggid.corp.com/api/v1/plugins \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -F "name=claim-transform" \
  -F "description=Adds risk_score claim to JWTs" \
  -F "wasm=@/path/to/plugin.wasm" \
  -F "config={\"threshold\":50}"

# Response:
# {"id":"uuid","name":"claim-transform","version":1,"enabled":false,"hooks":["auth.post_login"]}

# List plugins
curl https://ggid.corp.com/api/v1/plugins \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"

# Enable plugin
curl -X POST https://ggid.corp.com/api/v1/plugins/{id}/enable \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"

# Disable plugin
curl -X POST https://ggid.corp.com/api/v1/plugins/{id}/disable \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"

# Test plugin with sample input (dry-run)
curl -X POST https://ggid.corp.com/api/v1/plugins/{id}/test \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "hook": "auth.post_login",
    "tenant_id": "...",
    "user_id": "...",
    "context": {"login_count": 42, "failed_attempts": 3}
  }'

# Response:
# {"action":"modify","added_claims":{"risk_score":"20"},"execution_ms":3}

# List available hook points
curl https://ggid.corp.com/api/v1/plugins/hooks \
  -H "Authorization: Bearer $TOKEN"

# Rollback to previous version
curl -X POST https://ggid.corp.com/api/v1/plugins/{id}/versions/1/activate \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"

# Delete plugin
curl -X DELETE https://ggid.corp.com/api/v1/plugins/{id} \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"
```

### Error Responses

```json
// 400 Bad Request — Invalid WASM binary
{
  "error": { "code": "INVALID_WASM", "message": "WASM binary missing required 'execute' export" }
}

// 413 Payload Too Large — WASM exceeds size limit
{
  "error": { "code": "WASM_TOO_LARGE", "message": "WASM binary (8MB) exceeds limit (5MB)" }
}

// 429 Too Many Plugins
{
  "error": { "code": "PLUGIN_LIMIT_EXCEEDED", "message": "Tenant has 20/20 plugins" }
}
```

---

## 12. Performance Considerations

| Operation | Latency | Notes |
|-----------|---------|-------|
| Plugin compile (first load) | 5-20ms | wazero compiles WASM to native code |
| Plugin compile (cached) | 0ms | Compiled module reused |
| Plugin instantiate | 0.5-2ms | New module instance per request |
| Plugin execute (simple logic) | 0.1-1ms | Near-native speed |
| Plugin execute (complex computation) | 1-10ms | Within 100ms limit |
| Hook dispatch overhead | <0.5ms | Registry lookup + instantiation |
| **Total per-hook overhead** | **1-5ms** | Acceptable for per-request execution |

### Optimization Strategies

1. **Compiled module cache**: wazero CompiledModule reused; only instantiate per request
2. **Pre-warmed instances**: Pool of pre-instantiated modules for hot plugins
3. **Batch hook execution**: If multiple plugins subscribe to same hook, execute sequentially but in one pass
4. **Skip disabled hooks**: Registry checks enabled flag before dispatching

---

## 13. Console UI Design

### Plugin Management Dashboard

```
┌──────────────────────────────────────────────────────────────────┐
│  Plugins                                                        │
│                                                                  │
│  ┌────────────────┐  ┌────────────────┐  ┌────────────────┐     │
│  │  Active        │  │  Hooks         │  │  Executions    │     │
│  │  Plugins: 5    │  │  Available: 10 │  │  Today: 12.4K  │     │
│  └────────────────┘  └────────────────┘  └────────────────┘     │
│                                                                  │
│  Installed Plugins                                               │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │ ● claim-transform  v2  auth.post_login      Active   [⚙️] │  │
│  │   Adds risk_score claim. 3ms avg. 12K execs today         │  │
│  ├────────────────────────────────────────────────────────────┤  │
│  │ ● dlp-scanner      v1  gateway.post_proxy   Active   [⚙️] │  │
│  │   Scans responses for SSN/credit cards. 8ms avg.          │  │
│  ├────────────────────────────────────────────────────────────┤  │
│  │ ● custom-policy    v3  policy.pre_check     Disabled [⚙️] │  │
│  │   External HR-based policy. 15ms avg.                      │  │
│  └────────────────────────────────────────────────────────────┘  │
│                                                                  │
│  + Upload Plugin (.wasm)                                         │
│                                                                  │
│  Available Hooks                                                 │
│  auth.pre_login | auth.post_login | auth.pre_register            │
│  token.pre_issue | policy.pre_check | jit.pre_provision          │
│  gateway.pre_proxy ✅ | gateway.post_proxy ✅                    │
└──────────────────────────────────────────────────────────────────┘
```

---

## 14. Competitive Differentiation

| Feature | GGID (target) | Auth0 Actions | OPA | Envoy WASM | Keycloak SPI |
|---------|---------------|---------------|-----|-----------|-------------|
| **Runtime** | **WASM (wazero)** | V8 JavaScript | Go (compiled) | WASM (V8/wazero) | Java (.jar) |
| **Sandbox** | **WASM isolation** | V8 isolate | Process-level | WASM isolation | JVM classloader |
| **Language** | **Any WASM** | JavaScript only | Rego | Any WASM | Java only |
| **Auth pipeline hooks** | **10 hooks** | 5 triggers | Proxy-level | Proxy-level | SPI interfaces |
| **Claim transformation** | **Yes (WASM)** | Yes (JS) | No | No | Yes (Java) |
| **Policy evaluation** | **Yes (WASM)** | Via action | Native (Rego) | Via filter | Via SPI |
| **Resource limits** | **Memory + fuel** | Timeout only | None | Memory + fuel | JVM limits |
| **Per-tenant** | **Yes** | Yes (stores) | Namespace | Route-level | Realm |
| **Hot reload** | **Yes (API)** | Yes | Yes | Config reload | No (restart) |
| **Open source** | **Yes (Apache 2.0)** | No | Yes | Yes | Yes |

**Key differentiator**: GGID would be the only open-source IAM with WASM plugins across the **entire pipeline** (auth + token + policy + JIT + gateway), not just gateway-level filtering.

---

## 15. Migration Strategy

### Phase 1: Gateway-Only (Already Works)

Existing `gateway.pre_proxy` and `gateway.post_proxy` hooks remain unchanged. Plugins already runnable via `LoadPlugin` from filesystem.

### Phase 2: DB-Backed Plugin Registry

1. Deploy plugin registry DB tables
2. Migrate file-based plugins to DB storage
3. Add management API for CRUD
4. Add per-tenant isolation
5. Add resource limits

### Phase 3: Auth/Token/Policy Hooks

1. Add hook dispatcher to auth, oauth, and policy services
2. Each service calls HookDispatcher before/after key operations
3. Plugins registered for specific hooks execute in that service's WASM runtime
4. Results (block/modify/continue) integrated into service logic

### Phase 4: Plugin SDK + Marketplace

1. Publish Plugin SDK (Go/Rust/AssemblyScript bindings)
2. Documentation and examples
3. Community plugin marketplace
4. Console plugin editor (online WASM compilation)

---

## 16. Security Considerations

| Risk | Mitigation |
|------|-----------|
| **Malicious plugin** | WASM sandbox prevents host memory access; ResourceLimiter caps memory |
| **CPU exhaustion** | Fuel metering + execution timeout (100ms default) |
| **Plugin privilege escalation** | Per-tenant isolation; plugins can't access other tenants' data |
| **WASM escape** | wazero is a pure-Go interpreter (no native code execution); no known WASM escape CVEs in wazero |
| **Supply chain (malicious .wasm)** | SHA-256 hash verification; optional signing; admin approval for first activation |
| **Network exfiltration** | No network imports by default; HTTP host function is allow-listed and rate-limited |
| **Sensitive data in plugin logs** | Plugin log output sanitized; PII fields obfuscated before logging |

---

## 17. Implementation Backlog with DoD

### P0 — Core Plugin Engine Extension (3 sprints)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 1 | Plugin DB schema | ✅ CREATE TABLE in migration file ✅ go build PASS ✅ No log.Printf/内存 map | 2d |
| 2 | Plugin repository | ✅ CRUD backed by pgx ✅ `if err != nil` guards ✅ ≥3 tests | 3d |
| 3 | Plugin management API | ✅ Endpoints registered in server.go ✅ From main.go → handler → repo chain works ✅ curl test PASS | 3d |
| 4 | Per-tenant runtime isolation | ✅ Separate wazero runtime per tenant ✅ ≥3 tests | 3d |
| 5 | Resource limits (memory + fuel) | ✅ ResourceLimiter configured ✅ Plugin killed on limit exceed ✅ Test verifies OOM handling | 3d |
| 6 | Hook dispatcher | ✅ Dispatcher in gateway + auth + oauth ✅ Hooks called at correct lifecycle point ✅ ≥3 tests | 4d |

### P1 — Auth/Token/Policy Hooks (2 sprints)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 7 | auth.post_login hook | ✅ Hook called after successful auth ✅ Added claims appear in JWT ✅ Test verifies claim injection | 3d |
| 8 | token.pre_issue hook | ✅ Hook called before JWT signing ✅ Modified claims in output JWT ✅ Test PASS | 2d |
| 9 | policy.pre_check hook | ✅ Hook called before RBAC evaluation ✅ Plugin block overrides RBAC allow ✅ Test PASS | 3d |
| 10 | jit.pre_provision hook | ✅ Hook called before JIT user creation ✅ Attribute transform works ✅ Test PASS | 2d |
| 11 | Plugin dry-run/test API | ✅ POST /plugins/{id}/test returns execution result ✅ No side effects ✅ Test PASS | 2d |

### P2 — Console UI + Plugin SDK (2 sprints)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 12 | Plugin dashboard | ✅ List/enable/disable works ✅ Upload via form ✅ Shows execution stats | 3d |
| 13 | Plugin upload wizard | ✅ .wasm upload + metadata form ✅ Hook selector ✅ Test button | 3d |
| 14 | Plugin SDK (Go) | ✅ ggid-plugin-sdk Go module ✅ Example plugins ✅ README | 3d |
| 15 | Plugin SDK (Rust) | ✅ Cargo crate ✅ Example plugins ✅ README | 3d |

### P3 — Advanced Features (Future)

| # | Task | DoD |
|---|------|-----|
| 16 | Plugin KV store | `kvGet`/`kvSet` host functions with TTL |
| 17 | Plugin HTTP allow-list | Rate-limited HTTP access to allow-listed URLs |
| 18 | Plugin marketplace | Community plugin repository |
| 19 | Online plugin editor | WASM compilation in browser (no local toolchain) |
| 20 | Plugin metrics dashboard | Per-plugin latency, fuel usage, block rate |
| 21 | Plugin signing | Cryptographic signing of .wasm binaries |
| 22 | Cross-tenant plugin sharing | Share plugins across tenants with explicit grants |

---

## References

- [wazero: Go WebAssembly Runtime](https://github.com/tetratelabs/wazero) — Pure-Go WASM runtime used by GGID
- [WASI Preview 1](https://github.com/WebAssembly/WASI/blob/main/legacy/preview1/docs.md) — WebAssembly System Interface
- [Envoy WASM Extensions](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/wasm_filter) — Proxy-level WASM plugins
- [Istio WasmPlugin](https://istio.io/latest/docs/reference/config/proxy_extensions/wasmplugin/) — Service mesh WASM extension
- [Cloudflare Workers](https://developers.cloudflare.com/workers/) — Edge WASM execution model
- [Auth0 Actions](https://auth0.com/docs/customize/actions) — JavaScript-based auth pipeline extensibility
- [OPA (Open Policy Agent)](https://www.openpolicyagent.org/) — Rego-based policy engine
- [Wasmtime Security](https://docs.wasmtime.dev/security.html) — WASM sandbox security model
- [GGID WasmPluginHost](../services/gateway/internal/middleware/wasm_plugin.go) — Existing implementation at line 38
- [GGID Plugin Tests](../services/gateway/internal/middleware/wasm_plugin_test.go) — 10+ existing test cases
- [Resource Isolation Attack Surface (arxiv)](https://arxiv.org/html/2509.11242v1) — WASM resource exhaustion research
