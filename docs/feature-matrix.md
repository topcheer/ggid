# GGID IAM Platform — Feature Comparison Matrix

> Comprehensive feature benchmark of GGID against 10 leading IAM platforms.
> Last updated: 2025

## Target Platforms Compared

| # | Platform | Type | License |
|---|----------|------|---------|
| 1 | **Auth0 (Okta)** | Cloud SaaS | Commercial |
| 2 | **Keycloak** | Self-hosted | Apache 2.0 |
| 3 | **AWS Cognito** | Cloud SaaS | Commercial |
| 4 | **Casdoor** | Self-hosted | Apache 2.0 |
| 5 | **Authing** | Cloud SaaS | Commercial |
| 6 | **WorkOS** | Cloud SaaS | Commercial |
| 7 | **Clerk** | Cloud SaaS | Commercial |
| 8 | **Logto** | Self-hosted / Cloud | MIT / Commercial |
| 9 | **SuperTokens** | Self-hosted / Cloud | Apache 2.0 |
| 10 | **Ory** | Self-hosted / Cloud | Apache 2.0 |

---

## 1. Authentication (身份认证)

| Feature | GGID | Auth0 | Keycloak | Cognito | Casdoor | Authing | WorkOS | Clerk | Logto | SuperTokens | Ory | Priority | Best Practice Reference |
|---------|------|-------|----------|---------|---------|---------|--------|-------|-------|-------------|-----|----------|------------------------|
| Username/Password Login | ✅ 已实现 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P0 | All platforms |
| User Registration | ✅ 已实现 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P0 | All platforms |
| JWT Access + Refresh Tokens | ✅ 已实现 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P0 | Auth0 JWT pattern |
| Token Revocation | ⚠️ 部分实现 (logout) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P0 | RFC 7009 |
| Token Introspection | ⚠️ 骨架 (stub) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P1 | RFC 7662 |
| MFA — TOTP (Google Authenticator) | ✅ 已实现 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P0 | RFC 6238 |
| MFA — SMS/OTP | ❌ 未实现 | ✅ | ✅ (plugin) | ✅ | ✅ | ✅ | ❌ | ❌ | ✅ | ✅ | ❌ | P1 | Auth0 Guardian |
| MFA — Email OTP | ❌ 未实现 | ✅ | ✅ | ❌ | ✅ | ✅ | ❌ | ✅ | ✅ | ✅ | ✅ | P1 | Logto/Ory pattern |
| MFA — Push Notification | ❌ 未实现 | ✅ (Guardian) | ❌ | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | P2 | Auth0 Guardian, Duo |
| MFA — WebAuthn/Passkey | ⚠️ 骨架 | ✅ | ✅ (FIDO2) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P0 | FIDO2/WebAuthn, Clerk UX |
| MFA — Adaptive/Step-up | ❌ 未实现 | ✅ | ✅ | ❌ | ❌ | ✅ | ❌ | ❌ | ✅ | ❌ | ✅ | P1 | Auth0 Actions, Ory risk |
| OAuth 2.0 Authorization Code | ✅ 已实现 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P0 | RFC 6749 |
| OAuth 2.0 PKCE | ✅ 已实现 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P0 | RFC 7636 |
| OAuth Client Credentials (M2M) | ⚠️ 骨架 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ | ✅ | ✅ | ✅ | P1 | WorkOS M2M pattern |
| OIDC Discovery + JWKS | ✅ 已实现 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P0 | RFC 8414 |
| OIDC ID Token | ✅ 已实现 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P0 | OIDC Core 1.0 |
| OIDC UserInfo Endpoint | ⚠️ 骨架 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P0 | OIDC Core 1.0 |
| SAML 2.0 IdP | ⚠️ 骨架 (metadata) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ | P1 | WorkOS/Keycloak SAML |
| SAML 2.0 SP (consume assertions) | ❌ 未实现 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ | ✅ | ❌ | ✅ | P1 | Keycloak SP-initiated SSO |
| Social Login (Google/GitHub/etc.) | ❌ 未实现 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P0 | Logto/Clerk connectors |
| Social Login (WeChat/Alipay/DingTalk) | ❌ 未实现 | ⚠️ (limited) | ❌ | ❌ | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | P1 | Casdoor/Authing (China) |
| Passwordless — Magic Link (Email) | ❌ 未实现 | ✅ | ❌ | ✅ | ✅ | ✅ | ❌ | ✅ | ✅ | ✅ | ✅ | P1 | Clerk/Logto magic links |
| Passwordless — SMS OTP | ❌ 未实现 | ✅ | ❌ | ✅ | ✅ | ✅ | ❌ | ✅ | ✅ | ✅ | ✅ | P2 | Clerk phone OTP |
| Multi-factor enforcement policy | ❌ 未实现 | ✅ | ✅ | ⚠️ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P1 | Auth0 MFA policies |
| Session-based Auth (cookie) | ⚠️ 部分实现 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P1 | Clerk/Logto sessions |
| Remember Me / Long-lived sessions | ❌ 未实现 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P2 | Clerk long sessions |
| Single Sign-On (SSO) across apps | ⚠️ 部分 (OIDC) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P0 | Keycloak/WorkOS SSO |
| Single Logout (SLO) | ⚠️ 骨架 | ✅ | ✅ | ❌ | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ | ✅ | P2 | Keycloak SLO |

**Summary: 6/30 fully implemented, 7 skeleton/partial, 17 not implemented**

---

## 2. User Management (用户管理)

