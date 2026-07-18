package httpserver

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/ggid/ggid/services/audit/internal/repository"
	"github.com/google/uuid"
)

// ControlStatus represents the compliance state of a single control.
const (
	StatusPass = "pass"
	StatusWarn = "warn"
	StatusFail = "fail"
)

// CCMResult represents the evaluation result for a single compliance control.
type CCMResult struct {
	ControlID    string         `json:"control_id"`
	ControlName  string         `json:"control_name"`
	Category     string         `json:"category"`
	Status       string         `json:"status"` // pass/warn/fail
	MetricValue  float64        `json:"metric_value"`
	Threshold    float64        `json:"threshold"`
	ThresholdDir string         `json:"threshold_dir"` // "lt" or "gt" — metric should be lt/gt threshold
	Details      map[string]any `json:"details,omitempty"`
	CheckedAt    time.Time      `json:"checked_at"`
}

// CCMEngine evaluates compliance controls periodically.
type CCMEngine struct {
	mu       sync.RWMutex
	results  map[string]*CCMResult  // control_id → latest result
	history  []CCMResult             // all results chronologically
	repo     *repository.CCMRepository // PG persistence (nil = in-memory only)
}

func NewCCMEngine() *CCMEngine {
	return &CCMEngine{
		results: make(map[string]*CCMResult),
	}
}

// SetRepository injects a PostgreSQL-backed CCM repository for persistence.
func (e *CCMEngine) SetRepository(repo *repository.CCMRepository) {
	e.repo = repo
}

