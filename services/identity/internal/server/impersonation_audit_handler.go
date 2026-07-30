package server

import (
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ImpersonationRecord tracks an impersonation session for audit purposes.
type ImpersonationRecord struct {
	ID           string     `json:"id"`
	Impersonator string     `json:"impersonator"`
	Target       string     `json:"target"`
	TenantID     string     `json:"tenant_id"`
	StartedAt    time.Time  `json:"started_at"`
	EndedAt      *time.Time `json:"ended_at,omitempty"`
	Duration     string     `json:"duration,omitempty"`
	ActionsTaken []string   `json:"actions_taken"`
	Reason       string     `json:"reason"`
	IPAddress    string     `json:"ip_address"`
	UserAgent    string     `json:"user_agent"`
}

// impersonationStore holds impersonation audit records.
type impersonationStore struct {
	mu      sync.RWMutex
	records []*ImpersonationRecord
}

var impersonationAudit = &impersonationStore{}

// RecordImpersonation adds a new impersonation audit entry.
func RecordImpersonation(impersonator, target, tenantID, reason, ip, ua string) *ImpersonationRecord {
	rec := &ImpersonationRecord{
		ID:           uuid.New().String(),
		Impersonator: impersonator,
		Target:       target,
		TenantID:     tenantID,
		StartedAt:    time.Now().UTC(),
		Reason:       reason,
		IPAddress:    ip,
		UserAgent:    ua,
		ActionsTaken: []string{},
	}
	impersonationAudit.mu.Lock()
	impersonationAudit.records = append(impersonationAudit.records, rec)
	impersonationAudit.mu.Unlock()
	return rec
}

// GET /api/v1/audit/impersonation?impersonator=X&target=Y&from=Y&to=Z
// Returns impersonation audit records with filtering.
func (h *HTTPHandler) handleImpersonationAudit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	impersonator := r.URL.Query().Get("impersonator")
	target := r.URL.Query().Get("target")
	// SECURITY (R10): caller tenant from context is mandatory and always
	// applied — the optional query param previously let a tenant admin
	// enumerate every tenant's impersonation records.
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		writeJSONError(w, http.StatusForbidden, "tenant context required")
		return
	}
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	var fromTime, toTime *time.Time
	if fromStr != "" {
		if t, err := time.Parse(time.RFC3339, fromStr); err == nil {
			fromTime = &t
		}
	}
	if toStr != "" {
		if t, err := time.Parse(time.RFC3339, toStr); err == nil {
			toTime = &t
		}
	}

	impersonationAudit.mu.RLock()
	defer impersonationAudit.mu.RUnlock()

	result := []*ImpersonationRecord{}
	for _, rec := range impersonationAudit.records {
		if impersonator != "" && rec.Impersonator != impersonator {
			continue
		}
		if target != "" && rec.Target != target {
			continue
		}
		if tenantID != rec.TenantID {
			continue
		}
		if fromTime != nil && rec.StartedAt.Before(*fromTime) {
			continue
		}
		if toTime != nil && rec.StartedAt.After(*toTime) {
			continue
		}

		// Compute duration if ended — on a local copy, never mutate the
		// shared record under RLock (data race, R10).
		if rec.EndedAt != nil {
			recCopy := *rec
			recCopy.Duration = recCopy.EndedAt.Sub(recCopy.StartedAt).String()
			rec = &recCopy
		}

		result = append(result, rec)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"records": result,
		"count":   len(result),
	})
}
