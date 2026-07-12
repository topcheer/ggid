# Auth Service API Reference

Complete REST API reference for GGID's Auth service — register, login, refresh, MFA, WebAuthn, LDAP, password reset, sessions.

**Base URL**: `https://api.ggid.example.com`

## Registration

```
POST /api/v1/auth/register
```
```json
{"username":"alice","email":"alice@example.com","password":"SecurePass1!"}
```
**Response** (201): `{"id":"uuid","username":"alice","email":"alice@example.com"}`

## Login

```
POST /api/v1/auth/login
```
```json
{"username":"alice","password":"SecurePass1!"}
```
**Response** (200):
```json
{"access_token":"eyJ...","refresh_token":"eyJ...","expires_in":900,"token_type":"Bearer"}
```

**MFA Required** (200):
```json
{"mfa_required":true,"mfa_token":"mfa_temp_xxx"}
```

## Token Refresh

```
POST /api/v1/auth/refresh
{"refresh_token":"eyJ..."}
```

## Logout

```
POST /api/v1/auth/logout
```
Revokes session via jti blacklist.

## MFA — TOTP

### Enroll
```
POST /api/v1/auth/mfa/totp/enroll
```
**Response**: `{"secret":"BASE32SECRET","qr_url":"otpauth://tott/..."}`

### Verify Enrollment
```
POST /api/v1/auth/mfa/totp/verify
{"secret":"BASE32SECRET","code":"123456"}
```

### Verify MFA (Login Step-Up)
```
POST /api/v1/auth/mfa/verify
{"mfa_token":"mfa_temp_xxx","code":"123456"}
```

### Disable TOTP
```
DELETE /api/v1/auth/mfa/totp
```

## MFA — WebAuthn

### Registration
```
POST /api/v1/webauthn/register/begin   {"authenticator_attachment":"platform"}
POST /api/v1/webauthn/register/finish  {"attestation_object":"...","client_data_json":"..."}
```

### Authentication
```
POST /api/v1/webauthn/auth/begin   {"username":"alice"}
POST /api/v1/webauthn/auth/finish   {"assertion":"...","client_data_json":"..."}
```

## Password Reset

```
POST /api/v1/auth/password-reset/request   {"email":"alice@example.com"}
POST /api/v1/auth/password-reset/confirm   {"token":"reset_token","new_password":"NewPass1!"}
```

## LDAP Authentication

LDAP auth is transparent — if `LDAP_URL` is configured, GGID's authprovider chain tries Local first, then LDAP.

## Impersonation

```
POST /api/v1/auth/impersonate
{"user_id":"target-uuid"}
```
Requires `admin` scope.

## Session Management

```
GET /api/v1/sessions
DELETE /api/v1/sessions/{id}
DELETE /api/v1/sessions?user_id={user_id}
```

## Risk Scoring

```
GET /api/v1/auth/risk-scoring/config
PUT /api/v1/auth/risk-scoring/config
```

## Error Responses

| Status | Code | Description |
|--------|------|-------------|
| 401 | `unauthorized` | Wrong password |
| 403 | `forbidden` | Account locked |
| 429 | `rate_limit_exceeded` | Too many login attempts |

## See Also
- [REST API Reference](rest-api.md)
- [OAuth API](oauth.md)
- [Password Policy Guide](../guides/password-policy-guide.md)
