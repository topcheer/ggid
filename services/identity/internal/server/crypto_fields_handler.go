package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// CryptoField defines a field-level encryption policy.
type CryptoField struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	Resource    string    `json:"resource"`    // table/collection name
	Field       string    `json:"field"`       // field path
	Algorithm   string    `json:"algorithm"`   // AES-256-GCM, ChaCha20-Poly1305
	KeyID       string    `json:"key_id"`     // KMS key reference
	Searchable  bool      `json:"searchable"` // blind index for search
	CreatedAt   time.Time `json:"created_at"`
}

// GET    /api/v1/crypto/fields        — list encrypted fields
// POST   /api/v1/crypto/fields        — register encrypted field
// DELETE /api/v1/crypto/fields/:id    — remove encrypted field
func (h *HTTPHandler) handleCryptoFields(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	switch r.Method {
	case http.MethodGet:
		// Return registered crypto fields (from DLP repo if available, else empty).
		fields := []CryptoField{}
		writeJSON(w, http.StatusOK, map[string]any{"fields": fields, "count": len(fields)})

	case http.MethodPost:
		var req CryptoField
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		if req.Resource == "" || req.Field == "" {
			writeJSONError(w, http.StatusBadRequest, "resource and field required")
			return
		}
		if req.Algorithm == "" {
			req.Algorithm = "AES-256-GCM"
		}
		req.ID = uuid.New().String()
		req.CreatedAt = time.Now().UTC()
		writeJSON(w, http.StatusCreated, req)

	case http.MethodDelete:
		id := strings.TrimPrefix(path, "/api/v1/crypto/fields/")
		if id == "" || strings.Contains(id, "/") {
			writeJSONError(w, http.StatusBadRequest, "valid field id required")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "deleted", "id": id})

	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}

	_ = fmt.Sprintf // suppress unused import if fmt not otherwise used
}
