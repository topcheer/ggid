/* ============================================
   GGID.dev — Internationalization (i18n)
   Supports: English (en), Chinese (zh)
   ============================================ */

const translations = {
  en: {
    // Nav
    'nav.features': 'Features',
    'nav.architecture': 'Architecture',
    'nav.security': 'Security',
    'nav.docs': 'Docs',
    'nav.github': 'GitHub',
    'nav.getStarted': 'Get Started',

    // Hero
    'hero.badge': 'Apache 2.0 · Production Ready · v0.3.5',
    'hero.title1': 'Identity & Access',
    'hero.title2': 'for the AI Era',
    'hero.desc': 'A production-grade IAM platform: authentication, authorization, SSO, multi-tenancy, audit logging, and AI agent identity. Built with Go microservices. Open source.',
    'hero.cta.primary': 'Quick Start',
    'hero.cta.secondary': 'View on GitHub',
    'hero.terminal.title': 'deploy/e2e-docker-test.sh',
    'hero.terminal.comment': '# Start the entire stack in one command',
    'hero.terminal.cmd': 'docker compose',
    'hero.terminal.flag': '-f',
    'hero.terminal.up': 'deploy/docker-compose.yaml',
    'hero.terminal.up2': 'up -d',
    'hero.terminal.output1': '\u2713 Gateway healthz ........................ PASS',
    'hero.terminal.output2': '\u2713 Register user ............................ PASS',
    'hero.terminal.output3': '\u2713 Login + JWT .............................. PASS',
    'hero.terminal.output4': '\u2713 11/11 E2E tests ......................... ALL PASS',

    // Stats
    'stat.services': 'Microservices',
    'stat.sdks': 'Language SDKs',
    'stat.apis': 'API Endpoints',
    'stat.license': 'Open Source',

    // Features
    'features.label': 'Core Capabilities',
    'features.title': 'Everything You Need for Identity',
    'features.subtitle': 'From passwordless WebAuthn to SAML federation, GGID covers the entire identity lifecycle with enterprise-grade features.',

    'feat.oauth.title': 'OAuth 2.0 / OIDC',
    'feat.oauth.desc': 'Full RFC compliance: dynamic client registration, PKCE, PAR, device flow, FAPI 2.0, and token introspection.',
    'feat.oauth.tags': 'RFC 7591|PKCE|PAR|FAPI 2.0',

    'feat.saml.title': 'SAML 2.0 SSO',
    'feat.saml.desc': 'SP-initiated and IdP-initiated SSO with signed assertions, SLO, and SP metadata generation.',
    'feat.saml.tags': 'SSO|SLO|Signed Assertions',

    'feat.mfa.title': 'Multi-Factor Auth',
    'feat.mfa.desc': 'TOTP, WebAuthn/FIDO2, backup codes, and adaptive MFA with risk-based step-up authentication.',
    'feat.mfa.tags': 'TOTP|WebAuthn|Backup Codes',

    'feat.rbac.title': 'RBAC & ABAC',
    'feat.rbac.desc': 'Role-based and attribute-based access control with policy engine, dry-run evaluation, and permission inheritance.',
    'feat.rbac.tags': 'RBAC|ABAC|Policy Engine',

    'feat.multi.title': 'Multi-Tenancy',
    'feat.multi.desc': 'Native multi-tenant architecture with PostgreSQL Row-Level Security, tenant isolation, and per-tenant branding.',
    'feat.multi.tags': 'RLS|Isolation|Branding',

    'feat.scim.title': 'SCIM 2.0',
    'feat.scim.desc': 'Automated user provisioning and de-provisioning with SCIM 2.0 protocol support for enterprise SSO.',
    'feat.scim.tags': 'Provisioning|de-provisioning',

    'feat.audit.title': 'Audit & SIEM',
    'feat.audit.desc': 'Tamper-evident audit logging with hash chain, NATS JetStream delivery, and SIEM forwarding via webhook.',
    'feat.audit.tags': 'Hash Chain|JetStream|SIEM',

    'feat.agent.title': 'AI Agent Identity',
    'feat.agent.desc': 'First-class identity for AI agents: registration, token exchange, delegation chains, and MCP auth integration.',
    'feat.agent.tags': 'MCP Auth|Delegation|Scopes',

    'feat.webauthn.title': 'WebAuthn / Passkeys',
    'feat.webauthn.desc': 'Passwordless authentication with 6 attestation formats, passkey autofill, and device-bound SSO.',
    'feat.webauthn.tags': 'FIDO2|Passkeys|Device-Bound',

    'feat.ldap.title': 'LDAP / AD',
    'feat.ldap.desc': 'Enterprise directory integration with LDAP bind authentication, auto-provisioning, and OpenLDAP support.',
    'feat.ldap.tags': 'LDAP|Active Directory|Auto-Provision',

    // Architecture
    'arch.label': 'System Design',
    'arch.title': 'Microservices Architecture',
    'arch.subtitle': 'Seven independently deployable services communicating via gRPC and REST, with a centralized API gateway.',

    'arch.gateway': 'API Gateway',
    'arch.gateway.desc': 'JWT Verification · Rate Limiting · Routing',
    'arch.identity': 'Identity',
    'arch.identity.desc': 'Users · Tenants · SCIM',
    'arch.auth': 'Auth',
    'arch.auth.desc': 'Login · MFA · Sessions',
    'arch.oauth': 'OAuth',
    'arch.oauth.desc': 'OAuth 2.0 · OIDC · SAML',
    'arch.policy': 'Policy',
    'arch.policy.desc': 'RBAC · ABAC · Rules',
    'arch.org': 'Organization',
    'arch.org.desc': 'Teams · Departments',
    'arch.audit': 'Audit',
    'arch.audit.desc': 'Events · SIEM · Hash Chain',
    'arch.infra': 'Infrastructure',
    'arch.postgres': 'PostgreSQL 16',
    'arch.postgres.desc': 'RLS · Multi-tenant',
    'arch.redis': 'Redis 7',
    'arch.redis.desc': 'Sessions · Cache',
    'arch.nats': 'NATS JetStream',
    'arch.nats.desc': 'Event Bus',
    'arch.ldap': 'OpenLDAP',
    'arch.ldap.desc': 'Directory',

    // Code Examples
    'code.label': 'Developer Experience',
    'code.title': 'Build in Minutes, Not Days',
    'code.subtitle': 'Clean SDKs in 10 languages. Type-safe. Consistent. Documented.',

    // Security
    'security.label': 'Enterprise Ready',
    'security.title': 'Security & Compliance',
    'security.subtitle': 'Hardened by design with defense-in-depth at every layer.',

    'sec.gRPC': 'gRPC TLS Between Services',
    'sec.gRPC.desc': 'mTLS with fail-secure fallback for all inter-service communication.',
    'sec.sanitized': 'Sanitized Error Responses',
    'sec.sanitized.desc': 'Internal errors never leak to clients. All 500 responses are sanitized.',
    'sec.rate': 'Adaptive Rate Limiting',
    'sec.rate.desc': 'Per-IP, per-tenant, and per-endpoint rate limits with token bucket algorithm.',
    'sec.body': 'Request Body Size Limits',
    'sec.body.desc': 'Configurable payload limits to prevent DoS via oversized requests.',
    'sec.host': 'Host Header Validation',
    'sec.host.desc': 'DNS rebinding protection with configurable allowed hosts.',
    'sec.hsm': 'HSM / KMS Key Management',
    'sec.hsm.desc': 'PKCS#11 integration for hardware-backed JWT signing keys.',
    'sec.pii': 'PII Obfuscation',
    'sec.pii.desc': 'Automatic PII redaction in audit logs and SIEM forwarding.',
    'sec.csrf': 'CSRF Protection',
    'sec.csrf.desc': 'Cryptographically secure tokens with SameSite cookie enforcement.',
    'sec.ssrf': 'SSRF Prevention',
    'sec.ssrf.desc': 'Webhook URL validation with private IP range blocking.',
    'sec.chain': 'Tamper-Evident Audit',
    'sec.chain.desc': 'Cryptographic hash chain ensures audit log integrity.',
    'sec.compliance': 'NIS2 / CRA / PIPL',
    'sec.compliance.desc': 'Research-backed compliance frameworks for EU and China.',

    // SDKs
    'sdk.label': 'SDKs',
    'sdk.title': '10 Language SDKs',
    'sdk.subtitle': 'Official SDKs with consistent APIs across all supported languages.',

    // CTA
    'cta.title': 'Ready to Get Started?',
    'cta.desc': 'Deploy GGID in under 5 minutes with Docker Compose or the all-in-one image.',
    'cta.primary': 'Read the Docs',
    'cta.secondary': 'Star on GitHub',

    // Footer
    'footer.tagline': 'Production-grade Identity & Access Management for the AI era.',
    'footer.product': 'Product',
    'footer.developers': 'Developers',
    'footer.resources': 'Resources',
    'footer.company': 'Company',
    'footer.docs': 'Documentation',
    'footer.api': 'API Reference',
    'footer.sdks': 'SDKs',
    'footer.quickstart': 'Quick Start',
    'footer.guide': 'Deployment Guide',
    'footer.architecture': 'Architecture',
    'footer.security': 'Security',
    'footer.changelog': 'Changelog',
    'footer.contributing': 'Contributing',
    'footer.github': 'GitHub',
    'footer.license': 'Apache 2.0 License',
    'footer.copyright': '\u00a9 2024-2026 GGID. All rights reserved.',
  },

  zh: {
    // Nav
    'nav.features': '\u529f\u80fd',
    'nav.architecture': '\u67b6\u6784',
    'nav.security': '\u5b89\u5168',
    'nav.docs': '\u6587\u6863',
    'nav.github': 'GitHub',
    'nav.getStarted': '\u5f00\u59cb\u4f7f\u7528',

    // Hero
    'hero.badge': 'Apache 2.0 \u00b7 \u751f\u4ea7\u53ef\u7528 \u00b7 v0.3.5',
    'hero.title1': '\u8eab\u4efd\u8ba4\u8bc1\u4e0e\u8bbf\u95ee\u7ba1\u7406',
    'hero.title2': '\u4e3a AI \u65f6\u4ee3\u800c\u751f',
    'hero.desc': '\u751f\u4ea7\u7ea7 IAM \u5e73\u53f0\uff1a\u8eab\u4efd\u8ba4\u8bc1\u3001\u6388\u6743\u3001\u5355\u70b9\u767b\u5f55\u3001\u591a\u79df\u6237\u3001\u5ba1\u8ba1\u65e5\u5fd7\u548c AI \u4ee3\u7406\u8eab\u4efd\u3002\u57fa\u4e8e Go \u5fae\u670d\u52a1\u67b6\u6784\u3002\u5f00\u6e90\u514d\u8d39\u3002',
    'hero.cta.primary': '\u5feb\u901f\u5f00\u59cb',
    'hero.cta.secondary': 'GitHub \u67e5\u770b',
    'hero.terminal.title': 'deploy/e2e-docker-test.sh',
    'hero.terminal.comment': '# \u4e00\u6761\u547d\u4ee4\u542f\u52a8\u5168\u5806\u6808',
    'hero.terminal.cmd': 'docker compose',
    'hero.terminal.flag': '-f',
    'hero.terminal.up': 'deploy/docker-compose.yaml',
    'hero.terminal.up2': 'up -d',
    'hero.terminal.output1': '\u2713 \u7f51\u5173\u5065\u5eb7\u68c0\u67e5 ........................ \u901a\u8fc7',
    'hero.terminal.output2': '\u2713 \u6ce8\u518c\u7528\u6237 ............................ \u901a\u8fc7',
    'hero.terminal.output3': '\u2713 \u767b\u5f55 + JWT .............................. \u901a\u8fc7',
    'hero.terminal.output4': '\u2713 11/11 E2E \u6d4b\u8bd5 ......................... \u5168\u90e8\u901a\u8fc7',

    // Stats
    'stat.services': '\u5fae\u670d\u52a1',
    'stat.sdks': '\u8bed\u8a00 SDK',
    'stat.apis': 'API \u63a5\u53e3',
    'stat.license': '\u5f00\u6e90\u534f\u8bae',

    // Features
    'features.label': '\u6838\u5fc3\u80fd\u529b',
    'features.title': '\u8eab\u4efd\u7ba1\u7406\u7684\u4e00\u7ad9\u5f0f\u89e3\u51b3\u65b9\u6848',
    'features.subtitle': '\u4ece\u65e0\u5bc6\u7801 WebAuthn \u5230 SAML \u8054\u5408\u8ba4\u8bc1\uff0cGGID \u8986\u76d6\u4e86\u5168\u90e8\u8eab\u4efd\u751f\u547d\u5468\u671f\uff0c\u5177\u5907\u4f01\u4e1a\u7ea7\u7279\u6027\u3002',

    'feat.oauth.title': 'OAuth 2.0 / OIDC',
    'feat.oauth.desc': '\u5b8c\u5168 RFC \u5408\u89c4\uff1a\u52a8\u6001\u5ba2\u6237\u7aef\u6ce8\u518c\u3001PKCE\u3001PAR\u3001\u8bbe\u5907\u6388\u6743\u3001FAPI 2.0 \u548c\u4ee4\u724c\u81ea\u7701\u3002',
    'feat.oauth.tags': 'RFC 7591|PKCE|PAR|FAPI 2.0',

    'feat.saml.title': 'SAML 2.0 SSO',
    'feat.saml.desc': 'SP \u548c IdP \u53d1\u8d77\u7684\u5355\u70b9\u767b\u5f55\uff0c\u652f\u6301\u7b7e\u540d\u65ad\u8a00\u3001SLO \u548c SP \u5143\u6570\u636e\u751f\u6210\u3002',
    'feat.saml.tags': 'SSO|SLO|\u7b7e\u540d\u65ad\u8a00',

    'feat.mfa.title': '\u591a\u56e0\u7d20\u8ba4\u8bc1',
    'feat.mfa.desc': 'TOTP\u3001WebAuthn/FIDO2\u3001\u5907\u4efd\u7801\u548c\u57fa\u4e8e\u98ce\u9669\u7684\u81ea\u9002\u5e94 MFA \u52a0\u9a8c\u8ba4\u8bc1\u3002',
    'feat.mfa.tags': 'TOTP|WebAuthn|\u5907\u4efd\u7801',

    'feat.rbac.title': 'RBAC & ABAC',
    'feat.rbac.desc': '\u57fa\u4e8e\u89d2\u8272\u548c\u5c5e\u6027\u7684\u8bbf\u95ee\u63a7\u5236\uff0c\u5e26\u7b56\u7ea7\u5f15\u64ce\u3001\u9884\u6267\u884c\u548c\u6743\u9650\u7ee7\u627f\u3002',
    'feat.rbac.tags': 'RBAC|ABAC|\u7b56\u7ea7\u5f15\u64ce',

    'feat.multi.title': '\u591a\u79df\u6237',
    'feat.multi.desc': '\u539f\u751f\u591a\u79df\u6237\u67b6\u6784\uff0c\u57fa\u4e8e PostgreSQL \u884c\u7ea7\u5b89\u5168\u3001\u79df\u6237\u9694\u79bb\u548c\u54c1\u724c\u5b9a\u5236\u3002',
    'feat.multi.tags': 'RLS|\u9694\u79bb|\u54c1\u724c',

    'feat.scim.title': 'SCIM 2.0',
    'feat.scim.desc': '\u4f7f\u7528 SCIM 2.0 \u534f\u8bae\u81ea\u52a8\u5316\u7528\u6237\u914d\u7f6e\u548c\u53d6\u6d88\u914d\u7f6e\u3002',
    'feat.scim.tags': '\u914d\u7f6e|\u53d6\u6d88',

    'feat.audit.title': '\u5ba1\u8ba1\u4e0e SIEM',
    'feat.audit.desc': '\u9632\u7be1\u6539\u5ba1\u8ba1\u65e5\u5fd7\uff0c\u54c8\u5e0c\u94fe\u3001NATS JetStream \u6295\u9012\u548c SIEM \u8f6c\u53d1\u3002',
    'feat.audit.tags': '\u54c8\u5e0c\u94fe|JetStream|SIEM',

    'feat.agent.title': 'AI \u4ee3\u7406\u8eab\u4efd',
    'feat.agent.desc': '\u4e3a AI \u4ee3\u7406\u63d0\u4f9b\u7b2c\u4e00\u7c7b\u8eab\u4efd\uff1a\u6ce8\u518c\u3001\u4ee4\u724c\u4ea4\u6362\u3001\u59d4\u6258\u94fe\u548c MCP \u8ba4\u8bc1\u96c6\u6210\u3002',
    'feat.agent.tags': 'MCP \u8ba4\u8bc1|\u59d4\u6258|\u6743\u9650',

    'feat.webauthn.title': 'WebAuthn / \u5bc6\u94a5',
    'feat.webauthn.desc': '\u65e0\u5bc6\u7801\u8ba4\u8bc1\uff0c\u652f\u6301 6 \u79cd\u8bc1\u4e66\u683c\u5f0f\u3001\u5bc6\u94a5\u81ea\u52a8\u586b\u5145\u548c\u8bbe\u5907\u7ed1\u5b9a SSO\u3002',
    'feat.webauthn.tags': 'FIDO2|\u5bc6\u94a5|\u8bbe\u5907\u7ed1\u5b9a',

    'feat.ldap.title': 'LDAP / AD',
    'feat.ldap.desc': '\u4f01\u4e1a\u76ee\u5f55\u96c6\u6210\uff0c\u652f\u6301 LDAP \u7ed1\u5b9a\u8ba4\u8bc1\u3001\u81ea\u52a8\u914d\u7f6e\u548c OpenLDAP\u3002',
    'feat.ldap.tags': 'LDAP|\u57df\u63a7|\u81ea\u52a8\u914d\u7f6e',

    // Architecture
    'arch.label': '\u7cfb\u7edf\u8bbe\u8ba1',
    'arch.title': '\u5fae\u670d\u52a1\u67b6\u6784',
    'arch.subtitle': '\u4e03\u4e2a\u53ef\u72ec\u7acb\u90e8\u7f72\u7684\u670d\u52a1\uff0c\u901a\u8fc7 gRPC \u548c REST \u901a\u4fe1\uff0c\u5e26\u6709\u4e2d\u592e\u5316 API \u7f51\u5173\u3002',

    'arch.gateway': 'API \u7f51\u5173',
    'arch.gateway.desc': 'JWT \u9a8c\u8bc1 \u00b7 \u9650\u6d41 \u00b7 \u8def\u7531',
    'arch.identity': '\u8eab\u4efd\u670d\u52a1',
    'arch.identity.desc': '\u7528\u6237 \u00b7 \u79df\u6237 \u00b7 SCIM',
    'arch.auth': '\u8ba4\u8bc1\u670d\u52a1',
    'arch.auth.desc': '\u767b\u5f55 \u00b7 MFA \u00b7 \u4f1a\u8bdd',
    'arch.oauth': 'OAuth \u670d\u52a1',
    'arch.oauth.desc': 'OAuth 2.0 \u00b7 OIDC \u00b7 SAML',
    'arch.policy': '\u7b56\u7565\u670d\u52a1',
    'arch.policy.desc': 'RBAC \u00b7 ABAC \u00b7 \u89c4\u5219',
    'arch.org': '\u7ec4\u7ec7\u670d\u52a1',
    'arch.org.desc': '\u56e2\u961f \u00b7 \u90e8\u95e8',
    'arch.audit': '\u5ba1\u8ba1\u670d\u52a1',
    'arch.audit.desc': '\u4e8b\u4ef6 \u00b7 SIEM \u00b7 \u54c8\u5e0c\u94fe',
    'arch.infra': '\u57fa\u7840\u8bbe\u65bd',
    'arch.postgres': 'PostgreSQL 16',
    'arch.postgres.desc': 'RLS \u00b7 \u591a\u79df\u6237',
    'arch.redis': 'Redis 7',
    'arch.redis.desc': '\u4f1a\u8bdd \u00b7 \u7f13\u5b58',
    'arch.nats': 'NATS JetStream',
    'arch.nats.desc': '\u4e8b\u4ef6\u603b\u7ebf',
    'arch.ldap': 'OpenLDAP',
    'arch.ldap.desc': '\u76ee\u5f55\u670d\u52a1',

    // Code Examples
    'code.label': '\u5f00\u53d1\u8005\u4f53\u9a8c',
    'code.title': '\u51e0\u5206\u949f\u5185\u6784\u5efa\uff0c\u800c\u975e\u51e0\u5929',
    'code.subtitle': '10 \u79cd\u8bed\u8a00\u7684\u7eaf\u51c0 SDK\u3002\u7c7b\u578b\u5b89\u5168\u3002API \u4e00\u81f4\u3002\u6587\u6863\u5b8c\u5584\u3002',

    // Security
    'security.label': '\u4f01\u4e1a\u7ea7\u5c31\u7eea',
    'security.title': '\u5b89\u5168\u4e0e\u5408\u89c4',
    'security.subtitle': '\u8bbe\u8ba1\u5373\u5b89\u5168\uff0c\u6bcf\u4e00\u5c42\u90fd\u6709\u7eb5\u6df1\u9632\u5fa1\u3002',

    'sec.gRPC': '\u670d\u52a1\u95f4 gRPC TLS',
    'sec.gRPC.desc': '\u6240\u6709\u670d\u52a1\u95f4\u901a\u4fe1\u5747\u4f7f\u7528 mTLS\uff0c\u5e26\u5b89\u5168\u5931\u8d25\u56de\u9000\u3002',
    'sec.sanitized': '\u9519\u8bef\u54cd\u5e94\u6d88\u6bd2',
    'sec.sanitized.desc': '\u5185\u90e8\u9519\u8bef\u6c38\u4e0d\u6cc4\u9732\u7ed9\u5ba2\u6237\u7aef\u3002',
    'sec.rate': '\u81ea\u9002\u5e94\u9650\u6d41',
    'sec.rate.desc': '\u6309 IP\u3001\u79df\u6237\u548c\u7aef\u70b9\u9650\u6d41\uff0c\u4ee4\u724c\u6876\u7b97\u6cd5\u3002',
    'sec.body': '\u8bf7\u6c42\u4f53\u5927\u5c0f\u9650\u5236',
    'sec.body.desc': '\u53ef\u914d\u7f6e\u8d1f\u8f7d\u9650\u5236\uff0c\u9632\u6b62 DoS \u653b\u51fb\u3002',
    'sec.host': 'Host \u5934\u9a8c\u8bc1',
    'sec.host.desc': 'DNS \u91cd\u7ed1\u5b9a\u4fdd\u62a4\uff0c\u53ef\u914d\u7f6e\u5141\u8bb8\u7684 Host\u3002',
    'sec.hsm': 'HSM / KMS \u5bc6\u94a5\u7ba1\u7406',
    'sec.hsm.desc': 'PKCS#11 \u96c6\u6210\uff0c\u786c\u4ef6\u652f\u6301\u7684 JWT \u7b7e\u540d\u5bc6\u94a5\u3002',
    'sec.pii': 'PII \u8131\u654f',
    'sec.pii.desc': '\u5ba1\u8ba1\u65e5\u5fd7\u548c SIEM \u8f6c\u53d1\u4e2d\u81ea\u52a8\u8131\u654f\u3002',
    'sec.csrf': 'CSRF \u4fdd\u62a4',
    'sec.csrf.desc': '\u52a0\u5bc6\u5b89\u5168\u4ee4\u724c\u548c SameSite Cookie\u3002',
    'sec.ssrf': 'SSRF \u9632\u62a4',
    'sec.ssrf.desc': 'Webhook URL \u9a8c\u8bc1\uff0c\u5c4f\u853d\u5185\u7f51 IP\u3002',
    'sec.chain': '\u9632\u7be1\u6539\u5ba1\u8ba1',
    'sec.chain.desc': '\u52a0\u5bc6\u54c8\u5e0c\u94fe\u786e\u4fdd\u5ba1\u8ba1\u5b8c\u6574\u6027\u3002',
    'sec.compliance': 'NIS2 / CRA / PIPL',
    'sec.compliance.desc': '\u7814\u7a76\u652f\u6301\u7684\u6b27\u76df\u548c\u4e2d\u56fd\u5408\u89c4\u6846\u67b6\u3002',

    // SDKs
    'sdk.label': 'SDK',
    'sdk.title': '10 \u79cd\u8bed\u8a00 SDK',
    'sdk.subtitle': '\u5b98\u65b9 SDK\uff0c\u6240\u6709\u652f\u6301\u8bed\u8a00\u7684 API \u4e00\u81f4\u3002',

    // CTA
    'cta.title': '\u51c6\u5907\u5f00\u59cb\u4e86\u5417\uff1f',
    'cta.desc': 'Docker Compose \u6216 all-in-one \u955c\u50cf\uff0c5 \u5206\u949f\u5185\u90e8\u7f72 GGID\u3002',
    'cta.primary': '\u9605\u8bfb\u6587\u6863',
    'cta.secondary': 'GitHub \u70b9\u8d5e',

    // Footer
    'footer.tagline': '\u4e3a AI \u65f6\u4ee3\u8bbe\u8ba1\u7684\u751f\u4ea7\u7ea7\u8eab\u4efd\u8ba4\u8bc1\u4e0e\u8bbf\u95ee\u7ba1\u7406\u5e73\u53f0\u3002',
    'footer.product': '\u4ea7\u54c1',
    'footer.developers': '\u5f00\u53d1\u8005',
    'footer.resources': '\u8d44\u6e90',
    'footer.company': '\u5173\u4e8e',
    'footer.docs': '\u6587\u6863',
    'footer.api': 'API \u53c2\u8003',
    'footer.sdks': 'SDK',
    'footer.quickstart': '\u5feb\u901f\u5f00\u59cb',
    'footer.guide': '\u90e8\u7f72\u6307\u5357',
    'footer.architecture': '\u67b6\u6784',
    'footer.security': '\u5b89\u5168',
    'footer.changelog': '\u66f4\u65b0\u65e5\u5fd7',
    'footer.contributing': '\u8d21\u732e\u6307\u5357',
    'footer.github': 'GitHub',
    'footer.license': 'Apache 2.0 \u5f00\u6e90\u534f\u8bae',
    'footer.copyright': '\u00a9 2024-2026 GGID. \u4fdd\u7559\u6240\u6709\u6743\u5229\u3002',
  }
};

// Current language
let currentLang = localStorage.getItem('ggid-lang') || 'en';

// Apply translations
function applyTranslations(lang) {
  currentLang = lang;
  localStorage.setItem('ggid-lang', lang);
  document.documentElement.lang = lang;

  document.querySelectorAll('[data-i18n]').forEach(el => {
    const key = el.getAttribute('data-i18n');
    if (translations[lang] && translations[lang][key]) {
      el.textContent = translations[lang][key];
    }
  });

  document.querySelectorAll('[data-i18n-html]').forEach(el => {
    const key = el.getAttribute('data-i18n-html');
    if (translations[lang] && translations[lang][key]) {
      el.innerHTML = translations[lang][key];
    }
  });

  // Update active state in dropdown
  document.querySelectorAll('.lang-dropdown button').forEach(btn => {
    btn.classList.toggle('active', btn.dataset.lang === lang);
  });

  // Update button text
  const langLabel = lang === 'zh' ? '\u4e2d\u6587' : 'English';
  const langBtn = document.querySelector('.lang-btn .lang-current');
  if (langBtn) langBtn.textContent = langLabel;
}

// Initialize on load
document.addEventListener('DOMContentLoaded', () => {
  applyTranslations(currentLang);
});
