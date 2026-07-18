package server

import (
	"encoding/json"
	"net/http"
)

type CIBAConfig struct {
	Enabled               bool              `json:"enabled"`
	BindingMessage        string            `json:"binding_message"`
	MaxPollingInterval    int               `json:"max_polling_interval_seconds"`
	RequestedExpiryMax    int               `json:"requested_expiry_max_seconds"`
	TokenDeliveryMode     string            `json:"token_delivery_mode"`
	PerClientConfig       map[string]string `json:"per_client_config"`
}

func handleCIBAConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		result := CIBAConfig{
			Enabled:            true,
			BindingMessage:     "Approve login from {{client_name}}",
			MaxPollingInterval: 5,
			RequestedExpiryMax: 120,
			TokenDeliveryMode:  "poll",
			PerClientConfig:    map[string]string{"banking-app": "push", "mobile-app": "ping"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	case http.MethodPut:
		var req CIBAConfig
		json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": "updated", "config": req})
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
