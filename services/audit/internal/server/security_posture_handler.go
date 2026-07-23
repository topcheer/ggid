package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// NIST 800-207 Zero-Trust posture dimensions.
// Each dimension is scored 0-100 (higher = better posture).

// GET /api/v1/audit/security-posture — zero-trust posture score (NIST 800-207)
func (s *HTTPServer) handleSecurityPosture(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	tenantID := tenantIDFromRequest(r)

	dimensions := s.calculatePostureDimensions(tenantID)
	overall := weightedScore(dimensions)
	grade := scoreToGrade(overall)
	recs := generateRecommendations(dimensions)
	findings := generateFindings(dimensions)

	// Persist to history table for trend tracking.
	if s.pool != nil {
		s.savePostureHistory(tenantID, overall, dimensions, grade, findings, recs)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"score":          overall,
		"grade":          grade,
		"model":          "NIST 800-207",
		"dimensions":     dimensions,
		"findings":       findings,
		"recommendations": recs,
		"evaluated_at":   time.Now().UTC().Format(time.RFC3339),
	})
}

// GET /api/v1/audit/security-posture/history
func (s *HTTPServer) handleSecurityPostureHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	tenantID := tenantIDFromRequest(r)
	if s.pool == nil {
		writeJSON(w, http.StatusOK, map[string]any{"history": []any{}})
		return
	}

	limit := 30
	rows, err := s.pool.Query(r.Context(), `
		SELECT overall_score, identity_score, device_score, network_score,
		       data_score, workload_score, grade, evaluated_at
		FROM zt_posture_history
		WHERE tenant_id = $1 OR $1 = '00000000-0000-0000-0000-000000000000'
		ORDER BY evaluated_at DESC LIMIT $2`, tenantID, limit)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"history": []any{}})
		return
	}
	defer rows.Close()

	type entry struct {
		Overall   int       `json:"overall"`
		Identity  int       `json:"identity"`
		Device    int       `json:"device"`
		Network   int       `json:"network"`
		Data      int       `json:"data"`
		Workload  int       `json:"workload"`
		Grade     string    `json:"grade"`
		Evaluated time.Time `json:"evaluated_at"`
	}
	var history []entry
	for rows.Next() {
		var e entry
		rows.Scan(&e.Overall, &e.Identity, &e.Device, &e.Network, &e.Data, &e.Workload, &e.Grade, &e.Evaluated)
		history = append(history, e)
	}
	writeJSON(w, http.StatusOK, map[string]any{"history": history})
}

type postureDimensions struct {
	Identity int `json:"identity_score"`
	Device   int `json:"device_score"`
	Network  int `json:"network_score"`
	Data     int `json:"data_score"`
	Workload int `json:"workload_score"`
}

