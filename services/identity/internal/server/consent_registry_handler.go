package server

import (
	"encoding/json"
	"net/http"
)

type ConsentRecord struct {
	ConsentID  string   `json:"consent_id"`
	UserID     string   `json:"user_id"`
	Purpose    string   `json:"purpose"`
	Scopes     []string `json:"scopes"`
	Status     string   `json:"status"`
	GrantedAt  string   `json:"granted_at"`
	ExpiresAt  string   `json:"expires_at"`
}

type ConsentRegistryResult struct {
	Records      []ConsentRecord `json:"records"`
	TotalActive  int             `json:"total_active"`
	TotalExpired int             `json:"total_expired"`
	TotalRevoked int             `json:"total_revoked"`
	ByPurpose    map[string]int  `json:"by_purpose"`
}

func (h *HTTPHandler) handleConsentRegistry(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	result := ConsentRegistryResult{
		Records: []ConsentRecord{
			{ConsentID: "cr-001", UserID: "u-0342", Purpose: "marketing", Scopes: []string{"email", "profile"}, Status: "active", GrantedAt: "2025-01-10T10:00:00Z", ExpiresAt: "2026-01-10T10:00:00Z"},
			{ConsentID: "cr-002", UserID: "u-0517", Purpose: "analytics", Scopes: []string{"behavior"}, Status: "active", GrantedAt: "2025-01-08T14:00:00Z", ExpiresAt: "2025-07-08T14:00:00Z"},
			{ConsentID: "cr-003", UserID: "u-0891", Purpose: "third_party_share", Scopes: []string{"profile", "contacts"}, Status: "revoked", GrantedAt: "2024-12-01T00:00:00Z", ExpiresAt: "2025-06-01T00:00:00Z"},
			{ConsentID: "cr-004", UserID: "u-0342", Purpose: "analytics", Scopes: []string{"behavior"}, Status: "expired", GrantedAt: "2024-06-01T00:00:00Z", ExpiresAt: "2024-12-01T00:00:00Z"},
		},
		TotalActive:  2,
		TotalExpired: 1,
		TotalRevoked: 1,
		ByPurpose:    map[string]int{"marketing": 1, "analytics": 2, "third_party_share": 1},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
