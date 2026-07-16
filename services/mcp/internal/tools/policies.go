package tools

import (
	"context"

	"github.com/ggid/ggid/services/mcp/internal/client"
)

var policyTools = []Tool{
	{
		Name:        "list_policies",
		Description: "List all access control policies",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{},
		},
		RequiredScopes: []string{"policies:read"},
		Handler: func(ctx context.Context, c *client.Client, args map[string]any) (any, error) {
			var result any
			if err := c.Get(ctx, "/api/v1/policies", &result); err != nil {
				return nil, err
			}
			return result, nil
		},
	},
	{
		Name:        "create_policy",
		Description: "Create a new access control policy",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name":        map[string]any{"type": "string"},
				"description": map[string]any{"type": "string"},
				"effect":      map[string]any{"type": "string", "enum": []string{"allow", "deny"}},
				"actions":     map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				"resources":   map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			},
			"required": []string{"name", "effect"},
		},
		RequiredScopes: []string{"policies:write"},
		Handler: func(ctx context.Context, c *client.Client, args map[string]any) (any, error) {
			body := map[string]any{
				"name":        argStr(args, "name"),
				"description": argStr(args, "description"),
				"effect":      argStr(args, "effect"),
			}
			var result any
			if err := c.Post(ctx, "/api/v1/policies", body, &result); err != nil {
				return nil, err
			}
			return result, nil
		},
	},
	{
		Name:        "check_permission",
		Description: "Check if a subject has permission to perform an action on a resource",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"subject":  map[string]any{"type": "string"},
				"action":   map[string]any{"type": "string"},
				"resource": map[string]any{"type": "string"},
			},
			"required": []string{"subject", "action", "resource"},
		},
		RequiredScopes: []string{"policies:read"},
		Handler: func(ctx context.Context, c *client.Client, args map[string]any) (any, error) {
			body := map[string]any{
				"subject":  argStr(args, "subject"),
				"action":   argStr(args, "action"),
				"resource": argStr(args, "resource"),
			}
			var result any
			if err := c.Post(ctx, "/api/v1/policies/check", body, &result); err != nil {
				return nil, err
			}
			return result, nil
		},
	},
}
