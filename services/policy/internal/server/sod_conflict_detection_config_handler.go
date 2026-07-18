package httpserver

import (
	"encoding/json"
	"net/http"
)

type SoDConflictDetectionConfig struct {
	Rules struct {
		ByRole       []string `json:"by_role"`
		ByPermission []string `json:"by_permission"`
		ByResource   []string `json:"by_resource"`
	} `json:"rules"`
	SensitivityLevels   map[string]string `json:"sensitivity_levels"`
	AutoRemediate       string            `json:"auto_remediate"`
	ExceptionWorkflow   bool              `json:"exception_workflow"`
}

func (s *HTTPServer) handleSoDConflictDetectionConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		result := SoDConflictDetectionConfig{}
		result.Rules.ByRole = []string{"admin+auditor", "developer+deployer", "approver+executor"}
		result.Rules.ByPermission = []string{"write+delete:same_resource", "grant+revoke:same_scope"}
		result.Rules.ByResource = []string{"prod_db+prod_deploy", "user_create+user_delete"}
		result.SensitivityLevels = map[string]string{"critical": "require_ciso_approval", "high": "require_manager_approval", "medium": "log_only"}
		result.AutoRemediate = "request_approval"
		result.ExceptionWorkflow = true
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	case http.MethodPut:
		var req SoDConflictDetectionConfig
		json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": "updated"})
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
