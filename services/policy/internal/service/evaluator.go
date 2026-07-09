// Package service implements the Policy Engine business logic.
package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/ggid/ggid/pkg/errors"
	"github.com/ggid/ggid/services/policy/internal/domain"
	"github.com/ggid/ggid/services/policy/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Evaluator is the core permission evaluation engine.
// It combines RBAC (role-permission checks) and ABAC (policy evaluation)
// to produce a final allow/deny decision.
//
// Evaluation order:
//  1. Resolve user's roles including inherited ancestors.
//  2. Collect permissions from all roles — if any matches, RBAC allows.
//  3. Collect ABAC policies attached to the user and their roles.
//  4. Deny policies always override allow.
//  5. Default deny if no explicit allow.
type Evaluator struct {
	roleRepo     *repository.RoleRepository
	permRepo     *repository.PermissionRepository
	policyRepo   *repository.PolicyRepository
	userRoleRepo *repository.UserRoleRepository
}

// NewEvaluator creates a new permission evaluator.
func NewEvaluator(
	roleRepo *repository.RoleRepository,
	permRepo *repository.PermissionRepository,
	policyRepo *repository.PolicyRepository,
	userRoleRepo *repository.UserRoleRepository,
) *Evaluator {
	return &Evaluator{
		roleRepo:     roleRepo,
		permRepo:     permRepo,
		policyRepo:   policyRepo,
		userRoleRepo: userRoleRepo,
	}
}

// NewEvaluatorFromPool is a convenience constructor that creates all needed repos from a pool.
func NewEvaluatorFromPool(db *pgxpool.Pool) *Evaluator {
	return NewEvaluator(
		repository.NewRoleRepository(db),
		repository.NewPermissionRepository(db),
		repository.NewPolicyRepository(db),
		repository.NewUserRoleRepository(db),
	)
}

// Check performs a permission check and returns a boolean.
func (e *Evaluator) Check(ctx context.Context, req *domain.CheckRequest) (*domain.CheckResult, error) {
	if req.UserID == uuid.Nil {
		return &domain.CheckResult{Allowed: false, Reason: "anonymous user"}, nil
	}

	// Step 1: Get the user's direct role assignments.
	userRoleIDs, err := e.userRoleRepo.GetRoleIDsForUser(ctx, req.UserID)
	if err != nil {
		return nil, errors.Wrap(errors.ErrInternal, "get user roles", err)
	}
	if len(userRoleIDs) == 0 {
		return &domain.CheckResult{Allowed: false, Reason: "user has no role assignments"}, nil
	}

	// Step 2: Resolve role inheritance — collect all role IDs including ancestors.
	allRoleIDs := make(map[uuid.UUID]bool)
	for _, roleID := range userRoleIDs {
		ancestorIDs, err := e.roleRepo.GetAncestorChain(ctx, roleID)
		if err != nil {
			return nil, errors.Wrap(errors.ErrInternal, "resolve role chain", err)
		}
		for _, id := range ancestorIDs {
			allRoleIDs[id] = true
		}
	}

	resolvedIDs := make([]uuid.UUID, 0, len(allRoleIDs))
	for id := range allRoleIDs {
		resolvedIDs = append(resolvedIDs, id)
	}

	// Step 3: RBAC check — see if any permission matches.
	rbacAllowed := false
	perms, err := e.roleRepo.GetRolePermissions(ctx, resolvedIDs)
	if err != nil {
		return nil, errors.Wrap(errors.ErrInternal, "get role permissions", err)
	}
	for _, perm := range perms {
		if perm.ResourceType == req.ResourceType && perm.Action == req.Action {
			rbacAllowed = true
			break
		}
		// Also support wildcard action matching on permission key.
		if perm.ResourceType == req.ResourceType && perm.Action == "*" {
			rbacAllowed = true
			break
		}
	}

	// Step 4: ABAC evaluation — check policies attached to user and roles.
	abacPolicies, err := e.policyRepo.GetPoliciesForUserAndRoles(ctx, req.UserID, resolvedIDs)
	if err != nil {
		return nil, errors.Wrap(errors.ErrInternal, "get abac policies", err)
	}

	abacDenied := false
	abacAllowed := false
	var denyReason string
	var allowReason string

	// Sort: evaluate deny first (by priority desc), then allow.
	for _, p := range abacPolicies {
		if !matchActions(p.Actions, req.Action) {
			continue
		}
		if req.Resource != "" && len(p.Resources) > 0 && !matchResources(p.Resources, req.Resource) {
			continue
		}

		switch p.Effect {
		case domain.EffectDeny:
			abacDenied = true
			denyReason = fmt.Sprintf("policy:%s", p.Name)
		case domain.EffectAllow:
			abacAllowed = true
			allowReason = fmt.Sprintf("policy:%s", p.Name)
		}
	}

	// Step 5: Combine decisions. Deny always wins.
	if abacDenied {
		return &domain.CheckResult{
			Allowed:   false,
			Reason:    fmt.Sprintf("explicitly denied by %s", denyReason),
			MatchedBy: denyReason,
		}, nil
	}

	if rbacAllowed {
		return &domain.CheckResult{
			Allowed:   true,
			Reason:    "allowed by RBAC role permission",
			MatchedBy: "rbac",
		}, nil
	}

	if abacAllowed {
		return &domain.CheckResult{
			Allowed:   true,
			Reason:    fmt.Sprintf("allowed by %s", allowReason),
			MatchedBy: allowReason,
		}, nil
	}

	return &domain.CheckResult{
		Allowed: false,
		Reason:  "no matching allow rule found",
	}, nil
}

// matchActions checks if any pattern in actions matches the given action.
// Supports wildcard: "iam:users:*" matches "iam:users:read".
func matchActions(patterns []string, action string) bool {
	for _, pattern := range patterns {
		if pattern == "*" || pattern == action {
			return true
		}
		if strings.HasSuffix(pattern, "*") {
			prefix := strings.TrimSuffix(pattern, "*")
			if strings.HasPrefix(action, prefix) {
				return true
			}
		}
	}
	return false
}

// matchResources checks if any pattern in resources matches the given resource.
// Supports glob-style wildcards: "arn:ggid:iam::tenant:user/*" matches "arn:ggid:iam::tenant:user/123".
func matchResources(patterns []string, resource string) bool {
	for _, pattern := range patterns {
		if pattern == "*" || pattern == resource {
			return true
		}
		if matchGlob(pattern, resource) {
			return true
		}
	}
	return false
}

// matchGlob performs a simple glob match supporting * as a wildcard.
func matchGlob(pattern, s string) bool {
	// Split pattern by * and check that parts appear in order.
	parts := strings.Split(pattern, "*")
	if len(parts) == 1 {
		return pattern == s
	}

	idx := 0
	for i, part := range parts {
		if part == "" {
			continue
		}
		if i == 0 && !strings.HasPrefix(s, part) {
			return false
		}
		pos := strings.Index(s[idx:], part)
		if pos == -1 {
			return false
		}
		idx += pos + len(part)
	}

	// If pattern doesn't end with *, the last part must be a suffix.
	if !strings.HasSuffix(pattern, "*") {
		lastPart := parts[len(parts)-1]
		return strings.HasSuffix(s, lastPart)
	}
	return true
}
