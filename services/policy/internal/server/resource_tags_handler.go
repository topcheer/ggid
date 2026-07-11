package httpserver

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

type ResourceTag struct {
	ID        string            `json:"id"`
	Resource  string            `json:"resource"`
	Tags      map[string]string `json:"tags"`
	CreatedAt time.Time         `json:"created_at"`
}

var (
	resTagMu sync.RWMutex
	resTags  = make(map[string]*ResourceTag) // key: resource path
)

// POST /api/v1/policies/resource-tags — tag a resource
// GET /api/v1/policies/resource-tags?resource=X — query tags
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
		resTagMu.Lock()
		resTags[req.Resource] = tag
		resTagMu.Unlock()
		writeJSON(w, http.StatusCreated, tag)
	case http.MethodGet:
		resource := r.URL.Query().Get("resource")
		if resource != "" {
			resTagMu.RLock()
			tag, ok := resTags[resource]
			resTagMu.RUnlock()
			if !ok {
				writeJSON(w, http.StatusOK, map[string]any{"resource": resource, "tags": map[string]string{}})
				return
			}
			writeJSON(w, http.StatusOK, tag)
			return
		}
		resTagMu.RLock()
		all := make([]*ResourceTag, 0, len(resTags))
		for _, t := range resTags {
			all = append(all, t)
		}
		resTagMu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]any{"tags": all, "count": len(all)})
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
