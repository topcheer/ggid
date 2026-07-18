package server

import (
	"encoding/json"
	"net/http"
)

type OIDCFederationConfig struct {
	TrustAnchors             []string `json:"trust_anchors"`
	FederatedProviders       []string `json:"federated_providers"`
	AutoDiscovery            bool     `json:"auto_discovery"`
	TrustResolutionPolicy    string   `json:"trust_resolution_policy"`
	EntityCategoryRequirements []string `json:"entity_category_requirements"`
}

func handleOIDCFederationConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		result := OIDCFederationConfig{
			TrustAnchors:          []string{"https://fed.example.com/trust-anchor", "https://edugain.org/anchor"},
			FederatedProviders:    []string{"university-a.eu", "university-b.eu", "research-org.org"},
			AutoDiscovery:         true,
			TrustResolutionPolicy: "strict",
			EntityCategoryRequirements: []string{
				"https://refeds.org/category/research-and-scholarship",
				"https://edugain.org/entity-category",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	case http.MethodPut:
		var req OIDCFederationConfig
		json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": "updated", "config": req})
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