func (s *HTTPServer) calculatePostureDimensions(tenantID uuid.UUID) postureDimensions {
	d := postureDimensions{Identity: 100, Device: 100, Network: 100, Data: 100, Workload: 100}

	if s.pool == nil {
		return d
	}
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	// --- Identity dimension ---
	var totalUsers, mfaUsers, inactiveUsers int
	s.pool.QueryRow(ctx, `SELECT count(*) FROM users WHERE deleted_at IS NULL`).Scan(&totalUsers)
	s.pool.QueryRow(ctx, `SELECT count(DISTINCT user_id) FROM mfa_devices WHERE enabled=true`).Scan(&mfaUsers)
	s.pool.QueryRow(ctx, `SELECT count(*) FROM users WHERE deleted_at IS NULL AND last_login_at < now() - interval '90 days' OR last_login_at IS NULL`).Scan(&inactiveUsers)

	if totalUsers > 0 {
		mfaPct := mfaUsers * 100 / totalUsers
		inactivePct := inactiveUsers * 100 / totalUsers
		d.Identity = clampScore(100 - (100-mfaPct)*2/3 - inactivePct/2)
	}

	// --- Device dimension ---
	var activeSessions, revokedToday int
	s.pool.QueryRow(ctx, `SELECT count(*) FROM sessions WHERE revoked_at IS NULL AND expires_at > now()`).Scan(&activeSessions)
	s.pool.QueryRow(ctx, `SELECT count(*) FROM sessions WHERE revoked_at IS NOT NULL AND revoked_at > now() - interval '24 hours'`).Scan(&revokedToday)
	d.Device = clampScore(100 - revokedToday*3)

	// --- Network dimension ---
	var failedLogins, uniqueIPs int
	s.pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action LIKE '%login%failed%' AND created_at > now() - interval '24 hours'`).Scan(&failedLogins)
	s.pool.QueryRow(ctx, `SELECT count(DISTINCT ip_address) FROM audit_events WHERE created_at > now() - interval '24 hours' AND ip_address IS NOT NULL`).Scan(&uniqueIPs)
	if failedLogins > 10 {
		d.Network = clampScore(100 - (failedLogins-10)*2)
	}
	if uniqueIPs > 50 {
		d.Network -= 5
	}

	// --- Data dimension ---
	var tamperIssues int
	s.pool.QueryRow(ctx, `SELECT count(*) FROM audit_incidents WHERE data->>'type' = 'tamper_detected' AND data->>'status' = 'open'`).Scan(&tamperIssues)
	d.Data = clampScore(100 - tamperIssues*10)

	// --- Workload dimension ---
	var openIncidents, highSeverityThreats int
	s.pool.QueryRow(ctx, `SELECT count(*) FROM audit_incidents WHERE data->>'status' IN ('open','investigating')`).Scan(&openIncidents)
	s.pool.QueryRow(ctx, `SELECT count(*) FROM audit_incidents WHERE data->>'severity' = 'critical' AND data->>'status' = 'open'`).Scan(&highSeverityThreats)
	d.Workload = clampScore(100 - openIncidents*5 - highSeverityThreats*10)

	return d
}

func weightedScore(d postureDimensions) int {
	// NIST 800-207 weights: Identity 30%, Device 20%, Network 20%, Data 15%, Workload 15%
	return (d.Identity*30 + d.Device*20 + d.Network*20 + d.Data*15 + d.Workload*15) / 100
}

func scoreToGrade(score int) string {
	switch {
	case score >= 90: return "A"
	case score >= 80: return "B"
	case score >= 70: return "C"
	case score >= 60: return "D"
	default: return "F"
	}
}

type finding struct {
	Dimension string `json:"dimension"`
	Score     int    `json:"score"`
	Status    string `json:"status"`
	Detail    string `json:"detail"`
}

func generateFindings(d postureDimensions) []finding {
	findings := []finding{}
	for _, dim := range []struct{ name string; score int }{
		{"identity", d.Identity}, {"device", d.Device},
		{"network", d.Network}, {"data", d.Data}, {"workload", d.Workload},
	} {
		status := "healthy"
		if dim.score < 50 { status = "critical" } else if dim.score < 70 { status = "warning" }
		findings = append(findings, finding{
			Dimension: dim.name, Score: dim.score, Status: status,
			Detail: dim.name + " posture score: " + itoa(dim.score) + "/100",
		})
	}
	return findings
}

type recommendation struct {
	Dimension string `json:"dimension"`
	Priority  string `json:"priority"`
	Action    string `json:"action"`
}

func generateRecommendations(d postureDimensions) []recommendation {
	var recs []recommendation
	if d.Identity < 90 {
		recs = append(recs, recommendation{"identity", "high", "Increase MFA adoption — enforce for all users"})
	}
	if d.Device < 70 {
		recs = append(recs, recommendation{"device", "medium", "Review recent session revocations — investigate compromised sessions"})
	}
	if d.Network < 70 {
		recs = append(recs, recommendation{"network", "high", "High failed login rate — consider rate limiting or IP blocking"})
	}
	if d.Data < 70 {
		recs = append(recs, recommendation{"data", "critical", "Audit tamper incidents detected — investigate immediately"})
	}
	if d.Workload < 70 {
		recs = append(recs, recommendation{"workload", "high", "Open security incidents — prioritize remediation"})
	}
	return recs
}

func (s *HTTPServer) savePostureHistory(tenantID uuid.UUID, overall int, d postureDimensions, grade string, findings []finding, recs []recommendation) {
	findingsJSON, _ := json.Marshal(findings)
	recsJSON, _ := json.Marshal(recs)
	// Throttle: only save if last entry is > 1 hour ago
	var lastEval time.Time
	ctx2, cancel2 := contextWithTimeout(3 * time.Second)
	defer cancel2()
	_ = s.pool.QueryRow(ctx2,
		`SELECT evaluated_at FROM zt_posture_history WHERE tenant_id = $1 ORDER BY evaluated_at DESC LIMIT 1`,
		tenantID).Scan(&lastEval)
	if time.Since(lastEval) < time.Hour {
		return // already have a recent entry
	}
	ctx3, cancel3 := contextWithTimeout(3 * time.Second)
	defer cancel3()
	s.pool.Exec(ctx3,
		`INSERT INTO zt_posture_history (tenant_id, overall_score, identity_score, device_score, network_score, data_score, workload_score, grade, findings, recommendations)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		tenantID, overall, d.Identity, d.Device, d.Network, d.Data, d.Workload, grade, string(findingsJSON), string(recsJSON))
}

func contextWithTimeout(d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), d)
}
func clampScore(v int) int { if v < 0 { return 0 }; if v > 100 { return 100 }; return v }
func itoa(i int) string { return string(rune('0'+i/10)) + string(rune('0'+i%10)) }
