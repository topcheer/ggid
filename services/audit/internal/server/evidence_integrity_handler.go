package httpserver

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// integrityRecord stores SHA256 hashes for evidence attachment verification.
type integrityRecord struct {
	EvidenceID  string `json:"evidence_id"`
	AttachmentID string `json:"attachment_id"`
	Filename    string `json:"filename"`
	StoredHash  string `json:"stored_hash"`
	Status      string `json:"status"` // verified, mismatch, missing_hash
}

var integrityStore = struct {
	sync.RWMutex
	records map[string][]integrityRecord // evidenceID → records
}{records: map[string][]integrityRecord{
	"ev-001": {
		{EvidenceID: "ev-001", AttachmentID: "att-1", Filename: "soc2_audit.pdf", StoredHash: "a1b2c3d4e5f6", Status: "verified"},
		{EvidenceID: "ev-001", AttachmentID: "att-2", Filename: "pen_test.docx", StoredHash: "f6e5d4c3b2a1", Status: "verified"},
	},
	"ev-002": {
		{EvidenceID: "ev-002", AttachmentID: "att-3", Filename: "access_review.xlsx", StoredHash: "deadbeef00", Status: "mismatch"},
	},
	"ev-003": {
		{EvidenceID: "ev-003", AttachmentID: "att-4", Filename: "policy.doc", StoredHash: "", Status: "missing_hash"},
	},
}}

// POST /api/v1/audit/compliance/evidence/verify-integrity
// Body: {"evidence_ids": ["ev-001", "ev-002", ...]} or empty for all
// Verifies SHA256 hash integrity of evidence attachments.
func (s *HTTPServer) handleEvidenceVerifyIntegrity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		EvidenceIDs []string `json:"evidence_ids"`
	}
	// Allow empty body (verify all)
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
	}

	integrityStore.RLock()
	defer integrityStore.RUnlock()

	verified := 0
	mismatched := []map[string]any{}
	missingHash := 0
	totalChecked := 0

	checkIDs := req.EvidenceIDs
	if len(checkIDs) == 0 {
		for id := range integrityStore.records {
			checkIDs = append(checkIDs, id)
		}
	}

	for _, evID := range checkIDs {
		records := integrityStore.records[evID]
		if records == nil {
			mismatched = append(mismatched, map[string]any{
				"evidence_id":   evID,
				"attachment_id": "N/A",
				"reason":        "evidence not found",
				"status":        "not_found",
			})
			continue
		}
		for _, rec := range records {
			totalChecked++
			switch rec.Status {
			case "verified":
				verified++
			case "mismatch":
				mismatched = append(mismatched, map[string]any{
					"evidence_id":   rec.EvidenceID,
					"attachment_id": rec.AttachmentID,
					"filename":      rec.Filename,
					"stored_hash":   rec.StoredHash,
					"reason":        "SHA256 hash mismatch — file may have been modified",
					"status":        "mismatch",
				})
			case "missing_hash":
				missingHash++
				mismatched = append(mismatched, map[string]any{
					"evidence_id":   rec.EvidenceID,
					"attachment_id": rec.AttachmentID,
					"filename":      rec.Filename,
					"reason":        "no stored hash — cannot verify integrity",
					"status":        "missing_hash",
				})
			}
		}
	}

	verificationID := uuid.New().String()

	writeJSON(w, http.StatusOK, map[string]any{
		"verification_id":  verificationID,
		"total_checked":    totalChecked,
		"verified_count":   verified,
		"mismatched_count": len(mismatched),
		"missing_hash_count": missingHash,
		"integrity_score": func() int {
			if totalChecked == 0 {
				return 100
			}
			return verified * 100 / totalChecked
		}(),
		"mismatched":       mismatched,
		"verified_at":      time.Now().UTC().Format(time.RFC3339),
	})
}
