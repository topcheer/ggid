package posture

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostureCheck defines a single device posture check.
type PostureCheck struct {
	Name     string `json:"name"`      // os_version, disk_encrypted, jailbroken, screen_lock, mdm_enrolled, cert_valid
	Required bool   `json:"required"`  // if true, failure = non-compliant
	Weight   int    `json:"weight"`    // contribution to score (0-100 total)
	MinValue string `json:"min_value"` // for version comparisons
}

// PosturePolicy defines per-tenant device posture requirements.
type PosturePolicy struct {
	TenantID string         `json:"tenant_id"`
	Checks   []PostureCheck `json:"checks"`
	MinScore int            `json:"min_score"` // minimum score to be compliant (default 70)
	Action   string         `json:"action"`    // allow | step_up | block (when below min_score)
}

// PostureInput is the raw device telemetry for evaluation.
type PostureInput struct {
	DeviceID       string `json:"device_id"`
	OSVersion      string `json:"os_version"`
	DiskEncrypted  bool   `json:"disk_encrypted"`
	Jailbroken     bool   `json:"jailbroken"`
	ScreenLock     bool   `json:"screen_lock"`
	MDMEnrolled    bool   `json:"mdm_enrolled"`
	CertValid      bool   `json:"cert_valid"`
}

// PostureResult is the evaluation outcome.
type PostureResult struct {
	DeviceID   string           `json:"device_id"`
	Score      int              `json:"score"`       // 0-100
	Compliant  bool             `json:"compliant"`
	Action     string           `json:"action"`      // allow | step_up | block
	Checks     []CheckResult    `json:"checks"`
	EvaluatedAt time.Time       `json:"evaluated_at"`
}

// CheckResult is the outcome of a single posture check.
type CheckResult struct {
	Name    string `json:"name"`
	Passed  bool   `json:"passed"`
	Detail  string `json:"detail"`
}

// DefaultPosturePolicy returns a production-ready default policy.
func DefaultPosturePolicy(tenantID string) PosturePolicy {
	return PosturePolicy{
		TenantID: tenantID,
		MinScore: 70,
		Action:   "block",
		Checks: []PostureCheck{
			{Name: "os_version", Required: false, Weight: 15, MinValue: "13.0"},
			{Name: "disk_encrypted", Required: true, Weight: 25},
			{Name: "jailbroken", Required: true, Weight: 30},
			{Name: "screen_lock", Required: false, Weight: 10},
			{Name: "mdm_enrolled", Required: false, Weight: 10},
			{Name: "cert_valid", Required: true, Weight: 10},
		},
	}
}

// Engine evaluates device posture against policies.
type Engine struct {
	pool    *pgxpool.Pool
	mu      sync.RWMutex
	policies map[string]PosturePolicy // tenantID → policy
}

// NewEngine creates a new posture evaluation engine.
func NewEngine(pool *pgxpool.Pool) *Engine {
	return &Engine{
		pool:     pool,
		policies: make(map[string]PosturePolicy),
	}
}

