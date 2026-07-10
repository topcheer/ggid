# Edge Computing IAM: Authentication and Authorization at the CDN Edge

> Research document for GGID IAM Suite — evaluating edge-first authentication patterns.

---

## 1. Overview

Edge computing processes data near the user instead of a centralized data center. Applied to IAM, this means authenticating and authorizing requests at CDN edge nodes — geographically distributed points of presence (PoPs) that sit within milliseconds of the end user.

### Benefits

| Benefit | Detail |
|---------|--------|
| Lower latency | JWT validated locally — no round-trip to origin auth server |
| Better availability | Edge survives origin outages for token-validated reads |
| Origin offload | Invalid tokens rejected at edge, never reaching origin |
| DDoS mitigation | Rate limiting and geo-blocking at edge before traffic hits origin |

### Challenge

The edge is fundamentally untrusted and resource-constrained. Edge workers run on shared infrastructure across hundreds of PoPs. You **cannot** store signing keys, database credentials, or full user databases at the edge. Only public keys (JWKS) and non-sensitive configuration are safe to distribute.

### Platforms

| Platform | Runtime | Cold Start | Global PoPs |
|----------|---------|------------|-------------|
| **Cloudflare Workers** | V8 isolates | ~5ms | 330+ |
| **Fastly Compute@Edge** | WASM | ~1ms | 80+ |
| **Akamai EdgeWorkers** | V8 (JS) | ~10ms | 4,000+ |
| **AWS Lambda@Edge** | Node.js | ~100ms | 600+ |

Cloudflare Workers is the primary reference platform for this document due to its mature KV storage, zero cold-start penalty, and first-class JWT library (`jose`) support.

---

## 2. JWT Validation at CDN Edge

### How It Works

```
Client ──Authorization: Bearer <JWT>──> Edge Worker
                                          │
                              ┌───────────┴───────────┐
                              │  1. Parse JWT          │
                              │  2. Fetch JWKS (KV)    │
                              │  3. Verify signature   │
                              │  4. Validate claims    │
                              └───────────┬───────────┘
                                          │
                              ┌───── valid? ─────┐
                              │                  │
                            YES                 NO
                              │                  │
                   + X-User-ID header      401 Unauthorized
                   forward to origin      (at edge, no origin hit)
```

The edge worker validates the JWT signature locally using the JWKS (JSON Web Key Set) — a set of **public** keys that is safe to distribute to all edge nodes. No private key or network round-trip to the auth server is needed.

### Implementation: Cloudflare Workers

```javascript
// edge-auth-worker.js — JWT validation at the CDN edge
import { jwtVerify, createRemoteJWKSet } from 'jose';

// JWKS cached at edge via Workers KV (5-min TTL)
const JWKS_CACHE_KEY = 'jwks_cache';
const JWKS_ORIGIN = 'https://auth.ggid.example.com/.well-known/jwks.json';
let remoteJWKS = null;

async function getJWKS(env) {
  // Check KV cache first
  const cached = await env.KV.get(JWKS_CACHE_KEY);
  if (cached) {
    return JSON.parse(cached);
  }
  // Cold path: fetch from origin
  const resp = await fetch(JWKS_ORIGIN);
  const jwks = await resp.json();
  await env.KV.put(JWKS_CACHE_KEY, JSON.stringify(jwks), { expirationTtl: 300 });
  return jwks;
}

async function validateJWT(token, env) {
  const jwks = await getJWKS(env);
  const keyStore = createRemoteJWKSet(new URL(JWKS_ORIGIN));

  try {
    const { payload, protectedHeader } = await jwtVerify(token, keyStore, {
      issuer: 'https://auth.ggid.example.com',
      audience: 'ggid-api',
      algorithms: ['RS256', 'ES256'],
    });
    return { valid: true, claims: payload };
  } catch (err) {
    // Expired token, invalid signature, missing claims, etc.
    return { valid: false, error: err.message };
  }
}

export default {
  async fetch(request, env) {
    const auth = request.headers.get('Authorization');
    if (!auth || !auth.startsWith('Bearer ')) {
      return new Response(JSON.stringify({ error: 'missing_token' }), {
        status: 401, headers: { 'Content-Type': 'application/json' }
      });
    }

    const token = auth.slice(7);
    const result = await validateJWT(token, env);

    if (!result.valid) {
      return new Response(JSON.stringify({ error: result.error }), {
        status: 401, headers: { 'Content-Type': 'application/json' }
      });
    }

    // Inject verified claims as headers for origin
    const originRequest = new Request(request, {
      headers: new Headers(request.headers),
    });
    originRequest.headers.set('X-User-ID', result.claims.sub || '');
    originRequest.headers.set('X-Tenant-ID', result.claims.tenant_id || '');
    originRequest.headers.set('X-Scopes', (result.claims.scope || []).join(','));
    originRequest.headers.delete('Authorization'); // strip token at edge

    return fetch(originRequest);
  },

  // Scheduled job: refresh JWKS in KV every 5 minutes
  async scheduled(event, env) {
    const resp = await fetch(JWKS_ORIGIN);
    const jwks = await resp.json();
    await env.KV.put(JWKS_CACHE_KEY, JSON.stringify(jwks), { expirationTtl: 600 });
  },
};
```

