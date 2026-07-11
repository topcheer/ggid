# GGID Go SDK Reference Guide

This guide covers the GGID Go SDK — 11 packages with 50+ methods for authentication, user management, authorization, organization management, audit, and AI agent identity.

> **Related**: [Quickstart Go SDK](../quickstart/go-sdk.md)

## Installation

```bash
go get github.com/ggid/ggid/sdk/go/ggid@latest
```

## Packages

| Package | File | Purpose |
|---------|------|---------|
| `ggid` | `client.go` | Client initialization and configuration |
| `ggid` | `api.go` | Auth API (login, register, verify) |
| `ggid` | `identity.go` | User CRUD, lock/unlock, search |
| `ggid` | `policy.go` | Roles, permissions, policy evaluation |
| `ggid` | `org.go` | Organizations, departments, members |
| `ggid` | `audit.go` | Audit events, compliance, alerts, retention |
| `ggid` | `agent.go` | AI agent registration, token exchange |
| `ggid` | `jwt.go` | JWT verification via JWKS |
| `ggid` | `middleware.go` | HTTP middleware (auth, permission) |
| `ggid` | `token_manager.go` | Auto-refreshing token management |
| `ggid` | `errors.go` | SDK error types |

## Client Initialization

```go
import "github.com/ggid/ggid/sdk/go/ggid"

client := ggid.NewClient("https://api.ggid.example.com",
    ggid.WithHTTPClient(customHTTPClient),  // Optional
)
```

## Authentication

### Login

```go
tokens, err := client.Login(ctx, "user@example.com", "password")
// tokens.AccessToken  → JWT for API calls
// tokens.RefreshToken → Refresh after expiry
// tokens.ExpiresIn    → Seconds until expiry
```

### Register

```go
userID, err := client.Register(ctx, "newuser", "user@example.com", "SecurePass1!", "Display Name")
```

### Refresh Token

```go
tokens, err := client.Refresh(ctx, oldTokens.RefreshToken)
```

### Verify Token

```go
claims, err := client.VerifyToken(ctx, accessToken)
// claims["sub"], claims["tenant_id"], claims["scope"], claims["exp"]
```

## User Management

### Create User

```go
user, err := client.CreateUser(ctx, adminToken, &ggid.CreateUserRequest{
    Username: "alice",
    Email:    "alice@example.com",
    Password: "SecurePass1!",
    Name:     "Alice Chen",
})
```

### Get / Search / Lock / Unlock

```go
user, err := client.GetUser(ctx, token, "user-uuid")
users, err := client.SearchUsers(ctx, token, "alice")
err := client.LockUser(ctx, token, "user-uuid")
err := client.UnlockUser(ctx, token, "user-uuid")
users, err := client.ListUsers(ctx, token)
err := client.DeleteUser(ctx, token, "user-uuid")
```

### Update User

```go
user, err := client.UpdateUser(ctx, token, "user-uuid", &ggid.UpdateUserRequest{
    Email: "newemail@example.com",
    Phone: "+1234567890",
})
```

## Role & Permission Management

### Roles

```go
role, err := client.CreateRole(ctx, token, "developer", "dev", "Developer role")
role, err := client.GetRole(ctx, token, "role-uuid")
roles, err := client.ListRoles(ctx, token)
err := client.DeleteRole(ctx, token, "role-uuid")
```

### Assignment

```go
err := client.AssignRole(ctx, token, "user-uuid", "role-uuid")
err := client.RevokeRole(ctx, token, "user-uuid", "role-uuid")
roles, err := client.GetUserRoles(ctx, token, "user-uuid")
```

### Policy Evaluation

```go
result, err := client.CheckPermission(ctx, token, "document:report", "read")
// result.Allowed → true/false

result, err := client.CheckPolicy(ctx, token, &ggid.PolicyCheckRequest{
    UserID:   "user-uuid",
    Resource: "api:read",
    Action:   "execute",
    Context:  map[string]string{"ip": "192.168.1.1"},
})
```

### Permissions

```go
perms, err := client.ListPermissions(ctx, token)
```

## Organization Management

```go
org, err := client.CreateOrganization(ctx, token, "Engineering", "Engineering team")
org, err := client.GetOrganization(ctx, token, "org-uuid")
orgs, err := client.ListOrganizations(ctx, token)
err := client.DeleteOrganization(ctx, token, "org-uuid")

dept, err := client.CreateDepartment(ctx, token, "org-uuid", "Backend", "Backend team")
depts, err := client.ListDepartments(ctx, token, "org-uuid")

err := client.AddMember(ctx, token, "org-uuid", "user-uuid", "developer")
err := client.RemoveMember(ctx, token, "org-uuid", "user-uuid")
```

## Audit & Compliance

### Query Audit Events

