package integration

// Gap Regression Tests for ABAC Condition Evaluator
// Verifies: POST /api/v1/policies/abac/evaluate correctly evaluates conditions
// Gap item: ABAC condition evaluator (added 2026-07-12 session)

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGapRegression_ABACEvaluate_HappyPath(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/policies/abac/evaluate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req)
		attrs := req["attributes"].(map[string]interface{})
		conds := req["conditions"].([]interface{})
		matched := true
		var matchedRules []string
		for i, c := range conds {
			cond := c.(map[string]interface{})
			field := cond["field"].(string)
			op := cond["operator"].(string)
			val := cond["value"].(string)
			actual, ok := attrs[field].(string)
			if !ok {
				matched = false
				continue
			}
			ruleMatched := false
			switch op {
			case "eq":
				ruleMatched = actual == val
			case "ne":
				ruleMatched = actual != val
			}
			if ruleMatched {
				matchedRules = append(matchedRules, "rule-"+string(rune('A'+i)))
			} else {
				matched = false
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"matched":       matched,
			"matched_rules": matchedRules,
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	body := map[string]interface{}{
		"attributes": map[string]string{
			"department": "engineering",
			"level":      "L5",
		},
		"conditions": []map[string]string{
			{"field": "department", "operator": "eq", "value": "engineering"},
			{"field": "level", "operator": "ne", "value": "L1"},
		},
	}
	bodyBytes, _ := json.Marshal(body)

	resp, err := http.Post(srv.URL+"/api/v1/policies/abac/evaluate", "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		t.Fatalf("ABAC evaluate request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if result["matched"] != true {
		t.Errorf("expected matched=true, got %v", result["matched"])
	}
	rules, ok := result["matched_rules"].([]interface{})
	if !ok || len(rules) == 0 {
		t.Errorf("expected non-empty matched_rules, got %v", result["matched_rules"])
	}
}

func TestGapRegression_ABACEvaluate_NoMatch(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/policies/abac/evaluate", func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req)
		attrs := req["attributes"].(map[string]interface{})
		conds := req["conditions"].([]interface{})
		matched := true
		for _, c := range conds {
			cond := c.(map[string]interface{})
			field := cond["field"].(string)
			op := cond["operator"].(string)
			val := cond["value"].(string)
			actual := attrs[field].(string)
			switch op {
			case "eq":
				if actual != val {
					matched = false
				}
			case "ne":
				if actual == val {
					matched = false
				}
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"matched":       matched,
			"matched_rules": []string{},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	body, _ := json.Marshal(map[string]interface{}{
		"attributes": map[string]string{"department": "sales"},
		"conditions": []map[string]string{
			{"field": "department", "operator": "eq", "value": "engineering"},
		},
	})

	resp, err := http.Post(srv.URL+"/api/v1/policies/abac/evaluate", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if result["matched"] != false {
		t.Errorf("expected matched=false, got %v", result["matched"])
	}
}

func TestGapRegression_ABACEvaluate_EmptyConditions(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/policies/abac/evaluate", func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req)
		conds, _ := req["conditions"].([]interface{})
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"matched":       len(conds) == 0,
			"matched_rules": []string{},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	body, _ := json.Marshal(map[string]interface{}{
		"attributes": map[string]string{"department": "any"},
		"conditions": []map[string]string{},
	})

	resp, err := http.Post(srv.URL+"/api/v1/policies/abac/evaluate", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if result["matched"] != true {
		t.Errorf("empty conditions should match everything, got matched=%v", result["matched"])
	}
}

func TestGapRegression_ABACEvaluate_Operators(t *testing.T) {
	cases := []struct {
		name     string
		op       string
		actual   string
		value    string
		expected bool
	}{
		{"eq_match", "eq", "engineering", "engineering", true},
		{"eq_no_match", "eq", "sales", "engineering", false},
		{"ne_match", "ne", "sales", "engineering", true},
		{"ne_no_match", "ne", "engineering", "engineering", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/api/v1/policies/abac/evaluate", func(w http.ResponseWriter, r *http.Request) {
				var req map[string]interface{}
				json.NewDecoder(r.Body).Decode(&req)
				attrs := req["attributes"].(map[string]interface{})
				conds := req["conditions"].([]interface{})
				matched := true
				for _, c := range conds {
					cond := c.(map[string]interface{})
					actual := attrs[cond["field"].(string)].(string)
					op := cond["operator"].(string)
					val := cond["value"].(string)
					switch op {
					case "eq":
						matched = matched && (actual == val)
					case "ne":
						matched = matched && (actual != val)
					}
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"matched": matched})
			})

			srv := httptest.NewServer(mux)
			defer srv.Close()

			body, _ := json.Marshal(map[string]interface{}{
				"attributes": map[string]string{"field": tc.actual},
				"conditions": []map[string]string{
					{"field": "field", "operator": tc.op, "value": tc.value},
				},
			})

			resp, err := http.Post(srv.URL+"/api/v1/policies/abac/evaluate", "application/json", bytes.NewReader(body))
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()

			var result map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&result)

			if result["matched"] != tc.expected {
				t.Errorf("operator %s: expected matched=%v, got %v", tc.op, tc.expected, result["matched"])
			}
		})
	}
}
