# GGID Engineering Team — Autonomous Backlog

**每位 teammate 完成手头任务后，从这个 backlog 中按优先级认领下一个任务。**
**如果 backlog 中的任务都不在你的技能范围，自主研究竞品新功能并创建自己的任务。**

## 自主工作规则

1. 完成一个任务 → commit → 立即认领下一个，不等 arch 分配
2. 每完成 3 个任务，花 5 分钟做竞品研究（web search），发现新功能就实现
3. 如果发现 bug 或技术债，记录在 `docs/tech-debt.md` 并修复
4. 如果发现测试覆盖率低于 70% 的包，主动补充测试
5. 如果发现文档缺失，主动补充

## Task Status Convention
- `[TODO]` — 待认领
- `[IN PROGRESS: <name>]` — 正在做
- `[DONE]` — 完成已提交

---

## Backlog for dev (services/identity, auth, oauth, pkg/authprovider, pkg/social)

### P0 — Core
- [DONE] SAML 2.0 SP-initiated login flow (parse AuthnRequest, create Assertion)
- [DONE] SAML metadata endpoint: GET /saml/metadata
- [DONE] OAuth2 PKCE verification in authorize endpoint (S256 challenge)
- [DONE] Token revocation endpoint: POST /oauth/revoke (RFC 7009)
- [DONE] Back-channel logout: POST /oauth/logout (OIDC Back-Channel Logout 1.0)
- [DONE] OAuth client credentials rotation: rotate client_secret

### P1 — Enterprise
- [DONE] Password history enforcement (reject reused passwords)
- [DONE] Account lockout policy (configurable threshold + duration)
- [DONE] Email verification flow (token + verification endpoint)
- [DONE] Magic link authentication (passwordless email)
- [DONE] Phone-based OTP authentication
- [DONE] LDAP group → role mapping

### P2 — Enhancement
- [DONE] OAuth consent screen (user approves scopes)
- [DONE] JWT claim customization (add custom claims via rules)
- [TODO] Auth service coverage → 85%+ (currently 80.4%)
- [TODO] OAuth service coverage → 70%+ (currently 59.9%)
- [DONE] Social connector: Microsoft, Apple, GitLab, Discord, LinkedIn

### P3 — Innovation
- [TODO] Passkey autofill (WebAuthn conditional mediation)
- [DONE] Step-up authentication (re-challenge for sensitive operations)
- [DONE] Risk-based authentication (IP reputation, device fingerprinting)
- [DONE] Password expiration forced reset (max_age_days policy)

---

## WebAuthn / Passkey Roadmap (20 tasks)

> Source: `docs/research/webauthn-passkey-best-practices.md` Section 9.
> Grouped by area. Priority: P0=CRITICAL, P1=HIGH, P2=MEDIUM, P3=LOW.
> Effort: S=<1 day, M=1-3 days, L=3-5 days.

### Group A — Database Schema Changes

---

#### WA-1: Add backup-eligibility and backup-state flags to Credential

- **Priority:** P0 (CRITICAL)
- **Effort:** S

Store `backup_eligible`, `backup_state`, `user_verified`, and `attestation_type`
fields on the `webauthn_credentials` table and in the Go `Credential` struct.
These flags (WebAuthn L2+) indicate whether a credential is synced via iCloud /
Google Password Manager and are needed for UX badges, security policy, and
account-recovery planning.

**Files to modify/create:**
- `services/auth/internal/webauthn/handler.go` — add fields to `Credential` struct
- `services/auth/internal/webauthn/store.go` (or wherever the pgx implementation
  lives) — update INSERT/SELECT to read/write the new columns
- `deploy/migrations/NNNN_add_webauthn_backup_flags.sql` — new migration:
  ```sql
  ALTER TABLE webauthn_credentials ADD COLUMN IF NOT EXISTS backup_eligible BOOLEAN DEFAULT FALSE;
  ALTER TABLE webauthn_credentials ADD COLUMN IF NOT EXISTS backup_state BOOLEAN DEFAULT FALSE;
  ALTER TABLE webauthn_credentials ADD COLUMN IF NOT EXISTS user_verified BOOLEAN DEFAULT FALSE;
  ALTER TABLE webauthn_credentials ADD COLUMN IF NOT EXISTS attestation_type TEXT DEFAULT 'none';
  ALTER TABLE webauthn_credentials ADD COLUMN IF NOT EXISTS aaguid BYTEA;
  ```
- `services/auth/internal/webauthn/handler_test.go` — update mock store

**Interface/API design:**
```go
type Credential struct {
    // ... existing fields ...
    BackupEligible  bool
    BackupState     bool
    UserVerified    bool
    AttestationType string
    AAGUID          []byte
}
```
In `finishRegistration`, set these from `credential.Flags.BackupEligible`,
`credential.Flags.BackupState`, `credential.Flags.UserVerified`,
`credential.AttestationType`, and `credential.Authenticator.AAGUID`.

**Acceptance criteria:**
- After registration, the DB row for the new credential has non-default values
  for `backup_eligible` (platform authenticators on iOS/macOS → `true`).
- `listCredentials` endpoint response includes `backup_eligible`, `backup_state`,
  `attestation_type` fields.
- `go test ./services/auth/internal/webauthn/` passes.
- Migration is idempotent (`ADD COLUMN IF NOT EXISTS`).

---

#### WA-8: Persist authenticator transports array

- **Priority:** P1 (HIGH)
- **Effort:** S

Currently `finishRegistration` stores `credential.Authenticator.Attachment`
(a single string like `"platform"`) into the `Transports` field. This is wrong —
`Transports` should hold the authenticator transport list
(`internal`, `hybrid`, `usb`, `nfc`). Transports are needed for hybrid QR flows
and for the browser to pick the correct UI.

**Files to modify:**
- `services/auth/internal/webauthn/handler.go` — fix `finishRegistration` and
  `buildWebAuthnUser` to use `credential.Transport` instead of `.Attachment`
- `services/auth/internal/webauthn/handler_test.go` — add test asserting
  transports are `[]string{"internal"}` for a platform authenticator mock

**Interface/API design:**
```go
// In finishRegistration, replace:
//   Transports: []string{string(credential.Authenticator.Attachment)},
// with:
transports := make([]string, len(credential.Transport))
for i, t := range credential.Transport {
    transports[i] = string(t)
}
cred := &Credential{
    // ...
    Transports: transports,
}
```
In `buildWebAuthnUser`, reverse-map stored strings back to
`[]protocol.AuthenticatorTransport`.

**Acceptance criteria:**
- `Transports` field in the DB contains values like `"internal"` or
  `["hybrid","usb"]`, not `"platform"`.
- `beginAuthentication` returns `allowCredentials` with correct `transports`.
- `buildWebAuthnUser` round-trips transports correctly for re-auth.

---

### Group B — Registration Flow Improvements

---

#### WA-3: Add excludeCredentials to registration

- **Priority:** P0 (CRITICAL)
- **Effort:** S

`beginRegistration` currently calls `h.wbn.BeginRegistration(user)` with no
options, so a user can register the same authenticator twice. Pass
`excludeCredentials` listing the user's existing credential IDs.

**Files to modify:**
- `services/auth/internal/webauthn/handler.go` — `beginRegistration`

**Interface/API design:**
```go
func (h *Handler) beginRegistration(w http.ResponseWriter, r *http.Request) {
    // ... existing tenant/user resolution ...
    user, _ := h.buildWebAuthnUser(ctx, tenantID, userID)

    existingCreds := user.WebAuthnCredentials()
    excludeCreds := make([]protocol.CredentialDescriptor, len(existingCreds))
    for i, c := range existingCreds {
        excludeCreds[i] = protocol.CredentialDescriptor{
            CredentialID: c.ID,
            Type:         protocol.PublicKeyCredentialType,
            Transport:    c.Transport,
        }
    }

    options, sessData, err := h.wbn.BeginRegistration(
        user,
        webauthn.WithExclusions(excludeCreds),
        // ... other options from WA-4 ...
    )
    // ...
}
```

