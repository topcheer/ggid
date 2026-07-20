package httpserver

import (
	"fmt"
	"net/http"
	"time"
)

// handleGlobalAuditDashboard provides a cross-tenant audit view for super admins.
// GET /api/v1/admin/audit/global
// Query params: limit, offset, action, severity
func (s *HTTPServer) handleGlobalAuditDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := parseIntSafe(l); err == nil && v > 0 && v <= 500 {
			limit = v
		}
	}
	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		if v, err := parseIntSafe(o); err == nil && v >= 0 {
			offset = v
		}
	}

	actionFilter := r.URL.Query().Get("action")
	severityFilter := r.URL.Query().Get("severity")

	// Query audit events across all tenants from DB
	events := make([]map[string]any, 0)
	if s.pool != nil {
		query := `SELECT id, tenant_id, actor_type, actor_id, action, result, resource_type, resource_id,
			detail, created_at
			FROM audit_events WHERE 1=1`
		args := []any{}
		argIdx := 1
		if actionFilter != "" {
			query += ` AND action = $` + intToStr(argIdx)
			args = append(args, actionFilter)
			argIdx++
		}
		if severityFilter != "" {
			query += ` AND (detail->>'severity') = $` + intToStr(argIdx)
			args = append(args, severityFilter)
			argIdx++
		}
		query += ` ORDER BY created_at DESC LIMIT $` + intToStr(argIdx)
		args = append(args, limit)
		argIdx++
		query += ` OFFSET $` + intToStr(argIdx)
		args = append(args, offset)

		rows, err := s.pool.Query(r.Context(), query, args...)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var (
					id, tenantID, actorType, actorID, action, result, resourceType, resourceID, detail string
					createdAt                                                                 time.Time
				)
				if err := rows.Scan(&id, &tenantID, &actorType, &actorID, &action, &result, &resourceType, &resourceID, &detail, &createdAt); err != nil {
					continue
				}
				events = append(events, map[string]any{
					"id":            id,
					"tenant_id":     tenantID,
					"actor_type":    actorType,
					"actor_id":      actorID,
					"action":        action,
					"result":        result,
					"resource_type": resourceType,
					"resource_id":   resourceID,
					"detail":        detail,
					"created_at":    createdAt,
				})
			}
		}
	}

	// Summary statistics
	summary := map[string]any{
		"total_events":  len(events),
		"limit":         limit,
		"offset":        offset,
		"actions":       countByField(events, "action"),
		"results":       countByField(events, "result"),
		"tenants":       countByField(events, "tenant_id"),
		"resource_types": countByField(events, "resource_type"),
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"events":  events,
		"count":   len(events),
		"summary": summary,
	})
}

// handleGlobalThreatDashboard aggregates threat intelligence across tenants.
// GET /api/v1/admin/threats/dashboard
func (s *HTTPServer) handleGlobalThreatDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	now := time.Now().UTC()
	_24hAgo := now.Add(-24 * time.Hour)

	// Aggregate from ITDR detections
	itdrStats := map[string]any{
		"total_detections_24h": 0,
		"by_severity":          map[string]int{"critical": 0, "high": 0, "medium": 0, "low": 0},
		"by_status":            map[string]int{"open": 0, "investigating": 0, "resolved": 0},
	}

	if s.itdrRepo != nil {
		// Best-effort query — itdrRepo may have methods to count detections
		// We use the pool directly for a cross-tenant aggregation
		if s.pool != nil {
			// Count ITDR detections in the last 24h
			rows, err := s.pool.Query(r.Context(),
				`SELECT severity, status, COUNT(*) as cnt
				 FROM itdr_detections
				 WHERE detected_at >= $1
				 GROUP BY severity, status`,
				_24hAgo)
			if err == nil {
				defer rows.Close()
				for rows.Next() {
					var severity, status string
					var cnt int
					if err := rows.Scan(&severity, &status, &cnt); err != nil {
						continue
					}
					itdrStats["total_detections_24h"] = itdrStats["total_detections_24h"].(int) + cnt
					if sevMap, ok := itdrStats["by_severity"].(map[string]int); ok {
						sevMap[severity] += cnt
					}
					if statusMap, ok := itdrStats["by_status"].(map[string]int); ok {
						statusMap[status] += cnt
					}
				}
			}
		}
	}

	// Threat intelligence indicators count
	threatIndicators := map[string]any{
		"total_indicators": 0,
		"active_feeds":     0,
	}
	if s.pool != nil {
		var totalIocs int
		err := s.pool.QueryRow(r.Context(),
			`SELECT COUNT(*) FROM threat_intel_indicators WHERE expires_at IS NULL OR expires_at > $1`,
			now).Scan(&totalIocs)
		if err == nil {
			threatIndicators["total_indicators"] = totalIocs
		}
		var activeFeeds int
		err = s.pool.QueryRow(r.Context(),
			`SELECT COUNT(*) FROM threat_intel_sources WHERE enabled = true`).Scan(&activeFeeds)
		if err == nil {
			threatIndicators["active_feeds"] = activeFeeds
		}
	}

	// Active incidents count
	activeIncidents := 0
	if s.pool != nil {
		_ = s.pool.QueryRow(r.Context(),
			`SELECT COUNT(*) FROM audit_incidents WHERE status IN ('open', 'investigating')`).Scan(&activeIncidents)
	}

	// Per-tenant threat summary
	tenantThreats := make([]map[string]any, 0)
	if s.pool != nil {
		rows, err := s.pool.Query(r.Context(),
			`SELECT tenant_id, COUNT(*) FILTER (WHERE severity IN ('critical','high')) as critical_high,
			        COUNT(*) as total
			 FROM itdr_detections
			 WHERE detected_at >= $1
			 GROUP BY tenant_id
			 ORDER BY critical_high DESC
			 LIMIT 20`,
			_24hAgo)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var tenantID string
				var criticalHigh, total int
				if err := rows.Scan(&tenantID, &criticalHigh, &total); err != nil {
					continue
				}
				tenantThreats = append(tenantThreats, map[string]any{
					"tenant_id":    tenantID,
					"critical_high": criticalHigh,
					"total":         total,
				})
			}
		}
	}

	// Overall threat level
	threatLevel := "low"
	if critCount := itdrStats["by_severity"].(map[string]int)["critical"]; critCount > 10 {
		threatLevel = "critical"
	} else if critCount > 5 {
		threatLevel = "high"
	} else if highCount := itdrStats["by_severity"].(map[string]int)["high"]; highCount > 5 {
		threatLevel = "medium"
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"threat_level":       threatLevel,
		"itdr_stats":         itdrStats,
		"threat_intel":       threatIndicators,
		"active_incidents":   activeIncidents,
		"tenant_threats":     tenantThreats,
		"tenant_count":       len(tenantThreats),
		"window":             "24h",
		"generated_at":       now,
	})
}

// --- helpers ---

func parseIntSafe(s string) (int, error) {
	var v int
	_, err := fmt.Sscanf(s, "%d", &v)
	return v, err
}

func intToStr(i int) string {
	return fmt.Sprintf("%d", i)
}

func countByField(events []map[string]any, field string) map[string]int {
	result := make(map[string]int)
	for _, e := range events {
		if val, ok := e[field].(string); ok && val != "" {
			result[val]++
		}
	}
	return result
}
