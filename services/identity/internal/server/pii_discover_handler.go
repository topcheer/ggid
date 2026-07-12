package server

import (
	"encoding/json"
	"net/http"
)

type PIIDiscovery struct {
	DataSource    string  `json:"data_source"`
	ColumnType    string  `json:"column_type"`
	PIIType       string  `json:"pii_type"`
	Confidence    float64 `json:"confidence"`
	Masked        bool    `json:"masked"`
	SampleMasked  string  `json:"sample_masked"`
}

type PIIDiscoverResult struct {
	Discoveries     []PIIDiscovery `json:"discoveries"`
	TotalFields     int            `json:"total_fields_scanned"`
	PIIFieldsFound  int            `json:"pii_fields_found"`
	EncryptedFields int            `json:"encrypted_fields"`
	UnencryptedPII  int            `json:"unencrypted_pii"`
	GeneratedAt     string         `json:"generated_at"`
}

func (h *HTTPHandler) handlePIIDiscover(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	result := PIIDiscoverResult{
		Discoveries: []PIIDiscovery{
			{DataSource: "users.email", ColumnType: "varchar", PIIType: "email", Confidence: 0.99, Masked: true, SampleMasked: "j***@example.com"},
			{DataSource: "users.ssn", ColumnType: "varchar", PIIType: "ssn", Confidence: 0.97, Masked: true, SampleMasked: "***-**-1234"},
			{DataSource: "users.phone", ColumnType: "varchar", PIIType: "phone", Confidence: 0.95, Masked: false, SampleMasked: "555-1234"},
			{DataSource: "users.date_of_birth", ColumnType: "date", PIIType: "dob", Confidence: 0.92, Masked: false, SampleMasked: "****-03-15"},
			{DataSource: "profiles.bio", ColumnType: "text", PIIType: "free_text_pii", Confidence: 0.68, Masked: false, SampleMasked: "..."},
			{DataSource: "payments.card_number", ColumnType: "varchar", PIIType: "credit_card", Confidence: 0.98, Masked: true, SampleMasked: "************4242"},
		},
		TotalFields:     450,
		PIIFieldsFound:  6,
		EncryptedFields: 4,
		UnencryptedPII:  2,
		GeneratedAt:     "2025-01-15T10:00:00Z",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
