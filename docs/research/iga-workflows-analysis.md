# Identity Governance and Administration (IGA) Workflows Analysis

> **Author:** GGID Research Team  
> **Date:** 2025-01  
> **Status:** Strategic Analysis  
> **Scope:** IGA capability gap analysis for GGID, competitive benchmarking against Keycloak 26

---

## Table of Contents

1. [What is IGA?](#1-what-is-iga)
2. [Keycloak 26 IGA Workflows](#2-keycloak-26-iga-workflows)
3. [Access Request Workflow](#3-access-request-workflow)
4. [Access Review / Certification](#4-access-review--certification)
5. [Segregation of Duties (SoD)](#5-segregation-of-duties-sod)
6. [Provisioning Lifecycle (JML)](#6-provisioning-lifecycle-jml)
7. [Compliance Reporting](#7-compliance-reporting)
8. [How GGID's RBAC+ABAC Relates](#8-how-ggids-rbacabac-relates)
9. [Build vs Buy vs Integrate](#9-build-vs-buy-vs-integrate)
10. [Gap Analysis & Recommendations](#10-gap-analysis--recommendations)

---

## 1. What is IGA?

Identity Governance and Administration (IGA) is the discipline of managing digital
identities across their **full lifecycle** — from creation through modification to
deletion — with built-in **governance controls** that ensure access remains appropriate,
auditable, and compliant.

### Three Pillars of IGA

| Pillar | Description | Key Capabilities |
|--------|-------------|-----------------|
| **Provisioning** | Automate account creation, modification, and deprovisioning across target systems | SCIM connectors, HR-driven JML, just-in-time provisioning, deprovisioning workflows |
| **Access Governance** | Continuously verify that access is correct and necessary | Access requests with approvals, access reviews/certifications, SoD enforcement, role mining |
| **Compliance** | Demonstrate to auditors that controls are effective | Audit trails, compliance reports (SOX, HIPAA, GDPR), evidence collection, scheduled attestations |

### Why IGA Is Different from Basic IAM

Basic IAM answers: **"Who are you, and what can you do?"** — authentication + authorization.

IGA answers a broader set of questions:

- **"Should this person have this access?"** (access governance — periodic review)
- **"Who approved this access, and when?"** (audit trail — accountability)
- **"Is this access combination dangerous?"** (SoD — risk mitigation)
- **"What happens when this person changes roles or leaves?"** (JML — lifecycle automation)
- **"Can you prove to our auditor that access controls are working?"** (compliance — evidence)

In practice, many organizations run **two separate systems**: an IAM platform (like Keycloak,
Auth0, or GGID) for authentication/authorization, and a dedicated IGA tool (like SailPoint
IdentityIQ, Saviynt, or Okta Identity Governance) for governance on top. The market trend is
toward **convergence** — IAM platforms building governance capabilities natively, eliminating
the need for a separate IGA product.

### Market Context

The IGA market was valued at approximately **$6.8 billion in 2024** and is projected to grow
at 12-15% CAGR, driven by:
- Regulatory pressure (SOX, GDPR, HIPAA, SOC 2)
- Cloud migration increasing identity sprawl
- Zero Trust adoption requiring continuous access verification
- Remote work expanding attack surfaces

Key players: **SailPoint** (~30% market share), **Saviynt** (~15%), **Okta Identity Governance**
(~12%), **Microsoft Entra ID Governance** (~10%), **Oracle OIG** (~8%).

---

## 2. Keycloak 26 IGA Workflows

Keycloak 26 introduced a significant new capability: **Workflows** — a governance framework
built natively into the IAM platform, eliminating the need for a separate IGA product for
many organizations.

### What Are Keycloak Workflows?

Keycloak Workflows provide a structured way to define and execute **multi-step identity
processes** that require human approval or automated decision points. The core workflow
types shipped in Keycloak 26:

1. **Access Request Workflows** — A user requests a role or resource. The request routes
   through an approval chain (manager → app owner → security team). Upon final approval,
   the role is automatically provisioned. Upon rejection or timeout, the requester is notified.

2. **Manager Approval Workflows** — When a user is assigned to a new group or role, the
   system can require their manager to approve. This integrates with Keycloak's existing
   group/role hierarchy — the manager is resolved from the org structure.

3. **Access Review Campaigns** — Periodic campaigns where managers or application owners
   review who has access to their resources. Each reviewer sees a list of users and their
   access, and can certify (keep), modify (change scope), or revoke (remove) each assignment.

4. **Certification Campaigns** — Formal campaigns tied to compliance deadlines (e.g.,
   quarterly SOX access review). Track completion rates, send reminders, and escalate
   incomplete reviews to the next-level manager.

### How Workflows Integrate with RBAC/ABAC

Keycloak Workflows are **not a separate system** — they operate on the same roles, groups,
and permissions that Keycloak's RBAC and ABAC engines use:

- A workflow can **request** a role assignment → the role is evaluated by the existing
  RBAC engine once provisioned.
- ABAC policies can **trigger** workflows — e.g., a policy that says "if attribute
  `clearance_level < 3` and resource is classified, require manager approval."
- Workflow outcomes are **audited** through the existing event system.

### Go-Equivalent Design for GGID

GGID should implement IGA workflows as a **new microservice** (`services/governance/`)
that sits alongside the existing policy service:

```
┌─────────────────────────────────────────────────────┐
│                  API Gateway (:8080)                 │
│          /api/v1/governance/*                        │
├──────────────┬──────────────────────────────────────┤
│  Governance  │  Policy Service (RBAC + ABAC)         │
│  Service     │  ┌─────────────────────────────────┐  │
│  (NEW)       │  │  Evaluator.Check()              │  │
│              │  │  Role hierarchy + inheritance    │  │
│  ┌────────┐  │  │  ABAC conditions engine          │  │
│  │Workflow│←→│  └─────────────────────────────────┘  │
│  │ Engine │  │                                       │
│  ├────────┤  │  Identity Service                     │
│  │Campaign│  │  ┌─────────────────────────────────┐  │
│  │Manager │  │  │  User CRUD, SCIM 2.0            │  │
│  ├────────┤  │  └─────────────────────────────────┘  │
│  │SoD     │  │                                       │
│  │Checker │  │  Audit Service                        │
│  ├────────┤  │  ┌─────────────────────────────────┐  │
│  │JML     │  │  │  NATS → Event Store             │  │
│  │Engine  │  │  └─────────────────────────────────┘  │
│  └────────┘  │                                       │
└──────────────┴───────────────────────────────────────┘
```

The governance service would:
- Consume events from Identity and Policy services (role changes, user lifecycle events)
- Maintain workflow state in PostgreSQL with ACID transactions
- Use NATS JetStream for async notifications (approval emails, reminders)
- Expose REST APIs for the Admin Console to create/manage workflows

---

## 3. Access Request Workflow

The access request workflow is the most commonly used IGA process. It provides a
**self-service** mechanism for users to request access, with governance through approval.

### Workflow Design

```
User submits request
    │
    ▼
┌─────────────┐     ┌──────────────┐     ┌──────────────┐
│  Submitted   │────▶│  Pending     │────▶│  Approved    │
│  (created)   │     │  Approval    │     │  (provision) │
└─────────────┘     └──────┬───────┘     └──────────────┘
                           │
                     ┌─────┴─────┐
                     │           │
                     ▼           ▼
               ┌──────────┐ ┌──────────┐
               │ Escalated │ │ Rejected │
               │ (timeout) │ │          │
               └────┬─────┘ └──────────┘
                    │
                    ▼
              ┌──────────────────┐
              │ Auto-Approve or   │
              │ Route to Backup   │
              └──────────────────┘
```

### States and Transitions

| State | Trigger | Next State | Actor |
|-------|---------|-----------|-------|
| `submitted` | User creates request | `pending_approval` | System (auto-transition) |
| `pending_approval` | Assigned to approver | `approved` / `rejected` / `escalated` | Approver / Timer |
| `approved` | Final approval received | `provisioning` | System (auto-transition) |
| `provisioning` | Role assigned via Policy service | `completed` | System |
| `completed` | Role successfully provisioned | (terminal) | — |
| `rejected` | Any approver rejects | (terminal) | — |
| `escalated` | SLA timeout exceeded | `pending_approval` (new approver) | Timer |
| `cancelled` | User withdraws request | (terminal) | Requester |

### Go State Machine Design

```go
package governance

// RequestStatus represents the current state of an access request.
type RequestStatus string

const (
    StatusSubmitted       RequestStatus = "submitted"
    StatusPendingApproval RequestStatus = "pending_approval"
    StatusApproved        RequestStatus = "approved"
    StatusProvisioning    RequestStatus = "provisioning"
    StatusCompleted       RequestStatus = "completed"
    StatusRejected        RequestStatus = "rejected"
    StatusEscalated       RequestStatus = "escalated"
    StatusCancelled       RequestStatus = "cancelled"
)

// AccessRequest is the core domain entity for the access request workflow.
type AccessRequest struct {
    ID          uuid.UUID      `json:"id"`
    TenantID    uuid.UUID      `json:"tenant_id"`
    RequesterID uuid.UUID      `json:"requester_id"`
    RoleID      uuid.UUID      `json:"role_id"`
    ScopeType   string         `json:"scope_type"` // global, org, dept, team
    ScopeID     uuid.UUID      `json:"scope_id"`
    Reason      string         `json:"reason"`
    Status      RequestStatus  `json:"status"`
    Approvals   []ApprovalStep `json:"approvals"`
    CreatedAt   time.Time      `json:"created_at"`
    UpdatedAt   time.Time      `json:"updated_at"`
    ExpiresAt   time.Time      `json:"expires_at"` // SLA deadline
}

// ApprovalStep represents one level in a multi-level approval chain.
type ApprovalStep struct {
    ID          uuid.UUID  `json:"id"`
    RequestID   uuid.UUID  `json:"request_id"`
    Level       int        `json:"level"` // 1 = first approver, 2 = second, etc.
    ApproverID  uuid.UUID  `json:"approver_id"`
    Decision    string     `json:"decision"` // pending, approved, rejected
    Comment     string     `json:"comment"`
    DecidedAt   *time.Time `json:"decided_at"`
    SLADeadline time.Time  `json:"sla_deadline"`
}

// WorkflowEngine manages the full lifecycle of access requests.
type WorkflowEngine struct {
    requestRepo RequestRepository
    policySvc   PolicyClient  // calls Policy service for provisioning
    notifier    Notifier      // email/push notifications
    orgSvc      OrgClient     // resolve managers from org hierarchy
}

// SubmitRequest creates a new access request and routes it to the first approver.
func (e *WorkflowEngine) SubmitRequest(ctx context.Context, req *AccessRequest) error {
    req.ID = uuid.New()
    req.Status = StatusSubmitted
    req.CreatedAt = time.Now().UTC()

    // Build approval chain based on the requested role and scope.
    chain, err := e.buildApprovalChain(ctx, req)
    if err != nil {
        return fmt.Errorf("build approval chain: %w", err)
    }
    req.Approvals = chain

    // Auto-approve if no approval chain is needed (e.g., low-risk roles).
    if len(chain) == 0 {
        return e.autoApprove(ctx, req)
    }

    req.Status = StatusPendingApproval
    if err := e.requestRepo.Create(ctx, req); err != nil {
        return err
    }

    // Notify the first approver.
    return e.notifier.NotifyApprovalRequested(ctx, chain[0].ApproverID, req)
}

// ProcessApproval records an approver's decision and advances the workflow.
func (e *WorkflowEngine) ProcessApproval(ctx context.Context, stepID uuid.UUID, decision, comment string) error {
    step, err := e.requestRepo.GetApprovalStep(ctx, stepID)
    if err != nil {
        return err
    }

    now := time.Now().UTC()
    step.Decision = decision
    step.Comment = comment
    step.DecidedAt = &now

    if decision == "rejected" {
        return e.requestRepo.UpdateRequestStatus(ctx, step.RequestID, StatusRejected)
    }

    // Check if more approval levels remain.
    req, err := e.requestRepo.GetRequest(ctx, step.RequestID)
    if err != nil {
        return err
    }

    allApproved := true
    for i := range req.Approvals {
        if req.Approvals[i].Level > step.Level && req.Approvals[i].Decision == "pending" {
            allApproved = false
            // Notify next approver.
            e.notifier.NotifyApprovalRequested(ctx, req.Approvals[i].ApproverID, req)
            break
        }
    }

    if allApproved {
        // All levels approved — provision the role.
        if err := e.provision(ctx, req); err != nil {
            return fmt.Errorf("provision role: %w", err)
        }
    }

    return e.requestRepo.UpdateApprovalStep(ctx, step)
}

// CheckSLAs runs as a periodic job to escalate timed-out approvals.
func (e *WorkflowEngine) CheckSLAs(ctx context.Context) error {
    pending, err := e.requestRepo.ListPendingApprovalsPastSLA(ctx)
    if err != nil {
        return err
    }

    for _, step := range pending {
        // Escalate to the next-level manager.
        backupApprover, err := e.orgSvc.GetManager(ctx, step.ApproverID)
        if err != nil {
            continue
        }
        step.ApproverID = backupApprover
        step.SLADeadline = time.Now().UTC().Add(48 * time.Hour)
        e.requestRepo.UpdateApprovalStep(ctx, step)
        e.notifier.NotifyEscalation(ctx, backupApprover, step)
    }
    return nil
}
```

### Multi-Level Approval Chains

The approval chain is resolved dynamically based on:

1. **Role sensitivity** — System roles require security team approval; department roles
   require only the department head.
2. **Scope** — Global scope requires higher-level approval than team scope.
3. **Risk score** — ABAC conditions (e.g., access to financial systems) trigger
   additional approval levels.
4. **Org hierarchy** — The requester's manager is resolved from the Org service's
   LTREE-based org tree.

---

## 4. Access Review / Certification

Access reviews (also called **access recertification** or **user access reviews**) are
the process of periodically verifying that existing access assignments are still
appropriate. This is a critical control for SOX, HIPAA, and ISO 27001 compliance.

### Review Types

| Type | Reviewer | What's Reviewed | Frequency |
|------|---------|-----------------|-----------|
| **Manager Review** | Direct manager | All roles/permissions of their direct reports | Quarterly |
| **Application Owner Review** | App/resource owner | All users who have access to their application | Semi-annual |
| **Role Owner Review** | Role owner (often IT) | All users assigned to a specific role | Annual |
| **Privileged Access Review** | Security team | All admin/elevated privilege assignments | Monthly |
| **SoD Review** | Compliance team | All detected SoD violations for remediation | Quarterly |

### Campaign Lifecycle

```
1. Admin creates campaign (scope, reviewers, deadline)
       │
2. System generates review items (user-access pairs)
       │
3. Reviewers notified → review each item:
       │     ├─ Certify (keep access)
       │     ├─ Revoke (remove access)
       │     └─ Modify (change scope/role)
       │
4. Reminders sent to incomplete reviewers (T-7, T-3, T-1 days)
       │
5. Campaign deadline reached:
       │     ├─ Completed → generate report
       │     └─ Incomplete → escalate to next-level manager
       │
6. Revocations executed automatically
       │
7. Campaign archived for audit evidence
```

### Go Campaign Manager

```go
package governance

// Campaign represents a periodic access review cycle.
type Campaign struct {
    ID          uuid.UUID      `json:"id"`
    TenantID    uuid.UUID      `json:"tenant_id"`
    Name        string         `json:"name"`
    Type        string         `json:"type"` // manager_review, app_owner, role_owner
    Status      string         `json:"status"` // draft, active, completed, archived
    ScopeOrgID  uuid.UUID      `json:"scope_org_id"` // which org tree to review
    Deadline    time.Time      `json:"deadline"`
    CreatedAt   time.Time      `json:"created_at"`
    Items       []ReviewItem   `json:"items,omitempty"`
}

// ReviewItem is a single user-access pair to be reviewed.
type ReviewItem struct {
    ID           uuid.UUID  `json:"id"`
    CampaignID   uuid.UUID  `json:"campaign_id"`
    UserID       uuid.UUID  `json:"user_id"`
    ReviewerID   uuid.UUID  `json:"reviewer_id"`
    RoleID       uuid.UUID  `json:"role_id"`
    Decision     string     `json:"decision"` // pending, certify, revoke, modify
    Comment      string     `json:"comment"`
    DecidedAt    *time.Time `json:"decided_at"`
}

// CampaignManager handles campaign creation, execution, and completion.
type CampaignManager struct {
    campaignRepo CampaignRepository
    policySvc    PolicyClient
    orgSvc       OrgClient
    notifier     Notifier
}

// GenerateCampaign auto-creates review items by querying all active
// role assignments within the campaign scope.
func (m *CampaignManager) GenerateCampaign(ctx context.Context, c *Campaign) error {
    // Fetch all user-role assignments in scope.
    assignments, err := m.policySvc.ListUserRolesInScope(ctx, c.ScopeOrgID)
    if err != nil {
        return fmt.Errorf("fetch role assignments: %w", err)
    }

    for _, a := range assignments {
        // Resolve the reviewer based on campaign type.
        var reviewerID uuid.UUID
        switch c.Type {
        case "manager_review":
            mgr, err := m.orgSvc.GetManager(ctx, a.UserID)
            if err != nil {
                continue // skip if no manager
            }
            reviewerID = mgr
        case "app_owner":
            reviewerID = a.ResourceOwnerID
        case "role_owner":
            reviewerID = a.RoleOwnerID
        }

        c.Items = append(c.Items, ReviewItem{
            ID:         uuid.New(),
            CampaignID: c.ID,
            UserID:     a.UserID,
            ReviewerID: reviewerID,
            RoleID:     a.RoleID,
            Decision:   "pending",
        })
    }

    c.Status = "active"
    return m.campaignRepo.Create(ctx, c)
}

// CompleteCampaign processes all decisions: certifies keep access,
// revocations are executed through the Policy service.
func (m *CampaignManager) CompleteCampaign(ctx context.Context, campaignID uuid.UUID) error {
    items, err := m.campaignRepo.ListItems(ctx, campaignID)
    if err != nil {
        return err
    }

    for _, item := range items {
        switch item.Decision {
        case "revoke":
            err := m.policySvc.RevokeRole(ctx, item.UserID, item.RoleID)
            if err != nil {
                log.Printf("revoke role %s for user %s: %v", item.RoleID, item.UserID, err)
            }
        case "modify":
            // Modify scope — call policy service with updated scope.
            err := m.policySvc.ModifyRoleScope(ctx, item.UserID, item.RoleID, item.NewScope)
            if err != nil {
                log.Printf("modify role scope: %v", err)
            }
        case "certify":
            // No action — access stays as-is. Record certification for audit.
        case "pending":
            // Incomplete — escalate.
            m.notifier.NotifyEscalation(ctx, item.ReviewerID, item)
        }
    }

    return m.campaignRepo.UpdateStatus(ctx, campaignID, "completed")
}
```

### Scheduling

Campaigns should be **schedulable** via cron-like expressions. GGID can leverage its
existing infrastructure (NATS JetStream for delayed messages, or a simple Go ticker):
- Quarterly manager reviews: `0 0 1 */3 *` (first day of every quarter)
- Monthly privileged access reviews: `0 0 1 * *` (first of every month)

---

## 5. Segregation of Duties (SoD)

SoD prevents **toxic combinations** — scenarios where a single user holds two or more
roles that, combined, create a fraud or security risk. The classic example: the same
person who creates vendors should not also approve payments.

### Why SoD Matters

Without SoD enforcement, an IAM system can technically allow any role combination.
A user might accumulate roles over time (role creep) through moves, temporary
assignments, or lax approval processes. SoD provides a **safety net** that catches
dangerous combinations before they're provisioned.

### Common SoD Rules

| Rule | Risk | Example |
|------|------|---------|
| Vendor create + Payment approve | Fraud | Create fake vendor, approve payment to self |
| User create + User delete | Privilege abuse | Create backdoor account, then cover tracks |
| Config change + Config approve | Unauthorized changes | Make risky config change, self-approve |
| Data read + Data export | Data exfiltration | Read sensitive data, export to external system |
| Code deploy + Code approve | Supply chain attack | Push malicious code, self-approve deployment |

### SoD Policy Definition

An SoD policy defines **conflicting role pairs** (or role-permission pairs) that
cannot coexist for a single user:

```go
package governance

// SoDPolicy defines a set of roles that cannot be held simultaneously.
type SoDPolicy struct {
    ID          uuid.UUID `json:"id"`
    TenantID    uuid.UUID `json:"tenant_id"`
    Name        string    `json:"name"`
    Description string    `json:"description"`
    Severity    string    `json:"severity"` // critical, high, medium, low
    Conflicts   []RoleConflict `json:"conflicts"`
}

// RoleConflict defines two roles that conflict with each other.
type RoleConflict struct {
    RoleAID uuid.UUID `json:"role_a_id"`
    RoleBID uuid.UUID `json:"role_b_id"`
    Reason  string    `json:"reason"` // human-readable explanation
}

// SoDChecker evaluates whether a user's role set violates any SoD policy.
type SoDChecker struct {
    policyRepo SoDPolicyRepository
    roleReader RoleReader
}

// CheckUser evaluates all SoD policies for a given user.
// Returns a list of violations (empty if no violations).
func (c *SoDChecker) CheckUser(ctx context.Context, userID uuid.UUID) ([]SoDViolation, error) {
    // Get the user's current roles.
    userRoles, err := c.roleReader.GetUserRoles(ctx, userID)
    if err != nil {
        return nil, err
    }

    roleIDSet := make(map[uuid.UUID]bool)
    for _, ur := range userRoles {
        roleIDSet[ur.RoleID] = true
    }

    // Evaluate all SoD policies for the tenant.
    policies, err := c.policyRepo.ListAll(ctx)
    if err != nil {
        return nil, err
    }

    var violations []SoDViolation
    for _, policy := range policies {
        for _, conflict := range policy.Conflicts {
            if roleIDSet[conflict.RoleAID] && roleIDSet[conflict.RoleBID] {
                violations = append(violations, SoDViolation{
                    UserID:     userID,
                    PolicyID:   policy.ID,
                    PolicyName: policy.Name,
                    RoleAID:    conflict.RoleAID,
                    RoleBID:    conflict.RoleBID,
                    Severity:   policy.Severity,
                    Reason:     conflict.Reason,
                })
            }
        }
    }

    return violations, nil
}

// CheckBeforeProvision evaluates SoD policies BEFORE a role is assigned.
// This is the enforcement point — if a violation is detected, provisioning
// is blocked (or requires a risk acceptance exception).
func (c *SoDChecker) CheckBeforeProvision(ctx context.Context, userID, proposedRoleID uuid.UUID) (*SoDCheckResult, error) {
    // Get existing roles.
    userRoles, err := c.roleReader.GetUserRoles(ctx, userID)
    if err != nil {
        return nil, err
    }

    // Simulate adding the proposed role.
    roleIDSet := make(map[uuid.UUID]bool)
    roleIDSet[proposedRoleID] = true
    for _, ur := range userRoles {
        roleIDSet[ur.RoleID] = true
    }

    // Check all SoD policies.
    policies, err := c.policyRepo.ListAll(ctx)
    if err != nil {
        return nil, err
    }

    var violations []SoDViolation
    for _, policy := range policies {
        for _, conflict := range policy.Conflicts {
            if roleIDSet[conflict.RoleAID] && roleIDSet[conflict.RoleBID] {
                violations = append(violations, SoDViolation{
                    UserID:     userID,
                    PolicyID:   policy.ID,
                    PolicyName: policy.Name,
                    RoleAID:    conflict.RoleAID,
                    RoleBID:    conflict.RoleBID,
                    Severity:   policy.Severity,
                    Reason:     conflict.Reason,
                })
            }
        }
    }

    if len(violations) > 0 {
        return &SoDCheckResult{
            Allowed:    false,
            Violations: violations,
        }, nil
    }

    return &SoDCheckResult{Allowed: true}, nil
}

type SoDCheckResult struct {
    Allowed    bool
    Violations []SoDViolation
}

type SoDViolation struct {
    UserID     uuid.UUID
    PolicyID   uuid.UUID
    PolicyName string
    RoleAID    uuid.UUID
    RoleBID    uuid.UUID
    Severity   string
    Reason     string
}
```

### Enforcement Points

SoD checking should occur at **three** points:

1. **Preventive** (at provisioning time) — Block the role assignment if a critical SoD
   violation is detected. Require a documented exception for high-severity violations.
2. **Detective** (periodic scan) — Run a batch job that checks all users for SoD
   violations. Generate alerts for the compliance team.
3. **Corrective** (during access reviews) — Flag SoD violations in access review
   campaigns so reviewers can revoke one of the conflicting roles.

---

## 6. Provisioning Lifecycle (JML)

The **Joiner-Mover-Leaver (JML)** model describes the three key events in an identity's
lifecycle. Each event triggers a set of provisioning and deprovisioning actions.

### Joiner (Onboarding)

When a new employee joins:

1. HR system creates the employee record (via HR webhook or manual entry).
2. GGID Identity service creates the user account.
3. JML engine assigns **baseline roles** based on department, job title, or employment type.
4. Optional: manager approval for any non-baseline access.
5. Accounts are provisioned in downstream systems via SCIM.

**Trigger:** HR webhook (`POST /api/v1/governance/jml/joiner`) or SCIM `User.create`.

### Mover (Role Change)

When an employee changes roles (internal transfer, promotion):

1. HR system updates the employee record (new department, title, manager).
2. JML engine detects the change.
3. **Re-provisioning:** Assign roles for the new position.
4. **De-provisioning:** Revoke roles specific to the old position.
5. Manager review for any access that should transition (keep temporarily vs. revoke).
6. Accounts in downstream systems updated via SCIM PATCH.

**Trigger:** HR webhook (`POST /api/v1/governance/jml/mover`) or SCIM `User.update` with
changed attributes.

### Leaver (Offboarding)

When an employee leaves:

1. HR system sets termination date.
2. JML engine triggers immediate or scheduled deprovisioning.
3. **All roles revoked** — user_roles table rows deleted for this user.
4. User account **disabled** (not deleted — data retention policies).
5. Sessions **terminated** — revoke all active JWT refresh tokens.
6. Accounts **deprovisioned** from downstream systems via SCIM.
7. Data **archived** per retention policy.
8. Audit trail preserved (immutable — user_id references remain for compliance).

**Trigger:** HR webhook (`POST /api/v1/governance/jml/leaver`) or scheduled job that
checks termination dates.

### Go JML Engine

```go
package governance

// JMLEvent represents a joiner, mover, or leaver event.
type JMLEvent struct {
    ID         uuid.UUID `json:"id"`
    Type       string    `json:"type"` // joiner, mover, leaver
    UserID     uuid.UUID `json:"user_id"`
    TenantID   uuid.UUID `json:"tenant_id"`
    Department string    `json:"department"`
    JobTitle   string    `json:"job_title"`
    ManagerID  uuid.UUID `json:"manager_id"`
    Payload    map[string]any `json:"payload"` // HR-specific fields
    CreatedAt  time.Time `json:"created_at"`
}

// JMLEngine processes joiner-mover-leaver events and orchestrates provisioning.
type JMLEngine struct {
    eventRepo   JMLEventRepository
    policySvc   PolicyClient
    identitySvc IdentityClient
    notifier    Notifier
    rules       []ProvisioningRule
}

// ProvisioningRule maps HR attributes to baseline role assignments.
type ProvisioningRule struct {
    Name        string
    MatchAttr   string // e.g., "department"
    MatchValue  string // e.g., "engineering"
    AssignRoles []uuid.UUID // role IDs to auto-assign
}

// ProcessEvent handles a JML event and executes the appropriate lifecycle actions.
func (e *JMLEngine) ProcessEvent(ctx context.Context, event *JMLEvent) error {
    switch event.Type {
    case "joiner":
        return e.processJoiner(ctx, event)
    case "mover":
        return e.processMover(ctx, event)
    case "leaver":
        return e.processLeaver(ctx, event)
    default:
        return fmt.Errorf("unknown JML event type: %s", event.Type)
    }
}

func (e *JMLEngine) processJoiner(ctx context.Context, event *JMLEvent) error {
    // Apply baseline provisioning rules based on department/title.
    for _, rule := range e.rules {
        if e.matchesRule(event, rule) {
            for _, roleID := range rule.AssignRoles {
                err := e.policySvc.AssignRole(ctx, event.UserID, roleID, "global", event.TenantID)
                if err != nil {
                    log.Printf("auto-assign role %s: %v", roleID, err)
                }
            }
        }
    }

    // Notify manager that the user is provisioned.
    return e.notifier.NotifyJoinerComplete(ctx, event.ManagerID, event)
}

func (e *JMLEngine) processMover(ctx context.Context, event *JMLEvent) error {
    // Get current roles.
    currentRoles, err := e.policySvc.ListUserRoles(ctx, event.UserID)
    if err != nil {
        return err
    }

    // Calculate roles needed for new position.
    newRoles := make(map[uuid.UUID]bool)
    for _, rule := range e.rules {
        if e.matchesRule(event, rule) {
            for _, roleID := range rule.AssignRoles {
                newRoles[roleID] = true
            }
        }
    }

    // Revoke roles not needed in new position.
    for _, ur := range currentRoles {
        if !newRoles[ur.RoleID] {
            e.policySvc.RevokeRole(ctx, event.UserID, ur.RoleID)
        }
    }

    // Assign new roles.
    currentSet := make(map[uuid.UUID]bool)
    for _, ur := range currentRoles {
        currentSet[ur.RoleID] = true
    }
    for roleID := range newRoles {
        if !currentSet[roleID] {
            e.policySvc.AssignRole(ctx, event.UserID, roleID, "global", event.TenantID)
        }
    }

    return e.notifier.NotifyMoverComplete(ctx, event.ManagerID, event)
}

func (e *JMLEngine) processLeaver(ctx context.Context, event *JMLEvent) error {
    // Revoke ALL roles.
    currentRoles, err := e.policySvc.ListUserRoles(ctx, event.UserID)
    if err != nil {
        return err
    }
    for _, ur := range currentRoles {
        e.policySvc.RevokeRole(ctx, event.UserID, ur.RoleID)
    }

    // Disable the user account.
    if err := e.identitySvc.LockUser(ctx, event.UserID); err != nil {
        return fmt.Errorf("lock user account: %w", err)
    }

    // Terminate all active sessions (revoke refresh tokens).
    e.identitySvc.RevokeAllSessions(ctx, event.UserID)

    return e.notifier.NotifyLeaverComplete(ctx, event.ManagerID, event)
}

func (e *JMLEngine) matchesRule(event *JMLEvent, rule ProvisioningRule) bool {
    switch rule.MatchAttr {
    case "department":
        return event.Department == rule.MatchValue
    case "job_title":
        return event.JobTitle == rule.MatchValue
    default:
        val, ok := event.Payload[rule.MatchAttr]
        return ok && fmt.Sprintf("%v", val) == rule.MatchValue
    }
}
```

### Event-Driven Architecture

The JML engine should be **event-driven**, consuming events from:

- **HR webhooks** — Workday, BambooHR, or custom HR systems push JSON events to
  `POST /api/v1/governance/jml/events`.
- **SCIM 2.0** — GGID's existing SCIM endpoint can trigger JML processing when users
  are created, updated, or deactivated via `POST/PATCH/DELETE /scim/v2/Users`.
- **Scheduled polls** — For HR systems that don't support webhooks, a periodic poll
  job queries the HR API for changes.

Events flow through **NATS JetStream** for reliability (at-least-once delivery, retry
on failure, dead-letter queue for poison messages).

---

## 7. Compliance Reporting

Compliance reporting provides the **evidence** that auditors need to verify that
access controls are working. Pre-built reports eliminate the manual effort of
collecting data from multiple systems.

### Standard Compliance Reports

| Report | Audience | Frequency | Purpose |
|--------|---------|-----------|---------|
| **User Access List** | Auditors | On-demand | Who has access to resource X |
| **Role Assignment History** | Auditors | On-demand | Who changed role Y, when, by whom |
| **Access Review Completion** | Compliance team | Per campaign | Completion rate, overdue items |
| **SoD Violation Summary** | Security team | Monthly | Active violations, remediation status |
| **Privileged Access Report** | Security/Audit | Monthly | All users with admin-level access |
| **Termination Timeliness** | Auditors | Quarterly | Time from HR termination to access revocation |
| **Policy Decision Log** | Auditors | On-demand | All allow/deny decisions for a time period |
| **Failed Authentication Report** | Security team | Weekly | Brute force attempts, locked accounts |

### Go Report Engine

```go
package governance

// ReportRequest defines parameters for generating a compliance report.
type ReportRequest struct {
    Type      string    // user_access, role_history, review_completion, etc.
    TenantID  uuid.UUID
    StartDate time.Time
    EndDate   time.Time
    Filters   map[string]any
    Format    string // json, csv, pdf
}

// ReportEngine generates compliance reports from audit data.
type ReportEngine struct {
    auditSvc   AuditClient   // query NATS audit event store
    policySvc  PolicyClient  // query role assignments
    campaignRepo CampaignRepository
    sodChecker *SoDChecker
}

// Generate produces a compliance report based on the request type.
func (e *ReportEngine) Generate(ctx context.Context, req *ReportRequest) (*Report, error) {
    switch req.Type {
    case "user_access":
        return e.userAccessReport(ctx, req)
    case "role_history":
        return e.roleHistoryReport(ctx, req)
    case "review_completion":
        return e.reviewCompletionReport(ctx, req)
    case "sod_violations":
        return e.sodViolationReport(ctx, req)
    case "privileged_access":
        return e.privilegedAccessReport(ctx, req)
    case "termination_timeliness":
        return e.terminationTimelinessReport(ctx, req)
    default:
        return nil, fmt.Errorf("unknown report type: %s", req.Type)
    }
}

// userAccessReport answers: "Who has access to resource X?"
func (e *ReportEngine) userAccessReport(ctx context.Context, req *ReportRequest) (*Report, error) {
    resourceType, _ := req.Filters["resource_type"].(string)

    // Query all role assignments that grant access to this resource type.
    assignments, err := e.policySvc.ListAssignmentsByResource(ctx, resourceType)
    if err != nil {
        return nil, err
    }

    rows := make([]map[string]any, 0, len(assignments))
    for _, a := range assignments {
        rows = append(rows, map[string]any{
            "user_id":      a.UserID,
            "user_name":    a.UserName,
            "role_id":      a.RoleID,
            "role_name":    a.RoleName,
            "scope_type":   a.ScopeType,
            "granted_by":   a.GrantedBy,
            "granted_at":   a.GrantedAt,
            "expires_at":   a.ExpiresAt,
        })
    }

    return &Report{
        Type:      req.Type,
        GeneratedAt: time.Now().UTC(),
        RowCount:  len(rows),
        Rows:      rows,
    }, nil
}

// terminationTimelinessReport checks SLA compliance for access revocation
// after employee termination. Critical for SOX evidence.
func (e *ReportEngine) terminationTimelinessReport(ctx context.Context, req *ReportRequest) (*Report, error) {
    // Query JML leaver events in the date range.
    leavers, err := e.auditSvc.QueryEvents(ctx, &AuditQuery{
        EventType:  "jml.leaver",
        StartDate:  req.StartDate,
        EndDate:    req.EndDate,
        TenantID:   req.TenantID,
    })
    if err != nil {
        return nil, err
    }

    rows := make([]map[string]any, 0)
    for _, event := range leavers {
        // Find when access was actually revoked.
        revokeTime, err := e.findLastRevoke(ctx, event.UserID)
        if err != nil {
            continue
        }

        delay := revokeTime.Sub(event.Timestamp)
        slaMet := delay < 24*time.Hour // SLA: revoke within 24 hours

        rows = append(rows, map[string]any{
            "user_id":         event.UserID,
            "termination_at":  event.Timestamp,
            "access_revoked_at": revokeTime,
            "delay_hours":     delay.Hours(),
            "sla_met":         slaMet,
        })
    }

    return &Report{
        Type:      req.Type,
        GeneratedAt: time.Now().UTC(),
        RowCount:  len(rows),
        Rows:      rows,
    }, nil
}
```

### Scheduled Report Generation

Reports should be **schedulable** — automatically generated on a recurring basis and
delivered to stakeholders:

- Monthly SoD violation report → emailed to compliance team
- Quarterly access review completion → emailed to CISO
- Weekly privileged access changes → emailed to security team

GGID can implement this with a cron-like scheduler that invokes the report engine and
sends results via the notification system.

---

## 8. How GGID's RBAC+ABAC Relates

### Current GGID Policy Engine Capabilities

Reviewing the actual source code in `services/policy/`, GGID currently has:

**RBAC (Role-Based Access Control):**
- Roles with tenant-scoped unique keys (`domain.Role`)
- Permissions defined as resource_type + action pairs (`domain.Permission`)
- Role-permission junction table with optional ABAC conditions (`domain.RolePermission`)
- User-role assignments with scope (global/organization/department/team/resource)
- Role hierarchy with parent-child inheritance (`ParentRoleID`)
- Role assignment expiry (`ExpiresAt` on `UserRole`)
- Cycle detection in role hierarchy (`SetParent`)
- Effective permissions calculation (walk descendant tree)

**ABAC (Attribute-Based Access Control):**
- AWS IAM-style policies with effect (allow/deny), actions, resources (`domain.Policy`)
- Rich condition operators: StringEquals, StringNotEquals, StringLike, NumericLessThan,
  Bool, DateLessThan, IpAddress, and more (`matchConditions` / `evaluateOperator`)
- Policy attachments to principals (user, role, group)
- Priority-based evaluation order
- Deny-overrides-allow decision model

**Policy Management:**
- Policy export/import, templates, versioning
- Dry-run evaluation, diff analysis
- Decision logging with in-memory ring buffer
- Attribute mapping
- Default action configuration
- Time-based conditions

### What IGA Adds ON TOP of RBAC+ABAC

The policy engine answers **"can user X do action Y on resource Z right now?"** —
this is the enforcement layer. IGA adds the **governance layer** on top:

| Capability | Policy Engine (GGID has) | IGA (GGID needs) |
|-----------|-------------------------|------------------|
| **Who decided this access?** | `granted_by` field on UserRole | Full approval chain, comments, timestamps |
| **Is this access still needed?** | No periodic review | Access review campaigns |
| **Is this access combination safe?** | No SoD checking | SoD policy engine |
| **What happens when someone leaves?** | Manual role revocation | Automated JML deprovisioning |
| **Can you prove controls work?** | Decision log | Compliance reports, evidence packages |
| **Can users self-service request access?** | Admin assigns manually | Self-service portal with approval workflow |
| **Is access lifecycle automated?** | No | HR-driven provisioning, role-based baseline |

### Gap Analysis: Policy Engine → IGA

The gap between GGID's current policy engine and full IGA capability:

1. **No access request workflow** — Roles are assigned via `RoleService.AssignRole()`
   by an admin. There's no self-service request mechanism, no approval chain, no SLA
   tracking.

2. **No access review/certification** — There's no mechanism to periodically review
   role assignments. The `UserRole.GrantedBy` field records who assigned a role, but
   there's no campaign to verify that the assignment is still appropriate.

3. **No SoD enforcement** — The evaluator checks whether a user has permission for a
   specific action, but it doesn't check whether the user's combined role set creates
   a dangerous conflict.

4. **No JML automation** — `RoleService.AssignRole()` and `RoleService.RevokeRole()`
   exist but must be called manually. There's no event-driven automation for
   joiner/mover/leaver events.

5. **No compliance reporting** — The `DecisionLogger` records policy decisions in
   memory, and the audit service records events to NATS. But there are no pre-built
   compliance reports or evidence packages.

6. **No provisioning connectors** — SCIM 2.0 exists as an inbound API skeleton in the
   identity service (for receiving user data from external IdPs), but there's no
   outbound provisioning (pushing user data to downstream applications).

---

## 9. Build vs Buy vs Integrate

### Option 1: Build IGA from Scratch (Native GGID)

**Approach:** Implement a new `services/governance/` microservice with workflow engine,
campaign manager, SoD checker, JML engine, and compliance reporting.

| Pros | Cons |
|------|------|
| Full control over architecture and features | 12-18 months of development effort |
| No additional licensing costs | Requires IGA domain expertise on the team |
| Tight integration with existing policy engine | Maintenance burden for connectors |
| Open-source differentiation vs competitors | Risk of building a "lite" IGA that's not competitive |
| Can leverage GGID's existing RBAC/ABAC/audit | Compliance certifications needed (SOC 2, etc.) |

### Option 2: Integrate with Existing IGA Tools

**Approach:** GGID exposes standard APIs (SCIM 2.0, SPML) that allow external IGA tools
(SailPoint, Saviynt, Okta IGA) to manage identity lifecycle.

| Pros | Cons |
|------|------|
| Fastest time-to-market (weeks, not months) | Customers need to buy a separate IGA product |
| Leverages mature, audited IGA platforms | Integration complexity (SCIM mapping, event sync) |
| No IGA domain expertise needed | Revenue goes to the IGA vendor, not GGID |
| Standard protocol (SCIM 2.0) | No native governance UX in GGID console |

### Option 3: Partner / OEM

**Approach:** Partner with an IGA vendor for an embedded or co-branded governance module.

| Pros | Cons |
|------|------|
| Faster than building, more integrated than buying | Revenue share, dependency on partner |
| Co-marketing opportunities | Technical coupling to partner's roadmap |
| IGA vendor handles compliance certifications | May conflict with open-source philosophy |

### Recommendation for GGID

**Build a minimal IGA layer** focused on the highest-impact features:

1. **Access request workflow** with manager approval — 3-4 weeks
2. **SoD checker** integrated with the provisioning path — 2-3 weeks
3. **Access review campaigns** — 4-6 weeks
4. **Compliance reporting** (leveraging existing audit data) — 2-3 weeks
5. **JML engine** (leveraging existing SCIM + role assignment) — 3-4 weeks

**Total estimated effort: 14-20 weeks** for a single developer.

This gives GGID **native IGA capability** without the full complexity of a SailPoint.
For organizations that need deep IGA (role mining, identity analytics, hundreds of
provisioning connectors), GGID's SCIM 2.0 compliance allows integration with
enterprise IGA platforms.

The **hybrid approach** (build core governance + integrate for deep IGA) positions
GGID as competitive with Keycloak 26 while maintaining open-source flexibility.

---

## 10. Gap Analysis & Recommendations

### Priority Matrix

| Feature | Effort | Competitive Impact | Compliance Impact | Priority |
|---------|--------|-------------------|------------------|----------|
| Access Request Workflow | Medium | High | Medium | **P0** |
| SoD Checker | Low-Medium | Medium | High | **P0** |
| Access Review Campaigns | High | High | High | **P1** |
| Compliance Reports | Low-Medium | Medium | High | **P1** |
| JML Engine | Medium | Medium | Medium | **P2** |
| Role Mining | High | Low-Medium | Low | **P3** |
| Outbound SCIM Provisioning | Medium | Medium | Low | **P2** |
| Identity Analytics Dashboard | Medium | Low-Medium | Low | **P3** |

### Recommended Roadmap

#### Phase 1: Foundation (4-6 weeks)

Create `services/governance/` microservice with:

1. **Access Request Workflow** — Self-service portal where users request roles, managers
   approve/reject, roles auto-provision on approval. Integrate with GGID's existing
   `RoleService.AssignRole()`.

2. **SoD Checker** — Define conflicting role pairs in a new `sod_policies` table.
   Hook into the provisioning path: before any role assignment, check for SoD violations.
   Block critical violations, warn on high severity.

3. **Basic Compliance Reports** — Leverage existing audit data (NATS event store) to
   generate: user access list, role assignment history, SoD violation summary. Expose
   via REST API for the Admin Console.

#### Phase 2: Governance (6-8 weeks)

4. **Access Review Campaigns** — Campaign creation, review item generation, reviewer
   notifications, decision recording, automated revocation. Integrate with GGID's org
   service (LTREE hierarchy) to resolve managers.

5. **JML Engine** — Event-driven lifecycle automation. Consume HR webhook events,
   apply baseline role rules, execute joiner/mover/leaver provisioning. Integrate with
   existing identity and policy services.

6. **Scheduled Reporting** — Cron-based report generation with email delivery. Monthly
   compliance summary, quarterly access review completion, weekly privileged access changes.

#### Phase 3: Advanced (8-12 weeks, post-MVP)

7. **Outbound SCIM Provisioning** — Push user/role data to downstream applications
   (e.g., automatically provision accounts in Slack, Google Workspace, Salesforce).

8. **Role Mining** — Analyze actual permission usage patterns to suggest role
   consolidation and identify over-provisioned users.

9. **Identity Analytics** — Dashboard showing access risk heatmap, orphaned accounts,
   dormant accounts, role explosion metrics.

### Competitive Parity with Keycloak 26

To match Keycloak 26's IGA capabilities, GGID needs **at minimum**:

- Access request workflow with multi-level approval chains ✓ (Phase 1)
- Manager-based access reviews ✓ (Phase 2)
- SoD enforcement ✓ (Phase 1)
- Audit trail for compliance ✓ (existing audit service + Phase 1 reports)

GGID's **advantage over Keycloak** in this space:
- Go-based performance (Keycloak is Java/JVM)
- Multi-tenant native architecture
- gRPC-first design for service-to-service governance calls
- AWS IAM-style ABAC conditions (richer than Keycloak's attribute-based policies)
- Open-source Apache 2.0 (Keycloak is also open source, but GGID's Go stack is lighter)

### Key Architectural Decisions

1. **New microservice, not bolt-on** — IGA logic should live in `services/governance/`,
   not in the policy service. This follows GGID's microservice architecture and allows
   independent scaling.

2. **Event-driven, not synchronous** — JML and campaign operations should use NATS
   JetStream for async processing. Real-time SoD checks can be synchronous gRPC calls.

3. **Leverage existing infrastructure** — Use GGID's existing org service for manager
   resolution, policy service for role assignment, audit service for compliance data,
   NATS for notifications.

4. **Database tables alongside policy** — New tables (`access_requests`, `approval_steps`,
   `campaigns`, `review_items`, `sod_policies`, `jml_events`, `provisioning_rules`)
   in the governance service's own schema, with tenant_id for RLS.

---

## Summary

IGA is the **governance layer** that transforms a basic IAM system into a compliance-ready
platform. GGID's existing RBAC+ABAC engine is a strong foundation — it handles the
enforcement question ("can user X do Y?"). IGA adds the governance question ("should
user X have Y, and can we prove it?").

The recommended path is to **build a focused governance microservice** that adds the
four highest-impact IGA capabilities: access requests with approvals, SoD enforcement,
access review campaigns, and compliance reporting. This positions GGID competitively
against Keycloak 26 while maintaining GGID's core advantages: Go performance, multi-tenancy,
and rich ABAC policy evaluation.

**Estimated total effort: 14-20 weeks** for P0+P1 features, deliverable by a single
developer. This investment transforms GGID from "IAM platform" to "IAM + Governance
platform" — a meaningfully different market position.
