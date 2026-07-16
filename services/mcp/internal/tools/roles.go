package tools

import (
	"context"
	"fmt"

	"github.com/ggid/ggid/services/mcp/internal/client"
)

var roleTools = []Tool{
	{
		Name:        "list_roles",
		Description: "List all roles in the system",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{},
		},
		RequiredScopes: []string{"roles:read"},
		Handler: func(ctx context.Context, c *client.Client, args map[string]any) (any, error) {
			var result any
			if err := c.Get(ctx, "/api/v1/roles", &result); err != nil {
				return nil, err
			}
			return result, nil
		},
	},
	{
		Name:        "assign_role",
		Description: "Assign a role to a user",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"user_id":   map[string]any{"type": "string"},
				"role_key":  map[string]any{"type": "string"},
			},
			"required": []string{"user_id", "role_key"},
		},
		RequiredScopes: []string{"roles:manage"},
		Handler: func(ctx context.Context, c *client.Client, args map[string]any) (any, error) {
			body := map[string]any{"role_key": argStr(args, "role_key")}
			var result any
			if err := c.Post(ctx, fmt.Sprintf("/api/v1/users/%s/roles", argStr(args, "user_id")), body, &result); err != nil {
				return nil, err
			}
			return result, nil
		},
	},
	{
		Name:        "list_user_roles",
		Description: "List roles assigned to a specific user",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"user_id": map[string]any{"type": "string"},
			},
			"required": []string{"user_id"},
		},
		RequiredScopes: []string{"roles:read"},
		Handler: func(ctx context.Context, c *client.Client, args map[string]any) (any, error) {
			var result any
			if err := c.Get(ctx, fmt.Sprintf("/api/v1/users/%s/roles", argStr(args, "user_id")), &result); err != nil {
				return nil, err
			}
			return result, nil
		},
	},
}
