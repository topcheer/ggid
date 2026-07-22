package tools

import (
	"context"
	"fmt"

	"github.com/ggid/ggid/services/mcp/internal/client"
)

var auditTools = []Tool{
	{
		Name:        "list_audit_events",
		Description: "Query audit events with optional filters",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"limit":    map[string]any{"type": "integer", "description": "Max results (default 50)"},
				"action":   map[string]any{"type": "string", "description": "Filter by action (e.g. user.login)"},
			},
		},
		RequiredScopes: []string{"audit:read"},
		Handler: func(ctx context.Context, c *client.Client, args map[string]any) (any, error) {
			limit := argInt(args, "limit", 50)
			// tenant_id is resolved by the gateway from X-Tenant-ID header;
			// no need to pass it as a query parameter.
			path := fmt.Sprintf("/api/v1/audit/events?page_size=%d", limit)
			if action := argStr(args, "action"); action != "" {
				path += "&action=" + action
			}
			var result any
			if err := c.Get(ctx, path, &result); err != nil {
				return nil, err
			}
			return result, nil
		},
	},
	{
		Name:        "get_dashboard_stats",
		Description: "Get platform dashboard statistics (total users, sessions, etc.)",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{},
		},
		RequiredScopes: []string{"audit:read"},
		Handler: func(ctx context.Context, c *client.Client, args map[string]any) (any, error) {
			var result any
			if err := c.Get(ctx, "/api/v1/identity/dashboard/stats", &result); err != nil {
				return nil, err
			}
			return result, nil
		},
	},
}
