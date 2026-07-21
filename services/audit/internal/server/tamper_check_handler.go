package httpserver

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/ggid/ggid/services/audit/internal/domain"
	"github.com/google/uuid"
)

type TamperIssue struct {
	Type        string `json:"type"` // hash_chain_break, chain_link_break, gap_detected, timestamp_anomaly
	Description string `json:"description"`
	EventID     string `json:"event_id,omitempty"`
	Severity    string `json:"severity"`
}

// auditChainRow is one audit_events row as needed for chain verification.
type auditChainRow struct {
	event       domain.AuditEvent
	piiRedacted bool
}

// GET /api/v1/audit/tamper-check?tenant_id=<uuid>&limit=<n>
// Performs real hash-chain verification over stored audit events:
//  1. content integrity — recompute each event's HMAC hash from stored fields
//  2. chain linkage — each event's prev_hash must equal the prior event's hash
//  3. gap detection — events without a hash (legacy/pre-chain) are counted
//  4. timestamp anomaly — out-of-order created_at in chain order
//
// Critical findings trigger a tamper alert (audit_incidents + error log).
func (s *HTTPServer) handleTamperCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.pool == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "database not available")
		return
	}

	limit := 5000
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 50000 {
			limit = n
		}
	}

	// Resolve tenant scope: explicit query param, X-Tenant-ID, or all tenants.
	var tenantIDs []uuid.UUID
	if ts := r.URL.Query().Get("tenant_id"); ts != "" {
		if id, err := uuid.Parse(ts); err == nil {
			tenantIDs = append(tenantIDs, id)
		}
	} else if ts := r.Header.Get("X-Tenant-ID"); ts != "" {
		if id, err := uuid.Parse(ts); err == nil {
			tenantIDs = append(tenantIDs, id)
		}
	}
	if len(tenantIDs) == 0 {
		rows, err := s.pool.Query(r.Context(),
			`SELECT DISTINCT tenant_id FROM audit_events WHERE hash IS NOT NULL AND hash != '' LIMIT 50`)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to list tenants")
			return
		}
		for rows.Next() {
			var id uuid.UUID
			if rows.Scan(&id) == nil {
				tenantIDs = append(tenantIDs, id)
			}
		}
		rows.Close()
	}

	issues := []TamperIssue{}
	totalVerified, unhashed, redacted := 0, 0, 0

	for _, tenantID := range tenantIDs {
		evs, err := s.loadChainEvents(r, tenantID, limit)
		if err != nil {
			slog.Error("tamper-check: load events failed", "tenant", tenantID, "error", err)
			continue
		}
		v, u, rd, tenantIssues := verifyEventChain(evs)
		totalVerified += v
		unhashed += u
		redacted += rd
		issues = append(issues, tenantIssues...)
	}

	critical := 0
	for _, i := range issues {
		if i.Severity == "critical" {
			critical++
		}
	}
	isClean := critical == 0
	alertTriggered := false

	// Tamper detection alerting: record an incident on critical findings.
	if !isClean {
		alertTriggered = true
		slog.Error("AUDIT TAMPER DETECTED",
			"critical_issues", critical,
			"tenants_checked", len(tenantIDs),
			"events_verified", totalVerified,
		)
		s.recordTamperIncident(r, issues, totalVerified)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"is_clean":            isClean,
		"issues":              issues,
		"issue_count":         len(issues),
		"critical_count":      critical,
		"events_verified":     totalVerified,
		"unhashed_events":     unhashed,
		"redacted_events":     redacted,
		"tenants_checked":     len(tenantIDs),
		"hash_chain_enabled":  domain.IsHashChainEnabled(),
		"alert_triggered":     alertTriggered,
		"checks_run":          []string{"hash_chain_verification", "chain_linkage", "gap_detection", "timestamp_anomaly"},
		"verified_at":         time.Now().UTC().Format(time.RFC3339),
		"recommendation": func() string {
			if isClean {
				return "audit log integrity verified — no tampering detected"
			}
			return "integrity issues detected — investigate immediately"
		}(),
	})
}

