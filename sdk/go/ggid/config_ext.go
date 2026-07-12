package ggid

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// --- Configuration Management SDK ---
// Covers config endpoints implemented by backend service teams.

// --- User Lifecycle Config ---

// UserLifecycleConfig manages user lifecycle automation settings.
type UserLifecycleConfig struct {
	AutoDeactivateAfterDays int                      `json:"auto_deactivate_after_days"`
	DormantDetectionRules   map[string]interface{}   `json:"dormant_detection_rules"`
	StageTransitions        []map[string]interface{} `json:"stage_transitions"`
	NotificationBefore      int                      `json:"notification_before"`
	PerRoleOverride         map[string]interface{}   `json:"per_role_override"`
}

// GetUserLifecycleConfig retrieves user lifecycle configuration.
func (c *Client) GetUserLifecycleConfig(ctx context.Context, token string) (*UserLifecycleConfig, error) {
	data, err := c.do(ctx, http.MethodGet, "/api/v1/identity/user-lifecycle/config", nil, token)
	if err != nil {
		return nil, err
	}
	var cfg UserLifecycleConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal user lifecycle config: %w", err)
	}
	return &cfg, nil
}

// UpdateUserLifecycleConfig updates user lifecycle configuration.
func (c *Client) UpdateUserLifecycleConfig(ctx context.Context, token string, cfg *UserLifecycleConfig) error {
	_, err := c.do(ctx, http.MethodPut, "/api/v1/identity/user-lifecycle/config", cfg, token)
	return err
}

// --- ABAC Condition Config ---

// ABACConditionConfig manages ABAC attribute-based access control settings.
type ABACConditionConfig struct {
	AttributeSources     []string                 `json:"attribute_sources"`
	OperatorsPerType     map[string]interface{}   `json:"operators_per_type"`
	ConditionTemplates   []map[string]interface{} `json:"condition_templates"`
	EvaluationCacheTTL   int                      `json:"evaluation_cache_ttl"`
	DefaultDeny          bool                     `json:"default_deny"`
}

// GetABACConditionConfig retrieves ABAC condition configuration.
func (c *Client) GetABACConditionConfig(ctx context.Context, token string) (*ABACConditionConfig, error) {
	data, err := c.do(ctx, http.MethodGet, "/api/v1/policy/abac/condition-config", nil, token)
	if err != nil {
		return nil, err
	}
	var cfg ABACConditionConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal abac condition config: %w", err)
	}
	return &cfg, nil
}

// UpdateABACConditionConfig updates ABAC condition configuration.
func (c *Client) UpdateABACConditionConfig(ctx context.Context, token string, cfg *ABACConditionConfig) error {
	_, err := c.do(ctx, http.MethodPut, "/api/v1/policy/abac/condition-config", cfg, token)
	return err
}

// --- SCIM Provisioning Config ---

// SCIMProvisioningConfig manages SCIM user provisioning settings.
type SCIMProvisioningConfig struct {
	Endpoint             string                 `json:"endpoint"`
	MappingRules         map[string]interface{} `json:"mapping_rules"`
	Triggers             []string               `json:"triggers"`
	SyncDirection        string                 `json:"sync_direction"`
	DeprovisionOnDisable bool                   `json:"deprovision_on_disable"`
}

// GetSCIMProvisioningConfig retrieves SCIM provisioning configuration.
func (c *Client) GetSCIMProvisioningConfig(ctx context.Context, token string) (*SCIMProvisioningConfig, error) {
	data, err := c.do(ctx, http.MethodGet, "/api/v1/identity/scim/provisioning-config", nil, token)
	if err != nil {
		return nil, err
	}
	var cfg SCIMProvisioningConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal scim provisioning config: %w", err)
	}
	return &cfg, nil
}

// UpdateSCIMProvisioningConfig updates SCIM provisioning configuration.
func (c *Client) UpdateSCIMProvisioningConfig(ctx context.Context, token string, cfg *SCIMProvisioningConfig) error {
	_, err := c.do(ctx, http.MethodPut, "/api/v1/identity/scim/provisioning-config", cfg, token)
	return err
}

// --- Audit Export Schedule Config ---

// ExportJob defines a scheduled audit export job.
type ExportJob struct {
	Name       string                 `json:"name"`
	Cron       string                 `json:"cron"`
	Format     string                 `json:"format"`
	Filters    map[string]interface{} `json:"filters,omitempty"`
	Retention  int                    `json:"retention"`
	Destination string                `json:"destination"`
}

// AuditExportConfig manages scheduled audit data exports.
type AuditExportConfig struct {
	Jobs          []ExportJob            `json:"jobs"`
	MaxConcurrent int                    `json:"max_concurrent"`
	RetryPolicy   map[string]interface{} `json:"retry_policy"`
	Notification  map[string]interface{} `json:"notification"`
}

// GetAuditExportConfig retrieves audit export schedule configuration.
func (c *Client) GetAuditExportConfig(ctx context.Context, token string) (*AuditExportConfig, error) {
	data, err := c.do(ctx, http.MethodGet, "/api/v1/audit/export/schedule-config", nil, token)
	if err != nil {
		return nil, err
	}
	var cfg AuditExportConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal audit export config: %w", err)
	}
	return &cfg, nil
}

// UpdateAuditExportConfig updates audit export schedule configuration.
func (c *Client) UpdateAuditExportConfig(ctx context.Context, token string, cfg *AuditExportConfig) error {
	_, err := c.do(ctx, http.MethodPut, "/api/v1/audit/export/schedule-config", cfg, token)
	return err
}

// --- Token Rotation Config ---

