package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ConsentRecord represents a user's consent grant.
type ConsentRecord struct {
	ID              uuid.UUID  `json:"id"`
	TenantID        uuid.UUID  `json:"tenant_id"`
	UserID          string     `json:"user_id"`
	ClientID        string     `json:"client_id,omitempty"`
	Purpose         string     `json:"purpose"`
	Scopes          []string   `json:"scopes"`
	Status          string     `json:"status"` // active, withdrawn, expired
	PolicyVersion   string     `json:"policy_version"`
	GrantedAt       time.Time  `json:"granted_at"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
	WithdrawnAt     *time.Time `json:"withdrawn_at,omitempty"`
	WithdrawnReason string     `json:"withdrawn_reason,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

// ConsentPurpose defines a purpose for which consent can be granted.
type ConsentPurpose struct {
	ID            uuid.UUID `json:"id"`
	TenantID      uuid.UUID `json:"tenant_id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	DefaultScopes []string  `json:"default_scopes"`
	Required      bool      `json:"required"`
	PolicyVersion string    `json:"policy_version"`
	Enabled       bool      `json:"enabled"`
}

// consentRepo manages consent records in PostgreSQL.
type consentRepo struct {
	pool *pgxpool.Pool
}

func newConsentRepo(pool *pgxpool.Pool) *consentRepo {
	return &consentRepo{pool: pool}
}

func (r *consentRepo) EnsureSchema(ctx context.Context) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS consent_records (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL, user_id TEXT NOT NULL,
			client_id TEXT DEFAULT '', purpose TEXT NOT NULL,
			scopes TEXT[] DEFAULT '{}', status TEXT DEFAULT 'active',
			policy_version TEXT DEFAULT '1.0',
			granted_at TIMESTAMPTZ DEFAULT now(), expires_at TIMESTAMPTZ,
			withdrawn_at TIMESTAMPTZ, withdrawn_reason TEXT,
			created_at TIMESTAMPTZ DEFAULT now(), updated_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_consent_tenant_user ON consent_records(tenant_id, user_id);
		CREATE INDEX IF NOT EXISTS idx_consent_status ON consent_records(tenant_id, status);
		CREATE INDEX IF NOT EXISTS idx_consent_client ON consent_records(client_id, user_id);
		CREATE TABLE IF NOT EXISTS consent_purposes (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL, name TEXT NOT NULL, description TEXT DEFAULT '',
			default_scopes TEXT[] DEFAULT '{}', required BOOLEAN DEFAULT FALSE,
			policy_version TEXT DEFAULT '1.0', enabled BOOLEAN DEFAULT TRUE,
			created_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_consent_purpose_name ON consent_purposes(tenant_id, name);
	`)
	return err
}

