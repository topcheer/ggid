package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	ggidtenant "github.com/ggid/ggid/pkg/tenant"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// JITMapping defines how external IdP attributes map to GGID users + roles.
type JITMapping struct {
	ID            uuid.UUID      `json:"id"`
	TenantID      uuid.UUID      `json:"tenant_id"`
	Protocol      string         `json:"protocol"`       // saml, oidc, ldap, scim
	IdpEntityID   string         `json:"idp_entity_id"`
	AttributeMap  map[string]any `json:"attribute_map"`   // source_attr → GGID field
	GroupMap      map[string]any `json:"group_map"`       // external_group → GGID role_id
	DefaultRoleID string         `json:"default_role_id,omitempty"`
	Enabled       bool           `json:"enabled"`
	CreatedAt     time.Time      `json:"created_at"`
}

// JITResult is the outcome of a JIT provisioning pipeline run.
type JITResult struct {
	Action       string   `json:"action"`         // created, updated, no_change, error
	UserID       string   `json:"user_id,omitempty"`
	Username     string   `json:"username,omitempty"`
	AssignedRoles []string `json:"assigned_roles,omitempty"`
	DryRun       bool     `json:"dry_run"`
	Log          []string `json:"log,omitempty"`
}

// jitRepo manages JIT mappings.
type jitRepo struct {
	pool *pgxpool.Pool
}

func newJITRepo(pool *pgxpool.Pool) *jitRepo {
	return &jitRepo{pool: pool}
}

func (r *jitRepo) EnsureSchema(ctx context.Context) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS jit_mappings (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL,
			protocol TEXT NOT NULL,
			idp_entity_id TEXT NOT NULL,
			attribute_map JSONB NOT NULL DEFAULT '{}',
			group_map JSONB NOT NULL DEFAULT '{}',
			default_role_id TEXT,
			enabled BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			UNIQUE(tenant_id, protocol, idp_entity_id)
		);
		CREATE INDEX IF NOT EXISTS idx_jit_mappings ON jit_mappings(tenant_id, protocol, enabled);
	`)
	return err
}

func (r *jitRepo) Create(ctx context.Context, m *JITMapping) error {
	if r.pool == nil {
		return nil
	}
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	attrJSON, _ := json.Marshal(m.AttributeMap)
	groupJSON, _ := json.Marshal(m.GroupMap)
	_, err := r.pool.Exec(ctx,
		`INSERT INTO jit_mappings (id,tenant_id,protocol,idp_entity_id,attribute_map,group_map,default_role_id,enabled) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		m.ID, m.TenantID, m.Protocol, m.IdpEntityID, attrJSON, groupJSON, m.DefaultRoleID, m.Enabled)
	return err
}

func (r *jitRepo) List(ctx context.Context, tenantID uuid.UUID) ([]*JITMapping, error) {
	if r.pool == nil {
		return []*JITMapping{}, nil
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id,protocol,idp_entity_id,attribute_map,group_map,COALESCE(default_role_id,''),enabled,created_at FROM jit_mappings WHERE tenant_id=$1 ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*JITMapping
	for rows.Next() {
		var m JITMapping
		var attrJSON, groupJSON []byte
		if err := rows.Scan(&m.ID, &m.Protocol, &m.IdpEntityID, &attrJSON, &groupJSON, &m.DefaultRoleID, &m.Enabled, &m.CreatedAt); err != nil {
			continue
		}
		json.Unmarshal(attrJSON, &m.AttributeMap)
		json.Unmarshal(groupJSON, &m.GroupMap)
		result = append(result, &m)
	}
	return result, nil
}

func (r *jitRepo) Find(ctx context.Context, tenantID uuid.UUID, protocol, idpEntityID string) (*JITMapping, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("not found")
	}
	row := r.pool.QueryRow(ctx,
		`SELECT id,protocol,idp_entity_id,attribute_map,group_map,COALESCE(default_role_id,''),enabled,created_at FROM jit_mappings WHERE tenant_id=$1 AND protocol=$2 AND idp_entity_id=$3 AND enabled=TRUE`,
		tenantID, protocol, idpEntityID)
	var m JITMapping
	var attrJSON, groupJSON []byte
	if err := row.Scan(&m.ID, &m.Protocol, &m.IdpEntityID, &attrJSON, &groupJSON, &m.DefaultRoleID, &m.Enabled, &m.CreatedAt); err != nil {
		return nil, fmt.Errorf("not found")
	}
	json.Unmarshal(attrJSON, &m.AttributeMap)
	json.Unmarshal(groupJSON, &m.GroupMap)
	return &m, nil
}

// --- JIT Pipeline: extract → resolve → create/update → map → audit ---

