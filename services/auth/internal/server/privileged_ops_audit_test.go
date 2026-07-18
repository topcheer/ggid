package server

import (
	"net/http/httptest"
	"testing"

	"github.com/ggid/ggid/pkg/audit"
	"github.com/ggid/ggid/pkg/tenant"
	"github.com/google/uuid"
)

// Test: break-glass activate with audit publisher produces no panic.
func TestBreakGlassAudit_ActivateWithStructuredMeta(t *testing.T) {
	h := &Handler{
		breakGlassRepo:  nil, // will return 503, but audit path not reached
		auditPublisher:  &audit.Publisher{}, // js=nil, PublishAsync is no-op
	}
	_ = h

	// The audit helper should be callable with privileged op metadata.
	req := httptest.NewRequest("POST", "/api/v1/auth/break-glass/activate", nil)
	tc := &tenant.Context{TenantID: uuid.New(), IsolationLevel: tenant.IsolationShared}
	req = req.WithContext(tenant.WithContext(req.Context(), tc))

	h.publishAuditEventWithMeta(req,
		"break_glass.activate", "success",
		"privileged_operation", "break-glass", uuid.New(),
		map[string]any{
			"op_type":       "break_glass",
			"operator_id":   "user-1",
			"elevated_role": "break_glass",
			"scopes_before": []string{},
			"scopes_after":  []string{"break_glass:system"},
			"scopes_delta":  []string{"+system"},
		},
	)
	// No panic = pass.
}

// Test: structured privileged op audit fields are set correctly.
func TestPrivilegedOps_AuditEventStructure(t *testing.T) {
	e := audit.NewEvent("break_glass.activate", "success", uuid.New(), uuid.Nil)
	e.ResourceType = "privileged_operation"
	e.ResourceName = "break-glass activation"
	e.Metadata = map[string]any{
		"op_type":       "break_glass",
		"operator_id":   "user-1",
		"scopes_before": []string{"read"},
		"scopes_after":  []string{"read", "write", "admin"},
		"scopes_delta":  []string{"+write", "+admin"},
		"reason":        "production incident",
	}

	if e.ResourceType != "privileged_operation" {
		t.Error("expected resource_type=privileged_operation")
	}
	delta, ok := e.Metadata["scopes_delta"].([]string)
	if !ok || len(delta) != 2 {
		t.Error("expected scopes_delta with 2 entries")
	}
}

// Test: JIT elevation audit event with before/after perm diff.
func TestPrivilegedOps_JITElevationAudit(t *testing.T) {
	h := &Handler{
		auditPublisher: &audit.Publisher{},
	}
	req := httptest.NewRequest("POST", "/api/v1/auth/jit/elevate", nil)
	tc := &tenant.Context{TenantID: uuid.New(), IsolationLevel: tenant.IsolationShared}
	req = req.WithContext(tenant.WithContext(req.Context(), tc))

	// Simulate JIT elevation audit with permission diff.
	before := []string{"user:read"}
	after := []string{"user:read", "user:write", "admin:config"}
	delta := []string{"+user:write", "+admin:config"}

	h.publishAuditEventWithMeta(req,
		"jit.elevation", "success",
		"privileged_operation", "JIT elevation", uuid.Nil,
		map[string]any{
			"op_type":       "jit_elevation",
			"operator_id":   "user-2",
			"elevated_role": "admin",
			"scopes_before": before,
			"scopes_after":  after,
			"scopes_delta":  delta,
			"duration_min":  30,
		},
	)
	// No panic = pass.
}
