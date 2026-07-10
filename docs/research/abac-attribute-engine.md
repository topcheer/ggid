# ABAC Policy Evaluation Engine for IAM Systems

> **Focus**: Architecture and implementation patterns for Attribute-Based Access Control (ABAC) policy engines, with an audit of the GGID policy service and actionable gap analysis.

---

## Table of Contents

1. [ABAC vs RBAC](#1-abac-vs-rbac)
2. [Attribute Sources](#2-attribute-sources)
3. [Policy Combining Algorithms](#3-policy-combining-algorithms)
4. [Policy Definition Language](#4-policy-definition-language)
5. [Obligation and Advice](#5-obligation-and-advice)
6. [Policy Evaluation Pipeline](#6-policy-evaluation-pipeline)
7. [Policy Lifecycle Management](#7-policy-lifecycle-management)
8. [GGID Policy Engine Audit](#8-ggid-policy-engine-audit)
9. [Gap Analysis and Recommendations](#9-gap-analysis-and-recommendations)

---

## 1. ABAC vs RBAC

### Why ABAC is Needed Beyond RBAC

Role-Based Access Control (RBAC) assigns permissions to roles, and users inherit permissions through role membership. This works well for coarse-grained access ("admins can manage users") but breaks down when access decisions must consider **dynamic context**:

| Dimension | RBAC | ABAC |
|-----------|------|------|
| Decision basis | Role membership | Subject + Resource + Environment attributes |
| Granularity | Coarse (role-level) | Fine (attribute-level) |
| Dynamic context | No (requires role reassignment) | Yes (evaluated per-request) |
| Policy expressiveness | "User with role X can do Y" | "User in department X with clearance >= secret can do Y on resource Z during business hours from a managed device" |
| Scaling complexity | Role explosion | Composable attribute sets |

**ABAC enables**: time-based restrictions (business hours only), location-based restrictions (corporate network only), device posture checks (managed device required), data classification (confidential documents require elevated clearance), and delegated administration (department heads approve their team's access).

### When to Use Each Model

- **RBAC alone**: Simple applications with well-defined, stable role sets (<50 roles). Startups, internal tools, small teams.
- **ABAC alone**: Rare. High-security environments where every access decision is dynamic (government, defense).
- **Combined RBAC + ABAC** (recommended): RBAC provides the baseline role-permission structure; ABAC policies layer on top to add contextual constraints. This is the model GGID currently follows — RBAC permissions are checked first, then ABAC policies can override with allow or deny.

### NIST SP 800-162 Overview

NIST Special Publication 800-162 ("Guide to ABAC Definition and Planning") defines the ABAC reference architecture:

- **Policy Administration Point (PAP)**: Where policies are authored and managed.
- **Policy Decision Point (PDP)**: Where access decisions are computed.
- **Policy Enforcement Point (PEP)**: Where decisions are enforced (e.g., API gateway, application middleware).
- **Policy Information Point (PIP)**: Where attributes are retrieved from authoritative sources.

The PEP intercepts the access request, calls the PIP to resolve attributes, sends the request + attributes to the PDP, receives a decision, and enforces it. GGID's Gateway serves as the PEP; the Policy service is the PDP; repositories and external services serve as PIPs.

---

## 2. Attribute Sources

### Attribute Categories

**Subject Attributes** describe the user making the request:

| Source | Attributes | Resolution |
|--------|-----------|------------|
| JWT claims | sub, tenant_id, scopes, roles | Parsed at request time from token |
| User profile (DB) | department, title, manager_id | Queried from identity service |
| HR system | clearance_level, employment_status | External API call (cached) |

**Resource Attributes** describe the object being accessed:

| Source | Attributes | Resolution |
|--------|-----------|------------|
| Database row | owner_id, classification, created_by | Queried from resource service |
| Metadata service | tags, labels, project_id | External API or DB |
| File system | ACL, MIME type, size | File stat |

**Environment Attributes** describe the context of the request:

| Source | Attributes | Resolution |
|--------|-----------|------------|
| HTTP request | client IP, user-agent, TLS version | Extracted from request headers |
| Device posture | MDM enrollment, OS version | Device attestation service |
| Time | hour_of_day, day_of_week, timezone | Computed at request time |
| Session | MFA verified, session age | Session store (Redis) |

### Resolution Timing

- **Request-time resolution**: Attributes are fetched fresh on each access request. Most accurate but adds latency.
- **Pre-computed resolution**: Attributes are cached at session creation or periodically refreshed. Faster but may be stale.

Best practice: cache subject and resource attributes in Redis with a short TTL (30-60 seconds), always resolve environment attributes at request time.

### Go: Attribute Resolver

```go
package abac

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// AttributeResolver resolves attributes from multiple sources.
type AttributeResolver struct {
	rdb         *redis.Client
	identitySvc IdentityClient
	ttl         time.Duration
}

// IdentityClient fetches subject attributes from the identity service.
type IdentityClient interface {
	GetUserAttributes(ctx context.Context, userID string) (map[string]any, error)
	GetResourceAttributes(ctx context.Context, resourceType, resourceID string) (map[string]any, error)
}

// NewAttributeResolver creates a resolver with Redis caching.
func NewAttributeResolver(rdb *redis.Client, identitySvc IdentityClient) *AttributeResolver {
	return &AttributeResolver{
		rdb:         rdb,
		identitySvc: identitySvc,
		ttl:         60 * time.Second,
	}
}

// ResolveSubject fetches subject attributes, using cache when possible.
func (r *AttributeResolver) ResolveSubject(ctx context.Context, userID string, jwtClaims map[string]any) (map[string]any, error) {
	// Start with JWT claims (already trusted, no cache needed).
	attrs := make(map[string]any)
	for k, v := range jwtClaims {
		attrs["subject."+k] = v
	}

	// Try cache for profile attributes.
	cacheKey := fmt.Sprintf("attrs:subject:%s", userID)
	cached, err := r.rdb.Get(ctx, cacheKey).Bytes()
	if err == nil {
		var profile map[string]any
		if json.Unmarshal(cached, &profile) == nil {
			for k, v := range profile {
				attrs["subject."+k] = v
			}
			return attrs, nil
		}
	}

	// Cache miss — fetch from identity service.
	profile, err := r.identitySvc.GetUserAttributes(ctx, userID)
	if err != nil {
		// Graceful degradation: use JWT claims only.
		return attrs, nil
	}
	for k, v := range profile {
		attrs["subject."+k] = v
	}

	// Store in cache (best-effort).
	if data, err := json.Marshal(profile); err == nil {
		r.rdb.Set(ctx, cacheKey, data, r.ttl)
	}
	return attrs, nil
}

// ResolveResource fetches resource attributes from the identity service.
func (r *AttributeResolver) ResolveResource(ctx context.Context, resourceType, resourceID string) (map[string]any, error) {
	cacheKey := fmt.Sprintf("attrs:resource:%s:%s", resourceType, resourceID)
	cached, err := r.rdb.Get(ctx, cacheKey).Bytes()
	if err == nil {
		var attrs map[string]any
		if json.Unmarshal(cached, &attrs) == nil {
			return attrs, nil
		}
	}

	attrs, err := r.identitySvc.GetResourceAttributes(ctx, resourceType, resourceID)
	if err != nil {
		return map[string]any{}, nil // graceful degradation
	}

	if data, err := json.Marshal(attrs); err == nil {
		r.rdb.Set(ctx, cacheKey, data, r.ttl)
	}
	return attrs, nil
}

// ResolveEnvironment computes environment attributes at request time (never cached).
func ResolveEnvironment(clientIP, userAgent string, mfaVerified bool) map[string]any {
	now := time.Now().UTC()
	return map[string]any{
		"env.ip":             clientIP,
		"env.user_agent":     userAgent,
		"env.mfa_verified":   mfaVerified,
		"env.hour":           now.Hour(),
		"env.day_of_week":    int(now.Weekday()),
		"env.is_weekend":     now.Weekday() == time.Saturday || now.Weekday() == time.Sunday,
		"env.timestamp":      now.Format(time.RFC3339),
	}
}
```

---

## 3. Policy Combining Algorithms

When multiple policies apply to a request, the combining algorithm determines the final decision. GGID currently uses a fixed "deny-overrides" algorithm (deny always wins). XACML defines five standard combining algorithms:

### Algorithm Reference

| Algorithm | Rule | Use Case |
|-----------|------|----------|
| **Deny-overrides** | If any policy denies, result is Deny | High-security: fail-closed |
| **Permit-overrides** | If any policy permits, result is Permit | Maximize availability |
| **First-applicable** | First matching policy wins | Ordered policy chains |
| **Permit-unless-deny** | Permit unless explicitly denied | Default-allow systems |
| **Deny-unless-permit** | Deny unless explicitly permitted | Default-deny systems |

**Deny-overrides** is the safest default for IAM systems — a single deny policy blocks access regardless of other permits. This is what GGID currently implements.

**First-applicable** is useful for ordered rule sets where policies are evaluated by priority and the first match is authoritative (similar to firewall rules).

### Go: Policy Combining Engine

```go
package abac

// Decision is the outcome of a policy evaluation.
type Decision string

const (
	DecisionPermit  Decision = "permit"
	DecisionDeny    Decision = "deny"
	DecisionNotApp  Decision = "not_applicable" // policy did not match
)

// CombiningAlgorithm defines how multiple policy decisions are merged.
type CombiningAlgorithm func(decisions []Decision) Decision

// DenyOverrides: any deny wins. If no deny but at least one permit, permit.
// Otherwise not_applicable.
func DenyOverrides(decisions []Decision) Decision {
	hasPermit := false
	for _, d := range decisions {
		if d == DecisionDeny {
			return DecisionDeny
		}
		if d == DecisionPermit {
			hasPermit = true
		}
	}
	if hasPermit {
		return DecisionPermit
	}
	return DecisionNotApp
}

// PermitOverrides: any permit wins.
func PermitOverrides(decisions []Decision) Decision {
	hasDeny := false
	for _, d := range decisions {
		if d == DecisionPermit {
			return DecisionPermit
		}
		if d == DecisionDeny {
			hasDeny = true
		}
	}
	if hasDeny {
		return DecisionDeny
	}
	return DecisionNotApp
}

// FirstApplicable: first non-not_applicable decision wins.
func FirstApplicable(decisions []Decision) Decision {
	for _, d := range decisions {
		if d != DecisionNotApp {
			return d
		}
	}
	return DecisionNotApp
}

// PermitUnlessDeny: permit unless explicitly denied (no applicable = permit).
func PermitUnlessDeny(decisions []Decision) Decision {
	for _, d := range decisions {
		if d == DecisionDeny {
			return DecisionDeny
		}
	}
	return DecisionPermit
}

// DenyUnlessPermit: deny unless explicitly permitted (no applicable = deny).
func DenyUnlessPermit(decisions []Decision) Decision {
	for _, d := range decisions {
		if d == DecisionPermit {
			return DecisionPermit
		}
	}
	return DecisionDeny
}
```

---

## 4. Policy Definition Language

### DSL Options Compared

| DSL | Complexity | Expressiveness | Learning Curve | Use Case |
|-----|-----------|---------------|----------------|----------|
| **AWS IAM JSON** | Low | Medium | Low | Cloud-familiar teams, simple conditions |
| **ALFA** | Medium | High | Medium | Enterprise, XACML-compatible |
| **Rego (OPA)** | High | Very High | High | Cloud-native, Kubernetes, microservices |
| **CEL** | Medium | High | Medium | Embedded in Go apps, fast evaluation |

### Policy Structure

Every ABAC policy has three components:

1. **Target**: When does this policy apply? (subject, resource, action matching)
2. **Condition**: When is the policy true? (attribute expressions)
3. **Effect**: What is the outcome? (permit or deny)

GGID uses AWS IAM-style JSON policies with a `Conditions` map using operator keys (`StringEquals`, `NumericLessThan`, `IpAddress`, etc.). This is a pragmatic choice that covers most IAM use cases without the overhead of a full expression language.

### Go: Policy Parser (Simplified ALFA-like DSL)

```go
package abac

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// PolicyDoc is the user-facing policy document.
type PolicyDoc struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Effect      Decision          `json:"effect"`
	Actions     []string          `json:"actions"`
	Resources   []string          `json:"resources"`
	Conditions  map[string]any    `json:"conditions"`
	Combining   string            `json:"combining,omitempty"` // deny_overrides, permit_overrides, first_applicable
	Obligations []ObligationSpec  `json:"obligations,omitempty"`
}

// ParsePolicyDoc parses a JSON policy document.
func ParsePolicyDoc(data []byte) (*PolicyDoc, error) {
	var doc PolicyDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse policy JSON: %w", err)
	}
	if doc.Effect != DecisionPermit && doc.Effect != DecisionDeny {
		return nil, fmt.Errorf("effect must be 'permit' or 'deny', got '%s'", doc.Effect)
	}
	if len(doc.Actions) == 0 {
		doc.Actions = []string{"*"}
	}
	if doc.Combining == "" {
		doc.Combining = "deny_overrides"
	}
	return &doc, nil
}

// conditionExpr is a parsed boolean expression from a natural-language condition string.
type conditionExpr struct {
	Attribute string
	Operator  string
	Value     string
}

var exprPattern = regexp.MustCompile(`^(\w+(?:\.\w+)*)\s*(==|!=|<|<=|>|>=|in|not_in|like)\s*(.+)$`)

// ParseConditionExpr parses a natural-language condition like "subject.department == engineering".
func ParseConditionExpr(expr string) (*conditionExpr, error) {
	matches := exprPattern.FindStringSubmatch(strings.TrimSpace(expr))
	if len(matches) != 4 {
		return nil, fmt.Errorf("invalid condition expression: %s", expr)
	}
	return &conditionExpr{
		Attribute: matches[1],
		Operator:  matches[2],
		Value:     strings.Trim(matches[3], `"`),
	}, nil
}

// Evaluate checks the condition against resolved attributes.
func (c *conditionExpr) Evaluate(attrs map[string]any) bool {
	actual, ok := attrs[c.Attribute]
	if !ok {
		return false
	}
	actualStr := fmt.Sprintf("%v", actual)
	switch c.Operator {
	case "==":
		return actualStr == c.Value
	case "!=":
		return actualStr != c.Value
	case "like":
		return globMatch(c.Value, actualStr)
	default:
		return false
	}
}

func globMatch(pattern, s string) bool {
	parts := strings.Split(pattern, "*")
	for i, part := range parts {
		idx := strings.Index(s, part)
		if idx == -1 {
			return false
		}
		if i == 0 && idx != 0 {
			return false
		}
		s = s[idx+len(part):]
	}
	return true
}
```

---

## 5. Obligation and Advice

### Obligations vs Advice

| Feature | Obligation | Advice |
|---------|-----------|--------|
| Enforcement | Mandatory — PEP must fulfill | Optional — PEP may ignore |
| Failure handling | Decision is not applied | No effect on decision |
| Use case | Compliance logging, notifications | UX hints, warnings |
| Example | "Log all access to PII records" | "Show warning banner on export" |

Obligations are critical for compliance frameworks (HIPAA, GDPR, SOC 2). When a policy permits access to protected data, an obligation like `log_access` or `notify_data_owner` ensures the access is recorded even if the application forgets to log it.

### Go: Obligation Executor

```go
package abac

import (
	"context"
	"log"
)

// ObligationSpec defines a mandatory or advisory action tied to a policy decision.
type ObligationSpec struct {
	ID       string         `json:"id"`
	Type     string         `json:"type"`     // "obligation" or "advice"
	Action   string         `json:"action"`   // e.g. "log_access", "notify_owner"
	Params   map[string]any `json:"params"`
	Fulfilled bool          `json:"-"`
}

// ObligationHandler executes a specific obligation action.
type ObligationHandler interface {
	Execute(ctx context.Context, params map[string]any) error
}

// ObligationExecutor runs obligations after a policy decision.
type ObligationExecutor struct {
	handlers map[string]ObligationHandler
}

func NewObligationExecutor() *ObligationExecutor {
	return &ObligationExecutor{handlers: make(map[string]ObligationHandler)}
}

func (e *ObligationExecutor) Register(action string, handler ObligationHandler) {
	e.handlers[action] = handler
}

// ExecuteAll runs all mandatory obligations. If any mandatory obligation fails,
// the decision should be treated as Not Applicable (access must not be granted).
func (e *ObligationExecutor) ExecuteAll(ctx context.Context, obligations []ObligationSpec) error {
	for _, obl := range obligations {
		if obl.Type == "advice" {
			// Best-effort: try to execute, ignore failures.
			if h, ok := e.handlers[obl.Action]; ok {
				if err := h.Execute(ctx, obl.Params); err != nil {
					log.Printf("advice '%s' failed (non-critical): %v", obl.Action, err)
				}
			}
			continue
		}
		// Mandatory obligation.
		h, ok := e.handlers[obl.Action]
		if !ok {
			return fmt.Errorf("no handler for obligation '%s'", obl.Action)
		}
		if err := h.Execute(ctx, obl.Params); err != nil {
			return fmt.Errorf("obligation '%s' failed: %w", obl.Action, err)
		}
	}
	return nil
}

// LogAccessHandler logs access to protected resources (compliance obligation).
type LogAccessHandler struct{}

func (h *LogAccessHandler) Execute(ctx context.Context, params map[string]any) error {
	userID, _ := params["user_id"].(string)
	resource, _ := params["resource"].(string)
	action, _ := params["action"].(string)
	log.Printf("[OBLIGATION:audit] user=%s action=%s resource=%s", userID, action, resource)
	return nil
}
```

---

## 6. Policy Evaluation Pipeline

### Full Evaluation Flow

```
Request
  |
  v
[Attribute Resolution] -- subject attrs (JWT + cache)
  |                      -- resource attrs (DB + cache)
  |                      -- environment attrs (computed)
  v
[Target Matching]      -- which policies apply? (action + resource match)
  |
  v
[Condition Evaluation] -- do attribute values satisfy policy conditions?
  |
  v
[Combining Algorithm]  -- merge all applicable decisions
  |
  v
[Obligation Execution] -- run mandatory/advisory actions
  |
  v
Response (permit/deny + obligations)
```

### Performance Optimization

- **Attribute caching**: Redis cache with 60s TTL for subject/resource attributes.
- **Short-circuit evaluation**: In deny-overrides mode, return immediately on first deny.
- **Policy pre-filtering**: Filter policies by tenant + action + resource before evaluating conditions.
- **Condition index**: Build an inverted index from attribute names to policies that reference them, to avoid evaluating irrelevant policies.

### Go: Complete Evaluation Pipeline

```go
package abac

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Pipeline orchestrates the full ABAC evaluation flow.
type Pipeline struct {
	resolver   *AttributeResolver
	combining  CombiningAlgorithm
	obligations *ObligationExecutor
}

// EvalRequest is the input to the pipeline.
type EvalRequest struct {
	UserID       uuid.UUID
	TenantID     uuid.UUID
	Action       string
	ResourceType string
	ResourceID   string
	JWTClaims    map[string]any
	ClientIP     string
	UserAgent    string
	MFAVerified  bool
}

// EvalResponse is the output of the pipeline.
type EvalResponse struct {
	Decision       Decision
	Reason         string
	MatchedPolicies []string
	Obligations    []ObligationSpec
	EvalTimeMS     int64
}

// NewPipeline creates a new evaluation pipeline.
func NewPipeline(resolver *AttributeResolver, combining CombiningAlgorithm, oblExec *ObligationExecutor) *Pipeline {
	return &Pipeline{
		resolver:   resolver,
		combining:  combining,
		obligations: oblExec,
	}
}

// Evaluate runs the full pipeline and returns a decision.
func (p *Pipeline) Evaluate(ctx context.Context, req *EvalRequest, policies []PolicyDoc) (*EvalResponse, error) {
	start := time.Now()

	// Step 1: Resolve attributes.
	subjectAttrs, err := p.resolver.ResolveSubject(ctx, req.UserID.String(), req.JWTClaims)
	if err != nil {
		return nil, fmt.Errorf("resolve subject attributes: %w", err)
	}
	resourceAttrs, err := p.resolver.ResolveResource(ctx, req.ResourceType, req.ResourceID)
	if err != nil {
		return nil, fmt.Errorf("resolve resource attributes: %w", err)
	}
	envAttrs := ResolveEnvironment(req.ClientIP, req.UserAgent, req.MFAVerified)

	allAttrs := make(map[string]any)
	mergeAttrs(allAttrs, subjectAttrs)
	mergeAttrs(allAttrs, resourceAttrs)
	mergeAttrs(allAttrs, envAttrs)

	// Step 2: Target matching + condition evaluation.
	var decisions []Decision
	var matched []string
	var applicableObligations []ObligationSpec

	for _, policy := range policies {
		// Target: check action and resource patterns.
		if !matchAny(policy.Actions, req.Action) {
			continue
		}

		// Condition: check if attributes satisfy policy conditions.
		if !evaluateConditions(policy.Conditions, allAttrs) {
			continue
		}

		decisions = append(decisions, policy.Effect)
		matched = append(matched, policy.Name)
		applicableObligations = append(applicableObligations, policy.Obligations...)

		// Short-circuit: if combining is deny-overrides and we found a deny, stop.
		if policy.Effect == DecisionDeny {
			break
		}
	}

	// Step 3: Combine decisions.
	finalDecision := p.combining(decisions)
	if finalDecision == DecisionNotApp {
		finalDecision = DecisionDeny // fail-closed default
	}

	// Step 4: Execute obligations (only if permitted).
	if finalDecision == DecisionPermit && len(applicableObligations) > 0 {
		if err := p.obligations.ExecuteAll(ctx, applicableObligations); err != nil {
			// Obligation failure blocks access.
			return &EvalResponse{
				Decision:   DecisionDeny,
				Reason:     fmt.Sprintf("obligation failed: %v", err),
				EvalTimeMS: time.Since(start).Milliseconds(),
			}, nil
		}
	}

	return &EvalResponse{
		Decision:        finalDecision,
		Reason:          reasonFor(finalDecision, matched),
		MatchedPolicies: matched,
		Obligations:     applicableObligations,
		EvalTimeMS:      time.Since(start).Milliseconds(),
	}, nil
}

func mergeAttrs(dst, src map[string]any) {
	for k, v := range src {
		dst[k] = v
	}
}

func matchAny(patterns []string, s string) bool {
	for _, p := range patterns {
		if p == "*" || p == s {
			return true
		}
	}
	return false
}

func evaluateConditions(conditions map[string]any, attrs map[string]any) bool {
	for operator, condMap := range conditions {
		conds, ok := condMap.(map[string]any)
		if !ok {
			continue
		}
		for attr, expected := range conds {
			actual, exists := attrs[attr]
			if !exists {
				return false
			}
			if !evalOperator(operator, expected, actual) {
				return false
			}
		}
	}
	return true
}

func evalOperator(op string, expected, actual any) bool {
	es := fmt.Sprintf("%v", expected)
	as := fmt.Sprintf("%v", actual)
	switch op {
	case "StringEquals":
		return es == as
	case "StringNotEquals":
		return es != as
	case "Bool":
		return es == as
	default:
		return false
	}
}

func reasonFor(d Decision, matched []string) string {
	switch d {
	case DecisionPermit:
		return fmt.Sprintf("permitted by %d policy(ies)", len(matched))
	case DecisionDeny:
		return fmt.Sprintf("denied (matched: %v)", matched)
	default:
		return "no applicable policy"
	}
}
```

---

## 7. Policy Lifecycle Management

### Lifecycle Stages

```
Author --> Review --> Test --> Deploy --> Monitor --> Retire
  ^                                           |
  |___________ Feedback Loop __________________|
```

1. **Author**: Policy written in DSL (JSON, ALFA, or Rego) by security team.
2. **Review**: Peer review by security architects. Static analysis for conflicting policies.
3. **Test**: Dry-run against sample requests. Verify expected decisions for known scenarios.
4. **Deploy**: Versioned rollout. Blue/green or canary deployment.
5. **Monitor**: Track decision logs, deny rates, evaluation latency. Alert on anomalies.
6. **Retire**: Deprecate old policy versions. Archive with metadata for audit.

### Version Control and A/B Testing

Policies should be versioned with semantic versioning (v1.0.0). A/B testing involves deploying a new policy version to a subset of traffic (e.g., 10% of requests) and comparing deny rates against the control version. If the new version shows unexpected denials, automatic rollback is triggered.

### Go: Policy Lifecycle Manager

```go
package abac

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// PolicyVersion tracks a versioned policy with lifecycle metadata.
type PolicyVersion struct {
	ID          uuid.UUID
	PolicyID    uuid.UUID
	Version     string // semantic version, e.g. "1.2.0"
	Document    PolicyDoc
	Status      PolicyStatus
	CreatedAt   time.Time
	CreatedBy   string
	ActivatedAt *time.Time
	RetiredAt   *time.Time
}

type PolicyStatus string

const (
	StatusDraft     PolicyStatus = "draft"
	StatusReviewing PolicyStatus = "reviewing"
	StatusActive    PolicyStatus = "active"
	StatusRetired   PolicyStatus = "retired"
)

// LifecycleManager manages policy versioning, deployment, and rollback.
type LifecycleManager struct {
	mu       sync.RWMutex
	versions map[uuid.UUID][]PolicyVersion // policyID -> versions (newest first)
	active   map[uuid.UUID]uuid.UUID       // policyID -> active version ID
}

func NewLifecycleManager() *LifecycleManager {
	return &LifecycleManager{
		versions: make(map[uuid.UUID][]PolicyVersion),
		active:   make(map[uuid.UUID]uuid.UUID),
	}
}

// CreateVersion creates a new draft version of a policy.
func (m *LifecycleManager) CreateVersion(policyID uuid.UUID, doc PolicyDoc, author string) *PolicyVersion {
	v := PolicyVersion{
		ID:        uuid.New(),
		PolicyID:  policyID,
		Version:   nextVersion(m.versions[policyID]),
		Document:  doc,
		Status:    StatusDraft,
		CreatedAt: time.Now().UTC(),
		CreatedBy: author,
	}
	m.mu.Lock()
	m.versions[policyID] = append([]PolicyVersion{v}, m.versions[policyID]...)
	m.mu.Unlock()
	return &v
}

// Activate promotes a version to active status and retires the previous active version.
func (m *LifecycleManager) Activate(policyID, versionID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Find the version.
	var target *PolicyVersion
	for i := range m.versions[policyID] {
		if m.versions[policyID][i].ID == versionID {
			target = &m.versions[policyID][i]
			break
		}
	}
	if target == nil {
		return fmt.Errorf("version %s not found", versionID)
	}

	// Retire previous active version.
	if oldID, ok := m.active[policyID]; ok {
		for i := range m.versions[policyID] {
			if m.versions[policyID][i].ID == oldID {
				now := time.Now().UTC()
				m.versions[policyID][i].Status = StatusRetired
				m.versions[policyID][i].RetiredAt = &now
				break
			}
		}
	}

	// Activate new version.
	now := time.Now().UTC()
	target.Status = StatusActive
	target.ActivatedAt = &now
	m.active[policyID] = versionID
	return nil
}

// Rollback reverts to the previous active version.
func (m *LifecycleManager) Rollback(policyID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	versions := m.versions[policyID]
	// Find the most recent retired version.
	for _, v := range versions {
		if v.Status == StatusRetired {
			now := time.Now().UTC()
			// Retire current active.
			if currID, ok := m.active[policyID]; ok {
				for i := range versions {
					if versions[i].ID == currID {
						versions[i].Status = StatusRetired
						versions[i].RetiredAt = &now
					}
				}
			}
			// Activate the retired one.
			// Note: v is a copy from range, so we need to find and modify in the slice.
			for i := range versions {
				if versions[i].ID == v.ID {
					versions[i].Status = StatusActive
					versions[i].ActivatedAt = &now
					m.active[policyID] = versions[i].ID
					return nil
				}
			}
		}
	}
	return fmt.Errorf("no previous version to rollback to")
}

// GetActive returns the currently active policy version.
func (m *LifecycleManager) GetActive(policyID uuid.UUID) (*PolicyVersion, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	versionID, ok := m.active[policyID]
	if !ok {
		return nil, fmt.Errorf("no active version for policy %s", policyID)
	}
	for _, v := range m.versions[policyID] {
		if v.ID == versionID {
			return &v, nil
		}
	}
	return nil, fmt.Errorf("active version not found")
}

// AuditTrail returns all version history for a policy.
func (m *LifecycleManager) AuditTrail(policyID uuid.UUID) []PolicyVersion {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]PolicyVersion, len(m.versions[policyID]))
	copy(result, m.versions[policyID])
	return result
}

// ExportVersion serializes a policy version for backup or transfer.
func (m *LifecycleManager) ExportVersion(versionID uuid.UUID) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, versions := range m.versions {
		for _, v := range versions {
			if v.ID == versionID {
				return json.MarshalIndent(v, "", "  ")
			}
		}
	}
	return nil, fmt.Errorf("version %s not found", versionID)
}

func nextVersion(versions []PolicyVersion) string {
	if len(versions) == 0 {
		return "1.0.0"
	}
	return fmt.Sprintf("1.0.%d", len(versions))
}
```

---

## 8. GGID Policy Engine Audit

### Architecture Overview

GGID's policy service (`services/policy/`) implements a combined RBAC + ABAC engine with the following structure:

| Component | File | Responsibility |
|-----------|------|---------------|
| Domain models | `internal/domain/models.go` | Role, Permission, Policy, CheckRequest/Result |
| Evaluator | `internal/service/evaluator.go` | RBAC + ABAC evaluation with deny-overrides |
| Policy service | `internal/service/policy_service.go` | ABAC policy CRUD + attachment |
| Role service | `internal/service/role_service.go` | RBAC role CRUD, hierarchy, user-role assignment |
| HTTP server | `internal/server/http.go` | REST API for console (1647 lines, 20+ endpoints) |
| Repository | `internal/repository/` | PostgreSQL persistence for roles, permissions, policies |

### Evaluation Flow (Current)

The `Evaluator.Check()` method in `evaluator.go` implements this flow:

1. Resolve user's roles (including ancestor chain via `GetAncestorChain`)
2. Collect permissions from all resolved roles — RBAC allow check (exact match + wildcard `*`)
3. Fetch ABAC policies attached to the user and their roles
4. Evaluate ABAC conditions using AWS IAM-style operators
5. Combine: deny-overrides (any deny policy wins), then RBAC allow, then ABAC allow
6. Default: deny if no explicit allow

### RBAC Capabilities (Implemented)

- Role CRUD with tenant isolation (RLS)
- Role hierarchy with parent-child inheritance and cycle detection
- Permission management (resource_type + action)
- Role-permission grants with optional ABAC conditions on the junction
- User-role assignment with scope (global, org, department, team, resource)
- Effective permissions calculation (recursive hierarchy walk)
- Bulk role assignment
- Wildcard action matching (`iam:users:*`)

### ABAC Capabilities (Implemented)

- AWS IAM-style policy model: Effect (allow/deny), Actions, Resources, Conditions, Priority
- Policy attachment to principals (user, role, group)
- Condition operators: `StringEquals`, `StringNotEquals`, `StringEqualsIgnoreCase`, `StringLike`, `StringNotLike`, `NumericEquals`, `NumericNotEquals`, `NumericLessThan`, `NumericLessThanEquals`, `NumericGreaterThan`, `NumericGreaterThanEquals`, `Bool`, `DateLessThan`, `DateGreaterThan`, `IpAddress`, `NotIpAddress`
- Glob-style resource matching (`arn:ggid:iam::tenant:user/*`)
- Fail-closed behavior: policies with conditions match only if request provides matching attributes
- Decision logging with in-memory ring buffer (1000 entries) and callback hook
- Compliance templates (PCI-DSS, HIPAA, SOC 2, GDPR)
- Policy dry-run, diff, and analysis endpoints
- Time conditions endpoint
- Policy import/export

### What is Missing

The following capabilities are not present in the current GGID policy engine:

| Capability | Status | Impact |
|-----------|--------|--------|
| Attribute resolver (PIP) | Missing | No automatic attribute enrichment — callers must supply all conditions manually |
| Environment attributes | Missing | No automatic time/IP/device context — callers pass as conditions |
| Configurable combining algorithm | Missing | Fixed deny-overrides, no per-policy-set configuration |
| Obligations and advice | Missing | No mandatory post-decision actions |
| Policy versioning | Partial (endpoints exist but no persistence) | `versions` endpoint returns canned data |
| Attribute-based policy target | Missing | Target is action + resource only, no subject attribute targeting |
| Policy caching | Missing | All policies fetched from DB on every check |
| OPA/Rego integration | Missing | No external policy engine support |

---

## 9. Gap Analysis and Recommendations

### Priority Action Items

**Action 1: Implement Attribute Resolver (PIP)**
- **Effort**: 3-5 days
- **Impact**: Eliminates manual condition passing, enables automatic attribute enrichment from JWT, identity service, and environment context
- **Approach**: Create `internal/service/attribute_resolver.go` with Redis caching (60s TTL). Wire into `Evaluator.Check()` to merge resolved attributes with request conditions before policy evaluation.

**Action 2: Add Configurable Combining Algorithms**
- **Effort**: 2-3 days
- **Impact**: Enables policy-set-level control over decision merging — critical for compliance scenarios where first-applicable or permit-unless-deny is required
- **Approach**: Add `CombiningAlgorithm` field to a new `PolicySet` domain model. Implement the five XACML algorithms. Default to deny-overrides for backward compatibility.

**Action 3: Implement Obligation Framework**
- **Effort**: 3-4 days
- **Impact**: Compliance with HIPAA, GDPR, and SOC 2 requirements for mandatory access logging and notification
- **Approach**: Add `Obligations []ObligationSpec` to `domain.Policy`. Create `ObligationExecutor` with pluggable handlers. Wire into the evaluation pipeline — obligations are executed only on permit, and failure blocks access.

**Action 4: Add Policy Versioning with Persistence**
- **Effort**: 4-5 days
- **Impact**: Enables safe policy rollouts, A/B testing, and rollback — essential for production governance
- **Approach**: Create `policy_versions` table (policy_id, version, document JSONB, status, timestamps). Wire the existing `/versions` endpoint to real data. Implement activate/rollback/deprecate operations.

**Action 5: Add Policy Evaluation Caching**
- **Effort**: 2-3 days
- **Impact**: Reduces evaluation latency from ~5-10ms (DB + policy load) to ~0.5ms (cache hit) for repeated requests
- **Approach**: Cache evaluation results in Redis keyed by `(user_id, tenant_id, action, resource_type, conditions_hash)`. TTL of 15-30 seconds. Invalidate on policy create/update/delete for the affected tenant.

### Summary

GGID's policy engine has a solid RBAC + ABAC foundation with the AWS IAM-style condition model, role hierarchy, and decision logging. The primary gaps are in the PIP layer (attribute resolution), configurable combining, obligations, and versioning persistence — all of which are well-scoped additions that build on the existing architecture without requiring a rewrite.
