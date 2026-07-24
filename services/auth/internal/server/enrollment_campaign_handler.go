package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// EnrollmentCampaign represents a security enrollment campaign (passkey, webauthn, etc.)
type EnrollmentCampaign struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	TargetGroup string `json:"target_group"`
	Method      string `json:"method"`
	Deadline    string `json:"deadline"`
	Enrolled    int    `json:"enrolled"`
	Target      int    `json:"target"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
}

// handleEnrollmentCampaigns manages CRUD for enrollment campaigns.
// Uses in-memory storage (replaced by DB in production).
var campaignStore = []EnrollmentCampaign{}

func (h *Handler) handleEnrollmentCampaigns(w http.ResponseWriter, r *http.Request) {
	// Check for /{id} sub-path
	path := r.URL.Path
	if len(path) > len("/api/v1/auth/enrollment/campaigns/") {
		id := path[len("/api/v1/auth/enrollment/campaigns/"):]
		h.handleEnrollmentCampaignByID(w, r, id)
		return
	}

	switch r.Method {
	case http.MethodGet:
		if len(campaignStore) == 0 {
			// Return empty list instead of null
			writeJSON(w, http.StatusOK, map[string]any{"campaigns": []EnrollmentCampaign{}})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"campaigns": campaignStore})

	case http.MethodPost:
		var c EnrollmentCampaign
		if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		c.ID = uuid.New().String()
		c.CreatedAt = time.Now().UTC().Format(time.RFC3339)
		if c.Status == "" {
			c.Status = "active"
		}
		campaignStore = append(campaignStore, c)
		writeJSON(w, http.StatusCreated, c)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) handleEnrollmentCampaignByID(w http.ResponseWriter, r *http.Request, id string) {
	switch r.Method {
	case http.MethodDelete:
		for i, c := range campaignStore {
			if c.ID == id {
				campaignStore = append(campaignStore[:i], campaignStore[i+1:]...)
				writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
				return
			}
		}
		writeError(w, http.StatusNotFound, "campaign not found")

	case http.MethodPut:
		var c EnrollmentCampaign
		if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		c.ID = id
		for i, existing := range campaignStore {
			if existing.ID == id {
				campaignStore[i] = c
				writeJSON(w, http.StatusOK, c)
				return
			}
		}
		writeError(w, http.StatusNotFound, "campaign not found")

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
