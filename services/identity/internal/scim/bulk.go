// Package scim implements SCIM 2.0 Bulk operations per RFC 7644 Section 3.7.
package scim

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ggid/ggid/services/identity/internal/domain"
	"github.com/google/uuid"
)

// BulkOperationRequest represents a single operation in a SCIM bulk request.
type BulkOperationRequest struct {
	Method   string          `json:"method"`   // POST, PUT, PATCH, DELETE
	Path     string          `json:"path"`     // /Users, /Users/{id}, /Groups, /Groups/{id}
	BulkID   string          `json:"bulkId"`   // client-generated correlation ID
	Data     json.RawMessage `json:"data"`     // resource body for POST/PUT/PATCH
	Version  string          `json:"version,omitempty"`
}

// BulkRequest is the top-level SCIM bulk request.
type BulkRequest struct {
	Schemas          []string              `json:"schemas"`
	Operations       []BulkOperationRequest `json:"Operations"`
	FailOnErrors     *int                  `json:"failOnErrors,omitempty"`
}

// BulkOperationResponse represents the result of one bulk operation.
type BulkOperationResponse struct {
	Location string          `json:"location,omitempty"`
	Method   string          `json:"method"`
	BulkID   string          `json:"bulkId,omitempty"`
	Version  string          `json:"version,omitempty"`
	Status   string          `json:"status"` // HTTP status as string
	Response json.RawMessage `json:"response,omitempty"` // error detail on failure
}

// BulkResponse is the top-level SCIM bulk response.
type BulkResponse struct {
	Schemas    []string                 `json:"schemas"`
	Operations []BulkOperationResponse  `json:"Operations"`
}

const (
	maxBulkOperations = 1000
	bulkSchema        = "urn:ietf:params:scim:api:messages:2.0:BulkRequest"
	bulkResponseSchema = "urn:ietf:params:scim:api:messages:2.0:BulkResponse"
)

// HandleBulk processes a SCIM bulk request (POST /scim/v2/Bulk).
func (h *Handler) HandleBulk(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var req BulkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeSCIMErrorWithType(w, http.StatusBadRequest, ScimTypeInvalidSyntax, "invalid bulk request body")
		return
	}

	if len(req.Operations) == 0 {
		writeSCIMErrorWithType(w, http.StatusBadRequest, ScimTypeInvalidValue, "no operations provided")
		return
	}

	if len(req.Operations) > maxBulkOperations {
		writeSCIMErrorWithType(w, http.StatusRequestEntityTooLarge, ScimTypeTooMany,
			fmt.Sprintf("exceeded maximum of %d operations", maxBulkOperations))
		return
	}

	failOnErrors := -1
	if req.FailOnErrors != nil {
		failOnErrors = *req.FailOnErrors
	}

	responses := make([]BulkOperationResponse, 0, len(req.Operations))
	errorCount := 0

	for _, op := range req.Operations {
		resp, err := h.executeBulkOp(ctx, op)
		if err != nil {
			errorCount++
			resp.Response, _ = json.Marshal(ErrorResponse{
				Schemas: []string{scimErrSchema},
				Detail:  err.Error(),
				Status:  resp.Status,
			})
		}
		responses = append(responses, resp)

		// Check failOnErrors threshold
		if failOnErrors > 0 && errorCount >= failOnErrors {
			break
		}
	}

	writeSCIMJSON(w, http.StatusOK, BulkResponse{
		Schemas:    []string{bulkResponseSchema},
		Operations: responses,
	})
}

// executeBulkOp processes a single bulk operation and returns a response with status.
func (h *Handler) executeBulkOp(ctx context.Context, op BulkOperationRequest) (BulkOperationResponse, error) {
	resp := BulkOperationResponse{
		Method: op.Method,
		BulkID: op.BulkID,
	}

	switch op.Method {
	case "POST":
		return h.bulkCreateUser(ctx, op)
	case "PUT":
		return h.bulkReplaceUser(ctx, op)
	case "PATCH":
		return h.bulkPatchUser(ctx, op)
	case "DELETE":
		return h.bulkDeleteUser(ctx, op)
	default:
		resp.Status = "400"
		return resp, fmt.Errorf("unsupported method %q", op.Method)
	}
}

func (h *Handler) bulkCreateUser(ctx context.Context, op BulkOperationRequest) (BulkOperationResponse, error) {
	var scimUser SCIMUser
	if err := json.Unmarshal(op.Data, &scimUser); err != nil {
		return BulkOperationResponse{Method: op.Method, BulkID: op.BulkID, Status: "400"},
			fmt.Errorf("invalid user data: %w", err)
	}

	email := ""
	if len(scimUser.Emails) > 0 {
		email = scimUser.Emails[0].Value
	}

	user, err := h.svc.CreateUser(ctx, &domain.CreateUserInput{
		Username:    scimUser.UserName,
		Email:       email,
		Password:    "TempPass123!",
		DisplayName: scimUser.DisplayName,
		ExternalID:  scimUser.ExternalID,
	})
	if err != nil {
		return BulkOperationResponse{Method: op.Method, BulkID: op.BulkID, Status: "409"}, err
	}

	return BulkOperationResponse{
		Method:   op.Method,
		BulkID:   op.BulkID,
		Status:   "201",
		Location: "/scim/v2/Users/" + user.ID.String(),
		Version:  fmt.Sprintf("W/\"%d\"", user.UpdatedAt.UnixNano()),
	}, nil
}

