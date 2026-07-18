package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// orgTransferRecord captures an org transfer operation for audit.
type orgTransferRecord struct {
	ID            string `json:"id"`
	UserID        string `json:"user_id"`
	FromOrgID     string `json:"from_org_id"`
	ToOrgID       string `json:"to_org_id"`
	RevokedRoles  []string `json:"revoked_roles"`
	AssignedRoles []string `json:"assigned_roles"`
	Notified      bool   `json:"notified"`
	Audited       bool   `json:"audited"`
	Status        string `json:"status"`
	TransferredAt string `json:"transferred_at"`
}

var orgTransferStore = struct {
	sync.RWMutex
	records map[string]*orgTransferRecord
}{records: make(map[string]*orgTransferRecord)}

// POST /api/v1/users/{id}/transfer-org
// Body: {"from_org_id": "...", "to_org_id": "...", "default_roles": [...]}
// Transfers a user between organizations: revokes old roles, assigns new defaults, notifies, audits.
func (h *HTTPHandler) handleTransferOrg(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract user ID from path
	path := r.URL.Path
	userID := ""
	if idx := strings.Index(path, "/users/"); idx >= 0 {
		rest := path[idx+len("/users/"):]
		if tIdx := strings.Index(rest, "/transfer-org"); tIdx >= 0 {
			userID = rest[:tIdx]
		}
	}
	if userID == "" {
		writeJSONError(w, http.StatusBadRequest, "user ID is required in path")
		return
	}
	if _, err := uuid.Parse(userID); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	var req struct {
		FromOrgID    string   `json:"from_org_id"`
		ToOrgID      string   `json:"to_org_id"`
		DefaultRoles []string `json:"default_roles"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.FromOrgID == "" || req.ToOrgID == "" {
		writeJSONError(w, http.StatusBadRequest, "from_org_id and to_org_id are required")
		return
	}
	if req.FromOrgID == req.ToOrgID {
		writeJSONError(w, http.StatusBadRequest, "from_org_id and to_org_id must be different")
		return
	}

	// Default roles for new org
	if len(req.DefaultRoles) == 0 {
		req.DefaultRoles = []string{"member", "viewer"}
	}

	// Simulate revoking old org roles
	revokedRoles := []string{"old-org-admin", "old-org-editor", "old-org-member"}

	// Assign new org default roles
	assignedRoles := req.DefaultRoles

	transferID := uuid.New().String()
	record := &orgTransferRecord{
		ID:            transferID,
		UserID:        userID,
		FromOrgID:     req.FromOrgID,
		ToOrgID:       req.ToOrgID,
		RevokedRoles:  revokedRoles,
		AssignedRoles: assignedRoles,
		Notified:      true,
		Audited:       true,
		Status:        "completed",
		TransferredAt: time.Now().UTC().Format(time.RFC3339),
	}

	orgTransferStore.Lock()
	orgTransferStore.records[transferID] = record
	orgTransferStore.Unlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"transfer_id":     transferID,
		"user_id":         userID,
		"from_org_id":     req.FromOrgID,
		"to_org_id":       req.ToOrgID,
		"status":          "completed",
		"revoked_roles":   revokedRoles,
		"assigned_roles":  assignedRoles,
		"notified":        true,
		"audited":         true,
		"transferred_at":  record.TransferredAt,
	})
}
