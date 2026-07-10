# Open-Source IAM Landscape 2025: Competitive Analysis & Strategic Positioning

> Research date: July 2025
> Focus: Where GGID fits in the open-source IAM ecosystem and how to differentiate.

---

## 1. Executive Summary

The open-source Identity and Access Management (IAM) market in 2025 is more
competitive than ever. Enterprise demand for SSO, passkeys, and multi-tenant
identity has attracted significant investment, and at least eight active
projects compete for developer mindshare:

| Tier     | Projects                                           |
|----------|----------------------------------------------------|
| Mature   | Keycloak, Ory (Kratos + Hydra + Keto + Oathkeeper) |
| Growing  | Authentik, Zitadel, Casdoor, Logto                 |
| Emerging | Hanko, GGID                                        |

**Key 2025 trends observed:**

1. **Passkey adoption acceleration** — Nearly every project now ships
   WebAuthn/FIDO2 support. Passkey-first auth (Hanko) has gone mainstream.
2. **Multi-tenancy as a differentiator** — Zitadel, Logto, and GGID all
   position native multi-tenancy as a core feature; Ory's OSS edition
   remains single-tenant (enterprise license required for multi-tenant).
3. **Protocol convergence** — OIDC is table stakes. SAML 2.0 and SCIM 2.0
   are increasingly expected in enterprise-grade offerings. Zitadel added
   both SAML and SCIM to its core in 2024–2025.
4. **Go dominance** — Four of eight projects (Zitadel, Casdoor, Hanko, GGID)
   plus Ory are Go-based, reflecting the industry shift away from Java-heavy
   monoliths toward cloud-native, compiled-language services.
5. **AI/Agent positioning** — Casdoor rebranded as "Agent-first Identity,"
   signaling that AI authentication is the next competitive frontier.

GGID's positioning is unique: the only Go-native, microservices-based,
event-driven (NATS JetStream) IAM suite with built-in multi-tenancy via
PostgreSQL Row-Level Security (RLS). This combination does not exist in any
other open-source project.

---

## 2. Competitor Profiles

### 2.1 Keycloak 26+

- **Repo:** https://github.com/keycloak/keycloak
- **Docs:** https://www.keycloak.org/documentation
- **Architecture:** Java monolith on WildFly/Quarkus. Single deployable
  server with embedded admin console.
- **Language:** Java (Kotlin in some newer modules)
- **Key features:** OIDC certified, SAML 2.0 IdP/SP, organizations (added
  in v25+), user federation (LDAP/AD), fine-grained admin permissions,
  themeable login pages. SCIM 2.0 is available via community extensions,
  not in core.
- **Community:** ~25K+ GitHub stars. Backed by Red Hat (IBM). Hundreds of
  contributors. The de facto enterprise open-source IAM.
- **Differentiator:** Most battle-tested. Decades of enterprise deployments.
  Red Hat support available.
- **Weakness:** Monolithic architecture — difficult to scale individual
  functions independently. Java startup/memory footprint. No native
  multi-tenancy (realms provide logical separation, not physical isolation).
  SCIM is community-maintained, not core.

### 2.2 Authentik 2024.x

- **Repo:** https://github.com/goauthentik/authentik
- **Docs:** https://docs.goauthentik.io/
- **Architecture:** Python (Django) core with Go components (outpost proxy
  agent). Runs as server + worker + outpost pods.
- **Language:** Python (Django), Go (outpost)
- **Key features:** Visual flow builder for custom auth pipelines, LDAP/
  Active Directory, SAML 2.0, OAuth2/OIDC, RBAC, outpost reverse proxy
  for app-protected access, multi-tenancy via tenants feature.
- **Community:** ~15K+ stars. Community-driven with BeryJu as lead.
  Growing enterprise adoption.
- **Differentiator:** Visual flow builder — most flexible auth pipeline
  engine in open source. No-code customization for complex auth flows.