| Feature | GGID | Auth0 | Keycloak | Cognito | Casdoor | Authing | WorkOS | Clerk | Logto | SuperTokens | Ory | Priority | Best Practice Reference |
|---------|------|-------|----------|---------|---------|---------|--------|-------|-------|-------------|-----|----------|------------------------|
| User CRUD | ✅ 已实现 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P0 | All platforms |
| User List with Pagination/Filter | ✅ 已实现 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P0 | All platforms |
| User Lock/Unlock/Disable | ✅ 已实现 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P0 | Auth0 Management API |
| Email Verification | ✅ 已实现 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P0 | OIDC email_verified claim |
| Multi-Email per User | ✅ 已实现 | ✅ | ✅ | ❌ | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ | ✅ | P1 | Authing/Casdoor pattern |
| Phone Number Management | ❌ 未实现 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P1 | Cognito/Keycloak |
| Custom User Attributes | ⚠️ 部分 (metadata) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ (UDA) | ✅ | ✅ | ✅ | P0 | Auth0 app_metadata |
| User Groups | ❌ 未实现 | ✅ | ✅ | ⚠️ | ✅ | ✅ | ✅ | ✅ | ❌ | ✅ | ✅ | P1 | Keycloak groups |
| User Roles (local to org) | ✅ 已实现 | ✅ | ✅ | ⚠️ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P0 | All platforms |
| External Identity Linking | ✅ 已实现 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P0 | OIDC identity linking |
| JIT Auto-Provisioning | ✅ 已实现 (LDAP) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P0 | Keycloak/Ory JIT |
| User Import (CSV/JSON/Bulk) | ❌ 未实现 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ | ✅ | P1 | Auth0 import users API |
| User Export / Data Portability | ❌ 未实现 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ | ✅ | P1 | GDPR compliance |
| User Deletion (GDPR) | ✅ 已实现 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P1 | GDPR right to erasure |
| User Deactivation (soft delete) | ⚠️ 部分 (disable) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P1 | Auth0/Ory patterns |
| Profile Management (self-service) | ⚠️ 部分 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P1 | Clerk user profile UX |
| Account Linking (merge accounts) | ⚠️ 部分 | ✅ | ✅ | ❌ | ✅ | ✅ | ✅ | ✅ | ✅ | ⚠️ | ✅ | P2 | Auth0 account linking |
| User Search (full-text) | ❌ 未实现 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ | ✅ | P1 | ElasticSearch integration |
| Webhooks on User Events | ❌ 未实现 | ✅ | ✅ | ⚠️ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P1 | Auth0 Log Streams |
| Password Reset Flow | ✅ 已实现 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P0 | All platforms |
| Email Change Verification | ❌ 未实现 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ | ✅ | P1 | Auth0 change email flow |
| Account Enumeration Protection | ✅ 已实现 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P1 | Always return success on forgot |

**Summary: 10/24 fully implemented, 5 partial, 9 not implemented**

---

## 3. Authorization (权限控制)

| Feature | GGID | Auth0 | Keycloak | Cognito | Casdoor | Authing | WorkOS | Clerk | Logto | SuperTokens | Ory | Priority | Best Practice Reference |
|---------|------|-------|----------|---------|---------|---------|--------|-------|-------|-------------|-----|----------|------------------------|
| RBAC — Role Management | ✅ 已实现 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P0 | All platforms |
| RBAC — Permission Management | ✅ 已实现 | ✅ | ✅ | ⚠️ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P0 | Keycloak/Ory permission model |
| RBAC — Role Hierarchy/Inheritance | ✅ 已实现 | ❌ | ✅ | ❌ | ✅ | ✅ | ✅ | ❌ | ❌ | ⚠️ | ✅ | P1 | Keycloak composite roles |
| ABAC — Attribute-Based Policies | ✅ 已实现 | ✅ | ✅ | ⚠️ | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ | ✅ | P1 | Ory Keto / AWS IAM |
| Policy Engine (allow/deny rules) | ✅ 已实现 | ✅ (Actions) | ✅ | ❌ | ✅ (Casbin) | ✅ | ❌ | ❌ | ❌ | ❌ | ✅ | P0 | Casbin / Ory AccessControlPolicies |
| Policy Deny Override | ✅ 已实现 | ✅ | ✅ | ❌ | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ | ✅ | P1 | AWS IAM explicit deny |
| Wildcard/Pattern Resource Matching | ✅ 已实现 | ❌ | ❌ | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ | ✅ | P1 | AWS ARN wildcards |
| Scoped Role Assignment | ✅ 已实现 | ⚠️ | ✅ | ⚠️ | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ | ✅ | P1 | Keycloak role scope |
| Temporary Role Assignment (TTL) | ✅ 已实现 | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | P2 | GGID unique feature |
| Fine-Grained Permission Check API | ✅ 已实现 | ✅ | ✅ | ⚠️ | ✅ | ✅ | ❌ | ⚠️ | ❌ | ✅ | ✅ | P0 | Ory PermissionCheck |
| ReBAC (Relationship-Based) | ❌ 未实现 | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ (Keto) | P2 | Google Zanzibar / Ory Keto |
| Resource-Level Permissions | ⚠️ 部分 | ✅ | ✅ | ❌ | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ | ✅ | P1 | Ory relation tuples |
| Policy Versioning | ❌ 未实现 | ✅ (Actions) | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | P2 | Auth0 Actions deploy |
| Authorization SDK / Middleware | ✅ 已实现 | ✅ | ✅ | ⚠️ | ✅ | ✅ | ⚠️ | ✅ | ❌ | ✅ | ✅ | P0 | All platforms with SDK |
| Context-Aware Auth (IP/Time/Device) | ❌ 未实现 | ✅ | ✅ | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ | ❌ | ✅ | P1 | Auth0 Rules/Actions |

**Summary: 9/15 fully implemented, 3 partial, 3 not implemented**

---

## 4. Organization (组织架构)

