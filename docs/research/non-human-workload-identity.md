# Non-Human Identity & Workload Identity — GGID Gap Analysis

*Research date: 2026-07-12*

## Overview

Gartner and Forrester identify **non-human identity management** as a top 2025 IAM trend. The expansion of:
- Microservices and service-to-service auth
- CI/CD pipelines needing secrets
- API integrations (machine-to-machine)
- AI agents acting on behalf of users

...creates massive growth in non-human identities that traditional IAM tools weren't designed for.

CSA (Cloud Security Alliance) published "Agentic AI Identity & Access Management" framework highlighting:
- Decentralized Identifiers (DIDs) for agents
- Zero Trust for multi-agent systems
- Delegation chains for agent-to-agent interactions

## GGID Current State: PARTIAL

**Implemented:**
- [x] AI Agent Identity (`services/oauth/internal/service/agent_identity.go`) — registration, token exchange, verification
- [x] OAuth client credentials grant — machine-to-machine auth
- [x] Token exchange (RFC 8693) — delegation
- [x] DPoP proof verification — token binding

**Missing:**
- [ ] Workload identity federation (SPIFFE/SPIRE integration)
- [ ] Service account lifecycle management (rotation, expiry, revocation tracking)
- [ ] Machine identity inventory dashboard
- [ ] Secretless authentication (cloud metadata server, IRSA, workload identity)
- [ ] Agent-to-agent delegation policies
- [ ] Rate limiting per workload vs per user
- [ ] Attestation-based auth (TPM, SGX, SEV-SNP)

## Gap Analysis

### P1: Workload Identity Federation (Backend)
- Support SPIFFE SVID verification
- Allow external workload identity tokens (GCP/AWS/Azure metadata)
- Map workload identity → GGID service account
- `POST /api/v1/workloads/exchange-token` endpoint

### P1: Service Account Lifecycle (Backend + Frontend)
- Dedicated service account entity (not just OAuth client)
- Automatic token rotation policy (max_age, rotation_interval)
- Revocation tracking with audit trail
- Console: Service accounts management page

### P2: Machine Identity Inventory (Frontend)
- Dashboard showing all non-human identities
- Categorize: OAuth clients, service accounts, AI agents, workload identities
- Show: last_active, token_count, scopes, risk_score
- Alert on dormant accounts

### P2: Agent Delegation Policies (Backend)
- Extend `agent_identity.go` with delegation policy engine
- Policy: which agents can delegate to which other agents
- Max delegation depth per policy
- Audit trail for full delegation chain

## Competitive Landscape
- Auth0: Machine-to-machine auth via client credentials, no workload federation
- Okta: API tokens + OAuth client credentials, no SPIFFE
- AWS IAM: IRSA + instance profiles, deeply integrated but AWS-only
- HashiCorp Vault: Dynamic secrets + workload identity, but not full IAM

## Backlog Items Generated
- [ ] **P1** Backend: Workload identity federation (SPIFFE/SPIRE) (services/)
- [ ] **P1** Backend: Service account lifecycle + rotation (services/)
- [ ] **P1** Frontend: Service accounts management page (console/src/)
- [ ] **P2** Frontend: Machine identity inventory dashboard (console/src/)
- [ ] **P2** Backend: Agent-to-agent delegation policies (services/)
- [ ] **P2** SDK: Workload identity exchange helpers (sdk/)