- **Weakness:** Complex setup (multiple components: server, worker, Redis,
  PostgreSQL). Python performance for high-throughput scenarios. Outpost
  adds operational complexity. No gRPC API (REST only).

### 2.3 Zitadel v2

- **Repo:** https://github.com/zitadel/zitadel
- **Docs:** https://zitadel.com/docs
- **Architecture:** Go, cloud-native single binary. CockroachDB or
  PostgreSQL backend.
- **Language:** Go
- **Key features:** OIDC certified, SAML 2.0 (added 2024), SCIM 2.0,
  passkeys/WebAuthn, actions (JavaScript hooks), three-level hierarchy
  (instance → organization → project), built-in multi-tenancy with
  project-level isolation.
- **Community:** ~10–11K stars. VC-backed (Zitadel GmbH). Commercial
  cloud offering available.
- **Differentiator:** Best-in-class multi-tenancy model (instance/org/
  project). Go-based, cloud-native. Now covers OIDC + SAML + SCIM.
- **Weakness:** Complex data model with steep learning curve. Single
  binary (not true microservices — harder to scale individual concerns).
  SAML support is relatively new and may have edge cases.

### 2.4 Casdoor

- **Repo:** https://github.com/casdoor/casdoor
- **Docs:** https://casdoor.org/
- **Architecture:** Go + Beego framework. Single monolithic binary.
- **Language:** Go
- **Key features:** OIDC, SAML 2.0, OAuth2, CAS protocol, 40+ social
  login providers, Casbin-based RBAC/ABAC, organization model for
  multi-tenancy, repositioned in 2025 as "Agent-first Identity."
- **Community:** ~11K+ stars. Backed by Casbin (also maintains the Casbin
  authorization library). Strong adoption in China and APAC.
- **Differentiator:** Broadest social-login support (40+ IdPs). Casbin
  integration for policy. Agent/AI-auth positioning is unique.
- **Weakness:** Monolithic — no microservices option. Beego framework is
  less mainstream than Gin/Echo. Limited enterprise-grade features
  (audit, compliance, federation). API documentation gaps.

### 2.5 Logto

- **Repo:** https://github.com/logto-io/logto
- **Docs:** https://docs.logto.io/
- **Architecture:** TypeScript/Node.js monolith.
- **Language:** TypeScript (Node.js)
- **Key features:** OIDC/OAuth 2.1, social connectors (SDK framework),
  organizations, MFA, RBAC. Now claims multi-tenancy support. Developer-
  experience focused with SDKs for major frameworks.
- **Community:** ~10K+ stars (gained ~2,000 in 2025). VC-backed. Cloud
  offering with SOC 2 Type II. 2M+ identities in Logto Cloud.
- **Differentiator:** Best developer experience. Connector SDK model
  makes social login extension easy. TypeScript-native for JS/TS shops.
- **Weakness:** TypeScript/Node.js performance ceiling. Single binary,
  not microservices. Multi-tenancy is newer and less proven than
  Zitadel's. No gRPC API. No SAML IdP in OSS.

### 2.6 Hanko

- **Repo:** https://github.com/teamhanko/hanko
- **Docs:** https://www.hanko.io/docs
- **Architecture:** Go backend + TypeScript frontend SDK.
- **Language:** Go (backend), TypeScript (SDK)
- **Key features:** Originally passkey-first; now expanded to passwords,
  MFA (TOTP + security keys), social logins, and SAML 2.0 SSO.
  WebAuthn-first architecture. Passkey management UI.
- **Community:** ~7K+ stars. VC-backed (Hanko GmbH). Cloud offering
  available.
- **Differentiator:** WebAuthn-native architecture with the best passkey
  UX in open source. Simple integration, framework-agnostic.
- **Weakness:** Narrower scope than full IAM suites — no SCIM, no audit
  logging, no policy engine. Limited enterprise features. No
  multi-tenancy in OSS.

### 2.7 Ory (Kratos + Hydra + Keto + Oathkeeper)

