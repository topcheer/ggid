# GGID Documentation Index

## Getting Started

| Document | Description |
|----------|-------------|
| [Getting Started](guides/getting-started.md) | 5-minute quickstart from zero to first API call |
| [5-Minute Quickstart](guides/5-minute-quickstart.md) | Fastest path to running GGID |
| [Integration Guide](guides/integration-guide.md) | OAuth 2.1, Client Credentials, SAML, WebAuthn integration |
| [API Cookbook](api-cookbook.md) | 20 ready-to-run curl examples |
| [API Reference](guides/api-reference.md) | Endpoint catalog with parameters |
| [Console Admin Guide](guides/console-admin-guide.md) | Admin console walkthrough |
| [Onboarding Guide](guides/onboarding-guide.md) | New admin setup checklist |

## Authentication

| Document | Description |
|----------|-------------|
| [Authentication Flows](guides/authentication-flows.md) | All auth flows overview |
| [OAuth 2.1 Compliance](guides/oauth-2-1-compliance-checklist.md) | OAuth 2.1 spec compliance |
| [OAuth PKCE Deep Dive](guides/oauth-pkce-deep-dive.md) | PKCE flow details |
| [OAuth Token Exchange](guides/oauth-token-exchange.md) | RFC 8693 token exchange |
| [OIDC Advanced](guides/openid-connect-advanced.md) | OpenID Connect features |
| [MFA Architecture](guides/mfa-architecture.md) | Multi-factor authentication |
| [MFA Enforcement](guides/mfa-enforcement.md) | MFA policy configuration |
| [WebAuthn Deep Dive](guides/webauthn-deep-dive.md) | FIDO2/WebAuthn internals |
| [WebAuthn Setup](guides/webauthn-setup.md) | Passkey deployment guide |
| [Password Policy](guides/password-policy-guide.md) | Password requirements |
| [Password Strength](guides/password-strength-guide.md) | zxcvbn scoring engine (KB-082) |
| [Password Deprecation](guides/password-deprecation.md) | Passwordless migration |
| [Adaptive Authentication](guides/adaptive-authentication.md) | Risk-based step-up |
| [Step-Up Authentication](guides/step-up-authentication.md) | Progressive auth |
| [Social Login Setup](guides/social-login-setup.md) | Google/GitHub/Microsoft/Apple |
| [SAML Federation](guides/saml-federation-guide.md) | SAML 2.0 IdP/SP |
| [TAP (Temporary Access Pass)](guides/auth-method-policy.md) | Auth method policies |

## Authorization

| Document | Description |
|----------|-------------|
| [RBAC Design Patterns](guides/rbac-design-patterns.md) | Role-based access |
| [ABAC Policy](guides/abac-policy.md) | Attribute-based access |
| [ABAC Patterns](guides/abac-patterns.md) | ABAC implementation patterns |
| [ReBAC Console Guide](guides/rebac-console-guide.md) | Zanzibar relationship-based authz |
| [Conditional Access](guides/conditional-access-guide.md) | CAP policy configuration |
| [Continuous Access Evaluation](guides/continuous-access-evaluation.md) | CAE session-level (KB-081) |
| [Delegation Management](guides/delegation-management.md) | Scoped delegation API |
| [Access Reviews](guides/access-reviews.md) | Access certification |
| [Access Certification Guide](guides/access-certification-guide.md) | Campaign management |
| [Separation of Duties](guides/separation-of-duties-guide.md) | SoD rules and violations |
| [Policy Engine Internals](guides/policy-engine-internals.md) | PDP/PEP architecture |
| [Policy Evaluation](guides/policy-evaluation-engine.md) | Policy decision flow |
| [Privileged Access Management](guides/privileged-access-management.md) | PAM concepts |
| [Privileged Operations Audit](guides/privileged-operations-audit.md) | KB-279 audit logging |

## Security & ITDR

| Document | Description |
|----------|-------------|
| [ITDR Detection Rules](guides/itdr-detection-rules.md) | MITRE ATT&CK mapping |
| [ITDR Implementation](guides/itdr-implementation.md) | Identity threat detection |
| [Risk-Based Authentication](guides/risk-based-authentication.md) | Risk engine scoring |
| [Risk Score Guide](guides/risk-score-guide.md) | Risk signal categories |
| [Anomaly Detection](guides/anomaly-detection-guide.md) | Behavioral analytics |
| [Impossible Travel](guides/fraud-detection.md) | Geo-velocity detection |
| [Break-Glass Procedure](guides/break-glass-procedure.md) | Emergency access |
| [Audit Hash Chain](guides/audit-hash-chain.md) | Tamper-evident logging |
| [Audit Tamper Detection](guides/audit-tamper-detection.md) | Integrity verification |
| [SIEM Integration](guides/siem-integration-guide.md) | Splunk/ELA/Datadog |
| [Threat Modeling](guides/threat-modeling-guide.md) | STRIDE analysis |
| [Penetration Testing](guides/penetration-testing-guide.md) | Testing methodology |