func (h *Handler) bulkReplaceUser(ctx context.Context, op BulkOperationRequest) (BulkOperationResponse, error) {
	userID, err := extractIDFromPath(op.Path)
	if err != nil {
		return BulkOperationResponse{Method: op.Method, BulkID: op.BulkID, Status: "400"}, err
	}

	var scimUser SCIMUser
	if err := json.Unmarshal(op.Data, &scimUser); err != nil {
		return BulkOperationResponse{Method: op.Method, BulkID: op.BulkID, Status: "400"},
			fmt.Errorf("invalid user data: %w", err)
	}

	input := &domain.UpdateUserInput{
		DisplayName: &scimUser.DisplayName,
	}
	user, err := h.svc.UpdateUser(ctx, userID, input)
	if err != nil {
		return BulkOperationResponse{Method: op.Method, BulkID: op.BulkID, Status: "404"}, err
	}

	return BulkOperationResponse{
		Method:   op.Method,
		BulkID:   op.BulkID,
		Status:   "200",
		Location: "/scim/v2/Users/" + user.ID.String(),
		Version:  fmt.Sprintf("W/\"%d\"", user.UpdatedAt.UnixNano()),
	}, nil
}

func (h *Handler) bulkPatchUser(ctx context.Context, op BulkOperationRequest) (BulkOperationResponse, error) {
	userID, err := extractIDFromPath(op.Path)
	if err != nil {
		return BulkOperationResponse{Method: op.Method, BulkID: op.BulkID, Status: "400"}, err
	}

	var patchReq PatchRequest
	if err := json.Unmarshal(op.Data, &patchReq); err != nil {
		return BulkOperationResponse{Method: op.Method, BulkID: op.BulkID, Status: "400"},
			fmt.Errorf("invalid patch data: %w", err)
	}

	user, err := h.svc.GetUser(ctx, userID)
	if err != nil {
		return BulkOperationResponse{Method: op.Method, BulkID: op.BulkID, Status: "404"}, err
	}

	// Apply patch operations to user attributes
	attrs := scimUserToAttrs(toSCIMUser(user))
	patched, err := ApplyPatch(attrs, patchReq.Operations)
	if err != nil {
		return BulkOperationResponse{Method: op.Method, BulkID: op.BulkID, Status: "400"}, err
	}

	// Write back display name if changed
	patchedUser := PatchedAttrsToSCIMUser(patched)
	if patchedUser.DisplayName != "" && patchedUser.DisplayName != user.DisplayName {
		dn := patchedUser.DisplayName
		user, _ = h.svc.UpdateUser(ctx, userID, &domain.UpdateUserInput{DisplayName: &dn})
	}

	return BulkOperationResponse{
		Method:   op.Method,
		BulkID:   op.BulkID,
		Status:   "200",
		Location: "/scim/v2/Users/" + userID.String(),
	}, nil
}

func (h *Handler) bulkDeleteUser(ctx context.Context, op BulkOperationRequest) (BulkOperationResponse, error) {
	userID, err := extractIDFromPath(op.Path)
	if err != nil {
		return BulkOperationResponse{Method: op.Method, BulkID: op.BulkID, Status: "400"}, err
	}

	if err := h.svc.DeleteUser(ctx, userID); err != nil {
		return BulkOperationResponse{Method: op.Method, BulkID: op.BulkID, Status: "404"}, err
	}

	return BulkOperationResponse{
		Method: op.Method,
		BulkID: op.BulkID,
		Status: "204",
	}, nil
}

// extractIDFromPath extracts the UUID from paths like "/Users/{id}" or "/Groups/{id}".
func extractIDFromPath(path string) (uuid.UUID, error) {
	// Path format: /Users/550e8400-e29b-41d4-a716-446655440000
	trimmed := path
	for len(trimmed) > 0 && (trimmed[0] == '/') {
		trimmed = trimmed[1:]
	}
	// Split by /
	segments := splitPath(trimmed)
	if len(segments) < 2 {
		return uuid.Nil, fmt.Errorf("invalid path %q: expected /ResourceType/{id}", path)
	}
	return uuid.Parse(segments[1])
}

func splitPath(s string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '/' {
			if i > start {
				parts = append(parts, s[start:i])
			}
			start = i + 1
		}
	}
	if start < len(s) {
		parts = append(parts, s[start:])
	}
	return parts
}

// scimUserToAttrs converts a SCIMUser to a map for patch processing.
func scimUserToAttrs(u SCIMUser) map[string]any {
	attrs := map[string]any{
		"id":          u.ID,
		"userName":    u.UserName,
		"displayName": u.DisplayName,
		"active":      u.Active,
	}
	if u.ExternalID != "" {
		attrs["externalId"] = u.ExternalID
	}
	if u.Name.GivenName != "" || u.Name.FamilyName != "" {
		nameMap := map[string]any{}
		if u.Name.GivenName != "" {
			nameMap["givenName"] = u.Name.GivenName
		}
		if u.Name.FamilyName != "" {
			nameMap["familyName"] = u.Name.FamilyName
		}
		attrs["name"] = nameMap
	}
	if len(u.Emails) > 0 {
		emails := make([]any, len(u.Emails))
		for i, e := range u.Emails {
			emails[i] = map[string]any{
				"value":   e.Value,
				"type":    e.Type,
				"primary": e.Primary,
			}
		}
		attrs["emails"] = emails
	}
	return attrs
}