**Acceptance criteria:**
- When a user with one existing credential calls `/register/begin`, the response
  `excludeCredentials` array contains that credential's ID.
- Attempting to register the same authenticator a second time fails at the
  browser/library level (`InvalidStateError`).
- Users with zero existing credentials get an empty `excludeCredentials`.

---

#### WA-4: Add explicit AuthenticatorSelection to registration

- **Priority:** P0 (CRITICAL)
- **Effort:** S

Pass explicit `AuthenticatorSelection` to `BeginRegistration`: set
`residentKey: preferred` and `userVerification: preferred`. Also set
`attestation: none` for consumer privacy and request the `credProps` extension
so the server knows if the created credential is discoverable.

**Files to modify:**
- `services/auth/internal/webauthn/handler.go` — `beginRegistration`

**Interface/API design:**
```go
options, sessData, err := h.wbn.BeginRegistration(
    user,
    webauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{
        ResidentKey:      protocol.ResidentKeyRequirementPreferred,
        UserVerification: protocol.VerificationPreferred,
    }),
    webauthn.WithAttestation(protocol.PreferNoAttestation),
    webauthn.WithExclusions(excludeCreds), // from WA-3
    webauthn.WithExtensions(map[string]any{"credProps": true}),
)
```

**Acceptance criteria:**
- `/register/begin` response includes `authenticatorSelection.residentKey =
  "preferred"`.
- `attestation` in the response is `"none"`.
- `extensions.credProps = true` is present.
- `go test ./services/auth/internal/webauthn/` passes.

---

#### WA-7: Auto-generate credential names from User-Agent

- **Priority:** P1 (HIGH)
- **Effort:** S

`finishRegistration` reads the credential name from `r.URL.Query().Get("name")`.
When empty, auto-generate a friendly name from the `User-Agent` header
(e.g., "Chrome on macOS", "Safari on iPhone"). Also persist `created_at` and
add a `last_used_at` update during authentication.

**Files to modify:**
- `services/auth/internal/webauthn/handler.go` — `finishRegistration`
- `services/auth/internal/webauthn/handler_test.go` — test auto-naming

**Interface/API design:**
```go
// New helper:
func generateCredentialName(userAgent string) string {
    browser := parseBrowser(userAgent) // "Chrome", "Safari", "Firefox", "Edge"
    os := parseOS(userAgent)           // "macOS", "Windows", "iOS", "Android", "Linux"
    if browser == "" || os == "" {
        return "Passkey"
    }
    return fmt.Sprintf("%s on %s", browser, os)
}
```
In `finishRegistration`:
```go
name := r.URL.Query().Get("name")
if name == "" {
    name = generateCredentialName(r.UserAgent())
}
```
Also add `LastUsedAt` update in `finishAuthentication`:
```go
// Extend CredentialStore:
UpdateLastUsed(ctx context.Context, tenantID uuid.UUID, credID []byte, t time.Time) error
```

**Acceptance criteria:**
- Registering with no `name` query param produces a credential whose name is
  derived from the `User-Agent` (e.g., "Chrome on macOS").
- After successful authentication, the credential's `last_used_at` is updated.
- `listCredentials` response includes `name`, `created_at`, and `last_used_at`.

---

### Group C — Authentication UX (Conditional UI)

---

#### WA-5: Conditional UI / passkey autofill on console login page

- **Priority:** P1 (HIGH)
- **Effort:** M

Implement Conditional UI (passkey autofill) in the console login page. The
server-side `beginAuthentication` already returns empty `allowCredentials`
(suitable for discoverable credentials). The frontend needs
`autocomplete="username webauthn"` and the conditional mediation JS.

**Files to modify/create:**
- `console/src/app/(auth)/login/page.tsx` — add `autocomplete` tokens and
  conditional UI init script
- `console/src/lib/webauthn.ts` (new) — conditional mediation helper

**Interface/API design (frontend):**
```typescript
// console/src/lib/webauthn.ts
export async function initConditionalUI(tenantId: string): Promise<void> {
  if (!window.PublicKeyCredential?.isConditionalMediationAvailable) return;
  const available = await PublicKeyCredential.isConditionalMediationAvailable();
  if (!available) return;

  const resp = await fetch('/api/v1/webauthn/auth/begin', {
    method: 'POST',
    headers: { 'X-Tenant-ID': tenantId },
  });
  const { publicKey } = await resp.json();

  const abortController = new AbortController();
  try {
    const credential = await navigator.credentials.get({
      publicKey,
      mediation: 'conditional',
      signal: abortController.signal,
    });
    const authResp = await fetch('/api/v1/webauthn/auth/finish', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-Tenant-ID': tenantId },
      body: JSON.stringify(credential),
    });
    if (authResp.ok) {
      window.location.href = '/dashboard';
    }
  } catch (err) {
    // AbortError is expected when user picks password path
  }
}
```
HTML:
```html
<input type="text" autocomplete="username webauthn" />
<input type="password" autocomplete="current-password webauthn" />
```

**Acceptance criteria:**
- On a browser that supports Conditional UI (Chrome 107+, Safari 16+), the
  login page shows passkey suggestions in the username field's autofill menu.
- Selecting a passkey triggers biometric prompt and authenticates without
  typing a password.
- Clicking "Sign In" with password aborts the conditional flow cleanly.
- Falls back gracefully to password-only login on unsupported browsers.

---

#### WA-6: Post-login passkey enrollment nudge

- **Priority:** P1 (HIGH)
- **Effort:** M

After a successful password login, show a one-time prompt to enroll a passkey:
"Create a passkey for faster, more secure sign-in?". Show max 2-3 times before
going dormant. Respect "Not Now".

**Files to modify/create:**
- `console/src/components/PasskeyEnrollNudge.tsx` (new) — modal/banner component
- `console/src/app/(dashboard)/layout.tsx` — render the nudge after login
- `services/auth/internal/webauthn/handler.go` — add
  `PATCH /api/v1/webauthn/enrollment-dismissed` to record dismissal
- `deploy/migrations/NNNN_add_passkey_nudge_dismissed.sql` — new column:
  ```sql
  ALTER TABLE users ADD COLUMN IF NOT EXISTS passkey_nudge_dismissed_at TIMESTAMPTZ;
  ALTER TABLE users ADD COLUMN IF NOT EXISTS passkey_nudge_count INT DEFAULT 0;
  ```

**Interface/API design:**
```go
// New endpoint handlers:
func (h *Handler) dismissEnrollmentNudge(w http.ResponseWriter, r *http.Request)
func (h *Handler) getEnrollmentStatus(w http.ResponseWriter, r *http.Request)
// Returns {"show_nudge": true/false, "nudge_count": N}
```

**Acceptance criteria:**
- After password login, if user has 0 passkeys and `nudge_count < 3`, the
  console shows the enrollment banner.
- Clicking "Create Passkey" starts the WebAuthn registration flow.
- Clicking "Not Now" increments `nudge_count`; after 3 dismissals the nudge
  stops appearing.
- Users who already have a passkey never see the nudge.

---

#### WA-15: Forward transports in allowCredentials during authentication

- **Priority:** P2 (MEDIUM)
- **Effort:** S

When `beginAuthentication` returns `allowCredentials` (for non-discoverable
flows), each entry should include the `transports` array from the stored
credential. This helps the browser pick the right transport (USB vs. BLE vs.
internal) faster.

**Files to modify:**
- `services/auth/internal/webauthn/handler.go` — `beginAuthentication`, add an
  optional user-scoped path that populates `allowCredentials` with transports

**Interface/API design:**
```go
// When user_id is provided in beginAuthentication, build allowCredentials:
if userIDStr := r.URL.Query().Get("user_id"); userIDStr != "" {
    userID, _ := uuid.Parse(userIDStr)
    user, _ := h.buildWebAuthnUser(ctx, tenantID, userID)
    opts := []webauthn.LoginOption{
        webauthn.WithAllowedCredentials( /* descriptors from user creds */ ),
    }
    options, sessData, err := h.wbn.BeginLogin(user, opts...)
}
```