### Performance

| Metric | Value |
|--------|-------|
| JWT validation at edge | **< 1ms** (no network round-trip) |
| JWKS cache hit rate | **99.9%+** (refreshed every 5 min via scheduled trigger) |
| Cold-start JWKS fetch | ~50-100ms (first request after deploy only) |
| Origin load reduction | 401 responses handled at edge — origin never sees invalid tokens |

---

## 3. Geographic Session Routing

### Problem

A user in Tokyo authenticates against a US-East auth server and receives a JWT. Subsequent API requests route to the nearest edge node (Tokyo), which must validate the token against the US-East JWKS. Session state (refresh tokens, revocation lists) lives in Redis — which must be accessible from the edge region.

```
Tokyo Client ──> Tokyo Edge ──validate JWT──> (JWKS from KV, no round-trip)
                     │
                     ├──> US-East Origin (for stateful ops)
                     │         │
                     │         └──> US-East Redis (session store)
                     │
                     └──> AP-SE Redis replica (for revocation checks)
```

### Solution: Geo-Distributed Redis

Deploy a Redis cluster spanning regions: **US-East**, **EU-West**, **AP-Southeast**. Session data (refresh tokens, active sessions, revocation lists) replicates across all regions. The edge worker reads from the nearest Redis replica, avoiding cross-region latency.

| Data Type | Replication | TTL |
|-----------|-------------|-----|
| Refresh tokens | Sync (strong) | 7 days |
| Active sessions | Sync (strong) | 24 hours |
| Revocation list | Async (eventual) | 1 hour |
| Rate-limit counters | Local per-edge | 60 seconds |

### Token Binding to Region

The JWT includes a region hint claim:

```json
{
  "sub": "user-123",
  "iss": "https://auth.ggid.example.com",
  "region": "ap-southeast-1",
  "session_id": "sess-abc123",
  "exp": 1735689600
}
```

- **Stateless operations** (JWT validation): work anywhere — no region needed, JWKS is global.
- **Stateful operations** (session revocation, refresh): must hit the correct Redis region matching the `region` claim.

---

## 4. Edge-First IAM Architecture

### Components at Edge

| Component | Implementation | Latency |
|-----------|---------------|---------|
| JWT validation | WebCrypto in Worker | < 1ms |
| Rate limiting | Workers KV counter per IP/tenant | < 1ms |
| Geo-blocking | Country allow/deny list (cf-ipcountry header) | < 0.5ms |
| Bot detection | JA3 fingerprint + header analysis | < 1ms |
| CORS preflight | OPTIONS handled at edge | < 0.5ms |

### Components at Origin (GGID)

- Full authentication: login, MFA, password reset, LDAP/OAuth flows
- Session creation and storage (Redis)
- Policy evaluation: RBAC role checks, ABAC attribute rules
- Audit logging: NATS JetStream event publishing
- User management: CRUD against PostgreSQL

### Data Flow

