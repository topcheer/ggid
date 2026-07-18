package server

import (
	"encoding/json"
	"net/http"
)

type ClaimMappingEntry struct {
	ClaimName          string            `json:"claim_name"`
	Source             string            `json:"source"`
	Transform          string            `json:"transform"`
	PerClientOverride  map[string]string `json:"per_client_override"`
}

type ScopeToClaims struct {
	Scope  string   `json:"scope"`
	Claims []string `json:"claims"`
}

type OIDCClaimMappingResult struct {
	Mappings      []ClaimMappingEntry `json:"mappings"`
	ScopeToClaims []ScopeToClaims     `json:"scope_to_claims_matrix"`
}

func handleOIDCClaimMapping(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		result := OIDCClaimMappingResult{
			Mappings: []ClaimMappingEntry{
				{ClaimName: "sub", Source: "user_attr:id", Transform: "direct"},
				{ClaimName: "email", Source: "user_attr:email", Transform: "direct", PerClientOverride: map[string]string{"mobile-app": "user_attr:private_email"}},
				{ClaimName: "groups", Source: "group", Transform: "array:group.name", PerClientOverride: map[string]string{"analytics": "static:viewer"}},
				{ClaimName: "department", Source: "user_attr:department", Transform: "direct"},
				{ClaimName: "tenant_id", Source: "static", Transform: "constant:tenant_from_context"},
			},
			ScopeToClaims: []ScopeToClaims{
				{Scope: "openid", Claims: []string{"sub"}},
				{Scope: "profile", Claims: []string{"sub", "name", "department"}},
				{Scope: "email", Claims: []string{"sub", "email", "email_verified"}},
				{Scope: "groups", Claims: []string{"sub", "groups"}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	case http.MethodPut:
		var req OIDCClaimMappingResult
		json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": "updated", "count": len(req.Mappings)})
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
