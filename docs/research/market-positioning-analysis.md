# GGID Market Positioning Analysis: Early Adopters, Community, and Launch

> **Document Type**: Market Analysis & Launch Strategy
> **Scope**: Early adopter identification, community targeting, developer advocacy,
> and a concrete 12-month community-building plan
> **Companion Document**: `iam-differentiation-strategy.md` (high-level positioning,
> SWOT, competitive feature matrices, go-to-market strategy)
> **Date**: January 2025
> **Classification**: Strategic — Internal

---

## Executive Summary

This document complements the existing Differentiation Strategy by focusing on
the *operational* question: **how do we get GGID into the hands of the right
early adopters, and how do we build a self-sustaining open-source community
around it?**

The Differentiation Strategy answers "what makes GGID different." This document
answers "who cares, where do they live, and how do we reach them."

The thesis: GGID's first thousand users will not come from enterprise procurement
or marketing campaigns. They will come from **Go developers who are tired of
Java-heavy IAM stacks**, **DevOps teams who want a single static binary instead
of a JVM**, and **security-conscious startups who want source-level transparency
without Keycloak's operational burden.** We win by being technically excellent
where it matters — performance, deployability, security transparency — and by
being present in the communities where these developers already congregate.

---

## Table of Contents

1. [Who Adopts Go-Native IAM First?](#1-who-adopts-go-native-iam-first)
2. [Target Industries](#2-target-industries)
3. [Open-Source Community Targeting](#3-open-source-community-targeting)
4. [HackerNews Launch Strategy](#4-hackernews-launch-strategy)
5. [Reddit Strategy](#5-reddit-strategy)
6. [GitHub Strategy](#6-github-strategy)
7. [Content Marketing](#7-content-marketing)
8. [Developer Advocacy](#8-developer-advocacy)
9. [Competitor Response Playbook](#9-competitor-response-playbook)
10. [12-Month Community Building Plan](#12-month-community-building-plan)

---

## 1. Who Adopts Go-Native IAM First?

### 1.1 The Early Adopter Profile

Early adopters of a new IAM system share three characteristics:

1. **They have a painful problem right now** — not a future problem.
2. **They are willing to accept rough edges** in exchange for a core benefit.
3. **They have influence** — they write blog posts, give talks, and recommend
   tools to peers.

For GGID, these early adopters fall into four overlapping personas.

### 1.2 Persona A: The Go-Native Backend Engineer

**Who**: Backend engineers at startups and mid-size companies who have already
standardized on Go for their API layer, microservices, or infrastructure tooling.
They may be using a Go framework (Gin, Echo, Fiber, chi) and are comfortable
with Go's tooling ecosystem.

**Current pain**: They need authentication and authorization. Their options are:
- **Auth0/Clerk**: Hosted, easy to start, but adds a network dependency, costs
  scale with users, and they can't audit the internals.
- **Keycloak**: Feature-complete but it's Java/Quarkus — a 400MB container, a JVM
  to tune, and a paradigm mismatch with their Go stack.
- **Build from scratch**: They've done this. It's a 2000-line JWT middleware
  with TODO comments everywhere. It works until it doesn't.

**Why GGID wins**: It's Go. They can `go get` the SDK, read the source, and
deploy a single static binary alongside their existing services. The gRPC API
matches their service mesh. The memory footprint is 20-30MB, not 400MB. They
can contribute a PR without learning Java/Maven/Quarkus.

**Trigger event**: They're building a new product or migrating off a managed
service that got too expensive. They search "golang authentication" or "go IAM"
and find GGID.

**Estimated pool**: Go is the #4 most-used language on GitHub (after JS, Python,
Java). There are an estimated 2.7M professional Go developers worldwide. Even
capturing 0.1% who need IAM gives us 2,700 potential early users.

### 1.3 Persona B: The Kubernetes DevOps Engineer

**Who**: Platform engineers and SREs who manage Kubernetes clusters and want
self-hosted infrastructure they can control. They value: small container images,
fast cold starts, no JVM tuning, Helm charts, and GitOps-friendly deployments.

**Current pain**: Keycloak on Kubernetes is a well-known pain point. It needs
a JVM, persistent volumes, careful resource limits, and the container image is
400MB+. Auth0 is cloud-only — no airgapped deployment, no data sovereignty.

**Why GGID wins**:
- **Container image**: ~20-35MB per service (vs 400MB+ for Keycloak). Fits in a
  64MB memory limit.
- **Cold start**: Sub-second (vs 10-30s for JVM warmup).
- **Helm chart**: Deploy with one command. No JVM heap tuning.
- **Data sovereignty**: Self-hosted, airgap-capable. Relevant for EU and gov.
- **Microservice architecture**: Each service scales independently. Gateway,
  auth, policy, org, audit can each have their own HorizontalPodAutoscaler.

**Trigger event**: They're setting up a new cluster, or migrating Keycloak
to something lighter. They search "keycloak alternative kubernetes" or
"lightweight self-hosted IAM."

**Estimated pool**: There are ~7M Kubernetes clusters in production (CNCF
estimate). Even 0.01% needing a new IAM solution gives 700 clusters.

### 1.4 Persona C: The Performance-Obsessed Startup CTO

**Who**: CTOs and founding engineers at early-stage startups (seed to Series B)
who chose Go for performance and want their entire stack to be fast. They're in
fintech, gaming, real-time, or edge computing.

**Current pain**: Their auth layer is either:
- **Firebase Auth**: Works but they outgrew it. Vendor lock-in. Can't customize.
- **Auth0**: Getting expensive. The per-MAU pricing surprises them at scale.
- **Supabase Auth**: Go-based but limited to the Supabase ecosystem. Not a
  standalone IAM platform.

**Why GGID wins**:
- **Performance**: Go's compiled binary, goroutine concurrency, and low GC
  pressure mean sub-millisecond token validation. Benchmarks should show
  10-50x throughput vs Keycloak for token introspection.
- **Cost**: Free, self-hosted. No per-MAU pricing. Pay only for infrastructure.
- **Extensibility**: RBAC + ABAC policy engine that they can customize.
- **Multi-tenancy**: Built-in tenant isolation via PostgreSQL RLS. Essential
  for B2B SaaS.

**Trigger event**: They hit Auth0's pricing tier ceiling, or they're building
a new B2B SaaS and need multi-tenant IAM from day one.

### 1.5 Persona D: The Java-to-Go Migration Team

**Who**: Engineering teams at companies that have standardized on Java for years
but are selectively migrating critical services to Go for performance. They
already run Keycloak and are tired of its operational overhead.

**Current pain**: Keycloak's JVM tuning, memory consumption, and upgrade pain
(Quarkus migration was notoriously rough). They want to consolidate on Go.

**Why GGID wins**: Feature parity with the IAM essentials (OAuth 2.1, OIDC,
SAML, SCIM, RBAC/ABAC, MFA, WebAuthn) without the JVM tax. Migration guide
from Keycloak would be a killer asset.

**Trigger event**: A Keycloak upgrade breaks something. Or they're doing a
platform consolidation and want fewer languages/runtimes in their stack.

### 1.6 Why They'd Choose GGID Over Alternatives

| Decision Factor | Auth0/Okta | Keycloak | Ory | Clerk | **GGID** |
|----------------|-----------|----------|-----|-------|----------|
| Language ecosystem | JS/Hosted | Java | Go (partially) | JS/Hosted | **Go (native)** |
| Self-hosted | No | Yes | Yes | No | **Yes** |
| Container size | N/A | 400MB+ | ~50MB | N/A | **20-35MB** |
| Source transparency | No | Partial | Yes | No | **Yes (Apache 2.0)** |
| Per-MAU pricing | Yes | Free | Free/Cloud | Yes | **Free** |
| Microservice architecture | No | Monolith | Services | No | **7 services** |
| gRPC API | No | No | Partial | No | **Yes** |
| Multi-tenant RLS | No | Partial | No | No | **Yes** |
| Security research docs | No | No | Some | No | **145+ docs** |

---

## 2. Target Industries

### 2.1 Fintech and Trading

**Why GGID fits**: Go is the dominant language in fintech infrastructure.
Companies like Stripe, Square, Monzo, Revolut, and countless HFT firms use Go
for payment processing, trading engines, and API gateways. These teams already
have Go expertise and want their auth layer to match.

**Specific needs**:
- **High throughput**: Trading platforms need auth that doesn't add latency.
  Token validation must be sub-millisecond. GGID's compiled Go binary with
  in-process JWT verification delivers this.
- **Audit trails**: Financial regulations (PCI-DSS, SOX, MiFID II) require
  immutable audit logs. GGID's NATS JetStream audit pipeline with hash chaining
  is a strong fit.
- **Multi-tenancy**: B2B fintech platforms (payment processors, banking-as-a-
  service) need tenant isolation. GGID's PostgreSQL RLS-based multi-tenancy is
  purpose-built for this.
- **MFA and step-up auth**: Financial transactions require step-up
  authentication. GGID's TOTP + WebAuthn MFA pipeline supports this.

**Entry points**:
- **Open-source payment platforms**: Contribute auth to projects like Lago,
  Kill Bill, or Medusa (Go/JS e-commerce).
- **Fintech incubators**: YC fintech batch, Techstars fintech. Offer GGID as
  the auth layer for new fintech startups.
- **Go fintech meetups**: Speak at Go meetups in financial hubs (London, NYC,
  Singapore, Frankfurt).

**Target companies**: Monzo, Revolut, N26, Stripe (infrastructure teams),
Adyen, Wise, Checkout.com, Plaid, Marqeta.

### 2.2 Cloud Infrastructure and Kubernetes-Native

**Why GGID fits**: The CNCF ecosystem is Go-native. Kubernetes, Prometheus,
containerd, etcd, CoreDNS — all Go. Cloud infrastructure companies want auth
that fits this ecosystem, not a Java outlier.

**Specific needs**:
- **Kubernetes-native deployment**: Helm chart, operator, CRD-based config.
  GGID's microservice architecture maps naturally to K8s.
- **API gateway integration**: GGID's gateway service can sit behind ingress
  controllers or service meshes (Istio, Linkerd).
- **Service-to-service auth**: mTLS between services, JWT for external API.
  GGID's gRPC + JWT architecture supports this.
- **GitOps**: Declarative configuration, Git-backed tenant/role/policy management.

**Entry points**:
- **CNCF landscape**: Submit GGID to the CNCF landscape under "Identity" or
  "Security & Compliance." This is where K8s-native teams look for tools.
- **Kubernetes community**: K8s Slack (#security, #sig-auth), Kubernetes forums.
- **Service mesh integrations**: Write integration guides for Istio, Linkerd,
  Consul — showing how GGID provides the identity layer.

**Target companies**: HashiCorp, Grafana Labs, Datadog, Elastic, SUSE,
Rancher, DigitalOcean, Linode, Fly.io, Railway.

### 2.3 Telecom and 5G Edge

**Why GGID fits**: Telecom infrastructure is moving to cloud-native (Go-based)
deployments. 5G edge computing needs lightweight auth at the edge — not a JVM.
GGID's small binary, low memory footprint, and gRPC-first design fit edge
deployment scenarios.

**Specific needs**:
- **Edge deployment**: Auth at cell tower sites, CDN edge nodes, or regional
  data centers. GGID's 20MB binary can run on edge hardware.
- **High scale, low latency**: Millions of SIM/device authentications. GGID's
  Go concurrency model handles this efficiently.
- **Device identity**: OAuth 2.0 device flow (RFC 8628) for IoT/connected
  devices. GGID supports this.
- **Federation**: Inter-carrier identity federation via SAML/OIDC.

**Entry points**:
- **5G/edge consortia**: O-RAN Alliance, GSMA, Linux Foundation Networking.
- **Edge computing platforms**: Akamai EdgeWorkers, Cloudflare Workers,
  AWS Wavelength, Azure Edge Zones — integration guides.
- **IoT platforms**: Eclipse Hono, Losant, Particle — device auth use cases.

**Target companies**: Ericsson, Nokia, Mavenir, Meta (WhatsApp infra),
Telefonica, Deutsche Telekom, NTT, Reliance Jio.

### 2.4 Gaming and Real-Time

**Why GGID fits**: Gaming backends are increasingly Go-based (Agones, the Open
Match project). Game studios need auth that handles millions of concurrent
players, rapid login bursts, and low-latency token validation.

**Specific needs**:
- **Burst capacity**: Game launches create login spikes. GGID's Go goroutine
  model scales to handle burst traffic without JVM warmup delays.
- **Low latency**: Every millisecond matters for gaming. In-process JWT
  validation avoids network round-trips.
- **Social login**: Players expect Google/Apple/Discord/Steam login. GGID's
  social provider connectors cover these.
- **Anti-cheat**: Session validation, device fingerprinting, token binding
  to prevent account sharing.

**Entry points**:
- **Game backend platforms**: Agones (Google), Open Match, PlayFab alternatives.
- **Game developer communities**: r/gamedev, Game Developers Conference (GDC),
  indie game dev forums.
- **Game server hosting**: Unity Multiplay, Amazon GameLift — integration guides.

**Target companies**: Roblox, Epic Games, Supercell, Riot Games, Bungie,
Pocketpair, Playrix, Tencent Games.

### 2.5 Healthcare and Privacy-Conscious

**Why GGID fits**: Healthcare organizations need self-hosted, auditable auth
that they can run on-premises for HIPAA compliance. Cloud-only solutions (Auth0)
create data sovereignty concerns. Go's strong typing and the ability to audit
every line of source code appeals to security-conscious healthcare teams.

**Specific needs**:
- **HIPAA compliance**: Self-hosted deployment, audit logging, BAA requirements.
  GGID's self-hosted model and audit pipeline address this.
- **Patient consent management**: Fine-grained consent for data sharing.
  GGID's ABAC policy engine can encode consent rules.
- **FHIR integration**: SMART on FHIR (OAuth 2.0 for healthcare APIs).
  GGID's OAuth 2.1 implementation is close to this.
- **Multi-tenancy**: Hospital networks, clinics, insurance providers each need
  isolated identity domains.

**Entry points**:
- **FHIR/SMART on FHIR**: Write an integration guide for SMART App Launch
  (the healthcare OAuth profile).
- **Health tech startups**: Commure, Zus Health, 1upHealth — they need
  developer-friendly healthcare auth.
- **Open-source healthcare**: OpenMRS, LibreHealth — contribute auth modules.

**Target companies**: Commure, Zus Health, Cedar, 1upHealth, Redox, Elation
Health, Innovaccer, Olive AI.

---

## 3. Open-Source Community Targeting

### 3.1 Community Map

GGID's early community lives in these spaces. Each has different norms, content
expectations, and engagement strategies.

| Community | Size | Primary Content | Engagement Style |
|-----------|------|-----------------|------------------|
| r/golang | 400K+ | Code, libs, discussions | Technical, opinionated |
| Go Forum (forum.golangbridge.org) | 50K+ | Help, projects | Friendly, detailed |
| Kubernetes Slack | 200K+ | Real-time Q&A | Fast-paced, practical |
| CNCF Slack | 100K+ | Cloud-native projects | Strategic, ecosystem |
| GitHub | 100M+ | Code, stars, trending | Discovery, social proof |
| HackerNews | ~5M readers | Show HN, tech discussion | Skeptical, merit-based |
| Dev.to | 1.2M+ | Tutorials, deep dives | Beginner-friendly |
| Twitter/X (Golang) | ~200K | News, threads | Fast, networking |
| Discord (Go communities) | Various | Real-time chat | Casual, community |

### 3.2 Conference Circuit

Conferences are the highest-signal channel for reaching decision-makers and
influencers in the Go and cloud-native ecosystems.

**Tier 1 — Must Submit Talks**:

1. **GopherCon** (July, ~1,500 attendees)
   - The premier Go conference. A talk here reaches the most influential Go
     developers.
   - Talk ideas: "Building a Production IAM System in Go", "Multi-tenant
     Isolation with PostgreSQL RLS", "Threat Modeling a Go Authentication
     Service".
   - CFP typically opens January, closes February. Must submit early.

2. **KubeCon + CloudNativeCon** (NA in November, EU in April/May, ~12K attendees)
   - The CNCF flagship conference. Reaches platform engineers, SREs, and
     cloud-native decision-makers.
   - Talk ideas: "Identity at the Edge: Go-Native Auth for Kubernetes",
     "Self-Hosted IAM on K8s: A Keycloak Alternative".
   - CFP opens ~6 months before event.

3. **FOSDEM** (February, ~8K attendees, free)
   - European open-source conference. Strong security and infrastructure track.
   - Talk ideas: "Open-Source IAM: A Go-Native Approach to Identity".
   - Very competitive acceptance rate. Submit early (September).

4. **Open Source Summit EU / NA** (various, ~3K attendees)
   - Linux Foundation's flagship open-source conference.
   - Talk ideas: "The State of Open-Source IAM in 2025".

**Tier 2 — Regional and Specialized**:

5. **GopherCon UK** (August, ~800 attendees)
6. **GopherCon Israel / Singapore / India** (regional GopherCons)
7. **GoLab** (Florence, October, ~400 attendees)
8. **DEVintersection / DotGo** (Paris, various)
9. **BSides / Security conferences** (for security-focused talks)

**Strategy**: Aim for 1 Tier 1 talk in the first 12 months (GopherCon or
KubeCon). Submit to all four Tier 1 CFPs. Accept any Tier 2 speaking opportunity.

### 3.3 Blog Platforms

| Platform | Audience | Content Style | Why |
|----------|----------|--------------|-----|
| HackerNoon | 3M+ readers | Long-form, opinionated | Developer audience, good SEO |
| Dev.to | 1.2M+ devs | Tutorial, beginner-friendly | Broad developer reach |
| Medium (own pub) | Self-controlled | Technical deep dives | Owns the narrative |
| Substack | Email subscribers | Newsletter format | Direct audience ownership |
| Company blog | Customers | Product updates | SEO + lead gen |
| Lobste.rs | ~50K | Deep technical | High-signal, invite-only |

**Strategy**: Publish 2 blog posts per month. One on HackerNoon or Dev.to
(for reach), one on the company blog (for ownership and SEO).

### 3.4 Podcast Appearances

Podcasts are an excellent channel for reaching developers during commutes and
coding sessions.

**Target podcasts**:
- **Go Time** (Changelog Network) — The #1 Go podcast. Pitch a segment on
  "building auth in Go."
- **Kubernetes Podcast** (Google) — For the K8s audience.
- **The Changelog** — Broad open-source audience.
- **Software Engineering Daily** — Deep technical interviews.
- **InfoSec podcasts** (Risky Business, Defensive Security) — For the security
  angle.

**Pitch**: Offer to discuss "Why we open-sourced 145 security research docs"
or "Threat modeling an IAM system with STRIDE."

---

## 4. HackerNews Launch Strategy

### 4.1 Why HackerNews Matters

A successful HackerNews "Show HN" post can drive 10,000-50,000 visitors in 24
hours, generate 500+ GitHub stars, and put GGID on the radar of senior engineers
at major companies. It is the single highest-leverage launch event for a
developer tool.

### 4.2 The Post

**Title** (max 80 chars, must be factual, no clickbait):

> Show HN: GGID – Open-source IAM written in Go (OAuth 2.1, OIDC, SAML, WebAuthn)

**Alternative titles** (test 2-3 variants):
> Show HN: A Go-native alternative to Keycloak — 20MB binary, 7 microservices
> Show HN: We built an IAM system in Go and open-sourced 145 security research docs

**Opening comment** (the first comment by the submitter, ~150 words):

> Hi HN, we built GGID because we were tired of running Keycloak's JVM in our
> Go microservice stack. It's a multi-tenant IAM system with OAuth 2.1, OIDC,
> SAML 2.0, SCIM 2.0, RBAC + ABAC, MFA (TOTP + WebAuthn), and a NATS-based
> audit pipeline.
>
> A few things we think are interesting:
>
> 1. The entire auth stack deploys as 7 Go microservices — each ~20-35MB.
>    No JVM, no container bloat.
> 2. Multi-tenant isolation is enforced at the database level using PostgreSQL
>    Row-Level Security (RLS), not application-level checks.
> 3. We published 145+ security research documents covering everything from
>    OAuth mix-up attacks to WebAuthn attestation chains. Every design decision
>    is documented and threat-modeled.
>
> It's Apache 2.0, self-hosted, and works with Kubernetes out of the box.
> We'd love feedback, especially from anyone who has deployed Keycloak at
> scale and lived to tell the tale.
>
> Repo: [link] | Docs: [link] | Live demo: [link]

### 4.3 Timing

- **Best days**: Tuesday or Wednesday. Monday is crowded, Thursday/Friday
  traffic drops off.
- **Best time**: 9:00 AM Pacific Time (12:00 PM ET, 5:00 PM GMT). This catches
  the US East Coast morning break, US West Coast arrival, and European evening.
- **Avoid**: Major tech news days (Apple events, AWS re:Invent), holiday weeks,
  Friday afternoons.

### 4.4 What Resonates with HN

The HN audience rewards and punishes predictably:

**What works**:
- **Honest comparisons**: "Here's our benchmark vs Keycloak" with real numbers.
  HN loves data.
- **Technical depth**: Explaining the PostgreSQL RLS approach to multi-tenancy.
  HN respects engineering decisions.
- **Security transparency**: The 145 security research docs. HN is paranoid
  about security and respects open threat modeling.
- **Underdog narrative**: "We couldn't afford Auth0 at scale, so we built this."
- **Humble framing**: "We'd love feedback" not "The best IAM system ever."

**What flops**:
- **Marketing speak**: "Revolutionary," "game-changing," "next-generation."
  Instant downvotes.
- **Buzzword density**: "AI-powered zero-trust identity mesh" — HN will tear
  this apart.
- **Over-promising**: Claiming enterprise-readiness when SCIM is skeleton-only.
  Someone will check.
- **Defensiveness**: Responding to criticism with ego instead of data.

### 4.5 Pre-Launch Preparation

Before submitting to HN, the following must be ready:

1. **Live demo**: A deployed instance with a sample tenant. Users can register,
   login, and see the admin console without installing anything. This is
   non-negotiable — HN commenters will say "demo link?" within 5 minutes.

2. **Benchmark results**: A reproducible benchmark comparing GGID vs Keycloak
   on: token validation latency, throughput (req/s), memory usage, container
   image size, cold start time. Publish the methodology and make it reproducible.

3. **README polish**: Badges (build status, Go version, license, stars), a GIF
   demo, a comparison table, quick start instructions. See Section 6.

4. **Documentation quality**: The 5-minute quickstart must work flawlessly.
   Test it on a clean machine. HN will find broken docs.

5. **Docker Compose**: `docker compose up` must work on the first try. This is
   often the first thing an HN commenter tries.

6. **Prepare answers**: Pre-write responses for the top 5 likely questions:
   - "How does this compare to Keycloak?"
   - "Why not just use Ory?"
   - "Is this production-ready?"
   - "What about SCIM? Enterprise SSO?"
   - "How do you handle key rotation?"

7. **Notify supporters**: Quietly let 3-5 trusted community members know about
   the post so they can engage early. Early upvotes and thoughtful comments
   in the first 30 minutes are critical for reaching the front page.

### 4.6 Post-Launch

- **Be present**: Monitor the thread for 12+ hours. Respond to every comment
  within 30 minutes during peak hours.
- **Be gracious**: Thank people for feedback, even critical feedback.
- **Track metrics**: GitHub stars, repo traffic, issue creation, star/fork
  velocity. Document the results for future reference.
- **Follow up**: Write a blog post "What we learned from our HN launch" —
  this gets additional attention and shows transparency.

---

## 5. Reddit Strategy

### 5.1 Subreddit Map

| Subreddit | Members | Rules | Content Type |
|-----------|---------|-------|--------------|
| r/golang | 400K+ | No pure self-promo; must add value | Technical deep dives, lib announcements |
| r/selfhosted | 500K+ | Self-promo allowed in "Show your project" flair | Docker Compose, screenshots, homelab angle |
| r/kubernetes | 400K+ | No blog spam; must be substantive | Helm chart, K8s deployment guide |
| r/netsec | 500K+ | Strict: no marketing, technical only | Security research, threat model |
| r/devops | 600K+ | No direct promo; educational content | Infrastructure, automation |
| r/cybersecurity | 600K+ | No self-promo in posts; comments OK | Security analysis, compliance |
| r/SaaS | 200K+ | Showcase Saturdays for self-promo | B2B SaaS, multi-tenancy |

### 5.2 Content Strategy Per Subreddit

**r/golang**:
- **Do**: Post "I built an IAM system in Go — here's how I implemented
  multi-tenant isolation with PostgreSQL RLS." Include code snippets,
  architecture diagrams, lessons learned.
- **Don't**: Post "Check out my new project!" with just a link. This will be
  removed or ignored.
- **Rule of thumb**: The post must teach something. Even if no one uses GGID,
  they should learn about RLS-based multi-tenancy.

**r/selfhosted**:
- **Do**: Post in the weekly "Show Your Project" thread, or create a post with
  the "Show" flair. Include Docker Compose screenshot, admin console
  screenshots, and a comparison to Keycloak/Authentik.
- **Angle**: "Self-hosted Keycloak alternative in Go — 20MB containers, no JVM."
- **This community loves**: Small footprint, easy setup, no cloud dependency.

**r/kubernetes**:
- **Do**: Post a detailed guide on deploying GGID on K8s with Helm, HPA, and
  ingress configuration. Include YAML manifests.
- **Don't**: Post a link to the GitHub repo. It will be flagged as blog spam.
- **Angle**: "Deploying a microservice IAM stack on Kubernetes — 7 services,
  Helm chart, HPA, less than 200MB total RAM."

**r/netsec**:
- **Do**: Post about the security research — "We threat-modeled an entire IAM
  system with STRIDE and published the results." Link to the threat model
  document, not the product.
- **Don't**: Post anything that smells like marketing. r/netsec will remove it.
- **Angle**: Security research and transparency. The 145 docs are the hook.

**r/devops**:
- **Do**: Post about the operational aspects — "Running 7 Go microservices for
  IAM: monitoring, logging, scaling lessons." Focus on the ops experience.
- **Angle**: How GGID is easier to operate than Keycloak (no JVM tuning, smaller
  containers, faster deploys).

### 5.3 AMA Strategy

After building initial traction (500+ stars), propose an AMA:

**Target**: r/golang or r/devops.

**Title**: "We built an open-source IAM system in Go to replace Keycloak. Ask
us anything about Go, OAuth, multi-tenancy, or security."

**Preparation**:
- Have 3-4 team members ready to answer questions for 4-6 hours.
- Prepare answers for likely questions: "Why Go?", "How does this compare to
  Ory?", "Is it production-ready?", "What's the business model?"
- AMA must be transparent and honest. Reddit detects BS instantly.

### 5.4 Reddit Pitfalls

- **Posting too early**: If the product isn't polished, Reddit will notice and
  the narrative will be "it's not ready." Wait until the Docker Compose demo
  is flawless.
- **Same link, multiple subreddets**: Reddit's spam detection will flag this.
  Post in one subreddit, wait 2-3 days, then post a different angle in another.
- **Not engaging**: If someone comments and you don't respond within hours,
  the post loses momentum. Dedicate a full day to each Reddit post.

---

## 6. GitHub Strategy

### 6.1 GitHub Stars as Social Proof

GitHub stars are the primary social proof metric for open-source projects.
Developers judge a project's credibility by star count:

- **< 100 stars**: "Is this maintained? Is anyone using it?"
- **100-1,000 stars**: "Interesting, let me look closer."
- **1,000-5,000 stars**: "This has traction. I should try it."
- **5,000-10,000 stars**: "This is a serious project."
- **10,000+ stars**: "This is a community standard."

**Target milestones**:
- Month 1: 500 stars (HN launch + Reddit)
- Month 3: 1,000 stars
- Month 6: 2,500 stars
- Month 12: 5,000 stars

### 6.2 README Optimization

The README is the most important marketing asset. It must convert a visitor
into a starrer within 30 seconds.

**Essential elements** (in order):

1. **Project name + tagline** (1 line):
   > GGID — Production-grade, multi-tenant IAM written in Go.

2. **Badges** (1 row): CI status, Go Report Card, license (Apache 2.0),
   GitHub stars, latest release, Docker pulls.

3. **GIF demo** (5-10 second loop): Show the register → login → admin console
   flow. This is worth 1000 words.

4. **Comparison table** (compact): GGID vs Keycloak vs Auth0 vs Ory — 6 rows
   max, highlighting GGID's wins (Go-native, container size, self-hosted,
   security docs).

5. **Quick start** (5 lines max):
   ```bash
   git clone https://github.com/ggid/ggid
   cd ggid/deploy && docker compose up -d
   # Open http://localhost:3000 — register, login, explore
   ```

6. **Feature highlights** (bullet list): OAuth 2.1, OIDC, SAML, SCIM, RBAC/ABAC,
   MFA, WebAuthn, multi-tenant RLS, NATS audit, 7 microservices.

7. **Architecture diagram** (1 image): C4 model or service diagram.

8. **Links**: Documentation, quickstart guides, SDKs, contributing guide,
   security policy.

### 6.3 Awesome Lists

Getting GGID into curated awesome lists is high-value, low-effort exposure.

**Target lists** (submit PRs):
- `avelino/awesome-go` (35K+ stars) — under "Security" or "Authentication"
- `awesome-selfhosted` (200K+ stars) — under "Identity Management"
- `awesome-kubernetes` (15K+ stars) — if Helm chart is ready
- `awesome-security` (12K+ stars) — under "Identity"
- `go-awesome` / `golang-projects`
- `awesome-fintech` — if fintech use cases are documented
- `CNCF landscape` — under "Identity" category

**Strategy**: Submit to 2-3 lists at a time. Each PR should follow the list's
formatting rules exactly. Include a brief justification for inclusion.

### 6.4 GitHub Topics and Tags

Add relevant topics to the repo for discoverability:

`go`, `golang`, `iam`, `authentication`, `authorization`, `oauth2`, `openid-connect`,
`saml`, `scim`, `rbac`, `abac`, `multi-tenant`, `webauthn`, `passkey`, `mfa`,
`identity-management`, `self-hosted`, `kubernetes`, `microservices`, `security`

### 6.5 GitHub Discussions

Enable GitHub Discussions and seed it with categories:
- **Announcements** — releases, roadmap updates
- **Q&A** — usage questions, troubleshooting
- **Ideas** — feature requests, feedback
- **Show and Tell** — user projects, integrations
- **Security** — vulnerability discussions, hardening tips

Seed 3-5 discussions in the first week to avoid an empty forum. Respond to
every discussion within 24 hours for the first 3 months.

### 6.6 Release Strategy

- **Semantic versioning**: v0.x for pre-1.0, indicating API instability. Move
  to v1.0 when the API stabilizes and enterprise features (SCIM, compliance)
  are complete.
- **Release notes**: Every release must have human-readable notes with:
  - What's new (features)
  - What's fixed (bugs)
  - Breaking changes (if any)
  - Migration instructions (if breaking)
  - Contributors (with GitHub handles)
- **GitHub Releases**: Use the releases page with binaries attached. Include
  checksums and a changelog link.

---

## 7. Content Marketing

### 7.1 Blog Post Ideas (Ranked by Impact)

**Tier 1 — Launch content** (publish in first 3 months):

1. **"Why We Built GGID in Go Instead of Java"**
   - The origin story. Why Go, why microservices, why not Keycloak.
   - Target: HackerNoon, r/golang.
   - Key points: JVM memory tax, container bloat, operational simplicity.

2. **"Threat Modeling an IAM System with STRIDE"**
   - Walk through the STRIDE analysis of GGID. Show the 10 P0 findings, how
     they were fixed, and the remaining gaps.
   - Target: r/netsec, Dev.to.
   - Key points: Security transparency as competitive advantage.

3. **"Multi-Tenant Isolation with PostgreSQL Row-Level Security"**
   - Deep technical dive into how GGID uses RLS for tenant isolation. Code
     examples, migration patterns, gotchas.
   - Target: r/golang, r/devops.
   - Key points: Database-level security beats application-level checks.

4. **"145 Security Research Docs: Our Methodology"**
   - How we researched and documented 145 IAM security topics. The research
     process, the tooling, what we learned.
   - Target: HackerNoon, company blog.
   - Key points: Documentation as moat.

**Tier 2 — Growth content** (months 3-6):

5. **"Benchmarking GGID vs Keycloak: Token Validation at Scale"**
6. **"OAuth 2.1 Migration Guide: What Changed and Why It Matters"**
7. **"Deploying a Microservice IAM Stack on Kubernetes"**
8. **"Building a Go SDK for IAM: Lessons in API Design"**
9. **"How We Handle Key Rotation in a Distributed System"**
10. **"WebAuthn/Passkey Implementation: A Practical Guide"**

**Tier 3 — Evergreen content** (months 6-12):

11. **"RBAC vs ABAC: Choosing the Right Authorization Model"**
12. **"SAML 2.0 in 2025: Still Relevant, Still Painful"**
13. **"Audit Logging with NATS JetStream: A Pattern for Event Sourcing"**
14. **"Identity at the Edge: Running Auth in 20MB"**
15. **"The Open-Source IAM Landscape in 2025"**

### 7.2 SEO Keywords

Target long-tail keywords that have purchase intent and manageable competition:

**Primary keywords** (high volume, high competition):
- "open source IAM" — 4,400 searches/month
- "identity and access management" — 8,100/month
- "self-hosted authentication" — 1,900/month
- "Go authentication" — 720/month
- "golang auth" — 590/month

**Secondary keywords** (lower volume, lower competition, higher intent):
- "keycloak alternative" — 1,300/month
- "open source oauth server" — 880/month
- "self-hosted SSO" — 720/month
- "multi-tenant authentication" — 480/month
- "golang oauth2 server" — 320/month
- "RBAC authorization engine" — 260/month
- "WebAuthn implementation guide" — 210/month
- "PostgreSQL row level security multi-tenant" — 170/month

**Long-tail keywords** (very specific, very high intent):
- "open source IAM written in Go" — ~100/month
- "lightweight alternative to Keycloak" — ~80/month
- "self-hosted OAuth 2.1 server" — ~60/month
- "multi-tenant IAM PostgreSQL" — ~50/month
- "Go microservices authentication" — ~40/month

**Content-SEO alignment**: Each Tier 1 blog post should target 2-3 long-tail
keywords. The comparison docs should target "X alternative" keywords.

### 7.3 Video Content

- **YouTube channel**: Short (5-10 min) technical videos.
  - "GGID in 5 minutes" — demo.
  - "How multi-tenant RLS works" — whiteboard explanation.
  - "Deploying GGID on Kubernetes" — screen recording.
- **Screencasts**: Embed in README and docs.
- **Conference recordings**: After each talk, upload to YouTube and the repo.

---

## 8. Developer Advocacy

### 8.1 The Role

A Developer Advocate (DevRel) is the highest-ROI hire for an open-source IAM
project. This person is the bridge between the product and the community.

### 8.2 Who to Hire

**Profile**:
- **Deep Go expertise**: Must be able to write Go code on stage, review PRs,
  and answer deep technical questions. Not a marketing person who learned Go.
- **IAM/security domain knowledge**: Understands OAuth, OIDC, SAML, JWT. Has
  built or operated auth systems before.
- **Communication skills**: Can write clearly, speak confidently, and explain
  complex concepts to varied audiences.
- **Community presence**: Already known in Go or security communities. Has a
  blog, Twitter following, or conference speaking history.
- **Builder mindset**: Ships code, writes demos, creates sample apps.

**Ideal candidate backgrounds**:
- Former Go engineer at a company that built internal auth (Stripe, Monzo,
  DigitalOcean).
- Open-source maintainer in the Go security ecosystem.
- Former Auth0/Clerk/Okta developer advocate who wants to work on open-source.
- Conference speaker (GopherCon, BSides) with auth expertise.

### 8.3 What They Create

**Content (weekly)**:
- 1 blog post (technical deep dive or tutorial)
- 1-2 community engagement sessions (GitHub Discussions, Discord, Reddit)
- 1 demo or sample app update
- Social media presence (Twitter threads, LinkedIn posts)

**Content (monthly)**:
- 1 conference talk submission
- 1 video tutorial
- Community metrics report (stars, issues, PRs, downloads)

**Content (quarterly)**:
- 1 major sample application (e.g., "Build a B2B SaaS with GGID")
- Workshop materials updated
- Conference talk delivered

### 8.4 Conference Circuit

The Dev Advocate should aim to speak at 4-6 conferences per year:

| Conference | Timing | Audience | Talk Theme |
|-----------|--------|----------|------------|
| GopherCon | July | Go developers | Architecture, Go patterns |
| KubeCon NA | November | Cloud-native | K8s deployment, Helm |
| KubeCon EU | April | Cloud-native (EU) | Self-hosted IAM |
| FOSDEM | February | Open-source (EU) | Open-source IAM |
| GoLab | October | Go (EU) | Multi-tenancy |
| BSides (various) | Year-round | Security | Threat modeling |

### 8.5 Workshop Materials

Create reusable workshop content:

1. **"Getting Started with GGID"** (2 hours): Docker Compose, register/login,
   admin console, SDK integration.
2. **"Multi-Tenant IAM with GGID"** (4 hours): Tenant onboarding, RLS,
   RBAC/ABAC policy engine, SSO configuration.
3. **"Securing Your Go API with GGID"** (3 hours): JWT validation, middleware,
   scope enforcement, MFA flows.
4. **"Contributing to GGID"** (2 hours): Codebase tour, development setup,
   PR process, testing conventions.

These workshops can be delivered at conferences, meetups, or as online
webinars. Record them and publish to YouTube.

### 8.6 Office Hours

Host weekly office hours (1 hour, Zoom or Discord):
- **Format**: Open Q&A. Anyone can join and ask questions about GGID, Go auth,
  or IAM in general.
- **Purpose**: Build relationships, gather feedback, unblock users.
- **Recording**: Post a summary of interesting questions to the blog.

---

## 9. Competitor Response Playbook

### 9.1 How Auth0/Okta Might Respond

**Most likely response**: Ignore. GGID is too small to register on Okta's radar
until it reaches 5,000+ stars and starts appearing in deal cycles.

**If they notice**:
- **FUD campaign**: "Open-source IAM lacks enterprise support, compliance
  certifications, and SLAs." Counter with: GGID's security research depth
  (145 docs) exceeds Auth0's public documentation. Enterprise support is a
  roadmap item.
- **Feature acceleration**: Okta might accelerate features that GGID
  differentiates on (Go SDKs, multi-tenant APIs). Counter by staying ahead on
  developer experience and transparency.
- **Acquisition interest**: If GGID gains significant traction, Okta might
  express acquisition interest. This is a positive signal, not a threat.

**Defense**: Don't compete with Okta on enterprise features. Compete on
developer experience, transparency, and cost. Okta cannot match free +
open-source + Go-native.

### 9.2 How Keycloak Community Might React

**Most likely response**: Mixed. Some will see GGID as a welcome alternative.
Others will be defensive.

**Positive reactions**:
- "Finally, a Go option. Keycloak's JVM has been a pain point for years."
- "The security research docs are impressive. Keycloak should do this."

**Negative reactions**:
- "Keycloak is battle-tested. Why switch to a new project?"
- "Java is fine. The JVM overhead is a non-issue if you configure it properly."
- "Another IAM project? The world doesn't need another one."

**How to handle**:
- **Never bash Keycloak**. It's a respected project. Acknowledge its strengths.
- **Position as complementary**: "GGID isn't replacing Keycloak for everyone.
  It's for teams who want a Go-native option."
- **Focus on the differentiation**: Container size, deployment simplicity,
  security research depth. Let the data speak.
- **Engage respectfully**: Keycloak maintainers are smart, passionate people.
  Treat them as peers, not enemies.

### 9.3 How Ory Community Might React

Ory (Kratos/Keto/Hydra) is the closest competitor — also Go-based, also
open-source. This is the most important competitive relationship.

**Expected reactions**:
- "How is this different from Ory?"
- "Ory is more mature. Why use GGID?"
- "Ory has CNCF backing. What does GGID have?"

**How to handle**:
- **Acknowledge Ory's strengths**: Ory is a great project with CNCF sandbox
  status. They pioneered Go-native IAM.
- **Highlight differences**:
  - GGID is a single integrated platform (7 services, one deployment). Ory is
    4 separate projects with different maturity levels.
  - GGID has multi-tenant RLS at the database level. Ory doesn't.
  - GGID has 145 security research docs. Ory has good docs but not this depth.
  - GGID includes SAML, SCIM, WebAuthn in one stack. Ory requires combining
    Kratos + Hydra + Keto.
- **Don't start fights**: Ory's community is our potential community. Be
  collaborative, not combative. Consider cross-promotion opportunities.

### 9.4 Handling "Just Use X" Comments

Every launch will attract "just use Auth0/Keycloak/Firebase/Clerk" comments.
Prepare a standard response:

> Totally fair point. [X] is a great choice for many teams. We built GGID for
> teams who specifically want:
> 1. A Go-native stack (no JVM, no JS runtime for auth)
> 2. Self-hosted with full source auditability
> 3. Multi-tenant isolation at the database level
> 4. No per-MAU pricing
>
> If [X] works for you, that's great! GGID is for the teams where [X] doesn't
> fit. We're not trying to replace [X] for everyone — just providing an option
> for the Go ecosystem.

**Key principle**: Never be defensive. Validate their choice, explain the
differentiation, and let them decide.

### 9.5 FUD Defense

Common FUD (Fear, Uncertainty, Doubt) attacks and responses:

| FUD Attack | Response |
|-----------|----------|
| "It's not production-ready" | "We have 250+ tests, Docker Compose E2E, and threat modeling. Here's the test coverage report." |
| "No enterprise support" | "Community support now, enterprise support on the roadmap. The code is Apache 2.0 — you own it." |
| "No compliance certifications" | "SOC 2 / ISO 27001 are on the roadmap. The security research docs exceed what most certified products publish." |
| "It's a new project, no track record" | "Every project starts somewhere. The architecture is proven (gRPC, PostgreSQL, NATS). The code is auditable." |
| "Who's using it in production?" | Be honest. "We're in early adoption. Here are the teams evaluating it." Don't fabricate customer names. |

---

## 10. 12-Month Community Building Plan

### Month 1: Foundation (Pre-Launch)

**Goal**: Polish everything for the public launch.

**Tasks**:
- [ ] README optimization (badges, GIF, comparison table, quick start)
- [ ] Docker Compose `docker compose up` tested on clean machines
- [ ] 5-minute quickstart guide tested and flawless
- [ ] Live demo deployed (register, login, admin console accessible)
- [ ] Benchmark results: GGID vs Keycloak (latency, throughput, memory, size)
- [ ] GitHub Discussions enabled and seeded with 3-5 starter discussions
- [ ] CONTRIBUTING.md, SECURITY.md, CODE_OF_CONDUCT.md published
- [ ] Awesome list PRs prepared (awesome-go, awesome-selfhosted)
- [ ] Blog post "Why We Built GGID in Go" drafted
- [ ] HackerNews Show HN post drafted and reviewed
- [ ] Social media accounts created (Twitter/X: @ggid_iam)

**Metrics**: N/A (pre-launch). Internal readiness checklist.

**Prerequisites**: Docker Compose works, docs are complete, demo is live.

### Month 2: Launch

**Goal**: Maximize launch visibility.

**Week 1**:
- [ ] HackerNews Show HN post (Tuesday or Wednesday, 9 AM PT)
- [ ] r/golang post (different angle, 2 days after HN)
- [ ] Blog post published: "Why We Built GGID in Go"
- [ ] Awesome list PRs submitted
- [ ] Notify 5-10 supporters for early engagement

**Week 2**:
- [ ] r/selfhosted post ("Show" flair)
- [ ] r/devops post (operational angle)
- [ ] Dev.to cross-post of blog article
- [ ] Respond to all HN/Reddit/GitHub comments within hours

**Week 3-4**:
- [ ] r/netsec post (security research angle)
- [ ] Blog post: "STRIDE Threat Modeling an IAM System"
- [ ] GitHub Issues triage — respond to all launch-related issues
- [ ] Collect feedback, prioritize bug fixes

**Metrics**: 500+ GitHub stars, 50+ GitHub Discussions, 5,000+ unique visitors.

### Month 3: Content Velocity

**Goal**: Establish content cadence and begin SEO accumulation.

**Tasks**:
- [ ] Blog post: "Multi-Tenant Isolation with PostgreSQL RLS"
- [ ] Blog post: "145 Security Research Docs: Our Methodology"
- [ ] Submit CFP to GopherCon (deadline is typically February)
- [ ] Submit CFP to KubeCon EU
- [ ] YouTube: "GGID in 5 Minutes" demo video
- [ ] r/golang AMA (if 500+ stars reached)
- [ ] Begin podcast outreach (Go Time, Kubernetes Podcast)
- [ ] First awesome-list PR merged

**Metrics**: 1,000+ stars, 100+ Discussions, 10+ external blog mentions.

### Month 4-5: Community Maturation

**Goal**: Convert stars into contributors and users.

**Tasks**:
- [ ] Blog post: "Benchmarking GGID vs Keycloak"
- [ ] Blog post: "OAuth 2.1 Migration Guide"
- [ ] Helm chart published and documented
- [ ] Sample applications: "Build a Go API with GGID" (Go), "Node.js + GGID"
- [ ] First contributor onboarding guide published
- [ ] Discord server launched for community chat
- [ ] Weekly office hours begin
- [ ] Submit to CNCF landscape

**Metrics**: 1,500+ stars, 5+ external contributors, 20+ Docker pulls/day.

### Month 6: Conference Presence

**Goal**: Deliver first conference talk.

**Tasks**:
- [ ] Deliver GopherCon or KubeCon talk (if accepted)
- [ ] Blog post: "What We Learned Speaking at [Conference]"
- [ ] Conference recording published to YouTube
- [ ] Blog post: "Deploying Microservice IAM on Kubernetes"
- [ ] Mid-year community survey (what do users want next?)
- [ ] v0.5 release with community-requested features
- [ ] First podcast appearance

**Metrics**: 2,500+ stars, 50+ external contributors, conference talk delivered.

### Month 7-8: Ecosystem Expansion

**Goal**: Build integration ecosystem.

**Tasks**:
- [ ] Integration guides: Istio, Linkerd, Traefik, Envoy
- [ ] Integration guides: Grafana, Prometheus (monitoring)
- [ ] SDK releases: Go SDK v1, Node.js SDK v1
- [ ] Blog post: "Go SDK Design for IAM"
- [ ] Submit CFP to FOSDEM, GoLab
- [ ] Second podcast appearance
- [ ] Case study: first production deployment (if available)

**Metrics**: 3,000+ stars, SDK downloads, 3+ integration guides published.

### Month 9-9: Deepening Engagement

**Goal**: Establish thought leadership in IAM security.

**Tasks**:
- [ ] Blog post series: "OAuth Security Best Practices" (3 parts)
- [ ] Blog post: "WebAuthn/Passkey Implementation Guide"
- [ ] Security webinar: "Threat Modeling Your Auth Stack"
- [ ] FOSDEM talk delivered (if accepted)
- [ ] Workshop materials published
- [ ] Community spotlight blog series (featuring user projects)

**Metrics**: 3,500+ stars, webinar attendees, workshop materials downloaded.

### Month 10-11: Scale and Sustainability

**Goal**: Ensure project sustainability and contributor growth.

**Tasks**:
- [ ] Contributor recognition program (monthly blog post highlighting contributors)
- [ ] Blog post: "The Open-Source IAM Landscape in 2025"
- [ ] v1.0-beta release (feature-complete, API stable)
- [ ] Submit CFP to KubeCon NA
- [ ] Third podcast appearance
- [ ] Developer Advocate hire (if budget allows)
- [ ] Sponsorship program launched (GitHub Sponsors)

**Metrics**: 4,500+ stars, 100+ external contributors, v1.0-beta released.

### Month 12: v1.0 and Beyond

**Goal**: Ship v1.0 and plan year 2.

**Tasks**:
- [ ] v1.0 release with stable API guarantee
- [ ] Blog post: "GGID v1.0: What We Built and What's Next"
- [ ] HackerNews "Show HN: GGID v1.0" (second launch)
- [ ] KubeCon NA talk delivered (if accepted)
- [ ] Year-in-review blog post with metrics
- [ ] Year 2 roadmap published (community-informed)
- [ ] Anniversary community event (live stream, Q&A, demos)

**Metrics**: 5,000+ stars, v1.0 released, 150+ contributors, sustainable
community engagement.

### 12-Month Metrics Dashboard

| Metric | Month 3 | Month 6 | Month 9 | Month 12 |
|--------|---------|---------|---------|----------|
| GitHub Stars | 1,000 | 2,500 | 3,500 | 5,000 |
| Docker Pulls (total) | 2,000 | 10,000 | 25,000 | 50,000 |
| External Contributors | 5 | 20 | 50 | 100 |
| GitHub Discussions | 100 | 300 | 600 | 1,000 |
| Blog Posts Published | 6 | 12 | 18 | 24 |
| Conference Talks | 0 | 1 | 2 | 3 |
| Podcast Appearances | 0 | 1 | 2 | 3 |
| Discord Members | 50 | 200 | 500 | 1,000 |
| SDK Downloads | 500 | 5,000 | 15,000 | 30,000 |

### Dependencies and Prerequisites

| Milestone | Depends On |
|-----------|-----------|
| HN Launch | Docker Compose works, demo live, README polished |
| Reddit posts | HN launch completed, initial traction |
| Conference talk | CFP accepted (depends on submission timing) |
| Awesome list PR | Project is functional and documented |
| AMA | 500+ stars, active community |
| v1.0-beta | SCIM 2.0 complete, API stable, compliance work started |
| Dev Advocate hire | Funding/budget approved |
| CNCF submission | Project meets CNCF sandbox criteria (governance, contributors) |

---

## Appendix: Key Resources

### Community Links (to be created)
- GitHub: `github.com/ggid/ggid`
- Documentation: `ggid.dev/docs`
- Discord: `discord.gg/ggid`
- Twitter/X: `@ggid_iam`
- Blog: `ggid.dev/blog`

### Reference Projects (Launch Playbooks)
- **Ory** (2019 launch): Gradual build, CNCF sandbox, strong content marketing
- **Casdoor** (2021 launch): Chinese-first community, GitHub stars as primary metric
- **Logto** (2022 launch): Strong design focus, developer experience first
- **Zitadel** (2020 launch): Enterprise-first, cloud-native positioning

### Competitive Intelligence Sources
- `docs/research/auth0-comparison.md` — 67-feature Auth0 comparison
- `docs/research/keycloak-comparison.md` — Keycloak feature matrix
- `docs/research/ory-comparison.md` — Ory deep dive
- `docs/research/casdoor-comparison.md` — Casdoor analysis
- `docs/research/competitor-update-clerk-logto-casdoor.md` — Latest updates
- `docs/research/iam-differentiation-strategy.md` — Strategic positioning

---

*This document is a living strategy guide. Update quarterly with actual
results, community feedback, and market changes.*
