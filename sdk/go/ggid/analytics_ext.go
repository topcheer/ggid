package ggid

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// --- Policy Analytics ---

// PolicyConflict represents a detected conflict between two policies.
type PolicyConflict struct {
	PolicyA    string `json:"policy_a"`
	PolicyB    string `json:"policy_b"`
	Rule       string `json:"rule"`
	OverlapType string `json:"overlap_type"` // contradictory, duplicate, subset
	Severity   string `json:"severity"`
	Detail     string `json:"detail"`
}

// PolicyConflictsResult holds the result of a conflict detection scan.
type PolicyConflictsResult struct {
	ConflictPairs  []PolicyConflict    `json:"conflict_pairs"`
	BySeverity     map[string]int      `json:"by_severity"`
	TotalConflicts int                 `json:"total_conflicts"`
	CheckedAt      string              `json:"checked_at"`
}

// DetectPolicyConflicts scans for overlapping or conflicting policies.
func (c *Client) DetectPolicyConflicts(ctx context.Context, token string, policyIDs []string) (*PolicyConflictsResult, error) {
	body := map[string]any{"policy_ids": policyIDs}
	data, err := c.do(ctx, http.MethodPost, "/api/v1/policy/conflicts", body, token)
	if err != nil {
		return nil, err
	}
	var result PolicyConflictsResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("unmarshal policy conflicts: %w", err)
	}
	return &result, nil
}

// BlastRadiusSummary summarizes the impact of a policy change.
type BlastRadiusSummary struct {
	TotalUsersAffected     int    `json:"total_users_affected"`
	TotalRolesAffected     int    `json:"total_roles_affected"`
	TotalResourcesChanged  int    `json:"total_resources_changed"`
	TotalCascading         int    `json:"total_cascading"`
	BreakingChanges       bool   `json:"breaking_changes"`
	RiskLevel             string `json:"risk_level"`
	RecommendedAction     string `json:"recommended_action"`
}

// BlastRadiusResult holds the full blast radius analysis for a policy.
type BlastRadiusResult struct {
	PolicyID           string              `json:"policy_id"`
	AffectedUsers      []map[string]any    `json:"affected_users"`
	AffectedRoles      []map[string]any    `json:"affected_roles"`
	AffectedResources  []map[string]any    `json:"affected_resources"`
	CascadingPolicies  []map[string]any    `json:"cascading_policies"`
	Summary            BlastRadiusSummary  `json:"summary"`
	PreviewMode        string              `json:"preview_mode"`
	AnalyzedAt         string              `json:"analyzed_at"`
}

// GetPolicyBlastRadius analyzes the impact of changing a policy.
func (c *Client) GetPolicyBlastRadius(ctx context.Context, token, policyID string) (*BlastRadiusResult, error) {
	data, err := c.do(ctx, http.MethodGet, "/api/v1/policy/blast-radius/"+policyID, nil, token)
	if err != nil {
		return nil, err
	}
	var result BlastRadiusResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("unmarshal blast radius: %w", err)
	}
	return &result, nil
}

// --- Coverage Matrix ---

// CoverageMatrix represents the policy coverage across subjects and resources.
type CoverageMatrix struct {
	Grid                  [][]map[string]any  `json:"grid"`
	Subjects              []string            `json:"subjects"`
	Resources             []string            `json:"resources"`
	UncoveredCombinations []map[string]any    `json:"uncovered_combinations"`
	RedundantPolicies     []map[string]any    `json:"redundant_policies"`
	GapsCount             int                 `json:"gaps_count"`
}

// GetCoverageMatrix retrieves the policy coverage matrix.
func (c *Client) GetCoverageMatrix(ctx context.Context, token string) (*CoverageMatrix, error) {
	data, err := c.do(ctx, http.MethodGet, "/api/v1/policy/coverage-matrix", nil, token)
	if err != nil {
		return nil, err
	}
	var result CoverageMatrix
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("unmarshal coverage matrix: %w", err)
	}
	return &result, nil
}

// --- Policy Exceptions ---

// PolicyException represents an exception to a policy rule.
type PolicyException struct {
	ID                string            `json:"id"`
	PolicyID          string            `json:"policy_id"`
	ExceptionReason   string            `json:"exception_reason"`
	GrantedTo         string            `json:"granted_to"`
	ExpiresAt         string            `json:"expires_at,omitempty"`
	Approver          string            `json:"approver"`
	RiskOverrideLevel string            `json:"risk_override_level"`
	AuditTrail        []map[string]any  `json:"audit_trail"`
}

