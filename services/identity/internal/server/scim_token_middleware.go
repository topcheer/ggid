package server

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	ggidtenant "github.com/ggid/ggid/pkg/tenant"
)

// scimTokenAuth is middleware that authenticates /scim/v2/ requests using
// SCIM bearer tokens (ggid_scim_*). The token's tenant_id overrides any
// X-Tenant-ID header (prevents cross-tenant access).
func (h *HTTPHandler) scimTokenAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only apply to /scim/v2/ paths.
		if !strings.HasPrefix(r.URL.Path, "/scim/v2/") {
			next.ServeHTTP(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer "+scimTokenPrefix) {
			// Not a SCIM token — let other auth handle it.
			next.ServeHTTP(w, r)
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if h.scimRepo == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"error":"SCIM not configured"}`))
			return
		}

		tokenHash := hashSCIMToken(token)
		scimToken, err := h.scimRepo.FindByHash(r.Context(), tokenHash)
		if err != nil || scimToken == nil {
			log.Printf("SCIM auth: invalid or revoked token from %s", r.RemoteAddr)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"invalid SCIM token"}`))
			return
		}

		// Check expiry.
		if scimToken.ExpiresAt != nil && scimToken.ExpiresAt.Before(time.Now()) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"SCIM token expired"}`))
			return
		}

		// Inject tenant context from the token (overrides any header).
		ctx := r.Context()
		tc := &ggidtenant.Context{
			TenantID:       scimToken.TenantID,
			IsolationLevel: ggidtenant.IsolationShared,
		}
		ctx = ggidtenant.WithContext(ctx, tc)

		// Update last_used_at (async, best-effort).
		go h.scimRepo.UpdateLastUsed(context.Background(), scimToken.ID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// hashSCIMToken hashes a SCIM token using HMAC-SHA256 with a server-side secret key.
// This is deterministic (O(1) DB lookup) but safe against DB-only leaks —
// an attacker who steals the DB cannot reverse the hashes without the HMAC key.
// The key is loaded from GGID_INTERNAL_SECRET (same env var used for internal auth).
func hashSCIMToken(plaintext string) string {
	secret := os.Getenv("GGID_INTERNAL_SECRET")
	if secret == "" {
		secret = "dev-internal-secret"
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(plaintext))
	return hex.EncodeToString(mac.Sum(nil))
}
