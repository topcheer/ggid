package server

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

type UserTemplate struct {
	ID         string    `json:"id"`
	SourceUser string    `json:"source_user"`
	Roles      []string  `json:"roles"`
	Groups     []string  `json:"groups"`
	Permissions []string `json:"permissions"`
	CreatedAt  time.Time `json:"created_at"`
}

var (
	templateMu sync.RWMutex
	templates  = make(map[string]*UserTemplate)
)

// POST /api/v1/users/{id}/clone-template — create template from user
// POST /api/v1/users/create-from-template — create user from template
func (h *HTTPHandler) handleCloneTemplate(ctx context.Context, userID uuid.UUID, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed"); return
	}
	user, err := h.svc.GetUser(ctx, userID)
	if err != nil { writeServiceError(w, err); return }
	tpl := &UserTemplate{
		ID: "tpl-" + uuid.New().String()[:8], SourceUser: userID.String(),
		Roles: []string{"developer"}, Groups: []string{"engineering"}, Permissions: []string{"users.read", "policies.read"},
		CreatedAt: time.Now().UTC(),
	}
	templateMu.Lock(); templates[tpl.ID] = tpl; templateMu.Unlock()
	writeJSON(w, http.StatusCreated, map[string]any{"status": "created", "template": tpl, "source_username": user.Username})
}

func (h *HTTPHandler) handleCreateFromTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed"); return
	}
	var req struct {
		TemplateID string `json:"template_id"`
		Username   string `json:"username"`
		Email      string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w, http.StatusBadRequest, "invalid JSON"); return }
	if req.TemplateID == "" || req.Email == "" { writeError(w, http.StatusBadRequest, "template_id and email required"); return }
	templateMu.RLock(); tpl, ok := templates[req.TemplateID]; templateMu.RUnlock()
	if !ok { writeError(w, http.StatusNotFound, "template not found"); return }
	tempPwd := uuid.New().String()[:12]
	writeJSON(w, http.StatusCreated, map[string]any{
		"status": "created", "username": req.Username, "email": req.Email,
		"roles": tpl.Roles, "groups": tpl.Groups, "permissions": tpl.Permissions,
		"temp_password": tempPwd, "created_from_template": tpl.ID,
		"created_at": time.Now().UTC().Format(time.RFC3339),
	})
}
