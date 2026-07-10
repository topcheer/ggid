# GGID Plugin System Design

> Extension points for custom authentication, authorization, and user lifecycle logic.

---

## Overview

GGID supports plugins via three mechanisms:

1. **Webhook Hooks** — HTTP callbacks at lifecycle events (simplest, language-agnostic)
2. **Go Plugin Interface** — Native Go plugins for in-process execution (highest performance)
3. **gRPC Extensions** — Sidecar plugins for polyglot extensibility

---

## 1. Webhook Hooks

### Events

| Event | Trigger | Use Case |
|-------|---------|----------|
| `pre_register` | Before user registration | Invite-code validation, domain restriction |
| `post_register` | After user registration | Provision external resources, send welcome email |
| `pre_login` | Before credential check | IP reputation check, device fingerprinting |
| `post_login` | After successful login | Session enrichment, risk scoring |
| `pre_token_issue` | Before JWT issuance | Custom claims injection |
| `post_logout` | After user logout | Session cleanup notification |

### Configuration

```json
{
  "hooks": {
    "pre_login": {
      "url": "https://your-app.com/hooks/ggid-pre-login",
      "method": "POST",
      "headers": {
        "Authorization": "Bearer HOOK_SECRET",
        "X-Hook-Signature": "HMAC-SHA256"
      },
      "timeout": "5s",
      "on_error": "deny"
    },
    "post_login": {
      "url": "https://your-app.com/hooks/ggid-post-login",
      "method": "POST",
      "timeout": "3s",
      "on_error": "allow"
    }
  }
}
```

### Request Payload

```json
{
  "event": "pre_login",
  "timestamp": "2024-01-01T00:00:00Z",
  "tenant_id": "00000000-0000-0000-0000-000000000001",
  "request_id": "req-abc123",
  "user": {
    "id": "uuid",
    "username": "john",
    "email": "john@example.com",
    "roles": ["admin"],
    "metadata": {}
  },
  "context": {
    "ip": "1.2.3.4",
    "user_agent": "Mozilla/5.0...",
    "device_id": "fingerprint-hash"
  }
}
```

### Expected Response

```json
{
  "action": "allow",
  "modify": {
    "claims": {
      "custom_role": "super_admin"
    },
    "metadata": {
      "risk_score": 0.1
    }
  }
}
```

Or deny:

```json
{
  "action": "deny",
  "reason": "IP not in allowlist"
}
```

---

## 2. Go Plugin Interface

For high-performance in-process extensions:

```go
package plugins

// AuthPlugin extends authentication behavior.
type AuthPlugin interface {
    Name() string
    OnPreLogin(ctx context.Context, req *LoginRequest) (*LoginDecision, error)
    OnPostLogin(ctx context.Context, user *User) error
}

// PolicyPlugin extends policy evaluation.
type PolicyPlugin interface {
    Name() string
    Evaluate(ctx context.Context, req *PolicyRequest) (*PolicyDecision, error)
}
```

Registration in `main.go`:

```go
import (
    "github.com/ggid/ggid/pkg/plugins"
    "github.com/example/myplugin"
)

func main() {
    registry := plugins.NewRegistry()
    registry.Register(myplugin.NewCustomAuth())
    // ...
}
```

---

## 3. gRPC Extension Sidecar

For polyglot plugins (Python, Java, Node.js):

```
GGID Service ←──gRPC──→ Plugin Sidecar ←──in-process──→ Custom Logic
```

### Proto Definition

```protobuf
service AuthExtension {
  rpc PreLogin(PreLoginRequest) returns (PreLoginResponse);
  rpc PostLogin(PostLoginRequest) returns (PostLoginResponse);
  rpc PreRegister(PreRegisterRequest) returns (PreRegisterResponse);
}

message PreLoginRequest {
  string tenant_id = 1;
  string username = 2;
  string ip = 3;
  string user_agent = 4;
}

message PreLoginResponse {
  bool allow = 1;
  string deny_reason = 2;
  double risk_score = 3;
}
```

---

## Use Cases

### Custom Claims

Inject tenant-specific JWT claims:

```
Hook: pre_token_issue
Response: { "action": "allow", "modify": { "claims": { "department": "eng", "clearance": "L4" } } }
```

### Risk-Based Authentication

External risk engine evaluates login:

```
Hook: post_login → calls risk API → returns risk_score
If risk_score > 0.7 → force MFA step-up
```

### Provisioning Integration

Auto-provision on registration:

```
Hook: post_register → calls provisioning API → creates Slack account + Jira account
```

### Compliance Logging

Log to external SIEM:

```
Hook: post_login → forwards to Splunk/Datadog
```