- **Repos:**
  - https://github.com/ory/kratos (identity)
  - https://github.com/ory/hydra (OAuth2/OIDC)
  - https://github.com/ory/keto (access control)
  - https://github.com/ory/oathkeeper (API gateway/proxy)
- **Docs:** https://www.ory.com/docs/
- **Architecture:** Four separate Go microservices, each independently
  deployable. Headless/API-first design.
- **Language:** Go
- **Key features:** Full stack — identity management (Kratos), OAuth2/OIDC
  certified (Hydra), access control/Zanzibar-style permissions (Keto),
  reverse proxy/PDP (Oathkeeper). Unified versioning (v25.4) across all
  services. Actions/webhooks.
- **Community:** ~50K combined stars (Kratos ~15K, Hydra ~15K, Keto ~5K,
  Oathkeeper ~3K+). VC-backed (Ory GmbH). Major enterprise traction.
- **Differentiator:** Most mature Go microservices IAM stack. API-first
  headless architecture. OAuth2/OIDC certification.
- **Weakness:** **OSS is single-tenant only** — multi-tenancy requires
  Ory Enterprise License or Ory Network (managed cloud). No built-in
  audit logging in OSS. Four-service complexity is operationally heavy.
  No unified admin UI in OSS.

---

## 3. Feature Matrix

```
                    Keycloak   Authentik  Zitadel   Casdoor   Logto    Hanko    Ory       GGID
                    ────────   ─────────  ────────  ─────────  ──────   ──────   ────────  ────────
Language            Java       Python/Go  Go        Go        TS/Node  Go/TS    Go        Go
Architecture        Monolith   Mono+Wrkr  Single    Monolith  Mono     Single   4 Micro   7 Micro
Multi-tenancy       Realm      Tenants    Native    Orgs      New      None     Ent-only  Native+RLS
OIDC certified      Yes        Yes        Yes       No        No       No       Yes       In-prog
SAML 2.0            Yes        Yes        Yes       Yes       No       Yes      Limited  Skeleton
SCIM 2.0            Extension  No         Yes       No        No       No       No        Skeleton
WebAuthn/Passkey    Yes        Yes        Yes       Yes       Yes      Native   Yes       Yes
MFA (TOTP)          Yes        Yes        Yes       Yes       Yes      Yes      Yes       Yes
MFA (SMS)           Yes        Yes        Yes       Yes       No       No       No        Planned
RBAC                Yes        Yes        Yes       Casbin    Yes      No       Keto      Yes
ABAC                Limited    No         Actions    Casbin   No       No       Keto      Yes
Audit logging       Events     Audit      Audit      No       No       No       No        NATS+REST
gRPC API            No         No         Yes        No        No       No       Yes       Yes
SDK languages       Java       Python     JS/Go     10+       7+       JS       7+       Go/Node/Java
License             Apache-2   MIT        Apache-2  Apache-2  MPL-2.0  EUPL    Apache-2  Apache-2
Docker/K8s          Yes        Yes        Yes       Yes       Yes      Yes      Yes       Yes
GitHub stars        ~25K       ~15K       ~11K      ~11K      ~10K     ~7K      ~50K*     New
```

> *Ory ~50K combined across Kratos + Hydra + Keto + Oathkeeper.

**Notable gaps across the landscape:**
- **SCIM 2.0:** Only Zitadel supports it in core. Keycloak via extension.
  Everyone else — none. This is a major enterprise B2B gap.
- **Audit logging:** Only Keycloak, Authentik, Zitadel, and GGID have it.
  Ory, Casdoor, Logto, and Hanko lack built-in audit trails in OSS.
- **gRPC API:** Only Zitadel, Ory, and GGID expose gRPC. This matters for
  internal service-to-service communication and performance.
- **Native multi-tenancy with data isolation:** Only GGID (RLS) and
  Zitadel (hierarchical model) provide this in OSS. Ory requires
  enterprise license.

---

## 4. Community Growth

### GitHub Stars Trend (approximate, 2023 → 2025)

