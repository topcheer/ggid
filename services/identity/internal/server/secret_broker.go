package server

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	ggidtenant "github.com/ggid/ggid/pkg/tenant"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SecretTarget defines a brokered secret target (DB, SSH, cloud API key, etc.).
type SecretTarget struct {
	ID               uuid.UUID      `json:"id"`
	TenantID         uuid.UUID      `json:"tenant_id"`
	Name             string         `json:"name"`
	Type             string         `json:"type"` // db, ssh, cloud, api_key
	ConnectionConfig map[string]any `json:"connection_config"`
	TTLSeconds       int            `json:"ttl_seconds"`
	DefaultRole      string         `json:"default_role"`
	Enabled          bool           `json:"enabled"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

// SecretGrant is a short-lived dynamic credential issued to a user.
type SecretGrant struct {
	ID            uuid.UUID  `json:"id"`
	TenantID      uuid.UUID  `json:"tenant_id"`
	TargetID      uuid.UUID  `json:"target_id"`
	UserID        string     `json:"user_id"`
	Role          string     `json:"role"`
	Credential    string     `json:"credential"` // HMAC-signed short-lived token
	JITRequestID  *uuid.UUID `json:"jit_request_id,omitempty"`
	ExpiresAt     time.Time  `json:"expires_at"`
	Revoked       bool       `json:"revoked"`
	RevokedAt     *time.Time `json:"revoked_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

// secretBrokerRepo manages secret targets + grants in PostgreSQL.
type secretBrokerRepo struct {
	pool *pgxpool.Pool
}

func newSecretBrokerRepo(pool *pgxpool.Pool) *secretBrokerRepo {
	return &secretBrokerRepo{pool: pool}
}

func (r *secretBrokerRepo) EnsureSchema(ctx context.Context) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS secret_targets (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL, name TEXT NOT NULL, type TEXT NOT NULL,
			connection_config JSONB DEFAULT '{}', ttl_seconds INT DEFAULT 3600,
			default_role TEXT DEFAULT '', enabled BOOLEAN DEFAULT TRUE,
			created_at TIMESTAMPTZ DEFAULT now(), updated_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_secret_targets_tenant ON secret_targets(tenant_id, enabled);
		CREATE TABLE IF NOT EXISTS secret_grants (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL,
			target_id UUID NOT NULL REFERENCES secret_targets(id) ON DELETE CASCADE,
			user_id TEXT NOT NULL, role TEXT DEFAULT '',
			credential TEXT NOT NULL, jit_request_id UUID,
			expires_at TIMESTAMPTZ NOT NULL, revoked BOOLEAN DEFAULT FALSE,
			revoked_at TIMESTAMPTZ, created_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_secret_grants_tenant ON secret_grants(tenant_id, expires_at DESC);
		CREATE INDEX IF NOT EXISTS idx_secret_grants_user ON secret_grants(user_id, revoked);
		CREATE INDEX IF NOT EXISTS idx_secret_grants_target ON secret_grants(target_id, revoked);
	`)
	return err
}

// --- Target CRUD ---

func (r *secretBrokerRepo) CreateTarget(ctx context.Context, t *SecretTarget) error {
	if r.pool == nil {
		return nil
	}
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	cfgJSON, _ := json.Marshal(t.ConnectionConfig)
	_, err := r.pool.Exec(ctx, `INSERT INTO secret_targets (id,tenant_id,name,type,connection_config,ttl_seconds,default_role,enabled) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		t.ID, t.TenantID, t.Name, t.Type, cfgJSON, t.TTLSeconds, t.DefaultRole, t.Enabled)
	return err
}

func (r *secretBrokerRepo) ListTargets(ctx context.Context, tenantID uuid.UUID) ([]*SecretTarget, error) {
	if r.pool == nil {
		return []*SecretTarget{}, nil
	}
	rows, err := r.pool.Query(ctx, `SELECT id,name,type,connection_config,ttl_seconds,default_role,enabled,created_at,updated_at FROM secret_targets WHERE tenant_id=$1 ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*SecretTarget
	for rows.Next() {
		var t SecretTarget
		var cfgJSON []byte
		if err := rows.Scan(&t.ID, &t.Name, &t.Type, &cfgJSON, &t.TTLSeconds, &t.DefaultRole, &t.Enabled, &t.CreatedAt, &t.UpdatedAt); err != nil {
			continue
		}
		json.Unmarshal(cfgJSON, &t.ConnectionConfig)
		result = append(result, &t)
	}
	return result, nil
}

func (r *secretBrokerRepo) UpdateTarget(ctx context.Context, t *SecretTarget) error {
	if r.pool == nil {
		return nil
	}
	cfgJSON, _ := json.Marshal(t.ConnectionConfig)
	_, err := r.pool.Exec(ctx, `UPDATE secret_targets SET name=$2,type=$3,connection_config=$4,ttl_seconds=$5,default_role=$6,enabled=$7,updated_at=now() WHERE id=$1 AND tenant_id=$8`,
		t.ID, t.Name, t.Type, cfgJSON, t.TTLSeconds, t.DefaultRole, t.Enabled, t.TenantID)
	return err
}

func (r *secretBrokerRepo) DeleteTarget(ctx context.Context, id, tenantID uuid.UUID) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `DELETE FROM secret_targets WHERE id=$1 AND tenant_id=$2`, id, tenantID)
	return err
}

func (r *secretBrokerRepo) GetTarget(ctx context.Context, id, tenantID uuid.UUID) (*SecretTarget, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("no pool")
	}
	var t SecretTarget
	var cfgJSON []byte
	err := r.pool.QueryRow(ctx, `SELECT id,name,type,connection_config,ttl_seconds,default_role,enabled,created_at,updated_at FROM secret_targets WHERE id=$1 AND tenant_id=$2`, id, tenantID).Scan(&t.ID, &t.Name, &t.Type, &cfgJSON, &t.TTLSeconds, &t.DefaultRole, &t.Enabled, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(cfgJSON, &t.ConnectionConfig)
	return &t, nil
}

// --- Grant Operations ---

func (r *secretBrokerRepo) CreateGrant(ctx context.Context, g *SecretGrant) error {
	if r.pool == nil {
		return nil
	}
	if g.ID == uuid.Nil {
		g.ID = uuid.New()
	}
	_, err := r.pool.Exec(ctx, `INSERT INTO secret_grants (id,tenant_id,target_id,user_id,role,credential,jit_request_id,expires_at,revoked) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,FALSE)`,
		g.ID, g.TenantID, g.TargetID, g.UserID, g.Role, g.Credential, g.JITRequestID, g.ExpiresAt)
	return err
}

func (r *secretBrokerRepo) ListActiveGrants(ctx context.Context, tenantID uuid.UUID) ([]*SecretGrant, error) {
	if r.pool == nil {
		return []*SecretGrant{}, nil
	}
	rows, err := r.pool.Query(ctx, `SELECT id,tenant_id,target_id,user_id,role,credential,jit_request_id,expires_at,revoked,revoked_at,created_at FROM secret_grants WHERE tenant_id=$1 AND revoked=FALSE AND expires_at > now() ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanGrants(rows)
}

func (r *secretBrokerRepo) ListGrantsByTarget(ctx context.Context, targetID, tenantID uuid.UUID) ([]*SecretGrant, error) {
	if r.pool == nil {
		return []*SecretGrant{}, nil
	}
	rows, err := r.pool.Query(ctx, `SELECT id,tenant_id,target_id,user_id,role,credential,jit_request_id,expires_at,revoked,revoked_at,created_at FROM secret_grants WHERE target_id=$1 AND tenant_id=$2 ORDER BY created_at DESC`, targetID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanGrants(rows)
}

func (r *secretBrokerRepo) RevokeGrant(ctx context.Context, grantID, tenantID uuid.UUID) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `UPDATE secret_grants SET revoked=TRUE, revoked_at=now() WHERE id=$1 AND tenant_id=$2`, grantID, tenantID)
	return err
}

