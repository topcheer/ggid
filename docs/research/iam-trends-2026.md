# IAM Trends 2026: Research-Driven Backlog Generation

## Research Date: 2026-07-12

## Sources
- CDW: 5 IAM Trends to Watch in 2026
- Clarity Security: 5 IAM Trends to Watch in 2026
- CSA: Agentic AI Identity and Access Management — A New Approach
- Coalition for Secure AI: Agentic Identity and Access Control
- Strata.io: Agentic AI Governance
- Cloud Security Alliance: Non-Human Identity Governance Vacuum
- OWASP Top 10 for Agentic Applications 2026
- Pointsharp: IAM in 2026
- Okta: Future of CIAM
- Radiant Logic: Non-Human Identities Predictions 2026

---

## Trend 1: Agentic AI Identity Governance (P0 Competitive Gap)

### Summary
AI agents are autonomous software systems that perceive context, reason over goals, and take actions. Traditional IAM cannot address agentic AI risks: privilege drift, shadow agents, MCP bypass, and broken delegation chains.

### Key Findings
- 78% of organizations have no documented policy for creating or removing AI identities (NHI Reality Report 2026)
- OWASP Top 10 for Agentic Applications 2026 places identity as the #1 risk category
- CSA proposes a new IAM framework using Decentralized Identifiers (DIDs) for multi-agent AI systems
- Seven core requirements: unique agent identities, proactive identity protection, governance models for autonomous vs supervised agents, delegation chain tracking, tool access scoping, runtime access reviews, auditability

### GGID Current State
- AI Agent Identity already implemented (services/oauth/internal/service/agent_identity.go)
- Agent token claims with delegation_chain, mcp_servers, max_delegation_depth
- Go SDK agent methods: RegisterAgent, ListAgents, ExchangeAgentToken, VerifyAgentToken
- Console AI Agents page exists

### Gaps to Close
1. **Agent privilege drift detection** — monitor agent scope changes over time, alert on expansion
2. **Shadow agent discovery** — scan for unregistered agents using tokens
3. **Agent access review workflow** — periodic certification of agent permissions
4. **MCP tool access scoping** — per-agent allowed tool list with runtime enforcement
5. **Agent delegation chain visualization** — console UI showing delegation depth and chain
6. **Agent runtime anomaly detection** — unusual tool access patterns, geographic anomalies

### Backlog Items (Implementation)
- [ ] Backend: Agent privilege drift detector (services/oauth/internal/service/agent_drift.go)
- [ ] Backend: Shadow agent scanner (services/oauth/internal/service/shadow_scanner.go)
- [ ] Backend: Agent access review CRUD (services/oauth/internal/server/agent_review_handler.go)
- [ ] Frontend: Agent delegation chain visualization (console/src/app/agents/delegation-graph/page.tsx)
- [ ] Frontend: Agent access review page (console/src/app/settings/agent-access-review/page.tsx)
- [ ] SDK: GetAgentAccessReview, SubmitAgentReview methods
- [ ] Docs: Agentic AI identity governance guide (docs/guides/agentic-ai-governance.md)

---

## Trend 2: Non-Human Identity (NHI) Lifecycle Management (P1 Competitive Gap)

### Summary
Non-human identities (service accounts, API keys, AI agents, IoT devices) are a growing security risk. 78% of organizations lack NHI governance. Traditional IAM excludes NHIs from lifecycle management.

### Key Findings
- NHI explosion: 10:1 ratio of non-human to human identities in modern orgs
- Only 8% of organizations have automated NHI lifecycle management
- Key concerns: hardcoded credentials, never-expiring tokens, orphaned service accounts
- Required: provisioning, rotation, decommissioning with continuous monitoring
- NHI security tools: secrets detection, lifecycle governance, machine identity management, vault extensions

### GGID Current State
- API key management exists
- OAuth client registration exists
- Agent identity exists
- No unified NHI inventory or lifecycle management

### Gaps to Close
1. **NHI inventory dashboard** — unified view of all non-human identities (API keys, agents, service accounts, OAuth clients)
2. **NHI lifecycle automation** — auto-expiry, rotation reminders, orphan detection
3. **Credential rotation scheduler** — policy-driven rotation with notification
4. **NHI access review** — periodic certification of NHI permissions
5. **NHI risk scoring** — score based on age, scope breadth, last-used, privilege level

### Backlog Items (Implementation)
- [ ] Backend: NHI inventory endpoint (services/identity/internal/server/nhi_inventory_handler.go)
- [ ] Backend: NHI lifecycle automation (services/identity/internal/service/nhi_lifecycle.go)
- [ ] Backend: Credential rotation scheduler (services/auth/internal/service/rotation_scheduler.go)
- [ ] Frontend: NHI inventory dashboard (console/src/app/settings/nhi-inventory/page.tsx)
- [ ] Frontend: Credential rotation config (console/src/app/settings/credential-rotation/page.tsx)
- [ ] SDK: ListNHI, GetNHIDetails, RotateNHI, DecommissionNHI methods
- [ ] Docs: NHI lifecycle management guide (docs/guides/nhi-lifecycle-management.md)

