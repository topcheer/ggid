package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/ggid/ggid/services/identity/internal/service"
)

var vcIssuer = service.NewVCIssuer()

func (h *HTTPHandler) handleVCIssue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		IssuerDID      string         `json:"issuer_did"`
		SubjectDID     string         `json:"subject_did"`
		CredentialType string         `json:"credential_type"`
		Claims         map[string]any `json:"claims"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	vc, err := vcIssuer.IssueVC(req.IssuerDID, req.SubjectDID, req.CredentialType, req.Claims)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to issue credential")
		return
	}
	if err := vcIssuer.SignVC(vc, req.IssuerDID); err != nil {
		http.Error(w, `{"error":"signing failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(vc)
}

func (h *HTTPHandler) handleVCVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		VC        service.VerifiableCredential `json:"vc"`
		IssuerDID string                       `json:"issuer_did"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	err := vcIssuer.VerifyVC(&req.VC, req.IssuerDID)
	result := map[string]any{"valid": err == nil}
	if err != nil {
		result["error"] = err.Error()
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *HTTPHandler) handleVCList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	issuerDID := r.URL.Query().Get("issuer")
	vcs := vcIssuer.ListIssuedVCs(issuerDID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(vcs)
}

func (h *HTTPHandler) handleVCRevoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	parts := strings.Split(strings.TrimSuffix(r.URL.Path, "/"), "/")
	if len(parts) < 1 {
		http.Error(w, `{"error":"vc id required"}`, http.StatusBadRequest)
		return
	}
	vcID := parts[len(parts)-1]
	reason := r.URL.Query().Get("reason")
	if reason == "" {
		reason = "revoked by admin"
	}
	vcIssuer.RevokeVC(vcID, reason)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "revoked", "id": vcID})
}

func (h *HTTPHandler) handleVCPresent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		HolderDID string         `json:"holder_did"`
		VCIDs     []string       `json:"vc_ids"`
		Challenge string         `json:"challenge"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	vp := map[string]any{
		"@context":  []string{"https://www.w3.org/ns/credentials/v2"},
		"type":      []string{"VerifiablePresentation"},
		"holder":    req.HolderDID,
		"vc_ids":    req.VCIDs,
		"challenge": req.Challenge,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(vp)
}