package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// seedSnapshot populates Redis with a canned route-permission snapshot.
func seedSnapshot(t *testing.T, rdb *redis.Client, rows []routePermRow) {
	t.Helper()
	data, err := json.Marshal(rows)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := rdb.Set(context.Background(), rbacCacheKey, data, 0).Err(); err != nil {
		t.Fatalf("seed redis: %v", err)
	}
}

func TestRBACResolver_UnavailableWithoutData(t *testing.T) {
	r := NewRBACResolver(nil, "") // no redis, no db
	if r.Available() {
		t.Error("resolver should be unavailable with no data source")
	}
	_, handled := r.CheckAccess(context.Background(), "/api/v1/users", http.MethodGet, JWTCClaims{})
	if handled {
		t.Error("unavailable resolver must not handle decisions")
	}
}

func TestRBACResolver_RoleGrantAndDeny(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	seedSnapshot(t, rdb, []routePermRow{
		{RoleName: "Administrator", RoleKey: "admin", Prefix: "/api/v1/users", Level: "admin"},
		{RoleName: "Viewer", RoleKey: "viewer", Prefix: "/api/v1/users", Level: "read"},
	})

	r := NewRBACResolver(rdb, "")
	r.WarmStart(context.Background())
	if !r.Available() {
		t.Fatal("resolver should be available after warm start")
	}
	ctx := context.Background()

	// Viewer can GET (read).
	allow, handled := r.CheckAccess(ctx, "/api/v1/users", http.MethodGet, JWTCClaims{Roles: []string{"Viewer"}})
	if !handled || !allow {
		t.Errorf("viewer GET: allow=%v handled=%v", allow, handled)
	}
	// Viewer cannot POST (requires write).
	allow, handled = r.CheckAccess(ctx, "/api/v1/users", http.MethodPost, JWTCClaims{Roles: []string{"Viewer"}})
	if !handled || allow {
		t.Errorf("viewer POST: allow=%v handled=%v", allow, handled)
	}
	// Administrator (role name) can POST.
	allow, handled = r.CheckAccess(ctx, "/api/v1/users", http.MethodPost, JWTCClaims{Roles: []string{"Administrator"}})
	if !handled || !allow {
		t.Errorf("admin POST: allow=%v handled=%v", allow, handled)
	}
	// Role key match also works.
	allow, handled = r.CheckAccess(ctx, "/api/v1/users", http.MethodDelete, JWTCClaims{Roles: []string{"admin"}})
	if !handled || !allow {
		t.Errorf("admin key DELETE: allow=%v handled=%v", allow, handled)
	}
	// Unknown role denied.
	allow, handled = r.CheckAccess(ctx, "/api/v1/users", http.MethodGet, JWTCClaims{Roles: []string{"Stranger"}})
	if !handled || allow {
		t.Errorf("stranger GET: allow=%v handled=%v", allow, handled)
	}
	// Unmatched path → not handled (static fallback).
	_, handled = r.CheckAccess(ctx, "/api/v1/unknown", http.MethodGet, JWTCClaims{Roles: []string{"Viewer"}})
	if handled {
		t.Error("unmatched path should not be handled dynamically")
	}
}

func TestRBACResolver_AdminBypass(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()
	seedSnapshot(t, rdb, []routePermRow{})

	r := NewRBACResolver(rdb, "")
	r.WarmStart(context.Background())

	// platform:admin scope bypasses all dynamic rules.
	allow, handled := r.CheckAccess(context.Background(), "/api/v1/anything", http.MethodDelete,
		JWTCClaims{Scopes: []string{"platform:admin"}})
	if !allow || !handled {
		t.Errorf("admin bypass: allow=%v handled=%v", allow, handled)
	}
	// Admin role name in roles claim also bypasses.
	allow, handled = r.CheckAccess(context.Background(), "/api/v1/anything", http.MethodDelete,
		JWTCClaims{Roles: []string{"Tenant Administrator"}})
	if !allow || !handled {
		t.Errorf("tenant admin bypass: allow=%v handled=%v", allow, handled)
	}
}

