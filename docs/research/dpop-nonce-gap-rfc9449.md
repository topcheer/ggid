# DPoP Nonce Support Gap (RFC 9449)

> **Status:** GGID has basic DPoP token binding (dpop_token_bind.go) but does NOT implement the server-issued nonce mechanism required by RFC 9449 for replay protection. Without nonce support, stolen DPoP proof JWTs can be replayed within their validity window.

---

## 1. RFC 9449 Nonce Mechanism

RFC 9449 specifies that the authorization server **MAY** issue a `DPoP-Nonce` response header — a server-chosen nonce value that the client must include in subsequent DPoP proof JWTs (in the `nonce` claim).

**The flow:**
1. Client sends request with DPoP proof JWT
2. Server responds with `DPoP-Nonce: <server-nonce>` header
3. Client must include this nonce in the next DPoP proof JWT's `nonce` claim
4. Server validates: proof JWT `nonce` claim matches its last-issued nonce for that client
5. If mismatch → `use_dpop_nonce` error, new nonce issued

**Why it matters:** Without nonce, a DPoP proof JWT is valid for ~60 seconds and can be replayed by a man-in-the-middle within that window. The nonce mechanism makes each proof JWT single-use.

## 2. GGID Current State

| Component | Status | Notes |
|-----------|--------|-------|
| DPoP token binding | `dpop_token_bind.go` | In-memory map, binds access_token to `jkt` thumbprint |
| DPoP config handler | `dpop_config_handler.go` | Returns require_dpop + stats |
| DPoP proof validation | None | No JWS signature verification of DPoP header |
| DPoP nonce issuance | None | Missing entirely |
| DPoP nonce validation | None | Missing entirely |

## 3. Gap Analysis

GGID stores the `jkt` binding but **never validates the DPoP proof JWT** sent in the `DPoP` HTTP header. This means:
- The `cnf.jkt` claim in the access token is set but not verified on resource access
- No `htm` (HTTP method) and `htu` (HTTP URI) claim validation
- No `iat` (issued at) freshness check
- No `jti` (JWT ID) replay prevention
- No nonce mechanism at all

## 4. Auth0 / Okta Implementation Reference

Auth0 implements full RFC 9449 including:
- `DPoP-Nonce` header on every token response
- `use_dpop_nonce` error code when nonce mismatch
- Proof JWT validation: `htm`, `htu`, `iat` (±60s), `jti` uniqueness check
- Per-client nonce tracking in Redis

## 5. Recommended Implementation Path

1. **Middleware** (`pkg/middleware/dpop.go`):
   - Parse `DPoP` header → verify JWS signature
   - Validate `htm`, `htu`, `iat` freshness
   - Check `jti` against Redis replay cache
   - Validate `nonce` claim against per-client nonce

2. **Nonce issuance** (OAuth service):
   - Generate random nonce on each token response
   - Return in `DPoP-Nonce` response header
   - Store in Redis: `dpop:nonce:{client_id}` with 5min TTL

3. **Token binding verification**:
   - On resource access, check `cnf.jkt` in access token
   - Match against `DPoP` header proof JWT's `jkt`

4. **Tests**: DPoP proof generation helper for test clients

## 6. Priority

**P2** — DPoP without nonce is better than no DPoP (sender constraining still prevents token replay from different clients), but the nonce gap reduces the security guarantee to a ~60s replay window. Fix when DPoP is actively used by SDK consumers.

## 7. Related

- [RFC 9449](https://datatracker.ietf.org/doc/html/rfc9449)
- [WorkOS DPoP Guide](https://workos.com/blog/dpop-rfc-9449-explained) — good nonce dance walkthrough
- [Separating DPoP for Access vs Refresh Tokens](https://yaroslavros.github.io/oauth-dpop-rt/) — draft extension for different binding keys
