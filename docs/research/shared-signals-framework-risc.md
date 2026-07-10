# RISC Events: GGID Audit & Logout Integration

> **Scope:** RISC event types, GGID audit-event-to-RISC mapping, logout integration, multi-tenant streaming. For SET format, push/poll delivery, and stream management, see [Shared Signals Framework](shared-signals-framework.md).

---

## 1. RISC Overview

**RISC** (Risk and Incident Sharing and Coordination) is a profile within the OpenID **Shared Signals and Events (SSE)** framework. While SSE defines the SET transport layer, RISC defines **account-lifecycle security event types** that identity providers and relying parties exchange to coordinate responses to security incidents.

**Event URI prefix:** `urn:ietf:params:sse:events:risc:` (events append a type identifier, e.g. `account-disabled`).

**Relationship to CAEP:** RISC and CAEP (Continuous Access Evaluation Profile) are complementary, not competing:
- **RISC** — account-level lifecycle: creation, deletion, disabling, credential changes, identifier changes, recovery flows.
- **CAEP** — session-level and access-level: session revocation, assurance-level changes, device compliance, claims updates.

Both use the same SET envelope and SSE delivery. RISC answers "what happened to this account?" while CAEP answers "what is the current authorization state of this session?"

**Why GGID needs RISC:** GGID manages the full account lifecycle (register, auth, MFA, password reset, lock/unlock, delete). Each action is a RISC event trigger. Downstream relying parties need real-time notification to revoke cached sessions, purge local user data, block access for disabled accounts, and update identifier mappings. GGID already emits structured audit events to NATS JetStream — RISC provides the standardized taxonomy to expose them externally.

---

## 2. RISC Event Types

| Event Type | Trigger | Payload Fields | Receiver Action |
|------------|---------|---------------|-----------------|
| `account-deleted` | Account permanently removed | `subject`, `event_timestamp`, `reason` (`user_initiated`/`admin_initiated`/`policy`) | Remove local references, revoke all tokens, purge cached profile data |
| `account-disabled` | Account suspended (compromise, policy violation) | `subject`, `reason` (`compromised`/`policy`/`violation`) | Block new sessions, keep data for investigation |
| `account-enabled` | Account re-enabled after disable | `subject`, `reason` (`investigation_cleared`/`admin_action`) | Allow new sessions, notify user, clear blocked flags |
| `credential-change` | Password reset, MFA enroll/remove, passkey added | `subject`, `credential_type` (`password`/`totp`/`webauthn`/`oauth`/`backup_code`), `change_type` (`added`/`removed`/`changed`) | Password changed: invalidate sessions + refresh tokens. MFA removed: flag elevated risk, require re-consent |
| `sessions-revoked` | All sessions invalidated (global logout, incident) | `subject`, `reason` (`security_incident`/`user_request`/`admin_action`) | Purge session cache, force re-authentication |
| `tokens-revoked` | All access/refresh tokens invalidated | `subject`, `token_type` (`access`/`refresh`/`all`) | Add to revocation list, reject future presentations within TTL |
| `identifier-changed` | User email or phone changed | `subject`, `old_identifier`, `new_identifier`, `identifier_type` (`email`/`phone`) | Update local user-to-identifier mapping, verify new identifier |
| `recovery-activated` | Account recovery flow started | `subject`, `recovery_method` (`email`/`admin`/`recovery_code`) | Flag account for monitoring, require step-up auth |

> **Note:** CAEP also defines `credential-change`. Both profiles share the same URI. Receivers should handle it regardless of source profile.

---

## 3. RISC vs CAEP Comparison

| Aspect | RISC | CAEP |
|--------|------|------|
| **Event scope** | Account lifecycle (create, disable, delete, credential, identifier) | Session/access (session-revoked, assurance-level-change, device-compliance) |
| **Typical sender** | Identity Provider (account manager) | Identity Provider or Relying Party |
| **Typical receiver** | Relying Party (resource provider) | Relying Party or Identity Provider |
| **URI prefix** | `urn:ietf:params:sse:events:risc:` | `urn:ietf:params:sse:events:caep:` |
| **Overlapping events** | `credential-change` (primary definer) | `session-revoked` (primary definer) |
| **Directionality** | Predominantly IdP → RP | Bidirectional (IdP ↔ RP) |

