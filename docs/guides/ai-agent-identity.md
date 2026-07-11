# AI Agent Identity & MCP Auth

> Authenticate AI agents, delegate user authority via RFC 8693 token exchange, and authorize MCP server access.

---

## Overview

GGID's AI Agent Identity model lets autonomous AI agents act on behalf of users with scoped, delegatable tokens. Built on RFC 8693 (Token Exchange) with agent-specific claims.

```
User → Token Exchange → Agent Token (scoped, delegatable)
                         ↓
                    MCP Server Access
                    API Calls as Agent
                    Sub-agent Delegation
```

---

## Quick Start (3 Steps)

### 1. Register an Agent

```bash
curl -X POST http://localhost:8080/api/v1/agents/register \
  -H "Authorization: Bearer $USER_JWT" \
  -H "X-Tenant-ID: $TENANT" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Code Assistant",
    "type": "coding-assistant",
    "owner_user_id": "'"$USER_ID"'",
    "description": "Writes code on behalf of the user",
    "allowed_scopes": ["read:users", "write:users"],
    "allowed_mcp_servers": ["mcp://github.internal", "mcp://fs.internal"],
    "max_delegation_depth": 2,
    "rate_limit_per_min": 100
  }'
```

**Response (201):**
```json
{
  "id": "a1b2c3d4-...",
  "name": "Code Assistant",
  "type": "coding-assistant",
  "status": "active",
  "client_id": "agent_a1b2c3d4",
  "allowed_scopes": ["read:users", "write:users"],
  "allowed_mcp_servers": ["mcp://github.internal", "mcp://fs.internal"],
  "max_delegation_depth": 2,
  "rate_limit_per_min": 100
}
```

### 2. Exchange User Token for Agent Token

```bash
curl -X POST http://localhost:8080/api/v1/agents/token \
  -H "Authorization: Bearer $USER_JWT" \
  -H "X-Tenant-ID: $TENANT" \
  -H "Content-Type: application/json" \
  -d '{
    "agent_id": "a1b2c3d4-...",
    "requested_scope": ["read:users"],
    "mcp_servers": ["mcp://github.internal"]
  }'
```

**Response (200):**
```json
{
  "access_token": "eyJ...",
  "token_type": "Bearer",
  "expires_in": 900,
  "scope": "read:users",
  "agent_id": "a1b2c3d4-...",
  "delegation_depth_remaining": 2,
  "issued_token_type": "urn:ietf:params:oauth:token-type:access_token"
}
```

### 3. Verify Agent Token

```bash
curl -X POST http://localhost:8080/api/v1/agents/verify \
  -H "X-Tenant-ID: $TENANT" \
  -H "Content-Type: application/json" \
  -d '{"token": "eyJ..."}'
```

**Response (200):**
```json
{
  "valid": true,
  "agent_id": "a1b2c3d4-...",
  "agent_type": "coding-assistant",
  "delegation_chain": [
    {"sub": "usr_abc123", "agent_id": "", "agent_type": ""},
    {"sub": "agent_a1b2c3d4", "agent_id": "a1b2c3d4-...", "agent_type": "coding-assistant"}
  ],
  "mcp_servers": ["mcp://github.internal"],
  "max_delegation_depth": 2,
  "scope": "read:users"
}
```

---

## Agent Token JWT Claims

The agent token is a JWT with custom claims:

| Claim | Type | Description |
|-------|------|-------------|
| `agent_id` | string | UUID of the registered agent |
| `agent_type` | string | `coding-assistant`, `data-pipeline`, `customer-service`, `workflow-orchestrator`, `research-agent`, `custom` |
| `is_agent_token` | bool | Quick filter for introspection (always `true` for agent tokens) |
| `delegation_chain` | array | Ordered list from original user to current actor |
| `mcp_servers` | string[] | Authorized MCP server URIs |
| `max_delegation_depth` | int | Remaining delegation hops allowed (0 = no further delegation) |
| `act_sub` | string | Subject of the actor who delegated (RFC 8693 `act` claim) |
| `sub` | string | Agent's client ID (`agent_xxxx`) |
| `scope` | string | Space-delimited scopes granted |

### Decoded JWT Example

```json
{
  "sub": "agent_a1b2c3d4",
  "iss": "http://localhost:8080",
  "exp": 1723315200,
  "iat": 1723314300,
  "scope": "read:users",
  "agent_id": "a1b2c3d4-...",
  "agent_type": "coding-assistant",
  "is_agent_token": true,
  "delegation_chain": [
    {"sub": "usr_abc123"},
    {"sub": "agent_a1b2c3d4", "agent_id": "a1b2c3d4-...", "agent_type": "coding-assistant"}
  ],
  "mcp_servers": ["mcp://github.internal"],
  "max_delegation_depth": 2,
  "act_sub": "usr_abc123"
}
```

