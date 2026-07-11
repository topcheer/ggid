package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

type ClientLifecycle struct {
	ClientID    string     `json:"client_id"`
	Status      string     `json:"status"`
	SuspendedAt *time.Time `json:"suspended_at,omitempty"`
	SuspendedBy string     `json:"suspended_by,omitempty"`
	Reason      string     `json:"reason,omitempty"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

var (
	clientLifecycleMu sync.RWMutex
	clientLifecycles  = make(map[string]*ClientLifecycle)
)

func GetClientStatus(clientID string) string {
	clientLifecycleMu.RLock()
	defer clientLifecycleMu.RUnlock()
	if cl, ok := clientLifecycles[clientID]; ok {
		return cl.Status
	}
	return "active"
}

// POST /api/v1/oauth/clients/{id}/suspend
// POST /api/v1/oauth/clients/{id}/reinstate
func handleClientLifecycle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}

	path := r.URL.Path
	clientID := ""
	action := ""

	if strings.HasSuffix(path, "/suspend") {
		clientID = strings.TrimSuffix(strings.TrimPrefix(path, "/api/v1/oauth/clients/"), "/suspend")
		action = "suspend"
	} else if strings.HasSuffix(path, "/reinstate") {
		clientID = strings.TrimSuffix(strings.TrimPrefix(path, "/api/v1/oauth/clients/"), "/reinstate")
		action = "reinstate"
	}

	if clientID == "" || strings.Contains(clientID, "/") {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid client_id"})
		return
	}

	now := time.Now().UTC()

	switch action {
	case "suspend":
		var req struct {
			Reason string `json:"reason"`
			By     string `json:"suspended_by"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		cl := &ClientLifecycle{
			ClientID: clientID, Status: "suspended",
			SuspendedAt: &now, SuspendedBy: req.By, Reason: req.Reason, UpdatedAt: now,
		}
		clientLifecycleMu.Lock()
		clientLifecycles[clientID] = cl
		clientLifecycleMu.Unlock()
		writeJSON(w, http.StatusOK, map[string]any{
			"status": "suspended", "client_id": clientID,
			"suspended_at": now, "reason": req.Reason,
		})
	case "reinstate":
		clientLifecycleMu.Lock()
		if cl, ok := clientLifecycles[clientID]; ok {
			cl.Status = "active"
			cl.SuspendedAt = nil
			cl.UpdatedAt = now
		} else {
			clientLifecycles[clientID] = &ClientLifecycle{
				ClientID: clientID, Status: "active", UpdatedAt: now,
			}
		}
		clientLifecycleMu.Unlock()
		writeJSON(w, http.StatusOK, map[string]any{
			"status": "active", "client_id": clientID, "reinstate_at": now,
		})
	}
}
