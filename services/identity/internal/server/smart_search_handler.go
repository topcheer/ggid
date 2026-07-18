package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/ggid/ggid/services/identity/internal/domain"
)

// searchResult represents a ranked search result.
type searchResult struct {
	ID          string  `json:"id"`
	Username    string  `json:"username"`
	Email       string  `json:"email"`
	DisplayName string  `json:"display_name"`
	Status      string  `json:"status"`
	MatchType   string  `json:"match_type"` // username_exact, username_prefix, email_domain, name_contains, phone_match
	Score       float64 `json:"score"`
}

// GET /api/v1/users/smart-search?q=X&limit=10
// Ranked search across username, email, display_name, phone.
func (h *HTTPHandler) handleSmartSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		writeJSONError(w, http.StatusBadRequest, "q query parameter is required")
		return
	}

	limit := 10
	if l := r.URL.Query().Get("limit"); l != "" {
		var n int
		_, _ = fmt.Sscanf(l, "%d", &n)
		if n > 0 && n <= 100 {
			limit = n
		}
	}

	queryLower := strings.ToLower(query)

	// Use the identity service to list users, then score them
	ctx, ok := injectTenant(r)
	if !ok {
		writeJSONError(w, http.StatusBadRequest, "missing or invalid X-Tenant-ID header")
		return
	}

	result, err := h.svc.ListUsers(ctx, &domain.ListUsersFilter{PageSize: 1000})
	if err != nil || result == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"query":   query,
			"results": []searchResult{},
			"total":   0,
		})
		return
	}

	var scored []searchResult
	for _, u := range result.Users {
		var matchType string
		var score float64

		usernameLower := strings.ToLower(u.Username)
		emailLower := strings.ToLower(u.Email)
		displayLower := strings.ToLower(u.DisplayName)

		// Exact username match — highest score
		if usernameLower == queryLower {
			matchType = "username_exact"
			score = 100.0
		} else if strings.HasPrefix(usernameLower, queryLower) {
			matchType = "username_prefix"
			score = 85.0
		} else if strings.Contains(usernameLower, queryLower) {
			matchType = "username_contains"
			score = 70.0
		} else if emailLower == queryLower {
			matchType = "email_exact"
			score = 95.0
		} else if strings.HasPrefix(emailLower, queryLower) {
			matchType = "email_prefix"
			score = 80.0
		} else if strings.Contains(emailLower, queryLower) {
			// Check if matching email domain
			if strings.HasPrefix(queryLower, "@") {
				matchType = "email_domain"
				score = 65.0
			} else {
				matchType = "email_contains"
				score = 60.0
			}
		} else if displayLower != "" && strings.Contains(displayLower, queryLower) {
			matchType = "name_contains"
			score = 55.0
		} else if u.Phone != "" && strings.Contains(u.Phone, query) {
			matchType = "phone_match"
			score = 50.0
		} else {
			continue // No match
		}

		// Boost score for exact word boundary matches
		if matchType == "name_contains" || matchType == "username_contains" {
			words := strings.Fields(displayLower)
			for _, word := range words {
				if word == queryLower {
					score += 10
					break
				}
			}
		}

		// Penalty for inactive users
		if string(u.Status) != "active" {
			score -= 15
		}

		scored = append(scored, searchResult{
			ID:          u.ID.String(),
			Username:    u.Username,
			Email:       u.Email,
			DisplayName: u.DisplayName,
			Status:      string(u.Status),
			MatchType:   matchType,
			Score:       score,
		})
	}

	// Sort by score descending (simple bubble sort for small lists)
	for i := 0; i < len(scored); i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].Score > scored[i].Score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	// Apply limit
	if len(scored) > limit {
		scored = scored[:limit]
	}

	// Group by match type for summary
	matchTypeSummary := map[string]int{}
	for _, s := range scored {
		matchTypeSummary[s.MatchType]++
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"query":              query,
		"results":            scored,
		"total":              len(scored),
		"match_type_summary": matchTypeSummary,
	})
}