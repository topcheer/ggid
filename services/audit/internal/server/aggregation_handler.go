package httpserver

import (
	"fmt"
	"net/http"
	"time"

	"github.com/ggid/ggid/services/audit/internal/domain"
	"github.com/google/uuid"
)

// GET /api/v1/audit/aggregations?metric=count&group_by=user&action=login&interval=1h&tenant_id=X
func (s *HTTPServer) handleAggregations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
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

	metric := r.URL.Query().Get("metric")
	if metric == "" {
		metric = "count"
	}
	groupBy := r.URL.Query().Get("group_by")
	if groupBy == "" {
		groupBy = "action"
	}
	action := r.URL.Query().Get("action")
	intervalStr := r.URL.Query().Get("interval")
	if intervalStr == "" {
		intervalStr = "1h"
	}
	interval, err := time.ParseDuration(intervalStr)
	if err != nil || interval <= 0 {
		interval = time.Hour
	}

	to := time.Now().UTC()
	from := to.Add(-24 * time.Hour)

	filter := domain.ListFilter{
		TenantID:   tenantID,
		Action:     action,
		StartTime:  &from,
		EndTime:    &to,
		Descending: false,
	}

	events, _, err := s.svc.ListEvents(r.Context(), filter, 1, 10000)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	bucketCount := int(to.Sub(from) / interval)
	if bucketCount > 1000 {
		bucketCount = 1000
	}
	buckets := make([]map[string]any, bucketCount)
	for i := 0; i < bucketCount; i++ {
		bucketStart := from.Add(time.Duration(i) * interval)
		buckets[i] = map[string]any{
			"timestamp": bucketStart.Format(time.RFC3339),
			"count":     0,
		}
	}

	groupData := make(map[string]map[int]int)
	for _, e := range events {
		idx := int(e.CreatedAt.Sub(from) / interval)
		if idx < 0 || idx >= bucketCount {
			continue
		}

		c := buckets[idx]["count"].(int)
		buckets[idx]["count"] = c + 1

		groupVal := ""
		switch groupBy {
		case "user", "actor":
			if e.ActorID != nil {
				groupVal = e.ActorID.String()
			}
		case "action":
			groupVal = e.Action
		case "resource_type":
			groupVal = e.ResourceType
		case "result":
			groupVal = string(e.Result)
		}
		if groupVal != "" {
			if groupData[groupVal] == nil {
				groupData[groupVal] = make(map[int]int)
			}
			groupData[groupVal][idx]++
		}
	}

	series := make([]map[string]any, 0, len(groupData))
	for group, counts := range groupData {
		points := make([]int, bucketCount)
		for i := 0; i < bucketCount; i++ {
			points[i] = counts[i]
		}
		series = append(series, map[string]any{
			"group":  group,
			"counts": points,
		})
	}

	totalCount := 0
	for _, b := range buckets {
		totalCount += b["count"].(int)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"metric":       metric,
		"group_by":     groupBy,
		"action":       fmt.Sprintf("%v", action),
		"interval":     intervalStr,
		"from":         from.Format(time.RFC3339),
		"to":           to.Format(time.RFC3339),
		"total":        totalCount,
		"buckets":      buckets,
		"series":       series,
		"bucket_count": bucketCount,
	})
}
