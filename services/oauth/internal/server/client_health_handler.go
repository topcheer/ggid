package server

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

// clientHealth represents the health status of an OAuth client.
type clientHealth struct {
	ClientID       string         `json:"client_id"`
	ActiveTokens   int            `json:"active_tokens"`
	RecentErrors   int            `json:"recent_errors_24h"`
	ErrorRate      float64        `json:"error_rate_pct"`
	CertStatus     string         `json:"cert_status"`
	SecretStatus   string         `json:"secret_status"`
	LastUsed       string         `json:"last_used"`
	OverallHealth  string         `json:"overall_health"` // healthy, warning, critical
	Checks         []map[string]any `json:"checks"`
}

var clientHealthStore = struct {
	sync.RWMutex
	data map[string]*clientHealth
}{data: map[string]*clientHealth{
	"web-app": {
		ClientID: "web-app", ActiveTokens: 320, RecentErrors: 2, ErrorRate: 0.6,
		CertStatus: "valid", SecretStatus: "valid",
		LastUsed: time.Now().UTC().Add(-5 * time.Minute).Format(time.RFC3339),
		OverallHealth: "healthy",
	},
	"mobile-ios": {
		ClientID: "mobile-ios", ActiveTokens: 180, RecentErrors: 15, ErrorRate: 4.2,
		CertStatus: "valid", SecretStatus: "expiring_soon",
		LastUsed: time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339),
		OverallHealth: "warning",
	},
	"admin-cli": {
		ClientID: "admin-cli", ActiveTokens: 12, RecentErrors: 0, ErrorRate: 0.0,
		CertStatus: "valid", SecretStatus: "valid",
		LastUsed: time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339),
		OverallHealth: "healthy",
	},
	"service-backend": {
		ClientID: "service-backend", ActiveTokens: 5, RecentErrors: 42, ErrorRate: 12.5,
		CertStatus: "expired", SecretStatus: "valid",
		LastUsed: time.Now().UTC().Add(-30 * time.Minute).Format(time.RFC3339),
		OverallHealth: "critical",
	},
}}

// GET /api/v1/oauth/clients/{id}/health
func handleClientHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	clientID := strings.TrimPrefix(r.URL.Path, "/api/v1/oauth/clients/")
	clientID = strings.TrimSuffix(clientID, "/health")
	clientID = strings.TrimSuffix(clientID, "/")
	if clientID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "client_id is required"})
		return
	}

	clientHealthStore.RLock()
	health, exists := clientHealthStore.data[clientID]
	clientHealthStore.RUnlock()

	if !exists {
		writeJSON(w, http.StatusOK, map[string]any{
			"client_id":      clientID,
			"overall_health": "unknown",
			"message":        "no health data for this client",
		})
		return
	}

	// Build detailed checks
	checks := []map[string]any{
		{"name": "active_tokens", "status": "pass", "value": health.ActiveTokens},
		{"name": "error_rate", "status": health.checkStatus(health.ErrorRate, 1.0, 5.0), "value": health.ErrorRate},
		{"name": "cert_validity", "status": health.certCheck(), "value": health.CertStatus},
		{"name": "secret_validity", "status": health.secretCheck(), "value": health.SecretStatus},
		{"name": "recent_activity", "status": "pass", "value": health.LastUsed},
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"client_id":       health.ClientID,
		"active_tokens":   health.ActiveTokens,
		"recent_errors":   health.RecentErrors,
		"error_rate_pct":  health.ErrorRate,
		"cert_status":     health.CertStatus,
		"secret_status":   health.SecretStatus,
		"last_used":       health.LastUsed,
		"overall_health":  health.OverallHealth,
		"checks":          checks,
		"checked_at":      time.Now().UTC().Format(time.RFC3339),
	})
}

func (h *clientHealth) checkStatus(value, warn, crit float64) string {
	if value >= crit {
		return "fail"
	}
	if value >= warn {
		return "warn"
	}
	return "pass"
}

func (h *clientHealth) certCheck() string {
	if h.CertStatus == "valid" {
		return "pass"
	}
	return "fail"
}

func (h *clientHealth) secretCheck() string {
	switch h.SecretStatus {
	case "valid":
		return "pass"
	case "expiring_soon":
		return "warn"
	default:
		return "fail"
	}
}
