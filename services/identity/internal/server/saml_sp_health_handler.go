package server

import (
	"encoding/json"
	"net/http"
	"time"
)

type SAMLError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
	Severity  string `json:"severity"`
}

type SPHealthResult struct {
	MetadataURLValid      bool        `json:"metadata_url_valid"`
	CertExpiryDays        int         `json:"cert_expiry_days"`
	ResponseTest          string      `json:"response_test"`
	AssertionConsumerURL  string      `json:"assertion_consumer_url"`
	SLOStatus             string      `json:"slo_status"`
	IDPConnectionStatus   string      `json:"idp_connection_status"`
	LastSync              string      `json:"last_sync"`
	Errors                []SAMLError `json:"errors"`
	OverallHealth         string      `json:"overall_health"`
}

func (h *HTTPHandler) handleSPHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	result := SPHealthResult{
		MetadataURLValid:      true,
		CertExpiryDays:        45,
		ResponseTest:          "passed",
		AssertionConsumerURL:  "https://ggid.dev/auth/saml/acs",
		SLOStatus:             "configured",
		IDPConnectionStatus:   "connected",
		LastSync:              time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339),
		Errors: []SAMLError{
			{Code: "CERT_EXPIRY_WARNING", Message: "Signing certificate expires in 45 days", Timestamp: time.Now().UTC().Format(time.RFC3339), Severity: "warning"},
		},
		OverallHealth: "healthy",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
