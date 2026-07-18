package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// MDMConnector defines an MDM integration.
type MDMConnector struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"` // intune, jamf, android
	Enabled   bool      `json:"enabled"`
	LastSync  *time.Time `json:"last_sync,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// MDMDevice represents a device managed by MDM.
type MDMDevice struct {
	DeviceID    string `json:"device_id"`
	Name        string `json:"name"`
	Platform    string `json:"platform"`
	Compliant   bool   `json:"compliant"`
	Managed     bool   `json:"managed"`
	LastSeen    string `json:"last_seen"`
}

// GET /api/v1/mdm/connectors
// POST /api/v1/mdm/connectors
func (h *HTTPHandler) handleMDMConnectors(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req MDMConnector
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		if req.Name == "" || req.Type == "" {
			writeJSONError(w, http.StatusBadRequest, "name and type required")
			return
		}
		validTypes := map[string]bool{"intune": true, "jamf": true, "android": true}
		if !validTypes[req.Type] {
			writeJSONError(w, http.StatusBadRequest, "type must be intune, jamf, or android")
			return
		}
		req.ID = uuid.New().String()
		req.Enabled = true
		req.CreatedAt = time.Now().UTC()
		writeJSON(w, http.StatusCreated, req)
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{"connectors": []MDMConnector{}, "count": 0})
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// POST /api/v1/mdm/sync/:connector
func (h *HTTPHandler) handleMDMSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	connectorID := strings.TrimPrefix(r.URL.Path, "/api/v1/mdm/sync/")
	if connectorID == "" {
		writeJSONError(w, http.StatusBadRequest, "connector id required")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "synced", "connector_id": connectorID,
		"devices_synced": 0, "synced_at": time.Now().UTC(),
	})
}

// GET /api/v1/mdm/devices
func (h *HTTPHandler) handleMDMDevices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"devices": []MDMDevice{}, "count": 0})
}

// GET /api/v1/mdm/devices/:id/compliance
func (h *HTTPHandler) handleMDMCompliance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	deviceID := strings.TrimPrefix(r.URL.Path, "/api/v1/mdm/devices/")
	deviceID = strings.TrimSuffix(deviceID, "/compliance")
	if deviceID == "" {
		writeJSONError(w, http.StatusBadRequest, "device id required")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"device_id": deviceID, "compliant": true,
		"managed": true, "platform": "unknown",
		"checks": map[string]bool{
			"encryption":     true,
			"os_version":     true,
			"jailbreak":      false,
			"malware":        false,
		},
	})
}
