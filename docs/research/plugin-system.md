# Plugin System Architecture

## Status: PROPOSED (P2)

## Problem

GGID's authentication providers (LDAP, OAuth, SAML) are compiled into the binary. Users want to extend GGID with custom:
- Authentication providers (e.g., custom SSO, legacy systems)
- Audit sinks (e.g., Splunk, Datadog, custom SIEM)
- Policy evaluators (e.g., OPA, custom ABAC)
- Notification channels (e.g., SMS, push, custom webhook)

## Proposed Architecture

### Go Plugin Approach (CGo)

Use Go's `plugin` package for native extensions:

```
┌──────────────────────────────────────────┐
│              GGID Process                 │
│  ┌─────────────┐  ┌──────────────────┐  │
│  │ Plugin      │  │ Extension Points │  │
│  │ Manager     │──│ - AuthProvider   │  │
│  │             │  │ - AuditSink      │  │
│  │ .so files   │  │ - PolicyEngine   │  │
│  │             │  │ - Notifier       │  │
│  └─────────────┘  └──────────────────┘  │
└──────────────────────────────────────────┘
```

**Pros**: Native speed, direct memory access
**Cons**: Linux/macOS only, Go version must match exactly, security risk

### WASM Plugin Approach (RECOMMENDED)

Use WebAssembly for sandboxed extensions:

```
┌──────────────────────────────────────────┐
│              GGID Process                 │
│  ┌─────────────┐  ┌──────────────────┐  │
│  │ wazero      │  │ Plugin Host API   │  │
│  │ Runtime     │  │ - Auth()          │  │
│  │ (WASM)      │  │ - Audit()         │  │
│  │             │  │ - Evaluate()      │  │
│  │ .wasm files │  │ - Notify()        │  │
│  └─────────────┘  └──────────────────┘  │
└──────────────────────────────────────────┘
```

**Pros**: Sandboxed, cross-platform, language-agnostic, hot-reloadable
**Cons**: ~10-20% perf overhead, limited system access

### gRPC Sidecar Approach (SIMPLEST)

External plugins run as separate processes, communicating via gRPC:

```
┌─────────────────┐     gRPC     ┌─────────────────┐
│  GGID Gateway   │◄────────────►│  Custom Plugin  │
│                 │              │  (any language) │
│  PluginClient   │              │  PluginServer   │
└─────────────────┘              └─────────────────┘
```

**Pros**: Language-agnostic, fully isolated, independent scaling
**Cons**: Network latency, deployment complexity

## Recommendation: WASM + gRPC Hybrid

- **WASM**: For inline extensions (auth providers, policy hooks) where latency matters
- **gRPC**: For async extensions (audit sinks, notification channels) where latency is acceptable

## Plugin Interface (Go)

```go
// Plugin defines the contract for all extensions
type Plugin interface {
    Name() string
    Version() string
    Init(config map[string]any) error
}

// AuthProviderPlugin extends authentication
type AuthProviderPlugin interface {
    Plugin
    Authenticate(ctx context.Context, creds Credentials) (*User, error)
}

// AuditSinkPlugin extends audit logging
type AuditSinkPlugin interface {
    Plugin
    Publish(ctx context.Context, event AuditEvent) error
}
```

## Plugin Discovery

```
/etc/ggid/plugins/
  ├── auth-custom.so          # Native
  ├── audit-splunk.wasm       # WASM
  └── notifier-slack/         # gRPC sidecar
      └── plugin.toml         # Config
```

## Configuration

```yaml
plugins:
  enabled: true
  directories:
    - /etc/ggid/plugins
  wasm:
    max_memory: 64MB
    timeout: 5s
  grpc:
    discovery: dns
    namespace: ggid-plugins.svc.cluster.local
```

## Security

- WASM plugins run in capability-based sandbox (wazero)
- gRPC plugins use mTLS
- Plugins cannot access database directly
- All plugin calls logged for audit
- Plugin signing: verify Ed25519 signature before loading
