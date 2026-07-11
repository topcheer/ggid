package httpserver

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
)

// POST /api/v1/policies/abac/evaluate
func (s *HTTPServer) handleABACEvaluate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var req struct {
		Attributes map[string]any `json:"attributes"`
		Conditions []struct {
			Field    string `json:"field"`
			Operator string `json:"operator"`
			Value    any    `json:"value"`
		} `json:"conditions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	type matchedRule struct {
		Field   string `json:"field"`
		Matched bool   `json:"matched"`
	}
	var matched []matchedRule
	allMatched := true

	for _, cond := range req.Conditions {
		actual, exists := req.Attributes[cond.Field]
		matchedResult := false
		if exists {
			switch cond.Operator {
			case "eq":
				matchedResult = actual == cond.Value
			case "ne":
				matchedResult = actual != cond.Value
			case "in":
				if vals, ok := cond.Value.([]any); ok {
					for _, v := range vals {
						if v == actual {
							matchedResult = true
							break
						}
					}
				}
			case "startsWith":
				if s1, ok := actual.(string); ok {
					if s2, ok := cond.Value.(string); ok {
						matchedResult = strings.HasPrefix(s1, s2)
					}
				}
			case "endsWith":
				if s1, ok := actual.(string); ok {
					if s2, ok := cond.Value.(string); ok {
						matchedResult = strings.HasSuffix(s1, s2)
					}
				}
			case "regex":
				if s1, ok := actual.(string); ok {
					if s2, ok := cond.Value.(string); ok {
						if re, err := regexp.Compile(s2); err == nil {
							matchedResult = re.MatchString(s1)
						}
					}
				}
			}
		}
		matched = append(matched, matchedRule{Field: cond.Field, Matched: matchedResult})
		if !matchedResult {
			allMatched = false
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"matched":       allMatched,
		"matched_rules": matched,
	})
}
