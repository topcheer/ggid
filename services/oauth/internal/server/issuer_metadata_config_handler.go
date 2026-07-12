package server

import (
	"encoding/json"
	"net/http"
)

type IssuerMetadataConfig struct {
	IssuerURL                     string   `json:"issuer_url"`
	SupportedResponseTypes        []string `json:"supported_response_types"`
	SupportedSubjectTypes         []string `json:"supported_subject_types"`
	ClaimTypesSupported           []string `json:"claim_types_supported"`
	RequestParameterSupported     bool     `json:"request_parameter_supported"`
	RequireRequestURIRegistration bool     `json:"require_request_uri_registration"`
}

func handleIssuerMetadataConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	result := IssuerMetadataConfig{
		IssuerURL:                 "https://ggid.example.com/oauth",
		SupportedResponseTypes:    []string{"code", "token", "id_token", "code token", "code id_token", "code token id_token"},
		SupportedSubjectTypes:     []string{"public", "pairwise"},
		ClaimTypesSupported:       []string{"normal", "aggregated", "distributed"},
		RequestParameterSupported: true,
		RequireRequestURIRegistration: false,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
