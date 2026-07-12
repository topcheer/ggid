package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// evidenceVersion stores a single version of a compliance evidence record.
type evidenceVersion struct {
	Version     int                    `json:"version"`
	Content     map[string]any         `json:"content"`
	CreatedBy   string                 `json:"created_by"`
	CreatedAt   string                 `json:"created_at"`
	Description string                 `json:"description"`
	Checksum    string                 `json:"checksum"`
}

var evidenceVersionStore = struct {
	sync.RWMutex
	data map[string][]evidenceVersion // evidenceID → versions
}{data: make(map[string][]evidenceVersion)}

// POST /api/v1/audit/compliance/evidence/{id}/version — create a new version
// GET  /api/v1/audit/compliance/evidence/{id}/versions — list version history
// GET  /api/v1/audit/compliance/evidence/{id}/versions/diff?v1=1&v2=2 — diff two versions
func (s *HTTPServer) handleEvidenceVersioning(w http.ResponseWriter, r *http.Request) {
	// Extract evidence ID and action from path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/audit/compliance/evidence/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) < 2 {
		writeJSONError(w, http.StatusBadRequest, "evidence ID and action are required")
		return
	}
	evidenceID := parts[0]
	if evidenceID == "" {
		writeJSONError(w, http.StatusBadRequest, "evidence ID is required")
		return
	}

	// Determine action from the remaining path
	remaining := parts[1]

	switch {
	case remaining == "version" && r.Method == http.MethodPost:
		createEvidenceVersion(w, r, evidenceID)
	case remaining == "versions" && r.Method == http.MethodGet:
		listEvidenceVersions(w, r, evidenceID)
	case remaining == "versions/diff" && r.Method == http.MethodGet:
		diffEvidenceVersions(w, r, evidenceID)
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func createEvidenceVersion(w http.ResponseWriter, r *http.Request, evidenceID string) {
	var req struct {
		Content     map[string]any `json:"content"`
		CreatedBy   string         `json:"created_by"`
		Description string         `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.CreatedBy == "" {
		req.CreatedBy = "system"
	}

	evidenceVersionStore.Lock()
	versions := evidenceVersionStore.data[evidenceID]
	newVersion := evidenceVersion{
		Version:     len(versions) + 1,
		Content:     req.Content,
		CreatedBy:   req.CreatedBy,
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
		Description: req.Description,
		Checksum:    fmt.Sprintf("%x", uuid.New().ID()),
	}
	evidenceVersionStore.data[evidenceID] = append(versions, newVersion)
	evidenceVersionStore.Unlock()

	writeJSON(w, http.StatusCreated, newVersion)
}

func listEvidenceVersions(w http.ResponseWriter, r *http.Request, evidenceID string) {
	evidenceVersionStore.RLock()
	versions := evidenceVersionStore.data[evidenceID]
	if versions == nil {
		versions = []evidenceVersion{}
	}
	// Return a copy
	result := make([]evidenceVersion, len(versions))
	copy(result, versions)
	evidenceVersionStore.RUnlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"evidence_id":   evidenceID,
		"versions":      result,
		"total":         len(result),
		"latest_version": func() int {
			if len(result) > 0 {
				return result[len(result)-1].Version
			}
			return 0
		}(),
	})
}

func diffEvidenceVersions(w http.ResponseWriter, r *http.Request, evidenceID string) {
	v1Str := r.URL.Query().Get("v1")
	v2Str := r.URL.Query().Get("v2")
	if v1Str == "" || v2Str == "" {
		writeJSONError(w, http.StatusBadRequest, "v1 and v2 query params are required")
		return
	}

	var v1, v2 int
	fmt.Sscanf(v1Str, "%d", &v1)
	fmt.Sscanf(v2Str, "%d", &v2)

	evidenceVersionStore.RLock()
	versions := evidenceVersionStore.data[evidenceID]
	evidenceVersionStore.RUnlock()

	if v1 < 1 || v1 > len(versions) || v2 < 1 || v2 > len(versions) {
		writeJSONError(w, http.StatusBadRequest, "version out of range")
		return
	}

	oldV := versions[v1-1]
	newV := versions[v2-1]

	// Compute diff
	added := []string{}
	removed := []string{}
	modified := []map[string]any{}

	if oldV.Content != nil && newV.Content != nil {
		for key, newVal := range newV.Content {
			oldVal, exists := oldV.Content[key]
			if !exists {
				added = append(added, key)
			} else {
				oldStr := fmt.Sprintf("%v", oldVal)
				newStr := fmt.Sprintf("%v", newVal)
				if oldStr != newStr {
					modified = append(modified, map[string]any{
						"field": key,
						"old":   oldVal,
						"new":   newVal,
					})
				}
			}
		}
		for key := range oldV.Content {
			if _, exists := newV.Content[key]; !exists {
				removed = append(removed, key)
			}
		}
	}

	// Check description change
	if oldV.Description != newV.Description {
		modified = append(modified, map[string]any{
			"field": "description",
			"old":   oldV.Description,
			"new":   newV.Description,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"evidence_id": evidenceID,
		"v1":          v1,
		"v2":          v2,
		"diff": map[string]any{
			"added":    added,
			"removed":  removed,
			"modified": modified,
		},
		"summary": fmt.Sprintf("%d added, %d removed, %d modified", len(added), len(removed), len(modified)),
		"v1_created_at": oldV.CreatedAt,
		"v2_created_at": newV.CreatedAt,
	})
}