**Recommendation:** GGID should support both profiles. Overlapping events (`credential-change`, `session-revoked`) should be mapped to both URI prefixes since receivers may subscribe to only one.

---

## 4. GGID Audit Integration

GGID emits structured audit events to NATS JetStream via `pkg/audit.Publisher`. The `Event` struct (`Action`, `ActorID`, `TenantID`, `Result`, `Metadata`) has all fields needed to construct RISC SET payloads.

### Audit Event → RISC Mapping

| GGID Audit Action | RISC Event | Receiver Action |
|-------------------|------------|-----------------|
| `user.deleted` | `risc:account-deleted` | Purge local user data, revoke tokens |
| `user.disabled` | `risc:account-disabled` | Block sessions, keep data for forensics |
| `user.enabled` | `risc:account-enabled` | Allow new sessions, notify user |
| `auth.password_changed` | `risc:credential-change` | Invalidate sessions and refresh tokens |
| `auth.mfa_enrolled` | `risc:credential-change` | Informational; no session invalidation |
| `auth.mfa_removed` | `risc:credential-change` | Flag elevated risk, require re-consent |
| `session.revoked` | `risc:sessions-revoked` | Purge session cache, force re-auth |
| `token.revoked` | `risc:tokens-revoked` | Add to revocation list |
| `auth.login_failed` (threshold) | custom `risc:risk-detected` | Rate-limit, step-up auth, temporary lockout |
| `org.user_removed` | custom `risc:membership-revoked` | Remove user access to org-scoped resources |

### Go: Audit-to-RISC Bridge

Subscribes to the GGID audit NATS stream, transforms relevant events into SET JWTs, publishes to SSF transmitter for external receivers.

```go
type AuditToRISCBridge struct {
	js         jetstream.JetStream
	auditSub   string        // "audit.events"
	signingKey []byte        // HMAC or RSA key for SET signing
	dedup      Deduplicator
}

var riscEventMapping = map[string]string{
	"user.deleted":          "urn:ietf:params:sse:events:risc:account-deleted",
	"user.disabled":         "urn:ietf:params:sse:events:risc:account-disabled",
	"user.enabled":          "urn:ietf:params:sse:events:risc:account-enabled",
	"auth.password_changed": "urn:ietf:params:sse:events:risc:credential-change",
	"auth.mfa_enrolled":     "urn:ietf:params:sse:events:risc:credential-change",
	"auth.mfa_removed":      "urn:ietf:params:sse:events:risc:credential-change",
	"session.revoked":       "urn:ietf:params:sse:events:risc:sessions-revoked",
	"token.revoked":         "urn:ietf:params:sse:events:risc:tokens-revoked",
}

func (b *AuditToRISCBridge) TransformEvent(ae audit.Event) (*SET, error) {
	riscURI, ok := riscEventMapping[ae.Action]
	if !ok {
		return nil, nil // unmapped — skip
	}
	subject := fmt.Sprintf("urn:ggid:user:%s", ae.ActorID)
	events := map[string]map[string]any{
		riscURI: {
			"subject": subject, "event_timestamp": ae.CreatedAt.Unix(),
			"tenant_id": ae.TenantID.String(), "originating_event": ae.Action,
		},
	}
	// Enrich credential-change events.
	if ae.Action == "auth.password_changed" {
		events[riscURI]["credential_type"] = "password"
		events[riscURI]["change_type"] = "changed"
	}
	if strings.HasPrefix(ae.Action, "auth.mfa_") {
		events[riscURI]["credential_type"] = ae.Metadata["credential_type"]
		events[riscURI]["change_type"] = map[bool]string{true: "added", false: "removed"}[ae.Action == "auth.mfa_enrolled"]
	}
	return &SET{
		JTI: uuid.NewString(), ISS: "https://ggid.example.com",
		AUD: []string{"ggid-risc-receivers"}, IAT: ae.CreatedAt.Unix(),
		Subject: subject, Events: events,
	}, nil
}

func (b *AuditToRISCBridge) PublishSET(ctx context.Context, set *SET) error {
	if b.dedup != nil {
		if seen, _ := b.dedup.IsDuplicate(ctx, set.JTI); seen {
			return nil
		}
	}
	token, err := set.Sign(b.signingKey)
	if err != nil {
		return fmt.Errorf("sign SET: %w", err)
	}
	subject := fmt.Sprintf("risc.events.%s", set.Subject)
	_, err = b.js.Publish(ctx, subject, []byte(token))
	return err
}

// Start subscribes to the audit stream and bridges events to RISC.
func (b *AuditToRISCBridge) Start(ctx context.Context) error {
	cons, err := b.js.CreateOrUpdateConsumer(ctx, "AUDIT", jetstream.ConsumerConfig{
		Name: "risc-bridge", Durable: "risc-bridge",
		FilterSubject: b.auditSub, AckPolicy: jetstream.AckExplicitPolicy,
	})
	if err != nil {
		return err
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			batch, err := cons.FetchNoWait(10)
			if err != nil {
				time.Sleep(500 * time.Millisecond)
				continue
			}
			for msg := range batch.Messages() {
				var ae audit.Event
				if err := json.Unmarshal(msg.Data(), &ae); err != nil {
					msg.Ack()
					continue
				}
				if set, _ := b.TransformEvent(ae); set != nil {
					_ = b.PublishSET(ctx, set)
				}
				msg.Ack()
			}
		}
	}()
	return nil
}
```