**Acceptance criteria:**
- When `user_id` is passed to `/auth/begin`, the response `allowCredentials`
  entries include `transports` matching stored values.
- When no `user_id` is passed, `allowCredentials` remains empty (discoverable
  credential / conditional UI path).

---

### Group D — Credential Management

---

#### WA-9: Make RP Origins configurable

- **Priority:** P1 (HIGH)
- **Effort:** S

`NewHandler` hardcodes `RPOrigins` as `["https://" + rpID, "http://localhost:3000"]`.
Make it configurable so multi-domain and production deployments can set the
exact allowed origins.

**Files to modify:**
- `services/auth/internal/webauthn/handler.go` — `NewHandler` signature change
- `services/auth/cmd/main.go` — pass origins from env/config

**Interface/API design:**
```go
// Option pattern:
type Option func(*Handler)

func WithOrigins(origins []string) Option {
    return func(h *Handler) {
        // Update the webauthn.Config RPOrigins
    }
}

// Or simpler — change signature:
func NewHandler(rpID, rpName string, origins []string, store CredentialStore) (*Handler, error)
```
Env vars in main.go:
```
WEBAUTHN_RP_ID=auth.ggid.dev
WEBAUTHN_RP_DISPLAY_NAME="GGID IAM"
WEBAUTHN_RP_ORIGINS=https://auth.ggid.dev,https://console.ggid.dev
```

**Acceptance criteria:**
- `NewHandler` accepts a list of origins.
- Origins are validated (must start with `https://` or `http://localhost`).
- Default fallback when no origins provided: `["https://" + rpID]`.
- Tests verify that an assertion from an unlisted origin is rejected.

---

#### WA-10: Redis-backed session store for WebAuthn

- **Priority:** P1 (HIGH)
- **Effort:** M

Replace the in-memory `sessionStore` (a `sync.Mutex` + `map`) with a Redis-
backed implementation that supports TTL and multi-instance deployments.

**Files to modify/create:**
- `services/auth/internal/webauthn/session.go` (new) — extract `SessionStore`
  interface, add `redisSessionStore`
- `services/auth/internal/webauthn/handler.go` — use interface instead of
  concrete `*sessionStore`
- `services/auth/cmd/main.go` — wire Redis client
- `services/auth/internal/webauthn/session_test.go` (new) — test Redis store

**Interface/API design:**
```go
type SessionStore interface {
    Save(ctx context.Context, key string, data *SessionData, ttl time.Duration) error
    Get(ctx context.Context, key string) (*SessionData, error)
    Delete(ctx context.Context, key string) error
}

type SessionData struct {
    UserID    uuid.UUID             `json:"user_id,omitempty"`
    TenantID  uuid.UUID             `json:"tenant_id"`
    Challenge string                `json:"challenge"`
    Data      *webauthn.SessionData `json:"data"`
    CreatedAt time.Time             `json:"created_at"`
}

type redisSessionStore struct {
    client *redis.Client
    prefix string // "webauthn:session:"
}
```

**Acceptance criteria:**
- Sessions are stored in Redis with a configurable TTL (default 5 min).
- Two instances of the auth service can share sessions (register on instance A,
  finish on instance B).
- Existing in-memory implementation kept as fallback when Redis is not
  configured.
- `go test ./services/auth/internal/webauthn/` passes with both stores.

---

#### WA-16: Security event logging for WebAuthn

- **Priority:** P2 (MEDIUM)
- **Effort:** M

Log security-relevant WebAuthn events to the audit system: registration,
authentication success/failure, credential deletion, and clone detection.

**Files to modify/create:**
- `services/auth/internal/webauthn/handler.go` — add an `AuditLogger` interface
  and call it at key points
- `services/auth/internal/webauthn/audit.go` (new) — implement the logger

**Interface/API design:**
```go
type AuditLogger interface {
    LogWebAuthnEvent(ctx context.Context, event WebAuthnAuditEvent) error
}

type WebAuthnAuditEvent struct {
    TenantID     uuid.UUID
    UserID       uuid.UUID
    EventType    string // "register", "authenticate", "delete", "clone_detected"
    CredentialID string
    Success      bool
    Detail       string
    Timestamp    time.Time
}
```
Events to emit:
- `webauthn.register.success` / `webauthn.register.failed`
- `webauthn.authenticate.success` / `webauthn.authenticate.failed`
- `webauthn.credential.deleted`
- `webauthn.clone_detected` (from WA-2)

**Acceptance criteria:**
- Each registration, authentication, and deletion emits an audit event.
- Clone detection (from WA-2) logs a `clone_detected` event.
- Events include tenant, user, credential ID, success/failure.
- The audit logger is optional (nil-safe — no-op when not wired).

---

### Group E — Security Hardening

---

#### WA-2: Enforce signature count monotonicity (clone detection)

- **Priority:** P0 (CRITICAL)
- **Effort:** S

The handler stores `Counter` and calls `UpdateCounter` after authentication but
never checks monotonicity. When `signCount > 0` and the received count is <= the
stored count, this indicates a possible cloned credential. Implement the check,
log a security event, and reject authentication.

**Files to modify:**
- `services/auth/internal/webauthn/handler.go` — `finishAuthentication`
- `services/auth/internal/webauthn/handler_test.go` — test clone detection

**Interface/API design:**
```go
func (h *Handler) finishAuthentication(w http.ResponseWriter, r *http.Request) {
    // ... after ValidateLogin ...
    credential, err := h.wbn.ValidateLogin(user, *sd.data, parsedResponse)
    // ...

    // Clone detection
    if credential.Authenticator.SignCount > 0 {
        storedCred, _ := h.creds.GetCredentialByID(ctx, tenantID, credential.ID)
        if storedCred != nil && credential.Authenticator.SignCount <= storedCred.Counter {
            // Log security event (if audit logger wired via WA-16)
            writeError(w, http.StatusUnauthorized,
                "possible credential clone detected")
            return
        }
    }

    // Update counter
    h.creds.UpdateCounter(ctx, tenantID, credential.ID, credential.Authenticator.SignCount)
    // ...
}
```

**Acceptance criteria:**
- If `signCount == 0`, authentication proceeds normally (authenticator doesn't
  support counters).
- If `signCount > 0` and received count > stored count, authentication succeeds.
- If `signCount > 0` and received count <= stored count, authentication is
  rejected with HTTP 401 and a security event is logged.
- `go test ./services/auth/internal/webauthn/` passes with clone-detection
  test cases.

---

#### WA-14: FIDO Metadata Service (MDS) integration

- **Priority:** P2 (MEDIUM)
- **Effort:** L

Integrate the FIDO Metadata Service to verify authenticator attestation
certificates against known authenticator metadata. This is optional and only
relevant for deployments using `attestation: "direct"` or `"enterprise"`.

**Files to modify/create:**
- `pkg/webauthnmds/mds.go` (new) — MDS client, metadata cache, certificate
  chain validation
- `pkg/webauthnmds/mds_test.go` (new)
- `services/auth/internal/webauthn/handler.go` — optional MDS validation hook

**Interface/API design:**
```go
package webauthnmds

type MetadataService struct {
    cache map[string]*MetadataBLOB // keyed by AAGUID
}

type MetadataBLOB struct {
    AAGUID            string
    Description       string
    AttestationRoots  []string
    StatusReports     []StatusReport
}

func NewMetadataService(blobURL string) (*MetadataService, error)
func (m *MetadataService) VerifyAttestation(aaguid []byte, attestation []byte) error
func (m *MetadataService) GetAuthenticatorInfo(aaguid []byte) (*MetadataBLOB, error)
```

**Acceptance criteria:**
- When MDS is configured, registration with `attestation: "direct"` validates
  the attestation certificate against FIDO MDS.
- Authenticators with `REVOKED` status are rejected.
- When MDS is not configured, registration proceeds with `attestation: "none"`
  (current behavior).
- MDS blob is cached and refreshed periodically (daily).

---

#### WA-20: WebAuthn error classification for UX

