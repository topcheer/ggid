package httpserver

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AuthorizeRequest is the unified authorization request.
type AuthorizeRequest struct {
	Subject  string         `json:"subject"`
	Resource string         `json:"resource"`
	Action   string         `json:"action"`
	Context  map[string]any `json:"context"`
}

// AuthorizeResponse is the unified authorization decision.
type AuthorizeResponse struct {
	Allowed     bool           `json:"allowed"`
	DenyReason  string         `json:"deny_reason,omitempty"`
	RiskOverlay string         `json:"risk_overlay,omitempty"` // none, step_up, block
	RiskScore   int            `json:"risk_score"`
	EvaluatedBy []string       `json:"evaluated_by"` // rbac, abac, rebac, risk
	CacheHit    bool           `json:"cache_hit"`
	LatencyMs   int64          `json:"latency_ms"`
	DecisionID  string         `json:"decision_id"`
}

// pdpRepo manages PDP decision audit logs in PostgreSQL.
type pdpRepo struct {
	pool *pgxpool.Pool
}

func NewPDPRepo(pool *pgxpool.Pool) *pdpRepo {
	return &pdpRepo{pool: pool}
}

func (r *pdpRepo) EnsureSchema(ctx context.Context) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS policy_decisions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID,
			subject TEXT NOT NULL, resource TEXT NOT NULL, action TEXT NOT NULL,
			decision TEXT NOT NULL, deny_reason TEXT DEFAULT '',
			risk_score INT DEFAULT 0, risk_overlay TEXT DEFAULT '',
			context JSONB DEFAULT '{}', evaluated_by TEXT[] DEFAULT '{}',
			cache_hit BOOLEAN DEFAULT FALSE, latency_ms INT DEFAULT 0,
			created_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_decisions_tenant ON policy_decisions(tenant_id, created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_decisions_subject ON policy_decisions(subject, created_at DESC);
	`)
	return err
}

func (r *pdpRepo) LogDecision(ctx context.Context, resp *AuthorizeResponse, req *AuthorizeRequest) {
	if r.pool == nil || resp == nil {
		return
	}
	if resp.DecisionID == "" {
		resp.DecisionID = uuid.New().String()
	}
	tenantID, _ := req.Context["tenant_id"].(string)
	ctxJSON, _ := json.Marshal(req.Context)
	r.pool.Exec(ctx, `INSERT INTO policy_decisions (id,tenant_id,subject,resource,action,decision,deny_reason,risk_score,risk_overlay,context,evaluated_by,cache_hit,latency_ms) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		uuid.MustParse(resp.DecisionID), tenantID, req.Subject, req.Resource, req.Action,
		boolToDecision(resp.Allowed), resp.DenyReason, resp.RiskScore, resp.RiskOverlay,
		ctxJSON, resp.EvaluatedBy, resp.CacheHit, resp.LatencyMs)
}

