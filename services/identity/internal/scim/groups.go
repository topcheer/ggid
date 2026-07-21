package scim

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	ggidtenant "github.com/ggid/ggid/pkg/tenant"
	"github.com/google/uuid"
)

// SCIMGroup is the RFC 7643 Group resource representation.
type SCIMGroup struct {
	Schemas    []string          `json:"schemas"`
	ID         string            `json:"id"`
	DisplayName string           `json:"displayName"`
	Members    []SCIMGroupMember `json:"members,omitempty"`
	Meta       SCIMMeta          `json:"meta"`
}

// SCIMGroupMember represents a member reference in a SCIM group.
type SCIMGroupMember struct {
	Value   string `json:"value"`   // User ID
	Display string `json:"display"` // User display name
	Ref     string `json:"$ref"`    // SCIM ref
	Type    string `json:"type"`    // "User" or "Group"
}

// HandleGroupsCollectionPublic is the exported wrapper for handleGroupsCollection.
// Allows registering the handler under alternative route prefixes (e.g. /api/v1/scim/Groups).
func (h *Handler) HandleGroupsCollectionPublic(w http.ResponseWriter, r *http.Request) {
	h.handleGroupsCollection(w, r)
}

// HandleGroupResourcePublic is the exported wrapper for HandleGroupResource.
// Allows registering the handler under alternative route prefixes.
func (h *Handler) HandleGroupResourcePublic(w http.ResponseWriter, r *http.Request) {
	h.HandleGroupResource(w, r)
}

// dbPool returns the service DB pool (nil in tests), ensuring the schema
// exists on first use.
func (h *Handler) dbPool(ctx context.Context) *pgxpool.Pool {
	if h == nil || h.svc == nil {
		return nil
	}
	pool := h.svc.Pool()
	if pool != nil {
		ensureGroupSchema(ctx, pool)
	}
	return pool
}

// handleGroupsCollection handles GET (list) and POST (create) for /scim/v2/Groups.
// SCIM Groups map to GGID roles. Members are users assigned to that role.
func (h *Handler) handleGroupsCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listGroups(w, r)
	case http.MethodPost:
		h.createGroup(w, r)
	default:
		writeSCIMError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) listGroups(w http.ResponseWriter, r *http.Request) {
	tc, err := tenantFromRequest(r)
	if err != nil {
		writeSCIMError(w, http.StatusBadRequest, "missing tenant")
		return
	}

	startIndex := 1
	if si := r.URL.Query().Get("startIndex"); si != "" {
		if v, err := strconv.Atoi(si); err == nil && v > 0 {
			startIndex = v
		}
	}
	count := 100
	if c := r.URL.Query().Get("count"); c != "" {
		if v, err := strconv.Atoi(c); err == nil && v > 0 {
			count = v
		}
	}

	filter := r.URL.Query().Get("filter")
	displayName := ""
	if filter != "" {
		// Parse SCIM filter: displayName eq "value"
		if strings.Contains(filter, "displayName eq") {
			parts := strings.SplitN(filter, "displayName eq", 2)
			val := strings.Trim(strings.TrimSpace(parts[1]), `"`)
			displayName = val
		}
	}

	groups := h.getMockGroups(tc.TenantID.String(), displayName)

	// Merge DB-persisted groups (Task-C): created via POST and stored in
	// scim_groups. Role-derived groups keep their IDs; skip ID collisions.
	if pool := h.dbPool(r.Context()); pool != nil {
		if dbGroups, err := dbListGroups(r.Context(), pool, tc.TenantID); err == nil {
			seen := make(map[string]bool, len(groups))
			for _, g := range groups {
				seen[g.ID] = true
			}
			for _, g := range dbGroups {
				if seen[g.ID] {
					continue
				}
				if displayName != "" && !strings.EqualFold(g.DisplayName, displayName) {
					continue
				}
				groups = append(groups, g)
			}
		}
	}

	total := len(groups)
	end := startIndex + count
	if end > total {
		end = total
	}
	if startIndex > total {
		groups = nil
	} else {
		groups = groups[startIndex-1 : end]
	}

	writeSCIMJSON(w, http.StatusOK, map[string]any{
		"schemas":      []string{"urn:ietf:params:scim:api:messages:2.0:ListResponse"},
		"totalResults": total,
		"startIndex":   startIndex,
		"itemsPerPage": len(groups),
		"Resources":    groups,
	})
}

