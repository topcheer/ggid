// Package httpserver OpenAPI annotations for the audit service.
// These comments are consumed by swaggo/swag to generate OpenAPI documentation.
// To regenerate: swag init -g services/audit/internal/server/http.go
package httpserver

// --- Audit: Events ---

// GetEvents godoc
// @Summary List audit events
// @Description Retrieve paginated audit events with filtering by tenant, action, result, actor, and time range.
// @Tags audit
// @Produce json
// @Param tenant_id query string true "Tenant ID"
// @Param action query string false "Filter by action (e.g. user.login)"
// @Param result query string false "Filter by result (success/failure)"
// @Param actor_id query string false "Filter by actor UUID"
// @Param start query string false "Start time (RFC3339)"
// @Param end query string false "End time (RFC3339)"
// @Param page_size query int false "Page size (default 50)"
// @Success 200 {object} map[string]any "Paginated audit events"
// @Router /api/v1/audit/events [get]

// GetEventByID godoc
// @Summary Get audit event by ID
// @Description Retrieve a single audit event with full detail and hash chain verification.
// @Tags audit
// @Produce json
// @Param id path string true "Event ID"
// @Success 200 {object} map[string]any "Audit event detail"
// @Failure 404 {object} map[string]string "Event not found"
// @Router /api/v1/audit/events/{id} [get]

// --- Audit: Stats & Export ---

// GetStats godoc
// @Summary Audit statistics
// @Description Aggregate statistics for audit events including action counts, result breakdowns, and time series.
// @Tags audit
// @Produce json
// @Param tenant_id query string true "Tenant ID"
// @Param window query string false "Time window (1h/24h/7d/30d, default 24h)"
// @Success 200 {object} map[string]any "Aggregated stats"
// @Router /api/v1/audit/stats [get]

// ExportEvents godoc
// @Summary Export audit events
// @Description Export audit events as JSON or CSV. Supports streaming for large exports.
// @Tags audit
// @Produce json
// @Param tenant_id query string true "Tenant ID"
// @Param format query string false "Export format (json/csv, default json)"
// @Param start query string false "Start time (RFC3339)"
// @Param end query string false "End time (RFC3339)"
// @Success 200 {file} file "Exported audit data"
// @Router /api/v1/audit/export [get]

// --- Audit: Search & Integrity ---

// SearchEvents godoc
// @Summary Search audit events
// @Description Full-text search across audit event fields including action, detail, and metadata.
// @Tags audit
// @Accept json
// @Produce json
// @Param request body object true "Search request {query, filters, page_size}"
// @Success 200 {object} map[string]any "Search results"
// @Router /api/v1/audit/search [post]

// VerifyIntegrity godoc
// @Summary Verify audit chain integrity
// @Description Verify the hash chain integrity of all audit events. Detects tampering or gaps.
// @Tags audit
// @Produce json
// @Param tenant_id query string true "Tenant ID"
// @Success 200 {object} map[string]any "{valid: bool, broken_links: []}"
// @Router /api/v1/audit/verify-integrity [get]

// --- Audit: Compliance ---

// ComplianceReport godoc
// @Summary Generate compliance report
// @Description Generate a compliance evidence report for SOX/HIPAA/DORA including audit coverage and control status.
// @Tags compliance
// @Produce json
// @Param tenant_id query string true "Tenant ID"
// @Param framework query string false "Framework (sox/hipaa/dora, default all)"
// @Success 200 {object} map[string]any "Compliance report"
// @Router /api/v1/audit/compliance-report [get]

// GetCCMResults godoc
// @Summary Get CCM results
// @Description Retrieve the latest Continuous Compliance Monitoring results for all controls.
// @Tags ccm
// @Produce json
// @Success 200 {array} map[string]any "CCM control results"
// @Router /api/v1/audit/ccm/results [get]

// RunCCM godoc
// @Summary Trigger CCM scan
// @Description Manually trigger a compliance control evaluation sweep. Returns results and summary.
// @Tags ccm
// @Accept json
// @Produce json
// @Success 200 {object} map[string]any "{results: [], controls_run: int, summary: {}}"
// @Router /api/v1/audit/ccm/run [post]

// GetCCMHistory godoc
// @Summary Get CCM history
// @Description Retrieve historical CCM results for trend analysis, optionally filtered by control_id.
// @Tags ccm
// @Produce json
// @Param control_id query string false "Filter by control ID"
// @Success 200 {array} map[string]any "Historical CCM results"
// @Router /api/v1/audit/ccm/history [get]