// RunAll evaluates all 15 compliance controls and stores results.
func (e *CCMEngine) RunAll() []*CCMResult {
	now := time.Now()
	results := []*CCMResult{
		e.evalControl("mfa_coverage", "MFA Enrollment Coverage", "identity", 85.0, 92.0, "lt", now,
			map[string]any{"description": "Percentage of active users with MFA enrolled"}),
		e.evalControl("password_policy_compliance", "Password Policy Compliance", "identity", 5.0, 2.0, "gt", now,
			map[string]any{"description": "Percentage of users whose password violates current policy"}),
		e.evalControl("expired_permissions", "Expired Permission Reviews (>90 days)", "access", 12.0, 5.0, "gt", now,
			map[string]any{"description": "Number of permissions not reviewed in 90+ days"}),
		e.evalControl("dormant_accounts", "Dormant Accounts (90+ days inactive)", "identity", 8.0, 3.0, "gt", now,
			map[string]any{"description": "Number of accounts inactive for 90+ days"}),
		e.evalControl("orphan_accounts", "Orphan Accounts (HR terminated, still active)", "identity", 2.0, 0.0, "gt", now,
			map[string]any{"description": "Number of accounts with terminated HR status but active access"}),
		e.evalControl("privilege_creep", "Privilege Creep Detection", "access", 3.0, 1.0, "gt", now,
			map[string]any{"description": "Number of users with accumulated excess privileges"}),
		e.evalControl("service_account_rotation", "Service Account Key Rotation (>90 days)", "nhi", 15.0, 5.0, "gt", now,
			map[string]any{"description": "Number of API keys/service accounts not rotated in 90+ days"}),
		e.evalControl("audit_chain_integrity", "Audit Hash Chain Integrity", "audit", 0.0, 0.0, "gt", now,
			map[string]any{"description": "Number of broken hash chain links detected"}),
		e.evalControl("break_glass_review", "Break-Glass Usage Review (30 days)", "access", 4.0, 2.0, "gt", now,
			map[string]any{"description": "Number of break-glass activations in last 30 days"}),
		e.evalControl("admin_account_count", "Privileged Account Count", "access", 25.0, 15.0, "gt", now,
			map[string]any{"description": "Number of users with admin-level roles"}),
		e.evalControl("jit_elevation_active", "Active JIT Elevations", "access", 3.0, 5.0, "gt", now,
			map[string]any{"description": "Number of currently active JIT elevations"}),
		e.evalControl("unused_app_access", "Unused Application Access (60 days)", "access", 10.0, 5.0, "gt", now,
			map[string]any{"description": "Number of app entitlements unused for 60+ days"}),
		e.evalControl("group_ownership_freshness", "Group Ownership Review (>180 days)", "access", 6.0, 3.0, "gt", now,
			map[string]any{"description": "Number of groups with ownership not reviewed in 180+ days"}),
		e.evalControl("session_timeout_compliance", "Session Timeout Compliance", "session", 2.0, 1.0, "gt", now,
			map[string]any{"description": "Number of sessions exceeding max timeout policy"}),
		e.evalControl("risk_based_auth_coverage", "Risk-Based Authentication Coverage", "identity", 70.0, 90.0, "lt", now,
			map[string]any{"description": "Percentage of high-risk access paths with risk-based auth"}),
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	for _, r := range results {
		e.results[r.ControlID] = r
		e.history = append(e.history, *r)
	}

	// Persist to PostgreSQL if repo is configured.
	if e.repo != nil {
		records := make([]*repository.CCMResultRecord, 0, len(results))
		for _, r := range results {
			detailsJSON, _ := json.Marshal(r.Details)
			records = append(records, &repository.CCMResultRecord{
				ID:           uuid.Nil, // let DB generate
				ControlID:    r.ControlID,
				ControlName:  r.ControlName,
				Category:     r.Category,
				Status:       r.Status,
				MetricValue:  r.MetricValue,
				Threshold:    r.Threshold,
				ThresholdDir: r.ThresholdDir,
				Details:      detailsJSON,
				CheckedAt:    now,
			})
		}
		_ = e.repo.StoreBatch(context.Background(), records)
	}

	return results
}

// evalControl creates a CCMResult with simulated metric values.
// In production, these would query real data sources.
func (e *CCMEngine) evalControl(id, name, category string, metric, threshold float64, dir string, now time.Time, details map[string]any) *CCMResult {
	status := evalStatus(metric, threshold, dir)
	return &CCMResult{
		ControlID:    id,
		ControlName:  name,
		Category:     category,
		Status:       status,
		MetricValue:  metric,
		Threshold:    threshold,
		ThresholdDir: dir,
		Details:      details,
		CheckedAt:    now,
	}
}

// evalStatus determines pass/warn/fail based on metric vs threshold.
func evalStatus(metric, threshold float64, dir string) string {
	if dir == "lt" {
		// Metric should be >= threshold. Below = bad.
		if metric < threshold*0.7 {
			return StatusFail
		}
		if metric < threshold {
			return StatusWarn
		}
		return StatusPass
	}
	// dir == "gt": metric should be <= threshold. Above = bad.
	if metric > threshold*2 {
		return StatusFail
	}
	if metric > threshold && threshold >= 0 {
		return StatusWarn
	}
	return StatusPass
}

// GetResults returns the latest result for each control.
func (e *CCMEngine) GetResults() []*CCMResult {
	e.mu.RLock()
	defer e.mu.RUnlock()

	results := make([]*CCMResult, 0, len(e.results))
	for _, r := range e.results {
		results = append(results, r)
	}
	return results
}

// GetHistory returns historical results, optionally filtered by control_id.
func (e *CCMEngine) GetHistory(controlID string, limit int) []CCMResult {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if limit <= 0 || limit > 500 {
		limit = 100
	}

	var result []CCMResult
	for i := len(e.history) - 1; i >= 0 && len(result) < limit; i-- {
		if controlID == "" || e.history[i].ControlID == controlID {
			result = append(result, e.history[i])
		}
	}
	return result
}

// GetSummary returns a high-level compliance dashboard summary.
func (e *CCMEngine) GetSummary() map[string]any {
	e.mu.RLock()
	defer e.mu.RUnlock()

	pass, warn, fail := 0, 0, 0
	for _, r := range e.results {
		switch r.Status {
		case StatusPass:
			pass++
		case StatusWarn:
			warn++
		case StatusFail:
			fail++
		}
	}
	total := pass + warn + fail
	score := 100.0
	if total > 0 {
		score = float64(pass) / float64(total) * 100
	}

	return map[string]any{
		"total_controls":   total,
		"pass":             pass,
		"warn":             warn,
		"fail":             fail,
		"compliance_score": score,
		"last_run":         e.getLastRunTime(),
	}
}

func (e *CCMEngine) getLastRunTime() *time.Time {
	var latest *time.Time
	for _, r := range e.results {
		if latest == nil || r.CheckedAt.After(*latest) {
			t := r.CheckedAt
			latest = &t
		}
	}
	return latest
}

// MarshalJSON for CCMResult ensures details is never null.
func (r CCMResult) MarshalJSON() ([]byte, error) {
	type Alias CCMResult
	if r.Details == nil {
		r.Details = map[string]any{}
	}
	return json.Marshal(&struct{ Alias }{Alias(r)})
}
