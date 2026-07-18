package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	ggidtenant "github.com/ggid/ggid/pkg/tenant"
	"github.com/google/uuid"
)

// handleZTNA routes ZTNA Access Broker endpoints.
func (h *HTTPHandler) handleZTNA(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	switch {
	case strings.HasSuffix(path, "/ztna/apps"):
		switch r.Method {
		case http.MethodGet:
			h.ztnaListApps(w, r)
		case http.MethodPost:
			h.ztnaCreateApp(w, r)
		default:
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	case strings.HasSuffix(path, "/ztna/access-logs"):
		h.ztnaListLogs(w, r)
	case strings.HasSuffix(path, "/ztna/metrics"):
		h.ztnaMetrics(w, r)
	case strings.HasSuffix(path, "/ztna/test-policy"):
		h.ztnaTestPolicy(w, r)
	case strings.Contains(path, "/ztna/apps/"):
		h.ztnaAppByID(w, r)
	default:
		writeJSONError(w, http.StatusNotFound, "not found")
	}
}

func (h *HTTPHandler) ztnaCreateApp(w http.ResponseWriter, r *http.Request) {
	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "tenant context required")
		return
	}
	var app ProtectedApp
	if err := json.NewDecoder(r.Body).Decode(&app); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid body")
		return
	}
	app.TenantID = tc.TenantID
	if app.Name == "" || app.Slug == "" || app.UpstreamURL == "" {
		writeJSONError(w, http.StatusBadRequest, "name, slug, upstream_url required")
		return
	}
	if app.AuthMode == "" {
		app.AuthMode = "jwt"
	}
	if app.RateLimitPerMin == 0 {
		app.RateLimitPerMin = 100
	}
	app.Enabled = true
	if err := h.abRepo.CreateApp(r.Context(), &app); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to create app")
		return
	}
	writeJSON(w, http.StatusCreated, app)
}

func (h *HTTPHandler) ztnaListApps(w http.ResponseWriter, r *http.Request) {
	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "tenant context required")
		return
	}
	apps, err := h.abRepo.ListApps(r.Context(), tc.TenantID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed")
		return
	}
	if apps == nil {
		apps = []*ProtectedApp{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"apps": apps, "total": len(apps)})
}

func (h *HTTPHandler) ztnaAppByID(w http.ResponseWriter, r *http.Request) {
	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "tenant context required")
		return
	}
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		writeJSONError(w, http.StatusNotFound, "not found")
		return
	}
	appID, err := uuid.Parse(parts[len(parts)-1])
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid app id")
		return
	}
	switch r.Method {
	case http.MethodPut:
		var app ProtectedApp
		if err := json.NewDecoder(r.Body).Decode(&app); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid body")
			return
		}
		app.ID = appID
		app.TenantID = tc.TenantID
		if err := h.abRepo.UpdateApp(r.Context(), &app); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed")
			return
		}
		writeJSON(w, http.StatusOK, app)
	case http.MethodDelete:
		if err := h.abRepo.DeleteApp(r.Context(), appID, tc.TenantID); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed")
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *HTTPHandler) ztnaListLogs(w http.ResponseWriter, r *http.Request) {
	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "tenant context required")
		return
	}
	var appID *uuid.UUID
	if aid := r.URL.Query().Get("app_id"); aid != "" {
		if id, err := uuid.Parse(aid); err == nil {
			appID = &id
		}
	}
	logs, err := h.abRepo.ListAccessLogs(r.Context(), tc.TenantID, appID, 50)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed")
		return
	}
	if logs == nil {
		logs = []*AppAccessLog{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"logs": logs, "total": len(logs)})
}

func (h *HTTPHandler) ztnaMetrics(w http.ResponseWriter, r *http.Request) {
	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "tenant context required")
		return
	}
	apps, _ := h.abRepo.ListApps(r.Context(), tc.TenantID)
	writeJSON(w, http.StatusOK, map[string]any{
		"total_apps":     len(apps),
		"enabled_apps":   len(apps),
		"health_summary": map[string]int{"healthy": 0, "unhealthy": 0, "unknown": len(apps)},
	})
}

func (h *HTTPHandler) ztnaTestPolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		AccessPolicy map[string]any `json:"access_policy"`
		User         map[string]any `json:"user"`
		Security     map[string]any `json:"security"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid body")
		return
	}
	decision := evaluateAccessPolicy(req.AccessPolicy, req.User, req.Security)
	writeJSON(w, http.StatusOK, decision)
}

// PDPDecision is the result of an access policy evaluation.
type PDPDecision struct {
	Decision string `json:"decision"` // allow, deny, stepup
	Reason   string `json:"reason,omitempty"`
}

// evaluateAccessPolicy performs a simplified ABAC condition evaluation.
// Conditions are checked against user + security context maps.
func evaluateAccessPolicy(policy, user, security map[string]any) PDPDecision {
	if len(policy) == 0 {
		return PDPDecision{Decision: "allow", Reason: "no policy configured"}
	}
	conditions, ok := policy["conditions"].(map[string]any)
	if !ok {
		return PDPDecision{Decision: "allow", Reason: "no conditions in policy"}
	}
	// Check "and" conditions.
	andConds, ok := conditions["and"].([]any)
	if !ok {
		return PDPDecision{Decision: "allow", Reason: "no and-conditions"}
	}
	for _, cond := range andConds {
		condMap, ok := cond.(map[string]any)
		if !ok {
			continue
		}
		for key, expected := range condMap {
			actual := resolveAttribute(key, user, security)
			if !matchConditionValue(actual, expected) {
				return PDPDecision{Decision: "deny", Reason: "condition failed: " + key}
			}
		}
	}
	return PDPDecision{Decision: "allow"}
}

func resolveAttribute(key string, user, security map[string]any) any {
	if strings.HasPrefix(key, "$user.") {
		return user[strings.TrimPrefix(key, "$user.")]
	}
	if strings.HasPrefix(key, "$security.") {
		return security[strings.TrimPrefix(key, "$security.")]
	}
	return nil
}

func matchConditionValue(actual, expected any) bool {
	// Handle $eq, $in operators.
	if expMap, ok := expected.(map[string]any); ok {
		if eq, has := expMap["$eq"]; has {
			return anyEqual(actual, eq)
		}
		if in, has := expMap["$in"].([]any); has {
			for _, v := range in {
				if anyEqual(actual, v) {
					return true
				}
			}
			return false
		}
	}
	return anyEqual(actual, expected)
}

func anyEqual(a, b any) bool {
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}