- **Priority:** P3 (LOW)
- **Effort:** S

Currently the handler returns generic error messages. Classify WebAuthn errors
into actionable categories so the console frontend can show appropriate UX.

**Files to modify/create:**
- `services/auth/internal/webauthn/errors.go` (new) — error classification
- `services/auth/internal/webauthn/handler.go` — use classified errors in
  responses

**Interface/API design:**
```go
package webauthn

type ErrorCode string

const (
    ErrCodeUserCancelled  ErrorCode = "USER_CANCELLED"      // AbortError, NotAllowedError
    ErrCodeInvalidState   ErrorCode = "INVALID_STATE"        // credential already exists
    ErrCodeSecurity       ErrorCode = "SECURITY_ERROR"       // origin mismatch
    ErrCodeNotSupported   ErrorCode = "NOT_SUPPORTED"        // no authenticator
    ErrCodeTimeout        ErrorCode = "TIMEOUT"
    ErrCodeCloneDetected  ErrorCode = "CLONE_DETECTED"
    ErrCodeUnknown        ErrorCode = "UNKNOWN"
)

type ErrorResponse struct {
    Code    ErrorCode `json:"code"`
    Message string    `json:"message"`
}

func ClassifyError(err error) ErrorResponse
```

**Acceptance criteria:**
- `/register/finish` and `/auth/finish` return structured `{"code": "...",
  "message": "..."}` error objects.
- User cancellation (`NotAllowedError`) returns `USER_CANCELLED`.
- Clone detection returns `CLONE_DETECTED`.
- The console can show user-friendly messages based on `code`.

---

### Group F — Migration / Coexistence

---

#### WA-11: Related Origin Requests (ROR) support

- **Priority:** P2 (MEDIUM)
- **Effort:** M

Host a `.well-known/webauthn` endpoint returning the list of authorized origins
for multi-domain setups (e.g., `example.com` and `example.co.jp`).

**Files to modify/create:**
- `services/auth/internal/webauthn/handler.go` — new route
  `GET /.well-known/webauthn`
- `services/auth/internal/webauthn/ror.go` (new) — ROR config struct
- `services/auth/cmd/main.go` — load ROR origins from config

**Interface/API design:**
```go
// GET /.well-known/webauthn
// Response:
{
  "origins": [
    "https://www.example.com",
    "https://shop.example.co.jp"
  ]
}
```
Handler:
```go
func (h *Handler) wellKnownWebAuthn(w http.ResponseWriter, r *http.Request) {
    writeJSON(w, http.StatusOK, map[string]any{
        "origins": h.rpOrigins,
    })
}
```

**Acceptance criteria:**
- `GET /.well-known/webauthn` returns a JSON object with an `origins` array.
- Origins are configurable via env var `WEBAUTHN_RP_ORIGINS`.
- When not configured, returns the single RP origin.

---

#### WA-12: Mobile app integration (Digital Asset Links + Apple App Site Association)

- **Priority:** P2 (MEDIUM)
- **Effort:** S

Serve `assetlinks.json` (Android) and `apple-app-site-association` (iOS) from
the RP domain so mobile apps can participate in passkey authentication.

**Files to modify/create:**
- `services/auth/internal/webauthn/handler.go` — two new routes:
  `GET /.well-known/assetlinks.json`, `GET /.well-known/apple-app-site-association`
- `services/auth/cmd/main.go` — load app config from env

**Interface/API design:**
```go
// Android — GET /.well-known/assetlinks.json
[{"relation":["delegate_permission/common.get_login_creds"],
  "target":{"namespace":"android_app",
    "package_name":"com.ggid.console",
    "sha256_cert_fingerprints":["AB:CD:..."]}}]

// iOS — GET /.well-known/apple-app-site-association
{"webcredentials":{"apps":["TEAMID.com.ggid.console"]}}
```
Env vars:
```
WEBAUTHN_ANDROID_PACKAGE=com.ggid.console
WEBAUTHN_ANDROID_SHA256=AB:CD:EF:...
WEBAUTHN_IOS_APP_IDS=TEAMID.com.ggid.console
```

**Acceptance criteria:**
- `GET /.well-known/assetlinks.json` returns valid JSON with the configured
  Android package and SHA-256 fingerprint.
- `GET /.well-known/apple-app-site-association` returns valid JSON with iOS app
  IDs.
- When not configured, both endpoints return empty/default values.

---

#### WA-13: Account recovery via recovery codes

- **Priority:** P2 (MEDIUM)
- **Effort:** M

Generate and store recovery codes during passkey enrollment so users who lose
their device can still authenticate. Combine with email magic link (already
implemented) for consumer recovery.

**Files to modify/create:**
- `services/auth/internal/webauthn/recovery.go` (new) — recovery code
  generation, hashing, verification
- `services/auth/internal/webauthn/handler.go` —
  `POST /api/v1/webauthn/recovery/generate`, `POST /api/v1/webauthn/recovery/verify`
- `deploy/migrations/NNNN_add_webauthn_recovery_codes.sql`:
  ```sql
  CREATE TABLE IF NOT EXISTS webauthn_recovery_codes (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    user_id UUID NOT NULL,
    code_hash TEXT NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
  );
  ```

**Interface/API design:**
```go
func GenerateRecoveryCodes(count int) []string // returns e.g. ["XXXX-XXXX-XXXX", ...]
func HashRecoveryCode(code string) string       // bcrypt
func VerifyRecoveryCode(hash, code string) bool

// Endpoints:
// POST /api/v1/webauthn/recovery/generate?user_id=...
//   → {"codes": ["XXXX-XXXX-XXXX", ...]}  (shown once)
// POST /api/v1/webauthn/recovery/verify
//   Body: {"code": "XXXX-XXXX-XXXX"}
//   → {"status": "verified", "recovery_token": "..."}  (one-time use)
```

**Acceptance criteria:**
- After passkey enrollment, `POST /recovery/generate` returns N recovery codes
  (default 10) in `XXXX-XXXX-XXXX` format.
- Codes are stored as bcrypt hashes, never plaintext.
- `POST /recovery/verify` consumes a code (one-time use) and issues a
  short-lived recovery token.
- Used codes are marked `used_at` and cannot be reused.

---

### Group G — Advanced Extensions (Low Priority)

---

#### WA-17: PRF (Pseudo-Random Function) extension

- **Priority:** P3 (LOW)
- **Effort:** L

Support the PRF extension (WebAuthn L3 draft) to derive symmetric keys from
passkeys during authentication. Enables end-to-end encryption use cases.

**Files to modify/create:**
- `services/auth/internal/webauthn/prf.go` (new) — PRF derivation logic
- `services/auth/internal/webauthn/handler.go` — request PRF extension in
  `beginAuthentication`, process result in `finishAuthentication`

**Interface/API design:**
```go
// In beginAuthentication, request PRF:
opts := []webauthn.LoginOption{
    webauthn.WithExtensions(map[string]any{
        "prf": map[string]any{"eval": map[string][]byte{
            "first": prfSalt, // server-generated salt
        }},
    }),
}

// In finishAuthentication, extract PRF result:
// parsedResponse.ClientExtensionResults["prf"]["results"]["first"]
```

**Acceptance criteria:**
- `/auth/begin` response includes `extensions.prf` when enabled.
- `/auth/finish` extracts the PRF output and returns it (base64-encoded).
- PRF derivation is deterministic for the same credential + salt.
- Disabled by default; enabled via config flag.

---

#### WA-18: LargeBlob extension support

- **Priority:** P3 (LOW)
- **Effort:** L

Support the LargeBlob extension to store opaque, mutable data associated with a
credential on the authenticator (e.g., encrypted configuration, client-side
certificates).

**Files to modify/create:**
- `services/auth/internal/webauthn/largeblob.go` (new) — LargeBlob read/write
  helpers
- `services/auth/internal/webauthn/handler.go` — request LargeBlob during
  registration, provide read/write endpoints

