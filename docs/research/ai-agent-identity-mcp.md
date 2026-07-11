# AI Agent Identity & MCP Authentication

> Design doc: How GGID can support Model Context Protocol (MCP) authentication for AI agents, closing the #1 emerging competitive gap.

---

## Table of Contents

1. [What is MCP Auth?](#what-is-mcp-auth)
2. [Why It Matters](#why-it-matters)
3. [Competitive Landscape](#competitive-landscape)
4. [Proposed GGID Implementation](#proposed-ggid-implementation)
5. [New API Endpoints](#new-api-endpoints)
6. [Token Scoping for Agents](#token-scoping-for-agents)
7. [Security Considerations](#security-considerations)
8. [Implementation Roadmap](#implementation-roadmap)

---

## What is MCP Auth?

The Model Context Protocol (MCP) is an open standard for AI agents to connect to external tools and data sources. MCP authentication governs how AI agents prove their identity and receive scoped access to MCP servers.

### Key Concepts

| Concept | Description |
|---------|-------------|
| **Agent Identity** | An AI agent (e.g., Claude, GPT, custom LLM) has its own identity, distinct from the human user |
| **Agent Token** | Short-lived, scoped token the agent uses to call MCP servers |
| **Delegation** | Human user delegates specific permissions to the agent |
| **MCP Server** | A tool/data source that the agent calls (e.g., "read GitHub issues", "query database") |
| **Consent Flow** | User approves what the agent can access |

### Flow

```
Human User ──login──▶ GGID ──issues──▶ User JWT
User JWT ──delegate──▶ GGID ──issues──▶ Agent Token (scoped)
Agent Token ──calls──▶ MCP Server ──validates──▶ GGID JWKS
```

---

## Why It Matters

1. **AI agents need identity**: Without proper auth, agents share user tokens (security risk) or have no access (useless)
2. **Scoping is critical**: An agent that can read issues shouldn't necessarily delete repos
3. **Audit trail**: Must track which agent performed which action on behalf of which user
4. **Industry momentum**: Anthropic (Claude), OpenAI, Google all building MCP ecosystems

---

## Competitive Landscape

| Vendor | MCP Auth Support | Approach |
|--------|-----------------|---------|
| **Auth0** | In development | Agent-to-server OAuth 2.0 extension |
| **Keycloak** | Planned | Custom protocol mapper for agent identities |
| **WorkOS** | Not yet | — |
| **Clerk** | Not yet | — |
| **Supabase** | Not yet | — |
| **GGID** | **Proposed** | Token exchange (RFC 8693) + agent-scoped tokens |

### GGID Advantage

GGID already has the building blocks:
- **OAuth 2.1 Token Exchange** (RFC 8693) — already implemented
- **DPoP** (RFC 9449) — sender-constrained tokens
- **Fine-grained scopes** — `read:users`, `write:content`, etc.
- **Audit trail** — every API call logged
- **Multi-tenant** — agents scoped to a tenant

---

## Proposed GGID Implementation

### Phase 1: Agent Registration

```bash
# Register an AI agent as an OAuth client
POST /api/v1/oauth/clients
{
  "client_id": "agent-claude-prod",
  "client_secret": "...",
  "grant_types": ["client_credentials", "urn:ietf:params:oauth:grant-type:token-exchange"],
  "scope": "mcp:tools mcp:resources",
  "client_metadata": {
    "agent_type": "assistant",
    "model": "claude-3.5-sonnet",
    "max_session_duration": "1h"
  }
}
```

### Phase 2: Delegated Token Exchange

Human user delegates to agent via RFC 8693 Token Exchange:

```bash
POST /api/v1/oauth/token
Content-Type: application/x-www-form-urlencoded

grant_type=urn:ietf:params:oauth:grant-type:token-exchange
&subject_token=<user-jwt>
&subject_token_type=urn:ietf:params:oauth:token-type:access_token
&requested_token_type=urn:ietf:params:oauth:token-type:access_token
&audience=mcp-server-1
&scope=read:tools read:resources
&actor_token=<agent-client-assertion>
&actor_token_type=urn:ietf:params:oauth:token-type:jwt
```

Response:
```json
{
  "access_token": "eyJ...",
  "issued_token_type": "urn:ietf:params:oauth:token-type:access_token",
  "token_type": "Bearer",
  "expires_in": 3600,
  "scope": "read:tools read:resources"
}
```

### Token Claims

The agent token includes both `sub` (human) and `act` (agent):

```json
{
  "sub": "usr_human123",
  "act": {
    "sub": "agent-claude-prod",
    "type": "ai_agent"
  },
  "scope": "read:tools read:resources",
  "aud": "mcp-server-1",
  "exp": 1700003600,
  "delegation_source": "token_exchange"
}
```

---

## New API Endpoints

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/api/v1/agents` | GET, POST | Register/list AI agents |
| `/api/v1/agents/{id}` | GET, PUT, DELETE | Manage agent registration |
| `/api/v1/agents/{id}/scopes` | GET, PUT | Configure agent scopes |
| `/api/v1/agents/{id}/sessions` | GET, DELETE | List/revoke active agent sessions |
| `/api/v1/agents/{id}/audit` | GET | View agent action history |
| `/api/v1/oauth/delegate` | POST | Explicit delegation consent flow |

---

## Token Scoping for Agents

### MCP Scope Hierarchy

```
mcp:*                    — All MCP operations
  mcp:tools              — Call tools
    mcp:tools:read       — List available tools
    mcp:tools:execute    — Execute tool calls
  mcp:resources          — Access resources
    mcp:resources:read   — Read resource data
    mcp:resources:write  — Modify resources
  mcp:prompts            — Use prompt templates
```

### Per-Tool Scoping

```json
{
  "allowed_tools": ["github.read_issues", "jira.create_ticket"],
  "denied_tools": ["github.delete_repo", "*.*.admin"],
  "max_calls_per_hour": 100
}
```

---

## Security Considerations

| Risk | Mitigation |
|------|------------|
| Agent token theft | Short TTL (1h max), DPoP proof required |
| Over-permissive delegation | User must explicitly approve scope via consent flow |
| Agent impersonation | `act` claim identifies the agent, verified via client assertion |
| Unlimited tool calls | Per-agent rate limiting (configurable) |
| Silent permission escalation | Audit log records `act.sub` for every agent action |
| Agent runs after user leaves | Session TTL + idle timeout |

---

## Implementation Roadmap

| Phase | Features | Timeline |
|-------|----------|----------|
| **1** | Agent registration, token exchange, basic audit | 4 weeks |
| **2** | Per-tool scoping, rate limiting, consent UI | 4 weeks |
| **3** | DPoP enforcement for agent tokens, session management | 2 weeks |
| **4** | Console UI for agent management, delegation history | 2 weeks |
| **5** | MCP server SDK (Go, Node) for validating agent tokens | 2 weeks |

---

*Last updated: 2025-07-11*