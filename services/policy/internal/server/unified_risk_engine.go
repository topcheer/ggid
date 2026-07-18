package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// --- Signal Registry (20 signals, 5 categories) ---

type SignalCategory string

const (
	SigDevice   SignalCategory = "device"
	SigGeo      SignalCategory = "geo"
	SigNetwork  SignalCategory = "network"
	SigBehavior SignalCategory = "behavior"
	SigSession  SignalCategory = "session"
)

type SignalDef struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Category SignalCategory `json:"category"`
	Weight   float64        `json:"weight"`   // 0.0-1.0
	Default  float64        `json:"default"`  // default value 0.0 (no risk)
}

var signalRegistry = []SignalDef{
	// Device (6)
	{"device_posture", "Device Posture Score", SigDevice, 0.15, 0},
	{"device_managed", "Managed Device", SigDevice, 0.10, 0},
	{"device_encrypted", "Disk Encryption", SigDevice, 0.08, 0},
	{"device_jailbreak", "Jailbreak/Root Detected", SigDevice, 0.20, 0},
	{"device_compliant_os", "OS Compliance", SigDevice, 0.07, 0},
	{"device_trust_score", "Device Trust Score", SigDevice, 0.10, 0},
	// Geo (5)
	{"geo_impossible_travel", "Impossible Travel", SigGeo, 0.25, 0},
	{"geo_high_risk_country", "High-Risk Country", SigGeo, 0.15, 0},
	{"geo_new_location", "New Login Location", SigGeo, 0.08, 0},
	{"geo_tor_vpn", "TOR/VPN/Proxy Detected", SigGeo, 0.12, 0},
	{"geo_geofence_violation", "Geofence Violation", SigGeo, 0.10, 0},
	// Network (5)
	{"net_threat_intel", "Threat Intel Match", SigNetwork, 0.20, 0},
	{"net_ip_reputation", "IP Reputation Score", SigNetwork, 0.10, 0},
	{"net_new_asn", "New ASN", SigNetwork, 0.05, 0},
	{"net_ddos_source", "DDoS Source IP", SigNetwork, 0.15, 0},
	{"net_port_scan", "Port Scan Detected", SigNetwork, 0.08, 0},
	// Behavior (6)
	{"beh_ueba_anomaly", "UEBA Anomaly Score", SigBehavior, 0.18, 0},
	{"beh_off_hours", "Off-Hours Access", SigBehavior, 0.06, 0},
	{"beh_bulk_action", "Bulk Action Detected", SigBehavior, 0.12, 0},
	{"beh_privilege_escalation", "Privilege Escalation Attempt", SigBehavior, 0.20, 0},
	{"beh_mfa_fatigue", "MFA Fatigue Pattern", SigBehavior, 0.15, 0},
	{"beh_new_device_user", "First-Time Device for User", SigBehavior, 0.08, 0},
	// Session (4)
	{"sess_concurrent", "Concurrent Sessions", SigSession, 0.08, 0},
	{"sess_token_anomaly", "Token Usage Anomaly", SigSession, 0.10, 0},
	{"sess_session_age", "Session Age Exceeded", SigSession, 0.05, 0},
	{"sess_session_hijack", "Session Hijack Indicator", SigSession, 0.18, 0},
}

// --- Types ---

type RiskPolicy struct {
	TenantID         uuid.UUID      `json:"tenant_id"`
	AllowThreshold   int            `json:"allow_threshold"`
	StepUpThreshold  int            `json:"step_up_threshold"`
	StrongThreshold  int            `json:"strong_threshold"`
	Weights          map[string]float64 `json:"weights"`
}

type RiskEvaluationRequest struct {
	UserID    string         `json:"user_id"`
	SessionID string         `json:"session_id"`
	TenantID  string         `json:"tenant_id"`
	Context   map[string]any `json:"context"`
}

type RiskEvaluationResponse struct {
	Score       int            `json:"score"`
	Level       string         `json:"level"`       // low, medium, high, critical
	Decision    string         `json:"decision"`    // allow, step_up, step_up_strong, block
	Signals     []SignalResult `json:"signals"`
	EvaluatedAt time.Time      `json:"evaluated_at"`
	EvaluationID string        `json:"evaluation_id"`
}

type SignalResult struct {
	ID     string  `json:"id"`
	Value  float64 `json:"value"`
	Weight float64 `json:"weight"`
}

