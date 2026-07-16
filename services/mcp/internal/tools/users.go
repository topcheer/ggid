package tools

import (
	"context"
	"fmt"

	"github.com/ggid/ggid/services/mcp/internal/client"
)

var userTools = []Tool{
	{
		Name:        "list_users",
		Description: "List users in GGID with optional pagination",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"page":  map[string]any{"type": "integer", "description": "Page number (default 1)"},
				"limit": map[string]any{"type": "integer", "description": "Results per page (default 50)"},
			},
		},
		RequiredScopes: []string{"users:read"},
		Handler: func(ctx context.Context, c *client.Client, args map[string]any) (any, error) {
			page := argInt(args, "page", 1)
			limit := argInt(args, "limit", 50)
			path := fmt.Sprintf("/api/v1/users?page=%d&limit=%d", page, limit)
			var result any
			if err := c.Get(ctx, path, &result); err != nil {
				return nil, err
			}
			return result, nil
		},
	},
	{
		Name:        "get_user",
		Description: "Get detailed information about a specific user",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"user_id": map[string]any{"type": "string", "description": "User UUID"},
			},
			"required": []string{"user_id"},
		},
		RequiredScopes: []string{"users:read"},
		Handler: func(ctx context.Context, c *client.Client, args map[string]any) (any, error) {
			var result any
			if err := c.Get(ctx, fmt.Sprintf("/api/v1/users/%s", argStr(args, "user_id")), &result); err != nil {
				return nil, err
			}
			return result, nil
		},
	},
	{
		Name:        "create_user",
		Description: "Create a new user in GGID",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"username":     map[string]any{"type": "string"},
				"email":        map[string]any{"type": "string"},
				"display_name": map[string]any{"type": "string"},
			},
			"required": []string{"username", "email"},
		},
		RequiredScopes: []string{"users:write"},
		Handler: func(ctx context.Context, c *client.Client, args map[string]any) (any, error) {
			body := map[string]any{
				"username":     argStr(args, "username"),
				"email":        argStr(args, "email"),
				"display_name": argStr(args, "display_name"),
			}
			var result any
			if err := c.Post(ctx, "/api/v1/users", body, &result); err != nil {
				return nil, err
			}
			return result, nil
		},
	},
	{
		Name:        "lock_user",
		Description: "Lock a user account, preventing login",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"user_id": map[string]any{"type": "string"},
			},
			"required": []string{"user_id"},
		},
		RequiredScopes: []string{"users:write"},
		Handler: func(ctx context.Context, c *client.Client, args map[string]any) (any, error) {
			var result any
			if err := c.Post(ctx, fmt.Sprintf("/api/v1/users/%s/lock", argStr(args, "user_id")), nil, &result); err != nil {
				return nil, err
			}
			return result, nil
		},
	},
	{
		Name:        "unlock_user",
		Description: "Unlock a previously locked user account",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"user_id": map[string]any{"type": "string"},
			},
			"required": []string{"user_id"},
		},
		RequiredScopes: []string{"users:write"},
		Handler: func(ctx context.Context, c *client.Client, args map[string]any) (any, error) {
			var result any
			if err := c.Post(ctx, fmt.Sprintf("/api/v1/users/%s/unlock", argStr(args, "user_id")), nil, &result); err != nil {
				return nil, err
			}
			return result, nil
		},
	},
}
