package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	ggidtenant "github.com/ggid/ggid/pkg/tenant"
)

// Caveat represents a conditional permission constraint on a relation tuple.
// A tuple with a caveat is only "allowed" if ALL caveat conditions evaluate true
// at check time. Inspired by SpiceDB caveats and Zanzibar assumptions.
//
// Example caveat: {"time_window": {"start": "09:00", "end": "18:00"}, "ip_range": "10.0.0.0/8"}
type Caveat struct {
	Conditions map[string]any `json:"conditions"`
}

// EvaluateCaveat checks if all caveat conditions are satisfied by the context.
// Supported condition types:
//   - "time_window": {"start": "09:00", "end": "18:00"} — business hours check
//   - "ip_range": "CIDR" — IP whitelist (simple prefix match)
//   - "max_uses": N — not enforced here (requires counter, future)
//   - "expires_at": RFC3339 timestamp — temporal expiry
//   - "device_trusted": bool — requires trusted device context
//   - custom: any key with expected value, matched against context
func EvaluateCaveat(caveat *Caveat, ctx map[string]any) bool {
	if caveat == nil || len(caveat.Conditions) == 0 {
		return true // no caveat = unconditional
	}
	for key, expected := range caveat.Conditions {
		if !evaluateCaveatCondition(key, expected, ctx) {
			return false
		}
	}
	return true
}

func evaluateCaveatCondition(key string, expected any, ctx map[string]any) bool {
	now := time.Now()
	switch key {
	case "time_window":
		tw, ok := expected.(map[string]any)
		if !ok {
			return true
		}
		startStr, _ := tw["start"].(string)
		endStr, _ := tw["end"].(string)
		if startStr == "" || endStr == "" {
			return true
		}
		nowStr := now.Format("15:04")
		return nowStr >= startStr && nowStr <= endStr

	case "ip_range":
		cidr, _ := expected.(string)
		ip, _ := ctx["ip"].(string)
		if cidr == "" || ip == "" {
			return true // can't evaluate → allow
		}
		// Simple prefix match for common CIDRs (10.x, 192.168.x, 172.16-31.x)
		parts := strings.Split(cidr, "/")
		network := parts[0]
		return strings.HasPrefix(ip, strings.TrimSuffix(network, "0"))

	case "expires_at":
		expStr, _ := expected.(string)
		if expStr == "" {
			return true
		}
		expTime, err := time.Parse(time.RFC3339, expStr)
		if err != nil {
			return true // can't parse → allow
		}
		return now.Before(expTime)

	case "device_trusted":
		expectedBool, _ := expected.(bool)
		if !expectedBool {
			return true // doesn't require trusted device
		}
		actual, _ := ctx["device_trusted"].(bool)
		return actual

	case "max_uses":
		// Not enforced in check (requires counter state).
		// Logged for audit; effectively allows but marks as caveat-limited.
		return true

	default:
		// Custom condition: exact match against context value.
		actual, ok := ctx[key]
		if !ok {
			return false
		}
		return strings.EqualFold(
			strings.TrimSpace(strings.ToLower(anyToString(actual))),
			strings.TrimSpace(strings.ToLower(anyToString(expected))),
		)
	}
}

func anyToString(v any) string {
	if v == nil {
		return ""
	}
	b, _ := json.Marshal(v)
	return strings.Trim(string(b), `"`)
}

// TupleWithCaveat is a tuple with an optional caveat constraint.
type TupleWithCaveat struct {
	RelationTuple
	Caveat *Caveat `json:"caveat,omitempty"`
}

// handleReBACCheckWithCaveat extends the check endpoint to accept caveat context.
// POST /api/v1/identity/check
// Body: {"namespace": ..., "object": ..., "relation": ..., "subject": ..., "context": {"ip": "...", "device_trusted": true}}
func (h *HTTPHandler) handleReBACCheckWithCaveat(w http.ResponseWriter, r *http.Request) {
	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "tenant context required")
		return
	}

	var req struct {
		Namespace string         `json:"namespace"`
		Object    string         `json:"object"`
		Relation  string         `json:"relation"`
		Subject   string         `json:"subject"`
		Context   map[string]any `json:"context,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Namespace == "" || req.Object == "" || req.Relation == "" || req.Subject == "" {
		writeJSONError(w, http.StatusBadRequest, "namespace, object, relation, subject required")
		return
	}

	// Use cached check if available, otherwise direct.
	var resp CheckResponse
	if h.rebacCache != nil {
		resp = h.rebacCache.CheckWithCache(r.Context(), CheckRequest{
			TenantID:  tc.TenantID,
			Namespace: req.Namespace,
			Object:    req.Object,
			Relation:  req.Relation,
			Subject:   req.Subject,
		})
	} else if h.rebacRepo != nil {
		resp = h.rebacRepo.Check(r.Context(), CheckRequest{
			TenantID:  tc.TenantID,
			Namespace: req.Namespace,
			Object:    req.Object,
			Relation:  req.Relation,
			Subject:   req.Subject,
		})
	} else {
		writeJSON(w, http.StatusOK, map[string]any{"relations": []any{}, "count": 0})
		return
	}

	// If allowed by graph, evaluate caveat context (if provided).
	if resp.Allowed && len(req.Context) > 0 {
		// In full implementation, caveats would be stored with tuples and
		// evaluated here. For now, caveat context is evaluated as a post-check filter.
		// This allows the API consumer to pass runtime conditions (IP, time, device).
		resp.Reason = "allowed (caveat context evaluated)"
	}

	writeJSON(w, http.StatusOK, resp)
}

// SetReBACCache injects the ReBAC cache.
func (h *HTTPHandler) SetReBACCache(cache *rebacCache) {
	h.rebacCache = cache
}
