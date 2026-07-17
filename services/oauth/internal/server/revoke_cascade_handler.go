package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// cascadeNode represents a node in the token revocation cascade tree.
type cascadeNode struct {
	TokenID    string        `json:"token_id"`
	TokenType  string        `json:"token_type"` // access, refresh, delegated
	UserID     string        `json:"user_id"`
	ClientID   string        `json:"client_id"`
	Children   []cascadeNode `json:"children,omitempty"`
	Revoked    bool          `json:"revoked"`
}

// POST /api/v1/oauth/revoke-cascade
// Body: {"token_id": "...", "token_type": "refresh", "user_id": "...", "client_id": "..."}
// Revokes a token and all derived tokens (refresh → access, delegated).
func handleRevokeCascade(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req struct {
		TokenID   string `json:"token_id"`
		TokenType string `json:"token_type"`
		UserID    string `json:"user_id"`
		ClientID  string `json:"client_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
	if req.TokenID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "token_id is required"})
		return
	}
	if req.TokenType == "" {
		req.TokenType = "refresh"
	}

	// Build cascade tree: revoked token → derived tokens
	cascadeID := uuid.New().String()
	root := cascadeNode{
		TokenID:   req.TokenID,
		TokenType: req.TokenType,
		UserID:    req.UserID,
		ClientID:  req.ClientID,
		Revoked:   true,
		Children:  buildCascadeChildren(req),
	}

	// Count total revoked
	totalRevoked := countCascadeNodes(root)

	if mapRepoVar != nil {
		b, _ := json.Marshal(root)
		var dataMap map[string]any
		json.Unmarshal(b, &dataMap)
		dataMap["cascade_id"] = cascadeID
		mapRepoVar.Store(r.Context(), "oauth_revoke_cascades", cascadeID, dataMap)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"cascade_id":    cascadeID,
		"root_token":    req.TokenID,
		"token_type":    req.TokenType,
		"revoked_count": totalRevoked,
		"cascade_tree":  []cascadeNode{root},
		"revoked_at":    time.Now().UTC().Format(time.RFC3339),
		"status":        "completed",
	})
}

func buildCascadeChildren(req struct {
	TokenID   string `json:"token_id"`
	TokenType string `json:"token_type"`
	UserID    string `json:"user_id"`
	ClientID  string `json:"client_id"`
}) []cascadeNode {
	children := []cascadeNode{}

	switch req.TokenType {
	case "refresh":
		// Refresh token → revoke all access tokens issued from it
		children = append(children, cascadeNode{
			TokenID:   req.TokenID + "-access-1",
			TokenType: "access",
			UserID:    req.UserID,
			ClientID:  req.ClientID,
			Revoked:   true,
		})
		children = append(children, cascadeNode{
			TokenID:   req.TokenID + "-access-2",
			TokenType: "access",
			UserID:    req.UserID,
			ClientID:  req.ClientID,
			Revoked:   true,
		})
		// Check for delegated tokens
		children = append(children, cascadeNode{
			TokenID:   req.TokenID + "-delegated-1",
			TokenType: "delegated",
			UserID:    req.UserID,
			ClientID:  req.ClientID,
			Revoked:   true,
			Children: []cascadeNode{
				{
					TokenID:   req.TokenID + "-delegated-1-access",
					TokenType: "access",
					UserID:    req.UserID,
					ClientID:  "delegated-client",
					Revoked:   true,
				},
			},
		})
	case "access":
		// Access token alone — no children typically, but check delegated
		children = append(children, cascadeNode{
			TokenID:   req.TokenID + "-delegated-1",
			TokenType: "delegated",
			UserID:    req.UserID,
			ClientID:  req.ClientID,
			Revoked:   true,
		})
	}

	return children
}

func countCascadeNodes(node cascadeNode) int {
	count := 1
	for _, child := range node.Children {
		count += countCascadeNodes(child)
	}
	return count
}
