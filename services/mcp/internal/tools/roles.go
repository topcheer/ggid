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
		RequiredScopes: []string{"roles:write"},
		Handler: func(ctx context.Context, c *client.Client, args map[string]any) (any, error) {
			// Try role_id first, then role_key
			roleKey := argStr(args, "role_key")
			body := map[string]any{"role_id": roleKey}
			var result any
			if err := c.Post(ctx, fmt.Sprintf("/api/v1/users/%s/roles", argStr(args, "user_id")), body, &result); err != nil {
				// Fallback: lookup role by key, then use role_id
				var rolesResp any
				if err2 := c.Get(ctx, "/api/v1/roles", &rolesResp); err2 == nil {
					if roleID := findRoleIDByKey(rolesResp, roleKey); roleID != "" {
						body2 := map[string]any{"role_id": roleID}
						if err3 := c.Post(ctx, fmt.Sprintf("/api/v1/users/%s/roles", argStr(args, "user_id")), body2, &result); err3 != nil {
							return nil, err
						}
						return result, nil
					}
				}
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