```
Project     2023      2024      2025      Trend
───────     ────      ────      ────      ────
Keycloak    ~20K      ~23K      ~25K      Slowing (mature)
Ory(all)    ~40K      ~45K      ~50K      Steady (4 repos)
Authentik   ~10K      ~13K      ~15K      Growing
Casdoor     ~8K       ~10K      ~11K      Steady
Zitadel     ~7K       ~9K       ~11K      Accelerating
Logto       ~5K       ~8K       ~10K      Fastest growth %
Hanko       ~5K       ~6K       ~7K       Slowing
GGID        —         —         New       —
```

### Contributor Count & Activity

- **Keycloak:** 500+ contributors. Red Hat employees + community. Most
  active enterprise IAM. Frequent releases (monthly+).
- **Ory:** 400+ contributors across 4 repos. VC-backed with paid team.
  Regular releases, unified versioning (v25.4).
- **Authentik:** 200+ contributors. Community-led with enterprise sponsors.
- **Zitadel:** 150+ contributors. VC-backed team. Regular releases.
- **Casdoor:** 100+ contributors. Strong APAC community. Casbin ecosystem.
- **Logto:** 80+ contributors. VC-backed, growing rapidly.
- **Hanko:** 50+ contributors. Small but focused team.

### Backing Model

| Project   | Model                                | Sustainability          |
|-----------|--------------------------------------|-------------------------|
| Keycloak  | Corporate (Red Hat/IBM)              | Very high               |
| Ory       | VC-backed (Ory GmbH)                 | High (funded)           |
| Zitadel   | VC-backed (Zitadel GmbH)             | High (funded)           |
| Authentik | Community + enterprise sponsors      | Medium-high             |
| Logto     | VC-backed                            | Medium-high (early)     |
| Casdoor   | Community (Casbin org)               | Medium                  |
| Hanko     | VC-backed (Hanko GmbH)               | Medium (early)          |
| GGID      | Community (Apache-2.0)               | Building                |

---

## 5. Where GGID Fits

### GGID's Unique Position

GGID combines four properties that no other open-source IAM project offers
together:

1. **Go microservices** — 7 independently deployable services (gateway,
   identity, auth, oauth, policy, org, audit). Only Ory also uses a
   microservices architecture, but with 4 services and without built-in
   multi-tenancy in OSS.
2. **Native multi-tenancy via PostgreSQL RLS** — Row-Level Security
   provides database-level tenant isolation. No other project uses RLS
   for tenant isolation. Zitadel has a logical hierarchy model; Ory's
   OSS doesn't support multi-tenancy at all.
3. **Event-driven architecture (NATS JetStream)** — Built-in audit event
   streaming via NATS. No other IAM project ships with event streaming
   as a core architectural component. This enables real-time audit
   pipelines, event-sourced identity, and reactive security workflows.
4. **Apache 2.0 license** — Permissive, business-friendly. Some
   competitors use copyleft (Logto: MPL-2.0, Hanko: EUPL) which creates
   adoption friction.

### Closest Competitors

**Zitadel** is the closest competitor: Go-based, multi-tenant, growing
rapidly. Key differences:
- *GGID advantages:* True microservices (Zitadel is single binary), RLS-
  based tenant isolation (Zitadel uses logical separation), NATS event
  streaming (Zitadel has none), RBAC + ABAC policy engine (Zitadel uses
  JS actions), Apache-2.0 license (Zitadel is also Apache-2.0).
- *GGID gaps:* Zitadel is OIDC certified (GGID in progress), has more
  polished admin UI, SCIM 2.0 in core (GGID skeleton only), SAML in core
  (GGID skeleton only), and has a commercial cloud offering.

