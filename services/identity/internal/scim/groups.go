package scim

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

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

	_ = tc // tenant scoping in production

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
	groupID := pathParts[2]

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
	groups := h.getMockGroups("", "")
	for _, g := range groups {
		if g.ID == id {
			writeSCIMJSON(w, http.StatusOK, g)
			return
		}
	}
	writeSCIMError(w, http.StatusNotFound, "group not found")
}

func (h *Handler) patchGroup(w http.ResponseWriter, r *http.Request, id string) {
	var patch struct {
		Operations []struct {
			Op    string `json:"op"`    // "replace", "add", "remove"
			Path  string `json:"path"`  // e.g. "displayName" or "members"
			Value any    `json:"value"` // new value
		} `json:"Operations"`
	}
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		writeSCIMError(w, http.StatusBadRequest, "invalid PATCH body")
		return
	}

	// In production, apply operations to the persisted group.
	// For now, return success with a no-content response.
	writeSCIMJSON(w, http.StatusOK, map[string]any{
		"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
		"status":  "200",
	})
}

func (h *Handler) deleteGroup(w http.ResponseWriter, r *http.Request, id string) {
	// In production, delete from DB. Return 204 No Content.
	w.WriteHeader(http.StatusNoContent)
}

// getMockGroups returns sample SCIM groups for testing.
// In production this would query the role/user-mapping tables.
func (h *Handler) getMockGroups(tenantID, filter string) []SCIMGroup {
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
