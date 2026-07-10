# Privileged Access Management (PAM) for IAM Systems

**Document Type:** Security Research
**Target System:** GGID — Go-based Identity and Access Management Suite
**Author:** Security Research Team
**Status:** Final

---

## Table of Contents

1. [PAM Concepts for IAM](#1-pam-concepts-for-iam)
2. [Just-in-Time (JIT) Elevation](#2-just-in-time-jit-elevation)
3. [Approval Workflow](#3-approval-workflow)
4. [Break-Glass Accounts](#4-break-glass-accounts)
5. [Session Recording for Privileged Sessions](#5-session-recording-for-privileged-sessions)
6. [Command Audit and Behavioral Analysis](#6-command-audit-and-behavioral-analysis)
7. [Privilege Escalation Prevention](#7-privilege-escalation-prevention)
8. [GGID PAM Gap Analysis](#8-ggid-pam-gap-analysis)
9. [Gap Analysis & Recommendations](#9-gap-analysis--recommendations)

---

## 1. PAM Concepts for IAM

### What PAM Means in an IAM Context

Privileged Access Management (PAM) is the discipline of controlling, monitoring, and auditing
elevated (administrative) access to critical systems. In the context of an IAM system like GGID,
PAM is especially critical because IAM administrators wield powers that no other role has:

- **Create and delete users** — an admin can provision accounts for anyone in the tenant.
- **Modify roles and permissions** — an admin can grant themselves or others arbitrary permissions,
  effectively becoming a super-user.
- **View all tenant data** — admins can read PII, credentials metadata, and audit logs across the
  entire tenant scope.
- **Configure authentication** — admins control MFA enrollment requirements, SSO providers,
  WebAuthn policies, and session lifetimes.
- **Manage API keys** — admins can create long-lived API keys that bypass interactive login.

### Why IAM Systems Need Their Own PAM

An IAM system is the "keys to the kingdom." If an attacker compromises an IAM admin account, they
can:

1. Create a backdoor user with elevated privileges.
2. Disable MFA for sensitive accounts.
3. Modify audit configurations to cover their tracks.
4. Export all user data (mass PII exfiltration).
5. Inject malicious OAuth client registrations.

This makes IAM admin accounts the highest-value targets in the entire infrastructure. Traditional
RBAC alone is insufficient because it grants **standing privileges** — the admin has full power 24/7,
even when they only need it for a 15-minute configuration change.

### PAM vs RBAC vs ABAC

| Aspect | RBAC | ABAC | PAM |
|--------|------|------|-----|
| **What it defines** | Which roles can do what | Which attributes grant/deny access | How privileged access is requested, approved, monitored, and revoked |
| **Granularity** | Coarse (role-level) | Fine (attribute-level) | Temporal + contextual (time-bound, approval-bound) |
| **Persistence** | Standing (permanent until changed) | Standing or policy-driven | Ephemeral (time-bound elevation) |
| **Approval** | None | None | Required (multi-level) |
| **Audit** | Role assignment events | Policy evaluation decisions | Full session recording, command logging, anomaly detection |
| **Example** | "Admin role can delete users" | "Deny if user.department != eng" | "Admin requests 1-hour elevation, approved by 2 people, session recorded" |

PAM is **orthogonal** to RBAC/ABAC: it governs the lifecycle of privileged role assignments,
adding time-bounds, approvals, monitoring, and break-glass procedures on top of the existing
authorization model.

### Key PAM Controls

- **Just-in-Time (JIT) elevation** — privileges granted only when needed, auto-expire.
- **Approval workflows** — sensitive actions require multi-person sign-off.
- **Break-glass accounts** — emergency access when normal auth fails.
- **Session recording** — full capture of admin sessions for forensic review.
- **Command audit** — every privileged action logged with full context.
- **Anomaly detection** — behavioral baselines flag suspicious admin activity.
- **Privilege escallation prevention** — strict tenant boundaries and role inheritance guards.

---

## 2. Just-in-Time (JIT) Elevation

### How JIT Works

JIT elevation replaces standing privileges with a request-approve-grant-expire lifecycle:

```
User requests elevation
    → Request includes: role, duration, justification, scope
    → Approver(s) notified (email/Slack/NATS event)
    → On approval: time-bound role assigned (e.g., 1 hour)
    → Role auto-expires at end of duration
    → Session recorded for the entire elevated period
    → Audit event emitted with full context
```

### Why JIT Is Better Than Standing Privileges

- **Reduced attack surface** — if an admin account is compromised, the attacker only has elevated
  access during the brief elevation window, not 24/7.
- **Mandatory justification** — every elevation has a documented business reason.
- **Approval trail** — who approved what and when is immutable.
- **Natural rate-limiting** — frequent elevation requests signal a need for permanent role
  reassignment or a process problem.

### Dual-Control Principle for Super-Admin

For the highest-privilege operations (tenant deletion, security policy modification, break-glass
unlock), dual-control (two-person rule) is required:

- Two different approvers must independently approve.
- The requester cannot be one of the approvers.
- Both approvals must be within a time window (e.g., 30 minutes of each other).

### Go Code: JIT Elevation Handler

```go
package pam

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// ElevationRequest represents a user's request for temporary privilege elevation.
type ElevationRequest struct {
	ID           uuid.UUID  `json:"id"`
	UserID       uuid.UUID  `json:"user_id"`
	TenantID     uuid.UUID  `json:"tenant_id"`
	RoleID       uuid.UUID  `json:"role_id"`       // Target role to elevate to
	Duration     string     `json:"duration"`       // e.g., "1h", "30m"
	Justification string    `json:"justification"` // Business reason
	Scope        string     `json:"scope"`         // Resource scope if applicable
	Status       string     `json:"status"`        // pending, approved, denied, expired
	RequestedAt  time.Time  `json:"requested_at"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	ApprovedBy   []uuid.UUID `json:"approved_by,omitempty"`
}

// JITHandler processes just-in-time elevation requests.
type JITHandler struct {
	approverChain *ApprovalChain
	store         ElevationStore
	notifier      Notifier
}

// ElevationStore persists elevation requests (Redis or PostgreSQL in production).
type ElevationStore interface {
	SaveElevation(ctx context.Context, req *ElevationRequest) error
	GetElevation(ctx context.Context, id uuid.UUID) (*ElevationRequest, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string, approvedBy []uuid.UUID) error
	ListActiveForUser(ctx context.Context, userID uuid.UUID) ([]*ElevationRequest, error)
	ListPending(ctx context.Context, tenantID uuid.UUID) ([]*ElevationRequest, error)
}

// Notifier sends approval notifications (NATS, email, Slack).
type Notifier interface {
	NotifyApprovalRequested(ctx context.Context, req *ElevationRequest, approvers []uuid.UUID) error
	NotifyElevationGranted(ctx context.Context, req *ElevationRequest) error
	NotifyElevationDenied(ctx context.Context, req *ElevationRequest, reason string) error
}

// RequestElevation handles POST /api/v1/pam/elevate
func (h *JITHandler) RequestElevation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var body struct {
		RoleID        string `json:"role_id"`
		Duration      string `json:"duration"`
		Justification string `json:"justification"`
		Scope         string `json:"scope"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if body.Justification == "" {
		writeError(w, http.StatusBadRequest, "justification is required")
		return
	}

	duration, err := time.ParseDuration(body.Duration)
	if err != nil || duration <= 0 || duration > 8*time.Hour {
		writeError(w, http.StatusBadRequest, "duration must be between 1m and 8h")
		return
	}

	userID := userIDFromContext(r.Context())
	tenantID := tenantIDFromContext(r.Context())

	roleID, err := uuid.Parse(body.RoleID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid role_id")
		return
	}

	elevReq := &ElevationRequest{
		ID:            uuid.New(),
		UserID:        userID,
		TenantID:      tenantID,
		RoleID:        roleID,
		Duration:      body.Duration,
		Justification: body.Justification,
		Scope:         body.Scope,
		Status:        "pending",
		RequestedAt:   time.Now().UTC(),
	}

	// Determine required approvers based on role sensitivity
	approvers, dualControl := h.approverChain.GetApprovers(tenantID, roleID)
	if dualControl && len(approvers) < 2 {
		writeError(w, http.StatusInternalServerError, "dual-control requires 2+ approvers configured")
		return
	}

	if err := h.store.SaveElevation(r.Context(), elevReq); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save elevation request")
		return
	}

	// Notify approvers asynchronously
	go h.notifier.NotifyApprovalRequested(context.Background(), elevReq, approvers)

	writeJSON(w, http.StatusAccepted, map[string]any{
		"status":        "pending_approval",
		"request_id":    elevReq.ID.String(),
		"dual_control":  dualControl,
		"approvers":     len(approvers),
		"auto_deny_at":  elevReq.RequestedAt.Add(2 * time.Hour).Format(time.RFC3339),
	})
}

// ApproveElevation handles POST /api/v1/pam/elevate/{id}/approve
func (h *JITHandler) ApproveElevation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	reqID, err := uuid.Parse(pathParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid elevation request ID")
		return
	}

	elevReq, err := h.store.GetElevation(r.Context(), reqID)
	if err != nil {
		writeError(w, http.StatusNotFound, "elevation request not found")
		return
	}

	if elevReq.Status != "pending" {
		writeError(w, http.StatusConflict, fmt.Sprintf("request already %s", elevReq.Status))
		return
	}

	// Check SLA: auto-deny after 2 hours
	if time.Since(elevReq.RequestedAt) > 2*time.Hour {
		h.store.UpdateStatus(r.Context(), reqID, "denied", nil)
		writeError(w, http.StatusGone, "elevation request expired (SLA timeout)")
		return
	}

	approverID := userIDFromContext(r.Context())

	// Self-approval check
	if approverID == elevReq.UserID {
		writeError(w, http.StatusForbidden, "cannot self-approve elevation")
		return
	}

	// Check if this approver is authorized
	approvers, dualControl := h.approverChain.GetApprovers(elevReq.TenantID, elevReq.RoleID)
	if !contains(approvers, approverID) {
		writeError(w, http.StatusForbidden, "not an authorized approver for this role")
		return
	}

	// Add approver
	if contains(elevReq.ApprovedBy, approverID) {
		writeError(w, http.StatusConflict, "already approved by this user")
		return
	}
	elevReq.ApprovedBy = append(elevReq.ApprovedBy, approverID)

	// Check if enough approvals
	requiredApprovals := 1
	if dualControl {
		requiredApprovals = 2
	}

	if len(elevReq.ApprovedBy) >= requiredApprovals {
		// Grant elevation
		duration, _ := time.ParseDuration(elevReq.Duration)
		expiresAt := time.Now().UTC().Add(duration)
		elevReq.Status = "approved"
		elevReq.ExpiresAt = &expiresAt

		h.store.UpdateStatus(r.Context(), reqID, "approved", elevReq.ApprovedBy)

		// Assign time-bound role to user
		h.assignTimeBoundRole(r.Context(), elevReq.UserID, elevReq.RoleID, elevReq.TenantID, expiresAt)

		// Start session recording for elevated session
		h.startSessionRecording(r.Context(), elevReq)

		go h.notifier.NotifyElevationGranted(context.Background(), elevReq)
	} else {
		h.store.UpdateStatus(r.Context(), reqID, "pending", elevReq.ApprovedBy)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":            elevReq.Status,
		"approvals":         len(elevReq.ApprovedBy),
		"approvals_needed":  requiredApprovals,
		"expires_at":        elevReq.ExpiresAt,
	})
}

func (h *JITHandler) assignTimeBoundRole(ctx context.Context, userID, roleID, tenantID uuid.UUID, expiresAt time.Time) {
	// Delegates to policy service RoleService.AssignRole with ExpiresAt
	// roleSvc.AssignRole(ctx, userID, roleID, domain.ScopeOrganization, tenantID, systemUserID, &expiresAt)
}

func (h *JITHandler) startSessionRecording(ctx context.Context, req *ElevationRequest) {
	// Initialize session recorder for the elevated user
}

func contains(ids []uuid.UUID, target uuid.UUID) bool {
	for _, id := range ids {
		if id == target {
			return true
		}
	}
	return false
}
```

---

## 3. Approval Workflow

### Multi-Level Approval Chain

For sensitive operations, GGID needs a configurable approval chain:

```
Level 1: Direct Manager     — confirms business need
Level 2: Security Officer   — confirms risk is acceptable
Level 3: Tenant Owner       — for cross-tenant or system-level operations
```

### SLA Timers

| Risk Level | Auto-Approve After | Auto-Deny After | Required Approvers |
|------------|-------------------|-----------------|-------------------|
| Low (read-only) | 1 hour | 4 hours | 1 |
| Medium (config change) | Never | 2 hours | 1 |
| High (user modification) | Never | 1 hour | 2 (dual-control) |
| Critical (security policy) | Never | 30 min | 2 (dual-control) |

### Go Code: Approval Workflow Engine with NATS

```go
package pam

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/google/uuid"
)

// ApprovalChain determines who needs to approve a given operation.
type ApprovalChain struct {
	mu          sync.RWMutex
	configs     map[uuid.UUID]*ApprovalConfig // tenantID → config
	nc          *nats.Conn
}

// ApprovalConfig defines the approval policy for a tenant.
type ApprovalConfig struct {
	TenantID uuid.UUID
	Rules    []ApprovalRule
}

// ApprovalRule maps a role or operation to its approval requirements.
type ApprovalRule struct {
	RoleKey          string        // "super-admin", "user-admin", etc.
	RequiredApprovals int          // 1 = single, 2 = dual-control
	ApproverRoleKeys []string      // Who can approve (e.g., ["security-officer", "tenant-owner"])
	SLAAutoDeny      time.Duration // Auto-deny if not approved within this window
	SLAAutoApprove   time.Duration // Auto-approve after this for low-risk (0 = never)
	RiskLevel        string        // "low", "medium", "high", "critical"
}

// GetApprovers returns the list of approver user IDs for a given role elevation.
func (ac *ApprovalChain) GetApprovers(tenantID, roleID uuid.UUID) ([]uuid.UUID, bool) {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	cfg := ac.configs[tenantID]
	if cfg == nil {
		return nil, false
	}

	for _, rule := range cfg.Rules {
		if rule.RoleKey == roleID.String() { // Simplified; real impl resolves key
			dualControl := rule.RequiredApprovals >= 2
			// In production, query the identity service for users with approver role keys
			return ac.resolveApprovers(tenantID, rule.ApproverRoleKeys), dualControl
		}
	}
	return nil, false
}

func (ac *ApprovalChain) resolveApprovers(tenantID uuid.UUID, roleKeys []string) []uuid.UUID {
	// Query identity service: SELECT user_id FROM user_roles WHERE role_key IN (...)
	return []uuid.UUID{} // placeholder
}

// NATSApprovalNotifier sends approval events over NATS JetStream.
type NATSApprovalNotifier struct {
	nc *nats.Conn
	js nats.JetStreamContext
}

func NewNATSApprovalNotifier(nc *nats.Conn) (*NATSApprovalNotifier, error) {
	js, err := nc.JetStream()
	if err != nil {
		return nil, err
	}
	// Ensure the approval stream exists
	_, err = js.AddStream(&nats.StreamConfig{
		Name:     "PAM_APPROVALS",
		Subjects: []string{"pam.approval.>"},
		Storage:  nats.FileStorage,
		MaxAge:   7 * 24 * time.Hour, // 7-day retention
	})
	if err != nil {
		return nil, err
	}
	return &NATSApprovalNotifier{nc: nc, js: js}, nil
}

func (n *NATSApprovalNotifier) NotifyApprovalRequested(ctx context.Context, req *ElevationRequest, approvers []uuid.UUID) error {
	event := map[string]any{
		"event_type":     "approval_requested",
		"request_id":     req.ID.String(),
		"user_id":        req.UserID.String(),
		"tenant_id":      req.TenantID.String(),
		"role_id":        req.RoleID.String(),
		"duration":       req.Duration,
		"justification":  req.Justification,
		"approvers":      approvers,
		"requested_at":   req.RequestedAt.Format(time.RFC3339),
	}
	data, _ := json.Marshal(event)
	subject := fmt.Sprintf("pam.approval.requested.%s", req.TenantID)
	_, err := n.js.Publish(subject, data)
	return err
}

func (n *NATSApprovalNotifier) NotifyElevationGranted(ctx context.Context, req *ElevationRequest) error {
	event := map[string]any{
		"event_type":  "elevation_granted",
		"request_id":  req.ID.String(),
		"user_id":     req.UserID.String(),
		"expires_at":  req.ExpiresAt,
	}
	data, _ := json.Marshal(event)
	subject := fmt.Sprintf("pam.approval.granted.%s", req.TenantID)
	_, err := n.js.Publish(subject, data)
	return err
}

func (n *NATSApprovalNotifier) NotifyElevationDenied(ctx context.Context, req *ElevationRequest, reason string) error {
	event := map[string]any{
		"event_type":  "elevation_denied",
		"request_id":  req.ID.String(),
		"user_id":     req.UserID.String(),
		"reason":      reason,
	}
	data, _ := json.Marshal(event)
	subject := fmt.Sprintf("pam.approval.denied.%s", req.TenantID)
	_, err := n.js.Publish(subject, data)
	return err
}

// SLAMonitor runs as a background goroutine, checking for pending requests
// that have exceeded their SLA window.
func (ac *ApprovalChain) SLAMonitor(ctx context.Context, store ElevationStore, notifier Notifier) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ac.checkSLAs(store, notifier)
		}
	}
}

func (ac *ApprovalChain) checkSLAs(store ElevationStore, notifier Notifier) {
	// Iterate all pending requests, check auto-approve/auto-deny timers
	// Auto-approve for low-risk: if time.Since(req.RequestedAt) > rule.SLAAutoApprove → approve
	// Auto-deny for high-risk: if time.Since(req.RequestedAt) > rule.SLAAutoDeny → deny
}
```

---

## 4. Break-Glass Accounts

### Purpose

Break-glass accounts provide emergency access when normal authentication is unavailable:

- Auth service is down (database failure, network partition).
- All admins are locked out (MFA device loss, credential rotation error).
- Disaster recovery scenario requiring immediate system access.

### Break-Glass Account Creation

```
1. Generate strong random password (256-bit entropy, crypto/rand)
2. Hash password with Argon2id (same as normal users)
3. Seal the plaintext password in an envelope:
   - Option A: Physical sealed envelope in a safe (traditional)
   - Option B: HSM-protected key split across 2+ key custodians
   - Option C: Shamir's Secret Sharing across 3 of 5 custodians
4. Configure the account with:
   - No MFA (so it works even when TOTP infrastructure is down)
   - Elevated audit logging (every action flagged as break-glass)
   - IP allowlist restricted to incident-response VLAN
   - Rate-limited to 1 login per hour (prevents brute force)
5. Test break-glass quarterly (sealed envelope exercise)
```

### Usage Auditing

All break-glass account actions are:

- Logged with **elevated detail** — full request/response bodies, source IP, user agent, device fingerprint.
- **Real-time alerting** — Slack/email/SMS to the entire security team within 30 seconds.
- **Post-incident review** — mandatory within 24 hours of any break-glass usage.
- **Session recording** — full keystroke and screen capture.

### Go Code: Break-Glass Account Management

```go
package pam

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// BreakGlassManager manages emergency access accounts.
type BreakGlassManager struct {
	mu       sync.Mutex
	accounts map[uuid.UUID]*BreakGlassAccount
	notifier BreakGlassNotifier
}

// BreakGlassAccount represents an emergency access credential.
type BreakGlassAccount struct {
	ID              uuid.UUID  `json:"id"`
	TenantID        uuid.UUID  `json:"tenant_id"`
	Username        string     `json:"username"`
	PasswordHash    string     `json:"-"`           // Argon2id hash
	SealedPassword  string     `json:"-"`           // HSM or envelope-sealed plaintext
	Status          string     `json:"status"`      // "sealed", "opened", "used", "rotated"
	CreatedAt       time.Time  `json:"created_at"`
	LastUsedAt      *time.Time `json:"last_used_at,omitempty"`
	UsedBy          *uuid.UUID `json:"used_by,omitempty"`
	UsageReason     string     `json:"usage_reason,omitempty"`
	IPAllowlist     []string   `json:"ip_allowlist"` // Restricted to incident-response IPs
}

// BreakGlassNotifier sends real-time alerts.
type BreakGlassNotifier interface {
	AlertBreakGlassOpened(ctx context.Context, account *BreakGlassAccount, userIP string) error
	AlertBreakGlassAction(ctx context.Context, accountID uuid.UUID, action string, details map[string]any) error
}

// CreateBreakGlassAccount creates a new emergency access credential.
// The password is generated, hashed, and sealed. The plaintext is never stored in plaintext.
func (bgm *BreakGlassManager) CreateBreakGlassAccount(ctx context.Context, tenantID uuid.UUID, ipAllowlist []string) (*BreakGlassAccount, string, error) {
	// Generate 256-bit random password
	passwordBytes := make([]byte, 32)
	if _, err := rand.Read(passwordBytes); err != nil {
		return nil, "", fmt.Errorf("failed to generate random password: %w", err)
	}
	password := base64.URLEncoding.EncodeToString(passwordBytes)

	// Hash with Argon2id (production: use the existing pkg/crypto HashPassword)
	hash := hashPasswordArgon2id(password)

	// Seal the plaintext (production: use HSM or Shamir's Secret Sharing)
	sealed := sealWithHSM(password)

	account := &BreakGlassAccount{
		ID:             uuid.New(),
		TenantID:       tenantID,
		Username:       fmt.Sprintf("breakglass-%s", uuid.New().String()[:8]),
		PasswordHash:   hash,
		SealedPassword: sealed,
		Status:         "sealed",
		CreatedAt:      time.Now().UTC(),
		IPAllowlist:    ipAllowlist,
	}

	bgm.mu.Lock()
	bgm.accounts[account.ID] = account
	bgm.mu.Unlock()

	// The plaintext password is returned ONCE to be placed in a physical safe
	return account, password, nil
}

// OpenBreakGlass unseals an emergency account for use.
// This triggers immediate alerting.
func (bgm *BreakGlassManager) OpenBreakGlass(ctx context.Context, accountID uuid.UUID, userID uuid.UUID, reason string, userIP string) (*BreakGlassAccount, error) {
	bgm.mu.Lock()
	defer bgm.mu.Unlock()

	account, ok := bgm.accounts[accountID]
	if !ok {
		return nil, fmt.Errorf("break-glass account not found")
	}

	if account.Status != "sealed" {
		return nil, fmt.Errorf("break-glass account already opened (status: %s)", account.Status)
	}

	// Verify IP allowlist
	if !isIPAllowed(userIP, account.IPAllowlist) {
		return nil, fmt.Errorf("break-glass access from unauthorized IP: %s", userIP)
	}

	now := time.Now().UTC()
	account.Status = "opened"
	account.LastUsedAt = &now
	account.UsedBy = &userID
	account.UsageReason = reason

	// Immediate real-time alert
	go bgm.notifier.AlertBreakGlassOpened(context.Background(), account, userIP)

	return account, nil
}

// BreakGlassMiddleware wraps the main handler chain with break-glass detection.
// If a break-glass account is detected, it:
// 1. Enables enhanced logging (request/response body capture)
// 2. Sends a real-time alert for every API call
// 3. Applies a mandatory session timeout (30 minutes)
func (bgm *BreakGlassManager) BreakGlassMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := userIDFromContext(r.Context())

		bgm.mu.Lock()
		var bgAccount *BreakGlassAccount
		for _, acc := range bgm.accounts {
			if acc.UsedBy != nil && *acc.UsedBy == userID {
				bgAccount = acc
				break
			}
		}
		bgm.mu.Unlock()

		if bgAccount != nil {
			// Check session timeout
			if bgAccount.LastUsedAt != nil && time.Since(*bgAccount.LastUsedAt) > 30*time.Minute {
				http.Error(w, `{"error":"break-glass session expired"}`, http.StatusForbidden)
				return
			}

			// Enhanced audit logging
			go bgm.notifier.AlertBreakGlassAction(r.Context(), bgAccount.ID, r.URL.Path, map[string]any{
				"method":     r.Method,
				"remote_ip":  r.RemoteAddr,
				"user_agent": r.UserAgent(),
				"reason":     bgAccount.UsageReason,
			})
		}

		next.ServeHTTP(w, r)
	})
}

// RotateBreakGlassPassword generates a new password and re-seals the account.
// Called after each break-glass usage or quarterly, whichever comes first.
func (bgm *BreakGlassManager) RotateBreakGlassPassword(ctx context.Context, accountID uuid.UUID) (string, error) {
	bgm.mu.Lock()
	defer bgm.mu.Unlock()

	account, ok := bgm.accounts[accountID]
	if !ok {
		return "", fmt.Errorf("break-glass account not found")
	}

	// Generate new password
	passwordBytes := make([]byte, 32)
	if _, err := rand.Read(passwordBytes); err != nil {
		return "", err
	}
	newPassword := base64.URLEncoding.EncodeToString(passwordBytes)

	account.PasswordHash = hashPasswordArgon2id(newPassword)
	account.SealedPassword = sealWithHSM(newPassword)
	account.Status = "sealed"
	account.LastUsedAt = nil
	account.UsedBy = nil
	account.UsageReason = ""

	return newPassword, nil
}

// Stubs — implemented via pkg/crypto and HSM integration in production
func hashPasswordArgon2id(password string) string  { return "argon2id$..." }
func sealWithHSM(plaintext string) string           { return "sealed:" + plaintext }
func isIPAllowed(ip string, allowlist []string) bool { return true }
```

---

## 5. Session Recording for Privileged Sessions

### Recording Approaches

| Method | What It Captures | Storage Cost | Privacy Risk |
|--------|-----------------|-------------|-------------|
| Command logging | API calls, parameters | Low | Low |
| Keystroke logging | Every keystroke | Medium | High |
| Screen capture | Full visual replay | High | High |
| API request/response | Full payload | Medium | Medium |

### Privacy Considerations

- **Disclosure required** — users must be informed that sessions are being recorded (banner on login,
  documentation, employment agreement).
- **PII redaction** — recorded payloads containing passwords, tokens, and PII should be redacted
  before storage.
- **Retention limits** — recordings retained for 90 days by default, 1 year for regulated industries.
- **Access control** — recordings can only be accessed by security officers with a logged reason.

### Go Code: Privileged Session Recorder

```go
package pam

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// SessionRecorder captures privileged session activity.
type SessionRecorder struct {
	mu       sync.Mutex
	sessions map[uuid.UUID]*PrivilegedSession
	store    SessionStore
}

// PrivilegedSession tracks a single elevated session.
type PrivilegedSession struct {
	ID           uuid.UUID         `json:"id"`
	UserID       uuid.UUID         `json:"user_id"`
	TenantID     uuid.UUID         `json:"tenant_id"`
	ElevationID  *uuid.UUID        `json:"elevation_id,omitempty"`
	StartedAt    time.Time         `json:"started_at"`
	EndedAt      *time.Time        `json:"ended_at,omitempty"`
	Commands     []RecordedCommand `json:"commands"`
	IPAddress    string            `json:"ip_address"`
	UserAgent    string            `json:"user_agent"`
}

// RecordedCommand captures a single API call during a privileged session.
type RecordedCommand struct {
	Timestamp  time.Time              `json:"timestamp"`
	Method     string                 `json:"method"`
	Path       string                 `json:"path"`
	StatusCode int                    `json:"status_code"`
	RequestBody  string               `json:"request_body"`   // Redacted
	ResponseBody string               `json:"response_body"`  // Redacted
	Duration   time.Duration          `json:"duration_ms"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// SessionStore persists session recordings.
type SessionStore interface {
	SaveSession(ctx context.Context, session *PrivilegedSession) error
	GetSession(ctx context.Context, id uuid.UUID) (*PrivilegedSession, error)
	DeleteOlderThan(ctx context.Context, before time.Time) (int, error)
}

// RecordingMiddleware wraps the HTTP handler chain for privileged users.
// It captures every request/response for users with an active elevation.
func (sr *SessionRecorder) RecordingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := userIDFromContext(r.Context())

		// Only record if user has an active privileged session
		session := sr.getActiveSession(userID)
		if session == nil {
			next.ServeHTTP(w, r)
			return
		}

		// Capture request body
		var reqBody []byte
		if r.Body != nil {
			reqBody, _ = io.ReadAll(r.Body)
			r.Body = io.NopCloser(bytes.NewReader(reqBody))
		}

		// Wrap response writer to capture response body
		recorder := &responseRecorder{ResponseWriter: w, body: &bytes.Buffer{}}

		start := time.Now()
		next.ServeHTTP(recorder, r)
		duration := time.Since(start)

		// Record the command
		cmd := RecordedCommand{
			Timestamp:    start.UTC(),
			Method:       r.Method,
			Path:         r.URL.Path,
			StatusCode:   recorder.statusCode,
			RequestBody:  redactSensitive(r.URL.Path, string(reqBody)),
			ResponseBody: redactSensitive(r.URL.Path, recorder.body.String()),
			Duration:     duration,
		}

		sr.mu.Lock()
		session.Commands = append(session.Commands, cmd)
		sr.mu.Unlock()

		// Async persist (batch flush every N commands)
		go sr.flushIfNeeded(context.Background(), session)
	})
}

// responseRecorder wraps http.ResponseWriter to capture status code and body.
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

func (rr *responseRecorder) WriteHeader(code int) {
	rr.statusCode = code
	rr.ResponseWriter.WriteHeader(code)
}

func (rr *responseRecorder) Write(b []byte) (int, error) {
	rr.body.Write(b)
	return rr.ResponseWriter.Write(b)
}

// redactSensitive removes passwords, tokens, and PII from recorded payloads.
func redactSensitive(path string, body string) string {
	// Redact patterns: "password":"...", "token":"...", "secret":"..."
	// In production, use structured field-aware redaction
	type sensitiveField struct {
		key     string
		pattern string
	}
	fields := []sensitiveField{
		{"password", `"password"\s*:\s*"[^"]*"`},
		{"token", `"token"\s*:\s*"[^"]*"`},
		{"secret", `"secret"\s*:\s*"[^"]*"`},
		{"api_key", `"api_key"\s*:\s*"[^"]*"`},
	}
	result := body
	for _, f := range fields {
		result = redactPattern(result, f.pattern, fmt.Sprintf(`"%s":"[REDACTED]"`, f.key))
	}
	// Truncate very long bodies
	if len(result) > 10000 {
		result = result[:10000] + "...[TRUNCATED]"
	}
	return result
}

func redactPattern(input, pattern, replacement string) string {
	// Use regexp in production
	return input
}

// RetentionJanitor periodically deletes sessions older than the retention period.
func (sr *SessionRecorder) RetentionJanitor(ctx context.Context, retention time.Duration) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cutoff := time.Now().UTC().Add(-retention)
			deleted, _ := sr.store.DeleteOlderThan(ctx, cutoff)
			if deleted > 0 {
				_ = deleted // log in production
			}
		}
	}
}

func (sr *SessionRecorder) getActiveSession(userID uuid.UUID) *PrivilegedSession {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	for _, s := range sr.sessions {
		if s.UserID == userID && s.EndedAt == nil {
			return s
		}
	}
	return nil
}

func (sr *SessionRecorder) flushIfNeeded(ctx context.Context, session *PrivilegedSession) {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	if len(session.Commands) >= 50 {
		sr.store.SaveSession(ctx, session)
		session.Commands = session.Commands[:0] // Reset after flush
	}
}
```

---

## 6. Command Audit and Behavioral Analysis

### Full-Context Logging

Every privileged command should record:

- **Who** — user ID, tenant ID, role at time of action, elevation request ID.
- **What** — HTTP method, path, request body (redacted), response status, response body (redacted).
- **When** — precise timestamp (UTC), session duration so far.
- **From where** — source IP, geolocation, user agent, device fingerprint.
- **Why** — justification from the elevation request, approval chain.

### Anomaly Detection Heuristics

| Anomaly | Detection Logic | Severity |
|---------|----------------|----------|
| Unusual tenant access | Admin accesses tenant data they've never touched | Medium |
| Mass data export | >100 records exported in single session | High |
| Off-hours activity | Actions between 22:00-06:00 user's timezone | Medium |
| Rapid privilege changes | >5 role assignments in 10 minutes | High |
| Break-glass account usage | Any break-glass login | Critical |
| New device for admin | Admin logging in from unrecognized device | Medium |
| Geographic impossible | Login from two distant locations within 1 hour | Critical |

### Go Code: Command Audit with Anomaly Scoring

```go
package pam

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// CommandAuditor logs every privileged command and computes anomaly scores.
type CommandAuditor struct {
	mu      sync.RWMutex
	baselines map[uuid.UUID]*UserBaseline // userID → behavioral baseline
	scoring   *AnomalyScorer
}

// UserBaseline tracks normal behavior patterns for a user.
type UserBaseline struct {
	UserID           uuid.UUID
	CommonTenants    map[uuid.UUID]int    // tenant → access frequency
	CommonIPs        map[string]int        // IP → frequency
	UsualHours       [7][24]bool          // day-of-week × hour activity heatmap
	AvgCommandsPerHr float64
	LastUpdated      time.Time
}

// AuditEntry records a single privileged command with full context.
type AuditEntry struct {
	ID            uuid.UUID  `json:"id"`
	UserID        uuid.UUID  `json:"user_id"`
	TenantID      uuid.UUID  `json:"tenant_id"`
	ElevationID   *uuid.UUID `json:"elevation_id,omitempty"`
	Timestamp     time.Time  `json:"timestamp"`
	Method        string     `json:"method"`
	Path          string     `json:"path"`
	StatusCode    int        `json:"status_code"`
	SourceIP      string     `json:"source_ip"`
	GeoLocation   string     `json:"geo_location,omitempty"`
	UserAgent     string     `json:"user_agent"`
	RoleAtAction  string     `json:"role_at_action"`
	Justification string     `json:"justification"`
	AnomalyScore  float64    `json:"anomaly_score"`
	Anomalies     []string   `json:"anomalies,omitempty"`
}

// AnomalyScorer computes a risk score (0.0 to 1.0) for each command.
type AnomalyScorer struct {
	thresholds map[string]float64
}

// RecordCommand logs a privileged command and computes its anomaly score.
func (ca *CommandAuditor) RecordCommand(ctx context.Context, entry *AuditEntry) (*AuditEntry, error) {
	ca.mu.RLock()
	baseline := ca.baselines[entry.UserID]
	ca.mu.RUnlock()

	var anomalies []string
	score := 0.0

	if baseline != nil {
		// Check off-hours
		hour := entry.Timestamp.Hour()
		dayOfWeek := int(entry.Timestamp.Weekday())
		if !baseline.UsualHours[dayOfWeek][hour] {
			anomalies = append(anomalies, "off_hours_activity")
			score += 0.2
		}

		// Check unusual tenant access
		if baseline.CommonTenants != nil {
			if freq, ok := baseline.CommonTenants[entry.TenantID]; !ok || freq == 0 {
				anomalies = append(anomalies, "unusual_tenant_access")
				score += 0.3
			}
		}

		// Check unusual IP
		if baseline.CommonIPs != nil {
			if freq, ok := baseline.CommonIPs[entry.SourceIP]; !ok || freq == 0 {
				anomalies = append(anomalies, "new_ip_address")
				score += 0.15
			}
		}
	}

	// Check mass export pattern
	if isMassExportPath(entry.Method, entry.Path) {
		anomalies = append(anomalies, "mass_data_export")
		score += 0.4
	}

	// Check rapid privilege changes
	if isPrivilegeChangePath(entry.Method, entry.Path) {
		ca.mu.Lock()
		recentChanges := ca.countRecentPrivilegeChanges(entry.UserID, 10*time.Minute)
		ca.mu.Unlock()
		if recentChanges > 5 {
			anomalies = append(anomalies, "rapid_privilege_changes")
			score += 0.35
		}
	}

	// Clamp score
	if score > 1.0 {
		score = 1.0
	}

	entry.AnomalyScore = score
	if len(anomalies) > 0 {
		entry.Anomalies = anomalies
	}

	// Alert on high scores
	if score >= 0.7 {
		go ca.alertHighRiskCommand(context.Background(), entry)
	}

	// Update baseline
	go ca.updateBaseline(context.Background(), entry)

	return entry, nil
}

// updateBaseline adjusts the user's behavioral baseline with the new command.
func (ca *CommandAuditor) updateBaseline(ctx context.Context, entry *AuditEntry) {
	ca.mu.Lock()
	defer ca.mu.Unlock()

	baseline, ok := ca.baselines[entry.UserID]
	if !ok {
		baseline = &UserBaseline{
			UserID:        entry.UserID,
			CommonTenants: make(map[uuid.UUID]int),
			CommonIPs:     make(map[string]int),
			LastUpdated:   time.Now().UTC(),
		}
		ca.baselines[entry.UserID] = baseline
	}

	baseline.CommonTenants[entry.TenantID]++
	baseline.CommonIPs[entry.SourceIP]++
	hour := entry.Timestamp.Hour()
	dayOfWeek := int(entry.Timestamp.Weekday())
	baseline.UsualHours[dayOfWeek][hour] = true
	baseline.LastUpdated = time.Now().UTC()
}

func (ca *CommandAuditor) countRecentPrivilegeChanges(userID uuid.UUID, window time.Duration) int {
	// Query audit store for recent privilege change commands by this user
	return 0
}

func (ca *CommandAuditor) alertHighRiskCommand(ctx context.Context, entry *AuditEntry) {
	// Send real-time alert to security team via NATS/notification service
}

func isMassExportPath(method, path string) bool {
	return method == "GET" && (contains(path, "export") || contains(path, "bulk"))
}

func isPrivilegeChangePath(method, path string) bool {
	if method != "POST" && method != "PUT" && method != "DELETE" {
		return false
	}
	return contains(path, "/roles/") || contains(path, "/permissions/")
}
```

---

## 7. Privilege Escalation Prevention

### Horizontal Escalation

An admin of Tenant A should never be able to access Tenant B's data. Prevention:

- **Tenant-scoped queries** — every database query includes `WHERE tenant_id = $1`.
- **JWT tenant binding** — the JWT's `tenant_id` claim is authoritative; X-Tenant-ID header is secondary.
- **Cross-tenant API calls blocked** — the gateway enforces that the JWT tenant matches the request tenant.

### Vertical Escalation

A regular user should never be able to become an admin without authorization. Prevention:

- **Admin scope check** — `/api/v1/admin/*` routes require `admin` or `ggid:admin` scope.
- **Role assignment audit** — who assigned the admin role, when, and via what approval.
- **Self-promotion prevention** — users cannot assign roles to themselves.

### Role Inheritance Risks

GGID's role hierarchy uses parent-child inheritance where a parent role inherits all permissions
of child roles. Risks:

- **Cycle creation** — prevented by `SetParent` cycle detection in `role_service.go`.
- **Permission inflation** — adding a child role to a high-privilege parent silently grants new
  permissions. Mitigation: alert on effective permission changes.
- **Inheritance depth** — deep hierarchies make permission analysis difficult. Limit depth to 5.

### Go Code: Escalation Prevention Middleware

```go
package pam

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/ggid/ggid/pkg/tenant"
)

// EscalationGuard middleware prevents privilege escalation attacks.
type EscalationGuard struct {
	roleSvc     RoleAssignmentService
	selfAssignBlocklist []string // Role keys that cannot be self-assigned
}

type RoleAssignmentService interface {
	GetUserRoles(ctx context.Context, userID uuid.UUID) ([]string, error) // returns role keys
	AssignRole(ctx context.Context, userID, roleID uuid.UUID, grantedBy uuid.UUID) error
}

// Middleware returns an HTTP middleware that checks for escalation attempts.
func (eg *EscalationGuard) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only check write operations on role/permission endpoints
		if r.Method != "POST" && r.Method != "PUT" && r.Method != "PATCH" && r.Method != "DELETE" {
			next.ServeHTTP(w, r)
			return
		}

		if !isRoleAssignmentPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		userID := userIDFromContext(r.Context())
		jwtTenantID := tenantIDFromContext(r.Context())

		// 1. Prevent horizontal escalation: verify request tenant matches JWT tenant
		var body map[string]any
		bodyBytes, _ := io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		json.Unmarshal(bodyBytes, &body)

		if reqTenantID, ok := body["tenant_id"].(string); ok && reqTenantID != "" {
			if reqTenantID != jwtTenantID.String() {
				writeError(w, http.StatusForbidden, "cross-tenant operation denied")
				return
			}
		}

		// 2. Prevent vertical escalation via self-assignment
		if targetUserID, ok := body["user_id"].(string); ok {
			targetUUID, _ := uuid.Parse(targetUserID)
			if targetUUID == userID {
				if roleKey, ok := body["role_key"].(string); ok {
					if containsStr(eg.selfAssignBlocklist, roleKey) {
						writeError(w, http.StatusForbidden, "cannot self-assign privileged role")
						return
					}
				}
			}
		}

		// 3. Check for path-based tenant mismatch: /api/v1/roles/{id}
		// Extract tenant from the role being modified and compare
		if roleIDStr := extractRoleIDFromPath(r.URL.Path); roleIDStr != "" {
			roleID, err := uuid.Parse(roleIDStr)
			if err == nil {
				role, err := eg.roleSvc.GetRole(r.Context(), roleID)
				if err == nil && role.TenantID != jwtTenantID {
					writeError(w, http.StatusForbidden, "cannot modify role from different tenant")
					return
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

func isRoleAssignmentPath(path string) bool {
	return strings.Contains(path, "/roles/") ||
		strings.Contains(path, "/permissions/") ||
		strings.Contains(path, "/bulk-assign")
}

func extractRoleIDFromPath(path string) string {
	parts := strings.Split(path, "/")
	for i, p := range parts {
		if p == "roles" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

// PermissionChangeAlert watches for effective permission changes in role hierarchies.
// When a parent role gains new permissions due to child changes, alert security.
func (eg *EscalationGuard) PermissionChangeAlert(
	ctx context.Context,
	roleID uuid.UUID,
	oldPerms []string,
	newPerms []string,
) {
	added := diffStrings(newPerms, oldPerms)
	if len(added) > 0 {
		// Alert: "Role X gained N new permissions via inheritance change"
		// This is especially important if the role is a parent of an admin role
	}
}

func diffStrings(a, b []string) []string {
	set := make(map[string]bool)
	for _, s := range b {
		set[s] = true
	}
	var result []string
	for _, s := range a {
		if !set[s] {
			result = append(result, s)
		}
	}
	return result
}

func containsStr(slice []string, target string) bool {
	for _, s := range slice {
		if s == target {
			return true
		}
	}
	return false
}
```

---

## 8. GGID PAM Gap Analysis

### Current State Assessment

Based on review of the GGID codebase (`services/policy/`, `services/gateway/`), the following
PAM capabilities were identified:

#### What Exists

| Capability | Location | Status |
|-----------|----------|--------|
| **RBAC role management** | `services/policy/internal/service/role_service.go` | Full CRUD with role hierarchy |
| **Role inheritance** | `role_service.go` `GetEffectivePermissions()` | DFS traversal with cycle detection |
| **ABAC policy evaluation** | `services/policy/internal/service/evaluator.go` | Allow/deny with default-deny |
| **Decision logging** | `evaluator.go` `DecisionEntry` | In-memory, 1000-entry ring buffer |
| **Admin scope check** | `gateway/internal/router/router.go` `hasAdminScope()` | Checks `admin` or `ggid:admin` scope |
| **JWT tenant binding** | `gateway/internal/middleware/jwt_claims.go` | Extracts `tenant_id` from JWT |
| **UserRole expiry field** | `domain/models.go` `UserRole.ExpiresAt` | Field exists but never populated by any handler |
| **Time-based conditions** | `policy/internal/server/http.go` `handleTimeConditions` | In-memory storage, not wired to evaluator |
| **Policy templates** | `policy/internal/server/http.go` | PCI-DSS, HIPAA, SOC2, GDPR templates |
| **Policy export/import** | `policy/internal/server/http.go` | JSON export/import with tenant scoping |
| **Default action config** | `policy/internal/server/http.go` `handleDefaultAction` | Configurable deny-all/allow-all |

#### What Is Missing

| Capability | Priority | Impact |
|-----------|----------|--------|
| **JIT elevation** | P0 | Admins have standing privileges 24/7 |
| **Approval workflow engine** | P0 | No multi-person approval for sensitive operations |
| **Break-glass accounts** | P0 | No emergency access mechanism when auth is down |
| **Session recording** | P1 | No recording of privileged sessions |
| **Command-level audit for admins** | P1 | Decision log captures evaluations, not admin commands |
| **Anomaly detection** | P1 | No behavioral baseline or alerting for admin activity |
| **Dual-control for super-admin** | P0 | Single admin can perform any operation |
| **Tenant boundary enforcement on role mutation** | P1 | Role CRUD lacks cross-tenant guard (only JWT check) |
| **Self-promotion prevention** | P1 | No check preventing users from self-assigning admin roles |
| **Elevated audit detail for break-glass** | P0 | No special audit treatment for emergency access |
| **IP allowlist for admin operations** | P2 | Admin API accessible from any IP |
| **Role depth limiting** | P2 | No limit on inheritance depth (currently maxDepth=100 in cycle check) |
| **Permission change alerting** | P1 | No alert when role inheritance grants new permissions |
| **Time-based conditions wired to evaluator** | P1 | Time conditions exist but aren't evaluated by `Evaluator.Check()` |

### Code-Level Findings

1. **`UserRole.ExpiresAt` is unused** — the domain model has an `ExpiresAt *time.Time` field
   in `UserRole`, but `RoleService.AssignRole()` passes it through without any auto-expiry
   enforcement. This means even if JIT elevation sets an expiry, nothing revokes the role when
   the time passes.

2. **Admin scope check is binary** — `hasAdminScope()` returns true for anyone with `admin` or
   `ggid:admin` scope. There's no gradation: a read-only admin and a super-admin have the same
   access to `/api/v1/admin/*`.

3. **Policy HTTP server has no auth** — `policy/internal/server/http.go` `RegisterRoutes()`
   registers all endpoints on a plain `http.ServeMux`. There is no middleware-level auth check.
   Authentication is expected to come from the Gateway, but if the policy service is directly
   accessible (as in Docker Compose with port 8070 exposed), all endpoints are unauthenticated.

4. **Time conditions not evaluated** — `handleTimeConditions()` stores rules in an in-memory
   slice, but the `Evaluator.Check()` method never consults them. Time-based access control is
   a declared feature that doesn't actually work at evaluation time.

5. **Bulk assignment uses in-memory map** — `handleBulkAssign()` stores role assignments in
   a `sync.RWMutex`-protected map. This is a prototype, not production-grade. Role assignments
   should go through the identity service's persistence layer.

6. **Decision log is in-memory only** — `decisionLog` is a slice with `maxDecisions = 1000`.
   For compliance (SOC2, GDPR audit trails), decisions must be persisted to durable storage
   with tamper-evidence.

---

## 9. Gap Analysis & Recommendations

### Priority Action Items

| # | Action Item | Priority | Effort | Description |
|---|------------|----------|--------|-------------|
| 1 | **Implement JIT elevation service** | P0 | 3-5 days | New `services/pam` (or extend `policy`) with elevation request/approve/expire lifecycle. Wire `UserRole.ExpiresAt` to a background reaper that auto-revokes expired roles. Add NATS subjects `pam.elevation.*`. |
| 2 | **Add approval workflow engine** | P0 | 3-5 days | Multi-level approver chain with SLA timers. Integrate with NATS JetStream for async notifications. Dual-control for super-admin operations. New `ApprovalConfig` per tenant. |
| 3 | **Create break-glass account system** | P0 | 2-3 days | Break-glass account CRUD with HSM-sealed passwords. Real-time alerting middleware. Quarterly rotation cron. Break-glass-specific audit detail. |
| 4 | **Wire time conditions to evaluator** | P1 | 1-2 days | Modify `Evaluator.Check()` to consult `timeConditions` rules. Move from in-memory to database-backed storage. Add time-zone-aware evaluation. |
| 5 | **Add admin command audit pipeline** | P1 | 2-3 days | Gateway middleware that captures all `/api/v1/admin/*` requests with full context. Feed to audit service via NATS. Add anomaly scoring with behavioral baselines. |

### Summary

GGID has a solid RBAC/ABAC foundation with role hierarchy, policy evaluation, and basic admin
scope checks. However, the PAM layer — which governs how privileged access is requested, approved,
monitored, and revoked — is entirely absent. The most critical gaps are:

1. **No JIT elevation** means all admins have permanent standing privileges.
2. **No approval workflow** means a single compromised admin account can cause unlimited damage.
3. **No break-glass** means an auth outage could lock out all admins permanently.
4. **No session recording** means post-incident forensics for admin actions are limited to
   in-memory decision logs that are lost on restart.

Implementing items 1-3 (P0) would bring GGID's PAM posture from "none" to "basic" and address
the most critical risks. Items 4-5 (P1) would bring it to "production-ready" for compliance
frameworks like SOC2 and ISO 27001.

### Standards Alignment

| Standard | PAM Requirement | GGID Status |
|----------|----------------|-------------|
| SOC2 CC6.3 | Logical access security controls | Partial — RBAC exists, JIT missing |
| ISO 27001 A.9 | Access management | Partial — no privileged session monitoring |
| NIST 800-53 AC-6 | Least privilege | Partial — standing privileges, no JIT |
| NIST 800-53 AU-12 | Audit generation | Partial — decision log in-memory only |
| PCI-DSS 7.2 | Role-based access control | Met — RBAC with hierarchy |
| PCI-DSS 10.2 | Audit trails for privileged access | Not met — no admin command audit |
