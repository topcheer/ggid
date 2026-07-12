package server

import (
	"net/http"
	"strings"
)

// GET /api/v1/oauth/clients/{id}/onboarding-checklist
func handleOnboardingChecklist(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet { writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"}); return }
	clientID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/v1/oauth/clients/"), "/onboarding-checklist")
	if clientID == "" || strings.Contains(clientID, "/") { writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid client_id"}); return }
	steps := []map[string]any{
		{"step": "redirect_uris", "label": "Configure redirect URIs", "completed": true},
		{"step": "scopes", "label": "Select required scopes", "completed": true},
		{"step": "branding", "label": "Customize branding", "completed": false},
		{"step": "consent_tested", "label": "Test consent screen", "completed": false},
		{"step": "secret_stored", "label": "Store client secret securely", "completed": true},
		{"step": "admin_approved", "label": "Get admin approval", "completed": false},
	}
	completed := 0
	for _, s := range steps { if s["completed"].(bool) { completed++ } }
	writeJSON(w, http.StatusOK, map[string]any{"client_id": clientID, "steps": steps, "completion_pct": completed * 100 / len(steps), "completed_count": completed, "total_steps": len(steps), "ready_for_production": completed == len(steps)})
}
