// Package scim implements SCIM 2.0 endpoints for enterprise HR system integration.
// Spec: https://datatracker.ietf.org/doc/html/rfc7643
package scim

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	ggidtenant "github.com/ggid/ggid/pkg/tenant"
	"github.com/ggid/ggid/services/identity/internal/domain"
	"github.com/ggid/ggid/services/identity/internal/service"
	"github.com/google/uuid"
)

// Handler implements SCIM 2.0 HTTP endpoints.
type Handler struct {
	svc *service.IdentityService
}

// NewHandler creates a new SCIM handler.
func NewHandler(svc *service.IdentityService) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes registers SCIM endpoints on the given mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/scim/v2/Users", h.handleUsersCollection)
	mux.HandleFunc("/scim/v2/Users/", h.handleUserResource)
	mux.HandleFunc("/scim/v2/Groups", h.handleGroupsCollection)
	mux.HandleFunc("/scim/v2/Groups/", h.HandleGroupResource)
	mux.HandleFunc("/scim/v2/ServiceProviderConfig", h.handleServiceProviderConfig)
	mux.HandleFunc("/scim/v2/ResourceTypes", h.handleResourceTypes)
}

// --- SCIM Schema Types ---

// SCIMUser is the RFC 7643 User resource representation.
type SCIMUser struct {
	Schemas      []string        `json:"schemas"`
	ID           string          `json:"id"`
	ExternalID   string          `json:"externalId,omitempty"`
	UserName     string          `json:"userName"`
	Name         SCIMName        `json:"name"`
	DisplayName  string          `json:"displayName,omitempty"`
	Emails       []SCIMEmail     `json:"emails,omitempty"`
	PhoneNumbers []SCIMPhone     `json:"phoneNumbers,omitempty"`
	Active       bool            `json:"active"`
	Meta         SCIMMeta        `json:"meta"`
}

type SCIMName struct {
	GivenName  string `json:"givenName,omitempty"`
	FamilyName string `json:"familyName,omitempty"`
}

type SCIMEmail struct {
	Value   string `json:"value"`
	Type    string `json:"type,omitempty"`
	Primary bool   `json:"primary,omitempty"`
}

type SCIMPhone struct {
	Value string `json:"value"`
	Type  string `json:"type,omitempty"`
}

type SCIMMeta struct {
	ResourceType  string `json:"resourceType"`
	Location      string `json:"location,omitempty"`
	Created       *string `json:"created,omitempty"`
	LastModified  *string `json:"lastModified,omitempty"`
	Version       string  `json:"version,omitempty"`
}

// ListResponse is the standard SCIM paginated response.
type ListResponse struct {
	Schemas      []string    `json:"schemas"`
	TotalResults int         `json:"totalResults"`
	ItemsPerPage int         `json:"itemsPerPage"`
	StartIndex   int         `json:"startIndex"`
	Resources    []SCIMUser  `json:"Resources"`
}

// ErrorResponse is the SCIM standard error format.
type ErrorResponse struct {
	Schemas []string `json:"schemas"`
	Detail  string   `json:"detail"`
	Status  string   `json:"status"`
}

// --- Helpers ---

const (
	scimUserSchema = "urn:ietf:params:scim:schemas:core:2.0:User"
	scimListSchema = "urn:ietf:params:scim:api:messages:2.0:ListResponse"
	scimErrSchema  = "urn:ietf:params:scim:api:messages:2.0:Error"
)

func writeSCIMJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/scim+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeSCIMError(w http.ResponseWriter, status int, detail string) {
	writeSCIMJSON(w, status, ErrorResponse{
		Schemas: []string{scimErrSchema},
		Detail:  detail,
		Status:  strconv.Itoa(status),
	})
}

func injectTenant(r *http.Request) (bool, context.Context) {
	tenantIDStr := r.Header.Get("X-Tenant-ID")
	if tenantIDStr == "" {
		return false, nil
	}
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return false, nil
	}
	tc := &ggidtenant.Context{
		TenantID:       tenantID,
		IsolationLevel: ggidtenant.IsolationShared,
	}
	return true, ggidtenant.WithContext(r.Context(), tc)
}

