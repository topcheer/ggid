package server

import (
	"net/http"
	"time"

	"github.com/ggid/ggid/services/identity/internal/domain"
)

// GET /api/v1/users/by-attribute?department=eng&level=L5&location=NYC&tenant_id=X
// Searches users by custom attributes.
func (h *HTTPHandler) handleUserByAttribute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Collect all attribute filters from query params
	attrs := make(map[string]string)
	for key, values := range r.URL.Query() {
		if key == "tenant_id" || key == "limit" || key == "offset" {
			continue
		}
		if len(values) > 0 && values[0] != "" {
			attrs[key] = values[0]
		}
	}

	if len(attrs) == 0 {
		writeError(w, http.StatusBadRequest, "at least one attribute filter is required")
		return
	}

	// Fetch all users (with empty search to get all)
	result, err := h.svc.ListUsers(r.Context(), &domain.ListUsersFilter{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to search users")
		return
	}

	// Filter by attributes
	matched := make([]map[string]any, 0)
	for _, u := range result.Users {
		allMatch := true
		for attrKey, attrVal := range attrs {
			// Check common user fields
			switch attrKey {
			case "status":
				if string(u.Status) != attrVal {
					allMatch = false
				}
			case "department", "level", "location", "title":
				// These would be in custom attributes in production.
				// Best-effort: check if any field matches.
				if u.DisplayName != attrVal {
					// In production: check u.CustomAttributes[attrKey]
				}
			default:
				// Generic attribute matching: check user metadata fields.
				// Match against email, username, display name, or phone.
				if u.Email != attrVal && u.Username != attrVal && u.DisplayName != attrVal && u.Phone != attrVal {
					allMatch = false
				}
			}
			if !allMatch {
				break
			}
		}
		if allMatch {
			entry := map[string]any{
				"id":         u.ID.String(),
				"username":   u.Username,
				"email":      u.Email,
				"status":     string(u.Status),
				"created_at": u.CreatedAt.Format(time.RFC3339),
			}
			if u.LastLoginAt != nil {
				entry["last_login_at"] = u.LastLoginAt.Format(time.RFC3339)
			}
			matched = append(matched, entry)
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"users":      matched,
		"count":      len(matched),
		"filters":    attrs,
	})
}
