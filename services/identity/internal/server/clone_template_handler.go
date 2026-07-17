package server

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type UserTemplate struct {
	ID          string    `json:"id"`
	SourceUser  string    `json:"source_user"`
	Roles       []string  `json:"roles"`
	Groups      []string  `json:"groups"`
	Permissions []string  `json:"permissions"`
	CreatedAt   time.Time `json:"created_at"`
}

func (h *HTTPHandler) handleCloneTemplate(ctx context.Context, userID uuid.UUID, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	user, err := h.svc.GetUser(ctx, userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	tpl := &UserTemplate{
		ID: "tpl-" + uuid.New().String()[:8], SourceUser: userID.String(),
		Roles: []string{"developer"}, Groups: []string{"engineering"},
		Permissions: []string{"users.read", "policies.read"},
		CreatedAt: time.Now().UTC(),
	}
	if h.identityPolicyMap != nil {
		h.identityPolicyMap.Store(r.Context(), "identity_templates", tpl.ID, map[string]any{
			"source_user": tpl.SourceUser, "roles": tpl.Roles,
			"groups": tpl.Groups, "permissions": tpl.Permissions,
		})
	}
	writeJSON(w, http.StatusCreated, map[string]any{"status": "created", "template": tpl, "source_username": user.Username})
}

func (h *HTTPHandler) handleCreateFromTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		TemplateID string `json:"template_id"`
		Username   string `json:"username"`
		Email      string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.TemplateID == "" || req.Email == "" {
		writeError(w, http.StatusBadRequest, "template_id and email required")
		return
	}
	if h.identityPolicyMap != nil {
		tpl, _ := h.identityPolicyMap.Get(r.Context(), "identity_templates", req.TemplateID)
		if tpl == nil {
			writeError(w, http.StatusNotFound, "template not found")
			return
		}
		tempPwd := uuid.New().String()[:12]
		writeJSON(w, http.StatusCreated, map[string]any{
			"status": "created", "username": req.Username, "email": req.Email,
			"roles": tpl["roles"], "groups": tpl["groups"], "permissions": tpl["permissions"],
			"temp_password": tempPwd, "created_from_template": req.TemplateID,
			"created_at": time.Now().UTC().Format(time.RFC3339),
		})
		return
	}
	writeError(w, http.StatusNotFound, "template not found")
}
