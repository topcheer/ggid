# GGID Architecture — C4 Model

System architecture visualized using the [C4 model](https://c4model.com/) with
[Mermaid](https://mermaid.js.org/) diagrams.

---

## Level 1: System Context

```mermaid
graph TB
    User[End User]
    Admin[IAM Administrator]
    App[Client Application<br/>SPA / Mobile / API]
    IdP[External IdP<br/>Google / GitHub / Azure AD]
    LDAP[LDAP / Active Directory]

    subgraph GGID Platform
        GW[GGID Gateway<br/>:8080]
    end

    SMTP[SMTP Server<br/>Email Delivery]
    SIEM[SIEM / Splunk<br/>Audit Forwarding]

    User -->|HTTPS| GW
    Admin -->|HTTPS| GW
    App -->|OAuth2/OIDC| GW
    GW -->|SAML/OIDC| IdP
    GW -->|LDAP bind| LDAP
    GW -->|SMTP| SMTP
    GW -->|NATS events| SIEM

    style GGID Platform fill:#e1f5fe,stroke:#0288d1
    style GW fill:#bbdefb,stroke:#0288d1
```

---

## Level 2: Container

```mermaid
graph TB
    subgraph Client
        Browser[Web Browser]
        Console[Admin Console<br/>Next.js 15 :3000]
    end

    subgraph GGID Services
        GW[API Gateway<br/>Go :8080<br/>JWT verify + rate limit + proxy]
        Auth[Auth Service<br/>Go :9001<br/>Login/Register/MFA/WebAuthn]
        Identity[Identity Service<br/>Go :8080<br/>User CRUD/SCIM]
        OAuth[OAuth Service<br/>Go :9005<br/>OIDC/SAML]
        Policy[Policy Service<br/>Go :8070<br/>RBAC+ABAC engine]
        Org[Org Service<br/>Go :8071<br/>Org tree/departments]
        Audit[Audit Service<br/>Go :8072<br/>Event query/export]
    end

    subgraph Infrastructure
        PG[(PostgreSQL 16<br/>RLS-enabled)]
        Redis[(Redis 7<br/>Sessions/rate limit)]
        NATS[NATS JetStream<br/>Audit event bus]
    end

    Browser -->|HTTP| Console
    Console -->|REST API| GW
    Browser -->|REST API| GW

    GW -->|HTTP proxy| Auth
    GW -->|HTTP proxy| Identity
    GW -->|HTTP proxy| OAuth
    GW -->|HTTP proxy| Policy
    GW -->|HTTP proxy| Org
    GW -->|HTTP proxy| Audit

    Auth --> PG
    Auth --> Redis
    Identity --> PG
    OAuth --> PG
    Policy --> PG
    Org --> PG
    Audit --> PG
    Audit --> NATS

    Auth -.->|publish events| NATS
    Identity -.->|publish events| NATS
    Policy -.->|publish events| NATS
    Org -.->|publish events| NATS

    NATS -.->|consume| Audit

    style GGID Services fill:#e8f5e9,stroke:#388e3c
    style Infrastructure fill:#fff3e0,stroke:#f57c00
```

---

## Level 3: Component — API Gateway

```mermaid
graph LR
    subgraph Gateway
        RL[Rate Limiter]
        CORS[CORS Handler]
        JWT[JWT Verifier<br/>JWKS cache 15min]
        Tenant[Tenant Injector<br/>X-Tenant-ID → query/body]
        Router[Router<br/>Path prefix match]
        Proxy[Reverse Proxy<br/>Keep-alive pool]
        CB[Circuit Breaker<br/>Per-backend]
        HC[Health Checker<br/>Weighted LB]
        Metrics[Prometheus Metrics]
        OTel[OTel Tracing]
    end

    Req[Incoming Request] --> RL
    RL -->|pass| CORS
    CORS --> JWT
    JWT -->|verified| Tenant
    JWT -->|public path| Tenant
    Tenant --> Router
    Router --> Proxy
    Proxy --> CB
    CB -->|healthy| HC
    CB -->|tripped| Err503[503 Service Unavailable]
    HC -->|forward| Backend[Backend Service]
    Backend --> Metrics
    Metrics --> Resp[Response]

    RL -->|exceeded| Err429[429 Too Many Requests]
    JWT -->|invalid| Err401[401 Unauthorized]

    style Gateway fill:#e1f5fe,stroke:#0288d1
    style Err429 fill:#ffebee,stroke:#c62828
    style Err401 fill:#ffebee,stroke:#c62828
    style Err503 fill:#ffebee,stroke:#c62828
```

---

## Level 3: Component — Auth Service

```mermaid
graph TB
    subgraph Auth Service
        Handler[HTTP Handler<br/>/api/v1/auth/*]
        Svc[Auth Service<br/>Business Logic]
        
        subgraph Provider Chain
            Local[Local Provider<br/>Argon2id password]
            LDAP[LDAP Provider<br/>AD bind + search]
            Social[Social Provider<br/>Google/GitHub/MS]
            WebAuthn[WebAuthn Provider<br/>FIDO2/Passkey]
        end
        
        JWT[JWT Issuer<br/>RS256 signing]
        MFA[MFA Engine<br/>TOTP/Email/WebAuthn]
        Hook[Hook Engine<br/>Pre/Post auth hooks]
        RateLimit[Rate Limiter<br/>Login 5/min]
    end

    Redis[(Redis<br/>Token blocklist)]
    PG[(PostgreSQL<br/>Users/Credentials)]
    NATS[NATS<br/>Audit events]
    SMTP[SMTP<br/>Email/MFA codes]

    Handler --> Svc
    Svc --> Local
    Svc --> LDAP
    Svc --> Social
    Svc --> WebAuthn
    Svc --> MFA
    Svc --> Hook
    Svc --> JWT
    
    Local --> PG
    WebAuthn --> PG
    JWT --> Redis
    MFA --> SMTP
    Svc -.-> NATS
    RateLimit --> Redis

    style Auth Service fill:#e8f5e9,stroke:#388e3c
    style Provider Chain fill:#f1f8e9,stroke:#558b2f
```

---

## Level 3: Component — Policy Engine

```mermaid
graph TB
    subgraph Policy Service
        Handler[HTTP Handler<br/>/api/v1/policies/*]
        Engine[Policy Engine<br/>RBAC + ABAC evaluator]
        
        subgraph Evaluation
            DenyCheck[1. Deny Rules<br/>Explicit deny check]
            RBAC[2. RBAC Check<br/>Roles → Permissions<br/>Wildcard match]
            ABAC[3. ABAC Check<br/>Attribute conditions<br/>JSON operators]
            Default[4. Default Action<br/>Configurable allow/deny]
        end
        
        RoleCache[Role Cache<br/>Redis 5min TTL]
        PolicyCache[Policy Cache<br/>In-memory]
    end

    PG[(PostgreSQL<br/>Roles/Policies)]
    Redis[(Redis<br/>Role cache)]

    Request[Permission Check<br/>user_id, resource, action] --> Handler
    Handler --> Engine
    Engine --> DenyCheck
    DenyCheck -->|deny| DenyResult[DENY]
    DenyCheck -->|no match| RBAC
    RBAC --> RoleCache
    RoleCache --> Redis
    RBAC -->|match| AllowResult[ALLOW candidate]
    RBAC -->|no match| ABAC
    ABAC --> PolicyCache
    ABAC -->|conditions pass| AllowResult
    ABAC -->|conditions fail| Default
    Default -->|deny-all| DenyResult
    Default -->|allow-all| AllowResult

    Engine --> PG

    style Policy Service fill:#e8f5e9,stroke:#388e3c
    style Evaluation fill:#f1f8e9,stroke:#558b2f
    style DenyResult fill:#ffebee,stroke:#c62828
    style AllowResult fill:#e8f5e9,stroke:#2e7d32
```

---

## Level 3: Component — Audit Pipeline

```mermaid
graph LR
    subgraph Publishers
        Auth[Auth Service]
        Ident[Identity Service]
        Policy[Policy Service]
        Org[Org Service]
    end

    subgraph NATS JetStream
        Stream[AUDIT-EVENTS Stream<br/>File-backed<br/>7-day retention<br/>1M max messages]
    end

    subgraph Consumers
        AuditSvc[Audit Service Consumer<br/>Batch insert to PG]
        SIEM[SIEM Consumer<br/>Forward to Splunk]
        SSE[Real-time SSE<br/>Dashboard streaming]
    end

    PG[(PostgreSQL<br/>audit_events table)]

    Auth -.->|PublishAsync| Stream
    Ident -.->|PublishAsync| Stream
    Policy -.->|PublishAsync| Stream
    Org -.->|PublishAsync| Stream

    Stream -->|Pull batch=100| AuditSvc
    Stream -->|Pull| SIEM
    Stream -->|Push| SSE

    AuditSvc -->|Bulk INSERT| PG
    AuditSvc -->|Ack| Stream

    style NATS JetStream fill:#fff3e0,stroke:#f57c00
    style Publishers fill:#e8f5e9,stroke:#388e3c
    style Consumers fill:#e1f5fe,stroke:#0288d1
```

---

## Deployment View

```mermaid
graph TB
    subgraph Docker Compose / Kubernetes
        LB[Load Balancer<br/>nginx / Ingress]
        
        subgraph Gateway Tier
            GW1[Gateway Replica 1]
            GW2[Gateway Replica 2]
        end
        
        subgraph Service Tier
            Auth1[Auth :9001]
            Ident1[Identity :8080]
            OAuth1[OAuth :9005]
            Pol1[Policy :8070]
            Org1[Org :8071]
            Aud1[Audit :8072]
        end
        
        subgraph Data Tier
            PG[(PostgreSQL<br/>Primary + Replica)]
            Redis[(Redis Cluster)]
            NATS[NATS Cluster<br/>3 nodes)]
        end
    end

    Client[Client Traffic] --> LB
    LB --> GW1
    LB --> GW2
    GW1 --> Auth1
    GW1 --> Ident1
    GW1 --> OAuth1
    GW1 --> Pol1
    GW1 --> Org1
    GW1 --> Aud1
    GW2 --> Auth1
    GW2 --> Ident1
    GW2 --> Pol1
    
    Auth1 --> PG
    Auth1 --> Redis
    Ident1 --> PG
    Pol1 --> PG
    Org1 --> PG
    Aud1 --> PG
    Aud1 --> NATS
    
    Auth1 -.-> NATS

    style Gateway Tier fill:#e1f5fe,stroke:#0288d1
    style Service Tier fill:#e8f5e9,stroke:#388e3c
    style Data Tier fill:#fff3e0,stroke:#f57c00
```

---

## Data Flow: Login Request

```mermaid
sequenceDiagram
    participant C as Client
    participant GW as Gateway
    participant Auth as Auth Service
    participant PG as PostgreSQL
    participant R as Redis
    participant NATS as NATS

    C->>GW: POST /api/v1/auth/login {username, password}
    
    GW->>GW: Rate limit check (5/min/IP)
    GW->>GW: JWT bypass (public path)
    GW->>GW: Tenant injection (X-Tenant-ID)
    
    GW->>Auth: Proxy request
    
    Auth->>PG: Lookup credential (tenant-scoped)
    PG-->>Auth: User record + password hash
    
    Auth->>Auth: Argon2id verify password
    
    alt Password valid
        Auth->>R: Store session metadata
        Auth->>Auth: Generate JWT (RS256)
        Auth->>NATS: Publish audit event (user.login, success)
        Auth-->>GW: 200 {access_token, refresh_token}
        GW-->>C: 200 {access_token, refresh_token}
    else Password invalid
        Auth->>NATS: Publish audit event (user.login_failed)
        Auth-->>GW: 401 Unauthorized
        GW-->>C: 401 Unauthorized
    end
```

---

## Data Flow: Permission Check

```mermaid
sequenceDiagram
    participant App as Application
    participant GW as Gateway
    participant Pol as Policy Service
    participant R as Redis
    participant PG as PostgreSQL

    App->>GW: POST /api/v1/policies/check {user_id, resource, action}
    
    GW->>GW: JWT verification
    GW->>Pol: Proxy request
    
    Pol->>R: Get cached roles for user
    alt Cache hit
        R-->>Pol: Roles list
    else Cache miss
        Pol->>PG: Query user_roles + role hierarchy
        PG-->>Pol: Roles with inherited permissions
        Pol->>R: Cache roles (5min TTL)
    end
    
    Pol->>Pol: 1. Check deny rules
    Pol->>Pol: 2. Check RBAC (wildcard match)
    Pol->>Pol: 3. Check ABAC conditions
    Pol->>Pol: 4. Apply default action
    
    alt Allowed
        Pol-->>GW: {allowed: true, reason: "..."}
        GW-->>App: 200 {allowed: true}
    else Denied
        Pol-->>GW: {allowed: false, reason: "..."}
        GW-->>App: 200 {allowed: false}
    end
```
