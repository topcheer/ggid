package httpserver

import (
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/ggid/ggid/services/audit/internal/domain"
	"github.com/google/uuid"
)

var (
	emailRegex  = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	phoneRegex  = regexp.MustCompile(`\b\d{3}[-.]?\d{3}[-.]?\d{4}\b`)
	ssnRegex    = regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)
)

type piiFinding struct {
	EventID     string `json:"event_id"`
	Field       string `json:"field"`
	PIIType     string `json:"pii_type"`
	MaskedValue string `json:"masked_value"`
}

// POST /api/v1/audit/pii-scan?tenant_id=X
func (s *HTTPServer) handlePIIScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	tenantIDStr := r.URL.Query().Get("tenant_id")
	if tenantIDStr == "" {
		writeJSONError(w, http.StatusBadRequest, "tenant_id is required")
		return
	}
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid tenant_id")
		return
	}

	from := time.Now().UTC().Add(-24 * time.Hour)
	to := time.Now().UTC()
	filter := domain.ListFilter{TenantID: tenantID, StartTime: &from, EndTime: &to}
	events, _, err := s.svc.ListEvents(r.Context(), filter, 1, 1000)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	findings := []piiFinding{}
	piiCount := map[string]int{"email": 0, "phone": 0, "ssn": 0}

	for _, e := range events {
		scanPII(&findings, &piiCount, e.ID.String(), "actor_name", e.ActorName)
		scanPII(&findings, &piiCount, e.ID.String(), "resource_name", e.ResourceName)
		if e.Metadata != nil {
			for k, v := range e.Metadata {
				if s, ok := v.(string); ok {
					scanPII(&findings, &piiCount, e.ID.String(), "metadata."+k, s)
				}
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"tenant_id":      tenantIDStr,
		"events_scanned": len(events),
		"total_findings": len(findings),
		"by_type":        piiCount,
		"findings":       findings,
		"scanned_at":     time.Now().UTC().Format(time.RFC3339),
	})
}

func scanPII(findings *[]piiFinding, count *map[string]int, eventID, field, value string) {
	if value == "" {
		return
	}
	if m := emailRegex.FindString(value); m != "" {
		*findings = append(*findings, piiFinding{eventID, field, "email", maskPII(m)})
		(*count)["email"]++
	}
	if m := phoneRegex.FindString(value); m != "" {
		*findings = append(*findings, piiFinding{eventID, field, "phone", maskPII(m)})
		(*count)["phone"]++
	}
	if m := ssnRegex.FindString(value); m != "" {
		*findings = append(*findings, piiFinding{eventID, field, "ssn", maskPII(m)})
		(*count)["ssn"]++
	}
}

func maskPII(s string) string {
	if len(s) <= 4 {
		return strings.Repeat("*", len(s))
	}
	return s[:2] + strings.Repeat("*", len(s)-4) + s[len(s)-2:]
}