---

## Delegation Chain (Multi-Level)

Agents can delegate to sub-agents. Each hop reduces `max_delegation_depth` by 1.

```
User (usr_abc)
  └─ delegates to → Agent A (coding-assistant, depth=2)
       └─ delegates to → Agent B (data-pipeline, depth=1)
            └─ delegates to → Agent C (research-agent, depth=0, no further)
```

### Multi-Level Token Exchange

```bash
# Level 1: User → Agent A
AGENT_A_TOKEN=$(curl -s -X POST .../api/v1/agents/token \
  -H "Authorization: Bearer $USER_JWT" \
  -d '{"agent_id":"agent-a-uuid","requested_scope":["read:users"]}' \
  | jq -r .access_token)

# Level 2: Agent A → Agent B (Agent A's token is the subject_token)
AGENT_B_TOKEN=$(curl -s -X POST .../api/v1/agents/token \
  -H "Authorization: Bearer $AGENT_A_TOKEN" \
  -d '{"agent_id":"agent-b-uuid","requested_scope":["read:users"]}' \
  | jq -r .access_token)
```

**Agent B's delegation chain:**
```json
[
  {"sub": "usr_abc"},
  {"sub": "agent_a", "agent_id": "agent-a-uuid", "agent_type": "coding-assistant"},
  {"sub": "agent_b", "agent_id": "agent-b-uuid", "agent_type": "data-pipeline"}
]
```

When `max_delegation_depth` reaches 0, the token cannot be exchanged further.

---

## MCP Server Authorization

MCP (Model Context Protocol) servers are specified as URIs in `allowed_mcp_servers` during registration. The agent token includes an `mcp_servers` claim listing which servers the agent may access.

```bash
# Register agent with MCP access
curl -X POST .../api/v1/agents/register \
  -d '{
    "allowed_mcp_servers": [
      "mcp://github.internal",
      "mcp://filesystem.internal",
      "mcp://database.internal"
    ]
  }'

# Token exchange requests specific MCP servers
curl -X POST .../api/v1/agents/token \
  -d '{"mcp_servers": ["mcp://github.internal"]}'
```

MCP servers validate the `mcp_servers` claim in the agent token before accepting connections.

---

## List Agents

```bash
curl http://localhost:8080/api/v1/agents \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT"
```

**Response (200):**
```json
{
  "agents": [
    {
      "id": "a1b2c3d4-...",
      "name": "Code Assistant",
      "type": "coding-assistant",
      "status": "active",
      "owner_user_id": "usr_abc123",
      "allowed_scopes": ["read:users", "write:users"],
      "max_delegation_depth": 2,
      "rate_limit_per_min": 100,
      "created_at": "2025-07-11T12:00:00Z"
    }
  ],
  "total": 1
}
```

---

## Agent Lifecycle

| Status | Description |
|--------|-------------|
| `active` | Agent can exchange tokens and operate |
| `suspended` | Temporarily disabled (rate limit, suspicious activity) |
| `revoked` | Permanently disabled. All agent tokens become invalid. |

```bash
# Suspend an agent
curl -X PUT http://localhost:8080/api/v1/agents/{id}/status \
  -H "Authorization: Bearer $ADMIN_JWT" \
  -d '{"status":"suspended"}'
```

---

## Security Considerations

### Scope Reduction
Agent tokens **can never exceed** the user's original scopes. The `requested_scope` in token exchange is intersected with both the agent's `allowed_scopes` and the subject token's scopes.

### Delegation Depth
Set `max_delegation_depth` carefully:
- **0**: Agent cannot delegate to sub-agents (most restrictive)
- **1**: One level of sub-delegation
- **2-3**: Deep automation chains (rarely needed)

### Rate Limiting
Each agent has `rate_limit_per_min` (default 100). Enforced at the Gateway.

### Token Lifetime
Agent tokens inherit the standard 15-minute expiry. Refresh via re-exchange.

### Audit Trail
Every agent token exchange and verification is published to the audit pipeline as `oauth.token` events with `agent_id` and `delegation_chain` metadata.

---

## API Summary

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/v1/agents/register` | Register a new agent |
| `GET` | `/api/v1/agents` | List agents by tenant |
| `POST` | `/api/v1/agents/token` | Exchange user token for agent token (RFC 8693) |
| `POST` | `/api/v1/agents/verify` | Verify an agent token |
| `PUT` | `/api/v1/agents/{id}/status` | Update agent status (admin) |

---

*See: [REST API Reference](../api/rest-api.md) | [Security Overview](../architecture/security-overview.md) | [OAuth Flows](../oauth-flows-guide.md)*

*Last updated: 2025-07-11*
