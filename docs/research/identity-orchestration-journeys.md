# Identity Orchestration: Configurable Authentication Journeys for GGID

> **Focus**: Adding a declarative, configurable authentication journey engine — enabling admins to build multi-step, branching auth flows without code changes (like Auth0 Actions, Okta Workflows, PingOne DaVinci, Keycloak Authentication Flows).
>
> **Author**: ggcxf (researcher) | **Date**: 2026-07-15 | **Status**: Research Complete

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [What is Identity Orchestration?](#2-what-is-identity-orchestration)
3. [Industry Landscape](#3-industry-landscape)
4. [GGID Current State Analysis](#4-ggid-current-state-analysis)
5. [Gap Analysis](#5-gap-analysis)
6. [Proposed Architecture: Journey Engine](#6-proposed-architecture-journey-engine)
7. [Journey Definition Language (JDL)](#7-journey-definition-language-jdl)
8. [Node Types and Action Catalog](#8-node-types-and-action-catalog)
9. [API Design](#9-api-design)
10. [State Machine and Execution Model](#10-state-machine-and-execution-model)
11. [Security Considerations](#11-security-considerations)
12. [Performance Considerations](#12-performance-considerations)
13. [Console UI Design](#13-console-ui-design)
14. [Competitive Differentiation](#14-competitive-differentiation)
15. [Migration Strategy](#15-migration-strategy)
16. [Implementation Backlog](#16-implementation-backlog)

---

## 1. Executive Summary

GGID currently hardcodes its authentication flow in Go code: password check → risk assessment → MFA challenge → token issuance. While configurable via settings (MFA enabled/disabled, risk thresholds), the actual **sequence and branching logic** of authentication steps cannot be customized by tenant administrators without code changes.

Modern IAM platforms (Auth0, Okta, Ping Identity, Keycloak) all offer configurable **authentication journey engines** that let admins define multi-step, branching flows visually or declaratively. This is the #1 enterprise feature request for IAM platforms in 2026.

**Recommendation**: Build a **Journey Engine** for GGID — a state-machine-based execution framework that:
1. Defines auth flows as declarative YAML/JSON (Journey Definition Language)
2. Supports branching, looping, parallel evaluation, and conditional steps
3. Integrates with existing components (password, MFA, risk assessment, hooks, LDAP, WebAuthn)
4. Provides a visual flow builder in the Console UI
5. Enables per-tenant, per-application, per-user-group journey customization

**Estimated effort**: 4 sprints for MVP (engine + JDL + API + basic nodes) + 2 sprints for Console visual builder.

---

## 2. What is Identity Orchestration?

Identity orchestration is the **governed sequencing of authentication, step-up checks, fraud signals, exception handling, and recovery paths** across a user's journey. It abstracts authentication flow logic from application code into a configurable, auditable, reusable framework.

### The Seven Pillars of Modern Orchestration

| Pillar | Description | GGID Status |
|--------|-------------|-------------|
| **1. Flow Fundamentals** | Auth flows as named, documented steps with typed inputs/outputs | Partially (hardcoded in AuthService) |
| **2. Schema as Backbone** | Typed data contracts between steps, versioned | Missing |
| **3. Invocation Mode** | UI mode (redirect, hosted page) vs API mode (headless) | Partially (UI mode only) |
| **4. Integration Models** | Embedded SDK, hosted page, API-first, server-to-server | Partially (Gateway proxy only) |
| **5. Cloud-Native** | Stateless engine, horizontally scalable | Yes (microservices) |
| **6. Hybrid Support** | On-premises connectors for legacy (LDAP, AD) | Yes (LDAP provider) |
| **7. Patterns** | Reusable journey templates (login, registration, recovery) | Missing |

### What a Journey Looks Like

A **journey** is a directed graph of **nodes** (actions) connected by **transitions** (conditions). When a user starts authentication, they enter the journey at its entry node and traverse edges until reaching a terminal state (success, denied, redirect).

```
                    ┌──────────────┐
                    │  ENTRY POINT │
                    │  (login)     │
                    └──────┬───────┘
                           │
                    ┌──────▼───────┐
                    │  Risk        │     ┌──────────────┐
                    │  Assessment  ├────►│  DENY        │ (score > 80)
                    │  Engine      │     │  (blocked)   │
                    └──────┬───────┘     └──────────────┘
                           │ (score < 30)
                    ┌──────▼───────┐
                    │  Password    │     ┌──────────────┐
                    │  Check       ├────►│  Account     │ (locked)
                    │              │     │  Lockout     │
                    └──────┬───────┘     └──────────────┘
                           │ (success)
                    ┌──────▼───────┐
                    │  MFA         │     ┌──────────────┐
                    │  Challenge   ├────►│  WebAuthn    │ (platform)
                    │  Orchestrator│     │  biometric   │
                    └──────┬───────┘     └──────────────┘
                           │ (verified)
                    ┌──────▼───────┐
                    │  Post-Login  │
                    │  Hooks       │
                    │  (webhooks)  │
                    └──────┬───────┘
                           │
                    ┌──────▼───────┐
                    │  Issue JWT   │
                    │  + Refresh   │
                    └──────┬───────┘
                           │
                    ┌──────▼───────┐
                    │  SUCCESS     │
                    │  (redirect)  │
                    └──────────────┘
```

### Key Properties of a Journey Engine

| Property | Description |
|----------|-------------|
| **Declarative** | Journeys defined in YAML/JSON, not code |
| **Versioned** | Each journey version is immutable; rollbacks are instant |
| **Per-tenant** | Each tenant has its own journeys |
| **Per-application** | Different apps can use different journeys |
| **Branching** | Conditional transitions based on context signals |
| **Auditable** | Every step execution is logged with timing and result |
| **Testable** | Dry-run mode with synthetic inputs |
| **Reusable** | Journey templates shared across tenants |

---

## 3. Industry Landscape

### Auth0 Actions (Okta)

**Architecture**: Serverless functions triggered at specific points in the auth pipeline.

**Key features**:
- **Trigger points**: `post-login`, `pre-user-registration`, `post-user-registration`, `post-change-password`, `send-phone-message`
- **Actions**: JavaScript functions running in Auth0's sandboxed runtime
- **Flows**: Visual flow builder connecting actions in sequence
- **Secrets**: Encrypted key-value store per action
- **Marketplace**: Pre-built actions for common integrations

**Strengths**: Developer-friendly, JavaScript SDK, rich marketplace
**Weaknesses**: Code-required (not purely declarative), tied to Auth0 ecosystem, no visual branching

### Okta Workflows

**Architecture**: No-code automation engine with drag-and-drop flow builder.

**Key features**:
- **Visual builder**: Drag-and-drop nodes connected with conditional edges
- **App integrations**: 7,000+ pre-built connectors
- **Logic nodes**: If/else, switch, loop, parallel, delay, retry
- **Data transformation**: JSON manipulation within flows
- **Event-driven**: Triggered by Okta events or external webhooks

**Strengths**: True no-code, massive connector ecosystem, visual branching
**Weaknesses**: Okta ecosystem lock-in, complex for simple flows, expensive at scale

### PingOne DaVinci (Ping Identity)

**Architecture**: No-code identity orchestration engine with visual flow builder.

**Key features**:
- **Flow designer**: Visual drag-and-drop canvas for identity journeys
- **Connectors**: Library of IdP, MFA, fraud detection, directory connectors
- **Adaptive access**: Risk-based branching within flows
- **User journey templates**: Pre-built flows for common scenarios
- **Policy-based**: Conditional logic driven by access policies

**Strengths**: Purpose-built for identity, strong connector library, visual branching
**Weaknesses**: Ping ecosystem, commercial-only, limited extensibility

### Keycloak Authentication Flows

**Architecture**: XML/JSON-defined flow with Java SPI extensions.

**Key features**:
- **Flow types**: Browser flow, Direct Grant flow, Registration flow, Reset credentials flow
- **Execution types**: Required, Alternative, Conditional, Disabled
- **Sub-flows**: Nested flow composition
- **Authenticator SPI**: Java interface for custom authenticators
- **Built-in authenticators**: Username/password, OTP, WebAuthn, Identity Provider Redirector, Conditional Steps

**Strengths**: Open-source, sub-flow composition, configurable execution requirements
**Weaknesses**: XML config not intuitive, requires Java for extensions, no visual builder (community plugins only)

### Curity Identity Server

**Architecture**: Powerful user journey orchestration with drag-and-drop designer.

**Key features**:
- **Visual designer**: Full drag-and-drop overview of authentication actions
- **Action types**: Authentication, data retrieval, data transformation, claims management
- **Template-based**: Reusable journey templates
- **Protocol-aware**: OAuth/OIDC-aware flow steps
- **API-driven**: All journeys manageable via REST API

**Strengths**: Most mature visual journey builder, protocol-aware
**Weaknesses**: Commercial (open-core), steep learning curve

### Comparison Matrix

| Feature | Auth0 Actions | Okta Workflows | PingOne DaVinci | Keycloak Flows | Curity | **GGID (proposed)** |
|---------|--------------|----------------|-----------------|----------------|--------|---------------------|
| **Declarative definition** | JavaScript | Visual | Visual | XML | Visual + JSON | **YAML + Visual** |
| **Visual builder** | Partial | Yes | Yes | No (community) | Yes | **Yes** |
| **Branching** | Code logic | Visual edges | Visual edges | Conditional | Visual edges | **YAML + Visual** |
| **Sub-flows** | Via actions | Yes | Yes | Yes | Yes | **Yes** |
| **Open source** | No | No | No | Yes | Open-core | **Yes (Apache 2.0)** |
| **Per-tenant** | Yes (stores) | Yes | Yes | Realms | Yes | **Yes** |
| **Risk integration** | Via action | Adaptive | Native | Via SPI | Via connector | **Native** |
| **MFA orchestration** | Via action | Via flow | Native | Via SPI | Via action | **Native** |
| **Dry-run/test** | Playground | Test mode | Debug mode | No | Simulator | **Yes** |
| **Custom nodes** | JavaScript | Connectors | Connectors | Java SPI | Connectors | **Webhook + Go plugin** |

---

## 4. GGID Current State Analysis

### Current Authentication Pipeline

GGID's authentication flow is implemented in `AuthService.Login()`:

```
1. Rate limit check (rateLimiter)
2. Credential lookup (identityClient.GetUserByEmail)
3. Password verification (Argon2id)
4. Account lockout check (accountLockout)
5. Risk assessment (AssessLoginRisk)
6. MFA challenge (if enabled + user has factors)
7. Token issuance (tokenService.IssueTokens)
8. Post-login hooks (HookManager)
```

### Existing Extensibility Points

| Component | What It Does | Limitation |
|-----------|-------------|------------|
| `HookManager` | Fires webhooks at pre/post login/register | Notification only — cannot block or redirect |
| `RiskAssessment` | Evaluates login risk (IP, device, time) | Hardcoded into login flow, not configurable |
| `StepUpAuth` | Challenge with additional factor | Fixed: WebAuthn or TOTP, no fallback chain |
| `MFAService` | Manages TOTP, WebAuthn factors | No orchestrator for factor selection logic |
| `AnomalyDetection` | Detects impossible travel, brute force | Hardcoded thresholds, no per-tenant tuning |
| `DeviceTracking` | Tracks known devices | Not integrated into flow decisions |

### What's Missing

1. **No configurable flow**: The sequence of steps is hardcoded in Go. Adding a new step (e.g., "check if user accepted terms of service") requires code changes + deployment.

2. **No branching logic**: Cannot express "if user is in Finance department, require WebAuthn; otherwise, TOTP is sufficient" without code.

3. **No per-application flows**: All applications use the same auth flow. Cannot have different flows for different OAuth clients.

4. **No visual builder**: Admins cannot design or modify flows without developer involvement.

5. **No journey versioning**: Cannot test a new flow and rollback if it breaks.

6. **No A/B testing**: Cannot run two flows in parallel to compare conversion rates.

7. **No journey templates**: Each tenant starts from scratch; no reusable patterns.

---

## 5. Gap Analysis

### Use Cases GGID Cannot Serve Today

| # | Use Case | Why It Fails | Journey Solution |
|---|----------|-------------|-----------------|
| 1 | "Finance users must use WebAuthn; others can use TOTP" | No department-based branching | Conditional node: `if dept == finance → WebAuthn else → TOTP` |
| 2 | "New users must accept ToS on first login" | No terms acceptance step | Conditional node: `if first_login → show_tos_page` |
| 3 | "High-risk logins trigger push notification MFA" | Risk result not wired to MFA selection | Risk → branch → push notification node |
| 4 | "Users from blocked countries are denied" | No geo-fencing in flow | Geo-IP check node → deny branch |
| 5 | "Progressive profiling: collect phone after 3rd login" | No login count tracking in flow | Counter node → conditional → profile collection |
| 6 | "Admin users require approval from manager" | No approval workflow in auth | Approval request node → async wait |
| 7 | "Different auth flow for mobile vs web" | Single flow for all clients | Per-client-type journey binding |
| 8 | "Custom fraud check via external API" | HookManager is notification-only | HTTP call node with allow/deny response |
| 9 | "Account recovery with identity verification" | No recovery journey | Dedicated recovery journey template |
| 10 | "Migration: try legacy DB first, fallback to local" | No fallback chain | Credential provider chain in journey |

---

## 6. Proposed Architecture: Journey Engine

### High-Level Architecture

```
                    ┌─────────────────────────────────────┐
                    │        Auth Service                 │
                    │                                     │
                    │   POST /api/v1/auth/login           │
                    │         │                           │
                    │         ▼                           │
                    │   ┌─────────────┐                   │
                    │   │ Journey     │                   │
                    │   │ Resolver    │                   │
                    │   │             │                   │
                    │   │ tenant_id → │                   │
                    │   │   journey   │                   │
                    │   │ client_id → │                   │
                    │   │   journey   │                   │
                    │   └──────┬──────┘                   │
                    │          │                          │
                    │          ▼                          │
                    │   ┌─────────────┐                   │
                    │   │ Journey     │◄──── Journey      │
                    │   │ Engine      │      Definition   │
                    │   │             │      (YAML/JSON)  │
                    │   │ State       │                   │
                    │   │ Machine     │                   │
                    │   └──────┬──────┘                   │
                    │          │                          │
                    │     ┌────┼────┐                     │
                    │     │    │    │                     │
                    │     ▼    ▼    ▼                     │
                    │  ┌─────┐┌────┐┌──────┐              │
                    │  │Node ││Node││Node  │              │
                    │  │Exec ││Exec││Exec  │              │
                    │  │     ││    ││      │              │
                    │  │pw   ││mfa ││risk  │              │
                    │  └─────┘└────┘└──────┘              │
                    │     │    │    │                     │
                    │     └────┼────┘                     │
                    │          │                          │
                    │          ▼                          │
                    │   ┌─────────────┐                   │
                    │   │ Token       │                   │
                    │   │ Issuance    │                   │
                    │   └─────────────┘                   │
                    └─────────────────────────────────────┘
```

### Components

#### Journey Definition Store

```sql
-- Journey definitions (versioned)
CREATE TABLE journey_definitions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL,
    name        VARCHAR(128) NOT NULL,       -- e.g., "default-login", "high-security-login"
    version     INT NOT NULL DEFAULT 1,
    trigger     VARCHAR(64) NOT NULL,        -- "login", "registration", "recovery", "step_up"
    definition  JSONB NOT NULL,              -- Parsed journey graph
    raw_text    TEXT NOT NULL,                -- Original YAML
    status      VARCHAR(32) NOT NULL,        -- "draft", "active", "archived"
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(tenant_id, name, version)
);

-- Journey bindings (which journey to use for which context)
CREATE TABLE journey_bindings (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL,
    trigger     VARCHAR(64) NOT NULL,        -- "login", "registration"
    client_id   VARCHAR(128),                -- OAuth client (null = all clients)
    user_group  VARCHAR(128),                -- User group filter (null = all users)
    journey_id  UUID NOT NULL REFERENCES journey_definitions(id),
    priority    INT DEFAULT 0,               -- Higher = evaluated first
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Journey execution log (audit trail)
CREATE TABLE journey_executions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    journey_id      UUID NOT NULL,
    journey_version INT NOT NULL,
    user_id         UUID,
    session_id      VARCHAR(256) NOT NULL,
    status          VARCHAR(32) NOT NULL,    -- "running", "completed", "failed", "abandoned"
    started_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at    TIMESTAMPTZ,
    steps           JSONB NOT NULL,           -- Array of executed steps with results
    error           TEXT
);

-- Journey session state (for multi-step flows)
CREATE TABLE journey_sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    execution_id    UUID NOT NULL REFERENCES journey_executions(id),
    current_node    VARCHAR(128) NOT NULL,
    context         JSONB NOT NULL,           -- Accumulated context data
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ NOT NULL      -- 15-minute default
);
```

#### Journey Engine

```go
// JourneyEngine evaluates a journey definition against a request context.
type JourneyEngine struct {
    store       JourneyStore
    nodeExecs   map[string]NodeExecutor  // registered by node type
    logger      JourneyLogger
}

// NodeExecutor is the interface for all journey node types.
type NodeExecutor interface {
    Type() string
    Execute(ctx context.Context, input *NodeInput) (*NodeOutput, error)
}

// NodeInput contains everything a node needs to execute.
type NodeInput struct {
    TenantID   uuid.UUID
    UserID     *uuid.UUID
    SessionID  string
    Context    map[string]any   // Accumulated journey context
    NodeConfig map[string]any   // Node-specific config from definition
}

// NodeOutput is the result of a node execution.
type NodeOutput struct {
    Result     NodeResult       // "continue", "redirect", "deny", "complete"
    NextNode   string           // Target node ID (for branching)
    Context    map[string]any   // Additions to journey context
    Redirect   *RedirectSpec    // If result == "redirect"
    Error      *NodeError       // If result == "deny" or error
}

type NodeResult string
const (
    ResultContinue  NodeResult = "continue"
    ResultRedirect  NodeResult = "redirect"
    ResultDeny      NodeResult = "deny"
    ResultComplete  NodeResult = "complete"
    ResultAsync     NodeResult = "async"  // Waiting for user action
)
```

---

## 7. Journey Definition Language (JDL)

A YAML-based declarative format for defining authentication journeys.

### Example: Default Login Journey

```yaml
name: default-login
version: 1
trigger: login
description: "Standard login with risk-based MFA"

nodes:
  # Entry point
  - id: entry
    type: entry
    next: risk_check

  # Risk assessment
  - id: risk_check
    type: risk_assessment
    config:
      deny_threshold: 80       # Score >= 80 → deny
      stepup_threshold: 30     # Score >= 30 → require MFA
    transitions:
      - condition: "score >= 80"
        next: blocked
      - condition: "score >= 30"
        next: mfa_challenge
      - condition: default
        next: issue_tokens

  # Password verification
  - id: password_check
    type: password_verify
    config:
      max_attempts: 5
      lockout_duration: 900
    transitions:
      - condition: "success"
        next: risk_check
      - condition: "locked"
        next: account_locked
      - condition: default
        next: password_retry

  # MFA challenge
  - id: mfa_challenge
    type: mfa_orchestrate
    config:
      preferred_factors: [webauthn, totp]
      fallback_factor: email_otp
      remember_device: true
      remember_duration: 2592000  # 30 days
    transitions:
      - condition: "verified"
        next: post_login_hooks
      - condition: "failed"
        next: mfa_retry
      - condition: "max_attempts"
        next: account_locked

  # Post-login hooks
  - id: post_login_hooks
    type: webhook
    config:
      event: post_login
      timeout: 5000
      fail_open: true            # Continue even if webhook fails
    next: issue_tokens

  # Token issuance
  - id: issue_tokens
    type: issue_tokens
    config:
      include_claims:
        - roles
        - permissions
        - department
    transitions:
      - condition: default
        next: success

  # Terminal states
  - id: success
    type: terminal
    config:
      action: redirect
      redirect_url: "{{client.redirect_uri}}"

  - id: blocked
    type: terminal
    config:
      action: error
      status: 403
      message: "Access denied due to high risk score"

  - id: account_locked
    type: terminal
    config:
      action: error
      status: 423
      message: "Account locked due to too many failed attempts"
```

### Example: High-Security Journey (Finance Department)

```yaml
name: finance-login
version: 1
trigger: login
description: "Enhanced security for Finance department"
binding:
  user_group: finance
  priority: 10                  # Evaluated before default-login

nodes:
  - id: entry
    type: entry
    next: device_check

  # Device posture check (new node type)
  - id: device_check
    type: device_posture
    config:
      require_managed: true     # Must be MDM-enrolled
      require_updated: true     # OS must be current
    transitions:
      - condition: "compliant"
        next: password_check
      - condition: default
        next: device_noncompliant

  - id: password_check
    type: password_verify
    transitions:
      - condition: "success"
        next: webauthn_required
      - condition: default
        next: password_retry

  # Mandatory WebAuthn (no fallback for Finance)
  - id: webauthn_required
    type: webauthn_challenge
    config:
      user_verification: required
      attachment: platform       # Must be built-in biometric
    transitions:
      - condition: "verified"
        next: approval_check
      - condition: "failed"
        next: webauthn_retry

  # Manager approval for first-time logins (new node type)
  - id: approval_check
    type: conditional
    config:
      condition: "login_count == 0"
    transitions:
      - condition: "true"
        next: manager_approval
      - condition: default
        next: issue_tokens

  - id: manager_approval
    type: approval_request
    config:
      approver_role: manager
      timeout: 3600              # 1 hour
      deny_on_timeout: true
    transitions:
      - condition: "approved"
        next: issue_tokens
      - condition: default
        next: approval_denied

  - id: issue_tokens
    type: issue_tokens
    config:
      include_claims:
        - roles
        - permissions
        - department
        - clearance_level
      token_ttl: 3600            # 1 hour (shorter for Finance)
    next: success

  # Terminal states
  - id: success
    type: terminal
    config:
      action: redirect

  - id: device_noncompliant
    type: terminal
    config:
      action: redirect
      redirect_url: "/device-enrollment"

  - id: approval_denied
    type: terminal
    config:
      action: error
      status: 403
      message: "Manager approval required for first login"
```

---

## 8. Node Types and Action Catalog

### Core Nodes (Built-in)

| Node Type | Description | Config Options |
|-----------|-------------|----------------|
| `entry` | Journey entry point | — |
| `terminal` | Terminal state (success/deny) | `action`, `redirect_url`, `status`, `message` |
| `password_verify` | Username/password check | `max_attempts`, `lockout_duration` |
| `risk_assessment` | Evaluate risk signals | `deny_threshold`, `stepup_threshold` |
| `mfa_orchestrate` | MFA factor selection | `preferred_factors`, `fallback_factor`, `remember_device` |
| `webauthn_challenge` | WebAuthn/passkey challenge | `user_verification`, `attachment` |
| `totp_challenge` | TOTP code verification | `digits`, `window` |
| `email_otp` | Email one-time password | `ttl`, `code_length` |
| `sms_otp` | SMS one-time password | `ttl`, `code_length` |
| `ldap_authenticate` | LDAP credential check | `server`, `base_dn`, `filter` |
| `social_login` | OAuth2 social provider | `provider`, `scopes` |
| `issue_tokens` | JWT + refresh token issuance | `include_claims`, `token_ttl`, `refresh_ttl` |
| `webhook` | Call external HTTP endpoint | `url`, `headers`, `timeout`, `fail_open` |
| `conditional` | Evaluate condition expression | `condition` (CEL expression) |
| `device_posture` | Check device compliance | `require_managed`, `require_updated` |
| `redirect` | Redirect to external URL | `url`, `return_param` |

### Advanced Nodes (Phase 2)

| Node Type | Description |
|-----------|-------------|
| `approval_request` | Async approval workflow (manager, admin) |
| `identity_proofing` | KYC/identity verification step |
| `progressive_profile` | Collect missing user attributes |
| `terms_acceptance` | Show terms of service, require acceptance |
| `consent_collection` | GDPR consent gathering |
| `geo_fence` | Geographic location check |
| `rate_limit` | Journey-level rate limiting |
| `step_up` | Trigger step-up authentication sub-journey |
| `sub_journey` | Call another journey as a node |
| `custom_plugin` | Execute Go plugin (WASM) |

### Condition Expression Language

Use **CEL (Common Expression Language)** for transition conditions:

```yaml
# Simple conditions
condition: "score >= 80"
condition: "success == true"
condition: "dept == 'finance'"

# Complex conditions
condition: "score >= 50 && new_device == true"
condition: "login_count == 0 || ip_changed == true"
condition: "time.hour >= 9 && time.hour <= 17"   # Business hours
condition: "country in ['US', 'CA', 'GB']"
condition: "device.managed && device.os_version >= '14.0'"
```

---

## 9. API Design

### Journey Management

```
# Create/update journey
PUT /api/v1/auth/journeys
Content-Type: application/json

{
    "name": "default-login",
    "trigger": "login",
    "description": "Standard login flow",
    "definition": "<YAML or JSON>",
    "status": "draft"
}

# List journeys
GET /api/v1/auth/journeys?tenant_id={tenant}&trigger=login

# Get specific journey version
GET /api/v1/auth/journeys/{name}/versions/{version}

# Activate a journey version
POST /api/v1/auth/journeys/{name}/versions/{version}/activate

# Bind journey to context
POST /api/v1/auth/journeys/{name}/bindings
{
    "trigger": "login",
    "client_id": "mobile-app",
    "user_group": "finance",
    "priority": 10
}
```

### Journey Execution

```
# Start a journey (internal, called by auth service)
POST /api/v1/auth/journeys/execute
{
    "trigger": "login",
    "tenant_id": "...",
    "session_id": "...",
    "context": {
        "username": "alice@example.com",
        "password": "***",
        "client_id": "web-app",
        "ip": "10.0.1.50",
        "user_agent": "Mozilla/5.0..."
    }
}

# Response (immediate completion or redirect for async step)
{
    "execution_id": "uuid",
    "status": "redirect",
    "redirect": {
        "url": "/auth/webauthn-challenge",
        "state": "encrypted_token"
    }
}

# Resume journey after async step (e.g., after MFA verification)
POST /api/v1/auth/journeys/execute/{execution_id}/resume
{
    "step_result": {
        "verified": true,
        "factor": "webauthn"
    }
}

# Get journey execution status
GET /api/v1/auth/journeys/executions/{execution_id}
```

### Dry-Run / Test

```
# Simulate a journey execution without real side effects
POST /api/v1/auth/journeys/{name}/dry-run
{
    "context": {
        "username": "test@example.com",
        "ip": "1.2.3.4",
        "risk_score": 45,
        "device": { "managed": true, "known": true }
    }
}

# Response: step-by-step trace
{
    "trace": [
        { "node": "entry", "result": "continue", "duration_ms": 0 },
        { "node": "password_check", "result": "continue", "duration_ms": 12 },
        { "node": "risk_check", "result": "continue", "score": 45, "duration_ms": 3 },
        { "node": "mfa_challenge", "result": "redirect", "factor": "webauthn", "duration_ms": 1 },
    ],
    "final_status": "redirect",
    "total_duration_ms": 16
}
```

---

## 10. State Machine and Execution Model

### Synchronous vs Asynchronous Nodes

| Node Type | Execution Mode | Duration | Example |
|-----------|---------------|----------|---------|
| `password_verify` | Synchronous | <100ms | Check password hash |
| `risk_assessment` | Synchronous | <50ms | Evaluate risk signals |
| `conditional` | Synchronous | <1ms | Evaluate CEL expression |
| `webauthn_challenge` | **Asynchronous** | User-dependent | Redirect to WebAuthn page |
| `totp_challenge` | **Asynchronous** | User-dependent | Redirect to TOTP input |
| `approval_request` | **Asynchronous** | Minutes-hours | Wait for manager approval |
| `email_otp` | **Asynchronous** | Seconds-minutes | Send email, wait for code |

### Execution Flow

```
1. Client sends login request → Auth Service
2. Auth Service resolves journey (by tenant, client, user group)
3. Journey Engine creates execution context
4. Engine executes nodes sequentially:
   a. For synchronous nodes: execute immediately, get result
   b. For async nodes: pause execution, return redirect to client
   c. Client completes challenge (e.g., enters TOTP)
   d. Client resumes execution with step result
5. Engine continues until terminal node
6. Terminal node returns final result (success/deny/redirect)
```

### State Persistence

For async (multi-step) journeys, state is persisted in Redis:

```
journey:session:{session_id} → {
    execution_id: "uuid",
    current_node: "mfa_challenge",
    context: { ... },
    expires_at: "2026-07-15T15:00:00Z"
}
```

TTL: 15 minutes (configurable per journey). On expiry, execution is marked `abandoned`.

### Error Handling

| Error Type | Behavior |
|-----------|----------|
| Node execution panic | Catch, log, continue to error node |
| Node timeout | After configured timeout, transition to timeout branch |
| Invalid transition | Log error, deny execution |
| Redis unavailable | Fall back to in-memory state (single-instance mode) |
| Journey definition invalid | Validation at create time; old version runs until new validated |

---

## 11. Security Considerations

### Journey Definition Security

- **Tenant isolation**: Each tenant can only create/modify/view their own journeys
- **Definition validation**: Schema validation at creation — invalid YAML rejected
- **Version immutability**: Published versions are immutable; changes create new version
- **Admin-only**: Journey management requires `journey:manage` permission
- **Audit trail**: All journey create/update/activate/delete logged to audit service

### Runtime Security

- **Sandboxed execution**: Custom webhook nodes execute in separate timeout context
- **No code injection**: JDL is declarative YAML + CEL expressions, not arbitrary code
- **Rate limiting**: Journey executions rate-limited per IP + per user
- **Context sanitization**: Journey context is sanitized before passing to external webhooks
- **Secret management**: Webhook secrets stored encrypted, never exposed in logs
- **Session binding**: Journey execution bound to original session; cannot be hijacked

### CEL Expression Safety

CEL (Common Expression Language) is used for conditions:
- **No side effects**: CEL is a pure expression language (no I/O, no mutation)
- **Bounded execution**: CEL has configurable timeout and memory limits
- **Type-safe**: CEL expressions validated against context schema at definition time

---

## 12. Performance Considerations

### Expected Latency

| Operation | Latency | Notes |
|-----------|---------|-------|
| Journey resolution (Redis cache) | <1ms | Cached per (tenant, trigger, client) |
| Journey resolution (DB lookup) | 2-5ms | Cache miss fallback |
| Synchronous node execution | <100ms | Password verify, risk assessment |
| Async node pause/resume | <5ms overhead | Redis state save/load |
| Complete synchronous journey | <200ms | 5-7 synchronous nodes |
| CEL expression evaluation | <0.1ms | Compiled and cached |

### Optimization Strategies

1. **Journey definition caching**: Cache parsed journey graph in Redis with 5-min TTL. Invalidate on version activation.

2. **Node executor pooling**: Node executors are stateless and pooled for reuse.

3. **CEL compilation cache**: Compile CEL expressions once, cache the AST for repeated evaluations.

4. **Batch context updates**: Accumulate context changes and persist once at async boundary.

5. **Short-circuit evaluation**: If a deny condition is met, immediately stop and return without evaluating remaining nodes.

### Scalability

- **Stateless engine**: Journey engine is stateless; all state in Redis. Horizontally scalable.
- **Redis cluster**: Session state partitioned by session_id; supports Redis Cluster.
- **Database load**: Journey definitions are read-heavy (cache hit >99%). Write only on journey create/update.
- **Estimated capacity**: 10K concurrent journey executions per engine instance (Redis-bound).

---

## 13. Console UI Design

### Journey Visual Builder

A drag-and-drop canvas in the Console for designing auth journeys:

```
┌─────────────────────────────────────────────────────────────┐
│  Journey Builder: default-login (v3)            [Save] [Publish] │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────┐                                                  │
│  │ Palette │     ┌──────────────────────────────────────┐    │
│  │         │     │                                      │    │
│  │ ▸ Entry │     │   ┌──────┐    ┌──────┐    ┌──────┐  │    │
│  │ ▸ Auth  │     │   │Entry │───►│Risk  │───►│Pass- │  │    │
│  │ ▸ MFA   │     │   │      │    │Check │    │word  │  │    │
│  │ ▸ Risk  │     │   └──────┘    └──┬───┘    └──┬───┘  │    │
│  │ ▸ Webhook│    │                  │ deny      │      │    │
│  │ ▸ Logic │     │             ┌────▼───┐  ┌────▼───┐  │    │
│  │ ▸ Token │     │             │ BLOCK  │  │  MFA   │  │    │
│  │ ▸ Exit  │     │             └────────┘  └────┬───┘  │    │
│  │         │     │                          ┌────▼───┐  │    │
│  │ Custom: │     │                          │ Tokens │  │    │
│  │ ▸ HTTP  │     │                          └────┬───┘  │    │
│  │ ▸ CEL   │     │                          ┌────▼───┐  │    │
│  │         │     │                          │SUCCESS │  │    │
│  │         │     │                          └────────┘  │    │
│  └─────────┘     └──────────────────────────────────────┘    │
│                                                               │
│  Properties: [Risk Check]                                     │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │ deny_threshold: [80]  stepup_threshold: [30]            │ │
│  └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

### Features

- **Drag-and-drop canvas**: Add nodes by dragging from palette
- **Node configuration panel**: Click a node to edit its config
- **Edge labeling**: Transition conditions shown on edges
- **Live validation**: Schema validation as you build
- **Dry-run panel**: Test with synthetic inputs, see step trace
- **Version history**: View diff between versions, rollback
- **Template gallery**: Start from pre-built journey templates
- **Binding manager**: Assign journeys to clients/user groups

---

## 14. Competitive Differentiation

### How Journey Engine Makes GGID Stand Out

| Platform | Journey Engine | Visual Builder | Open Source | Risk-Aware Branching |
|----------|---------------|----------------|-------------|---------------------|
| **GGID** (proposed) | **YAML + Visual** | **Yes** | **Yes (Apache 2.0)** | **Native** |
| Auth0 | JavaScript | Partial | No | Via action code |
| Okta | Workflows | Yes | No | Via flow nodes |
| Keycloak | XML config | No (community) | Yes | No native risk |
| Curity | JSON + Visual | Yes | Open-core | Via connectors |
| Clerk | No | No | No | No |
| Casdoor | No | No | Yes | No |

**Key differentiators**:
1. **Only open-source IAM** with YAML-defined + visual builder journey engine
2. **Native risk assessment** as a first-class journey node (not external integration)
3. **CEL-based conditions** — more expressive than Keycloak's XML conditions, safer than Auth0's JavaScript
4. **Journey templates** — shareable across the community
5. **Dry-run mode** — test journeys without affecting real users

---

## 15. Migration Strategy

### Phase 1: Default Journey (Zero-Config)

1. Ship a built-in `default-login` journey that replicates the current hardcoded flow
2. When no custom journey is defined, the engine uses the default
3. No behavior change — existing tenants see identical auth flow
4. All current hooks, risk assessment, MFA logic wrapped as node executors

### Phase 2: Opt-In Custom Journeys

1. Admins can create custom journeys via API or Console UI
2. Custom journeys override the default for specified triggers/clients/groups
3. Default journey remains as fallback
4. Shadow mode: run custom journey alongside default, log differences

### Phase 3: Journey-First Architecture

1. All auth flows are journey-driven (hardcoded path removed)
2. Default journey is just another journey definition (editable)
3. Per-tenant, per-application, per-group journey customization
4. Journey templates marketplace

---

## 16. Implementation Backlog

### P0 — Core Engine (3 sprints)

| # | Task | Description | Effort |
|---|------|-------------|--------|
| 1 | Journey definition store | PostgreSQL tables + repository | 3 days |
| 2 | Journey definition parser | Parse YAML JDL into internal graph | 4 days |
| 3 | Journey engine | State machine executor with node dispatch | 5 days |
| 4 | Core node executors | entry, password_verify, risk_assessment, mfa_orchestrate, issue_tokens, terminal | 4 days |
| 5 | CEL condition evaluator | Parse + compile + evaluate CEL expressions | 3 days |
| 6 | Session state persistence | Redis-backed session state for async journeys | 2 days |
| 7 | Auth service integration | Replace hardcoded Login() flow with engine dispatch | 3 days |
| 8 | Journey management API | CRUD for journeys, bindings, versions | 3 days |
| 9 | Execution API | Start, resume, status endpoints | 2 days |
| 10 | Unit tests | 90%+ coverage for engine, parser, nodes | 4 days |

### P1 — Enhanced Features (2 sprints)

| # | Task | Description | Effort |
|---|------|-------------|--------|
| 11 | Additional node executors | webauthn_challenge, totp_challenge, email_otp, webhook, device_posture | 4 days |
| 12 | Async node support | Pause/resume with redirect for MFA steps | 3 days |
| 13 | Sub-journey support | Call another journey as a node | 2 days |
| 14 | Dry-run / simulation mode | Execute with synthetic inputs, no side effects | 3 days |
| 15 | Journey templates | Pre-built templates (login, registration, recovery, step-up) | 2 days |
| 16 | Per-application binding | Different journeys for different OAuth clients | 1 day |
| 17 | Integration tests | End-to-end journey execution tests | 3 days |

### P2 — Console UI (2 sprints)

| # | Task | Description | Effort |
|---|------|-------------|--------|
| 18 | Journey list page | View all journeys with status, version | 2 days |
| 19 | YAML editor | Code editor with syntax highlighting + validation | 2 days |
| 20 | Visual flow builder | Drag-and-drop canvas (React Flow or similar) | 5 days |
| 21 | Node config panel | Property editor for each node type | 2 days |
| 22 | Dry-run UI | Input form + execution trace visualization | 2 days |
| 23 | Binding manager | Assign journeys to clients/groups | 1 day |
| 24 | Template gallery | Browse + import journey templates | 2 days |

### P3 — Advanced Features (Future)

| # | Task | Description |
|---|------|-------------|
| 25 | Approval workflows | Async approval request nodes (manager, admin) |
| 26 | Progressive profiling | Collect missing attributes during auth |
| 27 | A/B testing | Run two journeys in parallel, compare metrics |
| 28 | Custom WASM plugins | Load custom node executors as WebAssembly modules |
| 29 | Journey analytics | Conversion rates, drop-off points, step latency |
| 30 | Migration assistant | Convert existing hooks to journey nodes |

---

## References

- [Seven Pillars of Modern Identity Orchestration](https://identitypro.blog/seven-pillars-of-modern-identity-orchestration-dc315e2ed897)
- [Top 11 Identity Orchestration Tools for 2026](https://blog.gitguardian.com/top-identity-orchestration-tools/)
- [Auth0 Actions Documentation](https://auth0.com/docs/customize/actions/actions-overview)
- [PingOne DaVinci](https://www.pingidentity.com/en/resources/identity-fundamentals/identity-orchestration.html)
- [Keycloak Authentication Flows](https://www.keycloak.org/docs/latest/server_admin/#_authentication-flows)
- [Curity User Journey Orchestration](https://curity.io/product/user-journey-orchestration/)
- [IBM: What Is Identity Orchestration?](https://www.ibm.com/think/topics/identity-orchestration)
- [Descope: Identity Orchestration](https://www.descope.com/learn/post/identity-orchestration)
- [CEL (Common Expression Language)](https://github.com/google/cel-spec) — Google's expression language for conditions
- [NIST SP 800-63B](https://pages.nist.gov/800-63-3/sp800-63b.html) — Authenticator Assurance Levels
