package httpserver

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/ggid/ggid/services/audit/internal/domain"
	"github.com/google/uuid"
)

// POST /api/v1/audit/gdpr/forget?user_id=X
// Anonymizes PII in audit_events for the given user while preserving the
// hash chain: hash/prev_hash columns are never touched, so chain linkage
// stays intact. Redacted events are marked with metadata.pii_redacted=true
// and tamper-check skips content recompute for them (linkage still verified).
// The UPDATE runs inside a transaction with the WORM bypass GUC.
func (s *HTTPServer) handleGDPRForget(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		writeJSONError(w, http.StatusBadRequest, "user_id required")
		return
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid user_id")
		return
	}
	if s.pool == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "database not available")
		return
	}

	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "begin transaction failed")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	// WORM bypass: authorized GDPR erasure (Article 17).
	if _, err := tx.Exec(r.Context(), `SET LOCAL app.allow_audit_mutation = 'on'`); err != nil {
		slog.Error("gdpr forget: set mutation GUC failed", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "anonymization not permitted")
		return
	}

	// Anonymize PII. ip_address is inet → set NULL (not a placeholder string).
	// hash/prev_hash are intentionally NOT modified to preserve chain linkage.
	tag, err := tx.Exec(r.Context(), `
		UPDATE audit_events
		SET actor_name = '[anonymized]',
		    user_agent = NULL,
		    ip_address = NULL,
		    metadata = COALESCE(metadata, '{}'::jsonb)
		        - 'name' - 'email' - 'phone' - 'ip'
		        || jsonb_build_object('pii_redacted', true,
		                              'redacted_at', now()::text,
		                              'redaction_basis', 'GDPR Article 17')
		WHERE actor_id = $1`, uid)
	if err != nil {
		slog.Error("gdpr forget: anonymize failed", "user", userID, "error", err)
		writeJSONError(w, http.StatusInternalServerError, "anonymization failed")
		return
	}
	affected := tag.RowsAffected()

	if err := tx.Commit(r.Context()); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "commit failed")
		return
	}

	// Record the erasure itself as a chained audit event.
	tenantID := tenantIDFromRequest(r)
	if s.svc != nil {
		erasureEvent := &domain.AuditEvent{
			TenantID:  tenantID,
			ActorType: domain.ActorSystem,
			Action:    "gdpr.erasure",
			Result:    domain.ResultSuccess,
			Metadata: map[string]any{
				"target_user_id":   userID,
				"events_anonymized": affected,
				"legal_basis":      "GDPR Article 17",
			},
		}
		if err := s.svc.InsertEvent(r.Context(), erasureEvent); err != nil {
			slog.Warn("gdpr forget: erasure event insert failed", "error", err)
		}
	}

	slog.Info("gdpr forget completed", "user", userID, "events_anonymized", affected)
	writeJSON(w, http.StatusOK, map[string]any{
		"status":               "completed",
		"user_id":              userID,
		"events_anonymized":    affected,
		"actions_taken": []map[string]any{
			{"action": "anonymize_audit_events", "affected": affected},
			{"action": "redact_ip_addresses", "affected": affected},
			{"action": "clear_pii_metadata", "affected": "name, email, phone, ip fields"},
		},
		"hash_chain_preserved": true,
		"completed_at":         time.Now().UTC().Format(time.RFC3339),
		"legal_basis":          "GDPR Article 17 — Right to erasure",
	})
}
