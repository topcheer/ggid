# RFC 8693 Token Exchange — Standard Grant Gap Analysis

> **Status**: Gap confirmed — GGID has a custom delegation endpoint but NOT the standard token-exchange grant.
> **Date**: 2026-07-17
> **Priority**: P1 (AI agent delegation is a strategic scenario; MCP ecosystem expects the standard grant)

---

## 1. Background

RFC 8693 (OAuth 2.0 Token Exchange, January 2020) defines a standard way to
exchange one token for another at the token endpoint. Two primary patterns:

- **Delegation** — an actor (e.g., an AI agent) acts *on behalf of* a subject
  (user). Resulting token carries `act` (actor) claims, possibly nested.
- **Impersonation** — the actor effectively becomes the subject; no `act`
  claim. Rarely recommended.

In 2025-2026 this became the de-facto standard for AI agent authorization:

- **MCP (Model Context Protocol)** authorization spec references OAuth 2.1
  flows; agent-to-tool-server delegation maps directly to RFC 8693 token
  exchange with `actor_token`.
- **Enterprise agent platforms** (Salesforce Agentforce, Microsoft Copilot
  Studio, AWS Bedrock Agents) all converge on "scoped-down token derived from
  user token" — exactly the token-exchange pattern.
- **IETF draft-ietf-oauth-identity-assertion-authz-grant** builds on RFC 8693
  for cross-domain agent identity.

## 2. The Standard Grant (what GGID is missing)

RFC 8693 Section 2.1 — the exchange happens at the **standard token endpoint**
with a dedicated grant type:

```
POST /oauth/token
Content-Type: application/x-www-form-urlencoded

grant_type=urn:ietf:params:oauth:grant-type:token-exchange
&subject_token=<user access token>
&subject_token_type=urn:ietf:params:oauth:token-type:access_token
&actor_token=<agent token>            (optional, for delegation)
&actor_token_type=urn:ietf:params:oauth:token-type:access_token
&scope=read:orders                    (requested narrower scope)
&resource=https://api.example.com/    (target service, RFC 8707)
```

Response:

```json
{
  "access_token": "...",
  "issued_token_type": "urn:ietf:params:oauth:token-type:access_token",
  "token_type": "Bearer",
  "expires_in": 300
}
```

Key requirements GGID must honor:

| Requirement | RFC section | Notes |
|-------------|-------------|-------|
| Validate subject_token (signature, expiry, revocation) | 2.1 | Must use existing introspection path |
| `act` claim nesting for delegation chains | 4.1 | `{"act": {"sub": "agent", "act": {...}}}` |
| Scope reduction only — never widen beyond subject's scope | Security cons. | Critical security invariant |
| `issued_token_type` in response | 2.2 | Signals resulting token type |
| Error `invalid_target` when resource/audience is invalid | 2.2.1 | Distinct error code |
| Client must be authorized to perform exchange | 4 | Per-client policy flag |

## 3. Current GGID State

| Component | Status |
|-----------|--------|
| Custom endpoint `/api/v1/oauth/token-exchange-delegation` | EXISTS — accepts JSON `{subject_token, actor_token, scope, reason}`, stores delegation chain in **in-memory map** (`delegationChains`) |
| Standard grant `urn:ietf:params:oauth:grant-type:token-exchange` in `/oauth/token` | **MISSING** — the grant switch in server.go does not include it |
| `act` claim in issued JWTs | Partial (custom endpoint path only) |
| Scope-reduction enforcement | Unknown / unverified |
| Delegation chain persistence | **In-memory** — lost on restart (same class of issue as the 6 PG-persistence fixes in Rounds 5-6) |
| Per-client token-exchange authorization | MISSING |

## 4. Gap Analysis

1. **Standards compliance** — MCP clients and generic OAuth libraries
   (e.g., `oauth4webapi`, AppAuth) expect the standard grant at the token
   endpoint. A custom JSON endpoint requires bespoke client code.
2. **Persistence** — `delegationChains` is an in-memory `sync.Map`; pod
   restart loses all chain records, breaking audit and revocation.
3. **Security invariant** — scope widening must be rejected. Needs explicit
   test: request scope superset of subject token scope → expect error.
4. **Revocation propagation** — revoking the subject token should invalidate
   derived exchanged tokens (chain revocation).

## 5. Recommended Implementation

**Phase 1 (Backend, P1)** — standard grant support:
1. Add `case "urn:ietf:params:oauth:grant-type:token-exchange"` to the
   `/oauth/token` grant switch in `services/oauth/internal/server/server.go`.
2. Implement `service.TokenExchange(ctx, req)` in oauth service:
   - Validate subject_token via existing token verification (JWKS + Redis
     revocation check).
   - If actor_token present: validate it, build `act` claim (nest if the
     subject token already has `act`).
   - Enforce scope ⊆ subject scope; reject `invalid_scope` otherwise.
   - Honor `resource` param (RFC 8707) for audience restriction.
   - Add `issued_token_type` to response.
3. Per-client policy: add `token_exchange_allowed` boolean to client
   registration; reject exchange for unauthorized clients.

**Phase 2 (Backend, P2)** — persistence + revocation:
4. Migrate `delegationChains` to PostgreSQL (follow the backup_codes_pg.go
   pattern: `EnsureSchema` at startup).
5. Chain revocation: when a subject token is revoked, mark all descendant
   exchanged tokens revoked (store parent token jti with each exchange).

**Phase 3 (SDK, P2)**:
6. Add `TokenExchange()` to Go/Node/Python SDKs (standard form-encoded
   request against `/oauth/token`).

## 6. Effort Estimate

| Phase | Files | Effort |
|-------|-------|--------|
| 1 | server.go (grant case), oauth_service.go (TokenExchange), client model | ~1 day |
| 2 | token_exchange_pg.go (new), revoke path | ~0.5 day |
| 3 | sdk/go/client.go, sdk/node/src, sdk/python | ~0.5 day |

## 7. References

- RFC 8693 — OAuth 2.0 Token Exchange
- RFC 8707 — Resource Indicators for OAuth 2.0
- draft-ietf-oauth-identity-assertion-authz-grant (enterprise agent identity)
- MCP Authorization specification (OAuth 2.1 based)
- Existing research: docs/research/token-exchange-rfc8693.md (design doc),
  docs/research/ai-agent-identity-mcp.md
