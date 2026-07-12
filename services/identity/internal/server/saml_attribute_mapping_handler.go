package server

import (
	"encoding/json"
	"net/http"
)

type AttributeMapping struct {
	SourceAttribute string            `json:"source_attribute"`
	TargetField     string            `json:"target_field"`
	TransformRule   string            `json:"transform_rule"`
	PerIdpOverride  map[string]string `json:"per_idp_override"`
}

type AttributeMappingResult struct {
	Mappings    []AttributeMapping `json:"mappings"`
	TestResult  struct {
		Input   string `json:"input"`
		Output  string `json:"output"`
		Valid   bool   `json:"valid"`
	} `json:"test_mapping"`
}

func (h *HTTPHandler) handleSAMLAttributeMapping(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		result := AttributeMappingResult{}
		result.Mappings = []AttributeMapping{
			{SourceAttribute: "email", TargetField: "user.email", TransformRule: "direct", PerIdpOverride: map[string]string{"azure-ad": "userPrincipalName"}},
			{SourceAttribute: "firstName", TargetField: "user.first_name", TransformRule: "direct"},
			{SourceAttribute: "lastName", TargetField: "user.last_name", TransformRule: "direct"},
			{SourceAttribute: "groups", TargetField: "user.groups", TransformRule: "regex:CN=([^,]+)", PerIdpOverride: map[string]string{"okta": "regex:app_([^,]+)"}},
			{SourceAttribute: "department", TargetField: "user.department", TransformRule: "constant:engineering"},
		}
		result.TestResult.Input = "CN=admins,OU=groups,DC=example"
		result.TestResult.Output = "admins"
		result.TestResult.Valid = true
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	case http.MethodPut:
		var req struct{ Mappings []AttributeMapping `json:"mappings"` }
		json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": "updated", "count": len(req.Mappings)})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