// --- Repo ---

type riskRepo struct {
	pool *pgxpool.Pool
}

func NewRiskRepo(pool *pgxpool.Pool) *riskRepo {
	return &riskRepo{pool: pool}
}

func (r *riskRepo) EnsureSchema(ctx context.Context) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS risk_policies (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL,
			allow_threshold INT DEFAULT 30, step_up_threshold INT DEFAULT 60,
			strong_threshold INT DEFAULT 85, weights JSONB DEFAULT '{}',
			enabled BOOLEAN DEFAULT TRUE,
			created_at TIMESTAMPTZ DEFAULT now(), updated_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_risk_policy_tenant ON risk_policies(tenant_id);
		CREATE TABLE IF NOT EXISTS risk_scores (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID, user_id TEXT NOT NULL, session_id TEXT DEFAULT '',
			score INT DEFAULT 0, level TEXT DEFAULT 'low', decision TEXT DEFAULT 'allow',
			signals JSONB DEFAULT '[]', evaluated_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_risk_scores_user ON risk_scores(user_id, evaluated_at DESC);
	`)
	return err
}

func (r *riskRepo) GetPolicy(ctx context.Context, tenantID uuid.UUID) (*RiskPolicy, error) {
	if r.pool == nil {
		return defaultPolicy(tenantID), nil
	}
	var p RiskPolicy
	var weightsJSON []byte
	err := r.pool.QueryRow(ctx, `SELECT allow_threshold, step_up_threshold, strong_threshold, weights FROM risk_policies WHERE tenant_id=$1 AND enabled=TRUE`, tenantID,
	).Scan(&p.AllowThreshold, &p.StepUpThreshold, &p.StrongThreshold, &weightsJSON)
	if err != nil {
		return defaultPolicy(tenantID), nil
	}
	p.TenantID = tenantID
	json.Unmarshal(weightsJSON, &p.Weights)
	return &p, nil
}

func (r *riskRepo) UpsertPolicy(ctx context.Context, p *RiskPolicy) error {
	if r.pool == nil {
		return nil
	}
	weightsJSON, _ := json.Marshal(p.Weights)
	_, err := r.pool.Exec(ctx, `INSERT INTO risk_policies (tenant_id,allow_threshold,step_up_threshold,strong_threshold,weights) VALUES ($1,$2,$3,$4,$5) ON CONFLICT (tenant_id) DO UPDATE SET allow_threshold=EXCLUDED.allow_threshold, step_up_threshold=EXCLUDED.step_up_threshold, strong_threshold=EXCLUDED.strong_threshold, weights=EXCLUDED.weights, updated_at=now()`,
		p.TenantID, p.AllowThreshold, p.StepUpThreshold, p.StrongThreshold, weightsJSON)
	return err
}

func (r *riskRepo) LogScore(ctx context.Context, resp *RiskEvaluationResponse, req *RiskEvaluationRequest) {
	if r.pool == nil || resp == nil {
		return
	}
	signalsJSON, _ := json.Marshal(resp.Signals)
	tenantID, _ := uuid.Parse(req.TenantID)
	r.pool.Exec(ctx, `INSERT INTO risk_scores (tenant_id,user_id,session_id,score,level,decision,signals) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		tenantID, req.UserID, req.SessionID, resp.Score, resp.Level, resp.Decision, signalsJSON)
}

func (r *riskRepo) GetLatestScore(ctx context.Context, userID string) (map[string]any, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("not found")
	}
	var score int
	var level, decision string
	var evaluatedAt time.Time
	var signalsJSON []byte
	err := r.pool.QueryRow(ctx, `SELECT score, level, decision, signals, evaluated_at FROM risk_scores WHERE user_id=$1 ORDER BY evaluated_at DESC LIMIT 1`, userID,
	).Scan(&score, &level, &decision, &signalsJSON, &evaluatedAt)
	if err != nil {
		return nil, fmt.Errorf("not found")
	}
	return map[string]any{"score": score, "level": level, "decision": decision, "evaluated_at": evaluatedAt}, nil
}

func defaultPolicy(tenantID uuid.UUID) *RiskPolicy {
	return &RiskPolicy{TenantID: tenantID, AllowThreshold: 30, StepUpThreshold: 60, StrongThreshold: 85, Weights: map[string]float64{}}
}

// --- Core Evaluation ---