```
1. Client ──────────────────────────> Edge (Tokyo)
   [GET /api/v1/users + Bearer JWT]

2. Edge Worker:
   ├── Validate JWT signature (JWKS from KV)     → <1ms
   ├── Check rate limit (KV counter)              → <1ms
   ├── Check geo rule (cf-ipcountry)             → <0.5ms
   └── Inject X-User-ID, X-Tenant-ID headers

3. Edge ────────────────────────────> Origin (GGID Gateway)
   [GET /api/v1/users + X-User-ID: user-123]

4. Origin:
   ├── Policy check (RBAC: users:read)
   ├── Database query (PostgreSQL + RLS)
   └── Audit event (NATS JetStream)

5. Origin ──> Edge ──────────────────> Client
   [200 OK + user list]
```

### What Stays at Origin

| Operation | Why |
|-----------|-----|
| Login flow | Needs password/credential validation against DB |
| OAuth consent | Needs interactive user redirects |
| Token issuance | Needs signing key — **never at edge** |
| User CRUD | Needs database access (PostgreSQL) |
| MFA challenge/response | Needs TOTP/OTP verification logic + secret |

---

## 5. Latency Optimization Patterns

### Token Prefetch

```
Client timeline:
  ──[token issued]──────────[edge detects exp < 5min]──[client prefetches]──[token refreshed]
   t=0                       t=55min                    t=55min              t=56min
```

The edge worker inspects `exp` on every validated token. If the token expires within 5 minutes, the edge adds a response header:

```
X-Token-Refresh-Hint: true
```

The client proactively refreshes without blocking the current request — no user-visible latency spike.

### JWKS Caching Strategy

| Layer | TTL | Purpose |
|-------|-----|---------|
| Edge KV | 5 min | Primary cache, shared across all PoPs |
| Scheduled Worker | Every 5 min | Background refresh, keeps KV warm |
| Origin fallback | On-demand | Cold-start only, fetches if KV miss |
| Key rotation | < 5 min | New keys propagate via KV automatically |

### Session Affinity

For stateful sessions (server-side session store), the edge routes to a **consistent origin**:

```
Set-Cookie: __affinity=us-east-1; HttpOnly; Secure; SameSite=Strict
```

The edge inspects this cookie and routes to the matching origin region. Alternative: go fully stateless with JWT-only validation (no server session) — eliminates the affinity problem entirely but loses the ability to revoke sessions instantly.

### Performance Targets

| Scenario | Latency |
|----------|---------|
| Edge JWT validation | < 2ms |
| Edge rate-limit check | < 1ms (in-memory KV) |
| Edge geo/bot check | < 1ms |
| Origin request (same-region) | 20-50ms |
| Origin request (cross-continent) | 100-200ms |
| **Total with edge (same-region)** | **22-53ms** |
| **Total without edge (cross-continent)** | **200-400ms** |

Edge validation delivers **5-10x latency reduction** for authenticated requests.

---

## 6. Cloudflare Workers Integration Design

### Worker Code Structure

```
┌──────────────────────────────────────────┐
│            Cloudflare Worker              │
│                                          │
│  fetch(request, env)                     │
│    ├── extractToken()    // parse Bearer │
│    ├── validateJWT()     // WebCrypto    │
│    ├── checkRateLimit()  // KV counter   │
│    ├── checkGeo()        // cf-ipcountry │
│    ├── injectHeaders()   // X-User-ID    │
│    └── forwardRequest()  // → GGID       │
│                                          │
│  scheduled(event, env)                   │
│    └── refreshJWKS()     // KV update    │
└──────────────────────────────────────────┘
```

### Workers KV Namespaces

| Namespace | Key Format | TTL | Description |
|-----------|-----------|-----|-------------|
| `jwks_cache` | `jwks:{issuer}` | 5 min | Public signing keys (safe to distribute) |
| `rate_limits` | `rl:{ip}:{tenant}` | 60s | Per-IP/per-tenant request counters |
| `geo_rules` | `geo:{tenant}` | 1 hour | Country block/allow lists per tenant |
| `config` | `cfg:{tenant}` | 5 min | Per-tenant rate limit thresholds, feature flags |

### Deploying GGID Behind Cloudflare

