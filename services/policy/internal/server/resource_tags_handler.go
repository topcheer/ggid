package httpserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type ResourceTag struct {
	ID        string            `json:"id"`
	Resource  string            `json:"resource"`
	Tags      map[string]string `json:"tags"`
	CreatedAt time.Time         `json:"created_at"`
}

func (s *HTTPServer) handleResourceTags(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req struct {
			Resource string            `json:"resource"`
			Tags     map[string]string `json:"tags"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		if req.Resource == "" || len(req.Tags) == 0 {
			writeJSONError(w, http.StatusBadRequest, "resource and tags required")
			return
		}
		tag := &ResourceTag{ID: uuid.New().String(), Resource: req.Resource, Tags: req.Tags, CreatedAt: time.Now().UTC()}
		if s.policyMap != nil {
			s.policyMap.Store(r.Context(), "policy_resource_tags", req.Resource, map[string]any{
				"id": tag.ID, "resource": tag.Resource, "tags": tag.Tags,
			})
		}
		writeJSON(w, http.StatusCreated, tag)
	case http.MethodGet:
		resource := r.URL.Query().Get("resource")
		if resource != "" && s.policyMap != nil {
			row, _ := s.policyMap.Get(r.Context(), "policy_resource_tags", resource)
			if row != nil {
				writeJSON(w, http.StatusOK, row)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"resource": resource, "tags": map[string]string{}})
			return
		}
		var result []map[string]any
		if s.policyMap != nil {
			rows, _ := s.policyMap.List(r.Context(), "policy_resource_tags")
			result = rows
		}
		if result == nil { result = []map[string]any{} }
		writeJSON(w, http.StatusOK, map[string]any{"tags": result, "count": len(result)})
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