func (s *HTTPServer) EvaluateRisk(ctx context.Context, req *RiskEvaluationRequest) *RiskEvaluationResponse {
	tenantID, _ := uuid.Parse(req.TenantID)
	policy, _ := s.riskRepo.GetPolicy(ctx, tenantID)

	// Collect signal values from context.
	signals := make([]SignalResult, 0, len(signalRegistry))
	totalScore := 0.0
	for _, sig := range signalRegistry {
		value := sig.Default
		if v, ok := req.Context[sig.ID]; ok {
			switch val := v.(type) {
			case float64:
				value = val
			case int:
				value = float64(val)
			case bool:
				if val { value = 1.0 }
			}
		}
		weight := sig.Weight
		if w, ok := policy.Weights[sig.ID]; ok && w > 0 {
			weight = w
		}
		contribution := value * weight
		totalScore += contribution
		signals = append(signals, SignalResult{ID: sig.ID, Value: value, Weight: weight})
	}

	// Clamp score 0-100.
	finalScore := int(totalScore * 100)
	if finalScore < 0 { finalScore = 0 }
	if finalScore > 100 { finalScore = 100 }

	// Determine level + decision.
	level, decision := "low", "allow"
	switch {
	case finalScore > policy.StrongThreshold:
		level, decision = "critical", "block"
	case finalScore > policy.StepUpThreshold:
		level, decision = "high", "step_up_strong"
	case finalScore > policy.AllowThreshold:
		level, decision = "medium", "step_up"
	}

	resp := &RiskEvaluationResponse{
		Score: finalScore, Level: level, Decision: decision,
		Signals: signals, EvaluatedAt: time.Now().UTC(),
		EvaluationID: uuid.New().String(),
	}

	// Audit log (async).
	if s.riskRepo != nil {
		go s.riskRepo.LogScore(context.Background(), resp, req)
	}
	return resp
}

// --- HTTP Handlers ---

func (s *HTTPServer) handleRiskEvaluate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req RiskEvaluationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.UserID == "" {
		writeJSONError(w, http.StatusBadRequest, "user_id required")
		return
	}
	resp := s.EvaluateRisk(r.Context(), &req)
	writeJSON(w, http.StatusOK, resp)
}

func (s *HTTPServer) handleRiskScores(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID := r.URL.Path[len("/api/v1/risk/scores/"):]
	if userID == "" {
		writeJSONError(w, http.StatusBadRequest, "user_id required")
		return
	}
	if s.riskRepo != nil {
		if score, _ := s.riskRepo.GetLatestScore(r.Context(), userID); score != nil {
			writeJSON(w, http.StatusOK, score)
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"score": 0, "level": "low", "decision": "allow"})
}

func (s *HTTPServer) handleRiskPolicy(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPut, http.MethodPost:
		var p RiskPolicy
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		if p.AllowThreshold == 0 { p.AllowThreshold = 30 }
		if p.StepUpThreshold == 0 { p.StepUpThreshold = 60 }
		if p.StrongThreshold == 0 { p.StrongThreshold = 85 }
		if p.Weights == nil { p.Weights = map[string]float64{} }
		if s.riskRepo != nil {
			s.riskRepo.UpsertPolicy(r.Context(), &p)
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "configured", "policy": p})
	case http.MethodGet:
		tenantIDStr := r.URL.Path[len("/api/v1/risk/policies/"):]
		tenantID, _ := uuid.Parse(tenantIDStr)
		if s.riskRepo != nil {
			p, _ := s.riskRepo.GetPolicy(r.Context(), tenantID)
			writeJSON(w, http.StatusOK, p)
			return
		}
		writeJSON(w, http.StatusOK, defaultPolicy(tenantID))
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *HTTPServer) handleRiskSignals(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	// Sort by category for readability.
	signals := make([]SignalDef, len(signalRegistry))
	copy(signals, signalRegistry)
	sort.Slice(signals, func(i, j int) bool {
		return signals[i].Category < signals[j].Category
	})
	writeJSON(w, http.StatusOK, map[string]any{"signals": signals, "count": len(signals)})
}

func (s *HTTPServer) SetRiskRepo(repo *riskRepo) {
	s.riskRepo = repo
}

// SetSodRepo injects the SoD PG repo.
func (s *HTTPServer) SetSodRepo(r *sodPGRepo) {
	s.sodRepo = r
}