| Feature | GGID | Auth0 | Keycloak | Cognito | Casdoor | Authing | WorkOS | Clerk | Logto | SuperTokens | Ory | Priority | Best Practice Reference |
|---------|------|-------|----------|---------|---------|---------|--------|-------|-------|-------------|-----|----------|------------------------|
| Multi-Tenant Architecture | ✅ 已实现 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ⚠️ | ✅ | P0 | All enterprise platforms |
| Tenant Isolation (RLS) | ✅ 已实现 | ✅ (managed) | ✅ | ✅ | ⚠️ | ✅ | ✅ | ✅ | ⚠️ | ❌ | ⚠️ | P0 | PostgreSQL RLS |
| Tenant CRUD | ✅ 已实现 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ | ✅ | P0 | WorkOS Organizations |
| Organization Tree (hierarchy) | ✅ 已实现 (LTREE) | ❌ | ✅ | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | P1 | Keycloak/Authing org tree |
| Departments | ✅ 已实现 | ❌ | ❌ | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | P1 | Authing departments |
| Teams | ✅ 已实现 | ❌ | ✅ | ❌ | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | P1 | Keycloak teams |
| Membership Management | ✅ 已实现 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ | ✅ | P0 | All platforms |
| Invitation System | ✅ 已实现 | ✅ | ❌ | ❌ | ❌ | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ | P1 | WorkOS/Clerk invitations |
| Per-Tenant SSO Configuration | ❌ 未实现 | ✅ | ✅ | ❌ | ❌ | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ | P0 | WorkOS/Logto per-org SSO |
| Per-Tenant Branding | ❌ 未实现 | ✅ | ✅ | ⚠️ | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ | P2 | Authing/Casdoor branding |
| Per-Tenant User Pools | ⚠️ 部分 (RLS) | ✅ | ✅ | ✅ | ⚠️ | ✅ | ✅ | ✅ | ✅ | ❌ | ⚠️ | P1 | Cognito user pools |
| Cross-Tenant User Federation | ❌ 未实现 | ✅ | ✅ | ❌ | ❌ | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ | P2 | Authing federation |

**Summary: 7/12 fully implemented, 2 partial, 3 not implemented**

---

## 5. Audit & Compliance (审计合规)

| Feature | GGID | Auth0 | Keycloak | Cognito | Casdoor | Authing | WorkOS | Clerk | Logto | SuperTokens | Ory | Priority | Best Practice Reference |
|---------|------|-------|----------|---------|---------|---------|--------|-------|-------|-------------|-----|----------|------------------------|
| Audit Log (Event Recording) | ✅ 已实现 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ⚠️ | ✅ | P0 | All platforms |
| Real-time Event Streaming | ✅ 已实现 (NATS) | ✅ (Streams) | ✅ | ⚠️ | ❌ | ✅ | ✅ | ❌ | ❌ | ❌ | ✅ | P1 | NATS/Kafka streaming |
| Audit Query API | ✅ 已实现 | ✅ | ✅ | ⚠️ | ✅ | ✅ | ✅ | ⚠️ | ✅ | ⚠️ | ✅ | P0 | Auth0 Log API |
| Event Filtering (time/actor/type) | ✅ 已实现 | ✅ | ✅ | ⚠️ | ✅ | ✅ | ✅ | ⚠️ | ✅ | ⚠️ | ✅ | P0 | Standard query filters |
| Compliance Reports (SOC2/HIPAA) | ❌ 未实现 | ✅ | ❌ | ✅ | ❌ | ✅ | ✅ | ⚠️ | ❌ | ❌ | ⚠️ | P1 | WorkOS/Keycloak reports |
| Log Retention Policies | ❌ 未实现 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ⚠️ | ✅ | P1 | Configurable TTL |
| SIEM Integration (Splunk/Datadog) | ❌ 未实现 | ✅ | ✅ | ✅ | ❌ | ✅ | ✅ | ❌ | ❌ | ❌ | ✅ | P1 | Auth0 Log Streams |
| Webhook Notifications | ❌ 未实现 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P1 | All platforms |
| GDPR Data Export | ❌ 未实现 | ✅ | ⚠️ | ✅ | ❌ | ✅ | ✅ | ✅ | ❌ | ❌ | ✅ | P1 | GDPR Article 20 |
| GDPR Data Erasure | ⚠️ 部分 (delete) | ✅ | ⚠️ | ✅ | ❌ | ✅ | ✅ | ✅ | ❌ | ❌ | ✅ | P1 | GDPR Article 17 |
| Immutable Audit Trail | ❌ 未实现 | ✅ | ⚠️ | ✅ | ❌ | ✅ | ✅ | ❌ | ❌ | ❌ | ⚠️ | P2 | Hash-chain / WORM storage |
| Admin Activity Logging | ⚠️ 部分 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ⚠️ | ✅ | P0 | SOC2 requirement |
| Login History per User | ❌ 未实现 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P1 | Authing/Clerk login logs |

**Summary: 4/13 fully implemented, 3 partial, 6 not implemented**

---

## 6. Developer Experience (开发者体验)

