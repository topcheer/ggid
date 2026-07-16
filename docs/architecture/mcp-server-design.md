# MCP Server Design — LLM-Managed Identity & Access

> Status: DESIGN (2026-07-16 Round 76)
> Related: `docs/research/ai-agent-identity-mcp.md`, `docs/research/mcp-cli-console-oauth-gaps.md`

## 1. Overview

An MCP (Model Context Protocol) server that exposes GGID management operations as tools, enabling LLMs (Claude, GPT, custom agents) to programmatically manage users, roles, policies, and audit events via the Model Context Protocol.

### Architecture

```
┌──────────────┐     MCP (stdio/SSE)     ┌──────────────────┐
│  LLM Agent   │◀────────────────────────│  GGID MCP Server │
│ (Claude/GPT) │   tool calls + results   │  services/mcp/   │
└──────────────┘                          └────────┬─────────┘
                                                   │ HTTP (JWT)
                                                   ▼
                                          ┌──────────────────┐
                                          │  GGID Gateway    │
                                          │  :8080           │
                                          └────────┬─────────┘
                                                   │ proxy
                                    ┌──────────────┼──────────────┐
                                    ▼              ▼              ▼
                              ┌──────────┐  ┌──────────┐  ┌──────────┐
                              │ Identity │  │  Policy  │  │  Audit   │
                              │  :8081   │  │  :8070   │  │  :8072   │
                              └──────────┘  └──────────┘  └──────────┘
```

## 2. Authentication

### OAuth 2.1 PKCE Flow for MCP Clients

MCP clients authenticate via GGID's existing OAuth 2.1 authorization code flow with PKCE:

1. **Client registration**: MCP client registers as GGID OAuth client (`POST /api/v1/oauth/register`)
2. **Authorization**: User approves MCP client via `GET /api/v1/oauth/authorize`
3. **Token exchange**: Client exchanges auth code for JWT access token (`POST /api/v1/oauth/token`)
4. **API calls**: MCP server includes `Authorization: Bearer <JWT>` on all Gateway calls
5. **Token refresh**: MCP server handles refresh token rotation automatically

### Required Scopes

| Scope | Operations |
|-------|-----------|
| `users:read` | List users, get user details |
| `users:write` | Create, update, lock/unlock, delete users |
| `roles:read` | List roles, get role details |
| `roles:manage` | Create roles, assign/revoke roles |
| `policies:read` | List policies, check permissions |
| `policies:write` | Create, update policies |
| `audit:read` | Query audit events, compliance reports |
| `org:read` | List org units, departments |
| `org:write` | Create org units |
| `admin` | All operations (bootstrap admin only) |

## 3. MCP Tools

### 3.1 User Management Tools

| Tool | Scopes | Backend API |
|------|--------|-------------|
| `list_users` | `users:read` | `GET /api/v1/users?page=X&limit=Y` |
| `get_user` | `users:read` | `GET /api/v1/users/{id}` |
| `create_user` | `users:write` | `POST /api/v1/users` |
| `update_user` | `users:write` | `PUT /api/v1/users/{id}` |
| `lock_user` | `users:write` | `POST /api/v1/users/{id}/lock` |
| `unlock_user` | `users:write` | `POST /api/v1/users/{id}/unlock` |
| `delete_user` | `users:write` | `DELETE /api/v1/users/{id}` |
| `assign_role` | `roles:manage` | `POST /api/v1/users/{id}/roles` |
| `list_user_roles` | `roles:read` | `GET /api/v1/users/{id}/roles` |

### 3.2 Role & Policy Tools

| Tool | Scopes | Backend API |
|------|--------|-------------|
| `list_roles` | `roles:read` | `GET /api/v1/roles` |
| `create_role` | `roles:manage` | `POST /api/v1/roles` |
| `check_permission` | `policies:read` | `POST /api/v1/policy/check` |
| `list_policies` | `policies:read` | `GET /api/v1/policies` |
| `create_policy` | `policies:write` | `POST /api/v1/policies` |

### 3.3 Audit & Compliance Tools

| Tool | Scopes | Backend API |
|------|--------|-------------|
| `query_audit_events` | `audit:read` | `GET /api/v1/audit?limit=X` |
| `get_dashboard_stats` | `audit:read` | `GET /api/v1/identity/dashboard/stats` |

### 3.4 Organization Tools

| Tool | Scopes | Backend API |
|------|--------|-------------|
| `list_org_units` | `org:read` | `GET /api/v1/org/units` |
| `create_org_unit` | `org:write` | `POST /api/v1/org/units` |

## 4. Implementation Plan

### 4.1 Directory Structure

