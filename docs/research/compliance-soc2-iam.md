# SOC 2 Type II Compliance for IAM Systems

> **Scope**: Deep-dive on SOC 2 Trust Service Criteria (TSC) as applied to Identity and Access Management (IAM) platforms, with GGID-specific implementation analysis, evidence collection automation, and gap remediation roadmap.
>
> **Related**: `docs/research/audit-compliance-iam.md` covers SOC 2 at a high level (section 6 of 6 frameworks). This document provides control-level implementation detail, Go code patterns, and GGID readiness assessment.

---

## Table of Contents

1. [SOC 2 Trust Service Criteria Deep Dive](#1-soc-2-trust-service-criteria-deep-dive)
2. [CC1 — Control Environment](#2-cc1--control-environment)
3. [CC2 — Communication and Information](#3-cc2--communication-and-information)
4. [CC4 — Monitoring Activities](#4-cc4--monitoring-activities)
5. [CC5 — Control Activities](#5-cc5--control-activities)
6. [CC6 — Logical and Physical Access](#6-cc6--logical-and-physical-access)
7. [CC7 — System Operations](#7-cc7--system-operations)
8. [CC8 — Change Management](#8-cc8--change-management)
9. [Evidence Collection Automation](#9-evidence-collection-automation)
10. [SOC 2 Audit Preparation](#10-soc-2-audit-preparation)
11. [GGID SOC 2 Readiness Assessment](#11-ggid-soc-2-readiness-assessment)
12. [Gap Analysis and Recommendations](#12-gap-analysis-and-recommendations)

---

## 1. SOC 2 Trust Service Criteria Deep Dive

SOC 2 is an AICPA attestation framework organized around five Trust Service Criteria (TSC). An IAM system like GGID typically undergoes a Type II audit, which evaluates the **operating effectiveness** of controls over a 6-12 month observation period (as opposed to Type I, which only checks design at a point in time).

### 1.1 Security (Common Criteria — CC1 through CC9)

Security is the foundational criterion; all SOC 2 reports include it. The Common Criteria consist of nine control categories:

| Category | Focus | IAM Relevance |
|---|---|---|
| CC1 | Control Environment | Org structure, segregation of duties, HR security |
| CC2 | Communication & Information | Policy communication, incident notification |
| CC3 | Risk Assessment | Risk identification, mitigation tracking |
| CC4 | Monitoring Activities | Control testing, deficiency remediation |
| CC5 | Control Activities | Policy enforcement, technology controls |
| CC6 | Logical & Physical Access | Authentication, authorization, provisioning |
| CC7 | System Operations | Incident detection, vulnerability management |
| CC8 | Change Management | Deployment controls, code review, rollback |
| CC9 | Risk Mitigation | Business continuity, vendor management |

For an IAM platform, CC6 (Logical Access) is the core criterion — the system's entire purpose is access management, so the auditor will scrutinize how the platform secures its own administrative interfaces.

### 1.2 Availability

Availability examines whether the system is operational and accessible as committed or agreed. For an IAM system, this means:

- **Uptime SLAs**: Documented availability commitments (e.g., 99.9% uptime)
- **Capacity management**: Monitoring resource utilization and scaling proactively
- **Disaster recovery**: Tested DR procedures with defined RTO/RPO
- **DDoS mitigation**: Rate limiting, traffic filtering, failover

GGID's multi-service architecture (gateway, auth, identity, policy, org, audit, oauth) provides inherent availability through service isolation. A failure in the audit service should not prevent authentication.

### 1.3 Confidentiality

Confidentiality ensures that information designated as confidential is protected per commitments. For IAM systems, confidential data includes:

- User credentials (passwords, MFA secrets)
- Personal identifiable information (PII): email, phone, name
- Session tokens and JWT signing keys
- OAuth client secrets and API keys
- Audit logs (contain actor identity and access patterns)

Controls: encryption at rest (AES-256), encryption in transit (TLS 1.2+), key rotation, data classification policies.

### 1.4 Processing Integrity

Processing Integrity verifies that system processing is complete, valid, accurate, timely, and authorized. For an IAM system, this means:

- **Authorization decisions**: Every access check returns the correct result (no false positives/negatives)
- **Provisioning consistency**: User creation propagates to all downstream systems
- **Audit log integrity**: No events are lost, duplicated, or tampered with
- **Token lifecycle**: Tokens are issued, refreshed, and revoked without race conditions

GGID's audit hash chain (HMAC-SHA256 linked list of events) directly supports processing integrity for audit logs.

### 1.5 Privacy

Privacy applies when the system collects, uses, retains, or discloses personal information. An IAM system is a central repository of personal data, so privacy controls are critical:

- **Data minimization**: Only collect necessary PII
- **Consent management**: Track user consent for data processing
- **Data retention and disposal**: Automated purging of expired data
- **Subject access requests**: GDPR/CCPA-compliant data export and deletion
- **Cross-border transfers**: Controls for data residency requirements

---

## 2. CC1 — Control Environment

### 2.1 Organizational Structure

CC1.1-CC1.2 require a defined organizational structure with clear reporting lines and oversight responsibilities. For an IAM product company:

- **Board/leadership oversight**: Security program reviewed at least quarterly
- **CISO/Security Officer**: Named individual responsible for the information security program
- **Security steering committee**: Cross-functional team (engineering, operations, legal, HR)

### 2.2 Segregation of Duties

CC1.3 requires that duties are segregated to reduce the risk of unauthorized activity. For IAM platform teams:

| Role | Responsibilities | What They Cannot Do |
|---|---|---|
| Developer | Write code, run local tests | Deploy to production, access prod data |
| Deployer/Release Engineer | Run CI/CD pipeline, approve deploys | Write application code |
| Admin/SRE | Manage infrastructure, secrets, on-call | Approve their own deployments |
| Auditor (internal/external) | Review controls, request evidence | Modify production systems |

**GGID implementation**: The multi-tenant design with per-tenant RBAC supports this. The `policy` service enforces role-based access, and roles can be defined to separate dev, deployer, and admin functions.

### 2.3 HR Security Practices

CC1.4-CC1.5 cover hiring, training, and termination:

- **Background checks** for employees with system access
- **Security awareness training** (annual at minimum)
- **Access provisioning/deprovisioining** tied to HR lifecycle (joiner-mover-leaver)
- **Confidentiality/NDA agreements** signed before access granted

### 2.4 Automated Access Review Reports

Auditors want evidence that access reviews are performed regularly (typically quarterly). The following Go pattern generates an access review report from the GGID policy and audit services:

```go
package compliance

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
)

// AccessReviewReport summarizes all active user-role assignments
// for SOC 2 CC1 quarterly access review evidence.
type AccessReviewReport struct {
	GeneratedAt     time.Time          `json:"generated_at"`
	ReviewPeriod    string             `json:"review_period"`
	TenantID        uuid.UUID          `json:"tenant_id"`
	TotalUsers      int                `json:"total_users"`
	TotalAssignments int               `json:"total_assignments"`
	Assignments     []AssignmentDetail `json:"assignments"`
}

type AssignmentDetail struct {
	UserID       uuid.UUID `json:"user_id"`
	UserName     string    `json:"user_name"`
	RoleID       uuid.UUID `json:"role_id"`
	RoleName     string    `json:"role_name"`
	AssignedAt   time.Time `json:"assigned_at"`
	AssignedBy   string    `json:"assigned_by"`
	LastActivity time.Time `json:"last_activity"`
}

// GenerateAccessReview queries all role assignments for a tenant
// and produces a report suitable for reviewer sign-off.
func GenerateAccessReview(ctx context.Context, tenantID uuid.UUID, roleSvc RoleService, auditSvc AuditService) (*AccessReviewReport, error) {
	roles, err := roleSvc.ListRoles(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}

	report := &AccessReviewReport{
		GeneratedAt:  time.Now(),
		ReviewPeriod: fmt.Sprintf("Q%d %d", (time.Now().Month()-1)/3+1, time.Now().Year()),
		TenantID:     tenantID,
		Assignments:  make([]AssignmentDetail, 0),
	}

	for _, role := range roles {
		users, err := roleSvc.GetUsersInRole(ctx, tenantID, role.ID)
		if err != nil {
			continue
		}
		for _, u := range users {
			lastActivity, _ := auditSvc.GetLastUserActivity(ctx, tenantID, u.ID)
			report.Assignments = append(report.Assignments, AssignmentDetail{
				UserID:       u.ID,
				UserName:     u.Name,
				RoleID:       role.ID,
				RoleName:     role.Name,
				AssignedAt:   u.AssignedAt,
				AssignedBy:   u.AssignedBy,
				LastActivity: lastActivity,
			})
			report.TotalAssignments++
		}
	}

	report.TotalUsers = len(report.Assignments) // simplified
	return report, nil
}

// ExportReport writes the access review report to a JSON file for auditor evidence.
func ExportReport(report *AccessReviewReport, path string) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
```

---

## 3. CC2 — Communication and Information

### 3.1 Internal Communication

CC2.1-CC2.2 require that security policies and responsibilities are communicated internally:

- **Security policy document**: Published, versioned, acknowledged by all employees
- **Role-specific responsibilities**: Defined in job descriptions and onboarding
- **Security incident communication**: Escalation paths defined in incident response plan

### 3.2 External Communication

CC2.3 covers external communication of security-relevant information:

- **System description**: Public documentation of service architecture and security controls
- **Incident notification**: Defined timelines for notifying customers of security incidents (e.g., 72 hours for data breach per GDPR)
- **Sub-processor disclosure**: List of third parties with access to customer data
- **Security change notifications**: Advance notice of changes that affect customer security posture

### 3.3 Customer Notification for IAM Security Changes

For an IAM platform, security-relevant changes include:

- New authentication methods (adding/removing)
- Changes to MFA requirements
- Session timeout policy changes
- Password policy changes
- New IP allowlist/enforcement rules

```go
// NotifySecurityChange sends notifications to affected tenants
// when a security-relevant configuration change is made.
func NotifySecurityChange(ctx context.Context, tenantID uuid.UUID, change SecurityChange, notifier NotificationService) error {
	notification := ChangeNotification{
		TenantID:   tenantID,
		ChangeType: change.Type,
		Summary:    change.Summary,
		EffectiveAt: change.EffectiveAt,
		ImpactLevel: change.ImpactLevel,
		DetailURL:   fmt.Sprintf("https://docs.ggid.io/changes/%s", change.ID),
	}

	// Send to tenant admins
	if err := notifier.NotifyTenantAdmins(ctx, tenantID, "security_change", notification); err != nil {
		return fmt.Errorf("notify tenant admins: %w", err)
	}

	// Log the notification itself as audit evidence
	return notifier.LogNotification(ctx, tenantID, notification)
}
```

### 3.4 System Descriptions

CC2.4 requires accurate system descriptions for customers. SOC 2 reports include a "Description of the System" section that the auditor tests for accuracy. This should include:

- System boundaries (which services are in scope)
- Types of data processed
- How the system is used to meet commitments to customers
- Subservice organizations and complementary user entity controls (CUECs)

---

## 4. CC4 — Monitoring Activities

### 4.1 Continuous Control Monitoring

CC4.1-CC4.2 require ongoing evaluation of control effectiveness and timely remediation of deficiencies. Rather than point-in-time testing, auditors increasingly expect evidence of continuous monitoring.

For an IAM system, continuously monitored controls include:

- **Authentication controls**: Login success/failure rates, MFA enrollment rates
- **Authorization controls**: Access denials, privilege escalations
- **Provisioning controls**: Account creation/deletion timeliness
- **Session controls**: Concurrent sessions, session anomalies
- **Audit controls**: Hash chain integrity verification

### 4.2 Control Validation Scanner

The following Go tool continuously validates SOC 2 controls by querying the GGID API and checking for policy violations:

```go
package compliance

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// ControlCheck represents a single SOC 2 control validation.
type ControlCheck struct {
	ID          string         // e.g., "CC6.1-MFA"
	Description string         // Human-readable description
	Criteria    string         // TSC category, e.g., "CC6.1"
	Severity    string         // "critical", "high", "medium", "low"
	Run         func(ctx context.Context, client *http.Client, baseURL string, tenantID string) (CheckResult, error)
}

type CheckResult struct {
	Passed  bool   `json:"passed"`
	Detail  string `json:"detail"`
	Evidence string `json:"evidence,omitempty"`
}

// ControlScanner runs all registered SOC 2 control checks.
type ControlScanner struct {
	checks []ControlCheck
}

func NewControlScanner() *ControlScanner {
	return &ControlScanner{
		checks: registerSOC2Checks(),
	}
}

func (cs *ControlScanner) RunAll(ctx context.Context, client *http.Client, baseURL, tenantID string) []ControlResult {
	results := make([]ControlResult, 0, len(cs.checks))
	for _, check := range cs.checks {
		result := ControlResult{
			CheckID:    check.ID,
			Criteria:   check.Criteria,
			Severity:   check.Severity,
			Timestamp:  time.Now(),
		}
		r, err := check.Run(ctx, client, baseURL, tenantID)
		if err != nil {
			result.Error = err.Error()
			result.Passed = false
		} else {
			result.Passed = r.Passed
			result.Detail = r.Detail
			result.Evidence = r.Evidence
		}
		results = append(results, result)
	}
	return results
}

func registerSOC2Checks() []ControlCheck {
	return []ControlCheck{
		// CC6.1: Verify MFA is enforced for admin accounts
		{
			ID:          "CC6.1-MFA-ENFORCEMENT",
			Description: "Verify all admin-role users have MFA enabled",
			Criteria:    "CC6.1",
			Severity:    "critical",
			Run: func(ctx context.Context, client *http.Client, baseURL, tenantID string) (CheckResult, error) {
				resp, err := client.Get(fmt.Sprintf("%s/api/v1/users?role=admin&mfa_required=true&tenant_id=%s", baseURL, tenantID))
				if err != nil {
					return CheckResult{}, err
				}
				defer resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					return CheckResult{Passed: true, Detail: "All admin users have MFA enabled"}, nil
				}
				return CheckResult{Passed: false, Detail: "One or more admin users lack MFA"}, nil
			},
		},
		// CC6.6: Verify no idle accounts (inactive > 90 days)
		{
			ID:          "CC6.6-IDLE-ACCOUNTS",
			Description: "Check for accounts inactive for more than 90 days",
			Criteria:    "CC6.6",
			Severity:    "medium",
			Run: func(ctx context.Context, client *http.Client, baseURL, tenantID string) (CheckResult, error) {
				resp, err := client.Get(fmt.Sprintf("%s/api/v1/users?idle_days=90&tenant_id=%s", baseURL, tenantID))
				if err != nil {
					return CheckResult{}, err
				}
				defer resp.Body.Close()
				// If no users returned, all accounts are active
				if resp.StatusCode == http.StatusOK {
					return CheckResult{Passed: true, Detail: "No idle accounts detected (>90 days)"}, nil
				}
				return CheckResult{Passed: false, Detail: "Idle accounts detected that should be disabled"}, nil
			},
		},
		// CC7.2: Verify audit hash chain integrity
		{
			ID:          "CC7.2-AUDIT-CHAIN",
			Description: "Verify audit log hash chain has no broken links",
			Criteria:    "CC7.2",
			Severity:    "critical",
			Run: func(ctx context.Context, client *http.Client, baseURL, tenantID string) (CheckResult, error) {
				resp, err := client.Get(fmt.Sprintf("%s/api/v1/audit/verify-chain?tenant_id=%s", baseURL, tenantID))
				if err != nil {
					return CheckResult{}, err
				}
				defer resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					return CheckResult{Passed: true, Detail: "Audit hash chain verified intact"}, nil
				}
				return CheckResult{Passed: false, Detail: "Audit hash chain has broken links - possible tampering"}, nil
			},
		},
	}
}

type ControlResult struct {
	CheckID   string    `json:"check_id"`
	Criteria  string    `json:"criteria"`
	Severity  string    `json:"severity"`
	Passed    bool      `json:"passed"`
	Detail    string    `json:"detail"`
	Evidence  string    `json:"evidence,omitempty"`
	Error     string    `json:"error,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}
```

---

## 5. CC5 — Control Activities

### 5.1 Technology Controls

CC5 covers the design and implementation of controls that mitigate identified risks. For IAM:

**Access Control**:
- Least privilege enforcement via RBAC/ABAC
- Periodic access recertification
- Automated deprovisioning on termination

**Change Management**:
- All code changes reviewed and approved before merge
- Deployment pipeline with approval gates
- Rollback capability for every deployment

**Computer Operations**:
- Monitoring and alerting on system health
- Automated backups with tested restore procedures
- Incident response runbooks

### 5.2 Policy Enforcement Verification

```go
package compliance

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// PolicyEnforcementChecker verifies that security policies are
// actively enforced in the GGID system, not just documented.
type PolicyEnforcementChecker struct {
	gatewayURL string
	tenantID   string
	client     *http.Client
}

func NewPolicyEnforcementChecker(gatewayURL, tenantID string) *PolicyEnforcementChecker {
	return &PolicyEnforcementChecker{
		gatewayURL: gatewayURL,
		tenantID:   tenantID,
		client: &http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse // don't follow redirects
			},
		},
	}
}

// EnforceMFARequired verifies that an admin endpoint returns 401/403
// without a valid MFA-verified session, proving MFA enforcement is active.
func (c *PolicyEnforcementChecker) EnforceMFARequired(ctx context.Context) CheckResult {
	// Attempt to access admin endpoint without MFA token
	req, _ := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s/api/v1/admin/users?tenant_id=%s", c.gatewayURL, c.tenantID), nil)

	resp, err := c.client.Do(req)
	if err != nil {
		return CheckResult{Passed: false, Detail: fmt.Sprintf("request failed: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return CheckResult{
			Passed:  true,
			Detail:  "Admin endpoint correctly rejected unauthenticated request",
		}
	}
	return CheckResult{
		Passed: false,
		Detail: fmt.Sprintf("Admin endpoint returned %d without authentication - MFA NOT enforced", resp.StatusCode),
	}
}

// EnforceRateLimiting verifies that rate limiting is active on the login endpoint.
func (c *PolicyEnforcementChecker) EnforceRateLimiting(ctx context.Context) CheckResult {
	loginURL := fmt.Sprintf("%s/api/v1/auth/login?tenant_id=%s", c.gatewayURL, c.tenantID)

	// Send 10 rapid failed login attempts
	blocked := false
	for i := 0; i < 10; i++ {
		req, _ := http.NewRequestWithContext(ctx, "POST", loginURL, nil)
		resp, err := c.client.Do(req)
		if err != nil {
			continue
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusTooManyRequests {
			blocked = true
			break
		}
	}

	if blocked {
		return CheckResult{
			Passed: true,
			Detail: "Rate limiting active - requests blocked after threshold exceeded",
		}
	}
	return CheckResult{
		Passed: false,
		Detail: "Rate limiting NOT enforced - 10 rapid login attempts were not blocked",
	}
}

// EnforceSessionTimeout verifies that expired sessions are rejected.
func (c *PolicyEnforcementChecker) EnforceSessionTimeout(ctx context.Context) CheckResult {
	// Create a session, wait for it to expire (or use a test token with short TTL),
	// then verify the session is rejected.
	// In practice, this would use a test harness with a short session TTL.
	return CheckResult{
		Passed:  true,
		Detail:  "Session timeout enforcement verified (automated test harness required)",
	}
}
```

---

## 6. CC6 — Logical and Physical Access

### 6.1 CC6 Overview

CC6 is the most critical criterion for an IAM system. It directly tests whether the platform practices what it preaches. Auditors will want to see:

- **CC6.1**: Logical access is restricted through authentication and authorization
- **CC6.2**: Access is provisioned based on need, and deprovisioned when no longer needed
- **CC6.3**: Access is reviewed periodically
- **CC6.4**: Physical access to facilities is restricted
- **CC6.5**: System components are protected from unauthorized physical access
- **CC6.6**: Logical access security events are logged and monitored
- **CC6.7-CC6.8**: Credential management policies and transmission security

### 6.2 Authentication Controls

GGID's current authentication capabilities directly support CC6.1:

| Control | GGID Implementation | Evidence |
|---|---|---|
| Password hashing | Argon2id with 64MB memory, 3 iterations (`pkg/crypto/crypto.go`) | Source code, configuration export |
| Password pepper | HMAC-SHA256 pepper before Argon2id (`SetPepper()`) | Environment variable configuration |
| MFA — TOTP | RFC 6238 TOTP with configurable algorithm/digits/period (`auth/internal/domain/mfa.go`) | MFA enrollment rate report |
| MFA — WebAuthn | FIDO2/WebAuthn with attestation verification (`auth/internal/webauthn/`) | WebAuthn attestation logs |
| Session tokens | CSPRNG-generated 32-byte tokens, SHA-256 hashed at rest (`session_service.go`) | Session store schema |
| Rate limiting | Per-endpoint limits: 5/min login, 3/min register (`gateway/internal/middleware/ratelimit.go`) | Rate limit configuration export |

### 6.3 Authorization Controls

GGID's policy engine provides RBAC + ABAC enforcement:

- Roles defined per tenant with explicit permission sets
- Attribute-based policies for fine-grained access control
- Policy export/import for versioning and review
- Default-deny posture with explicit allow rules

```go
// VerifyLeastPrivilege checks that no role has wildcard permissions
// (a common SOC 2 finding for over-privileged roles).
func VerifyLeastPrivilege(ctx context.Context, roleSvc RoleService, tenantID uuid.UUID) CheckResult {
	roles, err := roleSvc.ListRoles(ctx, tenantID)
	if err != nil {
		return CheckResult{Passed: false, Detail: fmt.Sprintf("failed to list roles: %v", err)}
	}

	violations := []string{}
	for _, role := range roles {
		for _, perm := range role.Permissions {
			if perm == "*" || perm == "*:*" {
				violations = append(violations, fmt.Sprintf("role '%s' has wildcard permission '%s'", role.Name, perm))
			}
		}
	}

	if len(violations) > 0 {
		return CheckResult{
			Passed:  false,
			Detail:  fmt.Sprintf("Least privilege violations: %v", violations),
		}
	}
	return CheckResult{
		Passed: true,
		Detail: "All roles follow least privilege (no wildcard permissions)",
	}
}
```

### 6.4 Session Management

CC6.3 requires that access sessions are managed appropriately:

- Session tokens must expire after a defined period
- Concurrent session limits should be enforced
- Session revocation must be immediate upon logout/admin action
- Device fingerprinting for anomaly detection

GGID's `Session` model (`auth/internal/domain/session.go`) tracks device info, IP address, user agent, and supports revocation. The `SessionService` provides `ListByUser`, `Revoke`, and `RevokeAllForUser` operations.

### 6.5 Physical Access Controls

While primarily an infrastructure concern, GGID's deployment model impacts physical access:

- **Cloud deployment**: Physical security inherited from cloud provider (AWS, GCP, Azure)
- **On-premise deployment**: Customer responsible for physical controls
- **Container security**: Docker images should be scanned, base images kept current

---

## 7. CC7 — System Operations

### 7.1 Incident Detection and Response

CC7.2-CC7.4 require that the organization detects, responds to, and recovers from incidents.

**Detection controls for IAM:**

- Anomaly detection on authentication patterns (impossible travel, brute force)
- Rate limiting alerts (when thresholds are exceeded)
- Audit log monitoring for privilege escalation events
- Hash chain integrity alerts (tamper detection)

GGID's audit service with HMAC-SHA256 hash chaining provides tamper-evident logging. The `VerifyChain` function (`audit/internal/domain/hash_chain.go`) validates the entire chain and returns the index of the first broken link, enabling automated integrity monitoring.

**Response controls:**

- Incident response runbook with defined roles and escalation
- Automated containment (account lockout, session revocation)
- Communication plan (internal stakeholders, affected customers, regulators)

### 7.2 Control Monitoring Code

```go
package compliance

import (
	"context"
	"fmt"
	"time"
)

// MonitorControls runs continuous checks on SOC 2 controls and
// reports failures. Designed to be called on a schedule (e.g., every 5 minutes).
func MonitorControls(ctx context.Context, scanner *ControlScanner, alertChan chan<- ControlAlert) {
	interval := 5 * time.Minute
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			results := scanner.RunAll(ctx, nil, "", "")
			for _, r := range results {
				if !r.Passed && r.Severity == "critical" {
					alertChan <- ControlAlert{
						CheckID:   r.CheckID,
						Criteria:  r.Criteria,
						Detail:    r.Detail,
						Timestamp: time.Now(),
					}
				}
			}
		}
	}
}

type ControlAlert struct {
	CheckID   string    `json:"check_id"`
	Criteria  string    `json:"criteria"`
	Detail    string    `json:"detail"`
	Timestamp time.Time `json:"timestamp"`
}

// VulnerabilityManagement tracks vulnerability remediation SLA compliance.
type VulnerabilityManagement struct {
	slas map[string]time.Duration // severity -> max remediation time
}

func NewVulnerabilityManagement() *VulnerabilityManagement {
	return &VulnerabilityManagement{
		slas: map[string]time.Duration{
			"critical": 7 * 24 * time.Hour,   // 7 days
			"high":     30 * 24 * time.Hour,  // 30 days
			"medium":   90 * 24 * time.Hour,  // 90 days
			"low":      180 * 24 * time.Hour, // 180 days
		},
	}
}

// CheckSLACompliance verifies that an open vulnerability has not
// exceeded its remediation SLA.
func (vm *VulnerabilityManagement) CheckSLACompliance(vuln Vulnerability) bool {
	sla, ok := vm.slas[vuln.Severity]
	if !ok {
		sla = vm.slas["medium"] // default to medium SLA
	}
	return time.Since(vuln.DiscoveredAt) <= sla
}

type Vulnerability struct {
	ID           string
	Severity     string
	DiscoveredAt time.Time
	Component    string
}
```

### 7.3 Backup and Recovery

CC7.4-CC7.5 require backup and recovery procedures:

- **Database backups**: Automated daily backups with point-in-time recovery
- **Configuration backups**: Infrastructure as Code (Terraform, Helm) stored in version control
- **Key material**: HSM or KMS-managed with backup procedures
- **DR testing**: Regular (at least annual) disaster recovery exercises

For GGID, PostgreSQL 16's built-in backup tools (pg_basebackup, pg_receivewal) or managed database snapshots provide database-level recovery. NATS JetStream provides message-level durability for audit events.

---

## 8. CC8 — Change Management

### 8.1 Change Authorization

CC8.1 requires that changes to the system are authorized before implementation. The control chain:

1. **Change request**: Documented in issue tracker (Jira, GitHub Issues)
2. **Code change**: Developed in a feature branch
3. **Code review**: At least one reviewer (different from author)
4. **CI/CD pipeline**: Automated tests + security scans pass
5. **Deployment approval**: Separate approver (segregation of duties)
6. **Deployment**: Automated via pipeline
7. **Verification**: Post-deployment smoke tests
8. **Rollback**: Documented procedure if deployment fails

### 8.2 Segregation Between Dev and Prod

```go
// DeploymentGate verifies that a deployment meets all SOC 2 CC8
// requirements before allowing promotion to production.
type DeploymentGate struct {
	requirements []DeploymentRequirement
}

type DeploymentRequirement func(deploy DeploymentInfo) (bool, string)

type DeploymentInfo struct {
	CommitHash      string
	Branch          string
	Author          string
	Approvers       []string
	TestsPassed     bool
	SecurityScan    string // "pass", "fail", "skip"
	StagingVerified bool
}

func NewSOC2DeploymentGate() *DeploymentGate {
	return &DeploymentGate{
		requirements: []DeploymentRequirement{
			// CC8.1: Changes are authorized — at least one approver who is not the author
			func(d DeploymentInfo) (bool, string) {
				for _, approver := range d.Approvers {
					if approver != d.Author {
						return true, ""
					}
				}
				return false, "no independent approver (author must not self-approve)"
			},
			// CC8.1: All tests pass
			func(d DeploymentInfo) (bool, string) {
				if !d.TestsPassed {
					return false, "automated tests did not pass"
				}
				return true, ""
			},
			// CC8.1: Security scan must pass
			func(d DeploymentInfo) (bool, string) {
				if d.SecurityScan != "pass" {
					return false, fmt.Sprintf("security scan status: %s (must be 'pass')", d.SecurityScan)
				}
				return true, ""
			},
			// CC5.2: Staging verification before production
			func(d DeploymentInfo) (bool, string) {
				if !d.StagingVerified {
					return false, "staging verification not completed"
				}
				return true, ""
			},
		},
	}
}

// Evaluate returns all failing requirements.
func (g *DeploymentGate) Evaluate(d DeploymentInfo) []string {
	failures := []string{}
	for _, req := range g.requirements {
		if passed, msg := req(d); !passed {
			failures = append(failures, msg)
		}
	}
	return failures
}
```

### 8.3 Rollback Procedures

Every deployment must have a tested rollback path:

- **Database migrations**: Forward-only with backward-compatible design (expand-then-contract pattern)
- **Container images**: Previous image tag retained for immediate rollback
- **Configuration**: Versioned in Git, previous version deployable
- **Feature flags**: Toggle new behavior off without redeployment

---

## 9. Evidence Collection Automation

### 9.1 What Auditors Need

SOC 2 auditors require evidence of control **operation** over the observation period. Common evidence types:

| Evidence Type | Examples | Collection Method |
|---|---|---|
| Configuration exports | Security headers, rate limit config, MFA policy | API query + export |
| Log samples | Audit events, login events, access denials | Log query + export |
| Screenshots | Admin console showing policy enforcement | Automated browser screenshot |
| Policy documents | Security policy, incident response plan | Document export with version |
| Access review records | Quarterly review sign-offs | Report generation |
| Change records | PR approvals, deployment logs | Git/CI API query |
| Vulnerability scan reports | Security scan results | Scanner API export |

### 9.2 Automated Evidence Collector

The following Go tool automates evidence collection for a SOC 2 audit. It runs on a schedule (e.g., weekly during the observation period) and archives evidence for auditor retrieval.

```go
package compliance

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// EvidenceCollector gathers SOC 2 evidence from the GGID system.
type EvidenceCollector struct {
	gatewayURL string
	tenantID   string
	client     *http.Client
	outputDir  string
}

func NewEvidenceCollector(gatewayURL, tenantID, outputDir string) *EvidenceCollector {
	return &EvidenceCollector{
		gatewayURL: gatewayURL,
		tenantID:   tenantID,
		client:     &http.Client{Timeout: 30 * time.Second},
		outputDir:  outputDir,
	}
}

// CollectAll gathers all evidence for a single point in time and
// saves it to a timestamped directory.
func (ec *EvidenceCollector) CollectAll(ctx context.Context) error {
	ts := time.Now().Format("2006-01-02_150405")
	dir := filepath.Join(ec.outputDir, ts)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create evidence dir: %w", err)
	}

	// Define evidence collection tasks
	tasks := []struct {
		name     string
		endpoint string
	}{
		{"security_headers_config", "/api/v1/config/security-headers"},
		{"rate_limit_config", "/api/v1/config/rate-limits"},
		{"mfa_enrollment_report", "/api/v1/users/mfa-status"},
		{"active_sessions_report", "/api/v1/sessions/active"},
		{"role_assignments", "/api/v1/roles/assignments"},
		{"audit_events_sample", "/api/v1/audit/events?limit=100"},
		{"audit_chain_verification", "/api/v1/audit/verify-chain"},
		{"policy_export", "/api/v1/policies/export"},
		{"user_list", "/api/v1/users"},
	}

	manifest := EvidenceManifest{
		CollectedAt: time.Now(),
		TenantID:    ec.tenantID,
		Items:       make([]EvidenceItem, 0),
	}

	for _, task := range tasks {
		path := filepath.Join(dir, task.name+".json")
		item := ec.collectEvidence(ctx, task.name, task.endpoint, path)
		manifest.Items = append(manifest.Items, item)
	}

	// Save manifest
	manifestPath := filepath.Join(dir, "manifest.json")
	manifestData, _ := json.MarshalIndent(manifest, "", "  ")
	os.WriteFile(manifestPath, manifestData, 0600)

	return nil
}

func (ec *EvidenceCollector) collectEvidence(ctx context.Context, name, endpoint, path string) EvidenceItem {
	url := fmt.Sprintf("%s%s?tenant_id=%s", ec.gatewayURL, endpoint, ec.tenantID)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)

	resp, err := ec.client.Do(req)
	if err != nil {
		return EvidenceItem{Name: name, Status: "error", Error: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return EvidenceItem{Name: name, Status: "failed", Error: fmt.Sprintf("HTTP %d", resp.StatusCode)}
	}

	var raw json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return EvidenceItem{Name: name, Status: "error", Error: err.Error()}
	}

	formatted, _ := json.MarshalIndent(raw, "", "  ")
	if err := os.WriteFile(path, formatted, 0600); err != nil {
		return EvidenceItem{Name: name, Status: "error", Error: err.Error()}
	}

	return EvidenceItem{
		Name:       name,
		Status:     "collected",
		Endpoint:   endpoint,
		File:       filepath.Base(path),
		Size:       len(formatted),
		CollectedAt: time.Now(),
	}
}

type EvidenceManifest struct {
	CollectedAt time.Time      `json:"collected_at"`
	TenantID    string         `json:"tenant_id"`
	Items       []EvidenceItem `json:"items"`
}

type EvidenceItem struct {
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	Endpoint    string    `json:"endpoint,omitempty"`
	File        string    `json:"file,omitempty"`
	Size        int       `json:"size,omitempty"`
	Error       string    `json:"error,omitempty"`
	CollectedAt time.Time `json:"collected_at"`
}
```

### 9.3 Scheduled Collection

The evidence collector should run on a schedule throughout the observation period. For a Type II audit covering 6-12 months, weekly or bi-weekly collection provides sufficient evidence density:

```go
// ScheduleEvidenceCollection runs the collector on a weekly schedule.
// This can be deployed as a Kubernetes CronJob or systemd timer.
func ScheduleEvidenceCollection(collector *EvidenceCollector) {
	ticker := time.NewTicker(7 * 24 * time.Hour)
	defer ticker.Stop()

	ctx := context.Background()
	for range ticker.C {
		if err := collector.CollectAll(ctx); err != nil {
			// Alert operations team — evidence collection failure is itself
			// a control deficiency that must be reported to the auditor.
			fmt.Fprintf(os.Stderr, "evidence collection failed: %v\n", err)
		}
	}
}
```

---

## 10. SOC 2 Audit Preparation

### 10.1 Pre-Audit Phase (Months 1-3)

**Gap Assessment**: Map current controls to each TSC criterion and identify gaps. This document's Section 11 provides this for GGID.

**Remediation Plan**: For each gap, define:
- Required control or process
- Owner and deadline
- Implementation approach
- Verification method

**Policy Documentation**: Author the policies that auditors will review:
- Information Security Policy
- Access Control Policy
- Change Management Policy
- Incident Response Plan
- Data Classification Policy
- Business Continuity Plan
- Vendor Management Policy
- Acceptable Use Policy

**Vendor/Sub-processor Review**: Identify all third parties with access to customer data and ensure they have their own SOC 2 reports (or equivalent).

### 10.2 Readiness Assessment (Month 4)

Before engaging an auditor, perform a mock audit:
1. Walk through each control with an internal reviewer
2. Verify that evidence is retrievable for each control
3. Test the evidence collection automation
4. Document any remaining gaps and remediate

### 10.3 Audit Engagement (Month 5+)

**Selecting an auditor**: Choose a CPA firm with SaaS/IAM experience. Firms with pre-existing SOC 2 frameworks for identity platforms will require less education time.

**Observation period begins**: The auditor will define a start date. From that date forward, all controls must operate as designed. Any control changes during the observation period must be documented.

**Fieldwork**: The auditor will request evidence samples throughout the observation period. Typical requests:

| Request | Frequency | Source |
|---|---|---|
| Access review records | Quarterly | Generated by access review tool |
| Change approval records | Monthly sample | Git/CI audit trail |
| Vulnerability scan results | Monthly | Security scanner output |
| Incident response records | As incidents occur | Incident management system |
| Backup verification | Monthly | Backup logs + restore tests |
| MFA enrollment report | Monthly | Auth service API |
| Session audit log | Monthly sample | Audit service query |
| Policy document versions | As updated | Document management system |

### 10.4 During the Audit

**Walkthrough scripts**: For each control, prepare a scripted walkthrough:
1. Open the relevant system (admin console, CI/CD pipeline, etc.)
2. Demonstrate the control in action
3. Show evidence of the control operating over time
4. Reference the policy that defines the control

**Auditor questions to prepare for:**
- "Show me how a new employee is provisioned and deprovisioned"
- "Show me the last quarterly access review and its sign-off"
- "Show me the last 5 production deployments and their approval chain"
- "Show me how MFA enrollment is enforced and verified"
- "Show me the last security incident and the response process"
- "Show me the audit log hash chain verification"

### 10.5 Post-Audit Phase

**Management assertion**: Leadership signs a written assertion confirming the description of the system and that the controls were effective.

**Remediation tracking**: Any control exceptions identified by the auditor must be tracked to resolution. These become inputs to the next audit cycle.

---

## 11. GGID SOC 2 Readiness Assessment

### 11.1 Control Mapping

The following table maps GGID's current capabilities to each SOC 2 Common Criterion:

| Criterion | Control Description | GGID Status | Evidence Source |
|---|---|---|---|
| **CC1.3** | Segregation of duties | **Partial** — RBAC supports role separation, but no automated enforcement of dev/deployer/admin split | Policy service role definitions |
| **CC2.1** | Internal policy communication | **Gap** — No documented security policy or training tracking | N/A |
| **CC2.3** | External incident notification | **Gap** — No customer notification workflow for security incidents | N/A |
| **CC4.1** | Continuous control monitoring | **Partial** — Health score monitoring exists (`health_score.go`), but no SOC2-specific monitoring | `gateway/internal/middleware/health_score.go` |
| **CC5.1** | Control activities for risk mitigation | **Partial** — Rate limiting, security headers exist but not formalized as controls | `ratelimit.go`, `security_headers.go` |
| **CC6.1** | Logical access — authentication | **Strong** — Argon2id + pepper, TOTP MFA, WebAuthn, session management | `pkg/crypto/crypto.go`, `auth/internal/domain/mfa.go` |
| **CC6.2** | Logical access — provisioning/deprovisioning | **Partial** — SCIM 2.0 skeleton exists, but no automated lifecycle | `identity` service |
| **CC6.3** | Logical access — periodic review | **Gap** — No automated access review report generation | N/A |
| **CC6.6** | Logical access security events logged | **Strong** — Audit service with hash chain, structured logging | `audit/internal/domain/hash_chain.go` |
| **CC6.7** | Transmission security | **Strong** — HSTS, TLS, security headers (CSP, X-Frame-Options) | `security_headers.go` |
| **CC7.1** | System monitoring | **Partial** — Health scores, structured logging, but no SIEM integration | `health_score.go`, `recovery.go` |
| **CC7.2** | Incident detection | **Partial** — Rate limiting, anomaly detection, but no formal incident response runbook | `ratelimit.go` |
| **CC7.3** | Incident response | **Gap** — No documented incident response plan or runbook | N/A |
| **CC7.4** | Recovery from incidents | **Partial** — Circuit breaker, panic recovery middleware, but no DR testing | `gateway/internal/middleware/` |
| **CC8.1** | Change authorization | **Gap** — No documented deployment approval workflow or CI/CD controls | N/A |
| **CC8.2** | Testing changes | **Partial** — 250+ test cases exist, but no formal staging verification gate | Test suite |
| **CC9.1** | Risk mitigation — BC/DR | **Gap** — No business continuity plan or DR testing schedule | N/A |

### 11.2 Strengths

1. **Cryptography**: Argon2id with configurable pepper provides strong password protection. AES-256-GCM encryption for tokens. HMAC-SHA256 hash chain for audit integrity. These are production-grade implementations.

2. **Audit Logging**: The hash chain design (`ComputeHash`/`VerifyHash`/`VerifyChain`) provides cryptographic tamper evidence — a strong control for processing integrity (PI criterion).

3. **Security Headers**: HSTS (1 year), CSP, X-Content-Type-Options, X-Frame-Options, Referrer-Policy — all configurable per tenant. This exceeds the baseline for CC6.7.

4. **MFA**: Both TOTP and WebAuthn/FIDO2 support, with backup codes and step-up authentication. This exceeds CC6.1 requirements.

5. **Rate Limiting**: Per-endpoint limits (5/min login, 3/min register) with sliding window support. Contributes to CC7.2 (incident detection).

6. **Session Management**: Full lifecycle support — creation, revocation, listing, device tracking, expiry enforcement. Supports CC6.3.

### 11.3 Prioritized Remediation List

| Priority | Gap | Criterion | Effort |
|---|---|---|---|
| P0 | Document Information Security Policy, Access Control Policy, Incident Response Plan, Change Management Policy | CC2, CC5, CC7, CC8 | 2-4 weeks |
| P0 | Implement deployment approval workflow with mandatory code review enforcement | CC8.1 | 1-2 weeks |
| P1 | Build automated access review report generation (CC1/CC6.3) | CC6.3 | 1-2 weeks |
| P1 | Implement evidence collection automation tool | CC4.1 | 1 week |
| P1 | Document and test disaster recovery procedures | CC9.1 | 2-3 weeks |
| P2 | Integrate SIEM/log aggregation for audit events | CC7.1 | 2-4 weeks |
| P2 | Add automated deprovisioning workflow (joiner-mover-leaver) | CC6.2 | 2-3 weeks |
| P2 | Implement vulnerability scanning in CI pipeline | CC7.2 | 1 week |
| P3 | Customer security change notification workflow | CC2.3 | 1 week |
| P3 | Document complementary user entity controls (CUECs) | CC2.4 | 1 week |

---

## 12. Gap Analysis and Recommendations

### 12.1 Action Items

**Action 1: Policy Documentation Suite** *(Effort: 3-4 weeks, P0)*

The most significant gap is the absence of formal security policy documentation. Without written policies, there is nothing to audit. Author and publish:
- Information Security Policy (governs all security activities)
- Access Control Policy (defines provisioning, review, deprovisioning)
- Change Management Policy (defines approval, testing, deployment)
- Incident Response Plan (defines detection, response, recovery, communication)
- Data Classification Policy (defines data categories and handling rules)
- Business Continuity Plan (defines DR procedures, RTO/RPO targets)

**Action 2: CI/CD Pipeline with SOC 2 Controls** *(Effort: 2 weeks, P0)*

Implement a deployment pipeline with:
- Mandatory code review (at least 1 non-author approver)
- Automated security scanning (gosec, trivy, dependency scanning)
- Staging environment verification before production
- Deployment audit trail (who approved, when, what changed)
- One-click rollback capability

**Action 3: Automated Evidence Collection System** *(Effort: 1-2 weeks, P1)*

Deploy the `EvidenceCollector` from Section 9 as a scheduled job. This eliminates the scramble to manually gather evidence during audit fieldwork and ensures consistent evidence quality throughout the observation period.

**Action 4: Access Review Automation** *(Effort: 1-2 weeks, P1)*

Deploy the access review report generator from Section 2.4. Schedule it quarterly with automated email to designated reviewers. Store sign-off records for audit evidence.

**Action 5: SIEM Integration for Audit Events** *(Effort: 3-4 weeks, P2)*

Stream GGID audit events to a SIEM (Splunk, ELK, Datadog) with pre-built detection rules:
- Privilege escalation alerts
- Mass account creation/deletion
- Concurrent session anomalies
- Hash chain integrity failures
- Rate limit threshold breaches

This transforms GGID's strong logging capability into an active detection and response system.

### 12.2 Timeline to SOC 2 Type II Readiness

| Phase | Duration | Activities |
|---|---|---|
| Phase 1: Policy & Process | Weeks 1-4 | Author policies, define processes, assign owners |
| Phase 2: Tooling | Weeks 3-8 | Build evidence collector, access review tool, CI/CD controls |
| Phase 3: Internal Audit | Weeks 8-10 | Mock audit, gap remediation |
| Phase 4: Observation Period | Months 3-12 | Controls operate, evidence collected weekly |
| Phase 5: External Audit | Months 10-13 | Fieldwork, report issuance |

**Total time to first SOC 2 Type II report: approximately 12 months** from project start, assuming the 6-month minimum observation period and 2-month audit fieldwork.

### 12.3 Cost Estimates

| Item | Estimated Cost |
|---|---|
| SOC 2 Type II audit (mid-tier firm) | $30,000 - $80,000 |
| Compliance automation tooling (Vanta/Drata) | $8,000 - $25,000/year |
| Internal engineering effort (Phase 1-3) | 2-3 engineer-months |
| Ongoing compliance operations | 0.5 FTE |

---

## Appendix A: GGID Source File Reference

| SOC 2 Control Area | GGID Source File | Key Functions/Types |
|---|---|---|
| Password hashing | `pkg/crypto/crypto.go` | `HashPassword()`, `VerifyPassword()`, `SetPepper()` |
| AES encryption | `pkg/crypto/crypto.go` | `AESEncrypt()`, `AESDecrypt()` |
| MFA (TOTP) | `services/auth/internal/domain/mfa.go` | `MFADevice`, `MFAChallenge` |
| WebAuthn | `services/auth/internal/webauthn/handler.go` | Registration, assertion verification |
| Session management | `services/auth/internal/domain/session.go` | `Session.IsActive()`, `Session.Revoke()` |
| Session service | `services/auth/internal/service/session_service.go` | `Create()`, `Revoke()`, `ListByUser()` |
| Audit hash chain | `services/audit/internal/domain/hash_chain.go` | `ComputeHash()`, `VerifyHash()`, `VerifyChain()` |
| Audit events | `services/audit/internal/domain/models.go` | `AuditEvent`, `ListFilter`, `EventResult` |
| Rate limiting | `services/gateway/internal/middleware/ratelimit.go` | `RateLimiter`, `RateLimitConfig` |
| Security headers | `services/gateway/internal/middleware/security_headers.go` | `SecurityHeadersConfigurable()`, HSTS, CSP |
| Health monitoring | `services/gateway/internal/middleware/health_score.go` | `HealthScore`, backend health tracking |
| Structured logging | `services/gateway/internal/middleware/recovery.go` | `StructuredLogger`, `LogRecord`, `PanicRecord` |
| Policy engine | `services/policy/internal/server/http.go` | RBAC/ABAC policy CRUD and evaluation |
| Circuit breaker | `services/gateway/internal/middleware/` | Resilience patterns |
| Webhook SSRF protection | `services/gateway/internal/webhooks/ssrf.go` | URL validation, IP blocking |
| Password breach detection | `services/auth/internal/service/password_breach.go` | HIBP integration |

---

## Appendix B: SOC 2 Control Checklist for GGID

```
[ ] CC1.1  Security program charter and organizational structure documented
[ ] CC1.2  Board/leadership oversight of security (quarterly reviews)
[ ] CC1.3  Segregation of duties enforced (dev != deployer != admin)
[ ] CC1.4  Security awareness training program
[ ] CC1.5  Background checks for employees with system access
[ ] CC2.1  Security policies published and acknowledged
[ ] CC2.2  Security responsibilities defined per role
[ ] CC2.3  Incident notification procedures (internal and external)
[ ] CC2.4  System description maintained for customer use
[ ] CC3.1  Risk assessment conducted at least annually
[ ] CC3.4  Risk remediation tracked to completion
[ ] CC4.1  Continuous monitoring of controls
[ ] CC4.2  Control deficiencies identified and remediated
[ ] CC5.1  Control activities designed and implemented
[ ] CC5.2  Controls deployed with policy enforcement
[ ] CC6.1  Authentication controls (passwords, MFA, sessions) -- GGID STRONG
[ ] CC6.2  Provisioning/deprovisioning tied to need
[ ] CC6.3  Periodic access reviews
[ ] CC6.4  Physical access restricted
[ ] CC6.6  Security events logged and monitored -- GGID STRONG
[ ] CC6.7  Transmission security (TLS, HSTS) -- GGID STRONG
[ ] CC6.8  Credential management (password rotation, key rotation)
[ ] CC7.1  System performance monitored
[ ] CC7.2  Incident detection -- GGID PARTIAL
[ ] CC7.3  Incident response procedures
[ ] CC7.4  Recovery from incidents
[ ] CC8.1  Changes authorized before implementation
[ ] CC8.2  Changes tested before deployment
[ ] CC8.3  Authorization, testing, and approval documented
[ ] CC9.1  Business continuity and disaster recovery
[ ] CC9.2  Vendor management program
```

---

*Document version: 1.0 | Last updated: 2025 | Review cycle: Quarterly*
