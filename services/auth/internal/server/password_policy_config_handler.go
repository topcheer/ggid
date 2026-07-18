package server

import (
	"encoding/json"
	"net/http"
)

type PasswordPolicyConfig struct {
	MinLength      int            `json:"min_length"`
	Complexity     map[string]int `json:"complexity"`
	DictionaryCheck bool          `json:"dictionary_check"`
	BreachCheck     bool          `json:"breach_check"`
	ExpiryDays      int            `json:"expiry_days"`
	HistoryCount    int            `json:"history_count"`
	PerRoleOverride map[string]int `json:"per_role_override"`
}

func (h *Handler) handlePasswordPolicyConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		result := PasswordPolicyConfig{
			MinLength:  12,
			Complexity: map[string]int{"uppercase": 1, "lowercase": 1, "digits": 1, "symbols": 1},
			DictionaryCheck: true, BreachCheck: true,
			ExpiryDays: 90, HistoryCount: 5,
			PerRoleOverride: map[string]int{"admin": 16, "service": 20},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	case http.MethodPut:
		var req PasswordPolicyConfig
		json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": "updated", "config": req})
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
