package server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/ggid/ggid/services/identity/internal/domain"
)

// GET /api/v1/users/search?q=X&status=active&org=Y&role=Z&last_login_before=W
func (h *HTTPHandler) handleUserSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	query := r.URL.Query().Get("q")
	status := r.URL.Query().Get("status")
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 20
	offset := 0
	if limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	if offsetStr != "" {
		if n, err := strconv.Atoi(offsetStr); err == nil && n >= 0 {
			offset = n
		}
	}

	searchFilter := &domain.ListUsersFilter{
		Search: query,
	}

	if status != "" {
		st := domain.UserStatus(status)
		searchFilter.Status = &st
	}

	if lastLoginBefore := r.URL.Query().Get("last_login_before"); lastLoginBefore != "" {
		if t, err := time.Parse(time.RFC3339, lastLoginBefore); err == nil {
			searchFilter.CreatedBefore = &t
		}
	}

	result, err := h.svc.ListUsers(r.Context(), searchFilter)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to search users")
		return
	}

	users := result.Users
	total := len(users)
	end := offset + limit
	if end > total {
		end = total
	}
	if offset > total {
		offset = total
	}
	page := users[offset:end]

	userList := make([]map[string]any, 0, len(page))
	for _, u := range page {
		entry := map[string]any{
			"id":         u.ID.String(),
			"username":   u.Username,
			"email":      u.Email,
			"status":     string(u.Status),
			"created_at": u.CreatedAt.Format(time.RFC3339),
		}
		if u.LastLoginAt != nil {
			entry["last_login_at"] = u.LastLoginAt.Format(time.RFC3339)
		} else {
			entry["last_login_at"] = ""
		}
		userList = append(userList, entry)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"users":  userList,
		"count":  len(userList),
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}