// ClientTokenRotation defines per-client token rotation settings.
type ClientTokenRotation struct {
	ClientID         string `json:"client_id"`
	RotationInterval int    `json:"rotation_interval"`
	MaxAge           int    `json:"max_age"`
	NotifyBefore     int    `json:"notify_before"`
	AutoRotate       bool   `json:"auto_rotate"`
	GracePeriod      int    `json:"grace_period"`
}

// TokenRotationConfig manages OAuth token rotation policies.
type TokenRotationConfig struct {
	PerClient      []ClientTokenRotation   `json:"per_client"`
	GlobalDefaults map[string]interface{}  `json:"global_defaults"`
}

// GetTokenRotationConfig retrieves token rotation configuration.
func (c *Client) GetTokenRotationConfig(ctx context.Context, token string) (*TokenRotationConfig, error) {
	data, err := c.do(ctx, http.MethodGet, "/api/v1/oauth/token-rotation/config", nil, token)
	if err != nil {
		return nil, err
	}
	var cfg TokenRotationConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal token rotation config: %w", err)
	}
	return &cfg, nil
}

// UpdateTokenRotationConfig updates token rotation configuration.
func (c *Client) UpdateTokenRotationConfig(ctx context.Context, token string, cfg *TokenRotationConfig) error {
	_, err := c.do(ctx, http.MethodPut, "/api/v1/oauth/token-rotation/config", cfg, token)
	return err
}

// --- Risk Scoring Config ---

// RiskScoringConfig manages identity risk scoring engine settings.
type RiskScoringConfig struct {
	RiskFactors       map[string]float64      `json:"risk_factors"`
	Weights           map[string]float64      `json:"weights"`
	Thresholds        map[string]int          `json:"thresholds"`
	ActionMapping     map[string]string       `json:"action_mapping"`
	AdaptiveLearning  bool                    `json:"adaptive_learning"`
}

// GetRiskScoringConfig retrieves risk scoring configuration.
func (c *Client) GetRiskScoringConfig(ctx context.Context, token string) (*RiskScoringConfig, error) {
	data, err := c.do(ctx, http.MethodGet, "/api/v1/identity/risk-scoring/config", nil, token)
	if err != nil {
		return nil, err
	}
	var cfg RiskScoringConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal risk scoring config: %w", err)
	}
	return &cfg, nil
}

// UpdateRiskScoringConfig updates risk scoring configuration.
func (c *Client) UpdateRiskScoringConfig(ctx context.Context, token string, cfg *RiskScoringConfig) error {
	_, err := c.do(ctx, http.MethodPut, "/api/v1/identity/risk-scoring/config", cfg, token)
	return err
}

// --- SOD Conflict Detection Config ---

// SODConflictConfig manages Separation of Duties conflict detection.
type SODConflictConfig struct {
	Rules             map[string]interface{} `json:"rules"`
	SensitivityLevels map[string]string      `json:"sensitivity_levels"`
	AutoRemediate     map[string]interface{} `json:"auto_remediate"`
	ExceptionWorkflow map[string]interface{} `json:"exception_workflow"`
}

// GetSODConflictConfig retrieves SOD conflict detection configuration.
func (c *Client) GetSODConflictConfig(ctx context.Context, token string) (*SODConflictConfig, error) {
	data, err := c.do(ctx, http.MethodGet, "/api/v1/policy/sod/conflict-detection-config", nil, token)
	if err != nil {
		return nil, err
	}
	var cfg SODConflictConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal sod conflict config: %w", err)
	}
	return &cfg, nil
}

// UpdateSODConflictConfig updates SOD conflict detection configuration.
func (c *Client) UpdateSODConflictConfig(ctx context.Context, token string, cfg *SODConflictConfig) error {
	_, err := c.do(ctx, http.MethodPut, "/api/v1/policy/sod/conflict-detection-config", cfg, token)
	return err
}

// --- SIEM Forwarder Config ---

// SIEMDestination defines a SIEM forwarding target.
type SIEMDestination struct {
	SIEMType      string `json:"siem_type"`
	Protocol      string `json:"protocol"`
	Host          string `json:"host"`
	Auth          string `json:"auth,omitempty"`
	Format        string `json:"format"`
	BatchSize     int    `json:"batch_size"`
	FlushInterval int    `json:"flush_interval"`
}

// SIEMForwarderConfig manages SIEM log forwarding settings.
type SIEMForwarderConfig struct {
	Destinations       []SIEMDestination      `json:"destinations"`
	FilterRules        map[string]interface{} `json:"filter_rules"`
	RetryPolicy        map[string]interface{} `json:"retry_policy"`
	CircuitBreaker     map[string]interface{} `json:"circuit_breaker"`
	HealthCheckInterval int                   `json:"health_check_interval"`
}

// GetSIEMForwarderConfig retrieves SIEM forwarder configuration.
func (c *Client) GetSIEMForwarderConfig(ctx context.Context, token string) (*SIEMForwarderConfig, error) {
	data, err := c.do(ctx, http.MethodGet, "/api/v1/audit/siem/forwarder-config", nil, token)
	if err != nil {
		return nil, err
	}
	var cfg SIEMForwarderConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal siem forwarder config: %w", err)
	}
	return &cfg, nil
}

// UpdateSIEMForwarderConfig updates SIEM forwarder configuration.
func (c *Client) UpdateSIEMForwarderConfig(ctx context.Context, token string, cfg *SIEMForwarderConfig) error {
	_, err := c.do(ctx, http.MethodPut, "/api/v1/audit/siem/forwarder-config", cfg, token)
	return err
}
