package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// evidenceAttachment stores metadata for a file attached to compliance evidence.
type evidenceAttachment struct {
	ID          string `json:"id"`
	EvidenceID  string `json:"evidence_id"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
	UploadedBy  string `json:"uploaded_by"`
	UploadedAt  string `json:"uploaded_at"`
	Checksum    string `json:"checksum"`
	Description string `json:"description,omitempty"`
}

var evidenceAttachmentStore = struct {
	sync.RWMutex
	data map[string][]evidenceAttachment // evidenceID → attachments
}{data: make(map[string][]evidenceAttachment)}

// POST /api/v1/audit/compliance/evidence/{id}/attach — upload attachment metadata
// GET  /api/v1/audit/compliance/evidence/{id}/attachments — list attachments
func (s *HTTPServer) handleEvidenceAttachments(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/audit/compliance/evidence/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) < 2 {
		writeJSONError(w, http.StatusBadRequest, "evidence ID and action are required")
		return
	}
	evidenceID := parts[0]
	action := parts[1]

	switch {
	case action == "attach" && r.Method == http.MethodPost:
		var req struct {
			Filename    string `json:"filename"`
			ContentType string `json:"content_type"`
			SizeBytes   int64  `json:"size_bytes"`
			UploadedBy  string `json:"uploaded_by"`
			Checksum    string `json:"checksum"`
			Description string `json:"description"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if req.Filename == "" {
			writeJSONError(w, http.StatusBadRequest, "filename is required")
			return
		}
		if req.UploadedBy == "" {
			req.UploadedBy = "system"
		}
		if req.ContentType == "" {
			req.ContentType = "application/octet-stream"
		}

		att := evidenceAttachment{
			ID:          uuid.New().String(),
			EvidenceID:  evidenceID,
			Filename:    req.Filename,
			ContentType: req.ContentType,
			SizeBytes:   req.SizeBytes,
			UploadedBy:  req.UploadedBy,
			UploadedAt:  time.Now().UTC().Format(time.RFC3339),
			Checksum:    req.Checksum,
			Description: req.Description,
		}

		evidenceAttachmentStore.Lock()
		evidenceAttachmentStore.data[evidenceID] = append(evidenceAttachmentStore.data[evidenceID], att)
		evidenceAttachmentStore.Unlock()

		writeJSON(w, http.StatusCreated, att)

	case action == "attachments" && r.Method == http.MethodGet:
		evidenceAttachmentStore.RLock()
		attachments := evidenceAttachmentStore.data[evidenceID]
		result := make([]evidenceAttachment, len(attachments))
		copy(result, attachments)
		evidenceAttachmentStore.RUnlock()

		totalSize := int64(0)
		for _, a := range result {
			totalSize += a.SizeBytes
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"evidence_id":  evidenceID,
			"attachments":  result,
			"total":        len(result),
			"total_size":   totalSize,
		})

	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
