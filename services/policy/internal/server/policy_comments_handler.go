package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// policyComment represents a single comment in a policy collaboration thread.
type policyComment struct {
	ID        string `json:"id"`
	PolicyID  string `json:"policy_id"`
	AuthorID  string `json:"author_id"`
	Body      string `json:"body"`
	ParentID  string `json:"parent_id,omitempty"` // for threaded replies
	CreatedAt string `json:"created_at"`
	Resolved  bool   `json:"resolved"`
}

var policyCommentStore = struct {
	sync.RWMutex
	data map[string][]policyComment // policyID → comments
}{data: make(map[string][]policyComment)}

// POST /api/v1/policies/{id}/comments — add a comment
// GET  /api/v1/policies/{id}/comments — list comments (threaded)
// PATCH /api/v1/policies/{id}/comments/{commentID}/resolve — mark resolved
func (s *HTTPServer) handlePolicyComments(w http.ResponseWriter, r *http.Request) {
	// Extract policy ID and optional action from path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/policies/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) < 2 || parts[1] != "comments" && !strings.HasPrefix(parts[1], "comments") {
		writeJSONError(w, http.StatusBadRequest, "invalid path")
		return
	}
	policyID := parts[0]
	if policyID == "" {
		writeJSONError(w, http.StatusBadRequest, "policy ID is required")
		return
	}

	// Check for resolve action: /comments/{commentID}/resolve
	commentAction := ""
	if strings.HasPrefix(parts[1], "comments/") {
		subParts := strings.SplitN(parts[1], "/", 3)
		if len(subParts) >= 3 && subParts[2] == "resolve" {
			commentAction = subParts[1] // commentID
		}
	}

	switch {
	case commentAction != "" && r.Method == http.MethodPatch:
		policyCommentStore.Lock()
		comments := policyCommentStore.data[policyID]
		found := false
		for i := range comments {
			if comments[i].ID == commentAction {
				comments[i].Resolved = true
				found = true
				break
			}
		}
		policyCommentStore.Unlock()
		if !found {
			writeJSONError(w, http.StatusNotFound, "comment not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"resolved": true, "comment_id": commentAction})

	case r.Method == http.MethodPost:
		var req struct {
			AuthorID string `json:"author_id"`
			Body     string `json:"body"`
			ParentID string `json:"parent_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if req.AuthorID == "" || req.Body == "" {
			writeJSONError(w, http.StatusBadRequest, "author_id and body are required")
			return
		}

		comment := policyComment{
			ID:        uuid.New().String(),
			PolicyID:  policyID,
			AuthorID:  req.AuthorID,
			Body:      req.Body,
			ParentID:  req.ParentID,
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
		}

		policyCommentStore.Lock()
		policyCommentStore.data[policyID] = append(policyCommentStore.data[policyID], comment)
		policyCommentStore.Unlock()

		writeJSON(w, http.StatusCreated, comment)

	case r.Method == http.MethodGet:
		policyCommentStore.RLock()
		raw := policyCommentStore.data[policyID]
		comments := make([]policyComment, len(raw))
		copy(comments, raw)
		policyCommentStore.RUnlock()

		// Build threaded view
		threads := buildCommentThreads(comments)

		writeJSON(w, http.StatusOK, map[string]any{
			"policy_id":    policyID,
			"comments":     comments,
			"threads":      threads,
			"total":        len(comments),
			"unresolved":   countUnresolved(comments),
		})

	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func buildCommentThreads(comments []policyComment) []map[string]any {
	byID := map[string]*policyComment{}
	for i := range comments {
		byID[comments[i].ID] = &comments[i]
	}

	var roots []map[string]any
	replies := map[string][]map[string]any{} // parentID → replies

	for i := range comments {
		c := &comments[i]
		if c.ParentID == "" {
			roots = append(roots, map[string]any{
				"id": c.ID, "author_id": c.AuthorID, "body": c.Body,
				"created_at": c.CreatedAt, "resolved": c.Resolved,
				"replies": []map[string]any{},
			})
		} else {
			replies[c.ParentID] = append(replies[c.ParentID], map[string]any{
				"id": c.ID, "author_id": c.AuthorID, "body": c.Body,
				"created_at": c.CreatedAt, "resolved": c.Resolved,
			})
		}
	}
	// Attach replies to roots
	for _, root := range roots {
		rootID := root["id"].(string)
		if r, ok := replies[rootID]; ok {
			root["replies"] = r
		}
	}
	return roots
}

func countUnresolved(comments []policyComment) int {
	count := 0
	for _, c := range comments {
		if !c.Resolved {
			count++
		}
	}
	return count
}
