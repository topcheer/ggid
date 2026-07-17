package server

import (
	"encoding/json"
	"net/http"
	"strings"

	ggidtenant "github.com/ggid/ggid/pkg/tenant"
	"github.com/google/uuid"
)

// handleFederation routes Federation Hub endpoints.
func (h *HTTPHandler) handleFederation(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	switch {
	case strings.HasSuffix(path, "/federation/entities"):
		h.fedEntities(w, r)
	case strings.HasSuffix(path, "/federation/transform-rules"):
		h.fedTransformRules(w, r)
	case strings.HasSuffix(path, "/federation-configuration"):
		h.fedDiscovery(w, r)
	case strings.HasSuffix(path, "/federation/route-email"):
		h.fedRouteEmail(w, r)
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}

func (h *HTTPHandler) fedEntities(w http.ResponseWriter, r *http.Request) {
	tc, _ := ggidtenant.FromContext(r.Context())
	switch r.Method {
	case http.MethodPost:
		var e FederationEntity
		if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
			writeError(w, http.StatusBadRequest, "invalid body")
			return
		}
		if tc != nil {
			e.TenantID = tc.TenantID
		}
		if e.EntityID == "" || e.EntityName == "" {
			writeError(w, http.StatusBadRequest, "entity_id and entity_name required")
			return
		}
		if e.TrustLevel == "" {
			e.TrustLevel = "pending"
		}
		if e.TrustDirection == "" {
			e.TrustDirection = "inbound"
		}
		e.Enabled = true
		if h.fedRepo != nil {
			if err := h.fedRepo.CreateEntity(r.Context(), &e); err != nil {
				writeError(w, http.StatusInternalServerError, "failed")
				return
			}
		}
		writeJSON(w, http.StatusCreated, e)
	case http.MethodGet:
		var entities []*FederationEntity
		if h.fedRepo != nil && tc != nil {
			entities, _ = h.fedRepo.ListEntities(r.Context(), tc.TenantID)
		}
		if entities == nil {
			entities = []*FederationEntity{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"entities": entities, "total": len(entities)})
	case http.MethodDelete:
		idStr := r.URL.Query().Get("id")
		if id, err := uuid.Parse(idStr); err == nil && h.fedRepo != nil && tc != nil {
			h.fedRepo.DeleteEntity(r.Context(), id, tc.TenantID)
		}
		writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
	}
}

func (h *HTTPHandler) fedTransformRules(w http.ResponseWriter, r *http.Request) {
	tc, _ := ggidtenant.FromContext(r.Context())
	switch r.Method {
	case http.MethodPost:
		var t TransformRule
		if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
			writeError(w, http.StatusBadRequest, "invalid body")
			return
		}
		if tc != nil {
			t.TenantID = tc.TenantID
		}
		t.Enabled = true
		if h.fedRepo != nil {
			h.fedRepo.CreateTransformRule(r.Context(), &t)
		}
		writeJSON(w, http.StatusCreated, t)
	case http.MethodGet:
		var rules []*TransformRule
		if h.fedRepo != nil && tc != nil {
			rules, _ = h.fedRepo.ListTransformRules(r.Context(), tc.TenantID)
		}
		if rules == nil {
			rules = []*TransformRule{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"rules": rules, "total": len(rules)})
	}
}

func (h *HTTPHandler) fedDiscovery(w http.ResponseWriter, r *http.Request) {
	// Unified federation discovery endpoint.
	writeJSON(w, http.StatusOK, map[string]any{
		"issuer":            "ggid",
		"protocols":         []string{"saml", "oidc", "ldap", "did", "scim"},
		"trust_framework":   "centralized",
		"entity_types":      []string{"idp", "sp", "both"},
		"cert_validation":   "fingerprint",
		"transform_support": true,
	})
}

func (h *HTTPHandler) fedRouteEmail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		Email string `json:"email"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Email == "" || !strings.Contains(req.Email, "@") {
		writeError(w, http.StatusBadRequest, "valid email required")
		return
	}
	domain := strings.SplitN(req.Email, "@", 2)[1]
	tc, _ := ggidtenant.FromContext(r.Context())
	var entityID string
	if h.fedRepo != nil && tc != nil {
		entityID, _ = h.fedRepo.RouteEmailDomain(r.Context(), tc.TenantID, domain)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"email":      req.Email,
		"domain":     domain,
		"entity_id":  entityID,
		"routed":     entityID != "",
	})
}

// SetFedRepo injects the federation repository.
func (h *HTTPHandler) SetFedRepo(repo *federationRepo) {
	h.fedRepo = repo
}
