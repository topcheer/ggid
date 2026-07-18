package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// POST /api/v1/auth/login/orchestrate
// Body: {"identifier": "user@example.com"}
// Auto-detects auth method and returns available methods.
func (h *Handler) handleLoginOrchestrate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Identifier string `json:"identifier"`
		TenantID   string `json:"tenant_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Identifier == "" {
		writeError(w, http.StatusBadRequest, "identifier is required")
		return
	}

	identifier := strings.ToLower(strings.TrimSpace(req.Identifier))

	// Detect identifier type
	idType := "username"
	if strings.Contains(identifier, "@") {
		idType = "email"
	} else if len(identifier) >= 10 && strings.HasPrefix(identifier, "+") {
		idType = "phone"
	}

	// Build available methods based on identifier type and system config
	var methods []map[string]any
	methods = append(methods, map[string]any{
		"method":     "password",
		"available":  true,
		"priority":   1,
	})

	if idType == "email" {
		methods = append(methods, map[string]any{
			"method":    "magic_link",
			"available": true,
			"priority":  2,
		})
		methods = append(methods, map[string]any{
			"method":    "email_otp",
			"available": true,
			"priority":  3,
		})
	}

	methods = append(methods, map[string]any{
		"method":    "webauthn",
		"available": true,
		"priority":  4,
	})

	if idType == "email" {
		methods = append(methods, map[string]any{
			"method":    "saml",
			"available": true,
			"priority":  5,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"identifier":        identifier,
		"identifier_type":   idType,
		"available_methods": methods,
		"method_count":      len(methods),
		"evaluated_at":      time.Now().UTC().Format(time.RFC3339),
	})
}
