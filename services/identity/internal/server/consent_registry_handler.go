package server

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// consentRecord represents a single user consent entry.
type consentRecord struct {
	ID        string `json:"id"`
	Type      string `json:"type"`   // data_processing, marketing, cookies, third_party_sharing
	Status    string `json:"status"` // granted, denied, withdrawn, pending
	GrantedAt string `json:"granted_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
	Source    string `json:"source"` // web, mobile, api, implicit
	Detail    string `json:"detail,omitempty"`
	Version   string `json:"version"` // consent version (policy version when consented)
}

var consentStore = struct {
	sync.RWMutex
	data map[string][]consentRecord // userID → consents
}{data: make(map[string][]consentRecord)}

// GET /api/v1/users/{id}/consent-registry — list all consents
// POST /api/v1/users/{id}/consent-registry — update/set a consent
func (h *HTTPHandler) handleConsentRegistry(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from path
	path := r.URL.Path
	userID := ""
	if idx := strings.Index(path, "/users/"); idx >= 0 {
		rest := path[idx+len("/users/"):]
		if cIdx := strings.Index(rest, "/consent-registry"); cIdx >= 0 {
			userID = rest[:cIdx]
		}
	}
	if userID == "" {
		writeError(w, http.StatusBadRequest, "user ID is required in path")
		return
	}
	if _, err := uuid.Parse(userID); err != nil {
		writeError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	switch r.Method {
	case http.MethodGet:
		consentStore.RLock()
		records := consentStore.data[userID]
		result := make([]consentRecord, len(records))
		copy(result, records)
		consentStore.RUnlock()

		// If no records, return default GDPR-required consent types
		if len(result) == 0 {
			result = defaultConsents(userID)
		}

		// Build summary
		summary := map[string]string{}
		for _, c := range result {
			summary[c.Type] = c.Status
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"user_id":    userID,
			"consents":   result,
			"summary":    summary,
			"total":      len(result),
			"regulation": "GDPR Article 7",
			"checked_at": time.Now().UTC().Format(time.RFC3339),
		})

	case http.MethodPost:
		var req struct {
			Type    string `json:"type"`
			Status  string `json:"status"`
			Source  string `json:"source"`
			Detail  string `json:"detail"`
			Version string `json:"version"`
		}
		if err := readJSONBody2(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		validTypes := map[string]bool{
			"data_processing": true, "marketing": true,
			"cookies": true, "third_party_sharing": true,
		}
		validStatuses := map[string]bool{
			"granted": true, "denied": true, "withdrawn": true,
		}

		if !validTypes[req.Type] {
			writeError(w, http.StatusBadRequest, "type must be one of: data_processing, marketing, cookies, third_party_sharing")
			return
		}
		if !validStatuses[req.Status] {
			writeError(w, http.StatusBadRequest, "status must be one of: granted, denied, withdrawn")
			return
		}
		if req.Source == "" {
			req.Source = "web"
		}
		if req.Version == "" {
			req.Version = "1.0"
		}

		now := time.Now().UTC().Format(time.RFC3339)
		record := consentRecord{
			ID:        uuid.New().String(),
			Type:      req.Type,
			Status:    req.Status,
			GrantedAt: now,
			UpdatedAt: now,
			Source:    req.Source,
			Detail:    req.Detail,
			Version:   req.Version,
		}

		consentStore.Lock()
		// Remove existing record of same type
		existing := consentStore.data[userID]
		filtered := existing[:0]
		for _, c := range existing {
			if c.Type != req.Type {
				filtered = append(filtered, c)
			}
		}
		filtered = append(filtered, record)
		consentStore.data[userID] = filtered
		consentStore.Unlock()

		writeJSON(w, http.StatusOK, record)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func defaultConsents(userID string) []consentRecord {
	now := time.Now().UTC().Format(time.RFC3339)
	return []consentRecord{
		{ID: uuid.New().String(), Type: "data_processing", Status: "granted", GrantedAt: now, Source: "implicit", Version: "1.0", Detail: "Required for core service functionality"},
		{ID: uuid.New().String(), Type: "marketing", Status: "pending", Source: "web", Version: "1.0"},
		{ID: uuid.New().String(), Type: "cookies", Status: "granted", GrantedAt: now, Source: "implicit", Version: "1.0"},
		{ID: uuid.New().String(), Type: "third_party_sharing", Status: "pending", Source: "web", Version: "1.0"},
	}
}

func readJSONBody2(r *http.Request, v any) error {
	return jsonDecode(r, v)
}