```
DNS: api.ggid.example.com → CNAME → workers.dev (Cloudflare)
                                   │
                          ┌────────┴────────┐
                          │  Cloudflare Edge │
                          │  (JWT validate)  │
                          └────────┬────────┘
                                   │
                          Origin (private IP)
                          ┌────────┴────────┐
                          │  GGID Gateway   │
                          │  10.0.0.5:8080  │
                          └────────┬────────┘
                                   │
                    ┌──────────────┼──────────────┐
                    │              │              │
              ┌─────┴─────┐ ┌─────┴─────┐ ┌─────┴─────┐
              │   Auth    │ │  Policy   │ │   Org     │
              │ :9001     │ │ :8070     │ │ :8071     │
              └───────────┘ └───────────┘ └───────────┘
```

**Key principle:** The origin (GGID gateway) is on a **private network** — never publicly accessible. All traffic flows through the edge worker. Signing keys live only at the origin auth service; only public JWKS is distributed to the edge.

---

## 7. Comparison: Edge Platforms

| Feature | Cloudflare Workers | Fastly Compute@Edge | Akamai EdgeWorkers | AWS Lambda@Edge |
|---------|-------------------|---------------------|---------------------|-----------------|
| **Runtime** | V8 isolates | WASM | V8 (JS) | Node.js |
| **Cold start** | ~5ms | ~1ms | ~10ms | ~100ms |
| **KV storage** | Workers KV | Object Store | EdgeKV | DynamoDB |
| **JWT library** | `jose` (native) | `wasm-jose` | Custom impl | `jsonwebtoken` |
| **Rate limiting** | KV counters (built-in) | Custom | EdgeKV counters | DynamoDB |
| **WebSocket** | Yes | Yes | Limited | Yes |
| **WASM support** | Partial | Yes (native) | No | No |
| **PoPs** | 330+ | 80+ | 4,000+ | 600+ |
| **Free tier** | 100k req/day | $0.50/M reqs | Enterprise only | Includes in CloudFront |

**Recommendation for GGID:** Cloudflare Workers — best balance of cold-start performance, mature KV, native JWT support, and generous free tier for development.

---

## 8. GGID Edge Integration Roadmap

| Phase | Scope | Effort | Dependencies |
|-------|-------|--------|--------------|
| **1. JWT Validation Worker** | Port GGID's Go JWKS verification to Cloudflare Worker (JS). Deploy behind Cloudflare DNS. | ~1 week | GGID JWKS endpoint (`/.well-known/jwks.json`) |
| **2. Edge Rate Limiting** | Workers KV per-IP/per-tenant counters. Aggregate to central Prometheus. | ~1 week | Phase 1 deployed |
| **3. Geo-Blocking + Bot Detection** | Country allow/deny per tenant. JA3 fingerprint filtering. | ~1 week | Phase 2 deployed |
| **4. Multi-Region Redis** | Geo-distributed Redis for session affinity and revocation. Region hint in JWT claims. | ~2 weeks | Redis cluster setup |
| **5. Full Edge-First** | OAuth authorize flow at edge (redirect + consent). Token refresh at edge. | ~4 weeks | Phase 4 + careful security review |

**Note:** Phase 5 is ambitious — most auth flows (login, MFA, credential issuance) inherently require database access and **must remain at origin**. Edge optimizes **token validation**, not issuance. Phases 1-3 (~3 weeks) deliver immediate latency and DDoS benefits; Phase 4 (~2 weeks) adds global sessions; Phase 5 (~4 weeks) requires careful ROI evaluation.

## Conclusion

Edge-first IAM moves **authorization validation** (JWT verification, rate limiting, geo policy) to where latency matters most. Signing keys, credential databases, and policy engines remain securely at origin. For GGID, the highest-ROI path is Phase 1-3: a Cloudflare Worker delivering sub-2ms authorization at 330+ PoPs while keeping credential logic behind the gateway.

*References: Cloudflare Workers docs (developers.cloudflare.com), Security Boulevard — "Validating JWTs at the Edge" (Nov 2025), Edge Computing Platform Comparison (wavesandalgorithms.com, 2025).*
