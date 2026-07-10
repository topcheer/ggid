# Design: Authentication Flow

> **Status:** Implemented

Complete authentication flow documentation covering all methods supported by GGID.

---

## Table of Contents

- [Password Login](#password-login)
- [Registration](#registration)
- [Token Refresh (Rotation)](#token-refresh-rotation)
- [MFA: TOTP](#mfa-totp)
- [MFA: WebAuthn / Passkey](#mfa-webauthn--passkey)
- [Step-Up Authentication](#step-up-authentication)
- [Magic Link (Passwordless)](#magic-link-passwordless)
- [Social Login (OAuth2)](#social-login-oauth2)
- [LDAP / AD Login](#ldap--ad-login)
- [Logout & Session Revocation](#logout--session-revocation)
- [Security Considerations](#security-considerations)

---

## Password Login

```
Client                Gateway              Auth Service          Redis
  │                     │                     │                    │
  │ POST /auth/login    │                     │                    │
  │ {username, password}│                     │                    │
  ├────────────────────►│                     │                    │
  │                     │ rate limit check    │                    │
  │                     ├────────────────────►│                    │
  │                     │                     │ lookup credential  │
  │                     │                     ├───────────────────►│
  │                     │                     │ verify password    │
  │                     │                     │ (Argon2id)         │
  │                     │                     │                    │
  │                     │     200 + tokens    │                    │
  │                     │◄────────────────────┤                    │
  │   {access_token,    │                     │                    │
  │    refresh_token}   │                     │                    │
  │◄────────────────────┤                     │                    │
```

**Steps:**
1. Client sends `POST /api/v1/auth/login` with `{username, password}`
2. Gateway checks rate limit (5 attempts/min per IP)
3. Auth service looks up credential by username within tenant
4. Verifies password using Argon2id (constant-time comparison)
5. If MFA is enabled → returns `mfa_token` instead of access token (see [MFA flow](#mfa-totp))
6. Generates JWT (RS256) with claims: `sub`, `tenant_id`, `roles`, `scopes`
7. Returns `{access_token, refresh_token, expires_in}`

**Error cases:**
- Invalid credentials → `401 Unauthorized`
- Account locked → `403 Forbidden`
- Rate limit exceeded → `429 Too Many Requests`

---

## Registration

```
Client                Gateway              Auth Service          Identity Svc
  │                     │                     │                    │
  │ POST /auth/register │                     │                    │
  │ {username,email,pwd}│                     │                    │
  ├────────────────────►│                     │                    │
  │                     │ rate limit (3/min)  │                    │
  │                     ├────────────────────►│                    │
  │                     │                     │ validate input     │
  │                     │                     │ check uniqueness   │
  │                     │                     │ hash password      │
  │                     │                     │ (Argon2id)         │
  │                     │                     │ create credential  │
  │                     │                     ├───────────────────►│ create user
  │                     │                     │                    │ record
  │                     │                     │ publish audit      │
  │                     │                     │ event (NATS)       │
  │                     │   201 + user_id     │                    │
  │                     │◄────────────────────┤                    │
  │   201 Created       │                     │                    │
  │◄────────────────────┤                     │                    │
```

**Key validations:**
- Username unique within tenant (`UNIQUE(tenant_id, username)`)
- Email format valid
- Password meets policy (min 8, upper/lower/digit/special)
- Rate limited (3/min per IP)

---

## Token Refresh (Rotation)

```
Client                Gateway              Auth Service          Redis
  │                     │                     │                    │
  │ POST /auth/refresh  │                     │                    │
  │ {refresh_token}     │                     │                    │
  ├────────────────────►│                     │                    │
  │                     ├────────────────────►│                    │
  │                     │                     │ verify refresh     │
  │                     │                     │ token signature    │
  │                     │                     │ check blocklist    │
  │                     │                     ├───────────────────►│
  │                     │                     │ invalidate old     │
  │                     │                     │ refresh token      │
  │                     │                     ├───────────────────►│
  │                     │                     │ issue new pair     │
  │                     │     200 + new pair  │                    │
  │                     │◄────────────────────┤                    │
  │  {access_token,     │                     │                    │
  │   refresh_token}    │                     │                    │
  │◄────────────────────┤                     │                    │
```

**Security:**
- Refresh tokens are **rotated** — old token is invalidated immediately
- If a used refresh token is presented again → detected as token theft → all tokens for that user are revoked
- Tokens checked against Redis blocklist before issuing new pair

---

## MFA: TOTP

### Setup Flow

```
Client                Auth Service
  │                     │
  │ POST /auth/mfa/setup│
  ├────────────────────►│
  │                     │ generate TOTP secret
  │                     │ store temporarily
  │   {secret, qr_code} │
  │◄────────────────────┤
  │                     │
  │ User scans QR code  │
  │ into Google Auth    │
  │                     │
  │ POST /auth/mfa/verify
  │ {code: "123456"}    │
  ├────────────────────►│
  │                     │ verify TOTP code
  │                     │ store secret permanently
  │   200 OK            │
  │◄────────────────────┤
```

### Login with MFA

```
Client              Gateway          Auth Service
  │                   │                 │
  │ POST /auth/login  │                 │
  │ {username,pass}   │                 │
  ├──────────────────►│────────────────►│
  │                   │                 │ password OK
  │                   │                 │ MFA required
  │   {mfa_required: true,             │
  │    mfa_token: "temp_xxx"}          │
  │◄──────────────────┤◄────────────────┤
  │                   │                 │
  │ POST /auth/mfa/login               │
  │ {mfa_token, code: "123456"}        │
  ├──────────────────►│────────────────►│
  │                   │                 │ verify TOTP code
  │                   │                 │ issue JWT pair
  │   {access_token, refresh_token}    │
  │◄──────────────────┤◄────────────────┤
```

---

## MFA: WebAuthn / Passkey

### Registration Flow

```
Browser              Auth Service          Authenticator
  │                     │                     │
  │ POST /auth/webauthn │                     │
  │   /register/begin   │                     │
  ├────────────────────►│                     │
  │                     │ generate challenge  │
  │                     │ store challenge     │
  │  PublicKeyCredential │                     │
  │  CreationOptions    │                     │
  │◄────────────────────┤                     │
  │                     │                     │
  │ navigator.credentials.create()            │
  ├──────────────────────────────────────────►│
  │                     │                     │ user verifies
  │                     │                     │ (biometric/PIN)
  │  AttestationResponse│                     │
  │◄──────────────────────────────────────────┤
  │                     │                     │
  │ POST /auth/webauthn │                     │
  │   /register/finish  │                     │
  │  {attestationResponse}                    │
  ├────────────────────►│                     │
  │                     │ verify attestation  │
  │                     │ store credential    │
  │   200 OK            │                     │
  │◄────────────────────┤                     │
```

### Login Flow

```
Browser              Auth Service          Authenticator
  │                     │                     │
  │ POST /auth/webauthn │                     │
  │   /login/begin      │                     │
  │  {username}         │                     │
  ├────────────────────►│                     │
  │                     │ get stored credential
  │                     │ generate challenge  │
  │  PublicKeyCredential │                     │
  │  RequestOptions     │                     │
  │◄────────────────────┤                     │
  │                     │                     │
  │ navigator.credentials.get()               │
  ├──────────────────────────────────────────►│
  │                     │                     │ user verifies
  │  AssertionResponse  │                     │
  │◄──────────────────────────────────────────┤
  │                     │                     │
  │ POST /auth/webauthn │                     │
  │   /login/finish     │                     │
  │  {assertionResponse}│                     │
  ├────────────────────►│                     │
  │                     │ verify signature    │
  │                     │ issue JWT pair      │
  │  {access_token,     │                     │
  │   refresh_token}    │                     │
  │◄────────────────────┤                     │
```

---

## Step-Up Authentication

Used when a user is already logged in but needs additional verification for sensitive operations (e.g., changing security settings, transferring funds).

```
Client              Gateway          Auth Service
  │                   │                 │
  │ Already has JWT   │                 │
  │                   │                 │
  │ GET /auth/step-up-check?scope=security
  ├──────────────────►│────────────────►│
  │                   │                 │ check if recent
  │                   │                 │ step-up exists
  │   {required: true}│                 │
  │◄──────────────────┤◄────────────────┤
  │                   │                 │
  │ POST /auth/step-up│                 │
  │ {scope, methods}  │                 │
  ├──────────────────►│────────────────►│
  │                   │                 │ create challenge
  │   {challenge_id,  │                 │
  │    methods:[totp]}│                 │
  │◄──────────────────┤◄────────────────┤
  │                   │                 │
  │ User enters code  │                 │
  │                   │                 │
  │ POST /auth/stepup/verify            │
  │ {challenge_id, code}                │
  ├──────────────────►│────────────────►│
  │                   │                 │ verify code
  │                   │                 │ issue elevated JWT
  │   {access_token,  │                 │
  │    elevated: true}│                 │
  │◄──────────────────┤◄────────────────┤
```

---

## Magic Link (Passwordless)

```
Browser              Auth Service          Email Server
  │                     │                     │
  │ POST /auth/magic-link                    │
  │  {email}           │                     │
  ├────────────────────►│                     │
  │                     │ generate token      │
  │                     │ store in Redis      │
  │                     │ (TTL: 15 min)       │
  │                     │ send email          │
  │                     ├────────────────────►│
  │   200 OK (always)  │                     │
  │◄────────────────────┤                     │
  │                     │                     │
  │ User clicks link in email                │
  │                     │                     │
  │ POST /auth/magic-link/verify             │
  │  {token}           │                     │
  ├────────────────────►│                     │
  │                     │ verify token        │
  │                     │ delete from Redis   │
  │                     │ issue JWT pair      │
  │  {access_token,    │                     │
  │   refresh_token}   │                     │
  │◄────────────────────┤                     │
```

---

## Social Login (OAuth2)

```
Browser              Gateway          Auth Service      Google/IdP
  │                   │                 │                 │
  │ GET /auth/social/google              │                 │
  ├──────────────────►│────────────────►│                 │
  │                   │                 │ build OAuth URL │
  │   302 Redirect    │                 │                 │
  │◄──────────────────┤◄────────────────┤                 │
  │                   │                 │                 │
  │ Browser redirects to Google         │                 │
  ├──────────────────────────────────────────────────────►│
  │                   │                 │    user consents │
  │   302 callback?code=xxx             │                 │
  │◄──────────────────────────────────────────────────────┤
  │                   │                 │                 │
  │ GET /auth/social/google/callback?code=xxx             │
  ├──────────────────►│────────────────►│                 │
  │                   │                 │ exchange code   │
  │                   │                 ├────────────────►│
  │                   │                 │  get user info  │
  │                   │                 │◄────────────────┤
  │                   │                 │ auto-provision  │
  │                   │                 │ (if new user)   │
  │                   │                 │ issue JWT pair  │
  │  {access_token}   │                 │                 │
  │◄──────────────────┤◄────────────────┤                 │
```

---

## LDAP / AD Login

```
Client              Gateway          Auth Service          LDAP Server
  │                   │                 │                     │
  │ POST /auth/login  │                 │                     │
  │ {username,pass}   │                 │                     │
  ├──────────────────►│────────────────►│                     │
  │                   │                 │ try Local provider  │
  │                   │                 │ (not found)         │
  │                   │                 │                     │
  │                   │                 │ try LDAP provider   │
  │                   │                 │ bind as user        │
  │                   │                 ├────────────────────►│
  │                   │                 │    bind success     │
  │                   │                 │◄────────────────────┤
  │                   │                 │                     │
  │                   │                 │ search user entry   │
  │                   │                 ├────────────────────►│
  │                   │                 │◄────────────────────┤
  │                   │                 │                     │
  │                   │                 │ auto-provision      │
  │                   │                 │ (if enabled)        │
  │                   │                 │ issue JWT pair      │
  │  {access_token}   │                 │                     │
  │◄──────────────────┤◄────────────────┤                     │
```

---

## Logout & Session Revocation

```
Client              Gateway          Auth Service          Redis
  │                   │                 │                     │
  │ POST /auth/logout │                 │                     │
  │ {access_token}    │                 │                     │
  ├──────────────────►│────────────────►│                     │
  │                   │                 │ add to blocklist    │
  │                   │                 ├───────────────────►│
  │                   │                 │  SET jti + TTL      │
  │   200 OK          │                 │                     │
  │◄──────────────────┤◄────────────────┤                     │
```

**Logout-all (revoke all sessions):**
- Iterates all sessions for the user
- Adds each JWT's `jti` to Redis blocklist
- Revokes all refresh tokens

---

## Security Considerations

### Password Storage

- **Argon2id** (RFC 9106) — memory-hard, resistant to GPU/ASIC brute-force
- Parameters: `time=1`, `memory=64MB`, `parallelism=2` (tunable)
- Each password gets a unique random salt

### JWT Security

| Aspect | Implementation |
|--------|---------------|
| Signing algorithm | RS256 (RSA 2048-bit) |
| Access token TTL | 1 hour (configurable down to 15 min) |
| Refresh token TTL | 30 days (rotated on each use) |
| Key rotation | Dual-key period (JWKS supports multiple keys) |
| Revocation | Redis blocklist (checked at refresh) |

### Rate Limiting

| Endpoint | Limit | Purpose |
|----------|-------|---------|
| `/auth/login` | 5/min/IP | Brute-force protection |
| `/auth/register` | 3/min/IP | Account spam prevention |
| `/auth/password/forgot` | 3/min/IP | Email enumeration prevention |

### Audit Trail

Every authentication event publishes an audit event via NATS:
- `user.login` (success)
- `user.login_failed` (failure)
- `user.register`
- `user.logout`
- `mfa.enable` / `mfa.disable`
- `password.reset` / `password.change`

### Token Theft Detection

If refresh token rotation detects reuse (same refresh token presented twice):
1. Both tokens are revoked
2. All sessions for that user are invalidated
3. An audit event (`security.token_theft`) is published
4. User must re-authenticate from scratch
