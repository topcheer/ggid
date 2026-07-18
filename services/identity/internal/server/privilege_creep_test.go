package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestComputeDiff_AllCases(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		roles          []string
		expected       []string
		actual         []string
		wantExcess     int
		wantMissing    int
	}{
		{
			name: "no diff", userID: "u1", roles: []string{"viewer"},
			expected: []string{"read"}, actual: []string{"read"},
			wantExcess: 0, wantMissing: 0,
		},
		{
			name: "excess only", userID: "u2", roles: []string{"viewer"},
			expected: []string{"read"}, actual: []string{"read", "write", "delete"},
			wantExcess: 2, wantMissing: 0,
		},
		{
			name: "missing only", userID: "u3", roles: []string{"admin"},
			expected: []string{"read", "write", "delete"}, actual: []string{"read"},
			wantExcess: 0, wantMissing: 2,
		},
		{
			name: "both", userID: "u4", roles: []string{"editor"},
			expected: []string{"read", "write"}, actual: []string{"read", "delete"},
			wantExcess: 1, wantMissing: 1,
		},
		{
			name: "empty actual", userID: "u5", roles: []string{"viewer"},
			expected: []string{"read"}, actual: []string{},
			wantExcess: 0, wantMissing: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff := ComputeDiff(tt.userID, tt.roles, tt.expected, tt.actual)
			if len(diff.ExcessPermissions) != tt.wantExcess {
				t.Errorf("excess: got %d, want %d (%v)", len(diff.ExcessPermissions), tt.wantExcess, diff.ExcessPermissions)
			}
			if len(diff.MissingPermissions) != tt.wantMissing {
				t.Errorf("missing: got %d, want %d (%v)", len(diff.MissingPermissions), tt.wantMissing, diff.MissingPermissions)
			}
		})
	}
}

func TestPrivilegeCreep_AlertsNotConfigured(t *testing.T) {
	h := &HTTPHandler{}
	req := httptest.NewRequest("GET", "/api/v1/identity/privilege-creep/alerts", nil)
	w := httptest.NewRecorder()
	h.handlePrivilegeCreep(w, req)
	if w.Code != http.StatusOK && w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 200 or 503, got %d", w.Code)
	}
}

func TestPrivilegeCreep_ScanNotConfigured(t *testing.T) {
	h := &HTTPHandler{}
	req := httptest.NewRequest("POST", "/api/v1/identity/privilege-creep/scan", nil)
	w := httptest.NewRecorder()
	h.handlePrivilegeCreep(w, req)
	if w.Code != http.StatusOK && w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 200 or 503, got %d", w.Code)
	}
}

func TestPrivilegeCreep_DiffRoute(t *testing.T) {
	h := &HTTPHandler{}
	req := httptest.NewRequest("GET", "/api/v1/identity/privilege-creep/diff/user-123", nil)
	w := httptest.NewRecorder()
	h.handlePrivilegeCreep(w, req)
	if w.Code != http.StatusOK && w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 200 or 503, got %d", w.Code)
	}
}

func TestPrivilegeCreep_WrongMethod(t *testing.T) {
	h := &HTTPHandler{}
	req := httptest.NewRequest("DELETE", "/api/v1/identity/privilege-creep/alerts", nil)
	w := httptest.NewRecorder()
	h.handlePrivilegeCreep(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestExtractRoleIDFromPath(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/api/v1/policy/roles/admin/baseline", "admin"},
		{"/api/v1/policy/roles/viewer/baseline", "viewer"},
		{"/api/v1/policy/roles/", ""},
		{"/no/roles/here", ""},
	}
	for _, tt := range tests {
		got := extractRoleIDFromPath(tt.path)
		if got != tt.want {
			t.Errorf("extractRoleIDFromPath(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}
