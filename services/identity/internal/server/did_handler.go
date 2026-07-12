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

var (
	didRegistry   = make(map[string]bool)
	didRegistryMu sync.RWMutex
)

func (h *HTTPHandler) handleDIDResolve(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	parts := strings.Split(strings.TrimSuffix(r.URL.Path, "/"), "/")
	if len(parts) < 1 {
		http.Error(w, `{"error":"did required"}`, http.StatusBadRequest)
		return
	}
	did := parts[len(parts)-1]
	doc, err := didResolver.ResolveDID(did)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(doc)
}

func (h *HTTPHandler) handleDIDList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	didRegistryMu.RLock()
	defer didRegistryMu.RUnlock()
	var dids []string
	for did := range didRegistry {
		dids = append(dids, did)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dids)
}

func (h *HTTPHandler) handleDIDRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		DID string `json:"did"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	if req.DID == "" {
		http.Error(w, `{"error":"did required"}`, http.StatusBadRequest)
		return
	}
	didRegistryMu.Lock()
	didRegistry[req.DID] = true
	didRegistryMu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "registered", "did": req.DID})
}

func (h *HTTPHandler) handleDIDDeactivate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	parts := strings.Split(strings.TrimSuffix(r.URL.Path, "/"), "/")
	if len(parts) < 1 {
		http.Error(w, `{"error":"did required"}`, http.StatusBadRequest)
		return
	}
	did := parts[len(parts)-1]
	didRegistryMu.Lock()
	delete(didRegistry, did)
	didRegistryMu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deactivated", "did": did})
}