## Identity Management

| Document | Description |
|----------|-------------|
| [Identity Lifecycle](guides/digital-identity-lifecycle.md) | JML automation |
| [HR Lifecycle](guides/hr-lifecycle.md) | Workday/HR sync |
| [HR JML Automation](guides/hr-jml-automation.md) | Joiner-Mover-Leaver |
| [SCIM Provisioning](guides/scim-provisioning-guide.md) | SCIM 2.0 inbound/outbound |
| [LDAP Integration](guides/ldap-integration-guide.md) | Active Directory/LDAP |
| [Bulk Import](guides/bulk-import-guide.md) | CSV/JSON user import |
| [Multi-Hash Password](guides/multi-hash-password.md) | Multi-algorithm hashing |
| [Attribute Mapping](guides/attribute-mapping-guide.md) | KB-063 attribute transformation |
| [Identity Federation](guides/identity-federation.md) | Federation patterns |
| [Federation Wizard](guides/federation-wizard.md) | Multi-IdP setup |
| [Tenant Onboarding](guides/tenant-onboarding-guide.md) | Multi-tenant setup |
| [Data Subject Rights](guides/dsar-automation.md) | GDPR/CCPA automation |

## Compliance

| Document | Description |
|----------|-------------|
| [Compliance Guide](guides/compliance-guide.md) | Framework overview |
| [Continuous Compliance Monitoring](guides/continuous-compliance-monitoring.md) | KB-280 CCM engine |
| [SOC2 Audit Prep](guides/soc2-audit-prep.md) | SOC 2 evidence |
| [NIS2/CRA Compliance](guides/nis2-cra-compliance.md) | EU regulations |
| [PIPL Compliance](guides/pipl-compliance.md) | China data protection |
| [Data Classification](guides/data-classification-guide.md) | PII/PHI handling |
| [Privacy by Design](guides/privacy-by-design.md) | Privacy patterns |

## Operations

| Document | Description |
|----------|-------------|
| [Deployment Guide](guides/deployment-guide.md) | Production deployment |
| [Docker Deployment](guides/docker-deployment.md) | Docker Compose setup |
| [Kubernetes Deployment](guides/kubernetes-deployment.md) | K8s/K3s manifests |
| [Database Setup](guides/database-setup.md) | PostgreSQL configuration |
| [Monitoring & Alerting](guides/monitoring-and-alerting.md) | Prometheus/Grafana |
| [Observability Guide](guides/observability-guide.md) | Tracing, metrics, logs |
| [Backup & Recovery](guides/backup-and-restore.md) | DR procedures |
| [Production Checklist](guides/production-checklist.md) | Go-live checklist |
| [Testing Guide](guides/testing-guide.md) | Test strategy and execution |
| [Key Rotation](guides/key-rotation.md) | JWT/cert/key lifecycle |
| [Secrets Management](guides/secrets-management.md) | Secret storage and rotation |

## API & SDKs

| Document | Description |
|----------|-------------|
| [API Reference](guides/api-reference.md) | Full endpoint catalog |
| [API Cookbook](api-cookbook.md) | 20 practical curl recipes |
| [API Security Checklist](guides/api-security-checklist.md) | API hardening |
| [API Rate Limiting](guides/rate-limiting-guide.md) | Throttling policies |
| [Webhook Setup](guides/webhook-setup.md) | Event subscriptions |
| [Webhook Delivery](guides/webhook-delivery.md) | HMAC signatures, retries |
| [Go SDK Guide](guides/go-sdk-guide.md) | Go integration |
| [Node SDK Guide](guides/node-sdk-guide.md) | Node.js integration |
| [React SDK](guides/sdk-integration-guide.md) | React hooks library |
| [Java SDK Guide](guides/java-sdk-guide.md) | Java integration |
| [GraphQL API](guides/graphql-api.md) | Identity queries |

## Platform & Console

| Document | Description |
|----------|-------------|
| [Feature Flags](guides/feature-flag-architecture.md) | Toggle configuration |
| [Branding Customization](guides/branding-guide.md) | White-label console |
| [i18n Setup](guides/i18n-setup.md) | 15-language support |
| [Frontend i18n](guides/frontend-i18n.md) | Console translation |
| [Notification Routing](guides/notification-routing.md) | Email/SMS/push |
| [Session Management](guides/session-management-guide.md) | Session policies |
| [Tenant Quota](guides/tenant-quota.md) | Resource limits |
| [Console Development](guides/console-development.md) | Contributing to console |

## Research Library (docs/research/)

48+ deep-dive technical research documents covering Zero Trust, ReBAC, OAuth 2.1, NIS2/CRA, WASM plugins, AI Agent Identity, Cloud IAM Federation, and more. See [research index](research/).