// toSCIMUser converts a domain User to SCIM format.
func toSCIMUser(u *domain.User) SCIMUser {
	created := formatSCIMTime(u.CreatedAt)
	lastMod := formatSCIMTime(u.UpdatedAt)
	version := fmt.Sprintf("W/\"%d\"", u.UpdatedAt.UnixNano())
	return SCIMUser{
		Schemas:     []string{scimUserSchema},
		ID:          u.ID.String(),
		UserName:    u.Username,
		DisplayName: u.DisplayName,
		Name: SCIMName{
			GivenName:  u.DisplayName,
		},
		Emails: []SCIMEmail{
			{Value: u.Email, Type: "work", Primary: true},
		},
		Active: u.Status == domain.UserStatusActive,
		Meta: SCIMMeta{
			ResourceType: "User",
			Location:     "/scim/v2/Users/" + u.ID.String(),
			Created:      &created,
			LastModified: &lastMod,
			Version:      version,
		},
	}
}

// formatSCIMTime formats a time.Time as RFC 3339 for SCIM meta timestamps.
func formatSCIMTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

// --- Handlers ---

func (h *Handler) handleUsersCollection(w http.ResponseWriter, r *http.Request) {
	ok, ctx := injectTenant(r)
	if !ok {
		writeSCIMError(w, http.StatusBadRequest, "missing or invalid X-Tenant-ID header")
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.listUsers(ctx, w, r)
	case http.MethodPost:
		h.createUser(ctx, w, r)
	default:
		writeSCIMError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) handleUserResource(w http.ResponseWriter, r *http.Request) {
	ok, ctx := injectTenant(r)
	if !ok {
		writeSCIMError(w, http.StatusBadRequest, "missing or invalid X-Tenant-ID header")
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/scim/v2/Users/")
	userID, err := uuid.Parse(idStr)
	if err != nil {
		writeSCIMError(w, http.StatusBadRequest, "invalid user ID")
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getUser(ctx, w, r, userID)
	case http.MethodPut:
		h.replaceUser(ctx, w, r, userID)
	case http.MethodPatch:
		h.patchUser(ctx, w, r, userID)
	case http.MethodDelete:
		h.deleteUser(ctx, w, r, userID)
	default:
		writeSCIMError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) listUsers(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	startIndex, _ := strconv.Atoi(r.URL.Query().Get("startIndex"))
	if startIndex <= 0 {
		startIndex = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("count"))
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	offset := startIndex - 1
	result, err := h.svc.ListUsers(ctx, &domain.ListUsersFilter{
		PageSize: pageSize,
		Offset:   offset,
	})
	if err != nil {
		writeSCIMError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resources := make([]SCIMUser, 0, len(result.Users))
	for _, u := range result.Users {
		resources = append(resources, toSCIMUser(u))
	}

	writeSCIMJSON(w, http.StatusOK, ListResponse{
		Schemas:      []string{scimListSchema},
		TotalResults: result.Total,
		ItemsPerPage: pageSize,
		StartIndex:   startIndex,
		Resources:    resources,
	})
}

func (h *Handler) createUser(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var scimUser SCIMUser
	if err := json.NewDecoder(r.Body).Decode(&scimUser); err != nil {
		writeSCIMError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	email := ""
	if len(scimUser.Emails) > 0 {
		email = scimUser.Emails[0].Value
	}

	user, err := h.svc.CreateUser(ctx, &domain.CreateUserInput{
		Username:    scimUser.UserName,
		Email:       email,
		Password:    "TempPass123!", // SCIM provisioned users get temp password
		DisplayName: scimUser.DisplayName,
	})
	if err != nil {
		writeSCIMError(w, http.StatusConflict, err.Error())
		return
	}

	resp := toSCIMUser(user)
	writeSCIMJSON(w, http.StatusCreated, resp)
}

func (h *Handler) getUser(ctx context.Context, w http.ResponseWriter, r *http.Request, userID uuid.UUID) {
	user, err := h.svc.GetUser(ctx, userID)
	if err != nil {
		writeSCIMError(w, http.StatusNotFound, "user not found")
		return
	}
	writeSCIMJSON(w, http.StatusOK, toSCIMUser(user))
}

func (h *Handler) replaceUser(ctx context.Context, w http.ResponseWriter, r *http.Request, userID uuid.UUID) {
	var scimUser SCIMUser
	if err := json.NewDecoder(r.Body).Decode(&scimUser); err != nil {
		writeSCIMError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	input := &domain.UpdateUserInput{
		DisplayName: &scimUser.DisplayName,
	}

	user, err := h.svc.UpdateUser(ctx, userID, input)
	if err != nil {
		writeSCIMError(w, http.StatusNotFound, err.Error())
		return
	}

	// Handle active/inactive
	if !scimUser.Active {
		user, _ = h.svc.LockUser(ctx, userID)
	} else {
		user, _ = h.svc.UnlockUser(ctx, userID)
	}

	writeSCIMJSON(w, http.StatusOK, toSCIMUser(user))
}

// SCIMPatchOp represents a single PATCH operation (RFC 7644 Section 3.5.2).
type SCIMPatchRequest struct {
	Schemas    []string       `json:"schemas"`
	Operations []SCIMPatchOp  `json:"Operations"`
}

type SCIMPatchOp struct {
	Op    string          `json:"op"`     // add, replace, remove
	Path  string          `json:"path"`   // attribute path (e.g. "displayName", "emails[type eq \"work\"]")
	Value json.RawMessage `json:"value"`  // value to set (for add/replace)
}

func (h *Handler) patchUser(ctx context.Context, w http.ResponseWriter, r *http.Request, userID uuid.UUID) {
	var patchReq SCIMPatchRequest
	if err := json.NewDecoder(r.Body).Decode(&patchReq); err != nil {
		writeSCIMError(w, http.StatusBadRequest, "invalid PATCH request body")
		return
	}

	user, err := h.svc.GetUser(ctx, userID)
	if err != nil {
		writeSCIMError(w, http.StatusNotFound, "user not found")
		return
	}

	// Track changes to apply.
	displayName := user.DisplayName
	active := user.Status == domain.UserStatusActive

	for _, op := range patchReq.Operations {
		opLower := strings.ToLower(op.Op)
		path := strings.ToLower(op.Path)

		switch {
		case path == "displayname" || path == "name.givenname":
			if opLower == "replace" || opLower == "add" {
				var val string
				if err := json.Unmarshal(op.Value, &val); err == nil {
					displayName = val
				}
			} else if opLower == "remove" {
				displayName = ""
			}

		case path == "active":
			if opLower == "replace" {
				var val bool
				if err := json.Unmarshal(op.Value, &val); err == nil {
					active = val
				}
			}
		}
	}

	// Apply updates.
	input := &domain.UpdateUserInput{
		DisplayName: &displayName,
	}

	updatedUser, err := h.svc.UpdateUser(ctx, userID, input)
	if err != nil {
		writeSCIMError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Handle active/inactive toggle.
	if active && updatedUser.Status != domain.UserStatusActive {
		updatedUser, _ = h.svc.UnlockUser(ctx, userID)
	} else if !active && updatedUser.Status == domain.UserStatusActive {
		updatedUser, _ = h.svc.LockUser(ctx, userID)
	}

	writeSCIMJSON(w, http.StatusOK, toSCIMUser(updatedUser))
}

func (h *Handler) deleteUser(ctx context.Context, w http.ResponseWriter, r *http.Request, userID uuid.UUID) {
	if err := h.svc.DeleteUser(ctx, userID); err != nil {
		writeSCIMError(w, http.StatusNotFound, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Skeleton endpoints ---
// handleGroupsCollection is implemented in groups.go

func (h *Handler) handleServiceProviderConfig(w http.ResponseWriter, r *http.Request) {
	writeSCIMJSON(w, http.StatusOK, map[string]any{
		"schemas":       []string{"urn:ietf:params:scim:schemas:core:2.0:ServiceProviderConfig"},
		"patch":         map[string]any{"supported": true},
		"bulk":          map[string]any{"supported": false, "maxOperations": 0, "maxPayloadSize": 0},
		"filter":        map[string]any{"supported": true, "maxResults": 100},
		"changePassword": map[string]any{"supported": true},
		"sort":          map[string]any{"supported": true},
		"etag":          map[string]any{"supported": false},
		"authenticationSchemes": []map[string]any{
			{
				"name":        "OAuth 2.0 Bearer",
				"description": "OAuth 2.0 Bearer Token",
				"type":        "oauthbearertoken",
			},
		},
	})
}

func (h *Handler) handleResourceTypes(w http.ResponseWriter, r *http.Request) {
	writeSCIMJSON(w, http.StatusOK, []map[string]any{
		{
			"schemas":      []string{"urn:ietf:params:scim:schemas:core:2.0:ResourceType"},
			"id":           "User",
			"name":         "User",
			"endpoint":     "/Users",
			"description":  "User Account",
			"schema":       "urn:ietf:params:scim:schemas:core:2.0:User",
		},
		{
			"schemas":      []string{"urn:ietf:params:scim:schemas:core:2.0:ResourceType"},
			"id":           "Group",
			"name":         "Group",
			"endpoint":     "/Groups",
			"description":  "Group",
			"schema":       "urn:ietf:params:scim:schemas:core:2.0:Group",
		},
	})
}
