package server

import (
	"encoding/json"
	"net/http"
)

type ImportValidationConfig struct {
	AllowedFormats       []string          `json:"allowed_formats"`
	MaxBatchSize         int               `json:"max_batch_size"`
	DryRunDefault        bool              `json:"dry_run_default"`
	FieldMappingTemplate map[string]string `json:"field_mapping_template"`
	RequiredFields       []string          `json:"required_fields"`
	ConflictResolution   string            `json:"conflict_resolution"`
	ValidationRules      []string          `json:"validation_rules"`
	PreImportHook        string            `json:"pre_import_hook"`
}

var globalImportValidationConfig = &ImportValidationConfig{
	AllowedFormats:       []string{"csv", "json", "scim"},
	MaxBatchSize:         10000,
	DryRunDefault:        true,
	FieldMappingTemplate: map[string]string{"username": "userName", "email": "emails[0].value", "first_name": "name.givenName", "last_name": "name.familyName"},
	RequiredFields:       []string{"username", "email"},
	ConflictResolution:   "skip",
	ValidationRules:      []string{"email_format", "phone_format", "unique_check"},
	PreImportHook:        "",
}

func (h *HTTPHandler) handleImportValidationConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(globalImportValidationConfig)
	case http.MethodPut:
		var cfg ImportValidationConfig
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if cfg.MaxBatchSize < 1 {
			writeJSONError(w, http.StatusBadRequest, "max_batch_size must be at least 1")
			return
		}
		globalImportValidationConfig = &cfg
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(cfg)
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}