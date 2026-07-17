package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net/http"
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

// hashSCIMToken hashes a SCIM token plaintext using SHA-256 for DB lookup.
func hashSCIMToken(plaintext string) string {
	h := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(h[:])
}