```
services/mcp/
├── cmd/
│   └── main.go              # MCP server entry point
├── internal/
│   ├── config/
│   │   └── config.go        # Server config (gateway URL, port, auth)
│   ├── client/
│   │   └── ggid_client.go   # HTTP client wrapping Gateway API calls
│   ├── tools/
│   │   ├── users.go          # User management tool implementations
│   │   ├── roles.go          # Role management tools
│   │   ├── policies.go       # Policy & permission tools
│   │   ├── audit.go          # Audit query tools
│   │   ├── org.go            # Organization tools
│   │   └── registry.go       # Tool registry + JSON schema definitions
│   └── server/
│       └── mcp_server.go     # MCP protocol handler (JSON-RPC over stdio/SSE)
├── go.mod                    # Module: github.com/ggid/ggid/services/mcp
└── README.md
```

### 4.2 MCP Protocol Implementation

The MCP server implements the Model Context Protocol specification:

```
Transport: stdio (for local) or SSE (for remote)
Protocol: JSON-RPC 2.0

Capabilities:
  - tools/list    → returns all registered tools with JSON schemas
  - tools/call    → executes a tool with arguments, returns result
  - resources/list → returns available GGID resources
```

### 4.3 Tool Definition Example

```go
type Tool struct {
    Name        string         `json:"name"`
    Description string         `json:"description"`
    InputSchema map[string]any `json:"inputSchema"`
    Handler     ToolHandler    `json:"-"`
}

// Example: create_user tool
var CreateUserTool = Tool{
    Name:        "create_user",
    Description: "Create a new user in GGID",
    InputSchema: map[string]any{
        "type": "object",
        "properties": map[string]any{
            "username":  map[string]any{"type": "string", "description": "Username"},
            "email":     map[string]any{"type": "string", "description": "Email address"},
            "password":  map[string]any{"type": "string", "description": "Initial password"},
            "display_name": map[string]any{"type": "string", "description": "Display name"},
        },
        "required": []string{"username", "email", "password"},
    },
    Handler: func(ctx context.Context, args map[string]any) (any, error) {
        // Call POST /api/v1/users via Gateway
    },
}
```

### 4.4 Gateway Integration

The MCP server acts as a regular GGID API client:

1. On startup, reads `GGID_GATEWAY_URL` (default `http://localhost:8080`)
2. Authenticates using OAuth 2.1 PKCE or pre-configured service token
3. All tool calls proxy through the Gateway → backend services
4. JWT token scope determines which tools are available

### 4.5 Dynamic Tool Filtering

The MCP server introspects the JWT scope claims and only exposes tools the agent is authorized to use:

```go
func (s *Server) ListTools() []Tool {
    scopes := s.getTokenScopes()
    var available []Tool
    for _, tool := range s.registry.All() {
        if hasRequiredScopes(scopes, tool.RequiredScopes) {
            available = append(available, tool)
        }
    }
    return available
}
```

## 5. Security Considerations

- **Token storage**: MCP server caches JWT in memory only (never disk)
- **Scope enforcement**: Every tool call validates JWT scope before executing
- **Rate limiting**: Respect Gateway rate limits (429 → return error to agent)
- **Audit trail**: All MCP-mediated operations produce audit events via Gateway
- **Tenant isolation**: JWT carries `tenant_id`, all calls are tenant-scoped

## 6. Configuration

```yaml
# MCP server config (env vars or config file)
GGID_GATEWAY_URL=http://localhost:8080
GGID_MCP_PORT=9060              # SSE transport (optional, stdio is default)
GGID_OAUTH_CLIENT_ID=mcp-server
GGID_OAUTH_CLIENT_SECRET=...    # Only for confidential clients
GGID_TENANT_ID=default          # Default tenant
```

## 7. Docker Deployment

Add to `deploy/docker-compose.yaml`:

```yaml
mcp:
  build:
    context: .
    dockerfile: services/mcp/Dockerfile
  ports:
    - "9060:9060"
  environment:
    - GGID_GATEWAY_URL=http://gateway:8080
    - GGID_OAUTH_CLIENT_ID=mcp-server
    - GGID_TENANT_ID=default
  depends_on:
    - gateway
```

## 8. Testing Strategy

- **Unit tests**: Mock Gateway client, test each tool handler independently
- **Integration tests**: Start MCP server + Gateway, verify tool calls end-to-end
- **Protocol tests**: Verify JSON-RPC message format compliance
- **Auth tests**: Verify scope enforcement (tool hidden when scope missing)

## 9. References

- [Model Context Protocol Specification](https://modelcontextprotocol.io/)
- `docs/research/ai-agent-identity-mcp.md` — competitive analysis and auth design
- `docs/research/mcp-cli-console-oauth-gaps.md` — gap analysis
- `services/oauth/internal/server/` — existing AI Agent Identity skeleton
