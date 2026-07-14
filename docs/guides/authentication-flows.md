# Authentication Flows Guide

This guide documents the 7 authentication flows supported by GGID, with sequence diagrams for each.

> **Related**: [OAuth Flows](../oauth-flows.md), [OAuth Flows Guide](../oauth-flows-guide.md)

## Overview

| Flow | Use Case | Protocol | MFA |
|------|----------|----------|-----|
| Password | Web/mobile login | Auth API | Optional |
| Passwordless | FIDO2/WebAuthn | WebAuthn | Built-in |
| SSO (SAML) | Enterprise federation | SAML 2.0 | Optional |
| SSO (OIDC) | Social/federated login | OIDC | Optional |
| OAuth Authorization Code | Third-party apps | OAuth 2.0 | Optional |
| Device Authorization | IoT/TV/CLI | RFC 8628 | Optional |
| Token Exchange | Agent delegation | RFC 8693 | N/A |

## 1. Password Flow

```
Client                    Gateway                   Auth Service
  │                          │                          │
  │── POST /auth/login ────→ │                          │
  │   {username, password}   │── forward ─────────────→ │
  │                          │                          │
  │                          │    1. Verify password (Argon2id)
  │                          │    2. Check MFA enrollment
  │                          │    3. Check account lockout
  │                          │    4. Check rate limit
  │                          │    5. Generate JWT (RS256)
  │                          │    6. Store jti in Redis
  │                          │                          │
  │←─── 200 {access_token} ──│←─── TokenSet ────────────│
  │     {refresh_token}      │                          │
```

```bash
curl -X POST https://api.ggid.example.com/api/v1/auth/login \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{"username":"alice","password":"SecurePass1!"}'
```

If MFA is enrolled, the response includes `mfa_required: true` with an `mfa_token` for step-up.

## 2. Passwordless (WebAuthn)

### Registration

```
Browser                 Gateway              Auth Service
  │                        │                      │
  │─ POST /webauthn/register/begin ─→             │
  │                        │── forward ─────────→ │
  │                        │                      │
  │                        │    1. Generate challenge
  │                        │    2. Store in session
  │                        │                      │
  │←── PublicKeyCredentialCreationOptions ────────│
  │                        │                      │
  │  navigator.credentials.create()               │
  │  → User touches security key / biometric       │
  │                        │                      │
  │─ POST /webauthn/register/finish ─→             │
  │   {attestation, clientData}                    │
  │                        │── forward ─────────→ │
  │                        │                      │
  │                        │    1. Verify attestation
  │                        │    (7 formats: none, packed,
  │                        │     fido-u2f, android-key,
  │                        │     android-safetynet, tpm, apple)
  │                        │    2. Store public key
  │                        │    3. Mark MFA enrolled
  │                        │                      │
  │←────── 200 {success} ─────────────────────────│
```

### Authentication

```
Browser                 Gateway              Auth Service
  │                        │                      │
  │─ POST /webauthn/auth/begin ─→                 │
  │   {username}            │                      │
  │                        │── forward ─────────→ │
  │                        │    Generate assertion challenge
  │←── PublicKeyCredentialRequestOptions ─────────│
  │                        │                      │
  │  navigator.credentials.get()                  │
  │  → User authenticates with key/biometric      │
  │                        │                      │
  │─ POST /webauthn/auth/finish ─→                 │
  │   {assertion, clientData}                     │
  │                        │── forward ─────────→ │
  │                        │    1. Verify signature
  │                        │    2. Check counter (replay)
  │                        │    3. Issue JWT
  │←────── 200 {access_token} ────────────────────│
```

## 3. SSO — SAML 2.0

```
Browser          Gateway       OAuth Service         IdP
  │                │                │                 │
  │── GET /saml/login ──→           │                 │
  │                │── forward ───→ │                 │
  │                │                │── AuthnRequest →│
  │←──────── Redirect to IdP ────────────────────────│
  │                │                │                 │
  │── POST credentials to IdP ──────────────────────→│
  │                │                │                 │
  │←── Redirect with SAML Response ──────────────────│
  │                │                │                 │
  │── POST /saml/acs (SAML Response) ─→              │
  │                │── forward ───→ │                 │
  │                │                │  1. Verify XML signature
  │                │                │  2. Verify assertion conditions
  │                │                │  3. Extract attributes (email, name)
  │                │                │  4. Auto-provision user (if new)
  │                │                │  5. Issue JWT
  │←────── 200 {access_token} ──────────────────────│
```

**SP Metadata**: `GET /.well-known/saml-metadata`

## 4. SSO — OIDC / Social Login