---

## 5. GGID Logout Integration

### GGID as Sender (logout → RISC events)

| GGID Logout Action | Emitted Event | Scope |
|--------------------|---------------|-------|
| Single session logout (`POST /oauth/revoke`) | `caep:session-revoked` | One session |
| Global logout (all sessions) | `risc:sessions-revoked` | All sessions |
| Token revocation (access + refresh) | `risc:tokens-revoked` | All tokens |

Already partially covered by `session.revoked` and `token.revoked` audit actions — the bridge (Section 4) maps them automatically.

### GGID as Receiver (inbound RISC events)

GGID acts as a RISC receiver for events from external/federated IdPs:

| Inbound Event | Source Example | GGID Action |
|---------------|----------------|-------------|
| `risc:account-disabled` | Okta, Azure AD | Disable user in GGID, block login |
| `risc:sessions-revoked` | Azure AD | Purge all GGID sessions for subject |
| `risc:tokens-revoked` | Okta | Invalidate GGID-issued tokens for subject |
| `risc:credential-change` | External IdP | Require re-auth, flag for step-up |

### Go: RISC Event Handler

```go
type RISCHandler interface {
	HandleAccountDisabled(ctx context.Context, subject, reason string) error
	HandleSessionsRevoked(ctx context.Context, subject, reason string) error
	HandleTokensRevoked(ctx context.Context, subject, tokenType string) error
	HandleCredentialChange(ctx context.Context, subject, credType, changeType string) error
}

type DefaultRISCHandler struct {
	authService  AuthServiceInterface
	sessionStore SessionStore
	tokenRevoker TokenRevoker
}

func (h *DefaultRISCHandler) HandleAccountDisabled(ctx context.Context, subject, reason string) error {
	userID := parseSubject(subject) // urn:ggid:user:{uuid}
	return h.authService.LockUser(ctx, userID, fmt.Sprintf("risc:account-disabled: %s", reason))
}

func (h *DefaultRISCHandler) HandleSessionsRevoked(ctx context.Context, subject, reason string) error {
	return h.sessionStore.PurgeByUser(ctx, parseSubject(subject))
}

func (h *DefaultRISCHandler) HandleTokensRevoked(ctx context.Context, subject, tokenType string) error {
	return h.tokenRevoker.RevokeAllForUser(ctx, parseSubject(subject), tokenType)
}

func (h *DefaultRISCHandler) HandleCredentialChange(ctx context.Context, subject, credType, changeType string) error {
	if credType == "password" && changeType == "changed" {
		return h.sessionStore.PurgeByUser(ctx, parseSubject(subject))
	}
	return nil // informational for other credential types
}
```

