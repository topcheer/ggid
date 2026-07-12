package server

import (
	"encoding/json"
	"net/http"
)

type IdPMetadataImport struct {
	ImportMethod    string   `json:"import_method"`
	EntityID        string   `json:"entity_id"`
	SSOURL          string   `json:"sso_url"`
	SLOURL          string   `json:"slo_url"`
	Certificates    []string `json:"certificates"`
	NameIDFormat    string   `json:"name_id_format"`
	ValidationStatus string  `json:"validation_status"`
}

type IdPMetadataImportResult struct {
	Imported IdPMetadataImport `json:"imported"`
	Warnings []string          `json:"warnings"`
}

func (h *HTTPHandler) handleIdPMetadataImport(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		result := IdPMetadataImportResult{
			Imported: IdPMetadataImport{
				ImportMethod:    "URL",
				EntityID:        "https://idp.example.com/metadata",
				SSOURL:          "https://idp.example.com/sso",
				SLOURL:          "https://idp.example.com/slo",
				Certificates:    []string{"MIIDazCCAl...base64...", "MIIEvAIBADAN...base64..."},
				NameIDFormat:    "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress",
				ValidationStatus: "valid",
			},
			Warnings: []string{"Certificate #2 expires in 30 days"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	case http.MethodPut:
		var req struct {
			ImportMethod string `json:"import_method"`
			URL          string `json:"url"`
			XML          string `json:"xml"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		result := IdPMetadataImportResult{
			Imported: IdPMetadataImport{
				ImportMethod:     req.ImportMethod,
				EntityID:         "https://idp.example.com/metadata",
				SSOURL:           "https://idp.example.com/sso",
				NameIDFormat:     "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress",
				ValidationStatus: "valid",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
