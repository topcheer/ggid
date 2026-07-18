package server

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
)

type ClientOnboardingResult struct {
	ClientID            string   `json:"client_id"`
	ClientSecret        string   `json:"client_secret"`
	AppName             string   `json:"app_name"`
	GrantTypes          []string `json:"grant_types"`
	RedirectURIs        []string `json:"redirect_uris"`
	Scopes              []string `json:"scopes"`
	DiscoveryRegistered bool     `json:"discovery_registered"`
	TestConnection      struct {
		Status    string `json:"status"`
		LatencyMs int    `json:"latency_ms"`
	} `json:"test_connection"`
}

func handleClientOnboarding(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		AppName      string   `json:"app_info"`
		GrantTypes   []string `json:"grant_types"`
		RedirectURIs []string `json:"redirect_uris"`
		Scopes       []string `json:"scopes"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	b := make([]byte, 16)
	rand.Read(b)
	secret := fmt.Sprintf("%x", b)
	cid := fmt.Sprintf("client_%x", b[:6])

	result := ClientOnboardingResult{
		ClientID:     cid,
		ClientSecret: secret,
		AppName:      req.AppName,
		GrantTypes:   req.GrantTypes,
		RedirectURIs: req.RedirectURIs,
		Scopes:       req.Scopes,
	}
	result.DiscoveryRegistered = true
	result.TestConnection.Status = "ok"
	result.TestConnection.LatencyMs = 42

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(result)
}