**Interface/API design:**
```go
// New endpoints:
// POST /api/v1/webauthn/credentials/{id}/blob/write
//   Body: {"blob": "<base64>"}
// GET  /api/v1/webauthn/credentials/{id}/blob/read
//   → {"blob": "<base64>"}
```
Registration option:
```go
webauthn.WithExtensions(map[string]any{"largeBlob": map[string]any{"support": "required"}})
```

**Acceptance criteria:**
- When LargeBlob is enabled, `/register/begin` includes the `largeBlob` extension.
- Blob write endpoint stores data via the authenticator.
- Blob read endpoint retrieves stored data.
- Disabled by default; enabled via config flag.

---

#### WA-19: Enterprise attestation support

- **Priority:** P3 (LOW)
- **Effort:** M

Support `attestation: "enterprise"` for corporate deployments that need device
management. This requires FIDO MDS integration (WA-14) and AAGUID-based
allow/deny policies.

**Files to modify/create:**
- `services/auth/internal/webauthn/handler.go` — add enterprise attestation
  option to `beginRegistration`
- `services/auth/internal/webauthn/enterprise.go` (new) — AAGUID allow/deny list

**Interface/API design:**
```go
type EnterprisePolicy struct {
    AllowedAAGUIDs  []string // whitelist of authenticator AAGUIDs
    BlockedAAGUIDs  []string // blacklist
}

func (h *Handler) beginRegistration(w http.ResponseWriter, r *http.Request) {
    if h.enterpriseMode {
        opts = append(opts, webauthn.WithAttestation(protocol.PreferEnterpriseAttestation))
    }
}

// In finishRegistration, check AAGUID against policy:
func (p *EnterprisePolicy) IsAllowed(aaguid []byte) bool
```
Config:
```
WEBAUTHN_ENTERPRISE_MODE=true
WEBAUTHN_ALLOWED_AAGUIDS=00a2b1a3-...,00b4c5d6-...
```

**Acceptance criteria:**
- When enterprise mode is enabled, `/register/begin` sets
  `attestation: "enterprise"`.
- Registration rejects authenticators not in the allowed AAGUID list.
- Works with FIDO MDS integration (WA-14).
- Disabled by default.

---

## Summary Table

| # | Task | Priority | Effort | Group |
|---|------|----------|--------|-------|
| WA-1 | backup_eligible / backup_state flags | P0 | S | DB Schema |
| WA-2 | Signature count monotonicity (clone detection) | P0 | S | Security |
| WA-3 | excludeCredentials in registration | P0 | S | Registration |
| WA-4 | AuthenticatorSelection (residentKey preferred) | P0 | S | Registration |
| WA-5 | Conditional UI / passkey autofill | P1 | M | Auth UX |
| WA-6 | Post-login passkey enrollment nudge | P1 | M | Auth UX |
| WA-7 | Credential auto-naming from User-Agent | P1 | S | Registration |
| WA-8 | Persist transports array | P1 | S | DB Schema |
| WA-9 | Configurable RP Origins | P1 | S | Credential Mgmt |
| WA-10 | Redis-backed session store | P1 | M | Credential Mgmt |
| WA-11 | Related Origin Requests (ROR) | P2 | M | Migration |
| WA-12 | Mobile app integration (DAL/AASA) | P2 | S | Migration |
| WA-13 | Account recovery codes | P2 | M | Migration |
| WA-14 | FIDO Metadata Service integration | P2 | L | Security |
| WA-15 | Forward transports in allowCredentials | P2 | S | Auth UX |
| WA-16 | Security event logging | P2 | M | Credential Mgmt |
| WA-17 | PRF extension | P3 | L | Advanced |
| WA-18 | LargeBlob extension | P3 | L | Advanced |
| WA-19 | Enterprise attestation | P3 | M | Advanced |
| WA-20 | WebAuthn error classification | P3 | S | Security |

---

## Backlog for dev2 (services/gateway, middleware, router, Docker)

### P0 — Core
- [DONE] Webhook system: registration + HMAC delivery + retry
- [DONE] Prometheus metrics endpoint: GET /metrics
- [DONE] Health check aggregation: GET /healthz returns all backend statuses
- [DONE] Health check split: /healthz/live + /healthz/ready (Kubernetes probes)
- [DONE] Request tracing: X-Request-ID propagation + W3C traceparent spans

### P1 — Security
- [DONE] API key authentication (alternative to JWT for M2M)
- [DONE] IP allowlist middleware (per-tenant configurable)
- [DONE] Bot detection (User-Agent + behavior analysis)
- [DONE] Gateway middleware coverage → 70%+ (currently 71.3%)

### P2 — Performance
- [DONE] Response caching middleware (ETag + conditional GET)
- [DONE] Connection pooling tuning + keep-alive optimization
- [DONE] Request body size limiting middleware
- [DONE] Compression middleware (gzip/brotli)
- [DONE] Per-route timeout configuration (RouteConfig + RouteTimeout)

### P3 — Innovation
- [TODO] GraphQL proxy support
- [DONE] WebSocket proxy support (HTTP hijack → bidirectional TCP tunnel)
- [DONE] Canary deployment routing (percentage-based traffic splitting with header/cookie override)
- [DONE] Circuit breaker pattern (closed/open/half-open with per-backend registry)
- [DONE] Request ID propagation to backends via X-Request-ID header in proxy Director
- [TODO] gRPC-Web protocol translation middleware

---

## Backlog for dev3 (services/policy, org, audit, console pages)

### P0 — Core
- [DONE] Policy engine: ABAC condition evaluation (resource attributes)
- [TODO] Role hierarchy: parent role inherits child permissions
- [TODO] Bulk user-role assignment API
- [TODO] Audit real-time streaming via WebSocket

### P1 — Enterprise
- [TODO] Policy export/import (JSON format for CI/CD integration)
- [TODO] Audit log retention policy + scheduled cleanup
- [DONE] Audit log export (CSV/JSON download)
- [TODO] Org service coverage → 70%+ (currently 65.8%)

### P2 — Enhancement
- [TODO] Console: OAuth client management page
- [TODO] Console: Webhook management page
- [DONE] Console: Audit dashboard with charts (recharts)
- [TODO] Console: User activity timeline
- [TODO] Console: Dark mode support
- [TODO] Console: i18n (Chinese + English)

### P3 — Innovation
- [TODO] Policy decision logging (record every allow/deny)
- [TODO] Permission analyzer (visualize which roles can access what)
- [TODO] Org chart visualization (interactive tree graph)

---

## SCIM 2.0 Compliance Tasks for dev3

> Source: `docs/research/scim2-compliance-test-suite.md` Section 8.3 — 20 GGID gaps.
> All tasks target files under `services/identity/internal/scim/` and `services/identity/internal/server/`.

### Group A: Core Schema Completion

#### SCIM-01: `externalId` Persistence (RFC 7643 Section 5)
- **Priority**: P1
- **Files**: `services/identity/internal/scim/handler.go`, `services/identity/internal/domain/user.go`, `services/identity/internal/repository/user_repo.go`
- **API Design**: Add `ExternalID string` to `domain.User` and `domain.CreateUserInput`. `SCIMUser` already has `ExternalID` field. Update `toSCIMUser()` to populate `ExternalID` from `u.ExternalID`. Update `createUser()` and `replaceUser()` to pass `scimUser.ExternalID` into `CreateUserInput`/`UpdateUserInput`.
- **Acceptance Criteria**:
  - POST `/Users` with `externalId` field returns it in the response body (RFC 7643 Section 5.1).
  - GET `/Users/{id}` returns persisted `externalId`.
  - PUT `/Users/{id}` can update `externalId`.
  - Test: create user with `externalId: "hr-001"`, re-fetch, verify field present.

