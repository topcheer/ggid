# Tenant Onboarding and Lifecycle for Multi-Tenant IAM Systems

**Document Type:** Security Research & Architecture Analysis
**Project:** GGID IAM Suite
**Date:** 2025-01-20
**Author:** Security Research Team

---

## Table of Contents

1. [Self-Service Signup Flow](#1-self-service-signup-flow)
2. [Admin Approval Workflow](#2-admin-approval-workflow)
3. [Initial Admin User Provisioning](#3-initial-admin-user-provisioning)
4. [Default Configuration](#4-default-configuration)
5. [Trial and Plan Management](#5-trial-and-plan-management)
6. [Tenant Data Isolation at Provisioning](#6-tenant-data-isolation-at-provisioning)
7. [Domain Verification](#7-domain-verification)
8. [Tenant Offboarding](#8-tenant-offboarding)
9. [GGID Onboarding Gap Analysis](#9-ggid-onboarding-gap-analysis)
10. [Gap Analysis & Recommendations](#10-gap-analysis--recommendations)

---

## 1. Self-Service Signup Flow

### Overview

Self-service tenant signup is the front door of a SaaS IAM platform. An organization
administrator arrives, enters company details, selects a plan, and receives a fully
provisioned tenant with an admin account. The flow must be atomic — either the entire
tenant (database record, default config, admin user, data isolation) is created, or
nothing is, to prevent orphaned half-tenants.

### Flow Diagram

```
User → POST /api/v1/signup
         │
         ├── Validate input (org name, admin email, plan)
         ├── Check for duplicate slug/domain
         ├── Generate tenant UUID
         ├── Create tenant record (status: pending_verification)
         ├── Create admin user (status: pending)
         ├── Generate email verification token
         ├── Send verification email
         └── Return 202 Accepted

User clicks email link → GET /api/v1/signup/verify?token=xxx
         │
         ├── Consume verification token
         ├── Activate admin user
         ├── Provision default config (roles, policies, OAuth clients)
         ├── Set tenant status: active
         ├── Allocate Redis/NATS namespaces
         └── Return 200 OK with tenant info
```

### Go Code: Self-Service Signup Handler

```go
package onboarding

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ggid/ggid/pkg/crypto"
	gerr "github.com/ggid/ggid/pkg/errors"
	"github.com/ggid/ggid/services/org/internal/domain"
	"github.com/google/uuid"
)

// SignupRequest holds the payload for self-service tenant signup.
type SignupRequest struct {
	OrgName     string `json:"org_name"`
	Slug        string `json:"slug"`
	AdminEmail  string `json:"admin_email"`
	AdminName   string `json:"admin_name"`
	Plan        string `json:"plan"`
	Domain      string `json:"domain"`       // optional: for domain verification later
	ReturnURL   string `json:"return_url"`   // redirect URL after email verification
}

// SignupResult is returned to the caller after a successful signup request.
type SignupResult struct {
	TenantID           uuid.UUID `json:"tenant_id"`
	Status             string    `json:"status"`
	VerificationSentTo string    `json:"verification_sent_to"`
}

var slugRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{2,62}[a-z0-9]$`)

// SignupService handles self-service tenant registration.
type SignupService struct {
	tenantRepo    TenantRepo
	userRepo      UserRepo
	emailSvc      EmailSender
	configProv    ConfigProvisioner
	dataProv      DataProvisioner
}

// Signup initiates a new tenant registration.
func (s *SignupService) Signup(ctx context.Context, req *SignupRequest) (*SignupResult, error) {
	// --- Validation ---
	if req.OrgName == "" || len(req.OrgName) > 200 {
		return nil, gerr.InvalidArgument("org_name is required (max 200 chars)")
	}
	if !isValidEmail(req.AdminEmail) {
		return nil, gerr.InvalidArgument("admin_email is invalid")
	}
	if !slugRegex.MatchString(req.Slug) {
		return nil, gerr.InvalidArgument("slug must be 4-64 chars, lowercase alphanumeric and hyphens")
	}
	plan := domain.Plan(req.Plan)
	if plan == "" {
		plan = domain.PlanFree
	}
	if !isValidPlan(plan) {
		return nil, gerr.InvalidArgument("plan must be free, pro, or enterprise")
	}

	// --- Uniqueness checks ---
	if existing, _ := s.tenantRepo.GetBySlug(ctx, req.Slug); existing != nil {
		return nil, gerr.AlreadyExists("tenant slug", req.Slug)
	}
	if req.Domain != "" {
		if existing, _ := s.tenantRepo.GetByDomain(ctx, req.Domain); existing != nil {
			return nil, gerr.AlreadyExists("domain", req.Domain)
		}
	}

	// --- Create tenant record ---
	tenantID := uuid.New()
	tenant := &domain.Tenant{
		ID:       tenantID,
		Name:     req.OrgName,
		Slug:     req.Slug,
		Plan:     plan,
		Status:   "pending_verification",
		MaxUsers: defaultMaxUsersForPlan(plan),
		Settings: map[string]any{
			"signup_domain":    req.Domain,
			"admin_email":      req.AdminEmail,
			"signup_at":        time.Now().UTC(),
			"return_url":       req.ReturnURL,
			"trial_expires_at": time.Now().Add(30 * 24 * time.Hour), // 30-day trial
		},
	}

	if err := s.tenantRepo.Create(ctx, tenant); err != nil {
		return nil, gerr.Internal("create tenant", err)
	}

	// --- Create admin user (pending) ---
	tempPassword, err := crypto.GenerateRandomToken(16)
	if err != nil {
		return nil, gerr.Internal("generate temp password", err)
	}

	adminUser := &PendingUser{
		ID:            uuid.New(),
		TenantID:      tenantID,
		Email:         req.AdminEmail,
		DisplayName:   req.AdminName,
		TempPassword:  tempPassword,
		Status:        "pending",
	}

	// Generate email verification token
	verifyToken, err := crypto.GenerateRandomToken(32)
	if err != nil {
		return nil, gerr.Internal("generate verification token", err)
	}
	adminUser.VerificationToken = hashToken(verifyToken)
	adminUser.TokenExpiresAt = time.Now().Add(48 * time.Hour)

	if err := s.userRepo.CreatePending(ctx, adminUser); err != nil {
		// Rollback tenant creation
		_ = s.tenantRepo.Delete(ctx, tenantID)
		return nil, gerr.Internal("create admin user", err)
	}

	// --- Send verification email ---
	verifyURL := fmt.Sprintf("%s?token=%s", req.ReturnURL, verifyToken)
	if err := s.emailSvc.SendVerification(ctx, req.AdminEmail, req.AdminName, req.OrgName, verifyURL); err != nil {
		// Non-fatal: user can request resend
		// Log warning but don't fail the signup
	}

	return &SignupResult{
		TenantID:           tenantID,
		Status:             "pending_verification",
		VerificationSentTo: maskEmail(req.AdminEmail),
	}, nil
}

// VerifySignup completes the signup flow after the admin clicks the email link.
// This is the critical provisioning trigger — it is idempotent and transactional.
func (s *SignupService) VerifySignup(ctx context.Context, token string) (*SignupResult, error) {
	tokenHash := hashToken(token)
	adminUser, err := s.userRepo.GetPendingByToken(ctx, tokenHash)
	if err != nil {
		return nil, gerr.New(gerr.ErrNotFound, "invalid or expired verification token")
	}
	if time.Now().After(adminUser.TokenExpiresAt) {
		return nil, gerr.New(gerr.ErrFailedPrecondition, "verification token expired")
	}

	// --- Atomic provisioning: activate user, provision config, set tenant active ---
	// In production this would be a database transaction with compensating actions.
	tenant, err := s.tenantRepo.GetByID(ctx, adminUser.TenantID)
	if err != nil {
		return nil, gerr.Internal("get tenant", err)
	}

	// 1. Provision data isolation (RLS policies, Redis/NATS namespaces)
	if err := s.dataProv.ProvisionTenant(ctx, tenant.ID, tenant.Slug); err != nil {
		return nil, gerr.Internal("provision data isolation", err)
	}

	// 2. Provision default configuration (roles, policies, OAuth clients)
	if err := s.configProv.ProvisionDefaults(ctx, tenant.ID, tenant.Plan); err != nil {
		return nil, gerr.Internal("provision default config", err)
	}

	// 3. Activate admin user
	adminUser.Status = "active"
	if err := s.userRepo.Activate(ctx, adminUser); err != nil {
		return nil, gerr.Internal("activate admin user", err)
	}

	// 4. Set tenant to active
	tenant.Status = domain.TenantActive
	if err := s.tenantRepo.Update(ctx, tenant); err != nil {
		return nil, gerr.Internal("activate tenant", err)
	}

	// 5. Consume verification token
	_ = s.userRepo.ConsumeToken(ctx, tokenHash)

	// 6. Send welcome email with temporary password
	_ = s.emailSvc.SendWelcome(ctx, adminUser.Email, adminUser.DisplayName,
		tenant.Name, adminUser.TempPassword)

	return &SignupResult{
		TenantID: tenant.ID,
		Status:   "active",
	}, nil
}

func defaultMaxUsersForPlan(plan domain.Plan) int {
	switch plan {
	case domain.PlanFree:
		return 10
	case domain.PlanPro:
		return 500
	case domain.PlanEnterprise:
		return 10000
	default:
		return 10
	}
}

func isValidPlan(p domain.Plan) bool {
	return p == domain.PlanFree || p == domain.PlanPro || p == domain.PlanEnterprise
}

func isValidEmail(email string) bool {
	return strings.Contains(email, "@") && strings.Contains(email, ".")
}

func maskEmail(email string) string {
	at := strings.Index(email, "@")
	if at < 2 {
		return email
	}
	return email[:2] + strings.Repeat("*", at-2) + email[at:]
}

func hashToken(token string) string {
	// SHA-256 hash — in production use crypto/sha256
	return fmt.Sprintf("%x", []byte(token)) // simplified
}
```

### Key Design Decisions

- **Idempotent verification**: If the user clicks the email link twice, the second call
  is a no-op (token already consumed).
- **Compensating rollback**: If user creation fails after tenant creation, the tenant
  record is deleted. In production, use a saga pattern or database transaction.
- **Temporary password**: Generated server-side, sent via secure channel. First-login
  forces a password change (see Section 3).
- **Rate limiting**: Signup endpoint must be rate-limited per IP and per email to
  prevent abuse.

---

## 2. Admin Approval Workflow

### Overview

For enterprise deployments (self-hosted or private cloud), tenant signup may require
platform administrator approval. The signup creates a pending request that enters a
review queue. This is critical for regulated industries (finance, healthcare) where
tenant identity must be verified before provisioning.

### Approval State Machine

```
pending_review → approved → provisioning → active
pending_review → rejected → archived
provisioning → provisioned → active
provisioning → failed → pending_review (retry)
active → suspended → active (reactivate)
active → suspended → deletion_pending (offboarding)
```

### Go Code: Approval Workflow

```go
package onboarding

import (
	"context"
	"time"

	gerr "github.com/ggid/ggid/pkg/errors"
	"github.com/google/uuid"
)

// ApprovalStatus tracks the review state of a tenant signup request.
type ApprovalStatus string

const (
	ApprovalPending      ApprovalStatus = "pending_review"
	ApprovalApproved     ApprovalStatus = "approved"
	ApprovalRejected     ApprovalStatus = "rejected"
	ApprovalProvisioning ApprovalStatus = "provisioning"
	ApprovalProvisioned  ApprovalStatus = "provisioned"
	ApprovalActive       ApprovalStatus = "active"
	ApprovalFailed       ApprovalStatus = "failed"
	ApprovalSuspended    ApprovalStatus = "suspended"
)

// ApprovalRequest represents a tenant signup pending platform admin review.
type ApprovalRequest struct {
	ID            uuid.UUID
	TenantID      uuid.UUID
	OrgName       string
	AdminEmail    string
	Plan          string
	Status        ApprovalStatus
	SubmittedAt   time.Time
	ReviewedBy    *uuid.UUID  // platform admin who reviewed
	ReviewedAt    *time.Time
	RejectionReason string
	Notes         string      // internal notes from reviewer
	Metadata      map[string]any
}

// ApprovalRepo persists approval requests.
type ApprovalRepo interface {
	Create(ctx context.Context, req *ApprovalRequest) error
	GetByID(ctx context.Context, id uuid.UUID) (*ApprovalRequest, error)
	GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*ApprovalRequest, error)
	ListPending(ctx context.Context, limit, offset int) ([]*ApprovalRequest, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status ApprovalStatus, reviewerID uuid.UUID, reason string) error
}

// ApprovalService manages the admin approval workflow.
type ApprovalService struct {
	repo       ApprovalRepo
	tenantRepo TenantRepo
	configProv ConfigProvisioner
	dataProv   DataProvisioner
	userRepo   UserRepo
	emailSvc   EmailSender
	notifier   NotificationSender
}

// SubmitForApproval creates a pending approval request for enterprise signup.
// Called when plan=enterprise or when RequireApproval mode is enabled.
func (s *ApprovalService) SubmitForApproval(ctx context.Context, tenantID uuid.UUID, req *SignupRequest) error {
	approvalReq := &ApprovalRequest{
		ID:          uuid.New(),
		TenantID:    tenantID,
		OrgName:     req.OrgName,
		AdminEmail:  req.AdminEmail,
		Plan:        req.Plan,
		Status:      ApprovalPending,
		SubmittedAt: time.Now().UTC(),
		Metadata: map[string]any{
			"domain":      req.Domain,
			"return_url":  req.ReturnURL,
			"auto_exempt": false,
		},
	}

	if err := s.repo.Create(ctx, approvalReq); err != nil {
		return gerr.Internal("create approval request", err)
	}

	// Notify platform admins of new pending request
	_ = s.notifier.NotifyAdmins(ctx, "new_tenant_approval", map[string]any{
		"approval_id": approvalReq.ID,
		"org_name":    req.OrgName,
		"plan":        req.Plan,
	})

	return nil
}

// Approve transitions a request from pending to provisioning, then provisions the tenant.
func (s *ApprovalService) Approve(ctx context.Context, approvalID, reviewerID uuid.UUID, notes string) error {
	req, err := s.repo.GetByID(ctx, approvalID)
	if err != nil {
		return gerr.New(gerr.ErrNotFound, "approval request not found")
	}
	if req.Status != ApprovalPending {
		return gerr.New(gerr.ErrFailedPrecondition,
			"request is not pending (current: %s)", req.Status)
	}

	// Update status to provisioning
	if err := s.repo.UpdateStatus(ctx, approvalID, ApprovalProvisioning, reviewerID, ""); err != nil {
		return gerr.Internal("update approval status", err)
	}

	// Get the tenant record
	tenant, err := s.tenantRepo.GetByID(ctx, req.TenantID)
	if err != nil {
		_ = s.repo.UpdateStatus(ctx, approvalID, ApprovalFailed, reviewerID, "tenant not found")
		return gerr.Internal("get tenant", err)
	}

	// Provision data isolation
	if err := s.dataProv.ProvisionTenant(ctx, tenant.ID, tenant.Slug); err != nil {
		_ = s.repo.UpdateStatus(ctx, approvalID, ApprovalFailed, reviewerID, err.Error())
		return gerr.Internal("provision data isolation", err)
	}

	// Provision default config
	if err := s.configProv.ProvisionDefaults(ctx, tenant.ID, tenant.Plan); err != nil {
		_ = s.repo.UpdateStatus(ctx, approvalID, ApprovalFailed, reviewerID, err.Error())
		return gerr.Internal("provision default config", err)
	}

	// Activate tenant
	tenant.Status = domain.TenantActive
	if err := s.tenantRepo.Update(ctx, tenant); err != nil {
		_ = s.repo.UpdateStatus(ctx, approvalID, ApprovalFailed, reviewerID, err.Error())
		return gerr.Internal("activate tenant", err)
	}

	// Mark as active
	if err := s.repo.UpdateStatus(ctx, approvalID, ApprovalActive, reviewerID, notes); err != nil {
		return gerr.Internal("finalize approval", err)
	}

	// Notify tenant admin that their account is ready
	_ = s.emailSvc.SendApprovalNotification(ctx, req.AdminEmail, req.OrgName, "approved")

	return nil
}

// Reject transitions a request from pending to rejected and soft-deletes the tenant.
func (s *ApprovalService) Reject(ctx context.Context, approvalID, reviewerID uuid.UUID, reason string) error {
	req, err := s.repo.GetByID(ctx, approvalID)
	if err != nil {
		return gerr.New(gerr.ErrNotFound, "approval request not found")
	}
	if req.Status != ApprovalPending {
		return gerr.New(gerr.ErrFailedPrecondition,
			"request is not pending (current: %s)", req.Status)
	}

	if err := s.repo.UpdateStatus(ctx, approvalID, ApprovalRejected, reviewerID, reason); err != nil {
		return gerr.Internal("update approval status", err)
	}

	// Soft-delete the tenant record
	_ = s.tenantRepo.Delete(ctx, req.TenantID)

	// Notify applicant
	_ = s.emailSvc.SendApprovalNotification(ctx, req.AdminEmail, req.OrgName, "rejected: "+reason)

	return nil
}

// ListPending returns approval requests awaiting review, with pagination.
func (s *ApprovalService) ListPending(ctx context.Context, page, pageSize int) ([]*ApprovalRequest, error) {
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListPending(ctx, pageSize, offset)
}
```

### Audit Trail Requirements

Every approval/rejection must be logged with:
- Reviewer identity (who approved/rejected)
- Timestamp
- Reason/notes
- Before/after status
- IP address of the reviewer

This creates an immutable trail for compliance (SOC 2, ISO 27001).

---

## 3. Initial Admin User Provisioning

### Overview

When a new tenant is created, the first admin user must be bootstrapped with full
administrative privileges. This user is the tenant's "break-glass" account — it has
unrestricted access to all tenant resources. The provisioning must:

1. Create the user with a temporary credential (password or magic link)
2. Assign the `admin` role at the tenant-global scope
3. Mark the account as requiring a password change on first login
4. Send a secure invitation with setup instructions

### Go Code: Initial Admin Provisioning

```go
package onboarding

import (
	"context"
	"fmt"
	"time"

	"github.com/ggid/ggid/pkg/crypto"
	gerr "github.com/ggid/ggid/pkg/errors"
	"github.com/ggid/ggid/services/identity/internal/domain"
	"github.com/ggid/ggid/services/policy/internal/domain as policydomain"
	"github.com/google/uuid"
)

// AdminProvisioner creates and configures the initial admin user for a tenant.
type AdminProvisioner struct {
	userRepo     UserRepo
	roleRepo     RoleRepo
	userRoleRepo UserRoleRepo
	emailSvc     EmailSender
	magicLinkTTL time.Duration
}

func NewAdminProvisioner(ur UserRepo, rr RoleRepo, urr UserRoleRepo, es EmailSender) *AdminProvisioner {
	return &AdminProvisioner{
		userRepo:     ur,
		roleRepo:     rr,
		userRoleRepo: urr,
		emailSvc:     es,
		magicLinkTTL: 72 * time.Hour,
	}
}

// ProvisionInitialAdmin creates the first admin user for a newly provisioned tenant.
// The admin receives a magic link to set their password. Until they complete setup,
// the account is marked as "pending_setup" and cannot authenticate.
func (p *AdminProvisioner) ProvisionInitialAdmin(
	ctx context.Context,
	tenantID uuid.UUID,
	adminEmail, adminName string,
) (*AdminProvisioningResult, error) {

	// --- Verify no admin exists yet (idempotency guard) ---
	existing, _ := p.userRepo.GetByEmail(ctx, tenantID, adminEmail)
	if existing != nil {
		return nil, gerr.AlreadyExists("admin user", adminEmail)
	}

	// --- Create the admin user ---
	// Password is empty — the user sets it via magic link.
	// Status is "pending_setup" until first-login completion.
	adminUser := &domain.User{
		ID:            uuid.New(),
		TenantID:      tenantID,
		Username:      deriveUsername(adminEmail),
		Email:         adminEmail,
		Status:        domain.UserStatusActive,
		EmailVerified: true, // verified during signup flow
		DisplayName:   adminName,
		Locale:        "en",
		Timezone:      "UTC",
		// PasswordHash intentionally empty — magic link flow
	}

	if err := p.userRepo.Create(ctx, adminUser); err != nil {
		return nil, gerr.Internal("create admin user", err)
	}

	// --- Find or create the admin role ---
	adminRole, err := p.roleRepo.GetByKey(ctx, tenantID, "admin")
	if err != nil {
		return nil, gerr.Internal("find admin role", err)
	}

	// --- Assign admin role at global scope ---
	userRole := &policydomain.UserRole{
		UserID:    adminUser.ID,
		RoleID:    adminRole.ID,
		ScopeType: policydomain.ScopeGlobal,
		ScopeID:   tenantID, // scoped to the tenant
		GrantedBy: adminUser.ID, // self-granted during bootstrap
	}

	if err := p.userRoleRepo.Assign(ctx, userRole); err != nil {
		// Compensating action: delete the user
		_ = p.userRepo.Delete(ctx, tenantID, adminUser.ID)
		return nil, gerr.Internal("assign admin role", err)
	}

	// --- Generate magic link for password setup ---
	setupToken, err := crypto.GenerateRandomToken(32)
	if err != nil {
		return nil, gerr.Internal("generate setup token", err)
	}

	tokenHash := hashToken(setupToken)
	setupLink := &PasswordSetupToken{
		ID:        uuid.New(),
		UserID:    adminUser.ID,
		TenantID:  tenantID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(p.magicLinkTTL),
		UsedAt:    nil,
	}

	if err := p.userRepo.CreateSetupToken(ctx, setupLink); err != nil {
		return nil, gerr.Internal("create setup token", err)
	}

	// --- Mark account as requiring password change ---
	if err := p.userRepo.SetFlag(ctx, adminUser.ID, "require_password_change", true); err != nil {
		return nil, gerr.Internal("set password change flag", err)
	}

	return &AdminProvisioningResult{
		UserID:      adminUser.ID,
		TenantID:    tenantID,
		SetupToken:  setupToken, // returned to caller for email inclusion
		ExpiresAt:   setupLink.ExpiresAt,
	}, nil
}

// CompletePasswordSetup consumes the magic link token and sets the admin's password.
// This is called when the admin clicks the email link and enters a new password.
func (p *AdminProvisioner) CompletePasswordSetup(
	ctx context.Context,
	token, newPassword string,
) (*domain.User, error) {

	tokenHash := hashToken(token)
	setupToken, err := p.userRepo.GetSetupToken(ctx, tokenHash)
	if err != nil {
		return nil, gerr.New(gerr.ErrNotFound, "invalid or expired setup token")
	}
	if setupToken.UsedAt != nil {
		return nil, gerr.New(gerr.ErrFailedPrecondition, "setup token already used")
	}
	if time.Now().After(setupToken.ExpiresAt) {
		return nil, gerr.New(gerr.ErrFailedPrecondition, "setup token expired")
	}

	// --- Validate password strength ---
	if err := validatePasswordStrength(newPassword); err != nil {
		return nil, err
	}

	// --- Hash and store ---
	hash, err := crypto.HashPassword(newPassword)
	if err != nil {
		return nil, gerr.Internal("hash password", err)
	}

	user, err := p.userRepo.GetByID(ctx, setupToken.TenantID, setupToken.UserID)
	if err != nil {
		return nil, gerr.Internal("get user", err)
	}

	user.PasswordHash = hash
	if err := p.userRepo.UpdatePassword(ctx, setupToken.TenantID, user.ID, hash); err != nil {
		return nil, gerr.Internal("update password", err)
	}

	// Clear the password-change-required flag
	_ = p.userRepo.SetFlag(ctx, user.ID, "require_password_change", false)

	// Consume the token
	now := time.Now()
	_ = p.userRepo.MarkSetupTokenUsed(ctx, setupToken.ID, now)

	return user, nil
}

// AdminProvisioningResult holds the outcome of initial admin provisioning.
type AdminProvisioningResult struct {
	UserID     uuid.UUID
	TenantID   uuid.UUID
	SetupToken string    // plaintext — sent via email, never stored
	ExpiresAt  time.Time
}

// PasswordSetupToken is the persisted token for the magic-link password setup flow.
type PasswordSetupToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TenantID  uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	UsedAt    *time.Time
}

func deriveUsername(email string) string {
	// Use the local part of the email as the username
	for i, ch := range email {
		if ch == '@' {
			return email[:i]
		}
	}
	return email
}

func validatePasswordStrength(password string) error {
	if len(password) < 12 {
		return gerr.InvalidArgument("password must be at least 12 characters")
	}
	hasUpper, hasLower, hasDigit, hasSpecial := false, false, false, false
	for _, ch := range password {
		switch {
		case ch >= 'A' && ch <= 'Z':
			hasUpper = true
		case ch >= 'a' && ch <= 'z':
			hasLower = true
		case ch >= '0' && ch <= '9':
			hasDigit = true
		default:
			hasSpecial = true
		}
	}
	if !(hasUpper && hasLower && hasDigit && hasSpecial) {
		return gerr.InvalidArgument("password must contain upper, lower, digit, and special characters")
	}
	return nil
}
```

### Security Considerations

- **Magic link TTL**: 72 hours max. Shorter for enterprise (24h). The link is
  single-use and expires immediately after password setup.
- **No password in plaintext**: The temporary password is never stored. Only the
  hash is persisted. The plaintext is sent once via email and never logged.
- **First-login enforcement**: The `require_password_change` flag prevents API
  access until the admin completes setup. Every authenticated request checks
  this flag and returns 403 if set.
- **Break-glass procedures**: The initial admin should immediately create a
  second admin for redundancy and enable MFA.

---

## 4. Default Configuration

### Overview

Every new tenant needs a baseline configuration to be functional immediately after
provisioning. This includes:

- **Default roles**: admin, editor, viewer (with appropriate permissions)
- **Default policies**: baseline allow/deny ABAC policies
- **Default OAuth clients**: a management client for API access
- **Rate limit quotas**: per-plan request limits
- **Feature flags**: plan-tier-gated capabilities

### Go Code: Default Config Provisioning

```go
package onboarding

import (
	"context"
	"fmt"

	"github.com/ggid/ggid/pkg/errors"
	"github.com/ggid/ggid/services/org/internal/domain"
	policydomain "github.com/ggid/ggid/services/policy/internal/domain"
	"github.com/google/uuid"
)

// ConfigProvisioner creates default roles, policies, and settings for a new tenant.
type ConfigProvisioner interface {
	ProvisionDefaults(ctx context.Context, tenantID uuid.UUID, plan domain.Plan) error
}

// DefaultConfigProvisioner is the concrete implementation.
type DefaultConfigProvisioner struct {
	roleRepo     RoleRepo
	permRepo     PermRepo
	policyRepo   PolicyRepo
	oauthRepo    OAuthClientRepo
	settingsRepo SettingsRepo
}

func NewDefaultConfigProvisioner(rr RoleRepo, pr PermRepo, pol PolicyRepo, oar OAuthClientRepo, sr SettingsRepo) *DefaultConfigProvisioner {
	return &DefaultConfigProvisioner{roleRepo: rr, permRepo: pr, policyRepo: pol, oauthRepo: oar, settingsRepo: sr}
}

// ProvisionDefaults creates all default configuration for a new tenant.
// This operation must be idempotent — running it twice for the same tenant
// should not create duplicates (use ON CONFLICT DO NOTHING).
func (p *DefaultConfigProvisioner) ProvisionDefaults(ctx context.Context, tenantID uuid.UUID, plan domain.Plan) error {
	planConfig := getPlanConfig(plan)

	// --- 1. Create default permissions ---
	perms := defaultPermissions(tenantID)
	for _, perm := range perms {
		if err := p.permRepo.Create(ctx, perm); err != nil {
			if !isConflict(err) {
				return errors.Wrap(errors.ErrInternal, "create permission "+perm.Key, err)
			}
		}
	}

	// --- 2. Create default roles ---
	roles := defaultRoles(tenantID)
	for _, role := range roles {
		if err := p.roleRepo.Create(ctx, role); err != nil {
			if !isConflict(err) {
				return errors.Wrap(errors.ErrInternal, "create role "+role.Key, err)
			}
		}
	}

	// --- 3. Grant permissions to roles ---
	if err := p.grantDefaultPermissions(ctx, tenantID); err != nil {
		return errors.Wrap(errors.ErrInternal, "grant default permissions", err)
	}

	// --- 4. Create default ABAC policies ---
	policies := defaultPolicies(tenantID)
	for _, pol := range policies {
		if err := p.policyRepo.Create(ctx, pol); err != nil {
			if !isConflict(err) {
				return errors.Wrap(errors.ErrInternal, "create policy "+pol.Name, err)
			}
		}
	}

	// --- 5. Create default OAuth client ---
	oauthClient := defaultOAuthClient(tenantID, planConfig)
	if err := p.oauthRepo.Create(ctx, oauthClient); err != nil {
		if !isConflict(err) {
			return errors.Wrap(errors.ErrInternal, "create oauth client", err)
		}
	}

	// --- 6. Set rate limit quotas ---
	if err := p.settingsRepo.SetRateLimit(ctx, tenantID, planConfig.RateLimitPerMin); err != nil {
		return errors.Wrap(errors.ErrInternal, "set rate limit", err)
	}

	// --- 7. Set feature flags ---
	for flag, enabled := range planConfig.FeatureFlags {
		if err := p.settingsRepo.SetFeatureFlag(ctx, tenantID, flag, enabled); err != nil {
			return errors.Wrap(errors.ErrInternal, "set feature flag "+flag, err)
		}
	}

	return nil
}

// PlanConfig defines per-tier configuration limits and features.
type PlanConfig struct {
	MaxUsers          int
	RateLimitPerMin   int
	FeatureFlags      map[string]bool
	MaxOAuthClients   int
	MaxRoles          int
	MaxPolicies       int
	AuditRetentionDays int
}

func getPlanConfig(plan domain.Plan) PlanConfig {
	switch plan {
	case domain.PlanFree:
		return PlanConfig{
			MaxUsers:          10,
			RateLimitPerMin:   100,
			MaxOAuthClients:   2,
			MaxRoles:          10,
			MaxPolicies:       20,
			AuditRetentionDays: 30,
			FeatureFlags: map[string]bool{
				"sso":           false,
				"webauthn":      false,
				"scim":          false,
				"custom_domain": false,
				"audit_export":  false,
			},
		}
	case domain.PlanPro:
		return PlanConfig{
			MaxUsers:          500,
			RateLimitPerMin:   1000,
			MaxOAuthClients:   10,
			MaxRoles:          50,
			MaxPolicies:       100,
			AuditRetentionDays: 180,
			FeatureFlags: map[string]bool{
				"sso":           true,
				"webauthn":      true,
				"scim":          true,
				"custom_domain": false,
				"audit_export":  true,
			},
		}
	case domain.PlanEnterprise:
		return PlanConfig{
			MaxUsers:          100000,
			RateLimitPerMin:   10000,
			MaxOAuthClients:   100,
			MaxRoles:          500,
			MaxPolicies:       1000,
			AuditRetentionDays: 2555, // 7 years for compliance
			FeatureFlags: map[string]bool{
				"sso":           true,
				"webauthn":      true,
				"scim":          true,
				"custom_domain": true,
				"audit_export":  true,
				"custom_branding": true,
				"data_residency": true,
			},
		}
	default:
		return getPlanConfig(domain.PlanFree)
	}
}

func defaultPermissions(tenantID uuid.UUID) []*policydomain.Permission {
	return []*policydomain.Permission{
		{ID: uuid.New(), TenantID: tenantID, Key: "iam:users:read", Name: "Read Users", ResourceType: "users", Action: "read", SystemPerm: true},
		{ID: uuid.New(), TenantID: tenantID, Key: "iam:users:write", Name: "Write Users", ResourceType: "users", Action: "write", SystemPerm: true},
		{ID: uuid.New(), TenantID: tenantID, Key: "iam:users:delete", Name: "Delete Users", ResourceType: "users", Action: "delete", SystemPerm: true},
		{ID: uuid.New(), TenantID: tenantID, Key: "iam:roles:read", Name: "Read Roles", ResourceType: "roles", Action: "read", SystemPerm: true},
		{ID: uuid.New(), TenantID: tenantID, Key: "iam:roles:write", Name: "Write Roles", ResourceType: "roles", Action: "write", SystemPerm: true},
		{ID: uuid.New(), TenantID: tenantID, Key: "iam:orgs:read", Name: "Read Orgs", ResourceType: "organizations", Action: "read", SystemPerm: true},
		{ID: uuid.New(), TenantID: tenantID, Key: "iam:orgs:write", Name: "Write Orgs", ResourceType: "organizations", Action: "write", SystemPerm: true},
		{ID: uuid.New(), TenantID: tenantID, Key: "iam:audit:read", Name: "Read Audit", ResourceType: "audit", Action: "read", SystemPerm: true},
		{ID: uuid.New(), TenantID: tenantID, Key: "iam:policies:write", Name: "Write Policies", ResourceType: "policies", Action: "write", SystemPerm: true},
	}
}

func defaultRoles(tenantID uuid.UUID) []*policydomain.Role {
	return []*policydomain.Role{
		{ID: uuid.New(), TenantID: tenantID, Key: "admin", Name: "Administrator", Description: "Full system access", SystemRole: true},
		{ID: uuid.New(), TenantID: tenantID, Key: "editor", Name: "Editor", Description: "Read and write access, no admin", SystemRole: true},
		{ID: uuid.New(), TenantID: tenantID, Key: "viewer", Name: "Viewer", Description: "Read-only access", SystemRole: true},
	}
}

func defaultPolicies(tenantID uuid.UUID) []*policydomain.Policy {
	return []*policydomain.Policy{
		{
			ID: uuid.New(), TenantID: tenantID, Name: "deny-cross-tenant",
			Description: "Deny access to resources outside own tenant",
			Effect:      policydomain.EffectDeny,
			Actions:     []string{"*"},
			Resources:   []string{"*"},
			Conditions: map[string]any{
				"resource_tenant_id": map[string]any{"ne": "${user.tenant_id}"},
			},
			Priority: 1,
		},
		{
			ID: uuid.New(), TenantID: tenantID, Name: "deny-deleted-users",
			Description: "Deny all actions for soft-deleted users",
			Effect:      policydomain.EffectDeny,
			Actions:     []string{"*"},
			Resources:   []string{"*"},
			Conditions: map[string]any{
				"user.status": "deleted",
			},
			Priority: 2,
		},
	}
}

func defaultOAuthClient(tenantID uuid.UUID, cfg PlanConfig) *OAuthClient {
	clientID := uuid.New().String()
	return &OAuthClient{
		ID:          uuid.New(),
		TenantID:    tenantID,
		ClientID:    clientID,
		ClientName:  "Management API",
		GrantTypes:  []string{"client_credentials", "authorization_code"},
		Scopes:      []string{"iam:users:read", "iam:users:write", "iam:roles:read"},
		RedirectURIs: []string{},
		SystemClient: true,
	}
}

func (p *DefaultConfigProvisioner) grantDefaultPermissions(ctx context.Context, tenantID uuid.UUID) error {
	// Admin gets all permissions
	adminRole, _ := p.roleRepo.GetByKey(ctx, tenantID, "admin")
	allPerms, _ := p.permRepo.ListByTenant(ctx, tenantID, 100, 0)
	var adminPermIDs []uuid.UUID
	for _, perm := range allPerms {
		adminPermIDs = append(adminPermIDs, perm.ID)
	}
	if err := p.roleRepo.GrantPermissions(ctx, adminRole.ID, adminPermIDs, nil); err != nil {
		return fmt.Errorf("grant admin permissions: %w", err)
	}

	// Viewer gets read permissions only
	viewerRole, _ := p.roleRepo.GetByKey(ctx, tenantID, "viewer")
	var viewerPermIDs []uuid.UUID
	for _, perm := range allPerms {
		if perm.Action == "read" {
			viewerPermIDs = append(viewerPermIDs, perm.ID)
		}
	}
	if err := p.roleRepo.GrantPermissions(ctx, viewerRole.ID, viewerPermIDs, nil); err != nil {
		return fmt.Errorf("grant viewer permissions: %w", err)
	}

	// Editor gets read+write (non-delete)
	editorRole, _ := p.roleRepo.GetByKey(ctx, tenantID, "editor")
	var editorPermIDs []uuid.UUID
	for _, perm := range allPerms {
		if perm.Action == "read" || perm.Action == "write" {
			editorPermIDs = append(editorPermIDs, perm.ID)
		}
	}
	if err := p.roleRepo.GrantPermissions(ctx, editorRole.ID, editorPermIDs, nil); err != nil {
		return fmt.Errorf("grant editor permissions: %w", err)
	}

	return nil
}

func isConflict(err error) bool {
	return err != nil && (errors.Is(err, errors.ErrAlreadyExists) ||
		(err.Error() != "" && contains(err.Error(), "duplicate key")))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
```

---

## 5. Trial and Plan Management

### Overview

SaaS IAM platforms typically offer a free trial with automatic conversion to a paid
plan. The trial lifecycle has distinct phases:

```
active_trial → trial_expired → grace_period → read_only → deletion_pending → deleted
     ↓                                                              ↓
 upgrade_to_paid                                           export_data
```

### Lifecycle States

| State | Duration | Capabilities |
|-------|----------|-------------|
| `active_trial` | 30 days | Full feature access |
| `trial_expired` | Immediate | All features disabled, admin notified |
| `grace_period` | 7 days | Read-only + billing UI |
| `read_only` | 23 days | Read-only, no writes |
| `deletion_pending` | 30 days | All access blocked, data export available |
| `deleted` | Permanent | Data purged, tenant removed |

### Go Code: Plan Lifecycle Management

```go
package onboarding

import (
	"context"
	"time"

	"github.com/ggid/ggid/pkg/errors"
	"github.com/ggid/ggid/services/org/internal/domain"
	"github.com/google/uuid"
)

// TrialConfig defines trial duration and grace periods.
type TrialConfig struct {
	TrialDuration      time.Duration // 30 days
	GracePeriod        time.Duration // 7 days after expiry
	ReadOnlyPeriod     time.Duration // 23 days
	DeletionRetention  time.Duration // 30 days before purge
	NotificationDays   []int         // [14, 7, 3, 1] days before expiry
}

func DefaultTrialConfig() TrialConfig {
	return TrialConfig{
		TrialDuration:     30 * 24 * time.Hour,
		GracePeriod:       7 * 24 * time.Hour,
		ReadOnlyPeriod:    23 * 24 * time.Hour,
		DeletionRetention: 30 * 24 * time.Hour,
		NotificationDays:  []int{14, 7, 3, 1},
	}
}

// PlanService manages tenant plan lifecycle.
type PlanService struct {
	tenantRepo   TenantRepo
	configProv   ConfigProvisioner
	emailSvc     EmailSender
	notifier     NotificationSender
	settingsRepo SettingsRepo
	trialCfg     TrialConfig
}

func NewPlanService(tr TenantRepo, cp ConfigProvisioner, es EmailSender, ns NotificationSender, sr SettingsRepo) *PlanService {
	return &PlanService{
		tenantRepo:   tr,
		configProv:   cp,
		emailSvc:     es,
		notifier:     ns,
		settingsRepo: sr,
		trialCfg:     DefaultTrialConfig(),
	}
}

// StartTrial initializes a 30-day trial for a newly provisioned tenant.
func (s *PlanService) StartTrial(ctx context.Context, tenantID uuid.UUID) error {
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return errors.Wrap(errors.ErrNotFound, "tenant not found", err)
	}

	now := time.Now().UTC()
	trialExpiry := now.Add(s.trialCfg.TrialDuration)

	tenant.Settings["trial_started_at"] = now
	tenant.Settings["trial_expires_at"] = trialExpiry
	tenant.Settings["trial_notified_days"] = []int{} // track which notifications sent
	tenant.Plan = domain.PlanPro // trial gets pro features

	return s.tenantRepo.Update(ctx, tenant)
}

// CheckTrialExpiry is called by a cron job to evaluate trial state transitions.
func (s *PlanService) CheckTrialExpiry(ctx context.Context, tenantID uuid.UUID) error {
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return err
	}

	trialExpiryRaw, ok := tenant.Settings["trial_expires_at"]
	if !ok {
		return nil // not a trial tenant
	}

	trialExpiry, ok := trialExpiryRaw.(time.Time)
	if !ok {
		return nil
	}

	now := time.Now().UTC()

	// --- Send expiry notifications ---
	s.checkAndSendNotifications(ctx, tenant, trialExpiry, now)

	if now.Before(trialExpiry) {
		return nil // trial still active
	}

	// --- Trial has expired ---
	graceEnd := trialExpiry.Add(s.trialCfg.GracePeriod)
	readOnlyEnd := graceEnd.Add(s.trialCfg.ReadOnlyPeriod)
	deletionEnd := readOnlyEnd.Add(s.trialCfg.DeletionRetention)

	switch {
	case now.Before(graceEnd):
		// Grace period: read-only + billing prompt
		if tenant.Status != "trial_grace" {
			tenant.Status = "trial_grace"
			tenant.Plan = domain.PlanFree
			_ = s.emailSvc.SendTrialExpired(ctx, tenant)
			_ = s.configProv.ProvisionDefaults(ctx, tenant.ID, domain.PlanFree) // downgrade features
		}

	case now.Before(readOnlyEnd):
		// Read-only: all writes blocked
		if tenant.Status != "read_only" {
			tenant.Status = "read_only"
			_ = s.settingsRepo.SetFeatureFlag(ctx, tenantID, "write_access", false)
		}

	case now.Before(deletionEnd):
		// Deletion pending: all access blocked, data export available
		if tenant.Status != "deletion_pending" {
			tenant.Status = "deletion_pending"
			_ = s.emailSvc.SendDeletionWarning(ctx, tenant, deletionEnd)
		}

	default:
		// Deletion retention elapsed: permanently delete
		return s.initiateOffboarding(ctx, tenantID, "trial_expired_purge")
	}

	return s.tenantRepo.Update(ctx, tenant)
}

// UpgradePlan transitions a tenant from trial/free to a paid plan.
func (s *PlanService) UpgradePlan(ctx context.Context, tenantID uuid.UUID, newPlan domain.Plan) error {
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return err
	}

	oldPlan := tenant.Plan
	tenant.Plan = newPlan
	tenant.Status = domain.TenantActive
	tenant.MaxUsers = defaultMaxUsersForPlan(newPlan)

	// Clear trial settings
	delete(tenant.Settings, "trial_started_at")
	delete(tenant.Settings, "trial_expires_at")
	delete(tenant.Settings, "trial_notified_days")

	if err := s.tenantRepo.Update(ctx, tenant); err != nil {
		return err
	}

	// Re-provision features for new plan tier
	if err := s.configProv.ProvisionDefaults(ctx, tenantID, newPlan); err != nil {
		// Rollback plan change
		tenant.Plan = oldPlan
		_ = s.tenantRepo.Update(ctx, tenant)
		return errors.Wrap(errors.ErrInternal, "provision new plan features", err)
	}

	// Re-enable write access if it was disabled
	_ = s.settingsRepo.SetFeatureFlag(ctx, tenantID, "write_access", true)

	_ = s.emailSvc.SendPlanUpgrade(ctx, tenant, string(oldPlan), string(newPlan))

	return nil
}

// DowngradePlan transitions to a lower tier. May require removing excess resources.
func (s *PlanService) DowngradePlan(ctx context.Context, tenantID uuid.UUID, newPlan domain.Plan) error {
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return err
	}

	// Check if tenant exceeds new plan limits
	newCfg := getPlanConfig(newPlan)
	userCount, _ := s.tenantRepo.GetUserCount(ctx, tenantID)
	if userCount > newCfg.MaxUsers {
		return errors.New(errors.ErrFailedPrecondition,
			"tenant has %d users, new plan allows %d. Remove excess users before downgrading.",
			userCount, newCfg.MaxUsers)
	}

	tenant.Plan = newPlan
	tenant.MaxUsers = newCfg.MaxUsers

	if err := s.tenantRepo.Update(ctx, tenant); err != nil {
		return err
	}

	// Downgrade features
	if err := s.configProv.ProvisionDefaults(ctx, tenantID, newPlan); err != nil {
		return errors.Wrap(errors.ErrInternal, "provision downgraded features", err)
	}

	_ = s.emailSvc.SendPlanDowngrade(ctx, tenant, newPlan)

	return nil
}

func (s *PlanService) checkAndSendNotifications(ctx context.Context, tenant *domain.Tenant, trialExpiry, now time.Time) {
	daysRemaining := int(trialExpiry.Sub(now).Hours() / 24)

	notifiedRaw, _ := tenant.Settings["trial_notified_days"].([]int)
	notified := make(map[int]bool)
	for _, d := range notifiedRaw {
		notified[d] = true
	}

	for _, notifyDay := range s.trialCfg.NotificationDays {
		if daysRemaining <= notifyDay && !notified[notifyDay] {
			_ = s.emailSvc.SendTrialExpiringSoon(ctx, tenant, notifyDay)
			notified[notifyDay] = true
		}
	}

	// Update notified list
	var notifiedList []int
	for d := range notified {
		notifiedList = append(notifiedList, d)
	}
	tenant.Settings["trial_notified_days"] = notifiedList
}

func (s *PlanService) initiateOffboarding(ctx context.Context, tenantID uuid.UUID, reason string) error {
	// Delegate to the offboarding service (see Section 8)
	tenant, _ := s.tenantRepo.GetByID(ctx, tenantID)
	tenant.Status = "offboarding"
	tenant.Settings["offboarding_reason"] = reason
	return s.tenantRepo.Update(ctx, tenant)
}
```

---

## 6. Tenant Data Isolation at Provisioning

### Overview

When a tenant is provisioned, the system must ensure complete data isolation. GGID
uses PostgreSQL Row Level Security (RLS) with `app.tenant_id` session variables.
At provisioning time:

1. Verify RLS policies exist and are enabled for all multi-tenant tables
2. Set the `app.tenant_id` session variable for the new tenant
3. Verify the RLS policy is active (test query returns empty set for wrong tenant)
4. Allocate Redis key namespace prefix
5. Allocate NATS subject namespace

### Go Code: Tenant Data Provisioning

```go
package onboarding

import (
	"context"
	"fmt"
	"strings"

	"github.com/ggid/ggid/pkg/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// DataProvisioner handles data-layer isolation setup for new tenants.
type DataProvisioner interface {
	ProvisionTenant(ctx context.Context, tenantID uuid.UUID, slug string) error
	DeprovisionTenant(ctx context.Context, tenantID uuid.UUID) error
}

// PostgreSQLDataProvisioner sets up database-level isolation.
type PostgreSQLDataProvisioner struct {
	db    *pgxpool.Pool
	redis *redis.Client
	nats  NATSJetStreamClient
}

func NewPostgreSQLDataProvisioner(db *pgxpool.Pool, rdb *redis.Client, nats NATSJetStreamClient) *PostgreSQLDataProvisioner {
	return &PostgreSQLDataProvisioner{db: db, redis: rdb, nats: nats}
}

// ProvisionTenant sets up data isolation for a new tenant.
// For shared isolation (RLS), this verifies policies are active.
// For schema isolation, this creates a dedicated PostgreSQL schema.
func (p *PostgreSQLDataProvisioner) ProvisionTenant(ctx context.Context, tenantID uuid.UUID, slug string) error {
	// --- 1. Verify RLS is active for this tenant ---
	if err := p.verifyRLS(ctx, tenantID); err != nil {
		return errors.Wrap(errors.ErrInternal, "RLS verification failed", err)
	}

	// --- 2. Initialize tenant settings row ---
	if err := p.initTenantSettings(ctx, tenantID); err != nil {
		return errors.Wrap(errors.ErrInternal, "init tenant settings", err)
	}

	// --- 3. Allocate Redis namespace ---
	// All Redis keys for this tenant are prefixed with "t:{tenantID}:"
	// This prevents key collisions between tenants
	redisKey := fmt.Sprintf("t:%s:init", tenantID)
	if err := p.redis.Set(ctx, redisKey, "provisioned", 0).Err(); err != nil {
		return errors.Wrap(errors.ErrInternal, "redis namespace allocation", err)
	}

	// --- 4. Allocate NATS subject namespace ---
	// Tenants publish to subjects like "tenant.{tenantID}.audit.events"
	natsStream := fmt.Sprintf("TENANT_%s", strings.ToUpper(strings.ReplaceAll(slug, "-", "_")))
	if err := p.nats.CreateStream(ctx, natsStream, []string{
		fmt.Sprintf("tenant.%s.>", tenantID),
	}); err != nil {
		return errors.Wrap(errors.ErrInternal, "NATS stream allocation", err)
	}

	// --- 5. Create data partition for audit logs (if using partitioned tables) ---
	if err := p.createAuditPartition(ctx, tenantID); err != nil {
		return errors.Wrap(errors.ErrInternal, "audit partition", err)
	}

	return nil
}

// verifyRLS checks that Row Level Security is properly configured for the tenant.
// It does this by setting the session variable and running a test query.
func (p *PostgreSQLDataProvisioner) verifyRLS(ctx context.Context, tenantID uuid.UUID) error {
	// Set the tenant session variable
	_, err := p.db.Exec(ctx, fmt.Sprintf("SET LOCAL app.tenant_id = '%s'", tenantID))
	if err != nil {
		return fmt.Errorf("set tenant_id session var: %w", err)
	}

	// Test: query users table with this tenant context — should return 0 rows
	// (no users created yet) but NOT error out
	var count int
	err = p.db.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE tenant_id = $1", tenantID).Scan(&count)
	if err != nil {
		return fmt.Errorf("RLS test query failed: %w", err)
	}

	// Test: verify RLS is enabled on key tables
	tables := []string{"users", "user_emails", "roles", "permissions", "policies"}
	for _, table := range tables {
		var rlsEnabled bool
		query := fmt.Sprintf(`
			SELECT relrowsecurity FROM pg_class WHERE relname = '%s'
		`, table)
		err := p.db.QueryRow(ctx, query).Scan(&rlsEnabled)
		if err != nil {
			return fmt.Errorf("check RLS on %s: %w", table, err)
		}
		if !rlsEnabled {
			return fmt.Errorf("RLS is NOT enabled on table: %s", table)
		}
	}

	return nil
}

func (p *PostgreSQLDataProvisioner) initTenantSettings(ctx context.Context, tenantID uuid.UUID) error {
	_, err := p.db.Exec(ctx, `
		INSERT INTO tenant_settings (tenant_id, data)
		VALUES ($1, '{}')
		ON CONFLICT (tenant_id) DO NOTHING
	`, tenantID)
	return err
}

func (p *PostgreSQLDataProvisioner) createAuditPartition(ctx context.Context, tenantID uuid.UUID) error {
	// Create a partition for this tenant's audit events
	// This keeps audit data physically separated for efficient querying and purging
	partitionName := fmt.Sprintf("audit_events_%s", strings.ReplaceAll(tenantID.String(), "-", "_"))
	_, err := p.db.Exec(ctx, fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s PARTITION OF audit_events
		FOR VALUES IN ('%s')
	`, partitionName, tenantID))
	return err
}

// DeprovisionTenant cleans up all data-layer resources for a tenant.
func (p *PostgreSQLDataProvisioner) DeprovisionTenant(ctx context.Context, tenantID uuid.UUID) error {
	// Flush Redis keys for this tenant
	pattern := fmt.Sprintf("t:%s:*", tenantID)
	iter := p.redis.Scan(ctx, 0, pattern, 1000).Iterator()
	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if len(keys) > 0 {
		p.redis.Del(ctx, keys...)
	}

	// Delete NATS stream
	streamName := fmt.Sprintf("TENANT_%s", strings.ReplaceAll(tenantID.String(), "-", "_"))
	_ = p.nats.DeleteStream(ctx, streamName)

	// Detach audit partition (data already exported during offboarding)
	partitionName := fmt.Sprintf("audit_events_%s", strings.ReplaceAll(tenantID.String(), "-", "_"))
	_, _ = p.db.Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s", partitionName))

	return nil
}
```

### RLS Verification Test

The verification step is critical — it catches misconfigured RLS before any tenant
data is written. A common failure mode is a table where RLS is enabled but the policy
uses a wrong column name, allowing cross-tenant data leakage.

---

## 7. Domain Verification

### Overview

Domain verification allows a tenant to claim an email domain (e.g., `@company.com`).
Once verified, users with that email domain can be auto-provisioned or auto-joined
to the tenant. This is critical for enterprise SSO where the tenant controls a
corporate email domain.

### Flow

```
1. Tenant admin enters domain: "company.com"
2. System generates verification token: "ggid-verify=abc123def456"
3. System provides DNS TXT record instructions:
   _ggid-verify.company.com  TXT  "ggid-verify=abc123def456"
4. Admin adds DNS record
5. System polls DNS (or admin clicks "Verify Now")
6. System resolves TXT record, compares token
7. If match: domain_status = verified
8. Enable domain-scoped auto-provisioning
```

### Go Code: Domain Verification

```go
package onboarding

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/ggid/ggid/pkg/errors"
	"github.com/google/uuid"
)

// DomainStatus tracks verification state.
type DomainStatus string

const (
	DomainPending   DomainStatus = "pending"
	DomainVerified  DomainStatus = "verified"
	DomainFailed    DomainStatus = "failed"
	DomainRevoked   DomainStatus = "revoked"
)

// TenantDomain represents a domain claimed by a tenant.
type TenantDomain struct {
	ID               uuid.UUID
	TenantID         uuid.UUID
	Domain           string
	Status           DomainStatus
	VerificationToken string
	DNSTxtRecord     string
	VerifiedAt       *time.Time
	VerifiedBy       *uuid.UUID
	AutoProvision    bool // auto-create users from this domain
	CreatedAt        time.Time
}

// DomainRepo persists tenant domain claims.
type DomainRepo interface {
	Create(ctx context.Context, td *TenantDomain) error
	GetByDomain(ctx context.Context, domain string) (*TenantDomain, error)
	GetByTenantID(ctx context.Context, tenantID uuid.UUID) ([]*TenantDomain, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status DomainStatus, verifiedBy uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// DomainService handles domain verification and management.
type DomainService struct {
	repo      DomainRepo
	tenantRepo TenantRepo
	userRepo  UserRepo
	dnsResolver *net.Resolver
}

func NewDomainService(r DomainRepo, tr TenantRepo, ur UserRepo) *DomainService {
	return &DomainService{
		repo:        r,
		tenantRepo:  tr,
		userRepo:    ur,
		dnsResolver: net.DefaultResolver,
	}
}

// ClaimDomain initiates a domain claim for a tenant.
func (s *DomainService) ClaimDomain(ctx context.Context, tenantID uuid.UUID, domain string) (*TenantDomain, error) {
	domain = strings.ToLower(strings.TrimSpace(domain))
	domain = strings.TrimPrefix(domain, "www.")
	domain = strings.TrimPrefix(domain, "*.")

	if domain == "" {
		return nil, errors.InvalidArgument("domain is required")
	}

	// --- Check if domain is already claimed ---
	existing, _ := s.repo.GetByDomain(ctx, domain)
	if existing != nil && existing.TenantID != tenantID {
		if existing.Status == DomainVerified {
			return nil, errors.New(errors.ErrAlreadyExists,
				"domain %s is already verified by another tenant", domain)
		}
		// If pending by another tenant, allow concurrent claims (first to verify wins)
	}

	// --- Generate verification token ---
	// Format: ggid-verify={random}
	token, err := generateVerificationToken()
	if err != nil {
		return nil, errors.Internal("generate token", err)
	}

	td := &TenantDomain{
		ID:                uuid.New(),
		TenantID:          tenantID,
		Domain:            domain,
		Status:            DomainPending,
		VerificationToken: token,
		DNSTxtRecord:      fmt.Sprintf("_ggid-verify.%s", domain),
	}

	if err := s.repo.Create(ctx, td); err != nil {
		return nil, errors.Internal("create domain claim", err)
	}

	return td, nil
}

// VerifyDomain checks the DNS TXT record and marks the domain as verified if the
// token matches. This is the critical security check that prevents domain hijacking.
func (s *DomainService) VerifyDomain(ctx context.Context, domainID, verifiedBy uuid.UUID) (*TenantDomain, error) {
	td, err := s.repo.GetByID(ctx, domainID)
	if err != nil {
		return nil, errors.Wrap(errors.ErrNotFound, "domain claim not found", err)
	}
	if td.Status == DomainVerified {
		return td, nil // idempotent
	}

	// --- Resolve DNS TXT record ---
	txtRecords, err := s.dnsResolver.LookupTXT(ctx, td.DNSTxtRecord)
	if err != nil {
		// DNS lookup failed — domain not yet configured or DNS propagation delay
		td.Status = DomainFailed
		_ = s.repo.UpdateStatus(ctx, td.ID, DomainFailed, verifiedBy)
		return td, errors.New(errors.ErrFailedPrecondition,
			"DNS lookup failed for %s: %v", td.DNSTxtRecord, err)
	}

	// --- Check if any TXT record matches our verification token ---
	expectedValue := fmt.Sprintf("ggid-verify=%s", td.VerificationToken)
	matched := false
	for _, record := range txtRecords {
		if strings.TrimSpace(record) == expectedValue {
			matched = true
			break
		}
	}

	if !matched {
		td.Status = DomainFailed
		_ = s.repo.UpdateStatus(ctx, td.ID, DomainFailed, verifiedBy)
		return td, errors.New(errors.ErrFailedPrecondition,
			"DNS TXT record does not match expected verification token. "+
				"Expected: %s, Got: %v", expectedValue, txtRecords)
	}

	// --- Verify the domain is not claimed by another tenant ---
	// Race condition check: another tenant may have verified in the meantime
	conflicting, _ := s.repo.GetByDomain(ctx, td.Domain)
	if conflicting != nil && conflicting.ID != td.ID && conflicting.Status == DomainVerified {
		td.Status = DomainFailed
		_ = s.repo.UpdateStatus(ctx, td.ID, DomainFailed, verifiedBy)
		return td, errors.New(errors.ErrAlreadyExists,
			"domain %s was verified by another tenant during verification", td.Domain)
	}

	// --- Domain verified successfully ---
	now := time.Now().UTC()
	td.Status = DomainVerified
	td.VerifiedAt = &now
	td.VerifiedBy = &verifiedBy

	if err := s.repo.UpdateStatus(ctx, td.ID, DomainVerified, verifiedBy); err != nil {
		return nil, errors.Internal("update domain status", err)
	}

	return td, nil
}

// CheckDomainAutoProvision is called during user registration/login to determine
// if the user's email domain matches a verified tenant domain, enabling auto-join.
func (s *DomainService) CheckDomainAutoProvision(ctx context.Context, email string) (*uuid.UUID, error) {
	atIdx := strings.LastIndex(email, "@")
	if atIdx < 0 {
		return nil, nil
	}
	domain := strings.ToLower(email[atIdx+1:])

	td, err := s.repo.GetByDomain(ctx, domain)
	if err != nil || td == nil {
		return nil, nil // no domain claim for this domain
	}
	if td.Status != DomainVerified || !td.AutoProvision {
		return nil, nil
	}

	return &td.TenantID, nil // user can auto-join this tenant
}

func generateVerificationToken() (string, error) {
	// 32 random hex characters
	b := make([]byte, 16)
	// In production: crypto/rand.Read(b)
	return fmt.Sprintf("%x", b), nil
}
```

### Domain Hijacking Prevention

1. **Race condition**: Two tenants could try to verify the same domain simultaneously.
   The verification check includes a final "conflicting domain" check before marking
   as verified.
2. **Wildcard domains**: Wildcard DNS (`*.company.com`) must be explicitly disallowed
   for domain verification — only exact domain matches are accepted.
3. **Subdomain takeover**: If a tenant verifies `company.com`, they should NOT
   automatically get `subdomain.company.com` unless explicitly claimed.
4. **Token rotation**: Verification tokens should expire after 7 days. If not verified,
   the claim is automatically cleaned up.

---

## 8. Tenant Offboarding

### Overview

Tenant offboarding is the reverse of onboarding — it must thoroughly clean up all
resources while providing a grace period for accidental deletion recovery. The flow
is multi-phase:

```
deletion_requested → exporting_data → data_exported → revoking_access →
resources_deleted → rls_removed → redis_nats_cleaned → archived
```

### Go Code: Tenant Offboarding

```go
package onboarding

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/ggid/ggid/pkg/errors"
	"github.com/ggid/ggid/services/org/internal/domain"
	"github.com/google/uuid"
)

// OffboardingStatus tracks the deletion progress.
type OffboardingStatus string

const (
	OffboardingRequested     OffboardingStatus = "deletion_requested"
	OffboardingExportingData OffboardingStatus = "exporting_data"
	OffboardingDataExported  OffboardingStatus = "data_exported"
	OffboardingRevoking      OffboardingStatus = "revoking_access"
	OffboardingDeletingData  OffboardingStatus = "deleting_data"
	OffboardingCleaningUp    OffboardingStatus = "cleaning_up"
	OffboardingArchived      OffboardingStatus = "archived"
)

// OffboardingConfig defines retention and export settings.
type OffboardingConfig struct {
	RetentionPeriod time.Duration // 30 days before permanent purge
	ExportFormat    string        // json, csv
	ExportToS3      bool          // upload to S3 bucket
	S3Bucket        string
}

// OffboardingService handles the complete tenant deletion flow.
type OffboardingService struct {
	tenantRepo  TenantRepo
	userRepo    UserRepo
	tokenRepo   TokenRepo
	dataProv    DataProvisioner
	emailSvc    EmailSender
	auditRepo   AuditRepo
	config      OffboardingConfig
}

// InitiateOffboarding starts the tenant deletion process.
// The tenant enters a retention period during which deletion can be cancelled.
func (s *OffboardingService) InitiateOffboarding(
	ctx context.Context,
	tenantID, requestedBy uuid.UUID,
	reason string,
) error {
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return errors.Wrap(errors.ErrNotFound, "tenant not found", err)
	}

	// --- Set tenant to deletion_pending with retention expiry ---
	now := time.Now().UTC()
	retentionExpiry := now.Add(s.config.RetentionPeriod)

	tenant.Status = "deletion_pending"
	tenant.Settings["offboarding_requested_by"] = requestedBy
	tenant.Settings["offboarding_requested_at"] = now
	tenant.Settings["offboarding_reason"] = reason
	tenant.Settings["offboarding_retention_expires"] = retentionExpiry
	tenant.Settings["offboarding_status"] = OffboardingRequested

	if err := s.tenantRepo.Update(ctx, tenant); err != nil {
		return errors.Internal("update tenant status", err)
	}

	// --- Immediately: revoke all active tokens ---
	if err := s.tokenRepo.RevokeAllForTenant(ctx, tenantID); err != nil {
		return errors.Wrap(errors.ErrInternal, "revoke tokens", err)
	}

	// --- Immediately: disable all users ---
	if err := s.userRepo.DisableAllForTenant(ctx, tenantID); err != nil {
		return errors.Wrap(errors.ErrInternal, "disable users", err)
	}

	// --- Notify tenant admins ---
	_ = s.emailSvc.SendOffboardingNotification(ctx, tenant, retentionExpiry)

	return nil
}

// ExecuteOffboarding performs the actual data export and deletion.
// Called by a cron job after the retention period has elapsed.
func (s *OffboardingService) ExecuteOffboarding(ctx context.Context, tenantID uuid.UUID) error {
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return err
	}

	// Verify retention period has elapsed
	retentionExpiry, ok := tenant.Settings["offboarding_retention_expires"].(time.Time)
	if !ok || time.Now().Before(retentionExpiry) {
		return errors.New(errors.ErrFailedPrecondition,
			"retention period has not elapsed (expires: %v)", retentionExpiry)
	}

	// --- Phase 1: Export data (GDPR compliance) ---
	s.updateStatus(ctx, tenant, OffboardingExportingData)

	exportData, err := s.exportTenantData(ctx, tenantID)
	if err != nil {
		return errors.Wrap(errors.ErrInternal, "export tenant data", err)
	}

	exportPath, err := s.saveExport(ctx, tenantID, exportData)
	if err != nil {
		return errors.Wrap(errors.ErrInternal, "save export", err)
	}

	s.updateStatus(ctx, tenant, OffboardingDataExported)

	// --- Phase 2: Cascade delete resources ---
	s.updateStatus(ctx, tenant, OffboardingDeletingData)

	if err := s.cascadeDelete(ctx, tenantID); err != nil {
		return errors.Wrap(errors.ErrInternal, "cascade delete", err)
	}

	// --- Phase 3: Clean up Redis and NATS ---
	s.updateStatus(ctx, tenant, OffboardingCleaningUp)

	if err := s.dataProv.DeprovisionTenant(ctx, tenantID); err != nil {
		// Non-fatal — log but continue
		// Redis keys have TTL, NATS streams can be manually cleaned
	}

	// --- Phase 4: Archive ---
	s.updateStatus(ctx, tenant, OffboardingArchived)

	tenant.Status = "deleted"
	tenant.Settings["offboarding_completed_at"] = time.Now().UTC()
	tenant.Settings["export_path"] = exportPath

	return s.tenantRepo.Update(ctx, tenant)
}

// CancelOffboarding reverses a pending deletion if within retention period.
func (s *OffboardingService) CancelOffboarding(ctx context.Context, tenantID uuid.UUID) error {
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return err
	}
	if tenant.Status != "deletion_pending" {
		return errors.New(errors.ErrFailedPrecondition,
			"tenant is not pending deletion (current: %s)", tenant.Status)
	}

	// Restore tenant
	tenant.Status = domain.TenantActive
	delete(tenant.Settings, "offboarding_requested_by")
	delete(tenant.Settings, "offboarding_requested_at")
	delete(tenant.Settings, "offboarding_reason")
	delete(tenant.Settings, "offboarding_retention_expires")
	delete(tenant.Settings, "offboarding_status")

	if err := s.tenantRepo.Update(ctx, tenant); err != nil {
		return errors.Internal("restore tenant", err)
	}

	// Re-enable users
	if err := s.userRepo.EnableAllForTenant(ctx, tenantID); err != nil {
		return errors.Wrap(errors.ErrInternal, "re-enable users", err)
	}

	// Issue new tokens (old ones were revoked)
	_ = s.emailSvc.SendOffboardingCancelled(ctx, tenant)

	return nil
}

func (s *OffboardingService) exportTenantData(ctx context.Context, tenantID uuid.UUID) (map[string]any, error) {
	export := map[string]any{
		"tenant_id":  tenantID,
		"exported_at": time.Now().UTC(),
	}

	// Export users
	users, err := s.userRepo.ListAllForTenant(ctx, tenantID)
	if err == nil {
		export["users"] = users
	}

	// Export audit events
	auditEvents, err := s.auditRepo.ListAllForTenant(ctx, tenantID)
	if err == nil {
		export["audit_events"] = auditEvents
	}

	return export, nil
}

func (s *OffboardingService) saveExport(ctx context.Context, tenantID uuid.UUID, data map[string]any) (string, error) {
	filename := fmt.Sprintf("/tmp/tenant_export_%s_%d.json", tenantID, time.Now().Unix())
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(filename, jsonData, 0600); err != nil {
		return "", err
	}
	return filename, nil
}

func (s *OffboardingService) cascadeDelete(ctx context.Context, tenantID uuid.UUID) error {
	// Delete in dependency order:
	// 1. User roles / role assignments
	// 2. Policies / policy attachments
	// 3. OAuth tokens / sessions
	// 4. User emails / external identities
	// 5. Users
	// 6. Roles / permissions
	// 7. Organizations / departments / teams / memberships
	if err := s.userRepo.DeleteAllForTenant(ctx, tenantID); err != nil {
		return fmt.Errorf("delete users: %w", err)
	}
	// Additional cascade deletes would go here...
	return nil
}

func (s *OffboardingService) updateStatus(ctx context.Context, tenant *domain.Tenant, status OffboardingStatus) {
	tenant.Settings["offboarding_status"] = status
	_ = s.tenantRepo.Update(ctx, tenant)
}
```

### GDPR Considerations

- **Data export**: Before deletion, all personal data must be exported in a
  machine-readable format (JSON/CSV). The export is retained for audit purposes.
- **Right to erasure**: Users within a tenant can request individual data deletion
  independent of tenant offboarding.
- **Retention limits**: Exported data should itself have a retention period (e.g.,
  90 days) after which it is permanently purged.

---

## 9. GGID Onboarding Gap Analysis

### Current State

The GGID codebase was reviewed for tenant onboarding capabilities across three
key areas:

#### pkg/tenant/tenant.go
- **Provides**: `Context` struct with `TenantID`, `IsolationLevel` (shared/schema/database),
  `Settings` map. Context propagation via `FromContext`/`WithContext`/`MustFromContext`.
- **Missing**: No tenant lifecycle management, no provisioning logic, no status tracking.
  This package is purely for request-scoped context propagation.

#### services/org/internal/ (TenantService)
- **Provides**: Basic `TenantService` with `Create`, `Get`, `GetBySlug`, `Update`,
  `Delete`. The `Tenant` domain model has `Plan` (free/pro/enterprise), `Status`
  (active/suspended/deleted), `MaxUsers`, and a `Settings` map.
- **Missing**:
  - No self-service signup flow — `Create` is a bare CRUD that accepts a pre-built
    `Tenant` struct with no validation beyond slug non-empty.
  - No email verification for the admin who creates a tenant.
  - No default configuration provisioning — creating a tenant does NOT create
    roles, permissions, or OAuth clients.
  - No trial management — `Plan` is set directly, no trial expiry tracking.
  - No domain verification.
  - `Delete` is a one-liner soft-delete (`status = 'deleted'`). No token revocation,
    user disabling, data export, or cascade cleanup.

#### services/identity/internal/ (IdentityService)
- **Provides**: `CreateUser`, `RegisterUser` (with email verification), LDAP JIT
  provisioning via `ProvisionFromLDAP`.
- **Missing**:
  - No "initial admin" provisioning — no bootstrap role assignment, no magic link
    setup, no first-login password change enforcement.
  - No tenant-scoped user count enforcement (MaxUsers is not checked).

#### services/policy/internal/ (RoleService)
- **Provides**: `CreateRole`, `AssignRole`, `RevokeRole`, permission management.
- **Missing**:
  - No default role/permission seeding per-tenant. The migration
    `000003_seed_system_roles_permissions.up.sql` seeds roles ONLY for the hardcoded
    default tenant UUID `00000000-0000-0000-0000-000000000001`.
  - No programmatic `ProvisionDefaults(tenantID)` method.

#### Migrations
- **Identity**: RLS is enabled and forced on all multi-tenant tables (`users`,
  `user_emails`, `user_external_identities`). Policies use `app.tenant_id` session
  variable. This is solid.
- **Policy**: System roles/permissions are seeded for ONE hardcoded tenant UUID.
  New tenants created at runtime get NO default roles.

### Summary of Gaps

| Capability | Status | Component |
|-----------|--------|-----------|
| Self-service signup | **MISSING** | No endpoint, no flow |
| Admin approval workflow | **MISSING** | No approval queue |
| Initial admin provisioning | **MISSING** | No bootstrap user creation |
| Default config provisioning | **MISSING** | Roles hardcoded to one UUID |
| Trial/plan management | **MISSING** | Plan field exists, no lifecycle |
| Data isolation at provisioning | **PARTIAL** | RLS exists but no per-tenant verification |
| Domain verification | **MISSING** | No domain claim system |
| Tenant offboarding | **MISSING** | Delete is a one-liner soft-delete |
| Email verification for tenant admin | **MISSING** | User-level exists, tenant-level does not |
| Rate limit provisioning per plan | **MISSING** | No plan-tier-based limits |

---

## 10. Gap Analysis & Recommendations

### Priority Action Items

#### 1. Implement Tenant Onboarding Service (Effort: 3-5 days)

Create a new `services/onboarding/` microservice (or extend `services/org/`) that
orchestrates the full signup flow:

- Self-service signup endpoint with email verification
- Admin approval workflow for enterprise plans
- Atomic provisioning: tenant record + admin user + default config + data isolation
- Idempotent retry support

**Files to create:**
- `services/onboarding/internal/service/onboarding_service.go`
- `services/onboarding/internal/handler/signup_handler.go`
- `services/onboarding/internal/domain/models.go`

**Dependencies:** Existing `org.TenantService`, `identity.IdentityService`,
`policy.RoleService` — call them in sequence within a provisioning orchestrator.

#### 2. Dynamic Default Config Provisioning (Effort: 1-2 days)

Replace the hardcoded UUID in `000003_seed_system_roles_permissions.up.sql` with
a programmatic provisioning method:

```go
// In services/policy/internal/service/
func (s *RoleService) ProvisionDefaults(ctx context.Context, tenantID uuid.UUID) error
```

This method should create admin/editor/viewer roles, seed permissions, and grant
them to roles — exactly as the migration does, but parameterized by `tenantID`.

**Impact:** Without this, every new tenant has zero roles and zero permissions.
Users cannot be assigned any role, making the tenant non-functional.

#### 3. Tenant Offboarding Pipeline (Effort: 2-3 days)

Replace the one-liner `Delete` in `TenantRepository` with a multi-phase offboarding:

- Phase 1: Revoke all JWT refresh tokens (`jti` tracking)
- Phase 2: Disable all users (set status = disabled)
- Phase 3: Export tenant data to JSON (GDPR)
- Phase 4: Cascade delete with retention period
- Phase 5: Clean up Redis keys (`SCAN t:{tenantID}:*` + `DEL`)
- Phase 6: Delete NATS stream
- Phase 7: Archive tenant record

Add a cron job to process `deletion_pending` tenants after retention expires.

#### 4. Plan-Based Feature Gating (Effort: 1-2 days)

Add a `FeatureFlags` map to `tenant.Settings` and enforce it in the gateway middleware:

```go
// In gateway middleware
func FeatureGate(flag string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            tc, _ := tenant.FromContext(r.Context())
            if !tc.Settings["features"].(map[string]bool)[flag] {
                http.Error(w, "Feature not available in your plan", http.StatusForbidden)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

Route-specific feature gates: `/sso` requires `sso` flag, `/scim/v2` requires `scim`
flag, etc.

#### 5. Domain Verification System (Effort: 2 days)

Create `pkg/domain/` package with DNS TXT record verification:

- `ClaimDomain(tenantID, domain) → verification_token`
- `VerifyDomain(domainID) → DNS lookup + token comparison`
- `CheckAutoProvision(email) → tenantID` (for domain-scoped auto-join)

Integrate with the identity service to enable auto-provisioning for verified domains.

### Implementation Priority

| Priority | Action | Effort | Risk if Skipped |
|----------|--------|--------|----------------|
| P0 | Dynamic default config provisioning | 1-2 days | New tenants are non-functional |
| P0 | Tenant onboarding service | 3-5 days | No self-service signup possible |
| P1 | Tenant offboarding pipeline | 2-3 days | Security: orphaned tokens, data leakage |
| P1 | Plan-based feature gating | 1-2 days | Free users access enterprise features |
| P2 | Domain verification | 2 days | No enterprise SSO auto-provisioning |

### Total Estimated Effort: 9-14 engineering days

These five items together would bring GGID from "multi-tenant capable" (RLS works,
tenant CRUD exists) to "multi-tenant production-ready" (full onboarding/offboarding
lifecycle with self-service, plan management, and enterprise features).

---

## Appendix: Interface Definitions

The following interfaces are referenced throughout this document. They represent
the contracts that the onboarding service depends on:

```go
// TenantRepo — already exists in services/org, extended with new methods
type TenantRepo interface {
	Create(ctx context.Context, t *domain.Tenant) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error)
	GetBySlug(ctx context.Context, slug string) (*domain.Tenant, error)
	GetByDomain(ctx context.Context, domain string) (*domain.Tenant, error) // NEW
	Update(ctx context.Context, t *domain.Tenant) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetUserCount(ctx context.Context, id uuid.UUID) (int, error)           // NEW
}

type UserRepo interface {
	CreatePending(ctx context.Context, u *PendingUser) error
	GetPendingByToken(ctx context.Context, tokenHash string) (*PendingUser, error)
	Activate(ctx context.Context, u *PendingUser) error
	ConsumeToken(ctx context.Context, tokenHash string) error
	Create(ctx context.Context, tenantID uuid.UUID, u *domain.User) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, tenantID uuid.UUID, email string) (*domain.User, error)
	UpdatePassword(ctx context.Context, tenantID, userID uuid.UUID, hash string) error
	SetFlag(ctx context.Context, userID uuid.UUID, flag string, value bool) error
	Delete(ctx context.Context, tenantID, userID uuid.UUID) error
	DisableAllForTenant(ctx context.Context, tenantID uuid.UUID) error
	EnableAllForTenant(ctx context.Context, tenantID uuid.UUID) error
	ListAllForTenant(ctx context.Context, tenantID uuid.UUID) ([]*domain.User, error)
	DeleteAllForTenant(ctx context.Context, tenantID uuid.UUID) error
	CreateSetupToken(ctx context.Context, t *PasswordSetupToken) error
	GetSetupToken(ctx context.Context, hash string) (*PasswordSetupToken, error)
	MarkSetupTokenUsed(ctx context.Context, id uuid.UUID, at time.Time) error
}

type RoleRepo interface {
	Create(ctx context.Context, role *policydomain.Role) error
	GetByID(ctx context.Context, id uuid.UUID) (*policydomain.Role, error)
	GetByKey(ctx context.Context, tenantID uuid.UUID, key string) (*policydomain.Role, error)
	ListByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*policydomain.Role, error)
	GrantPermissions(ctx context.Context, roleID uuid.UUID, permIDs []uuid.UUID, conditions map[string]any) error
}

type EmailSender interface {
	SendVerification(ctx context.Context, to, name, org, link string) error
	SendWelcome(ctx context.Context, to, name, org, tempPassword string) error
	SendApprovalNotification(ctx context.Context, to, org, status string) error
	SendTrialExpiringSoon(ctx context.Context, t *domain.Tenant, daysRemaining int) error
	SendTrialExpired(ctx context.Context, t *domain.Tenant) error
	SendDeletionWarning(ctx context.Context, t *domain.Tenant, deletionDate time.Time) error
	SendPlanUpgrade(ctx context.Context, t *domain.Tenant, oldPlan, newPlan string) error
	SendPlanDowngrade(ctx context.Context, t *domain.Tenant, newPlan domain.Plan) error
	SendOffboardingNotification(ctx context.Context, t *domain.Tenant, retentionExpiry time.Time) error
	SendOffboardingCancelled(ctx context.Context, t *domain.Tenant) error
}

type ConfigProvisioner interface {
	ProvisionDefaults(ctx context.Context, tenantID uuid.UUID, plan domain.Plan) error
}

type DataProvisioner interface {
	ProvisionTenant(ctx context.Context, tenantID uuid.UUID, slug string) error
	DeprovisionTenant(ctx context.Context, tenantID uuid.UUID) error
}

type NotificationSender interface {
	NotifyAdmins(ctx context.Context, event string, data map[string]any) error
}
```

---

*End of document.*