// ListPolicyExceptions retrieves all policy exceptions.
func (c *Client) ListPolicyExceptions(ctx context.Context, token string) ([]PolicyException, error) {
	data, err := c.do(ctx, http.MethodGet, "/api/v1/policy/exceptions", nil, token)
	if err != nil {
		return nil, err
	}
	var result struct {
		Exceptions []PolicyException `json:"exceptions"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("unmarshal policy exceptions: %w", err)
	}
	return result.Exceptions, nil
}

// CreatePolicyException creates a new policy exception.
func (c *Client) CreatePolicyException(ctx context.Context, token string, exc *PolicyException) error {
	_, err := c.do(ctx, http.MethodPost, "/api/v1/policy/exceptions", exc, token)
	return err
}

// --- Access Graph ---

// AccessGraph represents the permission graph for a subject.
type AccessGraph struct {
	SubjectID             string            `json:"subject_id"`
	DirectPermissions     []string          `json:"direct_permissions"`
	InheritedPermissions  []string          `json:"inherited_permissions"`
	ViaGroups             []string          `json:"via_groups"`
	ViaRoles              []string          `json:"via_roles"`
	EffectivePermissions  []string          `json:"effective_permissions"`
	GraphDepth            int               `json:"graph_depth"`
}

// GetAccessGraph retrieves the access graph for a subject.
func (c *Client) GetAccessGraph(ctx context.Context, token, subjectID string) (*AccessGraph, error) {
	data, err := c.do(ctx, http.MethodGet, "/api/v1/policy/access-graph/"+subjectID, nil, token)
	if err != nil {
		return nil, err
	}
	var result AccessGraph
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("unmarshal access graph: %w", err)
	}
	return &result, nil
}

// --- Batch Policy Simulation ---

// BatchSimulationRequest defines a batch policy evaluation.
type BatchSimulationRequest struct {
	Subjects  []string `json:"subjects"`
	Resources []string `json:"resources"`
	Actions   []string `json:"actions"`
}

// BatchSimulationResult holds the result of a batch policy simulation.
type BatchSimulationResult struct {
	Results        []map[string]any `json:"results"`
	AggregateStats map[string]int   `json:"aggregate_stats"`
	MismatchCount  int              `json:"mismatch_count"`
}

// SimulatePolicyBatch evaluates policies against a batch of subjects/resources/actions.
func (c *Client) SimulatePolicyBatch(ctx context.Context, token string, req *BatchSimulationRequest) (*BatchSimulationResult, error) {
	data, err := c.do(ctx, http.MethodPost, "/api/v1/policy/simulate/batch", req, token)
	if err != nil {
		return nil, err
	}
	var result BatchSimulationResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("unmarshal batch simulation: %w", err)
	}
	return &result, nil
}

// --- Identity Analytics ---

// RoleMiningResult holds role mining analysis data.
type RoleMiningResult struct {
	UnusedPermissions      []map[string]any  `json:"unused_permissions"`
	OverAssignedRoles      []map[string]any  `json:"over_assigned_roles"`
	SuggestedConsolidation []map[string]any  `json:"suggested_consolidation"`
	EntitlementCreepScore  float64           `json:"entitlement_creep_score"`
	TopRedundantRoles      []map[string]any  `json:"top_redundant_roles"`
}

// GetRoleMining retrieves role mining analysis.
func (c *Client) GetRoleMining(ctx context.Context, token string) (*RoleMiningResult, error) {
	data, err := c.do(ctx, http.MethodGet, "/api/v1/identity/role-mining", nil, token)
	if err != nil {
		return nil, err
	}
	var result RoleMiningResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("unmarshal role mining: %w", err)
	}
	return &result, nil
}

// SAMLSPHealth holds SAML service provider health check data.
type SAMLSPHealth struct {
	MetadataURLValid    bool     `json:"metadata_url_valid"`
	CertExpiryDays      int      `json:"cert_expiry_days"`
	ResponseTest        string   `json:"response_test"`
	AssertionConsumerURL string  `json:"assertion_consumer_url"`
	SLOStatus           string   `json:"slo_status"`
	IDPConnectionStatus string   `json:"idp_connection_status"`
	LastSync            string   `json:"last_sync"`
	Errors              []string `json:"errors"`
}

// GetSAMLSPHealth checks the health of the SAML service provider.
func (c *Client) GetSAMLSPHealth(ctx context.Context, token string) (*SAMLSPHealth, error) {
	data, err := c.do(ctx, http.MethodGet, "/api/v1/identity/saml/sp-health", nil, token)
	if err != nil {
		return nil, err
	}
	var result SAMLSPHealth
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("unmarshal SAML SP health: %w", err)
	}
	return &result, nil
}

// SCIMSyncHealth holds SCIM synchronization health data.
type SCIMSyncHealth struct {
	EndpointURL         string  `json:"endpoint_url"`
	LastSyncAt          string  `json:"last_sync_at"`
	ProvisioningErrors  []map[string]any `json:"provisioning_errors"`
	UserCountSynced     int     `json:"user_count_synced"`
	UserCountPending    int     `json:"user_count_pending"`
	UserCountFailed     int     `json:"user_count_failed"`
	RateLimits          map[string]int `json:"rate_limits"`
	ThroughputPerMin    int     `json:"throughput_per_min"`
}

// GetSCIMSyncHealth checks SCIM synchronization health.
func (c *Client) GetSCIMSyncHealth(ctx context.Context, token string) (*SCIMSyncHealth, error) {
	data, err := c.do(ctx, http.MethodGet, "/api/v1/identity/scim/sync-health", nil, token)
	if err != nil {
		return nil, err
	}
	var result SCIMSyncHealth
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("unmarshal SCIM sync health: %w", err)
	}
	return &result, nil
}

// --- Audit Analytics ---

// ForensicsTimeline holds hash chain forensics data.
type ForensicsTimeline struct {
	HashChainVerification string           `json:"hash_chain_verification"`
	TamperEvidence        []map[string]any `json:"tamper_evidence"`
	InsertionGaps         []map[string]any `json:"insertion_gaps"`
	ReorderDetected       bool             `json:"reorder_detected"`
	IntegrityScore        float64          `json:"integrity_score"`
}

// GetForensicsTimeline retrieves audit chain forensics data.
func (c *Client) GetForensicsTimeline(ctx context.Context, token string) (*ForensicsTimeline, error) {
	data, err := c.do(ctx, http.MethodGet, "/api/v1/audit/forensics/timeline", nil, token)
	if err != nil {
		return nil, err
	}
	var result ForensicsTimeline
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("unmarshal forensics timeline: %w", err)
	}
	return &result, nil
}

// FrameworkCoverage holds compliance framework coverage data.
type FrameworkCoverage struct {
	Frameworks []map[string]any `json:"frameworks"`
	Overall    map[string]any   `json:"overall"`
}

// GetFrameworkCoverage retrieves compliance framework coverage.
func (c *Client) GetFrameworkCoverage(ctx context.Context, token string) (*FrameworkCoverage, error) {
	data, err := c.do(ctx, http.MethodGet, "/api/v1/audit/framework-coverage", nil, token)
	if err != nil {
		return nil, err
	}
	var result FrameworkCoverage
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("unmarshal framework coverage: %w", err)
	}
	return &result, nil
}

// --- OAuth Analytics ---

// AuthorizeFlowStats holds OAuth authorization flow statistics.
type AuthorizeFlowStats struct {
	TotalAttempts        int              `json:"total_attempts"`
	ConsentRate          float64          `json:"consent_rate"`
	AbandonmentAtStep    map[string]int   `json:"abandonment_at_step"`
	AvgDurationMs        float64          `json:"avg_duration_ms"`
	TopClients           []map[string]any `json:"top_clients"`
	RedirectURIErrors    int              `json:"redirect_uri_errors"`
	PKCEAdoptionPct      float64          `json:"pkce_adoption_pct"`
}

// GetAuthorizeFlowStats retrieves OAuth authorize flow analytics.
func (c *Client) GetAuthorizeFlowStats(ctx context.Context, token string) (*AuthorizeFlowStats, error) {
	data, err := c.do(ctx, http.MethodGet, "/api/v1/oauth/stats/authorize-flow", nil, token)
	if err != nil {
		return nil, err
	}
	var result AuthorizeFlowStats
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("unmarshal authorize flow stats: %w", err)
	}
	return &result, nil
}

// TokenBindingStats holds token binding statistics.
type TokenBindingStats struct {
	BoundTokens     int              `json:"bound_tokens"`
	UnboundTokens   int              `json:"unbound_tokens"`
	BindingMethods  map[string]int   `json:"binding_methods"`
	CompliancePct   float64          `json:"compliance_pct"`
	ByClient        []map[string]any `json:"by_client"`
}

// GetTokenBindingStats retrieves token binding statistics.
func (c *Client) GetTokenBindingStats(ctx context.Context, token string) (*TokenBindingStats, error) {
	data, err := c.do(ctx, http.MethodGet, "/api/v1/oauth/stats/token-binding", nil, token)
	if err != nil {
		return nil, err
	}
	var result TokenBindingStats
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("unmarshal token binding stats: %w", err)
	}
	return &result, nil
}

// --- Auth Analytics ---

// PasswordlessStats holds passwordless authentication statistics.
type PasswordlessStats struct {
	MethodDistribution map[string]int   `json:"method_distribution"`
	SuccessRate        float64          `json:"success_rate"`
	AvgCompletionMs    float64          `json:"avg_completion_time_ms"`
	AbandonmentRate    float64          `json:"abandonment_rate"`
	ByDeviceType       map[string]int   `json:"by_device_type"`
}

// GetPasswordlessStats retrieves passwordless authentication statistics.
func (c *Client) GetPasswordlessStats(ctx context.Context, token string) (*PasswordlessStats, error) {
	data, err := c.do(ctx, http.MethodGet, "/api/v1/auth/passwordless/stats", nil, token)
	if err != nil {
		return nil, err
	}
	var result PasswordlessStats
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("unmarshal passwordless stats: %w", err)
	}
	return &result, nil
}

// HijackTimeline holds session hijack detection data.
type HijackTimeline struct {
	UserID             string           `json:"user_id"`
	Events             []map[string]any `json:"events"`
	ConfidenceScore    float64          `json:"confidence_score"`
	RecommendedActions []string         `json:"recommended_actions"`
}

// GetHijackTimeline retrieves session hijack detection timeline for a user.
func (c *Client) GetHijackTimeline(ctx context.Context, token, userID string) (*HijackTimeline, error) {
	data, err := c.do(ctx, http.MethodGet, "/api/v1/auth/hijack/timeline?user_id="+userID, nil, token)
	if err != nil {
		return nil, err
	}
	var result HijackTimeline
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("unmarshal hijack timeline: %w", err)
	}
	return &result, nil
}

// --- Org Analytics ---

// TeamInsights holds team collaboration analysis data.
type TeamInsights struct {
	CohesionScore         float64          `json:"cohesion_score"`
	CollaborationPatterns []map[string]any `json:"collaboration_patterns"`
	SiloDetection         []map[string]any `json:"silo_detection"`
	CrossTeamDeps         []map[string]any `json:"cross_team_deps"`
	ExpertiseDistribution map[string]int   `json:"expertise_distribution"`
	RiskOfAttrition       []map[string]any `json:"risk_of_attrition"`
}

// GetTeamInsights retrieves team collaboration insights.
func (c *Client) GetTeamInsights(ctx context.Context, token string) (*TeamInsights, error) {
	data, err := c.do(ctx, http.MethodGet, "/api/v1/org/team-insights", nil, token)
	if err != nil {
		return nil, err
	}
	var result TeamInsights
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("unmarshal team insights: %w", err)
	}
	return &result, nil
}

// ReportingStructure holds the org reporting hierarchy.
type ReportingStructure struct {
	Tree            []map[string]any `json:"tree"`
	SpanOfControl   map[string]int   `json:"span_of_control"`
	Layers          int              `json:"layers"`
	DotRepresentation string         `json:"dotty_representation"`
	OrphanManagers  []string         `json:"orphan_managers"`
	CircularReporting []string       `json:"circular_reporting"`
}

// GetReportingStructure retrieves the organization reporting structure.
func (c *Client) GetReportingStructure(ctx context.Context, token string) (*ReportingStructure, error) {
	data, err := c.do(ctx, http.MethodGet, "/api/v1/org/reporting-structure", nil, token)
	if err != nil {
		return nil, err
	}
	var result ReportingStructure
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("unmarshal reporting structure: %w", err)
	}
	return &result, nil
}
