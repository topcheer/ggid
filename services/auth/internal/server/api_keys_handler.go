package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	ggidcrypto "github.com/ggid/ggid/pkg/crypto"
	ggidtenant "github.com/ggid/ggid/pkg/tenant"
	"github.com/google/uuid"
)

// APIKey represents an API key for programmatic access (API response DTO).
type APIKey struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Scopes    []string   `json:"scopes"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	LastUsed  *time.Time `json:"last_used,omitempty"`
	Status    string     `json:"status"`
	// Key is the plaintext secret, returned ONLY in the creation response.
	Key string `json:"key,omitempty"`
}

// toAPIKey converts a DB record to the API response DTO (without key_hash).
func (r *APIKeyRecord) toAPIKey() APIKey {
	return APIKey{
		ID:        r.ID.String(),
		Name:      r.Name,
		Scopes:    r.Scopes,
		CreatedAt: r.CreatedAt,
		ExpiresAt: r.ExpiresAt,
		LastUsed:  r.LastUsedAt,
		Status:    r.Status,
	}
}

// handleAPIKeys handles GET/POST /api/v1/auth/api-keys and sub-paths.
func (h *Handler) handleAPIKeys(w http.ResponseWriter, r *http.Request) {
	// All API key operations require a DB pool.
	if h.pool == nil {
		writeError(w, http.StatusServiceUnavailable, "database not available")
		return
	}
	repo := newAPIKeyRepo(h.pool)

	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing tenant context")
		return
	}

	switch {
	// --- List ---
	case r.URL.Path == "/api/v1/auth/api-keys" && r.Method == http.MethodGet:
		records, err := repo.ListByTenant(r.Context(), tc.TenantID)
		if err != nil {
			slog.Error("api_keys: list failed", "error", err)
			writeError(w, http.StatusInternalServerError, "failed to list api keys")
			return
		}
		keys := make([]APIKey, 0, len(records))
		for i := range records {
			keys = append(keys, records[i].toAPIKey())
		}
		writeJSON(w, http.StatusOK, keys)

	// --- Create ---
	case r.URL.Path == "/api/v1/auth/api-keys" && r.Method == http.MethodPost:
		var req struct {
			Name      string   `json:"name"`
			Scopes    []string `json:"scopes"`
			ExpiresAt string   `json:"expires_at"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.Name == "" {
			writeError(w, http.StatusBadRequest, "name is required")
			return
		}

		// Generate the key ID first so we can embed it in the secret.
		keyID := uuid.New()
		plain, err := ggidcrypto.GenerateRandomToken(24)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to generate api key")
			return
		}
		// Format: ggid_sk_<keyID_hex>_<random_secret>
		// The keyID enables O(1) DB lookup; the random part is verified via Argon2id.
		secret := "ggid_sk_" + keyID.String() + "_" + plain

		var expiresAt *time.Time
		if req.ExpiresAt != "" {
			t, perr := time.Parse(time.RFC3339, req.ExpiresAt)
			if perr != nil {
				writeError(w, http.StatusBadRequest, "invalid expires_at format, expected RFC3339")
				return
			}
			expiresAt = &t
		}

		rec, err := repo.CreateWithID(r.Context(), tc.TenantID, keyID, req.Name, secret, req.Scopes, expiresAt)
		if err != nil {
			slog.Error("api_keys: create failed", "error", err)
			writeError(w, http.StatusInternalServerError, "failed to create api key")
			return
		}

		// Audit: API key created
		h.publishAuditEvent("api_key.create", "success", tc.TenantID, uuid.Nil)

		// Return the plaintext secret exactly once — it cannot be retrieved later.
		resp := rec.toAPIKey()
		resp.Key = secret
		writeJSON(w, http.StatusCreated, resp)

	// --- Rotate: POST /api/v1/auth/api-keys/{id}/rotate ---
	case strings.HasPrefix(r.URL.Path, "/api/v1/auth/api-keys/") && r.Method == http.MethodPost:
		parts := splitPath(r.URL.Path)
		if len(parts) >= 6 && parts[5] == "rotate" {
			keyID, perr := uuid.Parse(parts[4])
			if perr != nil {
				writeError(w, http.StatusBadRequest, "invalid api key id")
				return
			}

			plain, err := ggidcrypto.GenerateRandomToken(24)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "failed to generate api key")
				return
			}
			secret := "ggid_sk_" + keyID.String() + "_" + plain

			if err := repo.Rotate(r.Context(), tc.TenantID, keyID, secret); err != nil {
				slog.Error("api_keys: rotate failed", "error", err)
				writeError(w, http.StatusInternalServerError, "failed to rotate api key")
				return
			}

			// Fetch the updated record for the response.
			rec, err := repo.GetByID(r.Context(), tc.TenantID, keyID)
			if err != nil || rec == nil {
				writeError(w, http.StatusNotFound, "API key not found")
				return
			}
			resp := rec.toAPIKey()
			resp.Key = secret
			writeJSON(w, http.StatusOK, resp)
			return
		}
		writeError(w, http.StatusNotFound, "unknown path")

	// --- Delete (revoke): DELETE /api/v1/auth/api-keys/{id} ---
	case strings.HasPrefix(r.URL.Path, "/api/v1/auth/api-keys/") && r.Method == http.MethodDelete:
		parts := splitPath(r.URL.Path)
		if len(parts) < 5 {
			writeError(w, http.StatusBadRequest, "api key id is required")
			return
		}
		keyID, perr := uuid.Parse(parts[4])
		if perr != nil {
			writeError(w, http.StatusBadRequest, "invalid api key id")
			return
		}

		if err := repo.UpdateStatus(r.Context(), tc.TenantID, keyID, "revoked"); err != nil {
			slog.Error("api_keys: revoke failed", "error", err)
			writeError(w, http.StatusInternalServerError, "failed to revoke api key")
			return
		}

		// Audit: API key revoked
		h.publishAuditEvent("api_key.delete", "success", tc.TenantID, uuid.Nil)

		writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// splitPath splits a URL path into segments.
func splitPath(path string) []string {
	var parts []string
	for _, p := range strings.Split(path, "/") {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}