**Ory** is the other close competitor: Go microservices, mature ecosystem.
- *GGID advantages:* Built-in multi-tenancy with RLS (Ory OSS is single-
  tenant only — multi-tenancy requires enterprise license), built-in
  audit logging (Ory has none in OSS), unified deployment (7 services vs
  Ory's 4 services with different APIs/versioning), RBAC+ABAC engine
  (Ory requires separate Keto setup).
- *GGID gaps:* Ory is OIDC certified, has 50K+ combined stars, massive
  community, enterprise traction, and a managed cloud (Ory Network).

### Target Market

GGID targets **mid-to-large enterprises and SaaS platforms** that need:
- Multi-tenant identity with strict data isolation (B2B SaaS, MSPs)
- Go-native architecture for integration with Go microservice ecosystems
- Event-driven audit/compliance pipelines (finance, healthcare, gov)
- Permissive licensing for commercial embedding (Apache-2.0)
- Individual service scalability (not possible with monoliths)

---

## 6. Strategic Recommendations

### 5 Specific Actions

1. **Ship OIDC certification** — This is the #1 trust signal for
   enterprise buyers. Zitadel, Ory, and Keycloak all have it. GGID
   should prioritize passing the OpenID Conformance Suite. Without it,
   enterprises will not evaluate further.

2. **Complete SCIM 2.0 and SAML** — Both are skeleton/partial in GGID.
   Zitadel added both in 2024–2025 and it differentiated them. SCIM is
   critical for B2B enterprise customer provisioning (Okta, Entra ID).
   SAML is required for legacy enterprise federation. These are table
   stakes for the enterprise market GGID targets.

3. **Position NATS event streaming as the differentiator** — No
   competitor offers built-in event-driven identity. Build reference
   integrations: real-time audit dashboards, event-sourced user
   lifecycles, reactive security automation. This is GGID's most
   defensible unique feature and should be front-and-center in marketing.

4. **Polish the admin console for enterprise** — Zitadel and Keycloak
   win evaluations partly on admin UX. GGID's Next.js console needs to
   be feature-complete, professional, and demo-ready. Admin experience
   is often the deciding factor in procurement.

5. **Build a multi-tenant reference architecture** — Publish a
   production-ready K8s deployment guide with RLS-verified tenant
   isolation, NATS streaming, and per-tenant configuration. Make it
   trivial for a platform team to evaluate GGID for B2B SaaS. This is
   where GGID is technically superior to every competitor — make it
   visible.

### What GGID Should NOT Compete On

- **Social login breadth** — Casdoor has 40+ IdPs. Don't chase this.
  Support the top 10 and move on.
- **Visual flow builder** — Authentik owns this niche. The complexity
   is not worth the engineering investment for GGID's target market.
- **Developer-experience SDKs** — Logto wins here. GGID should provide
  competent SDKs but not try to match Logto's connector ecosystem.
- **Passkey-only positioning** — Hanko covers this. GGID should offer
  strong WebAuthn support as one of many features, not as identity.

### Where to Differentiate

1. **Multi-tenancy (RLS-based)** — GGID's technical advantage. No
   competitor offers database-level tenant isolation in OSS.
2. **Event-driven architecture (NATS)** — Unique architectural choice
   that enables use cases no competitor can match.
3. **True microservices** — Only GGID and Ory use microservices. GGID's
   7-service split is more granular and individually scalable.
4. **B2B enterprise focus** — Multi-tenancy + SCIM + SAML + audit =
   B2B SaaS platform identity. This is the most underserved market
   segment in open-source IAM.

---

## References

- Keycloak: https://github.com/keycloak/keycloak
- Authentik: https://github.com/goauthentik/authentik
- Zitadel: https://github.com/zitadel/zitadel
- Casdoor: https://github.com/casdoor/casdoor
- Logto: https://github.com/logto-io/logto
- Hanko: https://github.com/teamhanko/hanko
- Ory Kratos: https://github.com/ory/kratos
- Ory Hydra: https://github.com/ory/hydra
- Ory Keto: https://github.com/ory/keto
- Ory Oathkeeper: https://github.com/ory/oathkeeper
- OIDC Certification: https://openid.net/certification/
- Star History: https://www.star-history.com/
- IAM Comparison (community): https://github.com/generalistcodes/iam-comparison
