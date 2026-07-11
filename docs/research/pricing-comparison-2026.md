# Pricing Comparison 2026

> TCO analysis: GGID (open source) vs Auth0/Okta/Clerk (SaaS).

---

## SaaS Pricing (2026)

| Provider | Free Tier | Entry | Growth | Enterprise |
|----------|-----------|-------|--------|------------|
| Auth0 | 7.5k MAU | $35/mo (1k MAU) | $240/mo (5k MAU) | Custom ($1+/MAU) |
| Okta WIC | — | $2/user/mo | $6/user/mo | $15+/user/mo |
| Clerk | 10k MAU | $25/mo (1k MAU) | $99/mo (5k MAU) | Custom |
| Keycloak | Free (self-host) | — | — | — |
| **GGID** | **Free (self-host)** | **Infra cost** | **Infra cost** | **Infra + support** |

---

## TCO Comparison (10,000 MAU)

| Cost Item | Auth0 Growth | Clerk Growth | GGID Self-Host |
|-----------|-------------|-------------|----------------|
| License/mo | $480 (est.) | $198 | $0 |
| Infrastructure/mo | $0 | $0 | $150 (2 vCPU + 4GB) |
| DB + Redis + NATS | $0 | $0 | $100 (RDS + ElastiCache) |
| DevOps time/mo | $0 | $0 | $500 (0.25 FTE) |
| **Monthly total** | **$480** | **$198** | **$750** |
| **Annual total** | **$5,760** | **$2,376** | **$9,000** |

> Note: GGID TCO includes full control, no vendor lock-in, and no per-user pricing. At 100k MAU, SaaS costs scale linearly while GGID stays ~$750/mo.

---

## TCO at Scale (100,000 MAU)

| Provider | Monthly Cost |
|----------|-------------|
| Auth0 Enterprise | ~$2,000-$5,000 |
| Okta WIC | ~$6,000 |
| Clerk | ~$1,990 |
| GGID | ~$1,200 (larger instances) |

At 100k MAU, GGID is cheaper than all SaaS options.

---

## GGID Cost Breakdown

| Component | Cloud Cost (AWS/mo) |
|-----------|-------------------|
| EKS node (2 vCPU, 8GB) | $70 |
| RDS PostgreSQL (db.t4g.medium) | $60 |
| ElastiCache Redis (cache.t4g.small) | $25 |
| NATS (on EKS, shared) | $0 |
| Load Balancer | $20 |
| S3 backups | $5 |
| **Total** | **~$180/mo** |

---

## When GGID Wins

- >1,000 MAU (SaaS per-user pricing exceeds infra cost)
- Self-hosted requirement (compliance, data residency)
- Need ABAC/SCIM (not available in lower SaaS tiers)
- Multi-tenant (SaaS multi-tenant is expensive)

---

*See: [Competitive Analysis](competitive-analysis.md) | [Feature Matrix](../feature-matrix.md)*

*Last updated: 2025-07-11*
