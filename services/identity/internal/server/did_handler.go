package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ggid/ggid/services/identity/internal/service"
)

var didResolver = service.NewDIDResolver(30 * time.Minute)

// didActiveCache tracks active DIDs for quick listing (backed by PG).
var didActiveCache sync.Map

func (h *HTTPHandler) handleDIDRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		h.handleDIDRegister(w, r)
		return
	}
	if r.Method == http.MethodDelete {
		h.handleDIDDeactivate(w, r)
		return
	}
	h.handleDIDResolve(w, r)
}

func (h *HTTPHandler) handleDIDResolve(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	parts := strings.Split(strings.TrimSuffix(r.URL.Path, "/"), "/")
	if len(parts) < 1 {
		writeJSONError(w, http.StatusBadRequest, "did required")
		return
	}
	did := parts[len(parts)-1]
	doc, err := didResolver.ResolveDID(did)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "DID not found")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(doc)
}

func (h *HTTPHandler) handleDIDList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var dids []string
	if h.identityPolicyMap != nil {
		rows, _ := h.identityPolicyMap.List(r.Context(), "identity_did_registry")
		for _, row := range rows {
			if did, ok := row["id"].(string); ok {
				dids = append(dids, did)
			}
		}
	}
	if dids == nil { dids = []string{} }
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dids)
}

func (h *HTTPHandler) handleDIDRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		DID string `json:"did"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.DID == "" {
		writeJSONError(w, http.StatusBadRequest, "did required")
		return
	}
	didActiveCache.Store(req.DID, true)
	if h.identityPolicyMap != nil {
		h.identityPolicyMap.Store(r.Context(), "identity_did_registry", req.DID, map[string]any{"active": true})
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "registered", "did": req.DID})
}

func (h *HTTPHandler) handleDIDDeactivate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	parts := strings.Split(strings.TrimSuffix(r.URL.Path, "/"), "/")
	if len(parts) < 1 {
		writeJSONError(w, http.StatusBadRequest, "did required")
		return
	}
	did := parts[len(parts)-1]
	didActiveCache.Delete(did)
	if h.identityPolicyMap != nil {
		h.identityPolicyMap.Delete(r.Context(), "identity_did_registry", did)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deactivated", "did": did})
}