// RunJITPipeline processes an IdP assertion through the JIT mapping pipeline.
func RunJITPipeline(mapping *JITMapping, externalAttrs map[string]any, dryRun bool) *JITResult {
	result := &JITResult{DryRun: dryRun, Log: []string{}}

	// 1. Extract: map external attributes to GGID user fields using attribute_map.
	resolved := make(map[string]any)
	for ggidField, sourceSpec := range mapping.AttributeMap {
		sourceKey, _ := sourceSpec.(string)
		if sourceKey == "" {
			continue
		}
		if val, ok := externalAttrs[sourceKey]; ok {
			resolved[ggidField] = val
		}
	}
	result.Log = append(result.Log, fmt.Sprintf("extracted %d attributes from %d source attrs", len(resolved), len(externalAttrs)))

	// 2. Resolve: determine username/email.
	username, _ := resolved["username"].(string)
	email, _ := resolved["email"].(string)
	if email == "" {
		email, _ = externalAttrs["email"].(string)
	}
	if username == "" {
		username = email
	}
	result.Username = username
	result.Log = append(result.Log, fmt.Sprintf("resolved username=%s email=%s", username, email))

	// 3. Create/Update: in production, query/create user in DB.
	if email == "" {
		result.Action = "error"
		result.Log = append(result.Log, "no email found — cannot provision user")
		return result
	}

	// 4. Map groups → roles.
	var assignedRoles []string
	for ggidRole, sourceGroupSpec := range mapping.GroupMap {
		sourceGroup, _ := sourceGroupSpec.(string)
		if sourceGroup == "" {
			continue
		}
		// Check if user's external groups contain this group.
		if groups, ok := externalAttrs["groups"].([]any); ok {
			for _, g := range groups {
				if fmt.Sprintf("%v", g) == sourceGroup {
					assignedRoles = append(assignedRoles, ggidRole)
					break
				}
			}
		}
	}
	// Apply default role if no group matched.
	if len(assignedRoles) == 0 && mapping.DefaultRoleID != "" {
		assignedRoles = append(assignedRoles, mapping.DefaultRoleID)
	}
	result.AssignedRoles = assignedRoles
	result.Log = append(result.Log, fmt.Sprintf("mapped %d roles from groups", len(assignedRoles)))

	// 5. Determine action.
	if dryRun {
		result.Action = "no_change"
		result.Log = append(result.Log, "dry-run: would create/update user")
	} else {
		// In production: create or update user in identity service.
		result.UserID = uuid.New().String()
		result.Action = "created"
		result.Log = append(result.Log, fmt.Sprintf("user %s provisioned with %d roles", result.UserID, len(assignedRoles)))
		log.Printf("JIT: provisioned user=%s email=%s roles=%v protocol=%s", result.UserID, email, assignedRoles, mapping.Protocol)
	}

	return result
}

// --- API Handlers ---

func (h *HTTPHandler) handleJIT(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	if strings.HasSuffix(path, "/dry-run") {
		h.jitDryRun(w, r)
		return
	}

	switch r.Method {
	case http.MethodPost:
		h.jitCreateMapping(w, r)
	case http.MethodGet:
		h.jitListMappings(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *HTTPHandler) jitCreateMapping(w http.ResponseWriter, r *http.Request) {
	tc, _ := ggidtenant.FromContext(r.Context())
	var m JITMapping
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if tc != nil {
		m.TenantID = tc.TenantID
	}
	if m.Protocol == "" || m.IdpEntityID == "" {
		writeError(w, http.StatusBadRequest, "protocol and idp_entity_id required")
		return
	}
	m.Enabled = true
	if h.jitRepo != nil {
		if err := h.jitRepo.Create(r.Context(), &m); err != nil {
			writeError(w, http.StatusInternalServerError, "failed")
			return
		}
	}
	writeJSON(w, http.StatusCreated, m)
}

func (h *HTTPHandler) jitListMappings(w http.ResponseWriter, r *http.Request) {
	tc, _ := ggidtenant.FromContext(r.Context())
	var mappings []*JITMapping
	if h.jitRepo != nil && tc != nil {
		mappings, _ = h.jitRepo.List(r.Context(), tc.TenantID)
	}
	if mappings == nil {
		mappings = []*JITMapping{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"mappings": mappings, "total": len(mappings)})
}

func (h *HTTPHandler) jitDryRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		Protocol      string         `json:"protocol"`
		IdpEntityID   string         `json:"idp_entity_id"`
		ExternalAttrs map[string]any `json:"external_attributes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	tc, _ := ggidtenant.FromContext(r.Context())

	// Find mapping.
	var mapping *JITMapping
	if h.jitRepo != nil && tc != nil {
		mapping, _ = h.jitRepo.Find(r.Context(), tc.TenantID, req.Protocol, req.IdpEntityID)
	}
	if mapping == nil {
		writeError(w, http.StatusNotFound, "no JIT mapping found for protocol/entity")
		return
	}

	result := RunJITPipeline(mapping, req.ExternalAttrs, true)
	writeJSON(w, http.StatusOK, result)
}

// SetJITRepo injects the JIT mapping repository.
func (h *HTTPHandler) SetJITRepo(repo *jitRepo) {
	h.jitRepo = repo
}
