// Package service implements the Policy Engine business logic.
package service

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/ggid/ggid/pkg/errors"
	"github.com/ggid/ggid/services/policy/internal/domain"
	"github.com/google/uuid"
)

// RoleReader provides read access to roles and role-permission mappings.
// Implemented by *repository.RoleRepository; mocked in tests.
type RoleReader interface {
	GetAncestorChain(ctx context.Context, roleID uuid.UUID) ([]uuid.UUID, error)
	GetRolePermissions(ctx context.Context, roleIDs []uuid.UUID) ([]*domain.Permission, error)
}

// UserRoleReader provides read access to user-role assignments.
// Implemented by *repository.UserRoleRepository; mocked in tests.
type UserRoleReader interface {
	GetUserRoles(ctx context.Context, userID uuid.UUID) ([]*domain.UserRole, error)
}

// PolicyReader provides read access to ABAC policies.
// Implemented by *repository.PolicyRepository; mocked in tests.
type PolicyReader interface {
	GetPoliciesForUserAndRoles(ctx context.Context, userID uuid.UUID, roleIDs []uuid.UUID) ([]*domain.Policy, error)
}

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
// DecisionLogger is an optional callback invoked after every Check() call
// to record the policy decision (e.g. to an audit pipeline).
type DecisionLogger func(ctx context.Context, req *domain.CheckRequest, result *domain.CheckResult)

// LogDecisions is an in-memory store of recent decisions for inspection.
var (
	decisionMu       sync.Mutex
	decisionLog      []DecisionEntry
	decisionLoggerFn DecisionLogger
	maxDecisions     = 1000
)

// DecisionEntry records a single policy evaluation.
type DecisionEntry struct {
	Timestamp  time.Time
	UserID     uuid.UUID
	TenantID   uuid.UUID
	Action     string
	Resource   string
	Allowed    bool
	Reason     string
	MatchedBy  string
}

// SetDecisionLogger installs a custom decision logger callback.
func SetDecisionLogger(fn DecisionLogger) {
	decisionMu.Lock()
	defer decisionMu.Unlock()
	decisionLoggerFn = fn
}

// GetRecentDecisions returns up to limit recent decision entries (most recent first).
func GetRecentDecisions(limit int) []DecisionEntry {
	decisionMu.Lock()
	defer decisionMu.Unlock()
	if limit > len(decisionLog) {
		limit = len(decisionLog)
	}
	// Return most recent entries
	start := len(decisionLog) - limit
	result := make([]DecisionEntry, limit)
	copy(result, decisionLog[start:])
	// Reverse for most-recent-first
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return result
}

// logDecision records a decision to the in-memory store + invokes the callback.
func logDecision(ctx context.Context, req *domain.CheckRequest, result *domain.CheckResult) {
	entry := DecisionEntry{
		Timestamp: time.Now().UTC(),
		UserID:    req.UserID,
		TenantID:  req.TenantID,
		Action:    req.Action,
		Resource:  req.Resource,
		Allowed:   result.Allowed,
		Reason:    result.Reason,
		MatchedBy: result.MatchedBy,
	}

	decisionMu.Lock()
	decisionLog = append(decisionLog, entry)
	if len(decisionLog) > maxDecisions {
		decisionLog = decisionLog[len(decisionLog)-maxDecisions:]
	}
	logger := decisionLoggerFn
	decisionMu.Unlock()

	if logger != nil {
		logger(ctx, req, result)
	}
}

type Evaluator struct {
	roleReader     RoleReader
	userRoleReader UserRoleReader
	policyReader   PolicyReader
}

// NewEvaluator creates a new permission evaluator from the individual readers.
func NewEvaluator(roleReader RoleReader, userRoleReader UserRoleReader, policyReader PolicyReader) *Evaluator {
	return &Evaluator{
		roleReader:     roleReader,
		userRoleReader: userRoleReader,
		policyReader:   policyReader,
	}
}