| Feature | GGID | Auth0 | Keycloak | Cognito | Casdoor | Authing | WorkOS | Clerk | Logto | SuperTokens | Ory | Priority | Best Practice Reference |
|---------|------|-------|----------|---------|---------|---------|--------|-------|-------|-------------|-----|----------|------------------------|
| REST API | ✅ 已实现 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P0 | All platforms |
| gRPC API | ✅ 已实现 | ❌ | ⚠️ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ | P1 | Ory gRPC, GGID unique |
| SDK — Go | ✅ 已实现 | ✅ | ✅ | ✅ (AWS) | ✅ | ❌ | ✅ | ❌ | ✅ | ✅ | ✅ | P0 | Official SDK per language |
| SDK — Node.js / TypeScript | ✅ 已实现 | ✅ | ❌ | ✅ (AWS) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P0 | Clerk/Auth0 SDK |
| SDK — Java | ✅ 已实现 | ✅ | ✅ | ✅ (AWS) | ⚠️ | ✅ | ❌ | ❌ | ❌ | ⚠️ | ⚠️ | P1 | Auth0/Keycloak Java SDK |
| SDK — Python | ❌ 未实现 | ✅ | ✅ | ✅ (AWS) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P1 | Auth0/Ory Python |
| SDK — Ruby | ❌ 未实现 | ✅ | ❌ | ⚠️ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | P2 | Auth0 Ruby SDK |
| SDK — Mobile (iOS/Android) | ❌ 未实现 | ✅ | ✅ | ✅ | ✅ | ✅ | ⚠️ | ✅ | ✅ | ✅ | ⚠️ | P1 | Auth0/Clerk mobile SDK |
| Admin Console (Web UI) | ✅ 已实现 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ | ✅ | P0 | All platforms |
| API Gateway | ✅ 已实现 | ⚠️ (Actions) | ❌ | ⚠️ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | P1 | GGID unique |
| OIDC Discovery Endpoint | ✅ 已实现 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P0 | RFC 8414 |
| JWKS Endpoint | ✅ 已实现 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P0 | RFC 7517 |
| OpenAPI / Swagger Docs | ❌ 未实现 | ✅ | ⚠️ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P0 | OpenAPI 3.0 spec |
| GraphQL API | ❌ 未实现 | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ | P2 | Clerk GraphQL |
| Docker / Container Deployment | ✅ 已实现 | ❌ | ✅ | ❌ | ✅ | ❌ | ❌ | ❌ | ✅ | ✅ | ✅ | P0 | Docker Compose / K8s |
| Kubernetes Ready | ⚠️ 部分 (compose) | N/A | ✅ | N/A | ✅ | N/A | N/A | N/A | ✅ | ✅ | ✅ | P1 | Helm charts |
| Webhooks / Event Subscriptions | ❌ 未实现 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P1 | All platforms |
| Quick Start / Sample Apps | ❌ 未实现 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P1 | All platforms |
| CI/CD Integration Templates | ❌ 未实现 | ✅ | ⚠️ | ⚠️ | ❌ | ❌ | ❌ | ✅ | ❌ | ❌ | ⚠️ | P2 | GitHub Actions templates |
| Pre-built Login UI | ❌ 未实现 | ✅ (Universal) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ⚠️ | ✅ | P0 | Auth0 Universal Login, Clerk |
| Customizable Login Page | ⚠️ 部分 (console) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P0 | Auth0 Custom Domain |
| Localization / i18n | ❌ 未实现 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P1 | Authing (Chinese+English) |

**Summary: 9/22 fully implemented, 2 partial, 11 not implemented**

---

## 7. Security (安全特性)

| Feature | GGID | Auth0 | Keycloak | Cognito | Casdoor | Authing | WorkOS | Clerk | Logto | SuperTokens | Ory | Priority | Best Practice Reference |
|---------|------|-------|----------|---------|---------|---------|--------|-------|-------|-------------|-----|----------|------------------------|
| Argon2id Password Hashing | ✅ 已实现 | ✅ (bcrypt opt) | ✅ | ⚠️ (PBKDF2) | ✅ | ✅ | N/A | ✅ | ✅ | ✅ | ✅ | P0 | OWASP recommendation |
| Password Policy (complexity) | ✅ 已实现 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P0 | NIST 800-63B |
| Password History (reuse prevention) | ✅ 已实现 | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ | ⚠️ | ✅ | P1 | Common enterprise req |
| Breached Password Detection | ❌ 未实现 | ✅ | ⚠️ | ❌ | ❌ | ✅ | ❌ | ✅ | ❌ | ❌ | ❌ | P1 | HIBP API integration |
| Session Management | ✅ 已实现 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P0 | All platforms |
| Session Revocation | ✅ 已实现 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P0 | All platforms |
| Device/Session Listing | ✅ 已实现 | ✅ | ✅ | ❌ | ❌ | ✅ | ❌ | ✅ | ✅ | ⚠️ | ✅ | P1 | Clerk session dashboard |
| Rate Limiting | ✅ 已实现 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P0 | Brute-force protection |
| Account Lockout | ✅ 已实现 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P0 | OWASP recommendation |
| JWT with RSA Signing | ✅ 已实现 | ✅ | ✅ | ⚠️ (HMAC) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P0 | RS256 asymmetric |
| Key Rotation | ❌ 未实现 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P1 | Automatic key rotation |
| PKCE Enforcement | ✅ 已实现 | ✅ | ✅ | ⚠️ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P0 | RFC 7636 |
| Anomaly Detection (impossible travel) | ❌ 未实现 | ✅ | ❌ | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ | ❌ | ✅ | P1 | Auth0 Attack Protection |
| Bot Detection / CAPTCHA | ❌ 未实现 | ✅ | ❌ | ❌ | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | P2 | reCAPTCHA/Turnstile |
| IP Allowlisting / Blocklisting | ❌ 未实现 | ✅ | ✅ | ⚠️ | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ | ✅ | P1 | Enterprise security |
| CORS Configuration | ❌ 未实现 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P0 | Browser security |
| CSP / Security Headers | ❌ 未实现 | ✅ | ⚠️ | ✅ | ⚠️ | ✅ | ✅ | ✅ | ⚠️ | ⚠️ | ✅ | P1 | OWASP headers |
| Encryption at Rest | ✅ 已实现 (AES-256) | ✅ | ✅ | ✅ | ⚠️ | ✅ | ✅ | ✅ | ⚠️ | ⚠️ | ✅ | P0 | All enterprise platforms |
| Encryption in Transit (TLS) | ⚠️ 部分 | ✅ | ✅ | ✅ | ⚠️ | ✅ | ✅ | ✅ | ✅ | ⚠️ | ✅ | P0 | TLS 1.3 |
| Cookie Security (HttpOnly/Secure/SameSite) | ❌ 未实现 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P0 | OWASP cookie flags |
| CSRF Protection | ❌ 未实现 | ✅ | ✅ | ✅ | ⚠️ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P0 | Double-submit cookie |
| Passwordless Security Level | N/A | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | P1 | WebAuthn Level 2 |
| Audit All Security Events | ⚠️ 部分 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ⚠️ | ✅ | P1 | SOC2 compliance |