func (h *Handler) createGroup(w http.ResponseWriter, r *http.Request) {
	tc, err := tenantFromRequest(r)
	if err != nil {
		writeSCIMError(w, http.StatusBadRequest, "missing tenant")
		return
	}

	var req struct {
		DisplayName string             `json:"displayName"`
		Members     []SCIMGroupMember  `json:"members"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeSCIMError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.DisplayName == "" {
		writeSCIMError(w, http.StatusBadRequest, "displayName is required")
		return
	}

	groupID := uuid.New().String()
	group := SCIMGroup{
		Schemas:     []string{"urn:ietf:params:scim:schemas:core:2.0:Group"},
		ID:          groupID,
		DisplayName: req.DisplayName,
		Members:     req.Members,
		Meta: SCIMMeta{
			ResourceType: "Group",
			Location:     "/scim/v2/Groups/" + groupID,
		},
	}

	// Persist to PostgreSQL (Task-C) so groups survive restarts and are
	// visible to all replicas.
	if pool := h.dbPool(r.Context()); pool != nil {
		if err := dbCreateGroup(r.Context(), pool, tc.TenantID, &group); err != nil {
			writeSCIMError(w, http.StatusInternalServerError, "failed to persist group")
			return
		}
	}

	writeSCIMJSON(w, http.StatusCreated, group)
}

// --- SCIM Group Resource (GET/PATCH/DELETE by ID) ---

// HandleGroupResource handles operations on /scim/v2/Groups/{id}.
// This is registered as /scim/v2/Groups/ in the mux.
func (h *Handler) HandleGroupResource(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 3 {
		writeSCIMError(w, http.StatusNotFound, "group ID required")
		return
	}
	// Use the last path segment as group ID (supports /scim/v2/Groups/{id} and /api/v1/scim/Groups/{id})
	groupID := pathParts[len(pathParts)-1]

	switch r.Method {
	case http.MethodGet:
		h.getGroup(w, r, groupID)
	case http.MethodPatch:
		h.patchGroup(w, r, groupID)
	case http.MethodDelete:
		h.deleteGroup(w, r, groupID)
	default:
		writeSCIMError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) getGroup(w http.ResponseWriter, r *http.Request, id string) {
	// DB-persisted groups first (Task-C).
	if pool := h.dbPool(r.Context()); pool != nil {
		if g, err := dbGetGroup(r.Context(), pool, id); err == nil && g != nil {
			writeSCIMJSON(w, http.StatusOK, g)
			return
		}
	}
	groups := h.getMockGroups("", "")
	for _, g := range groups {
		if g.ID == id {
			writeSCIMJSON(w, http.StatusOK, g)
			return
		}
	}
	writeSCIMError(w, http.StatusNotFound, "group not found")
}

// patchGroupStore is kept only for backward-compatible test access.
// Production code uses DB via h.svc.Pool().
var patchGroupStore = map[string]*SCIMGroup{}

func (h *Handler) patchGroup(w http.ResponseWriter, r *http.Request, id string) {
	var patch struct {
		Operations []struct {
			Op    string `json:"op"`
			Path  string `json:"path"`
			Value any    `json:"value"`
		} `json:"Operations"`
	}
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		writeSCIMError(w, http.StatusBadRequest, "invalid PATCH body")
		return
	}

	// Try DB-backed approach first; fall back to in-memory for tests
	var pool *pgxpool.Pool
	if h.svc != nil {
		pool = h.dbPool(r.Context())
	}
	if pool != nil {
		// Load group from DB (Task-C: scim_groups table)
		group, err := dbGetGroup(r.Context(), pool, id)
		if err != nil || group == nil {
			writeSCIMError(w, http.StatusNotFound, "group not found")
			return
		}

		// Apply each operation, then persist once.
		for _, op := range patch.Operations {
			opPath := strings.ToLower(strings.TrimSpace(op.Path))
			switch strings.ToLower(op.Op) {
			case "replace":
				if opPath == "displayname" {
					if name, ok := op.Value.(string); ok {
						group.DisplayName = name
					}
				} else if opPath == "members" {
					group.Members = valueToMembers(op.Value)
				}
			case "add":
				if opPath == "members" {
					newMembers := valueToMembers(op.Value)
					existing := make(map[string]bool)
					for _, m := range group.Members {
						existing[m.Value] = true
					}
					for _, m := range newMembers {
						if !existing[m.Value] {
							group.Members = append(group.Members, m)
							existing[m.Value] = true
						}
					}
				}
			case "remove":
				if opPath == "members" || strings.HasPrefix(opPath, "members[") {
					removeIDs := parseMemberFilter(op.Path)
					if len(removeIDs) > 0 {
						var filtered []SCIMGroupMember
						for _, m := range group.Members {
							if !removeIDs[m.Value] {
								filtered = append(filtered, m)
							}
						}
						group.Members = filtered
					} else {
						group.Members = nil
					}
				}
			}
		}

		if err := dbUpdateGroup(r.Context(), pool, group); err != nil {
			writeSCIMError(w, http.StatusInternalServerError, "failed to update group")
			return
		}
		writeSCIMJSON(w, http.StatusOK, group)
		return
	}

	// Fallback: in-memory store for tests
	if len(patchGroupStore) == 0 {
		for _, g := range h.getMockGroups("", "") {
			gc := g
			patchGroupStore[g.ID] = &gc
		}
	}

	group, ok := patchGroupStore[id]
	if !ok {
		writeSCIMError(w, http.StatusNotFound, "group not found")
		return
	}

	// Apply each operation
	for _, op := range patch.Operations {
		opPath := strings.ToLower(strings.TrimSpace(op.Path))

		switch strings.ToLower(op.Op) {
		case "replace":
			if opPath == "displayname" {
				if name, ok := op.Value.(string); ok {
					group.DisplayName = name
				}
			} else if opPath == "members" {
				// Replace all members
				group.Members = valueToMembers(op.Value)
			}

		case "add":
			if opPath == "members" {
				newMembers := valueToMembers(op.Value)
				// Merge: add only members not already present
				existing := make(map[string]bool)
				for _, m := range group.Members {
					existing[m.Value] = true
				}
				for _, m := range newMembers {
					if !existing[m.Value] {
						group.Members = append(group.Members, m)
						existing[m.Value] = true
					}
				}
			}

		case "remove":
			if opPath == "members" || strings.HasPrefix(opPath, "members[") {
				// Remove members. If path has a filter like members[value eq "xxx"],
				// remove only matching members. Otherwise remove all.
				removeIDs := parseMemberFilter(op.Path)
				if len(removeIDs) > 0 {
					var filtered []SCIMGroupMember
					for _, m := range group.Members {
						if !removeIDs[m.Value] {
							filtered = append(filtered, m)
						}
					}
					group.Members = filtered
				} else {
					group.Members = nil
				}
			}
		}
	}

	writeSCIMJSON(w, http.StatusOK, group)
}

// valueToMembers converts a patch value (array of objects) to []SCIMGroupMember.
func valueToMembers(val any) []SCIMGroupMember {
	arr, ok := val.([]any)
	if !ok {
		return nil
	}
	var members []SCIMGroupMember
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		member := SCIMGroupMember{}
		if v, ok := m["value"].(string); ok {
			member.Value = v
		}
		if d, ok := m["display"].(string); ok {
			member.Display = d
		}
		if ref, ok := m["$ref"].(string); ok {
			member.Ref = ref
		}
		if t, ok := m["type"].(string); ok {
			member.Type = t
		}
		members = append(members, member)
	}
	return members
}

// parseMemberFilter extracts member IDs from a path like "members[value eq \"abc\"]".
func parseMemberFilter(path string) map[string]bool {
	result := make(map[string]bool)
	// Extract value between brackets
	idx := strings.Index(path, "[")
	if idx < 0 {
		return result
	}
	inner := path[idx+1:]
	endIdx := strings.Index(inner, "]")
	if endIdx >= 0 {
		inner = inner[:endIdx]
	}
	// Parse: value eq "abc" or value eq "abc" and value eq "def"
	parts := strings.Split(inner, " or ")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		// Also handle "and" conjunctions
		for _, p := range strings.Split(part, " and ") {
			p = strings.TrimSpace(p)
			if strings.HasPrefix(strings.ToLower(p), "value eq") {
				val := strings.Trim(strings.TrimSpace(p[len("value eq"):]), "\"")
				if val != "" {
					result[val] = true
				}
			}
		}
	}
	return result
}

func (h *Handler) deleteGroup(w http.ResponseWriter, r *http.Request, id string) {
	// Delete persisted group when DB is available (Task-C); 204 either way
	// per SCIM semantics.
	if pool := h.dbPool(r.Context()); pool != nil {
		_ = dbDeleteGroup(r.Context(), pool, id)
	}
	w.WriteHeader(http.StatusNoContent)
}

// getMockGroups returns SCIM groups from the database.
// SCIM Groups map to GGID roles. Members are users assigned to that role.
// Falls back to default groups if DB is unavailable.
func (h *Handler) getMockGroups(tenantID, filter string) []SCIMGroup {
	if h == nil || h.svc == nil {
		return defaultSCIMGroups(filter)
	}
	pool := h.svc.Pool()
	if pool == nil {
		return defaultSCIMGroups(filter)
	}

	// Query distinct roles from user_roles table to build SCIM groups.
	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil || tenantUUID == uuid.Nil {
		return defaultSCIMGroups(filter)
	}

	rows, err := pool.Query(context.Background(), `
		SELECT DISTINCT role_id, role_name FROM (
			SELECT ur.role_id::text as role_id, r.name as role_name
			FROM user_roles ur
			LEFT JOIN roles r ON r.id = ur.role_id
			WHERE ur.tenant_id = $1
		) t ORDER BY role_name
	`, tenantUUID)
	if err != nil {
		return defaultSCIMGroups(filter)
	}
	defer rows.Close()

	var groups []SCIMGroup
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			continue
		}
		if filter != "" && !strings.EqualFold(name, filter) {
			continue
		}
		groups = append(groups, SCIMGroup{
			Schemas:     []string{"urn:ietf:params:scim:schemas:core:2.0:Group"},
			ID:          id,
			DisplayName: name,
			Meta:        SCIMMeta{ResourceType: "Group", Location: "/scim/v2/Groups/" + id},
		})
	}

	if len(groups) == 0 {
		return defaultSCIMGroups(filter)
	}
	return groups
}

// defaultSCIMGroups returns static fallback groups when DB is unavailable.
func defaultSCIMGroups(filter string) []SCIMGroup {
	all := []SCIMGroup{
		{
			Schemas:     []string{"urn:ietf:params:scim:schemas:core:2.0:Group"},
			ID:          "role-admin-001",
			DisplayName: "Admin",
			Meta:        SCIMMeta{ResourceType: "Group", Location: "/scim/v2/Groups/role-admin-001"},
		},
		{
			Schemas:     []string{"urn:ietf:params:scim:schemas:core:2.0:Group"},
			ID:          "role-user-001",
			DisplayName: "User",
			Meta:        SCIMMeta{ResourceType: "Group", Location: "/scim/v2/Groups/role-user-001"},
		},
	}
	if filter != "" {
		var filtered []SCIMGroup
		for _, g := range all {
			if strings.EqualFold(g.DisplayName, filter) {
				filtered = append(filtered, g)
			}
		}
		return filtered
	}
	return all
}

func tenantFromRequest(r *http.Request) (*ggidtenant.Context, error) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		return nil, fmt.Errorf("missing tenant")
	}
	id, err := uuid.Parse(tenantID)
	if err != nil {
		return nil, fmt.Errorf("invalid tenant ID")
	}
	return &ggidtenant.Context{TenantID: id, IsolationLevel: ggidtenant.IsolationShared}, nil
}
