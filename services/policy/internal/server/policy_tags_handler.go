package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// policyTag represents a version tag on a policy.
type policyTag struct {
	ID         string `json:"id"`
	PolicyID   string `json:"policy_id"`
	Version    int    `json:"version"`
	Tag        string `json:"tag"`        // release, candidate, draft, deprecated
	TaggedBy   string `json:"tagged_by"`
	TaggedAt   string `json:"tagged_at"`
	Note       string `json:"note,omitempty"`
}

var policyTagStore = struct {
	sync.RWMutex
	tags []policyTag
}{tags: []policyTag{
	{ID: "tag-1", PolicyID: "pol-001", Version: 3, Tag: "release", TaggedBy: "admin", TaggedAt: time.Now().UTC().Add(-48 * time.Hour).Format(time.RFC3339)},
	{ID: "tag-2", PolicyID: "pol-001", Version: 4, Tag: "candidate", TaggedBy: "admin", TaggedAt: time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339)},
	{ID: "tag-3", PolicyID: "pol-002", Version: 1, Tag: "draft", TaggedBy: "analyst", TaggedAt: time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339)},
	{ID: "tag-4", PolicyID: "pol-003", Version: 2, Tag: "deprecated", TaggedBy: "admin", TaggedAt: time.Now().UTC().Add(-7 * 24 * time.Hour).Format(time.RFC3339), Note: "Replaced by pol-004"},
}}

// POST /api/v1/policies/{id}/tags — add a tag to a policy version
// GET  /api/v1/policies/tags?tag=release&policy_id=X — filter tags
func (s *HTTPServer) handlePolicyTags(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		// Extract policy ID from path
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/policies/")
		parts := strings.SplitN(path, "/", 2)
		if len(parts) < 2 || parts[1] != "tags" {
			writeJSONError(w, http.StatusBadRequest, "invalid path")
			return
		}
		policyID := parts[0]
		if policyID == "" {
			writeJSONError(w, http.StatusBadRequest, "policy ID is required")
			return
		}

		var req struct {
			Version  int    `json:"version"`
			Tag      string `json:"tag"`
			TaggedBy string `json:"tagged_by"`
			Note     string `json:"note"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		validTags := map[string]bool{"release": true, "candidate": true, "draft": true, "deprecated": true}
		if !validTags[req.Tag] {
			writeJSONError(w, http.StatusBadRequest, "tag must be one of: release, candidate, draft, deprecated")
			return
		}
		if req.TaggedBy == "" {
			req.TaggedBy = "system"
		}

		// Remove existing tag of same type for this policy (a policy can only have one tag of each type)
		policyTagStore.Lock()
		filtered := policyTagStore.tags[:0]
		for _, t := range policyTagStore.tags {
			if !(t.PolicyID == policyID && t.Tag == req.Tag) {
				filtered = append(filtered, t)
			}
		}
		policyTagStore.tags = filtered

		tag := policyTag{
			ID:       uuid.New().String(),
			PolicyID: policyID,
			Version:  req.Version,
			Tag:      req.Tag,
			TaggedBy: req.TaggedBy,
			TaggedAt: time.Now().UTC().Format(time.RFC3339),
			Note:     req.Note,
		}
		policyTagStore.tags = append(policyTagStore.tags, tag)
		policyTagStore.Unlock()

		writeJSON(w, http.StatusCreated, tag)

	case http.MethodGet:
		tagFilter := r.URL.Query().Get("tag")
		policyFilter := r.URL.Query().Get("policy_id")

		policyTagStore.RLock()
		result := []policyTag{}
		for _, t := range policyTagStore.tags {
			if tagFilter != "" && t.Tag != tagFilter {
				continue
			}
			if policyFilter != "" && t.PolicyID != policyFilter {
				continue
			}
			result = append(result, t)
		}
		policyTagStore.RUnlock()

		// Summary by tag type
		tagCounts := map[string]int{}
		for _, t := range result {
			tagCounts[t.Tag]++
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"tags":       result,
			"total":      len(result),
			"by_tag":     tagCounts,
		})

	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
