package server

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// DLPPolicyRule defines a single DLP egress redaction rule.
type DLPPolicyRule struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	FieldType       string `json:"field_type"`       // ssn, credit_card, email, phone, api_key, jwt, password
	Strategy        string `json:"strategy"`          // full_mask, partial_mask, email_mask, tokenize, remove
	Condition       string `json:"condition"`         // e.g. "role!=admin"
	Classification  string `json:"classification"`    // core, important, general
	Enabled         bool   `json:"enabled"`
	CreatedAt       string `json:"created_at"`
}

// handleDLPoliciesCRUD handles DB-backed DLP policy CRUD.
// Replaces the old hardcoded stub at line 26.
func (h *Handler) handleDLPPoliciesCRUD(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listDLPPolicies(w, r)
	case http.MethodPost:
		h.createDLPPolicy(w, r)
	case http.MethodPut:
		h.updateDLPPolicy(w, r)
	case http.MethodDelete:
		h.deleteDLPPolicy(w, r)
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) listDLPPolicies(w http.ResponseWriter, r *http.Request) {
	if h.memMapRepo != nil {
		policies, err := h.memMapRepo.ListJSON(r.Context(), "auth_dlp_policies")
		if err == nil && policies != nil {
			writeJSON(w, http.StatusOK, map[string]any{"policies": policies, "total": len(policies)})
			return
		}
	}
	// Fallback: return empty list (no hardcoded mock).
	writeJSON(w, http.StatusOK, map[string]any{"policies": []any{}, "total": 0})
}

func (h *Handler) createDLPPolicy(w http.ResponseWriter, r *http.Request) {
	var rule DLPPolicyRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if rule.Name == "" || rule.FieldType == "" || rule.Strategy == "" {
		writeJSONError(w, http.StatusBadRequest, "name, field_type, and strategy are required")
		return
	}
	rule.ID = uuid.New().String()
	if rule.CreatedAt == "" {
		rule.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	rule.Enabled = true

	if h.memMapRepo != nil {
		data, _ := json.Marshal(rule)
		var m map[string]any
		json.Unmarshal(data, &m)
		h.memMapRepo.StoreJSON(r.Context(), "auth_dlp_policies", rule.ID, m)
	}
	writeJSON(w, http.StatusCreated, rule)
}

func (h *Handler) updateDLPPolicy(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 1 {
		writeJSONError(w, http.StatusBadRequest, "policy id required")
		return
	}
	id := parts[len(parts)-1]
	if id == "" || id == "dlp" || id == "policies" {
		writeJSONError(w, http.StatusBadRequest, "policy id required")
		return
	}

	var rule DLPPolicyRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	rule.ID = id

	if h.memMapRepo != nil {
		data, _ := json.Marshal(rule)
		var m map[string]any
		json.Unmarshal(data, &m)
		h.memMapRepo.StoreJSON(r.Context(), "auth_dlp_policies", id, m)
	}
	writeJSON(w, http.StatusOK, rule)
}

func (h *Handler) deleteDLPPolicy(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 1 {
		writeJSONError(w, http.StatusBadRequest, "policy id required")
		return
	}
	id := parts[len(parts)-1]
	if id == "" || id == "dlp" || id == "policies" {
		writeJSONError(w, http.StatusBadRequest, "policy id required")
		return
	}

	if h.memMapRepo != nil {
		h.memMapRepo.DeleteJSON(r.Context(), "auth_dlp_policies", id)
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// handleDLPScan handles POST /api/v1/dlp/scan — test endpoint to scan a response body for PII.
func (h *Handler) handleDLPScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Use the gateway's PII detection (import-free standalone detection).
	matches := detectPII(req.Body)
	writeJSON(w, http.StatusOK, map[string]any{
		"matches":     matches,
		"match_count": len(matches),
		"scanned_at":  time.Now().UTC().Format(time.RFC3339),
	})
}

// detectPII scans a string for PII patterns (lightweight version for auth service).
func detectPII(input string) []map[string]string {
	matches := []map[string]string{}

	// Email
	for _, m := range findPattern(input, `[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}`) {
		matches = append(matches, map[string]string{"type": "email", "value": mask(m)})
	}
	// SSN
	for _, m := range findPattern(input, `\d{3}-\d{2}-\d{4}`) {
		matches = append(matches, map[string]string{"type": "ssn", "value": mask(m)})
	}
	// API key
	for _, m := range findPattern(input, `(?:sk_live_|sk_test_|AKIA)[A-Za-z0-9]{16,}`) {
		matches = append(matches, map[string]string{"type": "api_key", "value": mask(m)})
	}
	// JWT
	for _, m := range findPattern(input, `eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+`) {
		matches = append(matches, map[string]string{"type": "jwt", "value": mask(m)})
	}

	return matches
}

func mask(s string) string {
	if len(s) > 8 {
		return s[:4] + "..." + s[len(s)-4:]
	}
	return "***"
}

// findPattern finds all matches of a regex pattern in input.
func findPattern(input, pattern string) []string {
	re := regexp.MustCompile(pattern)
	return re.FindAllString(input, -1)
}