// Check performs a permission check and returns a boolean.
func (e *Evaluator) Check(ctx context.Context, req *domain.CheckRequest) (*domain.CheckResult, error) {
	if req.UserID == uuid.Nil {
		return &domain.CheckResult{Allowed: false, Reason: "anonymous user"}, nil
	}

	// Step 1: Get the user's direct role assignments (with ExpiresAt for filtering).
	userRoles, err := e.userRoleReader.GetUserRoles(ctx, req.UserID)
	if err != nil {
		return nil, errors.Wrap(errors.ErrInternal, "get user roles", err)
	}

	// Step 1b: Filter out expired role assignments (defense-in-depth).
	// Even if the database query already filters by expires_at > NOW(),
	// we enforce it here so any caching layer or alternative reader
	// implementation cannot bypass expiration.
	now := time.Now().UTC()
	var userRoleIDs []uuid.UUID
	for _, ur := range userRoles {
		if ur.ExpiresAt != nil && ur.ExpiresAt.Before(now) {
			continue // role assignment has expired
		}
		userRoleIDs = append(userRoleIDs, ur.RoleID)
	}
	if len(userRoleIDs) == 0 {
		return &domain.CheckResult{Allowed: false, Reason: "user has no role assignments"}, nil
	}

	// Step 2: Resolve role inheritance — collect all role IDs including ancestors.
	allRoleIDs := make(map[uuid.UUID]bool)
	for _, roleID := range userRoleIDs {
		ancestorIDs, err := e.roleReader.GetAncestorChain(ctx, roleID)
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
	perms, err := e.roleReader.GetRolePermissions(ctx, resolvedIDs)
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
	abacPolicies, err := e.policyReader.GetPoliciesForUserAndRoles(ctx, req.UserID, resolvedIDs)
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

		// Evaluate ABAC conditions if the policy or request has them.
		if len(p.Conditions) > 0 {
			if !matchConditions(p.Conditions, req.Conditions) {
				continue
			}
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
		result := &domain.CheckResult{
			Allowed:   false,
			Reason:    fmt.Sprintf("explicitly denied by %s", denyReason),
			MatchedBy: denyReason,
		}
		logDecision(ctx, req, result)
		return result, nil
	}

	if rbacAllowed {
		result := &domain.CheckResult{
			Allowed:   true,
			Reason:    "allowed by RBAC role permission",
			MatchedBy: "rbac",
		}
		logDecision(ctx, req, result)
		return result, nil
	}

	if abacAllowed {
		result := &domain.CheckResult{
			Allowed:   true,
			Reason:    fmt.Sprintf("allowed by %s", allowReason),
			MatchedBy: allowReason,
		}
		logDecision(ctx, req, result)
		return result, nil
	}

	result := &domain.CheckResult{
		Allowed: false,
		Reason:  "no matching allow rule found",
	}
	logDecision(ctx, req, result)
	return result, nil
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

// matchConditions evaluates ABAC conditions (AWS IAM-style operators).
// Policy conditions use operator keys like "StringEquals", "StringLike",
// "NumericLessThan", "Bool", "IpAddress", etc.
// Each operator maps to attribute->value pairs that are checked against
// the request conditions.
//
// Example policy conditions:
//
//	{
//	  "StringEquals": {"department": "engineering"},
//	  "NumericLessThan": {"hour": 18},
//	  "StringLike": {"name": "admin-*"}
//	}
//
// If no request conditions are provided, policies with conditions are
// considered non-matching (fail-closed).
func matchConditions(policyConds map[string]any, requestConds map[string]any) bool {
	if len(policyConds) == 0 {
		return true // No conditions = always match
	}
	if len(requestConds) == 0 {
		return false // Fail-closed: policy has conditions but request has none
	}

	for operator, condsVal := range policyConds {
		condMap, ok := condsVal.(map[string]any)
		if !ok {
			continue
		}
		for attr, expectedVal := range condMap {
			actualVal, exists := requestConds[attr]
			if !exists {
				return false // Required attribute missing from request
			}
			if !evaluateOperator(operator, expectedVal, actualVal) {
				return false
			}
		}
	}
	return true
}

// evaluateOperator checks a single condition using the given operator.
func evaluateOperator(operator string, expected, actual any) bool {
	switch operator {
	// String operators
	case "StringEquals":
		return toStr(expected) == toStr(actual)
	case "StringNotEquals":
		return toStr(expected) != toStr(actual)
	case "StringEqualsIgnoreCase":
		return strings.EqualFold(toStr(expected), toStr(actual))
	case "StringLike":
		return matchGlob(toStr(expected), toStr(actual))
	case "StringNotLike":
		return !matchGlob(toStr(expected), toStr(actual))

	// Numeric operators
	case "NumericEquals":
		return toFloat64(expected) == toFloat64(actual)
	case "NumericNotEquals":
		return toFloat64(expected) != toFloat64(actual)
	case "NumericLessThan":
		return toFloat64(actual) < toFloat64(expected)
	case "NumericLessThanEquals":
		return toFloat64(actual) <= toFloat64(expected)
	case "NumericGreaterThan":
		return toFloat64(actual) > toFloat64(expected)
	case "NumericGreaterThanEquals":
		return toFloat64(actual) >= toFloat64(expected)

	// Boolean operators
	case "Bool":
		return toBool(expected) == toBool(actual)

	// Date/time operators (RFC 3339 comparison)
	case "DateLessThan":
		return toDate(actual).Before(toDate(expected))
	case "DateGreaterThan":
		return toDate(actual).After(toDate(expected))

	// IP address operators
	case "IpAddress":
		return matchIP(toStr(expected), toStr(actual))
	case "NotIpAddress":
		return !matchIP(toStr(expected), toStr(actual))

	default:
		// Unknown operator: fail-closed
		return false
	}
}

// --- Type coercion helpers ---

func toStr(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case float64:
		return fmt.Sprintf("%g", val)
	case int:
		return fmt.Sprintf("%d", val)
	case bool:
		return fmt.Sprintf("%t", val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

func toFloat64(v any) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case string:
		var f float64
		fmt.Sscanf(val, "%g", &f)
		return f
	default:
		return 0
	}
}

func toBool(v any) bool {
	switch val := v.(type) {
	case bool:
		return val
	case string:
		return val == "true" || val == "1"
	default:
		return false
	}
}

func toDate(v any) time.Time {
	s := toStr(v)
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t, err = time.Parse("2006-01-02", s)
		if err != nil {
			return time.Time{}
		}
	}
	return t
}

func matchIP(cidrOrIP, actualIP string) bool {
	if !strings.Contains(cidrOrIP, "/") {
		return cidrOrIP == actualIP
	}
	// Parse CIDR and check if actualIP is in range
	_, ipNet, err := net.ParseCIDR(cidrOrIP)
	if err != nil {
		return false
	}
	ip := net.ParseIP(actualIP)
	if ip == nil {
		return false
	}
	return ipNet.Contains(ip)
}
