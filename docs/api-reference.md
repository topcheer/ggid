# GGID API Reference

Complete endpoint inventory organized by service. All authenticated endpoints require `Authorization: Bearer <JWT>` and `X-Tenant-ID` header.

> **Base URL**: `http://localhost:8080`  
> **Swagger UI**: `http://localhost:8080/docs`  
> **OpenAPI Spec**: `http://localhost:8080/swagger.json`

---

## Auth Service

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/v1/auth/register` | — | Self-service user registration |
| POST | `/api/v1/auth/login` | — | Login with username/password |
| POST | `/api/v1/auth/logout` | Bearer | Logout and invalidate session |
| POST | `/api/v1/auth/refresh` | — | Refresh access token |
| GET | `/api/v1/auth/profile` | Bearer | Get current user profile |
| PUT | `/api/v1/auth/profile` | Bearer | Update own profile |
| GET | `/api/v1/auth/verify-email` | — | Verify email with token |
| POST | `/api/v1/auth/password/forgot` | — | Request password reset |
| POST | `/api/v1/auth/password/reset` | — | Reset password with token |
| POST | `/api/v1/auth/password/change` | Bearer | Change password |
| POST | `/api/v1/auth/password/strength` | — | Evaluate password strength (0-4) |
| GET | `/api/v1/auth/password/policy` | — | Get password policy |

### Sessions
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/auth/sessions` | Bearer | List active sessions |
| DELETE | `/api/v1/auth/sessions/{id}` | Bearer | Revoke a session |

### MFA
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/v1/auth/mfa/enroll` | Bearer | Enroll TOTP MFA |
| POST | `/api/v1/auth/mfa/verify` | Bearer | Verify MFA code |
| POST | `/api/v1/auth/mfa/disable` | Bearer | Disable MFA |
| GET | `/api/v1/auth/mfa/backup-codes` | Bearer | List backup codes |
| POST | `/api/v1/auth/mfa/backup-codes` | Bearer | Generate new backup codes |

### WebAuthn
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/v1/auth/webauthn/begin` | Bearer | Begin registration |
| POST | `/api/v1/auth/webauthn/finish` | Bearer | Finish registration |
| POST | `/api/v1/auth/webauthn/login/begin` | — | Begin WebAuthn login |
| POST | `/api/v1/auth/webauthn/login/finish` | — | Finish WebAuthn login |
| GET | `/api/v1/auth/webauthn/aaguid` | Bearer | List AAGUID allowlist |
| POST | `/api/v1/auth/webauthn/aaguid` | Bearer | Add AAGUID to allowlist |
| DELETE | `/api/v1/auth/webauthn/aaguid/{id}` | Bearer | Remove AAGUID |

### Conditional Access (CAP)
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/auth/conditional-access/policies` | Bearer | List CAP policies |
| POST | `/api/v1/auth/conditional-access/policies` | Bearer | Create CAP policy |
| PUT | `/api/v1/auth/conditional-access/policies/{id}` | Bearer | Update CAP policy |
| DELETE | `/api/v1/auth/conditional-access/policies/{id}` | Bearer | Delete CAP policy |
| POST | `/api/v1/auth/conditional-access/evaluate` | Bearer | Evaluate conditions |

### TAP (Temporary Access Pass)
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/v1/auth/tap` | Bearer | Issue single TAP |
| POST | `/api/v1/auth/tap/batch` | Bearer | Batch issue TAPs |
| GET | `/api/v1/auth/tap/policy` | Bearer | Get TAP policy |
| PUT | `/api/v1/auth/tap/policy` | Bearer | Update TAP policy |

### Break Glass
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/v1/auth/break-glass/activate` | Bearer | Activate break-glass |
| GET | `/api/v1/auth/break-glass/history` | Bearer | Break-glass history |

---

## Identity Service

### Users
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/users` | Bearer | List users (paginated) |
| POST | `/api/v1/users` | Bearer | Create user |
| GET | `/api/v1/users/{id}` | Bearer | Get user by ID |
| PUT | `/api/v1/users/{id}` | Bearer | Update user |
| DELETE | `/api/v1/users/{id}` | Bearer | Delete user |
| POST | `/api/v1/users/{id}/lock` | Bearer | Lock user |
| POST | `/api/v1/users/{id}/unlock` | Bearer | Unlock user |
| POST | `/api/v1/users/import` | Bearer | Import users (CSV) |
| GET | `/api/v1/users/export` | Bearer | Export users (CSV) |

