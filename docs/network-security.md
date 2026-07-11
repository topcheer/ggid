# Network Security

> Network architecture, segmentation, firewall rules, mTLS planning, DNS security, and load balancer configuration for GGID deployments.

---

## Table of Contents

1. [Network Architecture](#network-architecture)
2. [Network Segmentation](#network-segmentation)
3. [Firewall Rules](#firewall-rules)
4. [DNS Security](#dns-security)
5. [Load Balancer Configuration](#load-balancer-configuration)
6. [mTLS Between Services](#mtls-between-services)
7. [VPC and Cloud Configuration](#vpc-and-cloud-configuration)
8. [DDoS Protection](#ddos-protection)

---

## Network Architecture

### Production Topology

```
Internet
    │
    ▼
┌──────────────────────────┐
│     CDN / WAF            │  ← Cloudflare, AWS WAF, Cloud Armor
│  (DDoS, WAF rules)       │
└───────────┬──────────────┘
            │
            ▼
┌──────────────────────────┐
│   Load Balancer          │  ← HAProxy, AWS ALB, GCP LB
│  (TLS termination)       │
└───────────┬──────────────┘
            │
    ┌───────┴────────┐
    │  DMZ Network   │
    │  (10.0.1.0/24) │
    │                │
    │  ┌──────────┐  │
    │  │ Gateway  │  │
    │  │ (:8080)  │  │
    │  └────┬─────┘  │
    └───────┼────────┘
            │
    ┌───────┴────────┐
    │  App Network   │
    │  (10.0.2.0/24) │
    │                │
    │  ┌───────────┐ │
    │  │ Identity  │ │
    │  │ Auth      │ │
    │  │ OAuth     │ │
    │  │ Policy    │ │
    │  │ Org       │ │
    │  │ Audit     │ │
    │  └─────┬─────┘ │
    └────────┼───────┘
             │
    ┌────────┴────────┐
    │  Data Network   │
    │  (10.0.3.0/24)  │
    │                 │
    │  ┌───────────┐  │
    │  │ PostgreSQL│  │
    │  │ Redis     │  │
    │  │ NATS      │  │
    │  │ OpenLDAP  │  │
    │  └───────────┘  │
    └─────────────────┘
```

---

## Network Segmentation

### Three-Tier Model

| Tier | Network | Services | Inbound From |
|------|---------|----------|-------------|
| **DMZ** | `10.0.1.0/24` | Gateway only | Load balancer (ports 8080) |
| **Application** | `10.0.2.0/24` | Identity, Auth, OAuth, Policy, Org, Audit | DMZ only (gateway) |
| **Data** | `10.0.3.0/24` | PostgreSQL, Redis, NATS, OpenLDAP | Application tier only |

### Segmentation Rules

1. Internet can ONLY reach DMZ (gateway)
2. DMZ can ONLY reach Application tier
3. Application tier can ONLY reach Data tier
4. Data tier has NO outbound internet access
5. No direct internet → Database path

---

## Firewall Rules

### iptables Example (Application Server)

```bash
# Allow loopback
iptables -A INPUT -i lo -j ACCEPT

# Allow established connections
iptables -A INPUT -m state --state ESTABLISHED,RELATED -j ACCEPT

# Allow SSH from management network only
iptables -A INPUT -p tcp --dport 22 -s 10.0.0.0/8 -j ACCEPT

# Allow HTTP from DMZ (gateway) only
iptables -A INPUT -p tcp --dport 8080 -s 10.0.1.0/24 -j ACCEPT

# Allow service-to-service within app tier
iptables -A INPUT -p tcp -s 10.0.2.0/24 -j ACCEPT

# Allow outbound to data tier only
iptables -A OUTPUT -p tcp -d 10.0.3.0/24 -j ACCEPT

# Drop everything else
iptables -A INPUT -j DROP
iptables -A OUTPUT -j DROP
```

### Kubernetes NetworkPolicy

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: gateway-policy
  namespace: ggid
spec:
  podSelector:
    matchLabels:
      app: ggid-gateway
  policyTypes:
  - Ingress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: ingress-nginx
    ports:
    - protocol: TCP
      port: 8080
```

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: data-tier-policy
  namespace: ggid
spec:
  podSelector:
    matchLabels:
      tier: data
  policyTypes:
  - Ingress
  ingress:
  - from:
    - podSelector:
        matchLabels:
          tier: app
    ports:
    - protocol: TCP
      port: 5432  # PostgreSQL
    - protocol: TCP
      port: 6379  # Redis
    - protocol: TCP
      port: 4222  # NATS
```

---

## DNS Security

### DNS Rebinding Prevention

The gateway includes Host header validation to prevent DNS rebinding attacks:

1. Check `Host` header against allowed hostnames
2. Reject requests with unexpected `Host` values
3. Prevents attacker from using their domain to reach internal services

Configuration:
```bash
ALLOWED_HOSTS=iam.example.com,api.example.com
```

### DNS Configuration for Services

```
iam.example.com           → Load Balancer (gateway)
api.example.com           → Load Balancer (gateway)
console.example.com       → Console (Next.js)
postgres.internal.example → PostgreSQL (internal only)
redis.internal.example    → Redis (internal only)
nats.internal.example     → NATS (internal only)
ldap.internal.example     → OpenLDAP (internal only)
```

---

## Load Balancer Configuration

### HAProxy Example

```haproxy
frontend https_front
    bind *:443 ssl crt /etc/ssl/ggid.pem
    mode http
    
    # Security headers
    http-response set-header Strict-Transport-Security "max-age=31536000; includeSubDomains"
    http-response set-header X-Content-Type-Options "nosniff"
    http-response set-header X-Frame-Options "DENY"
    
    # Rate limiting
    stick-table type ip size 100k expire 30s store http_req_rate(10s)
    http-request deny if { src_http_req_rate ge 100 }
    
    # Route to gateway
    default_backend gateway_backend

backend gateway_backend
    mode http
    option healthcheck
    server gateway1 10.0.1.10:8080 check
    server gateway2 10.0.1.11:8080 check
```

### Health Check Configuration

```
GET /healthz → 200 OK (liveness)
GET /healthz/deep → 200 OK (readiness, checks all backends)
```

---

## mTLS Between Services

### Current State

Internal service communication uses plaintext HTTP/gRPC. This is acceptable when:
- Services run on isolated network (VPC, Kubernetes pod network)
- Network access is restricted by firewall rules

### Planned: Service Mesh with mTLS

```
Gateway ←──mTLS──→ Identity
Gateway ←──mTLS──→ Auth
Gateway ←──mTLS──→ OAuth
Gateway ←──mTLS──→ Policy
Gateway ←──mTLS──→ Org
Gateway ←──mTLS──→ Audit

Services ←──mTLS──→ PostgreSQL
Services ←──mTLS──→ Redis
Services ←──mTLS──→ NATS
```

### Istio Configuration (Planned)

```yaml
apiVersion: security.istio.io/v1beta1
kind: PeerAuthentication
metadata:
  name: default
  namespace: ggid
spec:
  mtls:
    mode: STRICT
```

### PostgreSQL TLS

```bash
# Enable TLS in PostgreSQL
ssl = on
ssl_cert_file = '/etc/postgresql/server.crt'
ssl_key_file = '/etc/postgresql/server.key'

# Connection string
DATABASE_URL=postgres://ggid:pass@db:5432/ggid?sslmode=verify-full
```

---

## VPC and Cloud Configuration

### AWS VPC

```
VPC: 10.0.0.0/16

Subnets:
  DMZ:          10.0.1.0/24  (public, has internet gateway)
  Application:  10.0.2.0/24  (private)
  Data:         10.0.3.0/24  (private, most restricted)

Security Groups:
  gateway-sg:   Inbound 443 from ALB, Outbound to app-sg
  app-sg:       Inbound 8080-9005 from gateway-sg, Outbound to data-sg
  data-sg:      Inbound 5432, 6379, 4222 from app-sg, NO outbound
```

### GCP VPC

```
VPC: ggid-network (10.0.0.0/16)

Subnets:
  dmz-subnet:       10.0.1.0/24  (region: us-central1)
  app-subnet:       10.0.2.0/24  (region: us-central1)
  data-subnet:      10.0.3.0/24  (region: us-central1)

Firewall Rules:
  allow-lb-to-gw:   0.0.0.0/0 → 10.0.1.0/24 tcp:443
  allow-gw-to-app:  10.0.1.0/24 → 10.0.2.0/24 tcp:8080-9005
  allow-app-to-data: 10.0.2.0/24 → 10.0.3.0/24 tcp:5432,6379,4222
  deny-all-else:    Implicit deny
```

---

## DDoS Protection

### Layer 3/4 (Volumetric)

| Protection | Implementation |
|-----------|---------------|
| SYN flood | Cloud provider DDoS protection (AWS Shield, GCP Armor) |
| UDP flood | Cloud provider DDoS protection |
| Connection exhaustion | Load balancer connection limits |

### Layer 7 (Application)

| Protection | Implementation |
|-----------|---------------|
| HTTP flood | Gateway rate limiting (token bucket per IP) |
| Slowloris | Connection timeout (30s) |
Large payload | Body size limit (10MB) |
| Bot traffic | User-Agent filtering, CAPTCHA (planned) |

### Rate Limiting Tiers

```
Unauthenticated:  10 req/min per IP
Authenticated:    1000 req/min per user
Admin:            5000 req/min per user
Service Account:  10000 req/min per client
```

### CDN/WAF Rules

| Rule | Action |
|------|--------|
| Requests from known botnets | Block |
| SQL injection patterns | Block |
| XSS patterns | Block |
| Path traversal (`../`) | Block |
| Known CVE exploit signatures | Block |

---

*Last updated: 2025-07-11*