**Summary: 10/23 fully implemented, 4 partial, 9 not implemented**

---

## 8. Enterprise Features (企业特性)

| Feature | GGID | Auth0 | Keycloak | Cognito | Casdoor | Authing | WorkOS | Clerk | Logto | SuperTokens | Ory | Priority | Best Practice Reference |
|---------|------|-------|----------|---------|---------|---------|--------|-------|-------|-------------|-----|----------|------------------------|
| LDAP / Active Directory | ✅ 已实现 | ✅ | ✅ | ❌ | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ | P0 | Keycloak LDAP federation |
| LDAP Auto-Provisioning | ✅ 已实现 | ✅ | ✅ | ❌ | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ | P1 | JIT provisioning |
| SCIM 2.0 (Inbound) | ✅ 已实现 | ✅ | ⚠️ (plugin) | ⚠️ | ❌ | ✅ | ✅ | ⚠️ | ❌ | ❌ | ❌ | P1 | RFC 7643/7644 |
| SCIM 2.0 (Outbound) | ❌ 未实现 | ❌ | ⚠️ (plugin) | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | P2 | WorkOS directory sync |
| SAML 2.0 IdP (Provider) | ⚠️ 骨架 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ | P1 | WorkOS/Keycloak SAML |
| SAML 2.0 SP (Consumer) | ❌ 未实现 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ | ✅ | ❌ | ✅ | P0 | Enterprise SSO requirement |
| Directory Sync | ❌ 未实现 | ✅ | ⚠️ | ❌ | ❌ | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ | P1 | WorkOS/Okta directory sync |
| Enterprise SSO (pre-configured) | ❌ 未实现 | ✅ | ✅ | ⚠️ | ❌ | ✅ | ✅ | ❌ | ✅ | ❌ | ⚠️ | P0 | WorkOS 50+ SSO connectors |
| M2M (Machine-to-Machine) Auth | ⚠️ 骨架 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ | ⚠️ | ⚠️ | ✅ | P1 | Auth0 client credentials |
| Custom Domain | ❌ 未实现 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ | ✅ | P1 | Custom branding |
| White-Label / Embedded | ❌ 未实现 | ⚠️ | ✅ | ⚠️ | ✅ | ✅ | ❌ | ✅ | ✅ | ✅ | ✅ | P2 | Clerk embedded auth |
| API Rate Limit Tiers | ❌ 未实现 | ✅ | ✅ | ✅ | ❌ | ✅ | ✅ | ✅ | ✅ | ⚠️ | ✅ | P2 | Enterprise tier limits |
| SLA / High Availability | ❌ 未实现 | ✅ | ⚠️ | ✅ | ❌ | ✅ | ✅ | ✅ | ⚠️ | ⚠️ | ⚠️ | P1 | 99.99% uptime SLA |
| Compliance Certifications | ❌ 未实现 | ✅ (SOC2/ISO) | ⚠️ | ✅ | ❌ | ✅ | ✅ | ⚠️ | ❌ | ❌ | ⚠️ | P2 | SOC2/HIPAA/ISO27001 |
| Data Residency | ❌ 未实现 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ⚠️ | ⚠️ | ⚠️ | ⚠️ | P2 | EU/US data centers |
| User Impersonation | ❌ 未实现 | ✅ | ✅ | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | P2 | Admin impersonation |
| Delegated Administration | ❌ 未实现 | ✅ | ✅ | ⚠️ | ❌ | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ | P1 | WorkOS/Clerk org admin |
| Provisioning Webhooks | ❌ 未实现 | ✅ | ✅ | ⚠️ | ❌ | ✅ | ✅ | ⚠️ | ⚠️ | ❌ | ✅ | P1 | SCIM + webhook hybrid |

**Summary: 3/18 fully implemented, 5 skeleton/partial, 10 not implemented**

---

## Overall Feature Coverage Scorecard

| Category | Total Features | ✅ Implemented | ⚠️ Partial/Skeleton | ❌ Not Implemented | Coverage |
|----------|---------------|----------------|---------------------|--------------------|---------|
| Authentication | 30 | 6 | 7 | 17 | 20% |
| User Management | 24 | 10 | 5 | 9 | 42% |
| Authorization | 15 | 9 | 3 | 3 | 60% |
| Organization | 12 | 7 | 2 | 3 | 58% |
| Audit & Compliance | 13 | 4 | 3 | 6 | 31% |
| Developer Experience | 22 | 9 | 2 | 11 | 41% |
| Security | 23 | 10 | 4 | 9 | 43% |
| Enterprise | 18 | 3 | 5 | 10 | 17% |
| **TOTAL** | **157** | **58** | **31** | **68** | **37%** |

---

## Priority Roadmap (P0 Critical Gaps)

These are P0 features that GGID currently does NOT implement and are table stakes for any competitive IAM platform:

### P0 — Must Have (Blocks GA Release)