func (r *secretBrokerRepo) CleanupExpired(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	if r.pool == nil {
		return 0, nil
	}
	tag, err := r.pool.Exec(ctx, `DELETE FROM secret_grants WHERE tenant_id=$1 AND (expires_at < now() OR revoked=TRUE)`, tenantID)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func scanGrants(rows pgx.Rows) ([]*SecretGrant, error) {
	var result []*SecretGrant
	for rows.Next() {
		var g SecretGrant
		if err := rows.Scan(&g.ID, &g.TenantID, &g.TargetID, &g.UserID, &g.Role, &g.Credential, &g.JITRequestID, &g.ExpiresAt, &g.Revoked, &g.RevokedAt, &g.CreatedAt); err != nil {
			continue
		}
		result = append(result, &g)
	}
	return result, nil
}

// --- Credential Generation ---

// GenerateDynamicCredential creates a short-lived HMAC-signed credential.
// Format: ztb_<random>.<hmac>
func GenerateDynamicCredential(targetID uuid.UUID, userID string, expiresAt time.Time) string {
	raw := make([]byte, 24)
	rand.Read(raw)
	return signCredential(hex.EncodeToString(raw), targetID, userID, expiresAt)
}

func signCredential(nonce string, targetID uuid.UUID, userID string, expiresAt time.Time) string {
	secret := getBrokerSecret()
	payload := fmt.Sprintf("%s|%s|%s|%d", nonce, targetID, userID, expiresAt.Unix())
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	sig := hex.EncodeToString(mac.Sum(nil))
	return fmt.Sprintf("ztb_%s.%s", nonce, sig[:16])
}

func getBrokerSecret() string {
	secret := os.Getenv("GGID_INTERNAL_SECRET")
	if secret == "" {
		secret = "dev-broker-secret"
	}
	return secret
}

// --- API Handlers ---

func (h *HTTPHandler) handleSecretBroker(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	tc, _ := ggidtenant.FromContext(r.Context())

	switch {
	case strings.HasSuffix(path, "/targets"):
		h.sbTargets(w, r, tc)
	case strings.HasSuffix(path, "/targets/"):
		h.sbTargets(w, r, tc)
	case strings.Contains(path, "/targets/"):
		h.sbTargetByID(w, r, tc)
	case strings.HasSuffix(path, "/broker"):
		h.sbBroker(w, r, tc)
	case strings.HasSuffix(path, "/active"):
		h.sbActive(w, r, tc)
	case strings.HasSuffix(path, "/revoke"):
		h.sbRevoke(w, r, tc)
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}

func (h *HTTPHandler) sbTargets(w http.ResponseWriter, r *http.Request, tc *ggidtenant.Context) {
	switch r.Method {
	case http.MethodPost:
		var t SecretTarget
		if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
			writeError(w, http.StatusBadRequest, "invalid body")
			return
		}
		if tc != nil {
			t.TenantID = tc.TenantID
		}
		if t.Name == "" || t.Type == "" {
			writeError(w, http.StatusBadRequest, "name and type required")
			return
		}
		if t.TTLSeconds == 0 {
			t.TTLSeconds = 3600
		}
		t.Enabled = true
		if h.secretBrokerRepo != nil {
			if err := h.secretBrokerRepo.CreateTarget(r.Context(), &t); err != nil {
				writeError(w, http.StatusInternalServerError, "failed")
				return
			}
		}
		writeJSON(w, http.StatusCreated, t)
	case http.MethodGet:
		var targets []*SecretTarget
		if h.secretBrokerRepo != nil && tc != nil {
			targets, _ = h.secretBrokerRepo.ListTargets(r.Context(), tc.TenantID)
		}
		if targets == nil {
			targets = []*SecretTarget{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"targets": targets, "total": len(targets)})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *HTTPHandler) sbTargetByID(w http.ResponseWriter, r *http.Request, tc *ggidtenant.Context) {
	parts := strings.Split(strings.TrimSuffix(r.URL.Path, "/"), "/")
	idStr := parts[len(parts)-1]
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	switch r.Method {
	case http.MethodPut:
		var t SecretTarget
		if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
			writeError(w, http.StatusBadRequest, "invalid body")
			return
		}
		t.ID = id
		if tc != nil {
			t.TenantID = tc.TenantID
		}
		if h.secretBrokerRepo != nil {
			h.secretBrokerRepo.UpdateTarget(r.Context(), &t)
		}
		writeJSON(w, http.StatusOK, t)
	case http.MethodDelete:
		if h.secretBrokerRepo != nil && tc != nil {
			h.secretBrokerRepo.DeleteTarget(r.Context(), id, tc.TenantID)
		}
		writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
	case http.MethodGet:
		if h.secretBrokerRepo != nil && tc != nil {
			t, err := h.secretBrokerRepo.GetTarget(r.Context(), id, tc.TenantID)
			if err != nil {
				writeError(w, http.StatusNotFound, "not found")
				return
			}
			writeJSON(w, http.StatusOK, t)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *HTTPHandler) sbBroker(w http.ResponseWriter, r *http.Request, tc *ggidtenant.Context) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		TargetID     string `json:"target_id"`
		UserID       string `json:"user_id"`
		Role         string `json:"role"`
		JITRequestID string `json:"jit_request_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	targetID, err := uuid.Parse(req.TargetID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid target_id")
		return
	}
	if tc == nil {
		writeError(w, http.StatusBadRequest, "tenant context required")
		return
	}
	// Look up target for TTL + default role.
	target, err := h.secretBrokerRepo.GetTarget(r.Context(), targetID, tc.TenantID)
	if err != nil || target == nil {
		writeError(w, http.StatusNotFound, "target not found")
		return
	}
	if !target.Enabled {
		writeError(w, http.StatusForbidden, "target disabled")
		return
	}
	role := req.Role
	if role == "" {
		role = target.DefaultRole
	}
	ttl := time.Duration(target.TTLSeconds) * time.Second
	expiresAt := time.Now().UTC().Add(ttl)
	cred := GenerateDynamicCredential(targetID, req.UserID, expiresAt)

	grant := &SecretGrant{
		TenantID: tc.TenantID, TargetID: targetID, UserID: req.UserID,
		Role: role, Credential: cred, ExpiresAt: expiresAt,
	}
	if req.JITRequestID != "" {
		if jitID, err := uuid.Parse(req.JITRequestID); err == nil {
			grant.JITRequestID = &jitID
		}
	}
	h.secretBrokerRepo.CreateGrant(r.Context(), grant)
	writeJSON(w, http.StatusCreated, map[string]any{
		"grant_id": grant.ID, "target_id": targetID, "user_id": req.UserID,
		"role": role, "credential": cred, "expires_at": expiresAt,
	})
}

func (h *HTTPHandler) sbActive(w http.ResponseWriter, r *http.Request, tc *ggidtenant.Context) {
	if tc == nil {
		writeJSON(w, http.StatusOK, map[string]any{"grants": []any{}, "total": 0})
		return
	}
	grants, _ := h.secretBrokerRepo.ListActiveGrants(r.Context(), tc.TenantID)
	if grants == nil {
		grants = []*SecretGrant{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"grants": grants, "total": len(grants)})
}

func (h *HTTPHandler) sbRevoke(w http.ResponseWriter, r *http.Request, tc *ggidtenant.Context) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		GrantID string `json:"grant_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	grantID, err := uuid.Parse(req.GrantID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid grant_id")
		return
	}
	if tc == nil {
		writeError(w, http.StatusBadRequest, "tenant context required")
		return
	}
	h.secretBrokerRepo.RevokeGrant(r.Context(), grantID, tc.TenantID)
	writeJSON(w, http.StatusOK, map[string]bool{"revoked": true})
}

func (h *HTTPHandler) SetSecretBrokerRepo(repo *secretBrokerRepo) {
	h.secretBrokerRepo = repo
}