```
Browser          Gateway       OAuth Service         Provider
  │                │                │                 │
  │── GET /oauth/authorize? ──→     │                 │
  │   client_id, redirect_uri,      │                 │
  │   scope, state, code_challenge  │                 │
  │                │── forward ───→ │                 │
  │                │                │  Validate request│
  │                │                │  Redirect to provider
  │←──────── Redirect to Google/GitHub/Microsoft ────│
  │                │                │                 │
  │── User authorizes at provider ──────────────────→│
  │←── Redirect with authorization code ─────────────│
  │                │                │                 │
  │── GET /oauth/callback?code=xxx ─→                │
  │                │── forward ───→ │                 │
  │                │                │  1. Exchange code for provider tokens
  │                │                │── token exchange →│
  │                │                │←── provider tokens ─│
  │                │                │  2. Get user info from provider
  │                │                │── userinfo request →│
  │                │                │←── user profile ──│
  │                │                │  3. Auto-provision or link account
  │                │                │  4. Issue GGID JWT
  │←────── 200 {access_token} ──────────────────────│
```

**Supported providers**: Google, GitHub, Microsoft, Discord, Slack, LinkedIn, GitLab, Apple, generic OIDC.

## 5. OAuth Authorization Code Flow (Third-Party Apps)

```
Third-Party      Gateway       OAuth Service
    App            │                │
  │                │                │
  │── Redirect user to ──→          │
  │   /oauth/authorize?             │
  │   response_type=code            │
  │   &client_id=xxx                │
  │   &redirect_uri=xxx             │
  │   &scope=users:read             │
  │   &state=random                 │
  │   &code_challenge=xxx  (PKCE)   │
  │                │── forward ───→ │
  │                │                │  1. Validate client + redirect URI
  │                │                │  2. Store state in Redis
  │                │                │  3. Show consent screen
  │                │                │
  │←── User approves consent ───────│
  │                │                │
  │←── Redirect to app callback ────│
  │   ?code=auth_code               │
  │   &state=random                 │
  │                │                │
  │── POST /oauth/token ──→         │
  │   grant_type=authorization_code │
  │   code=auth_code                │
  │   code_verifier=xxx  (PKCE)     │
  │                │── forward ───→ │
  │                │                │  1. Verify code + PKCE
  │                │                │  2. Verify state
  │                │                │  3. Issue access + refresh tokens
  │←────── 200 {access_token, refresh_token} ───────│
```

## 6. Device Authorization Grant (RFC 8628)

```
TV/Device         Gateway       OAuth Service         User (Phone)
  │                │                │                    │
  │── POST /oauth/device ──→        │                    │
  │                │── forward ───→ │                    │
  │                │                │  Generate device_code
  │                │                │  + user_code
  │                │                │  + verification_uri
  │←── {device_code, user_code,     │                    │
  │     verification_uri} ──────────│                    │
  │                │                │                    │
  │  Display: "Go to ggid.example.com/device           │
  │   and enter code: ABCD-EFGH"                        │
  │                │                │                    │
  │── Poll POST /oauth/token ──→    │     ┌── User opens browser ──→ │
  │   grant_type=urn:ietf:params:   │     │   enters user_code       │
  │   oauth:grant-type:device_code  │     │   logs in + authorizes   │
  │                │── forward ───→ │←────┘                         │
  │←── 400 {authorization_pending} ─│                    │
  │                │                │                    │
  │── Poll again (slow_down) ──→    │                    │
  │←── 400 {authorization_pending} ─│                    │
  │                │                │                    │
  │── Poll again ──→                │                    │
  │                │                │  User approved!     │
  │←── 200 {access_token} ──────────│                    │
```

## 7. Token Exchange (Agent Delegation)

```
AI Agent         Gateway       OAuth Service
  │                │                │
  │── POST /api/v1/agents/exchange ─→│
  │   {agent_id, scope,             │
  │    delegation_chain}            │
  │   Bearer: admin_token           │
  │                │── forward ───→ │
  │                │                │  1. Verify agent is registered
  │                │                │  2. Check agent is not suspended
  │                │                │  3. Verify delegation depth
  │                │                │  4. Scope narrowing check
  │                │                │  5. Issue agent JWT with
  │                │                │     delegation_chain + mcp_servers
  │←── 200 {access_token,            │
  │     agent_id, delegation_chain} ─│
```

## MFA Step-Up Flow

```
After password login (MFA enrolled):

Client                 Auth Service
  │                        │
  │── POST /auth/login ──→ │
  │←── 200 {mfa_required: true,
  │         mfa_token: "xxx"} │
  │                        │
  │── POST /auth/mfa/verify ─→
  │   {mfa_token, totp_code}  │
  │                        │
  │                        │  Verify TOTP (RFC 6238)
  │                        │  Issue JWT
  │←── 200 {access_token} │
```

## See Also

- [OAuth Flows](../oauth-flows.md)
- [Passwordless Setup](passwordless-setup.md)
- [Per-Tenant IdP](per-tenant-idp.md)
- [AI Agent Identity](ai-agent-identity.md)
- MFA Setup
