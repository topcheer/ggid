package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/ggid/ggid/services/auth/internal/repository"
	ggidtenant "github.com/ggid/ggid/pkg/tenant"
	"github.com/google/uuid"
)

// BreakGlassRecord is an alias for the repository type, for JSON compatibility.
type BreakGlassRecord = repository.BreakGlassRecord

// SetBreakGlassRepo injects a DB-backed repository for break-glass records.
func (h *Handler) SetBreakGlassRepo(repo *repository.BreakGlassRepository) {
	h.breakGlassRepo = repo
}

// handleBreakGlass routes break-glass endpoints.
// GET  /api/v1/auth/break-glass/history — list records
// POST /api/v1/auth/break-glass/activate — activate emergency access
func (h *Handler) handleBreakGlass(w http.ResponseWriter, r *http.Request) {
	switch {
	case strings.HasSuffix(r.URL.Path, "/history"):
		h.handleBreakGlassHistory(w, r)
	case strings.HasSuffix(r.URL.Path, "/activate"):
		h.handleBreakGlassActivate(w, r)
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}

// GET /api/v1/auth/break-glass/history
func (h *Handler) handleBreakGlassHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing tenant context")
		return
	}

	if h.breakGlassRepo == nil {
		writeJSON(w, http.StatusOK, []interface{}{})
		return
	}

	limit := 50
	records, err := h.breakGlassRepo.ListByTenant(r.Context(), tc.TenantID, limit)
	if err != nil {
		slog.Error("break-glass history error", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to query break-glass history")
		return
	}
	if records == nil {
		records = []*BreakGlassRecord{}
	}

	writeJSON(w, http.StatusOK, records)
}

type breakGlassActivateRequest struct {
	Reason          string `json:"reason"`
	Scope           string `json:"scope"`
	DurationMinutes int    `json:"duration_minutes"`
	UserID          string `json:"user_id"`
}

// POST /api/v1/auth/break-glass/activate
func (h *Handler) handleBreakGlassActivate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing tenant context")
		return
	}

	var req breakGlassActivateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Reason == "" {
		writeError(w, http.StatusBadRequest, "reason is required for break-glass activation")
		return
	}
	if req.DurationMinutes <= 0 || req.DurationMinutes > 480 {
		req.DurationMinutes = 60 // default 1h, max 8h
	}

	// Extract user ID from request body or JWT.
	var requesterID uuid.UUID
	if req.UserID != "" {
		requesterID, err = uuid.Parse(req.UserID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid user_id")
			return
		}
	} else {
		// Try extracting from X-User-ID header (set by gateway JWT middleware).
		if uidStr := r.Header.Get("X-User-ID"); uidStr != "" {
			requesterID, _ = uuid.Parse(uidStr)
		}
	}
	if requesterID == uuid.Nil {
		writeError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	if h.breakGlassRepo == nil {
		writeError(w, http.StatusServiceUnavailable, "break-glass not configured")
		return
	}

	rec := &BreakGlassRecord{
		ID:              uuid.New(),
		TenantID:        tc.TenantID,
		Requester:       requesterID,
		RequesterName:   r.Header.Get("X-User-Name"),
		Reason:          req.Reason,
		Scope:           req.Scope,
		DurationMinutes: req.DurationMinutes,
		ActivatedAt:     time.Now(),
		Status:          "active",
	}

	if err := h.breakGlassRepo.Create(r.Context(), rec); err != nil {
		slog.Error("break-glass activate error", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to activate break-glass")
		return
	}

	// Audit: break-glass activated.
	h.publishAuditEvent("break_glass.activate", "success", tc.TenantID, requesterID)

	// In production, send SOC webhook notification here.
	slog.Warn("BREAK-GLASS activated", "user_id", requesterID, "tenant_id", tc.TenantID, "reason", req.Reason, "scope", req.Scope, "duration_min", req.DurationMinutes)

	writeJSON(w, http.StatusCreated, map[string]any{
		"id":               rec.ID,
		"status":           rec.Status,
		"activated_at":     rec.ActivatedAt,
		"duration_minutes": rec.DurationMinutes,
	})
}
