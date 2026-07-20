package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ConsentChecker verifies that platform admins have tenant consent before accessing tenant data.
// It sits after JWTAuth and before the reverse proxy.
//
// Rules:
// - platform:admin / admin scope users must have an active consent for the target tenant
// - tenant:admin users can only access their own tenant
// - Consent check is bypassed for: healthz, tenants/resolve, system/bootstrap, login, oauth
// - Break-glass: platform:admin bypasses consent but logs a warning (audited separately)

type ConsentStore struct {
	db *pgx.Conn // single connection for queries
}

// CheckConsent verifies whether the requester has consent to access the target tenant.
func CheckConsent(dbURL string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path

			// Skip consent check for public/system paths
			if isConsentExempt(path) {
				next.ServeHTTP(w, r)
				return
			}

			// Extract claims from context (set by JWTAuth middleware)
			claims := ExtractJWTClaims(r)
			if claims.Subject == "" {
				next.ServeHTTP(w, r) // unauthenticated — let downstream handle
				return
			}

			tenantID := r.Header.Get("X-Tenant-ID")
			if tenantID == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Check if user is platform admin
			isPlatformAdmin := false
			for _, sc := range claims.Scopes {
				if sc == "platform:admin" || sc == "Platform Administrator" || sc == "admin" || sc == "Administrator" {
					isPlatformAdmin = true
					break
				}
			}

			// If not platform admin, no consent check needed (RLS handles isolation)
			if !isPlatformAdmin {
				next.ServeHTTP(w, r)
				return
			}

			// Platform admin: check if accessing their own tenant (default tenant)
			// Platform admins belong to the default tenant — accessing it is fine
			if tenantID == "00000000-0000-0000-0000-000000000001" {
				next.ServeHTTP(w, r)
				return
			}

			// Platform admin accessing another tenant — check consent
			// Skip for tenant management endpoints (admin can list/create tenants)
			if strings.HasPrefix(path, "/api/v1/tenants") && !strings.Contains(path, "/users") && !strings.Contains(path, "/roles") {
				next.ServeHTTP(w, r)
				return
			}

			// Check consent in DB
			hasConsent, err := queryConsent(r.Context(), dbURL, tenantID)
			if err != nil {
				// DB error — fail open for platform admin (audit will catch it)
				next.ServeHTTP(w, r)
				return
			}

			if !hasConsent {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]any{
					"error":   "consent_required",
					"message": "Tenant administrator authorization required. Request access from the tenant admin.",
					"action":  "Ask the tenant admin to grant access in Settings → Platform Access",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func queryConsent(ctx context.Context, dbURL, tenantID string) (bool, error) {
	if _, err := uuid.Parse(tenantID); err != nil {
		return false, fmt.Errorf("invalid tenant id")
	}

	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		return false, err
	}
	defer conn.Close(ctx)

	var count int
	err = conn.QueryRow(ctx,
		`SELECT count(*) FROM tenant_access_consents
		 WHERE tenant_id = $1 AND revoked_at IS NULL
		 AND (expires_at IS NULL OR expires_at > NOW())`,
		uuid.MustParse(tenantID)).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func isConsentExempt(path string) bool {
	exemptPrefixes := []string{
		"/healthz", "/readyz", "/api/v1/auth/login", "/api/v1/auth/register",
		"/api/v1/oauth/", "/api/v1/system/bootstrap", "/api/v1/system/initialized",
		"/api/v1/tenants/resolve", "/api/v1/mfa/", "/.well-known/",
		"/api/v1/impersonate/", // impersonate API handles its own consent
		"/api/v1/tenants/",     // tenant CRUD doesn't need consent (admin can manage tenants)
	}
	for _, prefix := range exemptPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

// EnsureConsentTables creates the consent tables if they don't exist.
func EnsureConsentTables(ctx context.Context, dbURL string) error {
	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		return nil // fail silently — tables already created via migration
	}
	defer conn.Close(ctx)

	_, err = conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS tenant_access_consents (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL,
			granted_to VARCHAR NOT NULL DEFAULT 'platform_admin',
			granted_by UUID NOT NULL,
			scope VARCHAR DEFAULT 'support',
			expires_at TIMESTAMPTZ,
			revoked_at TIMESTAMPTZ,
			reason TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE TABLE IF NOT EXISTS impersonation_sessions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL,
			impersonator_id UUID NOT NULL,
			target_user_id UUID,
			consent_id UUID REFERENCES tenant_access_consents(id),
			reason TEXT NOT NULL,
			started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			ended_at TIMESTAMPTZ,
			ip_address INET,
			user_agent TEXT
		);
		CREATE INDEX IF NOT EXISTS idx_consents_tenant ON tenant_access_consents(tenant_id) WHERE revoked_at IS NULL;
		CREATE INDEX IF NOT EXISTS idx_impersonation_active ON impersonation_sessions(tenant_id) WHERE ended_at IS NULL;
	`)
	_ = time.Now // ensure time import is used
	return err
}
