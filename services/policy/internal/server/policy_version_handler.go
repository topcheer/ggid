package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/ggid/ggid/services/policy/internal/service"
)

var pvSvc = service.NewPolicyVersioningService()

func (s *HTTPServer) handlePolicyVersionRoute(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if strings.HasSuffix(path, "/versions/compare") {
		s.handlePolicyVersionCompare(w, r)
		return
	}
	if strings.HasSuffix(path, "/rollback") {
		s.handlePolicyVersionRollback(w, r)
		return
	}
	if strings.HasSuffix(path, "/versions") {
		if r.Method == http.MethodPost {
			s.handlePolicyVersionCreate(w, r)
		} else {
			s.handlePolicyVersionList(w, r)
		}
		return
	}
	// /versions/{vid}
	s.handlePolicyVersionGet(w, r)
}

func (s *HTTPServer) handlePolicyVersionCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	policyID := extractPolicyID(r.URL.Path, "versions")
	var req struct {
		CreatedBy string `json:"created_by"`
		Diff      string `json:"diff"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	v := pvSvc.CreateVersion(policyID, req.CreatedBy, req.Diff)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(v)
}

func (s *HTTPServer) handlePolicyVersionList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	policyID := extractPolicyID(r.URL.Path, "versions")
	versions := pvSvc.ListVersions(policyID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(versions)
}

func (s *HTTPServer) handlePolicyVersionGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	policyID := extractPolicyID(r.URL.Path, "versions")
	versionID := extractLastSegment(r.URL.Path)
	v := pvSvc.GetVersion(policyID, versionID)
	if v == nil {
		http.Error(w, `{"error":"version not found"}`, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func (s *HTTPServer) handlePolicyVersionRollback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	policyID := extractPolicyID(r.URL.Path, "versions")
	parts := strings.Split(r.URL.Path, "/")
	versionID := ""
	for i, p := range parts {
		if p == "rollback" && i > 0 {
			versionID = parts[i-1]
			break
		}
	}
	v, err := pvSvc.RollbackVersion(policyID, versionID)
	if err != nil || v == nil {
		http.Error(w, `{"error":"rollback failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func (s *HTTPServer) handlePolicyVersionCompare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	policyID := extractPolicyID(r.URL.Path, "versions")
	v1 := r.URL.Query().Get("v1")
	v2 := r.URL.Query().Get("v2")
	if v1 == "" || v2 == "" {
		http.Error(w, `{"error":"v1 and v2 query params required"}`, http.StatusBadRequest)
		return
	}
	diff := pvSvc.CompareVersions(policyID, atoiSafe(v1), atoiSafe(v2))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(diff)
}

func extractPolicyID(path, stopWord string) string {
	parts := strings.Split(strings.TrimSuffix(path, "/"), "/")
	for i, p := range parts {
		if p == stopWord && i > 0 {
			return parts[i-1]
		}
	}
	return ""
}

func extractLastSegment(path string) string {
	parts := strings.Split(strings.TrimSuffix(path, "/"), "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

func atoiSafe(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return n
		}
		n = n*10 + int(c-'0')
	}
	return n
}