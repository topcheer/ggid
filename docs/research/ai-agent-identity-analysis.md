# AI Agent Identity and MCP Authentication — Deep Competitive Analysis

> **Status**: Research & Strategic Analysis
> **Author**: GGID Competitive Analysis Team
> **Date**: 2025-07
> **Scope**: AI agent authentication, MCP (Model Context Protocol) auth, competitive landscape, GGID positioning
> **Related Docs**: [Token Exchange RFC 8693](token-exchange-rfc8693.md), [Token Exchange IAM Patterns](token-exchange-iam.md), [Headless Auth](headless-auth-iam.md), [OAuth 2.1 Analysis](oauth-2.1-analysis.md), Competitive Gap Audit

---

## Table of Contents

1. [The AI Agent Identity Problem](#1-the-ai-agent-identity-problem)
2. [MCP (Model Context Protocol)](#2-mcp-model-context-protocol)
3. [Auth0's Approach](#3-auth0s-approach)
4. [Keycloak's Approach](#4-keycloaks-approach)
5. [Casdoor's Approach](#5-casdoors-approach)
6. [AI Agent Auth Flow Requirements](#6-ai-agent-auth-flow-requirements)
7. [RFC 8693 Token Exchange for Agents](#7-rfc-8693-token-exchange-for-agents)
8. [OAuth 2.0 for AI Agents (Draft)](#8-oauth-20-for-ai-agents-draft)
9. [Agent Identity Model](#9-agent-identity-model)
10. [GGID Positioning Strategy](#10-ggid-positioning-strategy)
11. [Gap Analysis & Recommendations](#11-gap-analysis--recommendations)
12. [References](#12-references)

---

## 1. The AI Agent Identity Problem

### 1.1 The Rise of Autonomous Agents

AI agents — LLM-powered tools that browse the web, call APIs, execute code, and interact
with enterprise systems on behalf of users — have moved from experimental to production in
less than two years. Enterprises are deploying thousands of agents weekly: customer service
bots, coding assistants, data pipeline agents, financial analysis agents, and autonomous
workflow orchestrators. Each of these agents needs to authenticate to services, access
sensitive data, and make decisions — often without real-time human oversight.

The OpenID Foundation's October 2025 whitepaper, *"Identity Management for Agentic AI: The
new frontier of authorization, authentication, and security for an AI agent world"*, frames
the challenge precisely:

> *"The concept of identity in the context of AI agents is multifaceted and extends beyond
> simple user impersonation, playing a critical role in authentication, authorization, and
> auditability."*

The Cloud Security Alliance (CSA) goes further:

> *"Traditional identity management systems like OAuth and SAML were designed for human users
> and/or static machine identities. However, they fall short in the dynamic world of AI agents."*

### 1.2 Why Traditional OAuth Flows Break for Agents

OAuth 2.0 was designed for a specific interaction pattern: a human user in a browser grants
consent to a web application to access their data on a third-party service. This model assumes:

- **A browser is available** — The authorization code flow requires redirecting the user to
  an authorization endpoint in a web browser. Agents running in headless environments
  (servers, containers, CI/CD pipelines, serverless functions) have no browser.

- **Sessions are short-lived** — OAuth access tokens typically expire in 5-60 minutes, with
  refresh tokens enabling interactive renewal. Agents may need to operate autonomously for
  hours, days, or even weeks between human check-ins.

- **Single-hop delegation** — OAuth assumes a direct relationship: user → application → API.
  AI agents routinely chain through multiple hops: user → primary agent → sub-agent → MCP
  server → backend API. Traditional OAuth has no native mechanism for representing this
  delegation chain.

- **Predictable scopes** — OAuth scopes are defined at client registration time. AI agents
  need dynamic, context-sensitive scopes: an agent analyzing a customer's order history
  needs different permissions than the same agent processing a refund.

- **Interactive consent** — Each new scope requires user consent. An autonomous agent cannot
  pause mid-reasoning to ask the user "can I access the billing API?" — or if it must, the
  UX for such handoffs is undefined.

### 1.3 The Multi-Step Reasoning Problem

AI agents do not make single API calls. A typical agent task involves:

1. **Planning** — Decomposing a user request into sub-tasks
2. **Tool selection** — Choosing which tools/APIs to invoke
3. **Execution** — Making multiple sequential and parallel API calls
4. **Reasoning** — Processing results and deciding next steps
5. **Verification** — Checking outputs for correctness and safety

Each step may require different permissions, different tokens, and different rate limits. An
agent that needs to (a) search a CRM, (b) read a customer record, (c) update a support ticket,
and (d) send an email notification — each of these requires a different scope, a different
audience, and potentially a different downstream service.

Traditional OAuth would require four separate authorization flows or a single token with
over-broad scopes. Neither is acceptable for production security.

### 1.4 New Threat Vectors Unique to Agents

#### 1.4.1 Prompt Injection Stealing Tokens

Prompt injection attacks manipulate AI agent inputs to override instructions, extract sensitive
data, or trigger unauthorized actions. A carefully crafted input embedded in a web page, email,
or document can cause an agent to:

- **Exfiltrate tokens** — The injected prompt instructs the agent to include its OAuth token
  in an API call to an attacker-controlled endpoint.
- **Grant unauthorized access** — The agent is tricked into performing a token exchange or
  consent flow that grants broader permissions to an attacker.
- **Chain lateral movement** — A single compromised agent can pivot through connected services
  using its legitimate credentials.

Research from Obsidian Security shows that *"AI agents move 16x more data than human users
performing equivalent tasks, which dramatically expands the blast radius of any single
compromised agent."* The 2025 Salesloft-Drift breach demonstrated this: compromised OAuth
tokens granted attackers access to hundreds of downstream environments, with a blast radius
10x greater than previous incidents.

#### 1.4.2 Agent Hallucination Granting Access

AI agents can hallucinate — generating plausible but incorrect outputs. In the context of
identity and access management, this creates a novel threat: an agent might:

- **Grant permissions it doesn't have** — The agent's internal model of its capabilities
  diverges from its actual token scopes, leading it to attempt (or even "succeed" at)
  unauthorized actions if downstream services don't enforce scope checks.
- **Misattribute actions** — An agent might log actions under the wrong user identity, or
  fail to record its own involvement, breaking audit trails.
- **Bypass consent** — An agent reasoning about user intent might skip consent prompts it
  should have triggered, especially in long-running autonomous workflows.

#### 1.4.3 Token Compromise and Excessive Privilege

The CSA and Obsidian Security both report that approximately **90% of AI agents hold excessive
privileges**. Agents are typically granted broad API keys or long-lived OAuth tokens with far
more permissions than their workflows actually need. Combined with the non-deterministic nature
of LLM reasoning, this creates a toxic combination:

- Broad data reachability
- Minimal oversight
- Non-deterministic behavior patterns
- Elevated privilege across multiple SaaS platforms

#### 1.4.4 Shadow AI and Unauthorized Agents

Employees deploy AI tools without security review, creating visibility gaps similar to shadow
SaaS. Unauthorized AI agents operate outside governance frameworks, with unmanaged credentials,
untracked data access, and no centralized identity governance.

### 1.5 The Fundamental Mismatch

The core problem is a fundamental mismatch between identity infrastructure designed for
human-centric, predictable, session-based interactions and agent-centric, autonomous,
non-deterministic, long-running operations. The industry needs:

| Requirement | Traditional OAuth | AI Agent Need |
|-------------|------------------|---------------|
| Token lifetime | 5-60 minutes | Hours to days |
| Delegation depth | 1 hop | Multiple hops with act chains |
| Scope granularity | Static at registration | Dynamic per tool call |
| Consent model | Interactive browser prompt | Async, human-in-loop handoff |
| Audit granularity | Per-session | Per-tool-call with reasoning context |
| Identity type | Human user | Agent identity + human owner |
| Rate limiting | Per client | Per agent identity + per tool |
| Revocation | Token-level | Behavioral anomaly-triggered |

---

## 2. MCP (Model Context Protocol)

### 2.1 What Is MCP?

The Model Context Protocol (MCP) is an open protocol introduced by Anthropic in November 2024
for connecting AI assistants to external tools and data sources. In just 18 months, it has
become the de facto standard for AI agent integration, governed under the Linux Foundation
as of 2025.

MCP defines a host-client-server architecture:

```
┌─────────────┐     ┌─────────────┐     ┌──────────────┐
│  MCP Host   │     │  MCP Client │     │  MCP Server  │
│  (Claude,   │────▶│             │────▶│  (Database,  │
│   Cursor,   │     │  Protocol   │     │   API, Tool) │
│   Windsurf) │     │  Layer      │     │              │
└─────────────┘     └─────────────┘     └──────────────┘
```

The protocol provides primitives for:
- **Tools** — Callable functions the agent can invoke
- **Resources** — Data sources the agent can read
- **Prompts** — Pre-configured prompt templates
- **Sampling** — Server-initiated LM requests (deprecated in 2026-07-28)
- **Tasks** — Long-running async work (moved to extension in 2026-07-28)

### 2.2 MCP 2026-07-28 Spec — The Largest Revision Since Launch

On May 21, 2026, the MCP lead maintainers published the release candidate for MCP 2026-07-28,
calling it "the largest revision of the protocol since launch." The changes fundamentally
reshape how MCP authentication works.

#### 2.2.1 Stateless Core

The headline change: **MCP no longer manages sessions at the protocol layer.**

- The `initialize`/`initialized` handshake is **removed** (SEP-2575). Protocol version, client
  info, and client capabilities now travel in `_meta` on every request.
- The `Mcp-Session-Id` header is **removed** (SEP-2567). No more sticky sessions, shared
  session stores, or deep packet inspection at the gateway.
- **Impact**: MCP servers can now run behind plain round-robin load balancers. No sticky
  routing required. Any server instance can handle any request.

Stateless protocol does not mean stateless applications — servers that need cross-call state
mint explicit handles (e.g., `basket_id`, `browser_id`) that the model passes back as arguments.

#### 2.2.2 OAuth 2.1 Resource Server Model

The 2026-07-28 spec aligns MCP authorization with OAuth 2.1 and OpenID Connect:

- **MCP servers are formally OAuth 2.1 resource servers.** They MUST implement OAuth 2.0
  Protected Resource Metadata (RFC 9728) so clients can discover the authorization server
  automatically via a `.well-known/oauth-protected-resource` endpoint.

- **MCP clients MUST implement Resource Indicators (RFC 8707).** When a client requests a
  token, it must specify which MCP server the token is for. A token minted for Server A cannot
  be replayed against Server B. This directly addresses the confused deputy / token mix-up
  problem.

- **Client ID Metadata Documents (CIMD) replace Dynamic Client Registration.** CIMD is now
  the preferred method for client registration. Dynamic Client Registration (RFC 7591) is
  deprecated.

- **Issuer verification is required.** Clients must validate which authorization server issued
  an authorization response (RFC 9207) and bind registered credentials to the issuing AS's
  issuer. If a resource migrates between authorization servers, the client must re-register.

- **Refresh token handling is formalized** (SEP-2207), with scope accumulation during step-up
  authorization (SEP-2350).

- **Application type declaration** during registration (SEP-837) — clients declare their OIDC
  `application_type`, preventing the common failure where a CLI/desktop client is defaulted
  to "web" and then rejected for localhost redirect URIs.

#### 2.2.3 Extensions Framework

The 2026-07-28 spec introduces a formal extensions framework with reverse-DNS identifiers,
independent repositories, and delegated maintainers:

- **MCP Apps** — Servers can render interactive HTML UIs in the client, with all UI-initiated
  actions going through the same JSON-RPC consent path as tool calls.
- **Tasks** — First-class support for long-running async work with lifecycle management
  (`tasks/get`, `tasks/update`, `tasks/cancel`).

#### 2.2.4 What's Deprecated

- **Roots** → replaced by Resource URIs (plain URLs)
- **Sampling** → removed from core spec, may return as extension
- **Logging** → removed from core spec

### 2.3 Streamable HTTP Transport

MCP's Streamable HTTP transport is the primary transport for remote MCP servers. Key
characteristics:

- Uses HTTP POST for JSON-RPC requests
- Server-Sent Events (SSE) for streaming responses
- Single endpoint for all communication (no separate notification channels)
- Stateless-friendly: the 2026-07-28 spec removes session requirements

The transport layer itself does not handle authentication. Instead, it relies on standard HTTP
mechanisms — `Authorization: Bearer <token>` headers — validated against the OAuth 2.1
authorization server.

### 2.4 How MCP Servers Authenticate to Backends

An MCP server is both a resource server (validating incoming tokens) and potentially a client
(authenticating to backend APIs). The typical flow:

```
AI Agent ──token──▶ MCP Server ──scoped token──▶ Backend API
    │                   │
    │              Token Exchange
    │              (RFC 8693)
    │                   │
    ▼                   ▼
Auth Server        Backend validates
(Issues agent     scoped token, checks
 tokens)           act claim for audit
```

1. The AI agent presents its OAuth token to the MCP server
2. The MCP server validates the token (signature, expiry, audience)
3. The MCP server exchanges the token for a scoped-down token (RFC 8693) to call the backend
4. The backend validates the scoped token and logs the act claim for audit

### 2.5 Why Every IAM Vendor Is Adding MCP Auth Support

The MCP ecosystem is exploding. Enterprise adoption demands:

1. **Cross-app access** — Users authenticate through their organization's IdP (Okta, Azure AD,
   Google Workspace) and use MCP servers without separate OAuth flows
2. **Enterprise SSO integration** — MCP access should feel like normal enterprise SSO
3. **Audit trails** — Every tool call must be auditable
4. **Gateway/proxy patterns** — MCP servers behind enterprise API gateways
5. **Authorization propagation** — User identity flows through to backends

Every major IAM vendor — Auth0, Keycloak, Casdoor, WorkOS — is racing to be the authorization
server for MCP. This is the next platform war in identity.

### 2.6 MCP 2026 Roadmap: Beyond Auth

The MCP roadmap for 2026 extends well beyond authentication:

| Feature | Description | Timeline |
|---------|-------------|----------|
| Stateless transport | Remove session requirement for horizontal scaling | June 2026 cycle |
| Server Discovery | `.well-known` URLs for "MCP Server Cards" | 2026 |
| Tasks | Async work lifecycle (retry, expiry) | Extension |
| Enterprise auth | Cross-app SSO, audit, gateway patterns | Ongoing |
| Triggers | Webhook-like server-initiated notifications | On the horizon |
| Streaming | Incremental result delivery | On the horizon |
| Skills | Domain knowledge alongside tools | Extension |
| MCP Apps | Interactive HTML in MCP clients | Extension |
| SDK v2 | Python and TypeScript SDK rewrite | 2026 |

---

## 3. Auth0's Approach

### 3.1 Auth0 MCP Server — Overview

Auth0 has shipped an MCP server that connects AI agents (Claude Desktop, Cursor, Windsurf) to
the Auth0 Management API. This allows developers to use natural-language commands to create
applications, manage users, deploy Actions, and query logs.

**Key design decisions:**

- **OAuth 2.0 Device Authorization Flow** — The MCP server authenticates using the device flow
  (RFC 8628), which is ideal for CLI/desktop tools without a browser. The user authenticates
  on their phone/laptop while the MCP server polls for completion.

- **Secure credential storage** — Credentials are stored in the system keychain (macOS
  Keychain, Windows Credential Manager, Linux Secret Service), never in plain text.

- **Minimal API permissions** — Only essential permissions are requested, adhering to the
  principle of least privilege.

### 3.2 Architecture

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  MCP Client  │     │  Auth0 MCP   │     │  Auth0       │
│  (Cursor)    │────▶│  Server      │────▶│  Management  │
│              │     │              │     │  API         │
└──────────────┘     └──────┬───────┘     └──────────────┘
                            │
                     Device Auth Flow
                     (OAuth 2.0)
                            │
                            ▼
                     ┌──────────────┐
                     │  Auth0 AS    │
                     │  (tenant)    │
                     └──────────────┘
```

The flow:
1. Developer configures the MCP client (Cursor, Claude Desktop) with the Auth0 MCP server URL
2. On first use, the MCP server initiates the device authorization flow
3. The user authenticates at Auth0 (with MFA if configured) and grants consent
4. The MCP server receives an access token scoped to the Management API
5. Subsequent tool calls (create app, manage users) use this token

### 3.3 What Auth0 Got Right

1. **Device flow is correct for the use case** — Developers using Cursor/Windsurf are often in
   terminal environments where a browser redirect is awkward. Device flow (open URL on phone,
   approve, done) is seamless.

2. **Keychain storage** — Eliminates the most common token leak vector (tokens in `.env` files,
  config files, or environment variables).

3. **Minimal scopes** — The MCP server requests only what it needs. This is the right baseline
  but Auth0's broad Management API still means the token has significant power.

4. **MCP client ecosystem support** — Claude Desktop, Cursor, and Windsurf out of the box.
  This covers the majority of developer MCP usage.

### 3.4 What's Missing

1. **No agent identity** — The MCP server authenticates as the developer's user identity.
  There is no concept of "this is agent X operating on behalf of user Y." Every tool call
  is attributed to the user, not to an agent.

2. **No token exchange** — The MCP server receives one Management API token and uses it for
  everything. No per-tool scoped tokens, no audience restriction per operation.

3. **No multi-hop delegation** — If the agent spawns a sub-agent or calls another MCP server,
  there is no mechanism to propagate identity or scope down the chain.

4. **No behavioral monitoring** — Auth0 does not monitor whether agent behavior is anomalous.
  A prompt injection that tricks the agent into deleting users would succeed with the same
  privileges as a legitimate operation.

### 3.5 Pricing Implications

Auth0's pricing is per-Monthly Active User (MAU). With AI agents, the question arises: does
each agent count as a user? Current pricing models don't account for agents that make thousands
of API calls per day. Enterprises deploying hundreds of agents will face significant cost
questions, and Auth0 has not yet published agent-specific pricing.

---

## 4. Keycloak's Approach

### 4.1 Keycloak 26 — Experimental CIMD Support

Keycloak, the leading open-source IAM platform (backed by Red Hat), is positioning itself for
AI workloads through several mechanisms:

1. **Client ID Metadata Documents (CIMD)** — Keycloak 26 has experimental support for CIMD,
  aligning with the MCP 2026-07-28 spec's preference for CIMD over Dynamic Client Registration.

2. **Token Exchange (RFC 8693)** — Keycloak has native token exchange support, which is the
  critical capability for agent delegation patterns. An agent can exchange a user's token for
  a scoped-down, audience-bound agent token.

3. **Client Credentials for agent identity** — Each AI agent can be registered as a confidential
  client with service accounts enabled, receiving its own identity and scopes.

### 4.2 Five Patterns for AI Agent Auth with Keycloak

Skycloak's comprehensive guide (March 2026) documents five patterns Keycloak supports:

#### Pattern 1: Client Credentials for Agent Identity

For autonomous agents that operate as their own entity (background processors, data pipelines):

```
Agent registers as Keycloak client (ai-data-agent)
  → Service Accounts Enabled
  → Scoped roles: data:read, data:write, reports:generate
  → Uses client_credentials grant
  → Receives scoped access token
```

#### Pattern 2: Delegated Access via Token Exchange

For agents acting on behalf of users:

```python
# Exchange user token for constrained agent token
POST /realms/{realm}/protocol/openid-connect/token
grant_type=urn:ietf:params:oauth:grant-type:token-exchange
subject_token={user_token}
subject_token_type=urn:ietf:params:oauth:token-type:access_token
client_id={agent_client_id}
client_secret={agent_secret}
audience={target_audience}
scope={requested_scope}
```

The resulting token carries the user's identity but is scoped to only what the agent needs.

#### Pattern 3: MCP Server Authentication

MCP servers protected by Keycloak validate bearer tokens and enforce scopes:

```python
@app.post("/mcp/tools/search")
async def mcp_search_tool(claims=Depends(verify_agent_token)):
    scopes = claims.get("scope", "").split()
    if "search:execute" not in scopes:
        raise HTTPException(403, "Insufficient scope")
    agent_id = claims.get("azp", "unknown")
    user_sub = claims.get("sub", "unknown")
    # Log for audit
    print(f"Agent {agent_id} (user {user_sub}) executing search")
```

#### Pattern 4: Human-in-the-Loop Consent

For sensitive operations, Keycloak's step-up authentication and consent screens provide
human-in-the-loop approval:

```python
class HumanInTheLoopAgent:
    SENSITIVE_SCOPES = {"email:send", "payment:execute", "data:delete"}

    def request_action(self, scope, user_session):
        if scope in self.SENSITIVE_SCOPES:
            auth_url = build_auth_url(scope=scope, prompt="consent")
            raise ConsentRequiredError(consent_url=auth_url)
        return self._get_service_token(scope)
```

#### Pattern 5: LangChain Integration

Keycloak-issued tokens integrated with LangChain's tool-calling framework, with per-tool scoped
tokens:

```python
TOOL_SCOPES = {
    "web_search": "search:execute",
    "database_query": "db:read",
    "email_send": "email:send",
    "calendar_read": "calendar:read calendar:list",
}
```

### 4.3 Keycloak + SPIRE for Agent Identity

The `keycloak-agent-identity` project (by Christian Posta) integrates Keycloak with SPIRE
(SPIFFE Runtime Environment) for workload identity and MCP authentication. This provides:

- **SPIFFE IDs** for agents — cryptographic identity for each agent workload
- **Federated trust** — Agents across different clusters/domains can authenticate
- **MCP authentication** — SPIFFE-based mTLS between agents and MCP servers

### 4.4 Keycloak's Strengths for AI Agent Auth

| Capability | Keycloak Support |
|-----------|-----------------|
| Token Exchange (RFC 8693) | Native, production-ready |
| Client Credentials | Full support with service accounts |
| Fine-grained scopes | Full support |
| Audit logging | Event system with SIEM forwarding |
| MCP server protection | Via standard OAuth resource server pattern |
| Consent screens | Built-in with step-up auth |
| Session management | Full session lifecycle |
| Multi-tenancy | Realms for tenant isolation |

### 4.5 Keycloak's Limitations

1. **Java-only** — No Go-native SDK. Integrating with Go microservices requires HTTP calls.
2. **Heavy operationally** — JVM tuning, database management, clustering complexity.
3. **No native agent model** — Keycloak treats agents as OAuth clients, not first-class agent
  entities. No agent_type, no behavior monitoring.
4. **No MCP-specific tooling** — Each MCP server must implement its own Keycloak integration.

---

## 5. Casdoor's Approach

### 5.1 Agent-First Identity Positioning

Casdoor has repositioned itself as the "first open-source IAM platform with native MCP server,
OAuth 2.1 for AI agents." This is a bold positioning move aimed squarely at the AI agent
market.

From the Casdoor website:

> *"Casdoor: Identity & Access Management for the AI Agent era."*

### 5.2 Agent Registration Model

Casdoor introduces a first-class "Agent" entity:

| Field | Description |
|-------|-------------|
| Name | Unique identifier within the organization |
| Display name | Human-readable label |
| Listening URL | The agent's endpoint URL |
| Token | Bearer token for authenticating to the agent |
| Application | The Casdoor application associated with this agent |

Agents are managed via a dedicated "Agents" section in the Casdoor admin console, scoped to
organizations, and require admin privileges to create or edit.

### 5.3 Native MCP Server

Casdoor ships with a built-in MCP server, allowing AI agents to interact with Casdoor's
identity management APIs directly through the Model Context Protocol. This is a significant
differentiator — most other IAM platforms require separate MCP server implementations.

### 5.4 OAuth 2.1 for AI Agents

Casdoor's OAuth 2.1 implementation is specifically tailored for agent use cases:

- **Long-lived token support** — Tokens valid for hours or days, not minutes
- **Async token cleanup** — Automatic revocation of expired agent session tokens
- **Scope management** — Fine-grained scopes for agent-level access control
- **100+ identity provider support** — Agents can authenticate through any supported IdP

### 5.5 What Casdoor Is Doing Differently

1. **Agent as first-class entity** — Unlike Auth0 (where agents are just OAuth clients) or
  Keycloak (where agents are clients with service accounts), Casdoor treats agents as a
  distinct entity type with their own lifecycle management.

2. **Native MCP server** — Out of the box, not a bolt-on. This reduces integration friction.

3. **Async token cleanup** — Addresses the specific problem of long-lived agent sessions
  accumulating stale tokens. Traditional IAM systems clean up tokens lazily; Casdoor does it
  proactively for agent sessions.

4. **Casbin policy engine** — ABAC/PBAC policies for fine-grained agent authorization.

### 5.6 Casdoor's Limitations

1. **Go-based but different architecture** — Casdoor uses Casbin (Go) for authorization, which
  is different from GGID's RBAC+ABAC approach.

2. **Less mature enterprise features** — Casdoor lacks Keycloak's enterprise SSO breadth and
  Auth0's management API sophistication.

3. **No token exchange** — Casdoor does not appear to implement RFC 8693 token exchange, which
  is critical for agent delegation patterns.

4. **Limited audit trail** — While Casdoor has logging, it lacks the structured audit trail
  needed for compliance-grade agent action attribution.

---

## 6. AI Agent Auth Flow Requirements

### 6.1 What Agents Need That Humans Don't

Based on analysis of the OpenID Foundation whitepaper, CSA recommendations, and production
agent deployments, here are the six core requirements for AI agent authentication:

#### 6.1.1 Long-Lived OAuth Tokens

Human sessions are short (minutes). Agent sessions are long (hours to days).

**The problem**: An autonomous agent processing a batch of 10,000 records may run for 6 hours.
OAuth tokens expiring every 15 minutes force 24 token refreshes, each of which is a potential
failure point and a security event.

**The solution**: Configurable token lifetimes per agent type:
- Interactive agents (coding assistants): 1-4 hours
- Background agents (data pipelines): 24-48 hours
- Autonomous agents (workflows): 7 days with check-in

**Security mitigation**: Short-lived tokens with automatic refresh + behavioral monitoring.
The token itself is short-lived (15 min), but the refresh token is long-lived, and refresh
is automatic and transparent.

#### 6.1.2 Scoped Delegations

An agent authorized to "read calendar events" must not be able to "delete contacts."

**The problem**: Traditional OAuth scopes are defined at client registration time. AI agents
need per-tool, per-operation scopes that change dynamically based on what the agent is doing.

**The solution**: Token exchange (RFC 8693) for per-tool scoped tokens:
```
User token (scope: calendar:* email:* contacts:*)
  → Exchange for calendar-read token (scope: calendar:read, aud: calendar-api)
  → Exchange for email-send token (scope: email:send, aud: email-api)
```

Each tool gets only the scope it needs.

#### 6.1.3 Multi-Hop Delegation

User → Agent → Sub-agent → MCP Server → Backend API

**The problem**: Each hop in the delegation chain needs to know:
- Who the original user is
- Which agent initiated this request
- What scopes were delegated at each hop
- The full audit trail

**The solution**: RFC 8693 `act` claim chaining:
```json
{
  "sub": "user@example.com",
  "act": {
    "sub": "primary-agent",
    "act": {
      "sub": "sub-agent-researcher",
      "act": {
        "sub": "mcp-server-web-search"
      }
    }
  }
}
```

The backend sees the full chain: this request came from `mcp-server-web-search`, called by
`sub-agent-researcher`, called by `primary-agent`, on behalf of `user@example.com`.

#### 6.1.4 Audit Trail for Agent Actions

**The problem**: Compliance requires knowing exactly what an agent did, when, on whose behalf,
and why. A shared API key makes this impossible.

**The solution**: Structured audit logging with full delegation context:

```json
{
  "timestamp": "2025-07-15T10:30:00Z",
  "event_type": "agent_action",
  "agent_id": "customer-service-agent-001",
  "agent_type": "customer_service",
  "user_sub": "user@example.com",
  "action": "read_customer_data",
  "resource": "/api/v1/customers/12345",
  "scope_used": "customers:read",
  "tool": "crm_lookup",
  "reasoning_context": "User asked about order #67890",
  "result": "success",
  "ip": "10.0.0.5",
  "token_act_chain": ["user@example.com", "customer-service-agent-001"]
}
```

#### 6.1.5 Token Revocation for Anomalous Behavior

**The problem**: If an agent is compromised via prompt injection, its tokens must be revoked
immediately — but how do you know the agent is compromised?

**The solution**: Behavioral anomaly detection + automatic token revocation:
- Monitor agent API call patterns (volume, frequency, sensitivity)
- Flag deviations from baseline (e.g., agent suddenly accessing 10x normal data volume)
- Auto-revoke agent tokens and notify the owner
- Rate limit per agent identity

#### 6.1.6 Rate Limiting Per Agent Identity

**The problem**: AI agents can make thousands of API calls per minute. Without per-agent rate
limiting, a single agent can overwhelm a backend service or exhaust API quotas.

**The solution**: Rate limiting keyed on agent identity (not just client ID or IP):
- Per-agent request limits (e.g., 100 calls/minute for agent A, 500 for agent B)
- Per-tool rate limits (e.g., email:send limited to 10/minute regardless of agent)
- Burst control with circuit breakers
- Tenant-level aggregate limits

### 6.2 Summary: The Agent Auth Requirements Matrix

| Requirement | Priority | Standard | Complexity |
|-------------|----------|---------|------------|
| Long-lived tokens | P0 | OAuth 2.1 (configurable TTL) | Low |
| Scoped delegation | P0 | RFC 8693 Token Exchange | Medium |
| Multi-hop act chains | P1 | RFC 8693 act claim | Medium |
| Per-action audit | P0 | Custom logging + OAuth claims | Medium |
| Anomaly-based revocation | P1 | Custom monitoring | High |
| Per-agent rate limiting | P1 | Custom middleware | Medium |
| MCP server auth | P1 | OAuth 2.1 Resource Server | Medium |
| Agent identity model | P0 | Custom data model | High |
| Behavioral monitoring | P2 | Custom analytics | High |
| Consent handoff | P2 | OAuth + device flow | Medium |

---

## 7. RFC 8693 Token Exchange for Agents

### 7.1 Token Exchange as the Foundation of Agent Auth

RFC 8693 (OAuth 2.0 Token Exchange) is the single most important standard for AI agent
authentication. It provides the mechanism for:

- Exchanging a user's broad-scope token for an agent-specific scoped token
- Recording the delegation chain via the `act` claim
- Restricting token audience to specific services
- Maintaining audit trails across multi-hop delegation

### 7.2 The Two Modes: Impersonation vs Delegation

#### Impersonation

The agent obtains a token with the user's `sub` and **no `act` claim**. The token is
indistinguishable from one the user obtained directly.

```json
{
  "sub": "user@example.com",
  "scope": "read:profile"
}
```

**Use cases**: Legacy system compatibility, admin "login as user" support flows.

**Risk**: Actions are attributable only to the subject, not the actor. Breaks accountability.

#### Delegation

The agent obtains a token with the user's `sub` **and an `act` claim** recording the actor.

```json
{
  "sub": "user@example.com",
  "act": { "sub": "coding-agent" },
  "scope": "read:files",
  "aud": "file-service"
}
```

**Use cases**: All production agent scenarios. Microservice chains. "On behalf of" flows.

### 7.3 Token Exchange Request

```
POST /oauth/token HTTP/1.1
Host: oauth.ggid.dev
Content-Type: application/x-www-form-urlencoded
Authorization: Basic {base64(agent_client_id:agent_secret)}

grant_type=urn:ietf:params:oauth:grant-type:token-exchange
&subject_token={user_access_token}
&subject_token_type=urn:ietf:params:oauth:token-type:access_token
&actor_token={agent_client_credentials_token}
&actor_token_type=urn:ietf:params:oauth:token-type:access_token
&audience=file-service
&scope=read:files
&requested_token_type=urn:ietf:params:oauth:token-type:access_token
```

### 7.4 Response

```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIs...",
  "issued_token_type": "urn:ietf:params:oauth:token-type:access_token",
  "token_type": "Bearer",
  "expires_in": 3600
}
```

### 7.5 Go Implementation: Agent Token Exchange Flow

GGID's token exchange implementation design (already researched in `token-exchange-rfc8693.md`)
provides the foundation. Here is a complete agent-focused implementation:

```go
package agent

import (
    "context"
    "fmt"
    "net/http"
    "strings"
    "time"

    "github.com/golang-jwt/jwt/v5"
)

// AgentTokenExchanger handles token exchange for AI agent delegation.
type AgentTokenExchanger struct {
    OAuthServerURL string
    AgentClientID  string
    AgentSecret    string
    HTTPClient     *http.Client
}

// ExchangeRequest represents a token exchange request for agent delegation.
type ExchangeRequest struct {
    SubjectToken    string   // User's access token (the entity on whose behalf the agent acts)
    RequestedScopes []string // Scopes the agent needs (must be subset of subject token's scopes)
    Audience        string   // Target service for the exchanged token
    ActorToken      string   // Agent's own credential token (optional, for delegation mode)
    TokenType       string   // Requested token type (default: access_token)
}

// ExchangeResult contains the exchanged token and metadata.
type ExchangeResult struct {
    AccessToken  string
    ExpiresIn    int
    TokenType    string
    ActClaim     *ActClaim // The act claim embedded in the token (for audit)
}

// ActClaim represents the JWT "act" claim for delegation tokens (RFC 8693 Section 4.1).
type ActClaim struct {
    Sub string     `json:"sub"`              // Actor subject identifier
    Iss string     `json:"iss,omitempty"`    // Actor issuer
    Act *ActClaim  `json:"act,omitempty"`    // Nested actor for multi-hop delegation
}

// ExchangeForAgent exchanges a user token for an agent-scoped delegation token.
// This is the core operation enabling AI agents to act on behalf of users.
func (e *AgentTokenExchanger) ExchangeForAgent(
    ctx context.Context,
    req *ExchangeRequest,
) (*ExchangeResult, error) {
    // Build form data
    formData := url.Values{}
    formData.Set("grant_type", "urn:ietf:params:oauth:grant-type:token-exchange")
    formData.Set("subject_token", req.SubjectToken)
    formData.Set("subject_token_type", "urn:ietf:params:oauth:token-type:access_token")

    if len(req.RequestedScopes) > 0 {
        formData.Set("scope", strings.Join(req.RequestedScopes, " "))
    }
    if req.Audience != "" {
        formData.Set("audience", req.Audience)
    }
    if req.ActorToken != "" {
        // Delegation mode: include actor token to get act claim
        formData.Set("actor_token", req.ActorToken)
        formData.Set("actor_token_type", "urn:ietf:params:oauth:token-type:access_token")
    }

    tokenType := req.TokenType
    if tokenType == "" {
        tokenType = "urn:ietf:params:oauth:token-type:access_token"
    }
    formData.Set("requested_token_type", tokenType)

    // Make the exchange request
    httpReq, err := http.NewRequestWithContext(ctx, "POST",
        e.OAuthServerURL+"/oauth/token", strings.NewReader(formData.Encode()))
    if err != nil {
        return nil, fmt.Errorf("building request: %w", err)
    }
    httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    httpReq.SetBasicAuth(e.AgentClientID, e.AgentSecret)

    resp, err := e.HTTPClient.Do(httpReq)
    if err != nil {
        return nil, fmt.Errorf("token exchange request: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, parseTokenExchangeError(resp)
    }

    // Parse response
    var tokenResp struct {
        AccessToken      string `json:"access_token"`
        ExpiresIn        int    `json:"expires_in"`
        TokenType        string `json:"token_type"`
        IssuedTokenType  string `json:"issued_token_type"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
        return nil, fmt.Errorf("decoding response: %w", err)
    }

    // Decode the act claim from the JWT for audit logging
    actClaim, _ := extractActClaim(tokenResp.AccessToken)

    return &ExchangeResult{
        AccessToken: tokenResp.AccessToken,
        ExpiresIn:   tokenResp.ExpiresIn,
        TokenType:   tokenResp.TokenType,
        ActClaim:    actClaim,
    }, nil
}

// extractActClaim parses the JWT and extracts the act claim without verifying signature.
// For production use, always verify the JWT signature before trusting claims.
func extractActClaim(tokenStr string) (*ActClaim, error) {
    parser := jwt.NewParser(jwt.WithoutClaimsValidation())
    claims := jwt.MapClaims{}
    _, _, err := parser.ParseUnverified(tokenStr, claims)
    if err != nil {
        return nil, err
    }

    actRaw, ok := claims["act"]
    if !ok {
        return nil, nil // No act claim = impersonation mode
    }

    actBytes, _ := json.Marshal(actRaw)
    var act ActClaim
    json.Unmarshal(actBytes, &act)
    return &act, nil
}

// MultiHopExchange demonstrates a user → agent → sub-agent → API delegation chain.
// Each exchange adds a layer to the act claim, creating a full audit trail.
func MultiHopExchange(
    ctx context.Context,
    exchanger *AgentTokenExchanger,
    userToken string,
    agentToken string,
    subAgentToken string,
    targetAudience string,
) (string, error) {
    // Hop 1: User → Agent
    // The agent exchanges the user's token for a scoped agent token
    agentResult, err := exchanger.ExchangeForAgent(ctx, &ExchangeRequest{
        SubjectToken:    userToken,
        ActorToken:      agentToken,
        RequestedScopes: []string{"read:files", "search:web"},
        Audience:        "agent-orchestrator",
    })
    if err != nil {
        return "", fmt.Errorf("hop 1 (user→agent): %w", err)
    }
    // agentResult.ActClaim = { sub: "user@example.com", act: { sub: "coding-agent" } }

    // Hop 2: Agent → Sub-agent
    // The sub-agent exchanges the agent token for its own scoped token
    subAgentResult, err := exchanger.ExchangeForAgent(ctx, &ExchangeRequest{
        SubjectToken:    agentResult.AccessToken,
        ActorToken:      subAgentToken,
        RequestedScopes: []string{"search:web"}, // narrower scope
        Audience:        "web-search-mcp-server",
    })
    if err != nil {
        return "", fmt.Errorf("hop 2 (agent→sub-agent): %w", err)
    }
    // subAgentResult.ActClaim = { sub: "user@example.com",
    //   act: { sub: "coding-agent", act: { sub: "research-subagent" } } }

    // Hop 3: Sub-agent → MCP Server (final exchange for target API)
    finalResult, err := exchanger.ExchangeForAgent(ctx, &ExchangeRequest{
        SubjectToken:    subAgentResult.AccessToken,
        RequestedScopes: []string{"search:execute"}, // narrowest scope
        Audience:        targetAudience,
    })
    if err != nil {
        return "", fmt.Errorf("hop 3 (sub-agent→API): %w", err)
    }

    return finalResult.AccessToken, nil
}
```

### 7.6 Security Considerations for Agent Token Exchange

1. **Strict scope downgrade** — The exchanged token's scopes must be a subset of the subject
  token's scopes. No privilege escalation.

2. **Client authentication required** — Only registered agent clients can perform exchange.
  Use mTLS or Private Key JWT for high-security agents.

3. **may_act claim validation** — The authorization server should verify that the actor is
  authorized to act on behalf of the subject via the `may_act` claim or policy configuration.

4. **Short TTL for exchanged tokens** — Exchanged tokens should have shorter lifetimes than
  the original (e.g., 5-15 minutes), reducing replay risk.

5. **Audit every exchange** — Log all token exchange events with full context for compliance.

---

## 8. OAuth 2.0 for AI Agents (Draft)

### 8.1 The IETF Draft: AI Agent Authentication and Authorization

The IETF has published an internet draft: *"AI Agent Authentication and Authorization"*
(draft-klrc-aiagent-auth-00). This draft proposes a formal model for AI agent auth using
existing standards:

- **WIMSE (Workload Identity in Multi-System Environments)** architecture for agent identity
- **OAuth 2.0 family** of specifications for delegation and authorization
- **Token Exchange (RFC 8693)** for scoped delegation

Key proposals in the draft:

1. Agents have their own workload identities (not just user impersonation)
2. Agents authenticate using OAuth 2.0 client credentials (their own identity)
3. Agents receive delegated authorization via token exchange
4. Agent-to-agent communication uses nested act claims
5. Audit trails must record the full delegation chain

### 8.2 PKCE for Agent Enrollment

PKCE (Proof Key for Code Exchange, RFC 7636) is traditionally for browser-based flows, but
it has a role in agent enrollment:

**Agent enrollment flow:**

1. A new agent is deployed (e.g., a coding assistant on a developer's machine)
2. The agent generates a code verifier (random string) and code challenge (SHA-256 hash)
3. The agent initiates the device authorization flow with PKCE
4. The user authenticates on their phone/browser and approves the agent
5. The agent receives tokens, completing enrollment

```go
// PKCE enrollment for a new agent
func EnrollAgent(ctx context.Context, oauthURL, agentName string) (*AgentCredentials, error) {
    // Step 1: Generate PKCE pair
    verifier, err := generateCodeVerifier()
    if err != nil {
        return nil, err
    }
    challenge := generateCodeChallenge(verifier) // S256

    // Step 2: Initiate device authorization flow
    deviceResp, err := initiateDeviceAuth(ctx, oauthURL, agentName, challenge)
    if err != nil {
        return nil, err
    }

    // Step 3: Display instructions to user
    fmt.Printf("Visit %s and enter code: %s\n",
        deviceResp.VerificationURI, deviceResp.UserCode)

    // Step 4: Poll for token
    token, err := pollForToken(ctx, oauthURL, deviceResp, verifier)
    if err != nil {
        return nil, err
    }

    // Step 5: Store credentials securely
    return &AgentCredentials{
        AgentID:       generateAgentID(),
        RefreshToken:  token.RefreshToken,
        AccessToken:   token.AccessToken,
        EnrolledAt:    time.Now(),
    }, nil
}

func generateCodeVerifier() (string, error) {
    b := make([]byte, 32)
    if _, err := crand.Read(b); err != nil {
        return "", err
    }
    return base64.RawURLEncoding.EncodeToString(b), nil
}

func generateCodeChallenge(verifier string) string {
    h := sha256.Sum256([]byte(verifier))
    return base64.RawURLEncoding.EncodeToString(h[:])
}
```

### 8.3 Device Flow for Agent-to-Human Handoff

When an agent needs human approval for a sensitive action, the device flow provides a clean
mechanism:

```
Agent detects sensitive action (e.g., "delete database")
  → Agent requests elevated scope via device flow
  → User receives notification: "Agent X wants to delete database Y. Approve?"
  → User approves on their device
  → Agent receives scoped token with delete permission
  → Agent performs the action
  → Token expires after one use (or short TTL)
```

This pattern is critical for:
- Destructive operations (delete, overwrite)
- Financial transactions
- Privilege escalation
- Cross-domain data access

### 8.4 Grant Types for Agent Auth

| Grant Type | Use Case | Security |
|-----------|----------|----------|
| `client_credentials` | Agent acting as its own entity | Medium (requires client secret) |
| `urn:ietf:params:oauth:grant-type:token-exchange` | Agent acting on behalf of user | High (scoped, audience-bound) |
| `urn:ietf:params:oauth:grant-type:device_code` | Agent enrollment / consent handoff | High (human-in-the-loop) |
| `urn:ietf:params:oauth:grant-type:jwt-bearer` | Cross-domain agent auth (RFC 7523) | High (signed JWT) |

### 8.5 Go Code: Agent OAuth Client

```go
package agent

import (
    "context"
    "sync"
    "time"
)

// AgentOAuthClient manages OAuth tokens for an AI agent.
// It handles token exchange, refresh, and scoped token requests.
type AgentOAuthClient struct {
    Exchanger    *AgentTokenExchanger
    RefreshToken string
    AccessToken  string
    ExpiresAt    time.Time
    Scopes       []string
    mu           sync.RWMutex
}

// GetScopedToken returns a token scoped for a specific tool/service.
// If the current token doesn't have the required scope, it exchanges for one.
func (c *AgentOAuthClient) GetScopedToken(
    ctx context.Context,
    requiredScopes []string,
    audience string,
) (string, error) {
    c.mu.RLock()
    token := c.AccessToken
    c.mu.RUnlock()

    // Check if we need to exchange for a more specific token
    // (Always exchange for scoped tokens in agent context — defense in depth)
    result, err := c.Exchanger.ExchangeForAgent(ctx, &ExchangeRequest{
        SubjectToken:    token,
        RequestedScopes: requiredScopes,
        Audience:        audience,
    })
    if err != nil {
        return "", fmt.Errorf("scoped token exchange: %w", err)
    }

    return result.AccessToken, nil
}

// CallTool invokes an MCP tool with a properly scoped token.
func (c *AgentOAuthClient) CallTool(
    ctx context.Context,
    toolName string,
    toolEndpoint string,
    payload []byte,
) (*ToolResponse, error) {
    // Look up required scopes for this tool
    scopes, audience := getToolScopes(toolName)

    // Get a scoped token for this specific tool
    token, err := c.GetScopedToken(ctx, scopes, audience)
    if err != nil {
        return nil, fmt.Errorf("getting scoped token for %s: %w", toolName, err)
    }

    // Make the authenticated request
    req, err := http.NewRequestWithContext(ctx, "POST", toolEndpoint, bytes.NewReader(payload))
    if err != nil {
        return nil, err
    }
    req.Header.Set("Authorization", "Bearer "+token)
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-Agent-ID", c.AgentID)
    req.Header.Set("X-Tool-Name", toolName)

    resp, err := c.HTTPClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("tool call %s: %w", toolName, err)
    }
    defer resp.Body.Close()

    // Audit log the tool call
    auditLog(ctx, AuditEntry{
        AgentID:   c.AgentID,
        Tool:      toolName,
        Scopes:    scopes,
        Status:    resp.StatusCode,
        Timestamp: time.Now(),
    })

    return parseToolResponse(resp)
}
```

### 8.6 Agent Consent and Step-Up Authorization

```go
// StepUpAuthorizer handles progressive authorization for agents.
// It determines when a human must approve an action.
type StepUpAuthorizer struct {
    SensitiveActions map[string]bool
    OAuthURL         string
}

// CheckAuthorization determines if the agent can proceed or needs human approval.
func (a *StepUpAuthorizer) CheckAuthorization(
    ctx context.Context,
    agentID string,
    action string,
    currentScopes []string,
) (*AuthorizationDecision, error) {
    if !a.SensitiveActions[action] {
        // Not sensitive — agent can proceed with current scopes
        return &AuthorizationDecision{
            Allowed:  true,
            Reason:   "non-sensitive action",
        }, nil
    }

    // Check if agent already has elevated scope
    requiredScope := getScopeForAction(action)
    if containsScope(currentScopes, requiredScope) {
        return &AuthorizationDecision{
            Allowed:  true,
            Reason:   "already has required scope",
        }, nil
    }

    // Need human approval — initiate device flow
    deviceCode, err := initiateDeviceAuth(ctx, a.OAuthURL, agentID, requiredScope)
    if err != nil {
        return nil, err
    }

    return &AuthorizationDecision{
        Allowed:     false,
        Reason:      "requires human approval",
        ConsentURL:  deviceCode.VerificationURI,
        UserCode:    deviceCode.UserCode,
        DeviceCode:  deviceCode.DeviceCode,
    }, nil
}

type AuthorizationDecision struct {
    Allowed    bool
    Reason     string
    ConsentURL string  // For human-in-the-loop handoff
    UserCode   string
    DeviceCode string
}
```

---

## 9. Agent Identity Model

### 9.1 Representing an Agent in IAM

A first-class agent identity model is essential for production agent auth. The model must
capture:

- **Who** the agent is (identity)
- **What** type of agent it is (classification)
- **Who owns** it (accountability)
- **What** it can do (authorization)
- **How fast** it can do it (rate limiting)
- **What** it has done (audit history)

### 9.2 Go Data Model

```go
package agent

import (
    "time"
    "github.com/google/uuid"
)

// AgentType classifies the kind of AI agent.
type AgentType string

const (
    AgentTypeCodingAssistant  AgentType = "coding_assistant"
    AgentTypeCustomerService  AgentType = "customer_service"
    AgentTypeDataPipeline     AgentType = "data_pipeline"
    AgentTypeAutonomous       AgentType = "autonomous"
    AgentTypeRAGPipeline      AgentType = "rag_pipeline"
    AgentTypeMCPServer        AgentType = "mcp_server"
    AgentTypeSubAgent         AgentType = "sub_agent"
)

// AgentStatus represents the lifecycle state of an agent.
type AgentStatus string

const (
    AgentStatusActive     AgentStatus = "active"
    AgentStatusSuspended  AgentStatus = "suspended"
    AgentStatusRevoked    AgentStatus = "revoked"
    AgentStatusExpired    AgentStatus = "expired"
)

// Agent represents an AI agent registered in the IAM system.
type Agent struct {
    ID          uuid.UUID      `json:"id" db:"id"`
    TenantID    uuid.UUID      `json:"tenant_id" db:"tenant_id"`
    AgentID     string         `json:"agent_id" db:"agent_id"`        // Unique agent identifier
    Name        string         `json:"name" db:"name"`               // Human-readable name
    Description string         `json:"description" db:"description"`

    // Identity
    Type        AgentType      `json:"type" db:"type"`
    Status      AgentStatus    `json:"status" db:"status"`

    // Ownership
    OwnerID     uuid.UUID      `json:"owner_id" db:"owner_id"`       // User who registered the agent
    OwnerEmail  string         `json:"owner_email" db:"owner_email"`

    // Authorization
    Scopes      []string       `json:"scopes" db:"scopes"`           // Allowed scopes
    Audiences   []string       `json:"audiences" db:"audiences"`     // Allowed target services

    // Rate Limiting
    RateLimitPerMinute int     `json:"rate_limit_per_minute" db:"rate_limit_per_minute"`
    RateLimitPerHour   int     `json:"rate_limit_per_hour" db:"rate_limit_per_hour"`
    RateLimitPerDay    int     `json:"rate_limit_per_day" db:"rate_limit_per_day"`

    // Token Configuration
    MaxTokenTTL        time.Duration `json:"max_token_ttl" db:"max_token_ttl"`
    AllowRefresh       bool          `json:"allow_refresh" db:"allow_refresh"`
    RefreshTTL         time.Duration `json:"refresh_ttl" db:"refresh_ttl"`

    // Monitoring
    TrustScore         float64       `json:"trust_score" db:"trust_score"`     // 0.0 to 1.0
    AnomalyThreshold   float64       `json:"anomaly_threshold" db:"anomaly_threshold"`
    AutoRevokeOnAnomaly bool         `json:"auto_revoke_on_anomaly" db:"auto_revoke_on_anomaly"`

    // Sensitive Actions
    SensitiveActions   []string      `json:"sensitive_actions" db:"sensitive_actions"`
    RequireHumanApproval bool        `json:"require_human_approval" db:"require_human_approval"`

    // Metadata
    ClientID           string        `json:"client_id" db:"client_id"`       // OAuth client ID
    Metadata           map[string]any `json:"metadata" db:"metadata"`

    // Timestamps
    CreatedAt          time.Time     `json:"created_at" db:"created_at"`
    UpdatedAt          time.Time     `json:"updated_at" db:"updated_at"`
    LastActiveAt       time.Time     `json:"last_active_at" db:"last_active_at"`
    ExpiresAt          *time.Time    `json:"expires_at" db:"expires_at"`     // Agent registration expiry
}

// AgentAction represents an auditable action performed by an agent.
type AgentAction struct {
    ID              uuid.UUID  `json:"id" db:"id"`
    AgentID         string     `json:"agent_id" db:"agent_id"`
    TenantID        uuid.UUID  `json:"tenant_id" db:"tenant_id"`
    UserID          string     `json:"user_id" db:"user_id"`             // User on whose behalf
    Action          string     `json:"action" db:"action"`               // What was done
    Resource        string     `json:"resource" db:"resource"`           // What was accessed
    Method          string     `json:"method" db:"method"`               // HTTP method
    Scope           string     `json:"scope" db:"scope"`                 // Scope used
    Tool            string     `json:"tool" db:"tool"`                   // MCP tool name
    Result          string     `json:"result" db:"result"`               // success/denied/error
    StatusCode      int        `json:"status_code" db:"status_code"`
    ActChain        string     `json:"act_chain" db:"act_chain"`         // Full delegation chain
    IPAddress       string     `json:"ip_address" db:"ip_address"`
    ReasoningContext string    `json:"reasoning_context" db:"reasoning_context"` // Why the agent did this
    Timestamp       time.Time  `json:"timestamp" db:"timestamp"`
    Duration        int64      `json:"duration_ms" db:"duration_ms"`
}

// AgentRateLimit tracks per-agent rate limiting state.
type AgentRateLimit struct {
    AgentID       string    `json:"agent_id" db:"agent_id"`
    WindowMinute  int       `json:"window_minute" db:"window_minute"`   // Requests in current minute
    WindowHour    int       `json:"window_hour" db:"window_hour"`       // Requests in current hour
    WindowDay     int       `json:"window_day" db:"window_day"`         // Requests in current day
    LastRequestAt time.Time `json:"last_request_at" db:"last_request_at"`
    CircuitOpen   bool      `json:"circuit_open" db:"circuit_open"`     // Circuit breaker state
}

// AgentBehaviorProfile stores baseline behavior for anomaly detection.
type AgentBehaviorProfile struct {
    AgentID              string  `json:"agent_id" db:"agent_id"`
    AvgRequestsPerHour   float64 `json:"avg_requests_per_hour" db:"avg_requests_per_hour"`
    AvgDataVolumePerCall float64 `json:"avg_data_volume_per_call" db:"avg_data_volume_per_call"`
    CommonTools          map[string]float64 `json:"common_tools" db:"common_tools"`       // tool -> frequency
    CommonResources      map[string]float64 `json:"common_resources" db:"common_resources"` // resource -> frequency
    ErrorRate            float64 `json:"error_rate" db:"error_rate"`
    LastProfileUpdate    time.Time `json:"last_profile_update" db:"last_profile_update"`
}
```

### 9.3 Agent Registration Flow

```go
// RegisterAgent creates a new AI agent identity in the IAM system.
func (s *AgentService) RegisterAgent(
    ctx context.Context,
    ownerID uuid.UUID,
    input *RegisterAgentInput,
) (*Agent, error) {
    // Validate owner exists and has permission to register agents
    owner, err := s.userRepo.GetByID(ctx, ownerID)
    if err != nil {
        return nil, fmt.Errorf("owner lookup: %w", err)
    }

    // Create OAuth client for the agent
    clientResult, err := s.oauthService.CreateClient(ctx, &CreateClientInput{
        TenantID:   owner.TenantID,
        Name:       input.Name,
        Type:       ClientTypeConfidential,
        GrantTypes: []string{
            "client_credentials",
            "urn:ietf:params:oauth:grant-type:token-exchange",
            "urn:ietf:params:oauth:grant-type:device_code",
        },
        Scopes:     input.Scopes,
    })
    if err != nil {
        return nil, fmt.Errorf("creating OAuth client: %w", err)
    }

    // Create agent record
    agent := &Agent{
        ID:                  uuid.New(),
        TenantID:            owner.TenantID,
        AgentID:             generateAgentID(input.Name),
        Name:                input.Name,
        Type:                input.Type,
        Status:              AgentStatusActive,
        OwnerID:             ownerID,
        OwnerEmail:          owner.Email,
        Scopes:              input.Scopes,
        Audiences:           input.Audiences,
        RateLimitPerMinute:  input.RateLimitPerMinute,
        RateLimitPerHour:    input.RateLimitPerHour,
        RateLimitPerDay:     input.RateLimitPerDay,
        MaxTokenTTL:         input.MaxTokenTTL,
        AllowRefresh:        input.AllowRefresh,
        TrustScore:          1.0, // Start at full trust
        AnomalyThreshold:    0.7,
        AutoRevokeOnAnomaly: input.AutoRevokeOnAnomaly,
        SensitiveActions:    input.SensitiveActions,
        RequireHumanApproval: input.RequireHumanApproval,
        ClientID:            clientResult.Client.ClientID,
        CreatedAt:           time.Now(),
        UpdatedAt:           time.Now(),
    }

    if err := s.agentRepo.Create(ctx, agent); err != nil {
        return nil, fmt.Errorf("creating agent: %w", err)
    }

    // Log registration
    s.auditLog(ctx, AuditEntry{
        EventType: "agent_registered",
        AgentID:   agent.AgentID,
        UserID:    ownerID.String(),
        Details:   fmt.Sprintf("Agent %s registered by %s", agent.Name, owner.Email),
    })

    return agent, nil
}
```

### 9.4 Agent Token Issuance

```go
// IssueAgentToken issues a scoped token for an agent acting on behalf of a user.
func (s *AgentService) IssueAgentToken(
    ctx context.Context,
    agentID string,
    userToken string,
    requestedScopes []string,
    audience string,
) (*TokenResponse, error) {
    // 1. Look up the agent
    agent, err := s.agentRepo.GetByAgentID(ctx, agentID)
    if err != nil {
        return nil, fmt.Errorf("agent lookup: %w", err)
    }

    // 2. Validate agent is active
    if agent.Status != AgentStatusActive {
        return nil, ErrAgentNotActive
    }

    // 3. Validate requested scopes are within agent's allowed scopes
    if !isSubset(requestedScopes, agent.Scopes) {
        return nil, ErrScopeExceedsAgentPermissions
    }

    // 4. Validate audience is in agent's allowed audiences
    if !contains(agent.Audiences, audience) {
        return nil, ErrAudienceNotAllowed
    }

    // 5. Check rate limits
    if err := s.checkRateLimit(ctx, agent); err != nil {
        return nil, err
    }

    // 6. Perform token exchange
    result, err := s.exchanger.ExchangeForAgent(ctx, &ExchangeRequest{
        SubjectToken:    userToken,
        ActorToken:      agent.ClientCredentials,
        RequestedScopes: requestedScopes,
        Audience:        audience,
    })
    if err != nil {
        return nil, fmt.Errorf("token exchange: %w", err)
    }

    // 7. Update agent's last active time
    s.agentRepo.UpdateLastActive(ctx, agent.ID)

    // 8. Audit log
    s.auditLog(ctx, AuditEntry{
        EventType: "agent_token_issued",
        AgentID:   agent.AgentID,
        Scopes:    requestedScopes,
        Audience:  audience,
    })

    return &TokenResponse{
        AccessToken: result.AccessToken,
        ExpiresIn:   result.ExpiresIn,
        TokenType:   result.TokenType,
    }, nil
}
```

### 9.5 Agent Behavioral Monitoring

```go
// AgentMonitor monitors agent behavior for anomalies.
type AgentMonitor struct {
    agentRepo  AgentRepository
    actionRepo ActionRepository
    alerter    AlertService
}

// CheckAnomaly evaluates whether an agent's recent behavior is anomalous.
func (m *AgentMonitor) CheckAnomaly(
    ctx context.Context,
    agentID string,
) (*AnomalyReport, error) {
    // Get current behavior stats
    recentActions, err := m.actionRepo.GetRecent(ctx, agentID, 24*time.Hour)
    if err != nil {
        return nil, err
    }

    // Get baseline profile
    profile, err := m.getBehaviorProfile(ctx, agentID)
    if err != nil {
        return nil, err
    }

    report := &AnomalyReport{AgentID: agentID}

    // Check request volume anomaly
    currentVolume := float64(len(recentActions))
    if currentVolume > profile.AvgRequestsPerHour*3 {
        report.Anomalies = append(report.Anomalies, Anomaly{
            Type:    "volume_spike",
            Current: currentVolume,
            Baseline: profile.AvgRequestsPerHour,
            Severity: "high",
        })
    }

    // Check new tools being accessed
    for _, action := range recentActions {
        if _, known := profile.CommonTools[action.Tool]; !known && action.Tool != "" {
            report.Anomalies = append(report.Anomalies, Anomaly{
                Type:    "new_tool_access",
                Tool:    action.Tool,
                Severity: "medium",
            })
        }
    }

    // Check error rate spike
    errorCount := 0
    for _, a := range recentActions {
        if a.Result != "success" {
            errorCount++
        }
    }
    errorRate := float64(errorCount) / float64(len(recentActions))
    if errorRate > profile.ErrorRate*2 {
        report.Anomalies = append(report.Anomalies, Anomaly{
            Type:     "error_rate_spike",
            Current:  errorRate,
            Baseline: profile.ErrorRate,
            Severity: "high",
        })
    }

    // Auto-revoke if anomalies exceed threshold
    if len(report.Anomalies) > 0 {
        agent, _ := m.agentRepo.GetByAgentID(ctx, agentID)
        if agent != nil && agent.AutoRevokeOnAnomaly {
            score := calculateAnomalyScore(report)
            if score > agent.AnomalyThreshold {
                m.agentRepo.UpdateStatus(ctx, agent.ID, AgentStatusSuspended)
                m.alerter.Alert(ctx, Alert{
                    AgentID:  agentID,
                    Message:  fmt.Sprintf("Agent %s auto-suspended: anomaly score %.2f > threshold %.2f",
                        agentID, score, agent.AnomalyThreshold),
                    Severity: "critical",
                })
            }
        }
    }

    return report, nil
}
```

---

## 10. GGID Positioning Strategy

### 10.1 Why GGID Is Uniquely Positioned

GGID has several structural advantages for the AI agent auth market:

1. **Go-native** — AI infrastructure (Kubernetes, Terraform, Docker) runs on Go. GGID being
  Go-native means seamless integration with the cloud-native ecosystem where agents operate.

2. **Token Exchange already researched** — GGID has 1293 lines of RFC 8693 design
  documentation (`token-exchange-rfc8693.md`) and 1028 lines of implementation patterns
  (`token-exchange-iam.md`). The design foundation exists.

3. **Multi-tenant from day one** — Agent auth is inherently multi-tenant (each organization
  has its own agents). GGID's tenant isolation model maps naturally.

4. **OAuth 2.1 ready** — GGID already supports OAuth 2.1 patterns, PKCE, resource indicators,
  and the grant types needed for agent auth.

5. **Device flow implemented** — RFC 8628 device authorization flow is already implemented
  server-side, providing the enrollment and consent handoff mechanism agents need.

6. **Audit infrastructure** — NATS JetStream-based audit logging is already in place, capable
  of handling the high-volume per-action audit trail agents require.

### 10.2 The Four-Phase Strategy

#### Phase 1: Token Exchange for Agent Delegation (3-4 weeks)

**Goal**: Implement RFC 8693 token exchange, enabling agents to exchange user tokens for
scoped, audience-bound delegation tokens.

**Deliverables**:
- Token exchange endpoint: `POST /oauth/token` with `grant_type=urn:ietf:params:oauth:grant-type:token-exchange`
- `act` claim support in JWT issuance
- `may_act` claim validation
- Scope downgrade enforcement (exchanged scopes ⊆ subject scopes)
- Audience binding for exchanged tokens
- Audit logging for all exchange events

**Effort**: ~3-4 weeks (1 developer)

**Competitive advantage**: This is the foundational capability. Without it, no other agent
auth feature is possible.

```go
// Phase 1 implementation plan
// 1. Add token exchange grant type to oauth_service.go
// 2. Add act claim to JWT claims struct
// 3. Add scope validation (downgrade only)
// 4. Add audience binding
// 5. Add audit logging
// 6. Add tests
```

#### Phase 2: Agent Registration and Identity Management (4-6 weeks)

**Goal**: First-class agent identity in GGID — agents as a distinct entity type with
registration, lifecycle management, and scoped authorization.

**Deliverables**:
- Agent data model (as defined in Section 9)
- Agent registration API: `POST /api/v1/agents`
- Agent management API: `GET/PUT/DELETE /api/v1/agents/{id}`
- Per-agent OAuth client auto-creation
- Per-agent rate limiting middleware
- Agent lifecycle management (active, suspended, revoked)
- Console UI for agent management
- Agent-scoped token issuance

**Effort**: ~4-6 weeks (1-2 developers)

**Competitive advantage**: Casdoor has this. Keycloak partially has this (as clients). Auth0
does not have this. GGID can leapfrog by combining agent identity with Go-native performance.

#### Phase 3: MCP Server Auth Support (3-4 weeks)

**Goal**: GGID as an OAuth 2.1 authorization server for MCP servers, compliant with the
2026-07-28 spec.

**Deliverables**:
- OAuth 2.0 Protected Resource Metadata endpoint (RFC 9728): `.well-known/oauth-protected-resource`
- Resource Indicator support (RFC 8707) in token requests
- Client ID Metadata Documents (CIMD) support
- MCP server registration and management
- MCP tool-level scope enforcement
- Streamable HTTP transport authentication middleware
- MCP server discovery metadata

**Effort**: ~3-4 weeks (1 developer)

**Competitive advantage**: Aligns with the latest MCP spec. GGID becomes the auth server
for any MCP server, not just GGID's own.

#### Phase 4: Agent Behavior Monitoring and Anomaly Detection (6-8 weeks)

**Goal**: Real-time monitoring of agent actions with automatic revocation on anomalous
behavior.

**Deliverables**:
- Agent behavior profiling (baseline establishment)
- Real-time anomaly detection (volume, tools, error rate)
- Automatic token revocation on anomaly
- Trust scoring system
- Alerting and notification system
- Behavioral audit dashboard in Console
- SIEM integration for agent events

**Effort**: ~6-8 weeks (1-2 developers)

**Competitive advantage**: No competitor has comprehensive agent behavioral monitoring.
This is the differentiator that makes GGID the enterprise-grade choice.

### 10.3 Total Timeline and Effort

| Phase | Duration | Effort | Cumulative |
|-------|----------|--------|------------|
| Phase 1: Token Exchange | 3-4 weeks | 1 dev | 3-4 weeks |
| Phase 2: Agent Identity | 4-6 weeks | 1-2 devs | 7-10 weeks |
| Phase 3: MCP Server Auth | 3-4 weeks | 1 dev | 10-14 weeks |
| Phase 4: Behavior Monitoring | 6-8 weeks | 1-2 devs | 16-22 weeks |
| **Total** | **16-22 weeks** | | **~4-5 months** |

### 10.4 What Makes GGID Unique

| Differentiator | Auth0 | Keycloak | Casdoor | GGID |
|---------------|-------|----------|---------|------|
| Language | Node.js | Java | Go | **Go** |
| Token Exchange (RFC 8693) | Via Actions | Native | No | **Designed, ready to implement** |
| Agent as first-class entity | No | No (client only) | Yes | **Planned (Phase 2)** |
| MCP Server Auth | MCP server only | Via custom impl | Native MCP server | **Planned (Phase 3)** |
| Behavioral Monitoring | No | Event system | No | **Planned (Phase 4)** |
| Multi-tenant | Yes (per tenant) | Realms | Organizations | **Native, tenant-scoped** |
| Device Flow | Yes | Yes | No | **Already implemented** |
| Go-native | No | No | Yes | **Yes** |
| Audit Infrastructure | Logs | Events | Basic | **NATS JetStream (production-grade)** |
| Open Source | No | Yes (Apache 2.0) | Yes (Apache 2.0) | **Yes (Apache 2.0)** |

### 10.5 GGID's Go-Native Advantage

GGID being Go-native provides specific advantages for agent auth:

1. **Performance** — Go's concurrency model (goroutines) is ideal for handling thousands of
  concurrent agent token validations and API calls. Java's JVM overhead and Node.js's
  single-threaded event loop are disadvantages at agent scale.

2. **Deployment** — Go compiles to a single binary. GGID can be deployed as a sidecar to
  MCP servers, as an API gateway plugin, or as a standalone service. No JVM, no npm.

3. **SDK ecosystem** — Go SDKs for AI infrastructure (Kubernetes operators, Terraform
  providers) can embed GGID auth natively. The same cannot be said for Java/Node.js IAM.

4. **gRPC-native** — GGID's gRPC services provide the low-latency, type-safe communication
  that agent orchestration frameworks need.

---

## 11. Gap Analysis & Recommendations

### 11.1 Current GGID State

| Capability | Status |
|-----------|--------|
| OAuth 2.0 Authorization Code Flow | Implemented |
| OAuth 2.1 Patterns (PKCE, Resource Indicators) | Implemented |
| Device Authorization Flow (RFC 8628) | Implemented |
| Client Credentials Grant | Implemented |
| Token Exchange (RFC 8693) | Designed, NOT implemented |
| Agent Identity Model | NOT started |
| MCP Server Auth | NOT started |
| Per-Agent Rate Limiting | NOT started |
| Agent Behavioral Monitoring | NOT started |
| Protected Resource Metadata (RFC 9728) | NOT started |
| CIMD Support | NOT started |
| Act Claim in JWT | NOT started |

### 11.2 Competitive Gap Analysis

#### vs Auth0
- **GGID advantage**: Open source, Go-native, token exchange design ready
- **Auth0 advantage**: Mature ecosystem, MCP server shipped, enterprise SSO breadth
- **Gap to close**: MCP server auth, agent identity model

#### vs Keycloak
- **GGID advantage**: Go-native, lighter weight, modern architecture
- **Keycloak advantage**: Token exchange already implemented, massive enterprise install base
- **Gap to close**: Token exchange implementation, agent identity model

#### vs Casdoor
- **GGID advantage**: Token exchange design, behavioral monitoring planned, audit infrastructure
- **Casdoor advantage**: Native MCP server shipped, agent-first positioning, Casbin policy engine
- **Gap to close**: MCP server auth, agent registration

### 11.3 Action Items

#### Action 1: Implement Token Exchange (RFC 8693)
**Priority**: P0 — Blocking
**Effort**: 3-4 weeks (1 developer)
**Description**: Implement the token exchange grant type in the OAuth service. Add `act` claim
support to JWT issuance. Add scope downgrade enforcement and audience binding. This is the
foundation for all agent auth features.

**Files to modify**:
- `services/oauth/internal/service/oauth_service.go` — Add `ExchangeToken` method
- `services/oauth/internal/server/server.go` — Add token exchange endpoint handler
- `services/oauth/internal/domain/` — Add act claim types
- `pkg/jwt/` — Add act claim support to JWT builder

**Success criteria**:
- `POST /oauth/token` with `grant_type=urn:ietf:params:oauth:grant-type:token-exchange` returns
  scoped, audience-bound token
- Exchanged token has `act` claim when actor_token is provided
- Scope downgrade enforced (exchanged scopes ⊆ subject scopes)
- All exchange events logged to audit

#### Action 2: Add Agent Identity Model
**Priority**: P0 — Blocking
**Effort**: 4-6 weeks (1-2 developers)
**Description**: Create first-class Agent entity in GGID. Implement agent registration,
management, and scoped token issuance. Add per-agent rate limiting middleware.

**Files to create**:
- `services/identity/internal/domain/agent.go` — Agent data model
- `services/identity/internal/service/agent_service.go` — Agent CRUD and token issuance
- `services/identity/internal/repository/agent_repository.go` — Agent storage
- `services/gateway/internal/middleware/agent_rate_limit.go` — Per-agent rate limiting
- Database migration for agent table

**Success criteria**:
- `POST /api/v1/agents` creates a new agent with auto-provisioned OAuth client
- Agent-scoped tokens enforce agent's allowed scopes and audiences
- Per-agent rate limiting prevents abuse
- Console UI shows agent list with status, trust score, last active

#### Action 3: Add MCP Server Auth Support
**Priority**: P1
**Effort**: 3-4 weeks (1 developer)
**Description**: Implement OAuth 2.0 Protected Resource Metadata (RFC 9728), Resource
Indicators (RFC 8707), and CIMD support. Enable GGID to serve as the authorization server
for any MCP server.

**Files to create/modify**:
- `services/oauth/internal/server/well_known.go` — RFC 9728 metadata endpoint
- `services/oauth/internal/service/resource_indicator.go` — RFC 8707 enforcement
- `services/oauth/internal/service/cimd.go` — Client ID Metadata Documents
- `pkg/mcp/` — MCP server auth middleware for Go

**Success criteria**:
- `.well-known/oauth-protected-resource` returns correct metadata
- Token requests with `resource` parameter enforce audience binding
- MCP servers can register with GGID and receive scoped tokens
- Compatible with MCP 2026-07-28 spec

#### Action 4: Add Agent Behavioral Monitoring
**Priority**: P1
**Effort**: 6-8 weeks (1-2 developers)
**Description**: Implement baseline behavior profiling, real-time anomaly detection, automatic
token revocation, and trust scoring for agents.

**Files to create**:
- `services/audit/internal/service/agent_monitor.go` — Behavior analysis engine
- `services/audit/internal/service/anomaly_detector.go` — Anomaly detection algorithms
- `services/audit/internal/service/trust_scorer.go` — Trust score calculation
- Console dashboard for agent behavior visualization

**Success criteria**:
- Agent behavior baselines established after 1000 actions
- Anomalies detected within 5 minutes (MTTD)
- Auto-revoke triggers when anomaly score exceeds threshold
- Trust score visible in Console and included in audit events

#### Action 5: Build MCP Server for GGID Management
**Priority**: P2
**Effort**: 2-3 weeks (1 developer)
**Description**: Build an MCP server that exposes GGID's management APIs to AI agents, similar
to Auth0's MCP server. Agents can create users, manage roles, configure OAuth clients via
natural language.

**Files to create**:
- `services/mcp/cmd/main.go` — MCP server entry point
- `services/mcp/internal/tools/` — Tool definitions (user management, role management, etc.)
- `services/mcp/internal/auth/` — GGID auth integration

**Success criteria**:
- MCP server connectable from Claude Desktop, Cursor, Windsurf
- Tools for user/role/org/client management
- All tool calls audited with agent identity
- Device flow for initial authentication

### 11.4 Priority Matrix

```
        High Impact
            │
     P1     │     P0
  ┌─────────┼─────────┐
  │ Action 3│ Action 1│
  │ MCP Auth│ Token   │
  │         │ Exchange│
  │ Action 5│         │
  │ GGID    │ Action 2│
  │ MCP Srv │ Agent ID│
  └─────────┼─────────┘
     P2     │     P1
            │
          Action 4
          Monitoring
        Low Impact
```

### 11.5 Recommended Sequence

1. **Immediate** (Weeks 1-4): Action 1 — Token Exchange
2. **Short-term** (Weeks 5-10): Action 2 — Agent Identity
3. **Short-term** (Weeks 8-12): Action 3 — MCP Server Auth (can parallel with Action 2)
4. **Medium-term** (Weeks 11-18): Action 4 — Behavioral Monitoring
5. **Medium-term** (Weeks 13-16): Action 5 — GGID MCP Server (can parallel)

### 11.6 Key Metrics to Track

| Metric | Target | Measurement |
|--------|--------|-------------|
| Token exchange implementation | Complete in 4 weeks | Feature-complete + tests passing |
| Agent registration to first token | < 60 seconds | End-to-end timing |
| Agent token validation latency | < 5ms p99 | Gateway metrics |
| Anomaly detection MTTD | < 5 minutes | Monitoring alerts |
| MCP server auth spec compliance | 100% | MCP conformance suite |
| Agent rate limit enforcement | 100% | Gateway middleware tests |

---

## 12. References

### Standards and Specifications

- **RFC 8693** — OAuth 2.0 Token Exchange — [datatracker.ietf.org/doc/html/rfc8693](https://datatracker.ietf.org/doc/html/rfc8693)
- **RFC 8628** — OAuth 2.0 Device Authorization Grant — [datatracker.ietf.org/doc/html/rfc8628](https://datatracker.ietf.org/doc/html/rfc8628)
- **RFC 8707** — Resource Indicators for OAuth 2.0 — [datatracker.ietf.org/doc/html/rfc8707](https://datatracker.ietf.org/doc/html/rfc8707)
- **RFC 9728** — OAuth 2.0 Protected Resource Metadata — [datatracker.ietf.org/doc/html/rfc9728](https://datatracker.ietf.org/doc/html/rfc9728)
- **RFC 9207** — OAuth 2.0 Authorization Server Issuer Identification
- **RFC 7591** — OAuth 2.0 Dynamic Client Registration
- **RFC 7636** — PKCE (Proof Key for Code Exchange)
- **IETF Draft** — AI Agent Authentication and Authorization (draft-klrc-aiagent-auth-00)
- **MCP 2026-07-28** — Model Context Protocol Specification (Release Candidate)

### Industry Research

- **OpenID Foundation** — *"Identity Management for Agentic AI"* (October 2025) — [openid.net](https://openid.net/new-whitepaper-tackles-ai-agent-identity-challenges/)
- **Cloud Security Alliance** — *"Agentic AI Identity Management Approach"* (March 2025) — [cloudsecurityalliance.org](https://cloudsecurityalliance.org/blog/2025/03/11/agentic-ai-identity-management-approach)
- **Obsidian Security** — *"Top AI Agent Security Risks and How to Mitigate Them"* (October 2025)
- **WorkOS** — *"The biggest MCP spec update ships July 28"* (June 2026) — [workos.com/blog](https://workos.com/blog/mcp-2026-spec-agent-authentication)
- **MCP Blog** — *"The 2026 MCP Roadmap"* — [blog.modelcontextprotocol.io](https://blog.modelcontextprotocol.io/posts/2026-mcp-roadmap/)

### Vendor Documentation

- **Auth0** — [Auth0 MCP Server](https://auth0.com/docs/get-started/auth0-mcp-server) | [GitHub](https://github.com/auth0/auth0-mcp-server)
- **Keycloak** — [Keycloak AI Agent Auth Guide](https://skycloak.io/blog/keycloak-ai-agent-authentication/) | [Keycloak + SPIRE](https://github.com/christian-posta/keycloak-agent-identity)
- **Casdoor** — [Casdoor Agent Docs](https://casdoor.ai/docs/agent/overview) | [GitHub](https://github.com/casdoor/casdoor)
- **Aembit** — [Configuring MCP with Auth0](https://aembit.io/blog/configuring-an-mcp-server-with-auth0-as-the-authorization-server/)

### GGID Internal Documents

- [Token Exchange RFC 8693 — Full Design](token-exchange-rfc8693.md) (1293 lines)
- [Token Exchange IAM Patterns](token-exchange-iam.md) (1028 lines)
- [Headless and CLI Authentication](headless-auth-iam.md) (1553 lines)
- [OAuth 2.1 Analysis](oauth-2.1-analysis.md)
- Competitive Gap Audit
- [IAM Differentiation Strategy](iam-differentiation-strategy.md)
- [OAuth Resource Indicators RFC 8707](oauth-resource-indicators-rfc8707.md)

---

## Appendix A: MCP 2026-07-28 Migration Checklist

For teams running MCP servers in production, the migration to 2026-07-28 requires:

1. **Check for session dependencies** — Find every place your server stores state tied to
   `Mcp-Session-Id`. Replace with explicit handles passed as tool arguments.

2. **Update authorization implementation** — Add OAuth 2.1 Protected Resource Metadata
   (RFC 9728). Expose `.well-known/oauth-protected-resource` endpoint. Plan migration from
   Dynamic Client Registration to CIMD.

3. **Update tool schemas** — Full JSON Schema 2020-12 support for tool input schemas.

4. **Check Roots/Sampling/Logging usage** — Roots become Resource URIs. Sampling and Logging
   move out of core spec.

5. **Update MCP client** — Include `MCP-Protocol-Version`, `Mcp-Method`, and `Mcp-Name`
   headers on every request. Use `server/discover` instead of `initialize`. Resource Indicators
   (RFC 8707) required in token requests.

6. **Test multi-instance deployments** — Deploy multiple server instances behind round-robin
   load balancer. Any failing test indicates a hidden session dependency.

## Appendix B: Competitive Feature Comparison Matrix

| Feature | Auth0 | Keycloak | Casdoor | GGID (Current) | GGID (Target) |
|---------|-------|----------|---------|----------------|---------------|
| **OAuth 2.0** | Yes | Yes | Yes | Yes | Yes |
| **OAuth 2.1** | Yes | Yes | Yes | Yes | Yes |
| **OIDC** | Yes | Yes | Yes | Yes | Yes |
| **SAML** | Yes | Yes | Yes | Yes | Yes |
| **Token Exchange (8693)** | Via Actions | Native | No | Designed | Phase 1 |
| **Device Flow (8628)** | Yes | Yes | No | Yes | Yes |
| **Resource Indicators (8707)** | Yes | Yes | No | Yes | Yes |
| **Protected Resource Metadata (9728)** | Yes | Partial | No | No | Phase 3 |
| **CIMD Support** | Yes | Experimental | No | No | Phase 3 |
| **Agent Identity Model** | No | No (client) | Yes | No | Phase 2 |
| **Native MCP Server** | Yes | No | Yes | No | Phase 5 |
| **MCP Server Auth** | Via MCP server | Custom | Yes | No | Phase 3 |
| **Per-Agent Rate Limiting** | No | No | No | No | Phase 2 |
| **Behavioral Monitoring** | No | Events | No | No | Phase 4 |
| **Trust Scoring** | No | No | No | No | Phase 4 |
| **Audit Infrastructure** | Logs | Events | Basic | NATS JetStream | NATS JetStream |
| **Multi-tenant** | Yes | Realms | Orgs | Native | Native |
| **Go-Native** | No | No | Yes | Yes | Yes |
| **Open Source** | No | Yes (Apache) | Yes (Apache) | Yes (Apache) | Yes (Apache) |

## Appendix C: Glossary

| Term | Definition |
|------|-----------|
| **Agent** | An AI-powered system that acts autonomously on behalf of a user, making API calls and decisions |
| **Act Claim** | JWT claim (RFC 8693) recording the actor in a delegation chain |
| **CIMD** | Client ID Metadata Documents — MCP's preferred client registration method |
| **Delegation** | Agent acting on behalf of a user, with identity recorded via act claim |
| **Device Flow** | OAuth grant (RFC 8628) for devices without browsers |
| **Impersonation** | Agent acting as a user without recording actor identity |
| **MCP** | Model Context Protocol — Anthropic's protocol for AI agent ↔ tool communication |
| **MCP Server** | An endpoint exposing tools/resources to AI agents via MCP |
| **May_act** | JWT claim authorizing which actors can act on behalf of a subject |
| **Resource Indicator** | RFC 8707 parameter specifying token target audience |
| **STS** | Security Token Service — exchanges one token for another |
| **Token Exchange** | RFC 8693 mechanism for exchanging one OAuth token for another |
| **WIMSE** | Workload Identity in Multi-System Environments — IETF architecture for workload identity |

---

*End of document — 1000+ lines of competitive analysis on AI Agent Identity and MCP Authentication.*

*This document is a living research artifact. Update as the MCP spec, IETF drafts, and vendor
products evolve.*