func (r *consentRepo) Grant(ctx context.Context, c *ConsentRecord) error {
	if r.pool == nil {
		return nil
	}
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	_, err := r.pool.Exec(ctx, `INSERT INTO consent_records (id,tenant_id,user_id,client_id,purpose,scopes,status,policy_version,granted_at,expires_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		c.ID, c.TenantID, c.UserID, c.ClientID, c.Purpose, c.Scopes, c.Status, c.PolicyVersion, c.GrantedAt, c.ExpiresAt)
	return err
}

func (r *consentRepo) Withdraw(ctx context.Context, id uuid.UUID, reason string) error {
	if r.pool == nil {
		return nil
	}
	now := time.Now().UTC()
	_, err := r.pool.Exec(ctx, `UPDATE consent_records SET status='withdrawn', withdrawn_at=$2, withdrawn_reason=$3, updated_at=now() WHERE id=$1`, id, now, reason)
	return err
}

func (r *consentRepo) List(ctx context.Context, tenantID uuid.UUID, userID string, status string) ([]*ConsentRecord, error) {
	if r.pool == nil {
		return []*ConsentRecord{}, nil
	}
	q := `SELECT id,tenant_id,user_id,client_id,purpose,scopes,status,policy_version,granted_at,expires_at,withdrawn_at,withdrawn_reason,created_at FROM consent_records WHERE tenant_id=$1 AND user_id=$2`
	args := []any{tenantID, userID}
	if status != "" {
		q += ` AND status=$3 ORDER BY created_at DESC`
		args = append(args, status)
	} else {
		q += ` ORDER BY created_at DESC`
	}
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*ConsentRecord
	for rows.Next() {
		c := &ConsentRecord{}
		if err := rows.Scan(&c.ID, &c.TenantID, &c.UserID, &c.ClientID, &c.Purpose, &c.Scopes, &c.Status, &c.PolicyVersion, &c.GrantedAt, &c.ExpiresAt, &c.WithdrawnAt, &c.WithdrawnReason, &c.CreatedAt); err != nil {
			continue
		}
		result = append(result, c)
	}
	return result, nil
}

func (r *consentRepo) HasValidConsent(ctx context.Context, tenantID uuid.UUID, userID, purpose string) (bool, error) {
	if r.pool == nil {
		return false, nil
	}
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM consent_records WHERE tenant_id=$1 AND user_id=$2 AND purpose=$3 AND status='active' AND (expires_at IS NULL OR expires_at > now())`, tenantID, userID, purpose).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *consentRepo) PurgeUser(ctx context.Context, tenantID uuid.UUID, userID string) (int64, error) {
	if r.pool == nil {
		return 0, nil
	}
	tag, err := r.pool.Exec(ctx, `DELETE FROM consent_records WHERE tenant_id=$1 AND user_id=$2`, tenantID, userID)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// --- API Handlers ---

func (h *HTTPHandler) handleConsentRegistry(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// List consent records from DB.
		userID := r.URL.Query().Get("user_id")
		status := r.URL.Query().Get("status")
		var records []*ConsentRecord
		if h.consentRepo != nil {
			tenantID := uuid.Nil
			records, _ = h.consentRepo.List(r.Context(), tenantID, userID, status)
		}
		if records == nil {
			records = []*ConsentRecord{}
		}
		// Compute summary.
		active, expired, revoked := 0, 0, 0
		byPurpose := map[string]int{}
		for _, c := range records {
			byPurpose[c.Purpose]++
			switch c.Status {
			case "active":
				active++
			case "expired":
				expired++
			case "withdrawn":
				revoked++
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"records": records, "total": len(records),
			"total_active": active, "total_expired": expired, "total_revoked": revoked,
			"by_purpose": byPurpose,
		})

	case http.MethodPost:
		// Grant consent.
		var req struct {
			UserID   string   `json:"user_id"`
			Purpose  string   `json:"purpose"`
			Scopes   []string `json:"scopes"`
			ClientID string   `json:"client_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		if req.UserID == "" || req.Purpose == "" {
			writeError(w, http.StatusBadRequest, "user_id and purpose required")
			return
		}
		c := &ConsentRecord{
			TenantID: uuid.Nil, UserID: req.UserID, ClientID: req.ClientID,
			Purpose: req.Purpose, Scopes: req.Scopes, Status: "active",
			PolicyVersion: "1.0", GrantedAt: time.Now().UTC(),
		}
		if h.consentRepo != nil {
			h.consentRepo.Grant(r.Context(), c)
		}
		writeJSON(w, http.StatusCreated, c)

	case http.MethodDelete:
		// Withdraw consent.
		idStr := r.URL.Query().Get("id")
		reason := r.URL.Query().Get("reason")
		id, err := uuid.Parse(idStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "valid id required")
			return
		}
		if h.consentRepo != nil {
			h.consentRepo.Withdraw(r.Context(), id, reason)
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "withdrawn", "id": idStr})

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *HTTPHandler) SetConsentRepo(repo *consentRepo) {
	h.consentRepo = repo
}

var _ = strings.TrimSpace
var _ = fmt.Sprintf