### Groups
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/groups` | Bearer | List groups |
| POST | `/api/v1/groups` | Bearer | Create group |
| GET | `/api/v1/groups/{id}` | Bearer | Get group |
| PUT | `/api/v1/groups/{id}` | Bearer | Update group |
| DELETE | `/api/v1/groups/{id}` | Bearer | Delete group |
| GET | `/api/v1/groups/{id}/members` | Bearer | List members |
| POST | `/api/v1/groups/{id}/members` | Bearer | Add member |

### Organization
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/orgs` | Bearer | List organizations |
| POST | `/api/v1/orgs` | Bearer | Create organization |
| GET | `/api/v1/orgs/{id}` | Bearer | Get organization |
| PUT | `/api/v1/orgs/{id}` | Bearer | Update organization |
| DELETE | `/api/v1/orgs/{id}` | Bearer | Delete organization |
| GET | `/api/v1/orgs/tree` | Bearer | Full org tree |
| GET | `/api/v1/departments` | Bearer | List departments |
| POST | `/api/v1/departments` | Bearer | Create department |
| GET | `/api/v1/teams` | Bearer | List teams |
| POST | `/api/v1/teams` | Bearer | Create team |

### NHI (Non-Human Identity)
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/identity/nhi` | Bearer | List NHI inventory |
| GET | `/api/v1/identity/nhi/{id}/risk` | Bearer | Get NHI risk score |
| GET | `/api/v1/identity/nhi/risk-alerts` | Bearer | High-risk NHI list |
| POST | `/api/v1/identity/nhi/risk/scan` | Bearer | Trigger risk evaluation |

### Privileged Operations
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/identity/privileged-operations` | Bearer | Privileged op audit trail |

---

## OAuth Service

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/oauth/authorize` | Session | OAuth authorize endpoint |
| POST | `/api/v1/oauth/token` | Client | Issue token (all grants) |
| POST | `/api/v1/oauth/introspect` | Bearer | Token introspection |
| POST | `/api/v1/oauth/revoke` | Bearer | Revoke token |
| GET | `/api/v1/oauth/clients` | Bearer | List clients |
| POST | `/api/v1/oauth/clients` | Bearer | Create client |
| GET | `/api/v1/oauth/clients/{id}` | Bearer | Get client |
| PUT | `/api/v1/oauth/clients/{id}` | Bearer | Update client |
| DELETE | `/api/v1/oauth/clients/{id}` | Bearer | Delete client |
| GET | `/.well-known/openid-configuration` | — | OIDC discovery |
| GET | `/.well-known/jwks.json` | — | JWKS public keys |

---

## Policy Service

### Roles
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/roles` | Bearer | List roles |
| POST | `/api/v1/roles` | Bearer | Create role |
| GET | `/api/v1/roles/{id}` | Bearer | Get role |
| PUT | `/api/v1/roles/{id}` | Bearer | Update role |
| DELETE | `/api/v1/roles/{id}` | Bearer | Delete role |
| POST | `/api/v1/roles/assign` | Bearer | Assign role to user |

### Permissions
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/permissions` | Bearer | List permissions |
| POST | `/api/v1/permissions` | Bearer | Create permission |

### Policies (ABAC/RBAC)
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/policies` | Bearer | List policies |
| POST | `/api/v1/policies` | Bearer | Create policy |
| GET | `/api/v1/policies/{id}` | Bearer | Get policy |
| PUT | `/api/v1/policies/{id}` | Bearer | Update policy |
| DELETE | `/api/v1/policies/{id}` | Bearer | Delete policy |
| POST | `/api/v1/policies/check` | Bearer | Check single permission |
| POST | `/api/v1/policies/evaluate` | Bearer | Evaluate with decision trail |

### SoD (Separation of Duties)
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/v1/policies/sod/check` | Bearer | Check SoD violation |
| GET | `/api/v1/policies/sod/violations` | Bearer | List violations |
| GET | `/api/v1/policies/sod/matrix` | Bearer | SoD conflict matrix |

---

## Audit Service

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/audit/events` | Bearer | List audit events |
| GET | `/api/v1/audit/events/{id}` | Bearer | Get event by ID |
| GET | `/api/v1/audit/stats` | Bearer | Aggregate statistics |
| GET | `/api/v1/audit/export` | Bearer | Export events (JSON/CSV) |
| GET | `/api/v1/audit/stream` | Bearer | SSE event stream |

### CCM (Continuous Compliance Monitoring)
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/audit/ccm/results` | Bearer | Latest control results |
| GET | `/api/v1/audit/ccm/history` | Bearer | Historical trends |
| GET | `/api/v1/audit/ccm/summary` | Bearer | Compliance dashboard |
| POST | `/api/v1/audit/ccm/run` | Bearer | Trigger compliance scan |

---

## Admin & System

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/admin/backups` | Admin | List backups |
| POST | `/api/v1/admin/backups/trigger` | Admin | Trigger backup |
| GET | `/api/v1/admin/secrets` | Admin | List secret references |
| GET | `/api/v1/admin/keys` | Admin | List signing keys |
| GET | `/api/v1/quotas/{tenant_id}` | Admin | Get tenant quota |
| PUT | `/api/v1/quotas/{tenant_id}` | Admin | Update quota |
| GET | `/healthz` | — | Health check |
| GET | `/readyz` | — | Readiness check |
| POST | `/graphql` | Bearer | GraphQL endpoint |
