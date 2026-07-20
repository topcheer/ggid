package main

import (
	"encoding/json"
	"net/http"

	ggid "github.com/ggid/ggid/sdk/go"
)

func handleLogin(w http.ResponseWriter, r *http.Request) {
	if !methodAllowed(w, r, http.MethodPost) { return }
	var req struct{ Username, Password string }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w, 400, "invalid body"); return }
	tokens, err := ggidClient.Login(r.Context(), &ggid.LoginRequest{Username: req.Username, Password: req.Password})
	if err != nil { writeJSON(w, 401, map[string]string{"error": "login failed"}); return }
	writeJSON(w, 200, tokens)
}

func handleRefresh(w http.ResponseWriter, r *http.Request) {
	if !methodAllowed(w, r, http.MethodPost) { return }
	var req struct{ RefreshToken string `json:"refresh_token"` }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w, 400, "invalid body"); return }
	tokens, err := ggidClient.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil { writeError(w, 401, "refresh failed"); return }
	writeJSON(w, 200, tokens)
}

func handleVerify(w http.ResponseWriter, r *http.Request) {
	if !methodAllowed(w, r, http.MethodPost) { return }
	var req struct{ Token string }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w, 400, "invalid body"); return }
	info, err := ggidClient.VerifyToken(r.Context(), req.Token)
	if err != nil { writeJSON(w, 401, map[string]string{"error": "invalid token"}); return }
	writeJSON(w, 200, info)
}