#### SCIM-02: `meta.created` / `meta.lastModified` / `meta.version` (RFC 7643 Section 5.1)
- **Priority**: P0
- **Files**: `services/identity/internal/scim/handler.go`
- **API Design**: Extend `SCIMMeta` struct:
  ```go
  type SCIMMeta struct {
      ResourceType  string `json:"resourceType"`
      Created       string `json:"created,omitempty"`
      LastModified  string `json:"lastModified,omitempty"`
      Location      string `json:"location,omitempty"`
      Version       string `json:"version,omitempty"`
  }
  ```
  Update `toSCIMUser()` to set `Created: u.CreatedAt.Format(time.RFC3339)`, `LastModified: u.UpdatedAt.Format(time.RFC3339)`, `Version: fmt.Sprintf("W/\"%x\"", u.UpdatedAt.UnixNano())`.
- **Acceptance Criteria**:
  - Every User and Group response includes `meta.created` and `meta.lastModified` as RFC 3339 timestamps (RFC 7643 Section 5.1).
  - `meta.version` is a weak ETag string (`W/"..."`).
  - Test: create user, verify `meta.created` and `meta.lastModified` are valid RFC 3339 timestamps.

#### SCIM-03: `meta.location` URL for Users (RFC 7643 Section 5.1)
- **Priority**: P0
- **Files**: `services/identity/internal/scim/handler.go`
- **API Design**: Update `toSCIMUser()` to accept a base URL parameter or read from request context. Set `Location: fmt.Sprintf("/scim/v2/Users/%s", u.ID.String())`. Also add `Location` header in `createUser()` response: `w.Header().Set("Location", resp.Meta.Location)`.
- **Acceptance Criteria**:
  - GET `/Users/{id}` response has `meta.location` populated with the full resource URL (RFC 7643 Section 5.1).
  - POST `/Users` response includes `Location` header (RFC 7644 Section 3.3).
  - Test: create user, check `Location` header is present and matches `/scim/v2/Users/{id}`.

#### SCIM-04: EnterpriseUser Extension (RFC 7643 Section 4.3)
- **Priority**: P1
- **Files**: `services/identity/internal/scim/handler.go` (new types + serialization), `services/identity/internal/domain/user.go` (new fields)
- **API Design**: Add struct:
  ```go
  type EnterpriseUser struct {
      EmployeeNumber string        `json:"employeeNumber,omitempty"`
      CostCenter     string        `json:"costCenter,omitempty"`
      Organization   string        `json:"organization,omitempty"`
      Division       string        `json:"division,omitempty"`
      Department     string        `json:"department,omitempty"`
      Manager        *ManagerRef   `json:"manager,omitempty"`
  }
  ```
  Extend `SCIMUser` with `Enterprise map[string]any` keyed by URN. Add `EmployeeNumber`, `Department` fields to `domain.User` and `domain.CreateUserInput`. In `createUser()` and `replaceUser()`, detect `urn:ietf:params:scim:schemas:extension:enterprise:2.0:User` in the raw JSON body and extract enterprise fields.
- **Acceptance Criteria**:
  - POST `/Users` with enterprise extension schema returns the extension attributes (RFC 7643 Section 4.3).
  - GET `/Users/{id}` returns enterprise extension attributes if present.
  - Test: create user with `employeeNumber: "701984"`, re-fetch, verify field is persisted and returned.

#### SCIM-05: `ListResponse.Resources` Generic Typing (RFC 7644 Section 3.4.2)
- **Priority**: P2
- **Files**: `services/identity/internal/scim/handler.go`, `services/identity/internal/scim/groups.go`
- **API Design**: Replace `ListResponse.Resources []SCIMUser` with `Resources []any`. Update `listUsers()` to return `[]SCIMUser` (unchanged data). Update `listGroups()` to return `[]SCIMGroup` instead of current `map[string]any` workaround.
- **Acceptance Criteria**:
  - `GET /Users` returns `Resources` as array of User resources with correct `schemas` (RFC 7644 Section 3.4.2).
  - `GET /Groups` returns `Resources` as array of Group resources.
  - Both use the same `ListResponse` struct with typed resources.
  - Test: list groups, verify `Resources[0]` has `schemas` containing Group URN and `displayName` field.

#### SCIM-06: POST `/Users` Location Header (RFC 7644 Section 3.3)
- **Priority**: P2
- **Files**: `services/identity/internal/scim/handler.go`
- **API Design**: In `createUser()`, after successful create, set `w.Header().Set("Location", fmt.Sprintf("/scim/v2/Users/%s", user.ID.String()))` before calling `writeSCIMJSON`.
- **Acceptance Criteria**:
  - POST `/Users` with valid body returns `201 Created` with `Location` header set to the new resource URL (RFC 7644 Section 3.3).
  - Test: `resp.Header.Get("Location")` is non-empty and contains the created user's ID.