func TestRBACResolver_LongestPrefixWins(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	seedSnapshot(t, rdb, []routePermRow{
		{RoleName: "Viewer", RoleKey: "viewer", Prefix: "/api/v1/users", Level: "read"},
		{RoleName: "Manager", RoleKey: "manager", Prefix: "/api/v1/users/me", Level: "write"},
	})
	r := NewRBACResolver(rdb, "")
	r.WarmStart(context.Background())
	ctx := context.Background()

	// Longer prefix (/users/me) governs: manager has write there.
	allow, handled := r.CheckAccess(ctx, "/api/v1/users/me/settings", http.MethodPost, JWTCClaims{Roles: []string{"Manager"}})
	if !handled || !allow {
		t.Errorf("manager POST /users/me: allow=%v handled=%v", allow, handled)
	}
	// Viewer only matched on the shorter prefix → no grant at longer prefix.
	allow, handled = r.CheckAccess(ctx, "/api/v1/users/me/settings", http.MethodGet, JWTCClaims{Roles: []string{"Viewer"}})
	if !handled || allow {
		t.Errorf("viewer under longer prefix: allow=%v handled=%v", allow, handled)
	}
}

func TestRBACResolver_StaleMemoryFallback(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()
	seedSnapshot(t, rdb, []routePermRow{
		{RoleName: "Viewer", RoleKey: "viewer", Prefix: "/api/v1/users", Level: "read"},
	})
	r := NewRBACResolver(rdb, "")
	r.WarmStart(context.Background())

	// Kill Redis — resolver must keep serving the stale in-memory snapshot.
	mr.Close()
	allow, handled := r.CheckAccess(context.Background(), "/api/v1/users", http.MethodGet, JWTCClaims{Roles: []string{"Viewer"}})
	if !handled || !allow {
		t.Errorf("stale fallback: allow=%v handled=%v", allow, handled)
	}
}

func TestRequireAdminScope_DynamicAndFallback(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()
	seedSnapshot(t, rdb, []routePermRow{
		{RoleName: "Viewer", RoleKey: "viewer", Prefix: "/api/v1/users", Level: "read"},
	})
	res := NewRBACResolver(rdb, "")
	res.WarmStart(context.Background())
	SetRBACResolver(res)
	defer SetRBACResolver(nil)

	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mkJWT := func(roles ...string) string {
		payload, _ := json.Marshal(map[string]any{"sub": "u1", "roles": roles})
		return "Bearer x." + b64url(payload) + ".y"
	}

	// Dynamic allow: viewer GET.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.Header.Set("Authorization", mkJWT("Viewer"))
	rec := httptest.NewRecorder()
	RequireAdminScope(okHandler).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("viewer GET: %d, want 200", rec.Code)
	}

	// Dynamic deny: viewer POST.
	req = httptest.NewRequest(http.MethodPost, "/api/v1/users", nil)
	req.Header.Set("Authorization", mkJWT("Viewer"))
	rec = httptest.NewRecorder()
	RequireAdminScope(okHandler).ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("viewer POST: %d, want 403", rec.Code)
	}

	// Fallback path: resolver has no rule for /api/v1/audit/ → static admin
	// prefix list applies; non-admin role denied.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/audit/events", nil)
	req.Header.Set("Authorization", mkJWT("Viewer"))
	rec = httptest.NewRecorder()
	RequireAdminScope(okHandler).ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("static fallback audit: %d, want 403", rec.Code)
	}

	// Fallback path: non-admin path passes.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/flows", nil)
	req.Header.Set("Authorization", mkJWT("Viewer"))
	rec = httptest.NewRecorder()
	RequireAdminScope(okHandler).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("non-admin path: %d, want 200", rec.Code)
	}
}

func b64url(b []byte) string {
	const enc = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	out := make([]byte, 0, len(b)*2)
	for i := 0; i < len(b); i += 3 {
		var n uint32
		remain := len(b) - i
		n = uint32(b[i]) << 16
		if remain > 1 {
			n |= uint32(b[i+1]) << 8
		}
		if remain > 2 {
			n |= uint32(b[i+2])
		}
		out = append(out, enc[(n>>18)&63], enc[(n>>12)&63])
		if remain > 1 {
			out = append(out, enc[(n>>6)&63])
		}
		if remain > 2 {
			out = append(out, enc[n&63])
		}
	}
	return string(out)
}