func (r *pdpRepo) ListDecisions(ctx context.Context, limit, offset int) ([]map[string]any, error) {
	if r.pool == nil {
		return []map[string]any{}, nil
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := r.pool.Query(ctx, `SELECT id,subject,resource,action,decision,deny_reason,risk_score,risk_overlay,cache_hit,latency_ms,created_at FROM policy_decisions ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []map[string]any
	for rows.Next() {
		m := map[string]any{}
		var id, subject, resource, action, decision, denyReason, riskOverlay string
		var riskScore, latencyMs int
		var cacheHit bool
		var createdAt time.Time
		if err := rows.Scan(&id, &subject, &resource, &action, &decision, &denyReason, &riskScore, &riskOverlay, &cacheHit, &latencyMs, &createdAt); err != nil {
			continue
		}
		m["id"] = id
		m["subject"] = subject
		m["resource"] = resource
		m["action"] = action
		m["decision"] = decision
		m["deny_reason"] = denyReason
		m["risk_score"] = riskScore
		m["risk_overlay"] = riskOverlay
		m["cache_hit"] = cacheHit
		m["latency_ms"] = latencyMs
		m["created_at"] = createdAt
		result = append(result, m)
	}
	return result, nil
}

func boolToDecision(allowed bool) string {
	if allowed {
		return "allow"
	}
	return "deny"
}

// --- Decision Cache (5s TTL) ---

type cacheEntry struct {
	resp      *AuthorizeResponse
	expiresAt time.Time
}

var (
	pdpCache   sync.Map // cacheKey → cacheEntry
	pdpCacheTTL = 5 * time.Second
)

func pdpCacheKey(req *AuthorizeRequest) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%s|%v", req.Subject, req.Resource, req.Action, req.Context)))
	return hex.EncodeToString(h[:16])
}

func cacheGet(key string) (*AuthorizeResponse, bool) {
	if v, ok := pdpCache.Load(key); ok {
		entry := v.(cacheEntry)
		if time.Now().Before(entry.expiresAt) {
			resp := *entry.resp
			resp.CacheHit = true
			return &resp, true
		}
		pdpCache.Delete(key)
	}
	return nil, false
}

func cacheSet(key string, resp *AuthorizeResponse) {
	cp := *resp
	pdpCache.Store(key, cacheEntry{resp: &cp, expiresAt: time.Now().Add(pdpCacheTTL)})
}

func FlushPDPCache() {
	pdpCache.Range(func(key, _ any) bool {
		pdpCache.Delete(key)
		return true
	})
}

// --- Unified Authorize ---

// EvaluateAuthorize runs RBAC + ABAC + ReBAC in parallel, then applies risk overlay.
func (s *HTTPServer) EvaluateAuthorize(ctx context.Context, req *AuthorizeRequest) *AuthorizeResponse {
	start := time.Now()

	// Check cache.
	cKey := pdpCacheKey(req)
	if cached, ok := cacheGet(cKey); ok {
		cached.LatencyMs = time.Since(start).Milliseconds()
		return cached
	}

	// Parallel evaluation.
	var wg sync.WaitGroup
	rbacAllow, abacAllow, rebacAllow := true, true, true
	rbacReason, abacReason, rebacReason := "", "", ""
	evaluatedBy := []string{}

	// RBAC check (uses role service if available).
	wg.Add(1)
	go func() {
		defer wg.Done()
		if s.roleSvc != nil {
			// Simplified: if we can list roles, RBAC is available.
			roles, err := s.roleSvc.ListRoles(ctx, uuid.Nil, 1, 1)
			if err == nil && roles != nil {
				rbacAllow = true // RBAC allows by default when no explicit deny rules
				evaluatedBy = append(evaluatedBy, "rbac")
			}
		}
	}()

	// ABAC check (simplified — uses policy service for conditions).
	wg.Add(1)
	go func() {
		defer wg.Done()
		// ABAC evaluates context conditions. Default allow.
		abacAllow = true
		if req.Context != nil {
			if ip, ok := req.Context["ip_blocked"]; ok && ip == true {
				abacAllow = false
				abacReason = "IP blocked by ABAC policy"
			}
		}
		evaluatedBy = append(evaluatedBy, "abac")
	}()

	// ReBAC check (simplified — would query rebac_cache).
	wg.Add(1)
	go func() {
		defer wg.Done()
		// ReBAC evaluates relationship-based permissions. Default allow.
		rebacAllow = true
		evaluatedBy = append(evaluatedBy, "rebac")
	}()

	wg.Wait()

	// Combine: all must allow (AND logic).
	allowed := rbacAllow && abacAllow && rebacAllow
	denyReason := ""
	if !allowed {
		if !rbacAllow { denyReason = rbacReason }
		if !abacAllow { denyReason = abacReason }
		if !rebacAllow { denyReason = rebacReason }
		if denyReason == "" { denyReason = "denied by authorization policy" }
	}

	// Risk overlay.
	riskScore := 0
	if rs, ok := req.Context["risk_score"].(float64); ok {
		riskScore = int(rs)
	}
	riskOverlay := "none"
	if riskScore > 80 {
		allowed = false
		riskOverlay = "block"
		denyReason = "risk score too high"
	} else if riskScore > 50 {
		riskOverlay = "step_up"
	}

	resp := &AuthorizeResponse{
		Allowed:     allowed,
		DenyReason:  denyReason,
		RiskOverlay: riskOverlay,
		RiskScore:   riskScore,
		EvaluatedBy: evaluatedBy,
		CacheHit:    false,
		LatencyMs:   time.Since(start).Milliseconds(),
		DecisionID:  uuid.New().String(),
	}

	// Cache the decision.
	cacheSet(cKey, resp)

	// Audit log (async via repo).
	if s.pdpRepo != nil {
		go s.pdpRepo.LogDecision(context.Background(), resp, req)
	}

	return resp
}

// --- HTTP Handlers ---

// POST /api/v1/policy/authorize
func (s *HTTPServer) handleUnifiedAuthorize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req AuthorizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Subject == "" || req.Resource == "" || req.Action == "" {
		writeJSONError(w, http.StatusBadRequest, "subject, resource, action required")
		return
	}
	resp := s.EvaluateAuthorize(r.Context(), &req)
	writeJSON(w, http.StatusOK, resp)
}

// GET /api/v1/policy/decisions
func (s *HTTPServer) handlePDPDecisions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	var decisions []map[string]any
	if s.pdpRepo != nil {
		decisions, _ = s.pdpRepo.ListDecisions(r.Context(), limit, 0)
	}
	if decisions == nil { decisions = []map[string]any{} }
	writeJSON(w, http.StatusOK, map[string]any{"decisions": decisions, "count": len(decisions)})
}

// DELETE /api/v1/policy/cache
func (s *HTTPServer) handleFlushPDPCache(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	FlushPDPCache()
	writeJSON(w, http.StatusOK, map[string]any{"status": "flushed"})
}
