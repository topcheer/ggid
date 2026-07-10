# IAM Competitor Feature Update Report — 2024-2025

> **Research Date:** January 2025
> **Scope:** Auth0 (Okta), Keycloak, Clerk, Logto, Casdoor
> **Purpose:** Identify the latest feature trajectories of major IAM competitors and extract actionable insights for GGID's roadmap.
> **Related:** See `competitor-update-clerk-logto-casdoor.md` for an earlier focused analysis of those three platforms.

---

## Table of Contents

1. [Auth0 (Okta)](#1-auth0-okta--2024-2025-updates)
2. [Keycloak](#2-keycloak--2024-2025-updates)
3. [Clerk](#3-clerk--2024-2025-updates)
4. [Logto](#4-logto--2024-2025-updates)
5. [Casdoor](#5-casdoor--2024-2025-updates)
6. [Comprehensive Comparison Table](#6-comprehensive-comparison-table-2025)
7. [Key Trends 2024-2025](#7-key-trends-2024-2025)
8. [Top 5 Actions for GGID](#8-top-5-actions-for-ggid)
9. [Sources](#9-sources)

---

## 1. Auth0 (Okta) — 2024-2025 Updates

### Overview

Auth0 (acquired by Okta in 2021) remains the commercial IAM market leader with a $1B+ ARR business. The 2024-2025 cycle has focused on converging Auth0's developer experience with Okta's enterprise capabilities, strengthening security automation, and expanding B2B organization features.

**Sources:**
- [Auth0 Changelog](https://auth0.com/changelog)
- [Auth0 6-Month Product Lookback Blog](https://auth0.com/blog/unveiling-new-and-improved-product-features-6-month-lookback/)
- [Q4 2024 Platform Release Overview (Okta)](https://www.okta.com/resources/datasheets/auth0-platform-release-overview-q4-2024/)
- [Q2 2025 Platform Release Overview (Okta)](https://www.okta.com/sites/default/files/2025-07/Q2%20Auth0%20Platform%20Release%20Overview%20.pdf)
- [Auth0 Pricing](https://auth0.com/pricing)

### Key New Features (2024-2025)

#### Adaptive Security & Risk-Based Authentication
- **Adaptive MFA with Akamai Integration:** Combine Akamai and Auth0 risk signals to trigger MFA, deny sessions, or revoke access based on real-time risk indicators. Post-login Actions can read these risk signals for custom logic. ([Q2 2025 release](https://www.okta.com/sites/default/files/2025-07/Q2%20Auth0%20Platform%20Release%20Overview%20.pdf))
- **Bot Detection Slider:** Adjustable friction levels for the "When Risky" setting on the Bot Detection model, with integration support for Friendly Captcha and hCaptcha.
- **Security Center on Converged Platform:** Real-time monitoring of security events across the Okta + Auth0 unified platform.
- **Anomaly Detection:** Automated detection and blocking of credential stuffing, brute force, and suspicious IP patterns.

#### Organizations (B2B)
- **Organization Name for Login Flows:** Organizations can now launch login flows via the Authentication API using their human-readable name (not just the org ID), improving UX.
- **Home Realm Discovery with IdP Domains:** Identifier-first HRD automatically routes users to the correct identity provider based on their email domain and Organization membership.
- **Multi-Org Membership Prompt:** New Universal Login prompt displays when users belong to multiple organizations, letting them pick which org to authenticate into.
- **PKCE and Attribute Mapping on OIDC Connections:** More secure connections between Auth0 and external IdPs, with attribute sync into the Auth0 tenant.

#### Developer Experience
- **Node 18 for Actions/Rules/Hooks:** Upgraded runtime for serverless extensibility via Auth0 Actions.
- **Auth0 Terraform Provider V1 (Beta):** Infrastructure-as-code management of Auth0 resources.
- **Enhanced Management API:** Expanded API surface for programmatic tenant management.
- **Teams Security Policies (SSO Enforcement):** Mandate login through enterprise IdP for dashboard and team access.

#### Passwordless & Universal Login
- **Passwordless on New Universal Login:** Email/phone OTP-based passwordless login natively in Universal Login.
- **Additional Email Provider Support:** Azure Communication Service and Microsoft Modern Auth as preferred authentication methods.
- **Guardian App Localization:** Push-notification MFA authenticator now supports 6+ languages (French-Canadian, Portuguese-Brazil, Spanish-Argentina, etc.).
- **Universal Login Language Support:** Added Basque, Catalan, Galician, Norwegian, and Welsh.

#### Fine-Grained Authorization (FGA)
- **Auth0 FGA:** A standalone service (powered by Google Zanzibar-style relationship tuples) that enables centralized, flexible, fine-grained authorization models — from simple RBAC to complex ABAC and relationship-based access control. ([docs.fga.dev](https://docs.fga.dev/))

### Pricing (2025)

| Tier | Price | Key Features |
|------|-------|-------------|
| **Free** | $0 | 7,500 MAUs, basic auth, social login |
| **Essentials** | $35/mo | 10K MAUs, Organizations, Actions, passwordless |
| **Professional** | $240/mo | Adaptive MFA, Bot Detection, SSO |
| **Enterprise** | Custom | FGA, SCIM, SAML, SLA, dedicated support |

### What GGID Can Learn

1. **Extensibility Model (Actions):** Auth0's Actions framework — serverless functions triggered at pre-login, post-login, and pre-registration — is the gold standard for extensibility. GGID's webhooks system should evolve toward a similar plugin/action model where custom logic can run at well-defined lifecycle points.
2. **Adaptive/Risk-Based Auth:** Combining risk signals from multiple sources (CDN, IP reputation, device fingerprint) to dynamically adjust MFA requirements is becoming table stakes. GGID's audit + NATS event pipeline is well-positioned to feed a risk scoring engine.
3. **FGA as a Service:** Decoupling fine-grained authorization into a separate, relationship-based service is a powerful pattern. GGID's policy engine could evolve toward Zanzibar-style tuple-based authorization.
4. **Converged Platform Strategy:** Merging CIAM (Auth0) and workforce IAM (Okta) into a unified platform with shared security center is a strategic direction worth monitoring.

---

## 2. Keycloak — 2024-2025 Updates

### Overview

Keycloak, the Red Hat-backed open-source IAM, reached version 26.x in late 2024 and continues through 26.4+ in 2025. The v26 release is the most significant update in years, introducing persistent sessions, organizations (multi-tenancy), OpenID4VCI, and MCP authorization server support.

**Sources:**
- [Keycloak 26.0.0 Release Announcement](https://www.keycloak.org/2024/10/keycloak-2600-released)
- [Keycloak 26.4.0 Release Announcement](https://www.keycloak.org/2025/09/keycloak-2640-released)
- [Keycloak 26 Feature Overview](https://www.keycloak-saas.com/en/keycloak-26-all-the-new-features-of-the-latest-version)
- [OpenID4VCI Credential Issuer Guide](https://www.keycloak.org/2026/01/issue-credentials-over-openid4vci)
- [Keycloak GitHub](https://github.com/keycloak/keycloak) — ~24K stars

### Key New Features (2024-2025)

#### Persistent User Sessions (v26 Default)
- User sessions are now stored in the database by default, surviving server restarts, migrations, and multi-site failover.
- This is a fundamental architecture change enabling high-availability and multi-site deployments.

#### Organizations (Multi-Tenancy)
- **Organization Support (v26):** Hierarchical structure for grouping users, roles, and resources within realms.
- Organizations can have their own identity providers, branding, and authentication flows.
- This brings Keycloak's multi-tenancy closer to what Auth0 Organizations and Logto Organizations offer.

#### OpenTelemetry Integration
- Built-in distributed tracing support via OpenTelemetry, improving observability of authentication flows across services.
- Integrates with Jaeger, Zipkin, and other OTel-compatible backends.

#### DPoP (Demonstration of Proof of Possession)
- Enhanced DPoP support for preventing access token theft/replay.
- Sender-constrained tokens that bind access tokens to a specific client's key pair.

#### Verifiable Credentials (OpenID4VCI)
- Keycloak can act as a **verifiable credential issuer** using the OpenID4VCI protocol.
- Users can receive verifiable credentials as digital proofs of identity, aligning with decentralized identity trends.
- Supports credential types: personal identity, roles/permissions, professional qualifications.

#### MCP Authorization Server
- Keycloak 26.4+ can serve as an **authorization server for MCP (Model Context Protocol)**.
- Enables AI agents to obtain OAuth tokens via Keycloak for accessing MCP servers.
- Note: Full MCP 2025-06-18 spec compliance (resource indicators) is not yet complete.

#### Declarative User Profile
- Configurable user attributes without custom code — define attribute schemas, validation rules, and permissions through admin configuration.
- Reduces the need for custom SPIs for user profile customization.

#### Multiple Social Broker Instances
- Simultaneous configuration of multiple instances of the same social authentication provider (e.g., multiple Google Workspace tenants).
- Simplifies management of multi-tenant social login scenarios.

#### Redesigned Login Theme
- Completely redesigned login UI with a modern, intuitive interface.
- Improved theming system for brand customization.

#### Client Library Separation
- Starting from v26, client libraries (adapters) are released independently from the Keycloak server.
- Enables faster adapter updates and better version compatibility management.

### Pricing

- **Self-hosted:** Free (Apache 2.0)
- **Keycloak as a Service (Phase Two, etc.):** From ~$50/mo for managed hosting
- **Red Hat Build of Keycloak:** Enterprise subscription via Red Hat

### What GGID Can Learn

1. **Persistent Sessions Architecture:** Keycloak's move to DB-backed sessions by default validates GGID's PostgreSQL session approach. Consider adding session persistence to survive service restarts.
2. **Declarative User Profile:** Allowing user attribute schemas to be configured at runtime (no code changes) is a powerful DX feature. GGID could implement a JSON-schema-based user profile configuration system.
3. **Verifiable Credentials (OpenID4VCI):** Decentralized identity is gaining traction. GGID should track OpenID4VCI as a potential future feature for enterprise customers.
4. **MCP Authorization:** Keycloak's MCP support shows that even enterprise IAM platforms are moving toward AI agent authentication. GGID should add MCP OAuth token issuance.
5. **Admin UI Patterns:** Keycloak's redesigned admin console sets a benchmark for self-service identity management UIs. GGID's Next.js console should aim for similar functionality.

---

## 3. Clerk — 2024-2025 Updates

### Overview

Clerk has positioned itself as the developer-first authentication platform with the best DX in the market. In 2024-2025, Clerk expanded from pure authentication into a full BAAS (Backend-as-a-Service) layer with billing, AI agent support, and a mature CLI toolchain.

**Sources:**
- [Clerk Changelog](https://clerk.com/changelog)
- [Clerk Pricing](https://clerk.com/pricing)
- [Clerk AI/MCP Documentation](https://clerk.com/docs/guides/ai/overview)
- [Clerk MCP Server Guide](https://clerk.com/docs/guides/ai/mcp/clerk-mcp-server)
- [API Version 2025-11-10 Upgrade Guide](https://clerk.com/docs/guides/development/upgrading/upgrade-guides/2025-11-10)

### Key New Features (2024-2025)

#### Clerk Billing (BAAS Expansion)
- **Built-in Billing System:** Account credits manageable from the Clerk Dashboard or Backend API. Each User or Organization has a credit balance with automated adjustment tracking.
- **API Version 2025-11-10:** New API version introduces improved billing response formatting for consistency across Frontend and Backend APIs.
- Clerk is expanding beyond authentication into a full application backend layer.

#### AI & MCP Integration
- **Clerk MCP Server:** Connect AI agents to Clerk's MCP server to access SDK snippets, implementation patterns, and user data through natural language.
- **AI-Powered App Support:** Documentation and tooling for building AI applications with secure authentication — including OAuth for agents, scoped API keys, and MCP server hosting.
- Positioning as the auth layer for AI-native applications.

#### CLI 2.0 (Built for Agents)
- **`clerk webhooks` Command Group:** Self-contained local webhook testing toolkit — `clerk webhooks listen` (relay tunnel), `clerk webhooks verify` (offline HMAC-SHA256 verification), `clerk webhooks token` (stable relay URLs).
- **`clerk impersonate`:** Short-lived sign-in URLs for debugging as a specific user. Every impersonation is audit-stamped with the admin's account.
- **Agent Contract:** Every CLI command respects a stable agent contract — predictable error codes, stdout for data, stderr for UI, no hidden interactive prompts in non-TTY contexts.

#### Organizations (B2B)
- **Default Organization Structure:** New dashboard accounts now start as Organizations (not personal accounts), streamlining B2B onboarding.
- **Organization-Level RBAC:** Role and permission management scoped to organizations.
- **Multi-Organization Membership:** Users can belong to multiple organizations with context switching.

#### Passkeys & Authentication
- **Passkeys GA:** Full passkey/WebAuthn support across registration and login flows.
- **Component-Based Auth UI:** Pre-built React components for sign-in, sign-up, MFA, and organization switching — with deep customization via the Component system.
- **Account Linking:** Automatic linking of accounts across different authentication methods.

### Pricing (2025)

| Tier | Price | Key Features |
|------|-------|-------------|
| **Free** | $0 | 10,000 MAUs (updated), all core auth features |
| **Pro** | $25/mo | 50,000 MAUs, Organizations, SAML SSO, Webhooks |
| **Enterprise** | Custom | Dedicated support, SSO, SLA, custom domains |

### What GGID Can Learn

1. **Component-Based Auth UI:** Clerk's pre-built, deeply customizable React components are the best-in-class auth UI approach. GGID's Next.js console should offer embeddable auth components (not just a hosted page).
2. **Agent-Friendly CLI:** The "agent contract" design (predictable error codes, stdout/stderr separation, no hidden prompts) is a blueprint for making GGID's tooling usable by AI agents.
3. **BAAS Expansion (Billing):** Expanding beyond pure auth into billing/credits shows the path to deeper platform stickiness. GGID should consider adding metering/billing hooks.
4. **MCP Server for Auth:** Providing an MCP server that lets AI agents interact with the IAM platform is a forward-looking feature that GGID should implement.
5. **DX Focus:** Clerk's documentation, CLI tools, and SDK quality are the benchmark. GGID's SDK quality (Go, Node, Java) must match this standard.

---

## 4. Logto — 2024-2025 Updates

### Overview

Logto has rapidly evolved from an Auth0 alternative into a comprehensive, developer-first IAM platform with one of the broadest protocol support sets in the OSS space. In 2024-2025, Logto shipped SAML IdP support, organization roles, MCP/agent positioning, and reached v1.40+ with continuous bi-weekly releases.

**Sources:**
- [Logto GitHub Releases](https://github.com/logto-io/logto/releases)
- [Logto Changelog Blog](https://blog.logto.io/categories/changelogs)
- [Logto February 2025 Changelog](https://blog.logto.io/changelogs-2025-february)
- [Logto December 2025 Changelog](https://blog.logto.io/changelogs/2025-december)
- [Logto Pricing](https://logto.io/pricing)
- [Top OSS IAM Providers 2025](https://blog.logto.io/top-oss-iam-providers-2025)

### Key New Features (2024-2025)

#### SAML Identity Provider (IdP)
- Logto can now act as a **SAML identity provider**, enabling enterprise SSO through the standardized SAML protocol.
- Previously Logto was only a SAML SP (service provider); now it can serve as the IdP itself.
- This completes Logto's enterprise SSO story: SAML SP + SAML IdP + OIDC + OAuth 2.0.

#### Organization Roles & Membership
- **Organization-Level RBAC:** Scoped roles and permissions per organization with hierarchical inheritance.
- **Transaction Role Creation:** Organization role creation with initial scopes is now transactional — invalid scope IDs won't leave partially created roles.
- **Membership Delta Webhooks:** `Organization.Membership.Updated` webhook enriched with explicit delta fields (`addedUserIds`/`removedUserIds`, `addedApplicationIds`/`removedApplicationIds`).
- **JIT Provisioning:** Just-in-time user provisioning for email-domain and enterprise SSO scenarios, with webhook payloads on invitation accept.

#### Enterprise SSO Enhancements
- **reCAPTCHA Customization:** Domain customization and checkbox mode for Enterprise tier.
- **Third-Party App Support:** Expanded to SPA and Native applications (previously server-side only).
- **Client IP Tracking:** Passwordless connectors now track client IP for audit/security.

#### App-Level Access Control (v1.41)
- Application-scoped permissions and roles, allowing different auth configurations per application within a tenant.
- Password expiration policies for enterprise compliance.
- Major Account Center upgrades with configurable username and verification-code rules.

#### MCP & Agent Authentication
- Logto is positioning itself as the identity layer for AI applications.
- OAuth-based machine-to-machine (M2M) authentication and personal access tokens (PATs) for API access.
- OAuth consent flows for third-party app authorization with customizable consent screens.

#### Account Linking & User Management
- **Account Linking:** Automatic and manual linking of accounts across different identifiers (email, phone, social).
- **User Impersonation:** Token exchange-based user impersonation for debugging.
- **OAuth Apps:** Third-party OAuth application registration with consent screens.

### Pricing (2025)

| Tier | Price | Key Features |
|------|-------|-------------|
| **OSS** | Free | All core features (SSO, RBAC, Organizations, MFA) |
| **Cloud Free** | $0 | 5,000 MAUs, dev tenant |
| **Cloud Pro** | From $5/mo | Pay-as-you-go MAUs, production support |
| **Enterprise** | Custom | Dedicated, SLA, SAML SSO, SCIM |

### What GGID Can Learn

1. **Connector System:** Logto's modular connector architecture — where social providers, email/SMS delivery, and enterprise SSO are pluggable connectors — is an excellent pattern. GGID's authprovider chain should evolve toward a similar plugin model.
2. **100% Free OSS Strategy:** Logto offers ALL features (SSO, RBAC, Organizations, MFA) in the free OSS edition. This is a strong competitive differentiator against Auth0's paywalled features. GGID's Apache 2.0 license already follows this philosophy.
3. **API-First Design:** Logto's entire feature set is accessible via well-documented REST APIs. GGID should ensure every feature has a first-class API surface.
4. **Bi-Weekly Release Cadence:** Logto ships updates every two weeks with detailed changelogs. GGID should adopt a predictable, transparent release rhythm.
5. **MCP/Agent Auth Positioning:** Logto is actively positioning for AI agent authentication with M2M tokens, PATs, and OAuth consent flows.

---

## 5. Casdoor — 2024-2025 Updates

### Overview

Casdoor has undergone a strategic pivot from a general-purpose IAM to an **AI-first identity platform**. The most significant development is the built-in MCP Gateway and A2A (Agent-to-Agent) protocol support, making it the first IAM platform purpose-built for AI agent authentication. Built in Go (same as GGID), Casdoor offers the broadest protocol coverage of any IAM in this analysis.

**Sources:**
- [Casdoor Official Website](https://casdoor.ai/)
- [Casdoor GitHub](https://github.com/casdoor/casdoor)
- [Casdoor GitHub Releases](https://github.com/casdoor/casdoor/releases)

### Key New Features (2024-2025)

#### AI-First Pivot
- **MCP Gateway:** Built-in Model Context Protocol server that lets AI agents manage users, applications, and permissions through natural language. Every MCP tool call is authenticated and authorized with fine-grained, scope-based permissions.
- **A2A Protocol:** Agent-to-Agent communication protocol support, enabling AI agents to authenticate and communicate with each other through Casdoor.
- **AI-First Design:** The entire platform is now marketed and architected as an "AI-first IAM / MCP gateway."

#### Protocol Breadth (Broadest in Class)
Casdoor supports the widest range of authentication protocols of any IAM in this comparison:
- **Standard:** OAuth 2.0, OIDC (OAuth 2.x), SAML 2.0
- **Legacy/Enterprise:** CAS, LDAP, RADIUS
- **Provisioning:** SCIM 2.0
- **Passwordless:** WebAuthn/Passkeys, TOTP, Face ID
- **Directory:** Google Workspace, Azure AD, Active Directory, Kerberos
- **AI:** MCP, A2A

#### Face ID Authentication
- Biometric authentication via Face ID, supporting iOS and Android native integrations.
- One of the few IAM platforms with native biometric authentication beyond WebAuthn.

#### Technology Stack
- **Backend:** Go with Beego framework
- **Frontend:** React (web UI)
- **Database:** MySQL, PostgreSQL, and others
- **Cache:** Optional Redis
- **License:** Apache 2.0 (same as GGID)
- **Go Version:** 1.25 (aligned with GGID's Go 1.25)

#### Casbin Integration
- Deep integration with Casbin for fine-grained authorization (ACL, RBAC, ABAC).
- Casbin policy models can be configured via the web UI without code changes.

#### SDK Ecosystem
- One of the broadest SDK ecosystems: Go, Java, Python, Node.js, PHP, .NET, Rust, Dart, Ruby, C/C++, plus frontend frameworks (React, Next.js, Vue, Angular, Flutter, etc.).
- Includes game engine SDKs (Unity, Firebase).

### Pricing

- **Self-hosted:** Free (Apache 2.0)
- **Cloud:** Not offered (self-host only)
- **Enterprise Support:** Available via Casdoor team

### What GGID Can Learn

1. **MCP Gateway:** Casdoor's built-in MCP server is the most direct competitive threat. GGID must implement MCP gateway functionality — allowing AI agents to authenticate, obtain scoped tokens, and manage identity resources via MCP.
2. **A2A Protocol:** Agent-to-Agent authentication is an emerging protocol. GGID should track and potentially implement A2A support for inter-agent communication scenarios.
3. **Protocol Breadth as Strategy:** Casdoor's strategy of supporting every protocol imaginable (CAS, RADIUS, Kerberos, Face ID) is effective for maximum compatibility. GGID should prioritize adding the protocols most relevant to its target market.
4. **Face ID / Biometric Auth:** Native Face ID support (beyond WebAuthn) is a differentiator for mobile-first applications. GGID could add Face ID/Touch ID support to its mobile SDKs.
5. **Same Language Advantage:** Being Go-based means Casdoor and GGID share the same ecosystem. GGID can potentially learn from Casdoor's architecture patterns and even contribute to shared Go IAM libraries.

---

## 6. Comprehensive Comparison Table (2025)

| Feature / Capability | Auth0 (Okta) | Keycloak | Clerk | Logto | Casdoor | **GGID** |
|---|---|---|---|---|---|---|
| **Latest Version** | Cloud (SaaS) | 26.4.x (Sep 2025) | API v2025-11-10 | v1.41 | Continuous | Active dev |
| **Release Cadence** | Quarterly | Monthly | Bi-weekly | Bi-weekly | Continuous | Sprint-based |
| **Architecture** | SaaS (multi-tenant) | Monolith (Java) | SaaS (multi-tenant) | Monolith (Node.js/TS) | Monolith (Go/Beego) | **Microservices (Go)** |
| **Language** | Node.js/Go (proprietary) | Java | TypeScript/React | TypeScript/Node.js | Go | **Go** |
| **License** | Proprietary | Apache 2.0 | Proprietary | Apache 2.0 (OSS) | Apache 2.0 | **Apache 2.0** |
| **GitHub Stars** | N/A (proprietary) | ~24,000 | N/A (proprietary) | ~10,000 | ~11,000 | — |
| **Cloud Offering** | Yes (primary) | Via partners | Yes (primary) | Yes (Logto Cloud) | No (self-host) | Planned |
| **Self-Hosted** | Limited (Appliance) | Yes (primary) | No | Yes (OSS) | Yes (primary) | **Yes (Docker)** |
| **OAuth 2.0** | Yes | Yes | Yes | Yes | Yes | **Yes** |
| **OIDC** | Yes (certified) | Yes (certified) | Yes | Yes | Yes | **Yes** |
| **SAML 2.0** | Yes (SP + IdP) | Yes (SP + IdP) | Yes (SP) | Yes (SP + IdP) | Yes (SP + IdP) | **Yes (SP + IdP)** |
| **SCIM 2.0** | Yes (Pro+) | Yes | Yes (Pro+) | Yes | Yes | **Skeleton** |
| **LDAP** | Yes | Yes | No | No (limited) | Yes | **Yes** |
| **WebAuthn/Passkeys** | Yes | Yes | Yes (GA) | Yes | Yes | **Yes** |
| **Face ID** | No | No | No | No | Yes | No |
| **TOTP/MFA** | Yes (push, TOTP, SMS) | Yes (TOTP, push) | Yes (TOTP, SMS) | Yes (TOTP, backup codes) | Yes (TOTP) | **Yes (TOTP)** |
| **Adaptive MFA** | Yes (Akamai risk signals) | No | No | No | No | **No** |
| **Bot Detection** | Yes (slider, captcha) | No | No | No (reCAPTCHA Enterprise) | No | **No** |
| **Multi-Tenancy** | Organizations | Organizations (v26) | Organizations | Organizations | Organizations | **Tenant-scoped** |
| **B2B Organizations** | Yes (mature) | Yes (v26+) | Yes (default) | Yes (mature) | Yes | **Yes (org service)** |
| **RBAC** | Yes | Yes | Yes | Yes | Yes (Casbin) | **Yes (policy engine)** |
| **ABAC** | Yes (FGA) | Yes | No | No | Yes (Casbin) | **Yes (policy engine)** |
| **Fine-Grained Auth (FGA)** | Yes (Zanzibar-style) | Yes (policy) | No | No | Yes (Casbin) | **Partial** |
| **Declarative User Profile** | No | Yes (v26) | Partial | No | No | **No** |
| **Verifiable Credentials (OID4VCI)** | No | Yes (experimental) | No | No | No | **No** |
| **DPoP** | Yes | Yes (v26) | No | No | No | **No** |
| **MCP Authorization** | No | Yes (v26.4) | Yes (MCP server) | Partial (M2M/PAT) | Yes (MCP Gateway) | **No** |
| **A2A Protocol** | No | No | No | No | Yes | **No** |
| **AI Agent Auth** | Partial (Actions) | Partial (MCP) | Yes (MCP server) | Partial (M2M/PAT) | Yes (built-in) | **No** |
| **Social Login** | 30+ providers | 20+ providers | 20+ providers | 15+ providers | 10+ providers | **9 connectors** |
| **Enterprise SSO** | Yes (SAML/OIDC) | Yes (SAML/OIDC) | Yes (SAML) | Yes (SAML/OIDC) | Yes (SAML/OIDC) | **Yes (SAML/OIDC)** |
| **Audit Logs** | Yes | Yes | Yes | Yes | Yes | **Yes (NATS JetStream)** |
| **Compliance Reporting** | Yes (SOC2, HIPAA) | Self-managed | Yes (SOC2) | Yes (SOC2-ready) | Self-managed | **Planned** |
| **Webhooks** | Yes | Yes (events) | Yes | Yes (delta payloads) | Yes | **Yes** |
| **Actions/Extensibility** | Yes (Actions, Node 18) | Yes (SPIs, Java) | Yes (Backend API) | Yes (webhooks + API) | Limited | **Webhooks** |
| **Admin Console** | Yes (Dashboard) | Yes (Admin UI v26) | Yes (Dashboard) | Yes (Console) | Yes (Web UI) | **Yes (Next.js)** |
| **Pre-built Auth UI** | Universal Login | Login theme | Components (React) | Sign-in experience | Web UI templates | **Console only** |
| **CLI** | auth0-cli | kc.sh | clerk CLI 2.0 | logto CLI | N/A | **Planned** |
| **Infrastructure-as-Code** | Terraform (v1 beta) | No | Backend API | No | No | **No** |
| **Message Queue** | Internal | Internal | Internal | Internal | Internal | **NATS JetStream** |
| **Database** | Proprietary | Any (JPA) | Proprietary | PostgreSQL | MySQL/PostgreSQL | **PostgreSQL (RLS)** |
| **Cache** | Proprietary | Infinispan | Proprietary | Redis | Redis (optional) | **Redis** |
| **SDKs** | 10+ (JS, Python, etc.) | 7 (Java, JS, etc.) | 4 (JS/React, RN) | 25+ (all major) | 20+ (all major) | **3 (Go, Node, Java)** |
| **Docker/K8s** | Appliance | Yes (Helm) | N/A (SaaS) | Yes (Docker) | Yes (Docker/Helm) | **Yes (Compose)** |
| **Pricing (Entry)** | $0 (7.5K MAU) | Free | $0 (10K MAU) | Free (OSS/Cloud) | Free | **Free** |
| **Pricing (Pro)** | $35/mo | Free (self-host) | $25/mo | $5/mo | Free | **Free** |
| **OpenTelemetry** | Yes (internal) | Yes (v26) | No | No | No | **No** |

---

## 7. Key Trends 2024-2025

### Trend 1: AI Agent Authentication (MCP, OAuth for Agents)
**Impact: Transformative**

The Model Context Protocol (MCP) has become the standard interface for AI agents to interact with external systems, and every major IAM vendor is racing to support it:
- **Casdoor** built a full MCP Gateway with authenticated tool calls
- **Keycloak** added MCP authorization server support (v26.4)
- **Clerk** shipped an MCP server for agent-driven identity management
- **Logto** is positioning M2M/PAT tokens for agent scenarios
- **Auth0** has Actions extensibility but no native MCP support yet

The November 2025 MCP spec update (mandatory PKCE, async tasks, Streamable HTTP) has further hardened agent authentication requirements. IAM platforms without MCP support will be irrelevant for AI-native applications within 12-18 months.

### Trend 2: Passkey Adoption Acceleration
**Impact: High**

Passkeys have moved from experimental to GA across all platforms:
- **Clerk**: Full GA with component-based passkey UI
- **Auth0**: Passwordless on Universal Login with passkey support
- **Keycloak**: WebAuthn support enhanced in v26
- **Logto**: Passkeys as MFA authenticator
- **Casdoor**: WebAuthn + Face ID for biometric auth

The FIDO Alliance reported passkey usage doubled in 2024. All IAM platforms now treat passkeys as a first-class authentication method, not an opt-in feature.

### Trend 3: B2B Organization Features Maturing
**Impact: High**

Multi-tenancy through "Organizations" has become the standard B2B pattern:
- **Auth0**: Organizations with Home Realm Discovery, multi-org prompts
- **Keycloak**: Added Organizations in v26 (previously realm-only)
- **Clerk**: Made Organizations the default account structure
- **Logto**: Organization roles, JIT provisioning, membership delta webhooks
- **Casdoor**: Organization-based multi-tenancy

The pattern is converging: users belong to organizations, organizations have their own IdPs/roles/branding, and users can belong to multiple organizations with context switching.

### Trend 4: Declarative / Config-Driven Identity
**Impact: Medium-High**

Configuration over code is becoming the preferred approach for identity customization:
- **Keycloak**: Declarative User Profile (configure attributes without SPIs)
- **Casdoor**: Casbin policy configuration via web UI
- **Logto**: Connector-based architecture (plug in providers via config)
- **Clerk**: Component-based UI configuration

The trend is toward empowering admins and developers to customize identity flows through configuration files and admin UIs rather than writing custom code.

### Trend 5: Adaptive / Risk-Based Authentication
**Impact: Medium (Growing)**

Risk-based authentication that dynamically adjusts MFA requirements based on signals:
- **Auth0**: Adaptive MFA with Akamai risk signal integration, Bot Detection slider
- **Others**: Still primarily static MFA policies

This is currently an Auth0 differentiator but is expected to become table stakes. The architecture requires: event ingestion (audit logs), risk scoring engine, and policy enforcement at authentication time.

### Trend 6: Decentralized Identity (Verifiable Credentials)
**Impact: Low-Medium (Emerging)**

OpenID4VCI and verifiable credentials are gaining enterprise traction:
- **Keycloak**: Experimental verifiable credential issuer (OpenID4VCI)
- **Others**: Not yet implemented

This is a future-looking trend that may become significant for enterprise and government use cases over the next 2-3 years.

### Trend 7: Extensibility Platform Evolution
**Impact: Medium**

IAM platforms are evolving from authentication services to extensibility platforms:
- **Auth0**: Actions (serverless functions at auth lifecycle points) + FGA
- **Clerk**: BAAS expansion (billing, backend API, MCP server)
- **Logto**: Connector system + webhooks + OAuth apps
- **Casdoor**: Casbin integration + MCP tools

The direction is clear: IAM platforms want to be the identity and authorization backbone for the entire application stack, not just the login page.

---

## 8. Top 5 Actions for GGID

Based on the competitor analysis above, here are the top 5 priorities for GGID, ranked by urgency and competitive impact:

### Action 1: Implement MCP Gateway and Agent Authentication
**Priority: Critical | Timeline: Q1 2025**

**Justification:** Casdoor, Keycloak, Clerk, and Logto all have MCP support shipping or in progress. Without MCP gateway functionality, GGID will be excluded from the AI-native application market entirely. This is the single highest-impact gap.

**Implementation:**
- Add an MCP server endpoint to the gateway service that accepts OAuth 2.1 PKCE-based authentication
- Implement scope-based permission checks for MCP tool calls (user management, role assignment, org operations)
- Support the MCP 2025-06-18 spec (resource indicators, Streamable HTTP transport)
- Expose identity management operations (create user, assign role, check permission) as MCP tools

**Competitive Reference:** Casdoor's MCP Gateway, Clerk's MCP Server, Keycloak v26.4 MCP authorization

---

### Action 2: Ship Pre-Built, Embeddable Auth UI Components
**Priority: High | Timeline: Q1-Q2 2025**

**Justification:** Clerk's component-based auth UI is a major competitive advantage — developers can embed sign-in/sign-up/MFA flows as React components with deep customization. GGID currently only offers a hosted admin console, not embeddable end-user auth components. This is the biggest DX gap.

**Implementation:**
- Build a React component library (`@ggid/auth-components`) with: `<SignIn />`, `<SignUp />`, `<MFAVerify />`, `<OrganizationSwitcher />`, `<UserProfile />`
- Support theming, custom fields, and passkey enrollment
- Make components work with GGID's OAuth/OIDC endpoints
- Add a Next.js middleware integration for route protection

**Competitive Reference:** Clerk Components, Auth0 Universal Login, Logto sign-in experience

---

### Action 3: Complete SCIM 2.0 Implementation and Enterprise Provisioning
**Priority: High | Timeline: Q1 2025**

**Justification:** SCIM 2.0 is supported by Auth0, Keycloak, Logto, and Casdoor. GGID currently has only a SCIM skeleton. Enterprise customers (especially those migrating from Okta/Auth0) require SCIM for automated user provisioning from HR systems and IdPs. Without full SCIM, GGID cannot compete in the enterprise B2B market.

**Implementation:**
- Implement SCIM 2.0 `/Users` and `/Groups` endpoints with PATCH, POST, PUT, DELETE
- Support SCIM filtering (`eq`, `co`, `sw`, `ew`, `pr`, AND/OR)
- Add SCIM provisioning webhooks (user created, updated, deactivated)
- Test against the SCIM 2.0 conformance test suite (scim.dev)
- Integrate with Azure AD, Okta, and Google Workspace provisioning

**Competitive Reference:** Auth0 SCIM (Pro+), Keycloak SCIM, Logto JIT provisioning

---

### Action 4: Add Adaptive Risk Scoring to Authentication
**Priority: Medium | Timeline: Q2 2025**

**Justification:** Auth0's Adaptive MFA (with Akamai risk signals) is currently a unique differentiator, but risk-based authentication is becoming table stakes. GGID's audit service (NATS JetStream) and middleware already capture rich event data that can feed a risk scoring engine.

**Implementation:**
- Build a risk scoring service that consumes events from the audit pipeline
- Score based on: IP reputation, geolocation anomalies, device fingerprint, login velocity, failed attempt history
- Implement step-up authentication: if risk score exceeds threshold, require additional MFA
- Add configurable risk policies via the admin console
- Expose risk scores via API for post-login Actions/webhooks

**Competitive Reference:** Auth0 Adaptive MFA + Akamai, Bot Detection slider

---

### Action 5: Expand SDK Ecosystem and Add Developer Tooling
**Priority: Medium | Timeline: Q2-Q3 2025**

**Justification:** GGID currently has 3 SDKs (Go, Node, Java). Logto has 25+, Casdoor has 20+, and even Clerk has better SDK coverage with superior DX. The SDK gap directly limits adoption — developers choose IAM platforms based on SDK availability for their stack.

**Implementation:**
- Add Python SDK (FastAPI/Django integration)
- Add PHP SDK (Laravel/Symfony integration)
- Add .NET SDK (ASP.NET Core integration)
- Add Rust SDK (for performance-critical applications)
- Build a `ggid` CLI tool with agent-friendly design (stable error codes, stdout/stderr separation)
- Add Terraform provider for infrastructure-as-code management
- Publish SDKs to all major package registries (npm, PyPI, NuGet, crates.io, Maven)

**Competitive Reference:** Logto's 25+ SDKs, Casdoor's 20+ SDKs, Clerk CLI 2.0, Auth0 Terraform Provider

---

## 9. Sources

### Auth0 (Okta)
- [Auth0 Changelog](https://auth0.com/changelog)
- [Auth0 6-Month Product Lookback](https://auth0.com/blog/unveiling-new-and-improved-product-features-6-month-lookback/)
- [Q4 2024 Platform Release Overview](https://www.okta.com/resources/datasheets/auth0-platform-release-overview-q4-2024/)
- [Q2 2025 Platform Release Overview (PDF)](https://www.okta.com/sites/default/files/2025-07/Q2%20Auth0%20Platform%20Release%20Overview%20.pdf)
- [Auth0 Pricing](https://auth0.com/pricing)
- [Auth0 FGA Documentation](https://docs.fga.dev/)
- [Auth0 Pricing Guide 2025](https://dev.saasworthy.com/blog/auth0-pricing-plans-guide)

### Keycloak
- [Keycloak 26.0.0 Release](https://www.keycloak.org/2024/10/keycloak-2600-released)
- [Keycloak 26.4.0 Release](https://www.keycloak.org/2025/09/keycloak-2640-released)
- [Keycloak 26 Feature Overview](https://www.keycloak-saas.com/en/keycloak-26-all-the-new-features-of-the-latest-version)
- [OpenID4VCI Credential Issuer Guide](https://www.keycloak.org/2026/01/issue-credentials-over-openid4vci)
- [Keycloak GitHub](https://github.com/keycloak/keycloak)

### Clerk
- [Clerk Changelog](https://clerk.com/changelog)
- [Clerk Pricing](https://clerk.com/pricing)
- [Clerk AI/MCP Documentation](https://clerk.com/docs/guides/ai/overview)
- [Clerk MCP Server Guide](https://clerk.com/docs/guides/ai/mcp/clerk-mcp-server)
- [API Version 2025-11-10 Upgrade](https://clerk.com/docs/guides/development/upgrading/upgrade-guides/2025-11-10)

### Logto
- [Logto GitHub Releases](https://github.com/logto-io/logto/releases)
- [Logto Changelog Blog](https://blog.logto.io/categories/changelogs)
- [Logto February 2025 Changelog](https://blog.logto.io/changelogs-2025-february)
- [Logto December 2025 Changelog](https://blog.logto.io/changelogs/2025-december)
- [Logto Pricing](https://logto.io/pricing)
- [Top OSS IAM Providers 2025](https://blog.logto.io/top-oss-iam-providers-2025)

### Casdoor
- [Casdoor Official Website](https://casdoor.ai/)
- [Casdoor GitHub](https://github.com/casdoor/casdoor)
- [Casdoor GitHub Releases](https://github.com/casdoor/casdoor/releases)

### Industry Analysis
- [SSO and Identity Providers Deep Dive 2026](https://www.youngju.dev/blog/culture/2026-05-16-sso-identity-providers-2026-keycloak-26-authentik-authelia-auth0-okta-aws-cognito-entra-id-deep-dive.en)
- [MCP November 2025 Spec Analysis](https://agentmarketcap.ai/blog/2026/04/06/mcp-oauth-pkce-agent-authentication-enterprise-security)

---

*Document generated: January 2025 | GGID IAM Suite — Competitive Intelligence*
