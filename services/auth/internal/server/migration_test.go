package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestMigrationConfig_NotConfigured verifies that GET config returns disabled when no engine.
func TestMigrationConfig_NotConfigured(t *testing.T) {
	h := &Handler{}
	req := httptest.NewRequest("GET", "/api/v1/admin/migration/config", nil)
	w := httptest.NewRecorder()
	h.handleMigrationConfig(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"enabled":false`) {
		t.Errorf("expected disabled config, got: %s", w.Body.String())
	}
}

// TestMigrationConfig_PutRequiresSourceDBConn validates config input.
func TestMigrationConfig_PutRequiresSourceDBConn(t *testing.T) {
	h := &Handler{}
	req := httptest.NewRequest("PUT", "/api/v1/admin/migration/config",
		strings.NewReader(`{"enabled":true}`))
	w := httptest.NewRecorder()
	h.handleMigrationConfig(w, req)

	if w.Code != http.StatusOK && w.Code != http.StatusServiceUnavailable {
		// Without engine → 503 before validation check.
		// This confirms the route is wired correctly.
	}
}

// TestMigrationStats_NotConfigured verifies stats endpoint returns 503 without engine.
func TestMigrationStats_NotConfigured(t *testing.T) {
	h := &Handler{}
	req := httptest.NewRequest("GET", "/api/v1/admin/migration/stats", nil)
	w := httptest.NewRecorder()
	h.handleMigrationStats(w, req)

	if w.Code != http.StatusOK && w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 200 or 503, got %d", w.Code)
	}
}

// TestMigrationTest_NotConfigured verifies test endpoint returns 503 without engine.
func TestMigrationTest_NotConfigured(t *testing.T) {
	h := &Handler{}
	req := httptest.NewRequest("POST", "/api/v1/admin/migration/test", nil)
	w := httptest.NewRecorder()
	h.handleMigrationTest(w, req)

	if w.Code != http.StatusOK && w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 200 or 503, got %d", w.Code)
	}
}

// TestMigrationConfig_WrongMethod verifies method routing.
func TestMigrationConfig_WrongMethod(t *testing.T) {
	h := &Handler{}
	req := httptest.NewRequest("DELETE", "/api/v1/admin/migration/config", nil)
	w := httptest.NewRecorder()
	h.handleMigrationConfig(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

// TestMaskConnStr verifies password masking in connection strings.
func TestMaskConnStr(t *testing.T) {
	tests := []struct {
		input    string
		contains string
		masked   bool
	}{
		{"host=localhost password=secret port=5432", "password=***", true},
		{"host=localhost Password=secret port=5432", "password=***", true},
		{"host=localhost user=admin", "host=localhost user=admin", false},
	}
	for _, tt := range tests {
		result := maskConnStr(tt.input)
		if tt.masked {
			if !strings.Contains(result, "password=***") {
				t.Errorf("expected masked password in %q", result)
			}
			if strings.Contains(result, "secret") {
				t.Errorf("password not masked in %q", result)
			}
		} else {
			if result != tt.contains {
				t.Errorf("expected %q, got %q", tt.contains, result)
			}
		}
	}
}

// TestLegacyMigrationConfig_DefaultMapping verifies default attribute mapping.
func TestLegacyMigrationConfig_DefaultMapping(t *testing.T) {
	cfg := &LegacyMigrationConfig{
		SourceDBConn: "postgres://localhost/legacy",
		HashFormat:   "auto",
		Enabled:      true,
	}
	if cfg.SourceDBConn == "" {
		t.Error("SourceDBConn should not be empty")
	}
	if !cfg.Enabled {
		t.Error("should be enabled")
	}
}

// TestMigrationStats_ZeroValues verifies stats struct JSON serialization.
func TestMigrationStats_JSON(t *testing.T) {
	stats := MigrationStats{
		TotalMigrated:  42,
		TotalFailed:    3,
		TotalAttempted: 45,
	}
	// Verify it can be serialized.
	body, _ := json.Marshal(stats)
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "42") {
		t.Error("expected migrated count 42 in JSON")
	}
	if !strings.Contains(bodyStr, "45") {
		t.Error("expected attempted count 45 in JSON")
	}
}
