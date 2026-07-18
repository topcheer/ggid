package httpserver

import (
	"encoding/json"
	"net/http"
)

type ABACConditionConfig struct {
	AttributeSources     map[string]string `json:"attribute_sources"`
	OperatorsPerType     map[string][]string `json:"operators_per_type"`
	ConditionTemplates   []string          `json:"condition_templates"`
	EvaluationCacheTTL   int               `json:"evaluation_cache_ttl_seconds"`
	DefaultDeny          bool              `json:"default_deny"`
}

func (s *HTTPServer) handleABACConditionConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		result := ABACConditionConfig{
			AttributeSources: map[string]string{
				"user.department":  "ldap",
				"user.role":        "database",
				"user.location":    "ip_geolocation",
				"resource.owner":   "database",
				"resource.tag":     "metadata",
				"env.time":         "system",
				"env.risk_score":   "risk_engine",
			},
			OperatorsPerType: map[string][]string{
				"string": {"eq", "ne", "in", "not_in", "contains", "regex"},
				"number": {"eq", "ne", "gt", "lt", "gte", "lte", "range"},
				"bool":   {"eq"},
				"time":   {"before", "after", "between", "weekday", "weekend"},
			},
			ConditionTemplates: []string{
				"user.department == resource.department",
				"user.clearance_level >= resource.min_clearance",
				"env.time between 08:00-18:00",
				"env.risk_score < 0.7",
				"user.location == resource.allowed_region",
			},
			EvaluationCacheTTL: 30,
			DefaultDeny:        true,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	case http.MethodPut:
		var req ABACConditionConfig
		json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": "updated", "config": req})
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
