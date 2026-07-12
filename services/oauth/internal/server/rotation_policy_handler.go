package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
)

// rotationPolicy holds token auto-rotation config per client.
type rotationPolicy struct {
	ClientID           string `json:"client_id"`
	IntervalDays       int    `json:"interval_days"`
	MaxAgeHours        int    `json:"max_age_hours"`
	NotifyBeforeHours  int    `json:"notify_before_hours"`
	AutoRotate         bool   `json:"auto_rotate"`
}

var rotationPolicyStore = struct {
	sync.RWMutex
	data map[string]*rotationPolicy
}{data: map[string]*rotationPolicy{
	"web-app":         {ClientID: "web-app", IntervalDays: 90, MaxAgeHours: 1, NotifyBeforeHours: 168, AutoRotate: true},
	"mobile-ios":      {ClientID: "mobile-ios", IntervalDays: 180, MaxAgeHours: 24, NotifyBeforeHours: 336, AutoRotate: true},
	"admin-cli":       {ClientID: "admin-cli", IntervalDays: 30, MaxAgeHours: 1, NotifyBeforeHours: 72, AutoRotate: true},
	"service-backend": {ClientID: "service-backend", IntervalDays: 365, MaxAgeHours: 1, NotifyBeforeHours: 720, AutoRotate: false},
}}

// GET/PUT /api/v1/oauth/clients/{id}/rotation-policy
func handleRotationPolicy(w http.ResponseWriter, r *http.Request) {
	clientID := strings.TrimPrefix(r.URL.Path, "/api/v1/oauth/clients/")
	clientID = strings.TrimSuffix(clientID, "/rotation-policy")
	clientID = strings.TrimSuffix(clientID, "/")
	if clientID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "client_id is required"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		rotationPolicyStore.RLock()
		pol, exists := rotationPolicyStore.data[clientID]
		rotationPolicyStore.RUnlock()

		if !exists {
			pol = &rotationPolicy{ClientID: clientID, IntervalDays: 90, MaxAgeHours: 1, NotifyBeforeHours: 168, AutoRotate: true}
		}
		writeJSON(w, http.StatusOK, pol)

	case http.MethodPut, http.MethodPost:
		var req struct {
			IntervalDays      int  `json:"interval_days"`
			MaxAgeHours       int  `json:"max_age_hours"`
			NotifyBeforeHours int  `json:"notify_before_hours"`
			AutoRotate        *bool `json:"auto_rotate"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}

		rotationPolicyStore.Lock()
		pol, exists := rotationPolicyStore.data[clientID]
		if !exists {
			pol = &rotationPolicy{ClientID: clientID, IntervalDays: 90, MaxAgeHours: 1, NotifyBeforeHours: 168, AutoRotate: true}
		}
		if req.IntervalDays > 0 {
			pol.IntervalDays = req.IntervalDays
		}
		if req.MaxAgeHours > 0 {
			pol.MaxAgeHours = req.MaxAgeHours
		}
		if req.NotifyBeforeHours > 0 {
			pol.NotifyBeforeHours = req.NotifyBeforeHours
		}
		if req.AutoRotate != nil {
			pol.AutoRotate = *req.AutoRotate
		}
		rotationPolicyStore.data[clientID] = pol
		rotationPolicyStore.Unlock()

		writeJSON(w, http.StatusOK, map[string]any{
			"client_id": clientID,
			"policy":    pol,
			"updated":   true,
		})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}
