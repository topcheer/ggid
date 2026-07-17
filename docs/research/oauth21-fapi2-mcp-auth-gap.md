# OAuth 2.1 + FAPI 2.0 + MCP Auth — GGID Gap Analysis

> Research covering OAuth 2.1 consolidation (draft-15), FAPI 2.0 Final (Feb 2025), and AI Agent/MCP authentication patterns.

---

## 1. OAuth 2.1 (draft-ietf-oauth-v2-1-15)

**Status:** Draft 15, not yet RFC. Core requirements are well-defined and industry-standard:

| Requirement | Industry Standard | GGID Status |
|-------------|------------------|-------------|
| PKCE mandatory for all clients | YES | **PARTIAL** — PKCE enforced when `RequirePKCE=true` or public clients. Not globally mandatory by default. |
| Implicit flow removed | YES | **DONE** — No `response_type=token` handler found. Implicit flow not supported. |
| Redirect URI exact string matching | YES | **NEEDS VERIFICATION** — should reject wildcard matching |
| Refresh token rotation | YES | **DONE** — implemented in auth_service.go:288 |
| Resource owner password grant removed | YES | **NEEDS VERIFICATION** — check if `grant_type=password` still accepted |

**Gap:** PKCE should be globally mandatory (OAuth 2.1 requirement). Currently configurable per-client. Should default to required with opt-out only for trusted confidential clients.

## 2. FAPI 2.0 Security Profile (Final, Feb 2025)

**Status:** Final specification published February 2025. Conformance tests available.

**GGID Status:** **PARTIALLY IMPLEMENTED**
- `enforceFAPIAuthorize()` and `enforceFAPIToken()` exist (server.go:446, 546)
- FAPI config endpoint at `/api/v1/oauth/fapi-config` (server.go:1553)
- DPoP token binding exists (dpop_token_bind.go)

**Gap:** FAPI 2.0 requires:
1. **Sender-constrained tokens** (DPoP or mTLS) — GGID has DPoP but without nonce (tracked separately)
2. **RAR (Rich Authorization Requests)** — `authorization_details` parameter for granular, typed authorization. GGID does NOT implement RAR.
3. **JARM (JWT-Secured Authorization Response Mode)** — GGID has JWT responses but needs explicit JARM mode support
4. **PAR mandatory** — GGID has PAR but doesn't enforce it for FAPI clients

## 3. AI Agent Identity / MCP Auth

**Industry Direction (2025):** MCP (Model Context Protocol) adopts OAuth 2.1 as its auth framework. Key patterns:
- AI agents register as OAuth clients (machine-to-machine)
- Token scoping per agent capability (read-only, specific tools)
- On-behalf-of (OBO) flows for delegated user→agent access
- Token exchange (RFC 8693) for agent chaining

**GGID Status:** **STRONG**
- Agent registration: `/api/v1/agents/register`, `/api/v1/agents/list` (server.go:1658-1694)
- Agent token issuance: `/api/v1/agents/token` (server.go:1703)
- MCP server scoping: `mcp_servers` field in agent registration (server.go:1721)
- Token exchange: RFC 8693 implemented (server.go:614)
- Delegation chains: PG-backed (delegation_pg.go)

**Gap:** No agent token revocation API. No agent audit trail (which MCP servers accessed). No rate limiting per agent.

## Summary: New Backlog Items

1. **[P1] Make PKCE globally mandatory** — OAuth 2.1 compliance. Default `RequirePKCE=true`, allow opt-out only for trusted confidential clients.
2. **[P2] RAR (Rich Authorization Requests)** — `authorization_details` parameter support. Required for FAPI 2.0 compliance.
3. **[P2] Agent Token Revocation + Audit** — Revoke agent tokens, audit MCP server access per agent.
4. **[P3] PAR mandatory for FAPI clients** — Enforce PAR when FAPI profile is enabled.