// EnsureSchema creates posture tables.
func (e *Engine) EnsureSchema(ctx context.Context) error {
	if e.pool == nil {
		return nil
	}
	_, err := e.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS device_posture_policies (
			tenant_id TEXT PRIMARY KEY,
			checks JSONB NOT NULL DEFAULT '[]',
			min_score INT NOT NULL DEFAULT 70,
			action TEXT NOT NULL DEFAULT 'block',
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE TABLE IF NOT EXISTS device_posture_scores (
			device_id TEXT NOT NULL,
			tenant_id TEXT NOT NULL,
			score INT NOT NULL,
			checks JSONB NOT NULL DEFAULT '[]',
			evaluated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			PRIMARY KEY (tenant_id, device_id, evaluated_at)
		);
		CREATE INDEX IF NOT EXISTS idx_posture_scores_device ON device_posture_scores(device_id, evaluated_at DESC);
	`)
	return err
}

// SetPolicy configures a posture policy for a tenant.
func (e *Engine) SetPolicy(tenantID string, policy PosturePolicy) {
	e.mu.Lock()
	defer e.mu.Unlock()
	policy.TenantID = tenantID
	e.policies[tenantID] = policy
}

// GetPolicy returns the posture policy for a tenant.
func (e *Engine) GetPolicy(tenantID string) PosturePolicy {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if p, ok := e.policies[tenantID]; ok {
		return p
	}
	return DefaultPosturePolicy(tenantID)
}

// Evaluate computes the posture score for a device.
func (e *Engine) Evaluate(tenantID string, input PostureInput) PostureResult {
	policy := e.GetPolicy(tenantID)
	result := PostureResult{
		DeviceID:    input.DeviceID,
		EvaluatedAt: time.Now(),
	}

	totalWeight := 0
	earnedWeight := 0
	compliant := true

	for _, check := range policy.Checks {
		passed, detail := evaluateCheck(check, input)
		result.Checks = append(result.Checks, CheckResult{
			Name: check.Name, Passed: passed, Detail: detail,
		})

		totalWeight += check.Weight
		if passed {
			earnedWeight += check.Weight
		}

		// Required checks that fail = non-compliant regardless of score.
		if check.Required && !passed {
			compliant = false
		}
	}

	// Compute score.
	if totalWeight > 0 {
		result.Score = earnedWeight * 100 / totalWeight
	} else {
		result.Score = 100
	}

	// Compliance: must have score >= minScore AND all required checks passed.
	result.Compliant = compliant && result.Score >= policy.MinScore

	// Action.
	if result.Compliant {
		result.Action = "allow"
	} else {
		result.Action = policy.Action
	}

	return result
}

// evaluateCheck runs a single posture check against the input.
func evaluateCheck(check PostureCheck, input PostureInput) (bool, string) {
	switch check.Name {
	case "os_version":
		if check.MinValue == "" {
			return true, "no minimum version set"
		}
		if compareVersions(input.OSVersion, check.MinValue) >= 0 {
			return true, fmt.Sprintf("OS %s >= %s", input.OSVersion, check.MinValue)
		}
		return false, fmt.Sprintf("OS %s < minimum %s", input.OSVersion, check.MinValue)

	case "disk_encrypted":
		if input.DiskEncrypted {
			return true, "disk encryption enabled"
		}
		return false, "disk not encrypted"

	case "jailbroken":
		// This check PASSES when device is NOT jailbroken.
		if !input.Jailbroken {
			return true, "device not jailbroken"
		}
		return false, "device is jailbroken/rooted"

	case "screen_lock":
		if input.ScreenLock {
			return true, "screen lock enabled"
		}
		return false, "screen lock disabled"

	case "mdm_enrolled":
		if input.MDMEnrolled {
			return true, "MDM enrolled"
		}
		return false, "not MDM enrolled"

	case "cert_valid":
		if input.CertValid {
			return true, "device certificate valid"
		}
		return false, "device certificate invalid or missing"

	default:
		return true, "unknown check — skipping"
	}
}

// compareVersions compares semantic version strings (returns -1, 0, 1).
func compareVersions(a, b string) int {
	// Simple version comparison: split by "." and compare numerically.
	ai := parseVersionParts(a)
	bi := parseVersionParts(b)
	maxLen := len(ai)
	if len(bi) > maxLen {
		maxLen = len(bi)
	}
	for i := 0; i < maxLen; i++ {
		var av, bv int
		if i < len(ai) {
			av = ai[i]
		}
		if i < len(bi) {
			bv = bi[i]
		}
		if av < bv {
			return -1
		}
		if av > bv {
			return 1
		}
	}
	return 0
}

func parseVersionParts(v string) []int {
	var parts []int
	current := 0
	for _, c := range v {
		if c >= '0' && c <= '9' {
			current = current*10 + int(c-'0')
		} else if c == '.' {
			parts = append(parts, current)
			current = 0
		}
	}
	parts = append(parts, current)
	return parts
}

// PersistResult stores the posture result in PG.
func (e *Engine) PersistResult(ctx context.Context, tenantID string, result PostureResult) error {
	if e.pool == nil {
		return nil
	}
	checksJSON, _ := json.Marshal(result.Checks)
	_, err := e.pool.Exec(ctx,
		`INSERT INTO device_posture_scores (device_id, tenant_id, score, checks, evaluated_at) VALUES ($1,$2,$3,$4,$5)`,
		result.DeviceID, tenantID, result.Score, checksJSON, result.EvaluatedAt)
	return err
}

// GetLatestScore returns the most recent posture score for a device.
func (e *Engine) GetLatestScore(ctx context.Context, tenantID, deviceID string) (*PostureResult, error) {
	if e.pool == nil {
		return nil, nil
	}
	var result PostureResult
	var checksJSON []byte
	err := e.pool.QueryRow(ctx,
		`SELECT device_id, score, checks, evaluated_at FROM device_posture_scores WHERE tenant_id=$1 AND device_id=$2 ORDER BY evaluated_at DESC LIMIT 1`,
		tenantID, deviceID).Scan(&result.DeviceID, &result.Score, &checksJSON, &result.EvaluatedAt)
	if err != nil {
		return nil, nil
	}
	_ = json.Unmarshal(checksJSON, &result.Checks)
	result.Compliant = result.Score >= e.GetPolicy(tenantID).MinScore
	return &result, nil
}