---

## Trend 3: Decentralized Identity & Verifiable Credentials (P2 Competitive Gap)

### Summary
Decentralized identity using DIDs and Verifiable Credentials (VCs) is gaining traction. CSA proposes DIDs for multi-agent AI systems. Passwordless + self-sovereign identity (SSI) adoption accelerating.

### Key Findings
- DID-based agent identity proposed by CSA
- Verifiable Credentials for cross-organization trust
- W3C VC Data Model 2.0 in development
- Zero-knowledge proofs for privacy-preserving attribute verification

### GGID Current State
- No DID support
- No Verifiable Credentials issuance/verification
- WebAuthn exists for passwordless

### Gaps to Close
1. **DID resolution** — resolve DIDs to DID documents
2. **VC issuance** — issue verifiable credentials for user attributes
3. **VC verification** — verify VCs from external issuers
4. **ZKP identity proof** — zero-knowledge proof of attributes without disclosure

### Backlog Items (Research → Implementation)
- [x] Research: ZKP identity (docs/research/ — already exists)
- [ ] Backend: DID resolver (services/identity/internal/service/did_resolver.go)
- [ ] Backend: VC issuer (services/identity/internal/service/vc_issuer.go)
- [ ] Frontend: VC management page (console/src/app/settings/verifiable-credentials/page.tsx)
- [ ] Docs: Decentralized identity guide (docs/guides/decentralized-identity.md)

---

## Trend 4: Cyber Resilience Act (CRA) Compliance (P2 Regulatory)

### Summary
EU Cyber Resilience Act taking effect — requires security-by-design, vulnerability disclosure, lifecycle management for digital products. IAM systems must demonstrate compliance.

### Key Findings
- CRA mandates security by design and default
- Vulnerability and incident reporting requirements
- SBOM (Software Bill of Materials) required
- Digital product lifecycle security obligations

### GGID Current State
- Security headers, CSRF, rate limiting implemented
- Audit trail with hash chain
- No SBOM generation
- No CRA-specific compliance reporting

### Gaps to Close
1. **SBOM generation** — CycloneDX/Syft integration for dependency inventory
2. **CRA compliance report** — security-by-design checklist, vulnerability disclosure policy
3. **Vulnerability disclosure workflow** — security.txt, CVE tracking

### Backlog Items
- [ ] Docs: CRA compliance guide (docs/guides/cra-compliance.md)
- [ ] Backend: SBOM endpoint (services/audit/internal/server/sbom_handler.go)
- [ ] Frontend: Compliance dashboard with CRA section

---

## Trend 5: Passwordless Acceleration & Passkeys (P1 Competitive)

### Summary
Passwords finally going extinct per multiple predictions. Passkeys (synced WebAuthn credentials) becoming default. Platform authenticators (Apple/Google/Microsoft) syncing passkeys across devices.

### Key Findings
- Passkey adoption doubling year-over-year
- FIDO Alliance reporting 10B+ passkey-enabled accounts
- Synced credentials solving the multi-device problem
- Conditional UI (autofill) improving UX

### GGID Current State
- WebAuthn implemented (registration + authentication)
- TOTP MFA implemented
- No passkey-specific UI or sync support

### Gaps to Close
1. **Passkey enrollment flow** — simplified registration with platform authenticator
2. **Passkey management UI** — list, rename, delete passkeys per device
3. **Passkey recovery** — backup/recovery via additional device or QR code
4. **Passwordless migration** — progressive enhancement from password → TOTP → passkey

### Backlog Items
- [ ] Frontend: Passkey management page (console/src/app/settings/passkeys/page.tsx)
- [ ] Backend: Passkey enrollment endpoint (services/auth/internal/server/passkey_handler.go)
- [ ] Docs: Passkey deployment guide (docs/guides/passkey-deployment.md)

---

## Priority Ranking for Backlog Dispatch

| Priority | Trend | Effort | Impact |
|----------|-------|--------|--------|
| P0 | Agentic AI Governance | High | Critical — competitive differentiator |
| P1 | NHI Lifecycle Management | Medium | High — addresses #1 industry gap |
| P1 | Passwordless/Passkeys | Medium | High — user demand |
| P2 | Decentralized Identity | High | Medium — emerging, not yet mainstream |
| P2 | CRA Compliance | Low | Medium — regulatory, EU-specific |

## Recommended Next Dispatch

### Backend (5 tasks)
1. Agent privilege drift detector
2. Shadow agent scanner
3. NHI inventory endpoint
4. Credential rotation scheduler
5. Passkey enrollment endpoint

### Frontend (5 tasks)
1. Agent delegation chain visualization
2. Agent access review page
3. NHI inventory dashboard
4. Passkey management page
5. Credential rotation config

### Docs (3 tasks)
1. Agentic AI governance guide
2. NHI lifecycle management guide
3. Passkey deployment guide