| # | Feature | Category | Effort | Reference |
|---|---------|----------|--------|-----------|
| 1 | **Social Login Connectors** (Google, GitHub, Microsoft, Apple) | Auth | Medium | Logto connectors, Clerk social |
| 2 | **WebAuthn/Passkey Full Implementation** (currently skeleton) | Auth | Medium | go-webauthn library, Clerk passkeys |
| 3 | **OIDC UserInfo Endpoint** (return real user data) | Auth | Small | OIDC Core spec |
| 4 | **SAML SP** (consume SAML assertions from enterprise IdPs) | Enterprise | Large | Keycloak SP module |
| 5 | **Enterprise SSO Connectors** (pre-built for Okta/Azure AD/Google) | Enterprise | Large | WorkOS SSO templates |
| 6 | **Per-Tenant SSO Configuration** | Org | Medium | WorkOS/Logto per-org SSO |
| 7 | **Pre-built Login UI** (hosted auth page) | DevEx | Medium | Auth0 Universal Login |
| 8 | **OpenAPI / Swagger Documentation** | DevEx | Small | OpenAPI 3.0 codegen |
| 9 | **CORS Configuration** | Security | Small | HTTP middleware |
| 10 | **Cookie Security (HttpOnly/Secure/SameSite)** | Security | Small | HTTP middleware |
| 11 | **CSRF Protection** | Security | Medium | Double-submit pattern |
| 12 | **SDK — Python** | DevEx | Medium | Auto-generate from OpenAPI |

### P1 — Important (Next Quarter)

| # | Feature | Category | Effort | Reference |
|---|---------|----------|--------|-----------|
| 1 | Magic Link / Passwordless Email | Auth | Medium | Logto/Ory magic links |
| 2 | SMS/Email OTP MFA | Auth | Medium | Twilio integration |
| 3 | Adaptive/Step-up MFA | Auth | Large | Auth0 Actions |
| 4 | SAML IdP Full Implementation | Enterprise | Large | XML signing/validation |
| 5 | Directory Sync (SCIM outbound) | Enterprise | Large | WorkOS directory sync |
| 6 | User Groups | User Mgmt | Medium | Keycloak group model |
| 7 | User Import/Export | User Mgmt | Medium | CSV/JSON bulk API |
| 8 | User Search (full-text) | User Mgmt | Medium | PostgreSQL FTS |
| 9 | Webhooks / Event Subscriptions | DevEx | Medium | NATS consumer |
| 10 | Key Rotation | Security | Medium | JWKS rotation |
| 11 | Anomaly Detection | Security | Large | Impossible travel detection |
| 12 | Breached Password Detection | Security | Small | HIBP API |
| 13 | Compliance Reports | Audit | Medium | SOC2 templates |
| 14 | SIEM Integration | Audit | Medium | Log streaming |
| 15 | Delegated Administration | Enterprise | Medium | Org-scoped admin roles |
| 16 | SDK — Mobile (iOS/Android) | DevEx | Large | Native SDK |
| 17 | Kubernetes Helm Charts | DevEx | Medium | Helm + K8s manifests |
| 18 | Localization / i18n | DevEx | Medium | i18n framework |
| 19 | Quick Start / Sample Apps | DevEx | Small | Next.js/Express examples |

### P2 — Nice to Have (Future)

| # | Feature | Category | Reference |
|---|---------|----------|-----------|
| 1 | Push Notification MFA | Auth | Auth0 Guardian |
| 2 | ReBAC (Zanzibar model) | Authz | Ory Keto |
| 3 | GraphQL API | DevEx | Clerk |
| 4 | Bot Detection / CAPTCHA | Security | reCAPTCHA |
| 5 | User Impersonation | Enterprise | Keycloak |
| 6 | Data Residency | Enterprise | Multi-region |
| 7 | White-Label / Embedded Auth | Enterprise | Clerk embedded |
| 8 | Compliance Certifications (SOC2) | Enterprise | External audit |
| 9 | SDK — Ruby | DevEx | Auth0 Ruby |
| 10 | Immutable Audit Trail | Audit | Hash-chain |
| 11 | Remember Me / Long-lived Sessions | Auth | Clerk |
| 12 | Single Logout (SLO) across apps | Auth | Keycloak SLO |

---

## Competitive Positioning Analysis

### GGID's Unique Strengths (Competitive Advantages)

| Strength | Detail | Comparable Platform |
|----------|--------|---------------------|
| **gRPC First-Class API** | Full gRPC + REST dual protocol — rare in IAM | Ory (partial), Keycloak (partial) |
| **RBAC + ABAC Policy Engine** | Combined engine with deny-override + wildcards | Casdoor (Casbin), Ory |
| **Organization Tree (LTREE)** | PostgreSQL LTREE for hierarchical org modeling | Authing, Keycloak |
| **Temporary Role Assignment (TTL)** | Auto-expiring role grants — no competitor has this | None (unique) |
| **Multi-Service Architecture** | 7 independently deployable microservices | Keycloak (monolith), Auth0 (SaaS) |
| **Open Source Apache 2.0** | Fully open source, self-hostable | Keycloak, Casdoor, Ory, Logto |
| **Multi-Tenant by Design** | RLS-based tenant isolation from day one | Authing, Cognito |
| **Argon2id Default** | OWASP-recommended hashing (many still use bcrypt/PBKDF2) | Logto, Ory |

### GGID's Critical Gaps vs Market

