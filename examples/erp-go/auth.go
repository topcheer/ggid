package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	ggid "github.com/ggid/ggid/sdk/go"
)

// handleLogin authenticates via GGID SDK
func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, 405, map[string]string{"error": "method not allowed"})
		return
	}
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid body"})
		return
	}
	// Use GGID SDK to login
	tokens, err := ggidClient.Login(r.Context(), &ggid.LoginRequest{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		writeJSON(w, 401, map[string]string{"error": "login failed: " + err.Error()})
		return
	}
	writeJSON(w, 200, tokens)
}

// handleRefresh refreshes an access token
func handleRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, 405, map[string]string{"error": "method not allowed"})
		return
	}
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid body"})
		return
	}
	tokens, err := ggidClient.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		writeJSON(w, 401, map[string]string{"error": "refresh failed"})
		return
	}
	writeJSON(w, 200, tokens)
}

// handleVerify verifies a token and returns user info
func handleVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, 405, map[string]string{"error": "method not allowed"})
		return
	}
	var req struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid body"})
		return
	}
	info, err := ggidClient.VerifyToken(r.Context(), req.Token)
	if err != nil {
		writeJSON(w, 401, map[string]string{"error": "invalid token"})
		return
	}
	writeJSON(w, 200, map[string]any{
		"user_id":     info.UserID,
		"username":    info.Username,
		"email":       info.Email,
		"roles":       info.Roles,
		"scopes":      info.Scopes,
		"permissions": info.Permissions,
	})
}

// getTokenFromHeader extracts Bearer token
func getTokenFromHeader(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}

// getAuthToken convenience wrapper
func getAuthToken(r *http.Request) string {
	return getTokenFromHeader(r)
}

// currentUserID extracts user ID from JWT context
func currentUserID(r *http.Request) string {
	// Parse JWT to get sub claim
	token := getTokenFromHeader(r)
	if token == "" {
		return ""
	}
	info, err := ggidClient.VerifyToken(r.Context(), token)
	if err != nil {
		return ""
	}
	return info.UserID
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func methodAllowed(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method != method {
		writeJSON(w, 405, map[string]string{"error": "method not allowed"})
		return false
	}
	return true
}

func parseID(r *http.Request) string {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

func toString(v any) string {
	return fmt.Sprintf("%v", v)
}
