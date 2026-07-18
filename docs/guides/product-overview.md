# Product Overview

## What is GGID?

GGID is an open-source Identity and Access Management (IAM) platform built in Go, designed for enterprises that need unified authentication, authorization, and identity threat detection in a single system. Unlike traditional IAM products that bolt on security features as afterthoughts, GGID is built from the ground up as a Zero Trust identity layer — every request is authenticated, every action is authorized, and every event is audit-logged with cryptographic integrity. The platform exposes standard protocols (OAuth 2.1, OIDC, SAML 2.0, SCIM 2.0, WebAuthn/FIDO2) so your applications integrate without lock-in, while internally delivering fine-grained authorization (ReBAC + ABAC), continuous access evaluation, and AI agent identity management that legacy IAM products cannot match.

## What Problems Does It Solve?

Organizations struggle with fragmented identity infrastructure: one system for SSO, another for RBAC, a third for audit compliance, and manual processes for user lifecycle management. GGID consolidates these into a single platform — eliminating the integration tax of stitching together Auth0 for authentication, CyberArk for privileged access, Splunk for audit, and custom code for policy enforcement. Key problems solved: **password fatigue** (passwordless WebAuthn/passkey support), **privilege creep** (automated access reviews and SoD enforcement), **compliance overhead** (continuous monitoring with SOC2/SOX/HIPAA/DORA evidence generation), **identity threats** (15 MITRE ATT&CK detection rules with real-time alerting), and **multi-tenant complexity** (PostgreSQL Row-Level Security for tenant isolation without application changes). For modern workloads, GGID also provides first-class **AI agent identity** — treating automated agents as principals with scoped delegation, not shared API keys.

## Why Choose GGID?

| Advantage | Detail |
|-----------|--------|
| **Open source** | Apache 2.0 license — no vendor lock-in, no per-user pricing |
| **Go performance** | Compiled binary, ~10x faster than Java IAM products (Keycloak), lower memory footprint |
| **Zero Trust native** | CAE continuous evaluation, per-request PDP, break-glass with full audit — not bolted on |
| **11 SDKs** | Go, React, TypeScript, Java, Python, C#, Rust, Ruby, PHP, Dart, React Native |
| **Compliance-ready** | Hash-chained audit trail, CCM engine, GDPR/CCPA/PIPL/NIS2/CRA framework support |
| **Cloud-native** | Docker Compose, Helm, K8s Operator, Terraform — deploy anywhere in minutes |
| **China GM support** | SM2/SM3/SM4 cryptographic compliance for Chinese market |

## Platform Capabilities

### Authentication
- OAuth 2.1 (PKCE, PAR, JAR, DPoP, Token Exchange)
- OpenID Connect (Discovery, UserInfo, Back-Channel Logout)
- WebAuthn / FIDO2 / Passkeys
- SAML 2.0 federation
- Social login (Google, GitHub, Microsoft, Apple)
- MFA (TOTP, SMS, push, biometric)
- Adaptive risk-based authentication
- China GM (SM2/SM3) signing

### Authorization
- ReBAC (Zanzibar-style relationship tuples)
- ABAC (attribute-based conditions)
- RBAC with role hierarchy
- Conditional Access Policies (CAP)
- Continuous Access Evaluation (CAE)
- Delegation management with scoped tokens
- PostgreSQL Row-Level Security for tenant isolation

### Security & ITDR
- 15 MITRE ATT&CK identity detection rules
- Unified risk engine (5 signal categories, 20 types)
- Hash-chained audit trail (HMAC-SHA256, tamper-evident)
- DLP egress with PII redaction
- SOAR playbook integration
- Impossible travel / geo-velocity detection
- Privileged operations audit (KB-279)

### Platform
- Multi-tenant with per-tenant branding and i18n (15 languages)
- 825+ console pages for admin self-service
- 363+ documentation guides
- WASM plugin architecture (wazero runtime)
- Webhook engine (HMAC signed, retry, dead-letter)
- SCIM 2.0 inbound/outbound provisioning
