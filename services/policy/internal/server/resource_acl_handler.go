package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type ResourceACL struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenant_id"`
	ResourcePath string    `json:"resource_path"`
	Principal    string    `json:"principal"`
	PrincipalType string   `json:"principal_type"` // user, role, group
	Effect       string    `json:"effect"` // allow, deny
	Priority     int       `json:"priority"`
	CreatedAt    time.Time `json:"created_at"`
}

var (resourceACLMu sync.RWMutex; resourceACLs = make(map[string]*ResourceACL))

func (s *HTTPServer) handleResourceACL(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req struct{ TenantID, ResourcePath, Principal, PrincipalType, Effect string; Priority int }
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeJSONError(w, http.StatusBadRequest, "invalid JSON"); return }
		if req.ResourcePath == "" || req.Principal == "" { writeJSONError(w, http.StatusBadRequest, "resource_path and principal required"); return }
		if req.Effect == "" { req.Effect = "allow" }
		if req.PrincipalType == "" { req.PrincipalType = "user" }
		acl := &ResourceACL{ID: uuid.New().String(), TenantID: req.TenantID, ResourcePath: req.ResourcePath, Principal: req.Principal, PrincipalType: req.PrincipalType, Effect: req.Effect, Priority: req.Priority, CreatedAt: time.Now().UTC()}
		resourceACLMu.Lock(); resourceACLs[acl.ID] = acl; resourceACLMu.Unlock()
		writeJSON(w, http.StatusCreated, acl)
	case http.MethodGet:
		resource := r.URL.Query().Get("resource")
		resourceACLMu.RLock(); result := []*ResourceACL{}
		for _, acl := range resourceACLs {
			if resource != "" && !pathMatch(acl.ResourcePath, resource) { continue }
			result = append(result, acl)
		}
		resourceACLMu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]any{"acls": result, "count": len(result)})
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func pathMatch(pattern, path string) bool {
	if strings.HasSuffix(pattern, "/*") { return strings.HasPrefix(path, strings.TrimSuffix(pattern, "*")) }
	return pattern == path
}