| Gap | Impact | Market Standard |
|-----|--------|-----------------|
| No Social Login | Blocks CIAM/B2C use cases | ALL competitors have this |
| No Pre-built Login UI | Developers must build everything | Auth0/Clerk/Casdoor |
| No Enterprise SSO Connectors | Blocks B2B enterprise deals | WorkOS/Authing |
| No Passwordless | Missing modern auth expectations | Clerk/Logto/Ory |
| No SDK for Python | Limits developer adoption | All major platforms |
| No Webhooks | Blocks event-driven integrations | All platforms |
| No Breached Password Check | Security gap for enterprise | Auth0/Clerk |
| No OpenAPI Spec | API discoverability gap | All platforms |

---

## Platform-by-Platform Best Practice Takeaways

### 1. Auth0 (Okta) — Gold Standard for DX
- **Copy:** Universal Login (hosted auth page), Actions extensibility, comprehensive SDKs, clear docs
- **Copy:** Breached password detection, anomaly detection, adaptive MFA
- **Copy:** Pre-built social connectors (30+ providers)

### 2. Keycloak — Gold Standard for Enterprise
- **Copy:** Full SAML IdP + SP, LDAP federation with mapper, comprehensive group/role model
- **Copy:** Realm-per-tenant isolation pattern
- **Avoid:** Monolithic deployment (GGID's microservice approach is better)

### 3. AWS Cognito — Gold Standard for Scale
- **Copy:** User pool concept (separate pools per app/customer)
- **Avoid:** Poor developer experience, complex setup

### 4. Casdoor — Gold Standard for China Market
- **Copy:** WeChat/DingTalk/Alipay social login, Casbin policy engine
- **Copy:** Plugin extensibility model
- **Copy:** AI-native identity (emerging trend)

### 5. Authing — Gold Standard for CIAM
- **Copy:** Rich social login (30+ Chinese + international providers), visual org tree
- **Copy:** Delegated administration, self-service portal
- **Copy:** Compliance reporting templates

### 6. WorkOS — Gold Standard for B2B SaaS
- **Copy:** Enterprise SSO with 50+ pre-built connectors, directory sync (SCIM)
- **Copy:** "Add enterprise features in minutes" — developer-friendly B2B API
- **Copy:** Admin Portal (embeddable for customers)

### 7. Clerk — Gold Standard for DX + UX
- **Copy:** Pre-built React/Next.js components, beautiful login UI
- **Copy:** Session management with device tracking
- **Copy:** Account dashboard (user self-service)
- **Copy:** Quick start with copy-paste components

### 8. Logto — Gold Standard for Open Source Modern Auth
- **Copy:** Connector system (pluggable social/OAuth providers)
- **Copy:** Organization-based multi-tenancy
- **Copy:** Clean developer API design
- **Copy:** Magic link + social + passwordless unified UX

### 9. SuperTokens — Gold Standard for Session Security
- **Copy:** Anti-CSRF with front-end token, session recipe system
- **Copy:** Core + service architecture (separation of concerns)
- **Copy:** Self-hosted with managed option

### 10. Ory — Gold Standard for Cloud-Native IAM
- **Copy:** Microservice decomposition (Kratos=identity, Keto=authz, Hydra=oauth2)
- **Copy:** ReBAC (Google Zanzibar model) in Ory Keto
- **Copy:** Risk-based authentication (Ory Kratos risk score)
- **Copy:** Kubernetes-native deployment

---

## Gap Resolution Status (vs Auth0/Keycloak Matrix)

Based on the [competitive research matrix](./research/auth0-keycloak-ggid-matrix.md),
31 gaps were identified. Resolution status:

### Resolved Gaps (21/31)

| # | Gap | Resolution | Phase |
|---|-----|-----------|-------|
| 1 | OpenAPI 3.1 spec | Complete spec (2400+ lines) | 10 |
| 2 | API reference docs | 561-line reference + examples | 10 |
| 3 | Architecture docs | C4 model with Mermaid | 10 |
| 4 | Security whitepaper | STRIDE + compliance mapping | 10 |
| 5 | Helm Chart | Full K8s deployment guide | 10 |
| 6 | Performance benchmarks | k6 scripts + baselines | 10 |
| 7 | Migration guides | Auth0 + Keycloak + Clerk | 10 |
| 8 | SDK documentation | Go + Node + Java + Python | 10 |
| 9 | Circuit breaker | Gateway middleware | 9 |
| 10 | Rate limiting (sliding window) | Redis sorted sets + Lua | 9 |
| 11 | Shadow traffic | Gateway middleware | 9 |
| 12 | IP allowlist | Gateway middleware | 9 |
| 13 | WASM plugins | Plugin SDK + TinyGo | 9 |
| 14 | Request ID propagation | Middleware | 9 |
| 15 | Audit SSE streaming | Real-time audit events | 9 |
| 16 | Role hierarchy | Effective permissions recursion | 9 |
| 17 | Webhook HMAC signing | Signed webhooks + retry | 9 |
| 18 | Custom JWT claims | Claim hooks + rules engine | 9 |
| 19 | Observability docs | Structured logging + SIEM | 10 |
| 20 | Password policy docs | NIST compliance + breach check | 10 |
| 21 | Multi-tenant docs | RLS deep dive | 10 |

### Pending Gaps (10/31)

| # | Gap | Plan | Phase |
|---|-----|------|-------|
| 22 | Token introspection (RFC 7662) | OAuth service | 11 |
| 23 | Token revocation (RFC 7009) | OAuth service | 11 |
| 24 | OIDC back-channel logout | Session management | 11 |
| 25 | DPoP (proof-of-possession) | OAuth enhancement | 11 |
| 26 | PAR (pushed auth requests) | OAuth enhancement | 11 |
| 27 | Risk-based authentication | Adaptive MFA | 11 |
| 28 | Passwordless (magic link) | Auth service | 11 |
| 29 | GDPR data export automation | Compliance | 11 |
| 30 | Key rotation automation | JWKS management | 11 |
| 31 | Step-up enforcement | Policy engine | 11 |

### Gap Closure Rate

```
Resolved:     21 / 31  (67.7%)
Pending:      10 / 31  (32.3%)
Target:       28 / 31  (90%) by end of Phase 11
```

---

## Quick Comparison: GGID vs Key Competitors

A focused comparison against the 5 most comparable IAM platforms.

### Feature Summary (30+ rows)

| # | Feature | GGID | Auth0 | Keycloak | Clerk | Logto | Casdoor |
|---|---------|------|-------|----------|-------|-------|---------|
| **Authentication** | | | | | | | |
| 1 | Username/Password | Yes | Yes | Yes | Yes | Yes | Yes |
| 2 | MFA (TOTP) | Yes | Yes | Yes | Yes | Yes | Yes |
| 3 | WebAuthn / Passkeys | Yes | Yes | Beta | Yes | Beta | No |
| 4 | Social Login (9 providers) | Yes | Yes | Yes | Yes | Yes | Yes |
| 5 | Passwordless (magic link) | Phase 11 | Yes | Yes | Yes | Yes | No |
| 6 | LDAP/Active Directory | Yes | Yes (Enterprise) | Yes | No | No | No |
| 7 | Step-up authentication | Yes | Yes (Actions) | No | No | No | No |
| | | | | | | | |
| **Standards & Protocols** | | | | | | | |
| 8 | OAuth 2.0 / OIDC | Yes | Yes | Yes | Yes | Yes | Yes |
| 9 | OAuth 2.1 alignment | Yes | Partial | No | No | Partial | No |
| 10 | SAML 2.0 SSO | Yes | Yes (Enterprise) | Yes | No | No | No |
| 11 | SCIM 2.0 provisioning | Yes | Yes (Enterprise) | Yes | No | No | No |
| 12 | PKCE enforcement | Yes | Yes | Optional | Yes | Yes | No |
| 13 | DPoP (RFC 9449) | Phase 11 | No | No | No | No | No |
| | | | | | | | |
| **Access Control** | | | | | | | |
| 14 | RBAC | Yes | Yes | Yes | Yes | Yes | Yes |
| 15 | ABAC | Yes | Custom rules | Yes | No | No | No |
| 16 | Role hierarchy | Yes | No | Yes | No | No | No |
| 17 | Custom JWT claims | Yes (hooks) | Yes (Actions) | Yes (mappers) | No | Yes | No |
| | | | | | | | |
| **Multi-Tenancy** | | | | | | | |
| 18 | Tenant isolation (RLS) | Yes (PostgreSQL) | Orgs | Realms | Orgs | No | No |
| 19 | Per-tenant branding | Yes | Yes | Yes (themes) | Yes | No | No |
| 20 | Per-tenant rate limits | Yes | No | No | No | No | No |
| 21 | Per-tenant config | Yes | Yes | Yes (realm) | Yes | No | No |
| | | | | | | | |
| **Developer Experience** | | | | | | | |
| 22 | Go SDK | Yes | No | No | No | No | Yes |
| 23 | Node.js SDK | Yes | Yes | No | Yes | Yes | Yes |
| 24 | Java SDK | Yes | Yes | No | No | No | No |
| 25 | Admin Console (Web UI) | Yes (Next.js) | Yes | Yes | Yes | Yes | Yes |
| 26 | OpenAPI 3.1 spec | Yes | Yes | Yes | Yes | Yes | No |
| 27 | GraphQL API | No | No | No | No | No | No |
| | | | | | | | |
| **Infrastructure** | | | | | | | |
| 28 | Docker / Docker Compose | Yes | N/A (SaaS) | Yes | N/A | Yes | Yes |
| 29 | Kubernetes / Helm | Yes | N/A | Yes (Operator) | N/A | Yes | Yes |
| 30 | Horizontal scaling (stateless) | Yes | N/A | Yes | N/A | Yes | Yes |
| 31 | PostgreSQL (RLS) | Yes | N/A | Yes | N/A | Yes | Yes |
| | | | | | | | |
| **Security & Compliance** | | | | | | | |
| 32 | Audit log (immutable) | Yes (NATS + hash chain) | Yes | Yes | Yes | Yes | Yes |
| 33 | Webhooks (HMAC signed) | Yes | Yes | Yes (events) | Yes | Yes | No |
| 34 | GDPR tools | Phase 11 | Yes | Manual | Manual | Manual | Manual |
| 35 | License | Apache 2.0 | Commercial | Apache 2.0 | Commercial | MIT | Apache 2.0 |

### Competitive Positioning

| Strength | GGID Advantage |
|----------|---------------|
| **Open Source** | Apache 2.0 (vs Auth0/Clerk commercial) |
| **Multi-Tenancy** | PostgreSQL RLS at DB level (strongest isolation) |
| **Policy Engine** | RBAC + ABAC in one engine (most competitors only RBAC) |
| **Go-Native** | Built in Go, SDK in Go (Auth0/Keycloak are Java) |
| **Self-Hosted** | Full self-host with Docker/K8s (vs SaaS lock-in) |
| **Audit Pipeline** | NATS JetStream with hash chaining (tamper-evident) |

### Where Competitors Excel

| Gap | Best Alternative | GGID Plan |
|-----|-----------------|-----------|
| Marketplace/Integrations | Auth0 (400+ connections) | Build connector framework |
| Hosted UI customization | Clerk (best-in-class) | Expand white-label widget |
| Community size | Keycloak (large ecosystem) | Grow open-source community |
| Enterprise SSO breadth | Auth0 (all protocols) | SAML done, add CAS/WS-Fed |
| Analytics dashboard | Clerk (beautiful insights) | Expand Console analytics |