```go
events, err := client.ListAuditEvents(ctx, token, ggid.AuditEventFilter{
    EventType:  "user.login",
    UserID:     "user-uuid",
    StartDate:  "2025-01-01",
    EndDate:    "2025-01-31",
})
```

### Compliance Reports

```go
report, err := client.GetComplianceReport(ctx, token, "soc2", "2025-01-01", "2025-03-31")
// report.Type, report.Summary, report.Sections
```

### Alert Rules

```go
rules, err := client.GetAlertRules(ctx, token)
err := client.UpsertAlertRule(ctx, token, ggid.AlertRule{
    Name:      "Failed login burst",
    Condition: "failed_logins > 10 in 5m",
    Action:    "email",
})
err := client.TestAlert(ctx, token)
```

### Retention

```go
policy, err := client.GetRetentionPolicy(ctx, token)
err := client.UpdateRetentionPolicy(ctx, token, ggid.RetentionPolicy{
    MaxAgeDays: 365,
    MaxCount:   1000000,
})
```

### Hash Chain Verification

```go
valid, err := client.VerifyAuditIntegrity(ctx, token)
// valid → true if hash chain is intact
```

### Export

```go
data, err := client.ExportAuditEvents(ctx, token, "csv")  // or "json"
```

## AI Agent Identity

```go
agent, err := client.RegisterAgent(ctx, &ggid.AgentRegistration{
    Name:          "my-agent",
    Type:          "service",
    Scopes:        []string{"users:read"},
    MaxDelegationDepth: 3,
}, adminToken)

agents, err := client.ListAgents(ctx, adminToken)

tokenResp, err := client.ExchangeAgentToken(ctx, agent.ID, subjectToken, []string{"users:read"})

claims, err := client.VerifyAgentToken(ctx, agentToken)
// claims.AgentID, claims.DelegationChain, claims.MCPServers
```

## JWT Verification

```go
verifier := ggid.NewJWTVerifier("https://api.ggid.example.com/.well-known/jwks.json")
claims, err := verifier.Verify(ctx, accessToken)
```

## Token Manager (Auto-Refresh)

```go
tm := ggid.NewTokenManager(client)
tokens, err := tm.Login(ctx, "user@example.com", "password")
tm.SetTokens(tokens)

// Auto-refreshes if expired
accessToken, err := tm.AccessToken(ctx)
```

## HTTP Middleware

### Auth Middleware

```go
mux := http.NewServeMux()
mux.Handle("/api/protected", client.Middleware(protectedHandler))
// Verifies JWT on every request, populates request context
```

### Permission Middleware

```go
mux.Handle("/api/admin",
    client.Middleware(
        client.RequirePermission("admin", "access")(adminHandler),
    ),
)
```

## Access Requests (IGA)

```go
req, err := client.SubmitAccessRequest(ctx, token, ggid.AccessRequest{
    UserID:      "user-uuid",
    ResourceType: "role",
    ResourceID:   "role-uuid",
    Reason:      "Need access for project",
})

requests, err := client.ListAccessRequests(ctx, token, "pending")
err := client.ApproveAccessRequest(ctx, token, "request-uuid")
err := client.DenyAccessRequest(ctx, token, "request-uuid")
```

## Branding

```go
branding, err := client.GetBranding(ctx, token, "tenant-uuid")
err := client.UpdateBranding(ctx, token, "tenant-uuid", ggid.BrandingConfig{
    LogoURL:   "https://example.com/logo.png",
    PrimaryColor: "#0066CC",
})
```

## Complete Example

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/ggid/ggid/sdk/go/ggid"
)

func main() {
    ctx := context.Background()
    client := ggid.NewClient("https://api.ggid.example.com")

    // Login
    tokens, err := client.Login(ctx, "admin@example.com", "password")
    if err != nil { log.Fatal(err) }

    // Create user
    user, err := client.CreateUser(ctx, tokens.AccessToken, &ggid.CreateUserRequest{
        Username: "alice",
        Email:    "alice@example.com",
        Password: "SecurePass1!",
    })
    if err != nil { log.Fatal(err) }

    // Create and assign role
    role, err := client.CreateRole(ctx, tokens.AccessToken, "dev", "developer", "Developer")
    if err != nil { log.Fatal(err) }
    client.AssignRole(ctx, tokens.AccessToken, user.ID, role.ID)

    // Check permission
    result, _ := client.CheckPermission(ctx, tokens.AccessToken, "api:read", "execute")
    fmt.Printf("Permission: %v\n", result.Allowed)

    // Cleanup
    client.DeleteUser(ctx, tokens.AccessToken, user.ID)
}
```

## See Also

- [Java SDK Guide](java-sdk-guide.md)
- [Node.js SDK Guide](node-sdk-guide.md)
- [API Reference](api-reference.md)
- [Authentication Flows](authentication-flows.md)
