package server

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type PasskeyRegistrationSession struct {
	SessionID    string    `json:"session_id"`
	UserID       string    `json:"user_id"`
	Challenge    string    `json:"challenge"`
	RPID         string    `json:"rp_id"`
	CreatedAt    time.Time `json:"created_at"`
	Status       string    `json:"status"`
}

type PasskeyAuthSession struct {
	SessionID    string    `json:"session_id"`
	Challenge    string    `json:"challenge"`
	RPID         string    `json:"rp_id"`
	CreatedAt    time.Time `json:"created_at"`
	Status       string    `json:"status"`
}

type PasskeyCredential struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	PublicKey  string    `json:"public_key"`
	Counter    int       `json:"counter"`
	CreatedAt  time.Time `json:"created_at"`
	Revoked    bool      `json:"revoked"`
}

var (
	pkRegSessions   = make(map[string]*PasskeyRegistrationSession)
	pkAuthSessions  = make(map[string]*PasskeyAuthSession)
	pkCredentials   = make(map[string]*PasskeyCredential)
	pkMu            sync.RWMutex
	pkSeq           int
)

// resolveWebAuthnRPID reads RP ID from sys_config DB table first, falls back to env.
// Returns error if neither source provides a value.
func resolveWebAuthnRPID(h *Handler) (string, error) {
	// 1. Try DB
	if h.pool != nil {
		var configJSON []byte
		err := h.pool.QueryRow(context.Background(),
			`SELECT value::text FROM sys_config WHERE key = 'webauthn_config'`).Scan(&configJSON)
		if err == nil && len(configJSON) > 0 {
			var cfg struct {
				RPID string `json:"rp_id"`
			}
			if json.Unmarshal(configJSON, &cfg) == nil && cfg.RPID != "" {
				return cfg.RPID, nil
			}
		}
	}
	// 2. Fallback to env
	if rpID := os.Getenv("WEBAUTHN_RP_ID"); rpID != "" {
		return rpID, nil
	}
	return "", fmt.Errorf("WebAuthn RP ID not configured — set via /api/v1/system/config or WEBAUTHN_RP_ID env")
}

// resolveRPIDForConfig returns RP ID for display purposes (no error).
func resolveRPIDForConfig(h *Handler) string {
	rpID, err := resolveWebAuthnRPID(h)
	if err != nil {
		return ""
	}
	return rpID
}

func (h *Handler) handlePasskeyRevoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	parts := strings.Split(strings.TrimSuffix(r.URL.Path, "/"), "/")
	if len(parts) < 1 {
		writeError(w, http.StatusBadRequest, "credential id required")
		return
	}
	id := parts[len(parts)-1]

	// Revoke in DB
	if h.pool != nil {
		tag, err := h.pool.Exec(r.Context(), `
			UPDATE auth_passkey_credentials SET revoked = true WHERE id = $1`, id)
		if err != nil || tag.RowsAffected() == 0 {
			// Try in-memory fallback
			pkMu.Lock()
			if cred, ok := pkCredentials[id]; ok {
				cred.Revoked = true
			}
			pkMu.Unlock()
		}
	} else {
		pkMu.Lock()
		defer pkMu.Unlock()
		if cred, ok := pkCredentials[id]; ok {
			cred.Revoked = true
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"status": "revoked", "id": id})
}

func (h *Handler) handlePasskeyStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Query from DB
	if h.pool != nil {
		userID := r.URL.Query().Get("user_id")
		rows, err := h.pool.Query(r.Context(), `
			SELECT id, credential_id, device_name, platform, created_at,
			       COALESCE(last_used, created_at), transports::text, backup_eligible
			FROM auth_passkey_credentials
			WHERE revoked = false AND ($1 = '' OR user_id = $1)
			ORDER BY created_at DESC`, userID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to query passkeys")
			return
		}
		defer rows.Close()

		passkeys := []map[string]any{}
		for rows.Next() {
			var id, credID, deviceName, platform string
			var createdAt, lastUsed time.Time
			var transportsStr string
			var backupEligible bool
			if err := rows.Scan(&id, &credID, &deviceName, &platform, &createdAt, &lastUsed, &transportsStr, &backupEligible); err != nil {
				continue
			}
			var transports []string
			_ = json.Unmarshal([]byte(transportsStr), &transports)
			passkeys = append(passkeys, map[string]any{
				"id":             id,
				"device_name":    deviceName,
				"platform":       platform,
				"credential_id":  credID,
				"created_at":     createdAt,
				"last_used":      lastUsed,
				"transports":     transports,
				"backup_eligible": backupEligible,
				"sync_status":    "synced",
			})
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"passkeys": passkeys, "total": len(passkeys)})
		return
	}

	// Fallback to in-memory
	pkMu.RLock()
	defer pkMu.RUnlock()
	passkeys := []map[string]any{}
	for _, c := range pkCredentials {
		if !c.Revoked {
			passkeys = append(passkeys, map[string]any{
				"id": c.ID, "device_name": "", "platform": "",
				"credential_id": c.ID, "created_at": c.CreatedAt,
			})
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"passkeys": passkeys, "total": len(passkeys)})
}

func fmtPKID(n int) string {
	const hex = "0123456789abcdef"
	if n == 0 {
		return "pk_0"
	}
	var buf []byte
	for n > 0 {
		buf = append([]byte{hex[n%16]}, buf...)
		n /= 16
	}
	return "pk_" + string(buf)
}

// --- map[string]any type-assertion helpers ---

func getString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		switch s := v.(type) {
		case string:
			return s
		case fmt.Stringer:
			return s.String()
		default:
			return fmt.Sprintf("%v", v)
		}
	}
	return ""
}

func getInt(m map[string]any, key string) int {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case int:
			return n
		case int64:
			return int(n)
		case float64:
			return int(n)
		case json.Number:
			i, _ := n.Int64()
			return int(i)
		}
	}
	return 0
}

func getBool(m map[string]any, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

// generateChallenge creates a cryptographically random challenge for WebAuthn.
func generateChallenge() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// Fallback to UUID-based (still unique, just not crypto-random)
		return base64.StdEncoding.EncodeToString([]byte(uuid.New().String()))
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// hasAdminScope checks if the request has platform:admin or tenant:admin scope.
// The gateway injects X-Is-Admin header after JWT validation for admin users.
func hasAdminScope(r *http.Request) bool {
	// X-Is-Admin is set by gateway JWTAuth middleware for platform/tenant admins
	if r.Header.Get("X-Is-Admin") == "true" {
		return true
	}
	// Fallback: check X-User-Role header (also set by gateway)
	role := r.Header.Get("X-User-Role")
	return role == "platform:admin" || role == "tenant:admin"
}