---

## 6. Event Deduplication

The same event may arrive via multiple channels (direct RISC feed + federated SAML/OIDC logout). Without dedup, downstream systems process twice, causing duplicate revocations.

- **Key:** `jti` (SET JWT ID) + event type URI + subject
- **Storage:** Redis `SETNX` with 24h TTL
- **Idempotent handlers:** Safe to process the same event twice (revoking an already-revoked token is a no-op)

```go
type RedisDeduplicator struct {
	rdb *redis.Client
	ttl time.Duration
}

func (d *RedisDeduplicator) IsDuplicate(ctx context.Context, jti string) (bool, error) {
	key := fmt.Sprintf("risc:dedup:%s", jti)
	added, err := d.rdb.SetNX(ctx, key, "1", d.ttl).Result()
	if err != nil {
		return false, err
	}
	return !added, nil // added==false → key existed → duplicate
}
```

---

## 7. Multi-Tenant Considerations

GGID is multi-tenant. RISC events must be strictly scoped to prevent cross-tenant leakage.

- **Event scoping:** Every SET includes `tenant_id`. Handlers verify tenant before acting — a `account-disabled` for tenant A must not affect tenant B users.
- **NATS subject hierarchy:** Tenant-scoped subjects enable per-tenant consumers and messaging-layer access control:
  ```
  risc.events.{tenant_id}.{event_type}
  ```
  Example: `risc.events.00000000-0000-0000-0000-000000000001.account-disabled`
- **SET audience:** The `aud` claim is scoped per-tenant receiver config. Each tenant has its own authorized RISC receiver list.
- **Subject format:** SET subjects include tenant context: `urn:ggid:tenant:{tenant_id}:user:{user_id}` — prevents collision across tenants.
- **Per-tenant streams:** High-volume tenants get dedicated JetStream consumers with independent rate limits and retention policies.

---

## 8. Implementation Roadmap

| Phase | Scope | Effort | Dependencies |
|-------|-------|--------|--------------|
| **Phase 1** | Map audit events to RISC URIs. Implement `AuditToRISCBridge` — subscribe to audit NATS, transform to SETs, sign, publish. | ~2 weeks | SET signing keys, transmitter stream config |
| **Phase 2** | Inbound RISC receiver. `RISCHandler` interface + `DefaultRISCHandler`. HTTP endpoint for SET ingestion. Event dedup via Redis. | ~1 week | Redis, auth service lock/purge APIs |
| **Phase 3** | Multi-tenant event stream management. Per-tenant NATS subjects, audience scoping, receiver registration API. | ~1 week | Tenant-scoped JetStream consumers |
| **Phase 4** | Webhook delivery in addition to NATS. SSF push delivery with retry, receiver health monitoring. | ~1 week | Webhook registry, HTTP client with backoff |

**Phase 1** is self-contained: reads existing audit events, emits standardized RISC SETs without changing GGID service behavior. Phases 2–4 build incrementally.

---

## References

- [OpenID RISC Profile Specification 1.0](https://openid.net/specs/openid-risc-1_0-final.html)
- [OpenID SSE (Shared Signals and Events) Framework](https://openid.net/sse/)
- [RFC 8417 — Security Event Token (SET) Format](https://datatracker.ietf.org/doc/html/rfc8417)
- [Continuous Authorization: CAEP, RISC, and Real-Time Session Revocation](https://www.systemshardening.com/articles/cross-cutting/continuous-authorization-caep/)
- GGID audit infrastructure: `pkg/audit/publisher.go`, `services/audit/internal/consumer/nats_consumer.go`
