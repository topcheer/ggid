# A2A Protocol — Agent-to-Agent Authentication & Authorization

> Research Date: 2026-07-18
> Status: GAP — no A2A-specific implementation
> Priority: P1 (competitive pressure from Auth0, Google)

## Executive Summary

Google's Agent2Agent (A2A) protocol enables AI agents from different frameworks to communicate securely. Auth0 has partnered with Google to define A2A auth specs. GGID has agent identity infrastructure (agent_lifecycle_handler, token_exchange_delegation) but no A2A protocol support. This is a competitive gap as Auth0 and Entra are already shipping A2A auth.

## What Is A2A?

The Agent2Agent protocol is an open standard from Google Cloud (60+ contributors) that enables:
- **Agent discovery**: Agents publish Agent Cards describing their capabilities and auth requirements.
- **Cross-framework communication**: LangGraph ↔ Google ADK ↔ Vercel AI SDK ↔ custom agents.
- **Delegation chains**: Agent acting on behalf of a user → another agent → a service.

### Agent Card Structure

```json
{
  "name": "HR Agent",
  "description": "Handles employment verification",
  "url": "https://hr-agent.internal/",
  "version": "1.0.0",
  "capabilities": {
    "streaming": true
  },
  "skills": [{
    "id": "is_active_employee",
    "name": "Check Employment Status"
  }],
  "securitySchemes": {
    "oauth2_m2m_client": {
      "flows": {
        "clientCredentials": {
          "tokenUrl": "https://ggid.iot2.win/oauth/token",
          "scopes": {
            "read:employee_status": "Verify employment"
          }
        }
      }
    }
  }
}
```

## Auth Patterns in A2A

| Pattern | Flow | Use Case |
|---------|------|----------|
| **M2M Client Credentials** | Agent → Agent token | Server agents verifying identity |
| **Delegation (on-behalf-of)** | User → Agent A → Agent B | Chain of delegated actions |
| **CIBA** | Headless agent → push to user | UI-less agent auth |
| **Fine-grained scopes** | Per-skill OAuth scopes | "read:employee_status" only |

## Current GGID State

| Component | Status | A2A Ready? |
|-----------|--------|------------|
| Agent lifecycle | Implemented | Partial (no Agent Card) |
| Token exchange (RFC 8693) | Implemented | Yes |
| Delegation chains | Implemented | Yes |
| CIBA | Implemented | Yes |
| Agent Card endpoint | Missing | No |
| Skill-based scopes | Partial (standard scopes only) | Needs per-skill scope management |
| A2A SDK | Missing | No |

## Competitive Analysis

| Vendor | A2A Support |
|--------|-------------|
| **Auth0** | Shipping (Auth0 for AI Agents, Google partnership) |
| **Microsoft Entra** | Agent ID with A2A patterns |
| **Keycloak** | Experimental CIMD (26.6+) |
| **Okta** | Not yet |
| **Cognito** | Not yet |

## Gap Items

### KB-234: Agent Card endpoint (/.well-known/agent-card)
**Type**: Backend (services/oauth)
**Priority**: P1
Implement `/.well-known/agent-card` endpoint returning JSON Agent Card with capabilities, skills, and OAuth security schemes.

### KB-235: Skill-based OAuth scope management
**Type**: Backend (services/oauth)
**Priority**: P1
Extend scope management to support per-skill scopes (e.g., `read:employee_status`) that map to specific agent capabilities.

### KB-236: A2A delegation chain validation
**Type**: Backend (services/oauth)
**Priority**: P1
Validate multi-hop delegation chains (User → AgentA → AgentB → Service) with proper audit trail.

### KB-237: A2A console — agent registry
**Type**: Frontend (console/src)
**Priority**: P2
Console page for registering external agents, viewing Agent Cards, managing inter-agent trust relationships.

## Architecture Proposal

```
External Agent (LangGraph)
         ↓ GET /.well-known/agent-card
    GGID returns Agent Card
         ↓ Agent requests token with skill scope
    POST /oauth/token (client_credentials, scope=read:employee_status)
         ↓ Token with delegation chain
    Agent calls GGID API with Bearer token
         ↓ Validate delegation chain + scope
    Authorize or deny
```

## Implementation Estimate

| Component | Effort |
|-----------|--------|
| Agent Card endpoint | 2d |
| Skill-based scopes | 3d |
| Delegation chain validation | 3d |
| Console agent registry | 3d |
| **Total** | **~11d** |

## Business Value

- **Competitive parity**: Auth0 and Entra are shipping A2A — GGID must match.
- **AI ecosystem readiness**: A2A is becoming the standard for agent interoperability.
- **Enterprise demand**: Multi-agent systems need identity governance.
- **Revenue opportunity**: Agent identity management is a new product category.

## References

- [A2A Protocol (Google)](https://github.com/google-a2a/a2a)
- [Auth0 + Google A2A Partnership](https://auth0.com/blog/auth0-google-a2a/)
- [Microsoft Entra Agent ID](https://learn.microsoft.com/en-us/entra/agent-id/)