// loadChainEvents fetches the most recent events (up to limit) for one
// tenant and returns them in chain order (oldest first).
func (s *HTTPServer) loadChainEvents(r *http.Request, tenantID uuid.UUID, limit int) ([]*auditChainRow, error) {
	rows, err := s.pool.Query(r.Context(), `
		SELECT id, tenant_id, actor_type, actor_id, action,
		       COALESCE(resource_type, ''), resource_id, result,
		       COALESCE(ip_address::text, ''),
		       COALESCE(prev_hash, ''), COALESCE(hash, ''),
		       created_at, COALESCE(metadata, '{}')
		FROM audit_events
		WHERE tenant_id = $1
		ORDER BY created_at DESC, id DESC
		LIMIT $2`, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*auditChainRow
	for rows.Next() {
		e := domain.AuditEvent{}
		var metaBytes []byte
		if err := rows.Scan(
			&e.ID, &e.TenantID, &e.ActorType, &e.ActorID, &e.Action,
			&e.ResourceType, &e.ResourceID, &e.Result,
			&e.IPAddress, &e.PrevHash, &e.Hash, &e.CreatedAt, &metaBytes,
		); err != nil {
			return nil, err
		}
		row := &auditChainRow{event: e}
		if len(metaBytes) > 0 {
			var meta map[string]any
			if json.Unmarshal(metaBytes, &meta) == nil {
				if v, ok := meta["pii_redacted"].(bool); ok && v {
					row.piiRedacted = true
				}
			}
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// Reverse into chain order (oldest first) for sequential verification.
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, nil
}

// verifyEventChain runs all integrity checks over one tenant's chain.
// Returns (verified, unhashed, redacted, issues).
func verifyEventChain(rows []*auditChainRow) (int, int, int, []TamperIssue) {
	issues := []TamperIssue{}
	verified, unhashed, redacted := 0, 0, 0

	var prev *domain.AuditEvent
	for i, row := range rows {
		e := &row.event

		if e.Hash == "" {
			unhashed++
			prev = e
			continue
		}

		// Chain linkage: prev_hash must equal the previous hashed event's hash.
		if i > 0 && prev != nil && prev.Hash != "" && e.PrevHash != "" && e.PrevHash != prev.Hash {
			issues = append(issues, TamperIssue{
				Type:        "chain_link_break",
				Description: fmt.Sprintf("event %s prev_hash does not match previous event %s hash — possible deletion or reordering", e.ID, prev.ID),
				EventID:     e.ID.String(),
				Severity:    "critical",
			})
		}

		// Content integrity: recompute the HMAC hash from stored fields.
		if row.piiRedacted {
			// GDPR-erased event: content was intentionally anonymized, so the
			// stored hash no longer matches a recompute. Linkage is still
			// verified above; the redaction marker must be present (it is).
			redacted++
		} else if domain.IsHashChainEnabled() {
			if !e.VerifyHash(e.PrevHash) {
				issues = append(issues, TamperIssue{
					Type:        "hash_chain_break",
					Description: fmt.Sprintf("event %s stored hash does not match recomputed content hash — possible tampering", e.ID),
					EventID:     e.ID.String(),
					Severity:    "critical",
				})
			} else {
				verified++
			}
		}

		// Timestamp anomaly: out-of-order timestamps in chain order.
		if prev != nil && e.CreatedAt.Before(prev.CreatedAt) {
			issues = append(issues, TamperIssue{
				Type:        "timestamp_anomaly",
				Description: fmt.Sprintf("event %s created_at %s is before previous event %s", e.ID, e.CreatedAt.Format(time.RFC3339), prev.ID),
				EventID:     e.ID.String(),
				Severity:    "low",
			})
		}

		prev = e
	}
	return verified, unhashed, redacted, issues
}

// recordTamperIncident persists a tamper-detection incident to audit_incidents.
func (s *HTTPServer) recordTamperIncident(r *http.Request, issues []TamperIssue, eventsVerified int) {
	critical := []TamperIssue{}
	for _, i := range issues {
		if i.Severity == "critical" {
			critical = append(critical, i)
		}
	}
	if len(critical) > 20 {
		critical = critical[:20]
	}
	data := map[string]any{
		"id":              "tamper-" + strconv.FormatInt(time.Now().UnixNano(), 36),
		"type":            "tamper_detected",
		"title":           "Audit log integrity violation detected",
		"severity":        "critical",
		"status":          "open",
		"issues":          critical,
		"issue_count":     len(issues),
		"events_verified": eventsVerified,
		"detected_at":     time.Now().UTC().Format(time.RFC3339),
		"source":          "tamper-check",
	}
	dataJSON, err := json.Marshal(data)
	if err != nil {
		slog.Error("tamper incident marshal failed", "error", err)
		return
	}
	id := data["id"].(string)
	if _, err := s.pool.Exec(r.Context(),
		`INSERT INTO audit_incidents (id, data) VALUES ($1, $2) ON CONFLICT (id) DO UPDATE SET data = $2`,
		id, string(dataJSON)); err != nil {
		slog.Error("tamper incident insert failed", "error", err)
	}
}
