# OpenID Shared Signals Framework (SSF) 1.0 FINAL — Implementation Gap

> **Status:** SSF 1.0 + CAEP 1.0 finalized September 2025. GGID has proprietary CAE (Redis JTI blocklist) but NOT the standardized SET transmitter/receiver. This is the #1 interoperability gap for enterprise federations.

---

## 1. What Changed

The OpenID Foundation published the **final** Shared Signals Framework (SSF) 1.0 and CAEP 1.0 specifications in September 2025. This is no longer draft — it's a stable standard.

**Key specs:**
- [SSF 1.0 Final](https://openid.net/specs/openid-sharedsignals-framework-1_0-final.html) — SET transport, stream management, verification
- CAEP 1.0 — session-level events (session-revoked, assurance-level-change, device-compliance-change, token-claims-change)
- RISC — account-level events (account-disabled, identifier-changed, credential-change-required)

## 2. GGID Current State

| Capability | GGID Implementation | Standard Compliant? |
|-----------|---------------------|---------------------|
| Session revocation | Redis JTI ZSET blocklist + NATS events | NO — proprietary format, not SET |
| Audit event emission | NATS JetStream (internal) | NO — internal only, no external SET stream |
| Stream management | None | NO — no /sse/streams API |
| Event verification | None | NO — no SET JWT signing or verification |
| Poll/Push delivery | None | NO — no SET delivery endpoint |

## 3. Gap: SET Format

GGID emits JSON audit events to NATS. SSF requires **Security Event Tokens (SETs)** — JWT-signed JSON events with standard envelope:

```json
{
  "iss": "https://ggid.example.com",
  "sub": "user-uuid",
  "aud": "rp-client-id",
  "iat": 1718000000,
  "jti": "event-uuid",
  "events": {
    "https://schemas.openid.net/secevent/caep/event-type/session-revoked": {
      "subject": {"format": "iss_sub", "iss": "...", "sub": "..."},
      "event_timestamp": 1718000000
    }
  }
}
```

## 4. Gap: Stream Management API

SSF requires:
- `POST /sse/streams` — create stream configuration
- `GET /sse/streams/{id}` — stream status
- `POST /sse/streams/{id}/verification` — verification flow
- `POST /sse/streams/{id}/add-subject` / `remove-subject` — subject management

## 5. Recommended Implementation Path

1. **SET envelope** — wrap existing audit events in SET JWT format (pkg/audit/set.go)
2. **Stream management API** — new endpoints in audit service for SSF stream CRUD
3. **Delivery** — push (HTTP POST to configured endpoints) and poll modes
4. **Verification** — add verification event flow to prevent event loss
5. **CAEP mapping** — map existing ITDR detections → CAEP session-revoked events
6. **RISC mapping** — map account lifecycle events → RISC event types

## 6. Priority

**P1** — Enterprise customers in federated environments require SSF for interop. GGID's CAE implementation already handles the logic (session revocation, threat detection), just needs standardization of the output format.

## 7. Competitive Landscape

- Auth0: SSF/CAEP support announced 2025
- Okta: Native SSF support (they helped author the spec)
- Keycloak: Community discussion open, not yet implemented
- Ping Identity: SSF transmitter + receiver
