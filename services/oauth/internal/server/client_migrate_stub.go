package server

import (
	"net/http"
)

// handleClientMigration handles client migration requests.
// GET /api/v1/oauth/clients/{id}/migrate — returns migration plan
func handleClientMigration(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "migration_plan_generated",
		"message": "client migration plan available via /migration-data endpoint",
	})
}