#### SCIM-07: `password` Attribute Support (RFC 7643 Section 4.1.1 / RFC 7644 Section 3.5.1)
- **Priority**: P3
- **Files**: `services/identity/internal/scim/handler.go`
- **API Design**: Add `Password string `json:"password,omitempty"` to `SCIMUser`. In `createUser()`, if `scimUser.Password != ""`, pass it as the password instead of the hardcoded `"TempPass123!"`. Support `PUT` with `password` to trigger password change via the auth service.
- **Acceptance Criteria**:
  - POST `/Users` with `password` field uses the provided password (RFC 7643 Section 4.1.1).
  - PUT `/Users/{id}` with `password` field changes the user's password.
  - Test: create user with custom password, verify login works with that password.

### Group B: Filter Engine

#### SCIM-08: SCIM Filter Parser and Engine (RFC 7644 Section 3.4.2)
- **Priority**: P0
- **Files**: `services/identity/internal/scim/filter.go` (new), `services/identity/internal/scim/filter_test.go` (new), `services/identity/internal/scim/handler.go`
- **API Design**:
  ```go
  // filter.go
  type FilterNode interface{ eval(user SCIMUser) bool }
  type ComparisonExpr struct { Attr, Op string; Value any }
  type LogicalExpr struct { Op string; Left, Right FilterNode }
  type NotExpr struct{ Inner FilterNode }
  type PresentExpr struct{ Attr string }

  func ParseFilter(filterStr string) (FilterNode, error)
  func (f ComparisonExpr) eval(u SCIMUser) bool
  ```
  Support operators: `eq`, `ne`, `co`, `sw`, `ew`, `pr`, `gt`, `ge`, `lt`, `le`, `and`, `or`, `not`, parentheses grouping, and complex multi-valued attribute paths (`emails[type eq "work"].value`). In `listUsers()`, parse the `filter` query param, iterate results, and filter in-memory (or translate to SQL in repository layer as a follow-up).
- **Acceptance Criteria**:
  - `userName eq "bjensen"` returns matching user (RFC 7644 Section 3.4.2.2).
  - `active eq true and userName sw "admin"` returns filtered set.
  - `emails[type eq "work"].value eq "x@example.com"` matches nested value.
  - Invalid filter returns `400 Bad Request` with `scimType: invalidFilter` (once SCIM-20 is done).
  - Tests: F-001 through F-018 from test suite doc Section 9.3.
  - **Effort**: L

### Group C: Pagination and Sort

#### SCIM-09: Sort Support in SCIM Endpoints (RFC 7644 Section 3.4.2)
- **Priority**: P1
- **Files**: `services/identity/internal/scim/handler.go`
- **API Design**: In `listUsers()`, read `sortBy` and `sortOrder` query params. Pass them to `domain.ListUsersFilter` (which already has `SortBy` and `SortDesc`). Map SCIM attributes: `userName`→`username`, `displayName`→`display_name`, `meta.created`→`created_at`, `meta.lastModified`→`updated_at`. Default `sortOrder` to `ascending`.
- **Acceptance Criteria**:
  - `GET /Users?sortBy=userName&sortOrder=ascending` returns sorted results (RFC 7644 Section 3.4.2).
  - `sortOrder=descending` reverses order.
  - `sortBy=name.familyName` sorts by display name (mapped to `display_name`).
  - Missing `sortOrder` defaults to ascending.
  - Tests: S-001 through S-007.

### Group D: PATCH Completeness

#### SCIM-10: Full PATCH Path Engine for Users (RFC 7644 Section 3.5.2)
- **Priority**: P0
- **Files**: `services/identity/internal/scim/patch.go` (new), `services/identity/internal/scim/handler.go`
- **API Design**:
  ```go
  // patch.go
  func ApplyPatch(user *SCIMUser, ops []SCIMPatchOp) (*SCIMUser, error)
  func parsePath(path string) (attrPath, valFilter, subAttr string, err error)
  ```
  Support `add`, `replace`, `remove` on: `displayName`, `active`, `name.givenName`, `name.familyName`, `nickName`, `title`, `userType`, `preferredLanguage`, `locale`, `timezone`, `emails` (append/replace/remove with value filter), `phoneNumbers`, `addresses`. Support path filters: `emails[type eq "work"].value`, `emails[type eq "home" and value co "jensen.org"]`. In `patchUser()`, replace the current hardcoded switch with a call to `ApplyPatch`.
- **Acceptance Criteria**:
  - `add` on `emails` appends to the array (RFC 7644 Section 3.5.2.3).
  - `replace` on `displayName` updates the value (RFC 7644 Section 3.5.2.3).
  - `replace` on `emails[type eq "work"].value` updates only matching entry (RFC 7644 Section 3.5.2.3).
  - `remove` on `emails[type eq "home"]` removes matching entries (RFC 7644 Section 3.5.2.3).
  - `remove` on `nickName` clears the singular attribute.
  - Unknown path returns `400` with `scimType: invalidPath` (once SCIM-20 is done).
  - Tests: U-008 through U-012.
  - **Effort**: L

#### SCIM-11: Group PATCH Application (RFC 7644 Section 3.5.2)
- **Priority**: P0
- **Files**: `services/identity/internal/scim/groups.go`
- **API Design**: Replace `patchGroup()` stub. Implement:
  ```go
  func (h *Handler) patchGroup(w http.ResponseWriter, r *http.Request, id string) {
      // Load group from persistence
      // For each operation:
      //   "add" path="members": append members (dedup by value)
      //   "remove" path="members[value eq \"id\"]": remove matching member
      //   "replace" path="displayName": update display name
      // Persist and return updated group with 200 OK
  }
  ```
- **Acceptance Criteria**:
  - `add` on `members` appends member to group (RFC 7644 Section 3.5.2.3).
  - `remove` on `members[value eq "..."]` removes specific member.
  - `replace` on `displayName` updates group name.
  - Response is the updated Group resource, not a hardcoded stub.
  - Tests: G-005, G-006, G-007.
  - Depends on: SCIM-16 (Groups persistence).

#### SCIM-12: PUT Full Replacement Semantics (RFC 7644 Section 3.5.1)
- **Priority**: P2
- **Files**: `services/identity/internal/scim/handler.go`
- **API Design**: Rewrite `replaceUser()` to perform full replacement. Currently only updates `displayName` and `active`. Should update all writable attributes from the PUT body: `userName`, `displayName`, `active`, `emails`, `phoneNumbers`, `name`, `title`, `userType`, `locale`, `timezone`, `nickName`, `profileUrl`. Attributes not present in PUT body should be set to their default/zero value (per RFC 7644 Section 3.5.1). Must NOT allow changing `id`, `schemas`, or `meta`.
- **Acceptance Criteria**:
  - PUT with full body replaces all attributes (RFC 7644 Section 3.5.1).
  - Attributes omitted from PUT body are cleared (set to default).
  - `id` is immutable — cannot be changed via PUT.
  - Test U-007 passes: PUT with full resource, GET returns all updated attributes.

### Group E: Groups Persistence

#### SCIM-13: Groups Database-Backed Persistence (RFC 7644 Section 3)
- **Priority**: P0
- **Files**: `services/identity/internal/scim/groups.go`, `services/identity/internal/repository/group_repo.go` (new), `services/identity/internal/domain/group.go` (new), `services/identity/internal/server/http.go` (inject repo)
- **API Design**:
  ```go
  // domain/group.go
  type Group struct {
      ID          uuid.UUID
      TenantID    uuid.UUID
      DisplayName string
      Members     []GroupMember
      CreatedAt   time.Time
      UpdatedAt   time.Time
  }
  type GroupMember struct {
      UserID uuid.UUID
      Type   string // "User" or "Group"
  }
  // repository/group_repo.go
  type GroupRepository interface {
      Create(ctx context.Context, g *domain.Group) error
      GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Group, error)
      List(ctx context.Context, tenantID uuid.UUID, filter *GroupListFilter) ([]*domain.Group, int, error)
      Update(ctx context.Context, g *domain.Group) error
      Delete(ctx context.Context, tenantID, id uuid.UUID) error
  }
  ```
  Replace `getMockGroups()` calls with repository calls. Add `GroupRepository` to `Handler` struct. Create migration for `scim_groups` + `scim_group_members` tables.
- **Acceptance Criteria**:
  - POST `/Groups` persists to database; re-fetch returns the stored group (RFC 7644 Section 3.3).
  - GET `/Groups/{id}` returns from database; 404 for non-existent (RFC 7644 Section 3.4.1).
  - DELETE `/Groups/{id}` removes from database; returns 204 (RFC 7644 Section 3.6).
  - Groups are tenant-scoped (RLS or tenant_id filter).
  - Tests: G-001 through G-010.
  - **Effort**: L

### Group F: Bulk Operations

#### SCIM-14: Bulk Endpoint (RFC 7644 Section 3.7)
- **Priority**: P2
- **Files**: `services/identity/internal/scim/bulk.go` (new), `services/identity/internal/scim/handler.go` (register route), `services/identity/internal/server/http.go`
- **API Design**:
  ```go
  // bulk.go
  type BulkRequest struct {
      Schemas      []BulkOperation `json:"schemas"`
      FailOnErrors int             `json:"failOnErrors"`
      Operations   []BulkOperation `json:"Operations"`
  }
  type BulkOperation struct {
      Method  string          `json:"method"`
      Path    string          `json:"path"`
      BulkID  string          `json:"bulkId"`
      Data    json.RawMessage `json:"data"`
  }
  type BulkResponse struct {
      Schemas    []string             `json:"schemas"`
      Operations []BulkResponseOp     `json:"Operations"`
  }
  func (h *Handler) handleBulk(w http.ResponseWriter, r *http.Request)
  ```
  Register route: `mux.HandleFunc("/scim/v2/Bulk", h.handleBulk)`. Process operations sequentially. Resolve `bulkId` references (e.g., `/Users/bulkId:id-001` → actual UUID from earlier POST). Honor `failOnErrors` threshold.
- **Acceptance Criteria**:
  - POST `/scim/v2/Bulk` with 2 POST operations returns 200 with both statuses `201` (RFC 7644 Section 3.7).
  - `bulkId` cross-references resolve correctly.
  - `failOnErrors: 1` stops processing after first error.
  - DELETE + POST in same request both succeed.
  - Update `ServiceProviderConfig.bulk.supported` to `true`.
  - Tests: B-001 through B-008.
  - **Effort**: L

### Group G: ETag and Concurrency

#### SCIM-15: ETag / If-Match / If-None-Match (RFC 7644 Section 3.14)
- **Priority**: P2
- **Files**: `services/identity/internal/scim/handler.go`, `services/identity/internal/scim/etag.go` (new)
- **API Design**:
  ```go
  // etag.go
  func computeETag(u *domain.User) string {
      return fmt.Sprintf("W/\"%x\"", u.UpdatedAt.UnixNano())
  }
  func checkIfMatch(r *http.Request, etag string) bool
  func checkIfNoneMatch(r *http.Request, etag string) bool
  ```
  In `getUser()`: set `ETag` response header. Check `If-None-Match` → return `304 Not Modified` if match. In `replaceUser()`, `patchUser()`, `deleteUser()`: check `If-Match` header. If present and ETag doesn't match → return `412 Precondition Failed`. If `If-Match: *` → proceed only if resource exists. Update `ServiceProviderConfig.etag.supported` to `true`.
- **Acceptance Criteria**:
  - GET `/Users/{id}` response includes `ETag` header (RFC 7644 Section 3.14).
  - GET with matching `If-None-Match` returns `304 Not Modified`.
  - PUT/PATCH/DELETE with stale `If-Match` returns `412 Precondition Failed`.
  - `If-Match: *` with existing resource succeeds; with non-existent returns 404.
  - Tests: E-001 through E-010.

### Group H: Schemas Endpoint and Search

#### SCIM-16: Schemas Endpoint (RFC 7643 Section 7)
- **Priority**: P2
- **Files**: `services/identity/internal/scim/schemas.go` (new), `services/identity/internal/scim/handler.go` (register route)
- **API Design**:
  ```go
  // schemas.go
  func (h *Handler) handleSchemas(w http.ResponseWriter, r *http.Request)
  ```
  Register: `mux.HandleFunc("/scim/v2/Schemas", h.handleSchemas)` and `mux.HandleFunc("/scim/v2/Schemas/", h.handleSchemaByID)`. Return schema definitions for `urn:ietf:params:scim:schemas:core:2.0:User`, `urn:ietf:params:scim:schemas:core:2.0:Group`, and `urn:ietf:params:scim:schemas:extension:enterprise:2.0:User`. Each schema includes: `id`, `name`, `description`, `attributes[]` (with `name`, `type`, `multiValued`, `required`, `mutability`, `returned`, `uniqueness`, `subAttributes[]` for complex types).
- **Acceptance Criteria**:
  - GET `/Schemas` returns `ListResponse` with all schema definitions (RFC 7643 Section 7).
  - GET `/Schemas/urn:ietf:params:scim:schemas:core:2.0:User` returns User schema with all attributes.
  - GET `/Schemas/urn:ietf:params:scim:schemas:core:2.0:Group` returns Group schema.
  - Schema attributes include `mutability`, `returned`, `uniqueness` fields.
  - Tests: SD-005, SD-006, SD-007.

#### SCIM-17: POST `.search` Endpoint (RFC 7644 Section 3.4.3)
- **Priority**: P3
- **Files**: `services/identity/internal/scim/handler.go`, `services/identity/internal/server/http.go`
- **API Design**: Register `mux.HandleFunc("/scim/v2/Users/.search", h.handleUserSearch)` and `mux.HandleFunc("/scim/v2/Groups/.search", h.handleGroupSearch)`. Accept POST with body:
  ```go
  type SearchRequest struct {
      Schemas          []string `json:"schemas"`
      Attributes       []string `json:"attributes"`
      ExcludedAttributes []string `json:"excludedAttributes"`
      Filter           string   `json:"filter"`
      SortBy           string   `json:"sortBy"`
      SortOrder        string   `json:"sortOrder"`
      StartIndex       int      `json:"startIndex"`
      Count            int      `json:"count"`
  }
  ```
  Parse body, delegate to same logic as GET list with query params.
- **Acceptance Criteria**:
  - POST `/Users/.search` with filter in body returns filtered `ListResponse` (RFC 7644 Section 3.4.3).
  - POST `/Groups/.search` works similarly.
  - Body parameters (`filter`, `sortBy`, `startIndex`, `count`) are parsed correctly.

### Group I: Attribute Projection

#### SCIM-18: `attributes` / `excludedAttributes` Support (RFC 7644 Section 3.4.2.5 / 3.9)
- **Priority**: P2
- **Files**: `services/identity/internal/scim/handler.go`, `services/identity/internal/scim/projection.go` (new)
- **API Design**:
  ```go
  // projection.go
  func projectResource(resource map[string]any, attributes, excludedAttributes []string) map[string]any
  ```
  In `listUsers()`, `getUser()`, `listGroups()`, `getGroup()`: read `attributes` and `excludedAttributes` query params (comma-separated). Apply projection to each resource before serialization. `attributes` = whitelist (only return these). `excludedAttributes` = blacklist (omit these). Always include `schemas`, `id`, and `meta.resourceType`.
- **Acceptance Criteria**:
  - `GET /Users?attributes=userName,displayName` returns only `userName`, `displayName`, `id`, `schemas` (RFC 7644 Section 3.9).
  - `GET /Users?excludedAttributes=emails,phoneNumbers` omits those fields.
  - `id` and `schemas` are always present.
  - Works with both `GET /Users` and `GET /Users/{id}`.

### Group J: Error Handling

#### SCIM-19: `scimType` in Error Responses (RFC 7644 Section 3.12)
- **Priority**: P1
- **Files**: `services/identity/internal/scim/handler.go`
- **API Design**: Extend `ErrorResponse`:
  ```go
  type ErrorResponse struct {
      Schemas  []string `json:"schemas"`
      Detail   string   `json:"detail"`
      Status   string   `json:"status"`
      ScimType string   `json:"scimType,omitempty"`
  }
  ```
  Add helper: `func writeSCIMErrorWithType(w http.ResponseWriter, status int, scimType, detail string)`. Map error scenarios: `invalidFilter` (400), `invalidSyntax` (400), `invalidPath` (400), `tooMany` (400), `uniqueness` (409), `invalidValue` (400).
- **Acceptance Criteria**:
  - Invalid JSON body returns `400` with `scimType: invalidSyntax` (RFC 7644 Section 3.12).
  - Invalid filter returns `400` with `scimType: invalidFilter`.
  - Unknown PATCH path returns `400` with `scimType: invalidPath`.
  - Duplicate `userName` returns `409` with `scimType: uniqueness`.
  - Tests: ER-001 through ER-013.

#### SCIM-20: `userName` Uniqueness and Validation (RFC 7643 Section 4.1.1)
- **Priority**: P1
- **Files**: `services/identity/internal/scim/handler.go`
- **API Design**: In `createUser()`, before calling `svc.CreateUser()`:
  1. Validate `scimUser.UserName != ""` → if empty, return `400` with `scimType: invalidSyntax`.
  2. Validate `scimUser.Emails` has valid format → if invalid, return `400` with `scimType: invalidValue`.
  3. Distinguish error types from `svc.CreateUser()`: if error is `ErrAlreadyExists`, return `409` with `scimType: uniqueness`; if `ErrInvalidArgument`, return `400` with `scimType: invalidSyntax`; otherwise `500`.
  Remove the current blanket `writeSCIMError(w, http.StatusConflict, ...)` catch-all.
- **Acceptance Criteria**:
  - POST `/Users` without `userName` returns `400` with `scimType: invalidSyntax` (RFC 7643 Section 4.1.1).
  - POST `/Users` with duplicate `userName` returns `409` with `scimType: uniqueness`.
  - POST `/Users` with invalid email format returns `400` with `scimType: invalidValue`.
  - Non-conflict errors return appropriate status codes, not blanket 409.
  - Tests: U-002, U-003, ER-006.

---

## Backlog for arch (pkg, sdk, console, deploy, docs, CI/CD)

### P0 — Core
- [DONE] Quick start guide: 5-minute integration tutorial
- [DONE] Go SDK: full client implementation (42 tests, 65.8% coverage)
- [DONE] Node.js SDK: TypeScript client + Express middleware
- [DONE] Java SDK: client + exception + README
- [TODO] Docker Compose production hardening (secrets, TLS, volumes)

### P1 — Enterprise
- [TODO] Helm chart for Kubernetes deployment
- [TODO] Terraform module for AWS/GCP deployment
- [TODO] Architecture documentation (C4 model diagrams)
- [TODO] Security whitepaper (threat model + mitigations)

### P2 — Enhancement
- [TODO] Performance benchmark suite (k6 load tests)
- [DONE] Migration guide: Auth0 (docs/migration-from-auth0.md assigned to doc)
- [DONE] Migration guide: Keycloak (docs/migration-from-keycloak.md assigned to doc)
- [DONE] Plugin system design (docs/plugin-api-reference.md + docs/plugin-development.md)
- [TODO] Brand customization (login page theming)

### P3 — Innovation
- [TODO] AI-powered anomaly detection in audit logs
- [TODO] Natural language policy query ("can user X access Y?")
- [TODO] Identity graph visualization (user → roles → orgs → permissions)
