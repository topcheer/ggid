package tools

import (
	"context"
	"fmt"

	"github.com/ggid/ggid/services/mcp/internal/client"
)

// Additional admin tools to cover tenant administrator functionality.

var adminTools = []Tool{
	{
		Name:        "delete_user",
		Description: "Delete a user account (soft delete — sets status to deleted)",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"user_id": map[string]any{"type": "string", "description": "User UUID"},
			},
			"required": []string{"user_id"},
		},
		RequiredScopes: []string{"users:delete"},
		Handler: func(ctx context.Context, c *client.Client, args map[string]any) (any, error) {
			var result any
			if err := c.Delete(ctx, fmt.Sprintf("/api/v1/users/%s", argStr(args, "user_id")), &result); err != nil {
				return nil, err
			}
			return result, nil
		},
	},
	{
		Name:        "update_user",
		Description: "Update user attributes (display_name, email, phone, status, locale, timezone)",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"user_id":      map[string]any{"type": "string", "description": "User UUID"},
				"display_name": map[string]any{"type": "string"},
				"email":        map[string]any{"type": "string"},
				"phone":        map[string]any{"type": "string"},
				"status":       map[string]any{"type": "string", "enum": []string{"active", "locked", "suspended"}},
				"locale":       map[string]any{"type": "string"},
				"timezone":     map[string]any{"type": "string"},
			},
			"required": []string{"user_id"},
		},
		RequiredScopes: []string{"users:write"},
		Handler: func(ctx context.Context, c *client.Client, args map[string]any) (any, error) {
			body := map[string]any{}
			for _, k := range []string{"display_name", "email", "phone", "status", "locale", "timezone"} {
				if v, ok := args[k]; ok {
					body[k] = v
				}
			}
			var result any
			if err := c.Patch(ctx, fmt.Sprintf("/api/v1/users/%s", argStr(args, "user_id")), body, &result); err != nil {
				return nil, err
			}
			return result, nil
		},
	},
	{
		Name:        "create_role",
		Description: "Create a new role in the system",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name":        map[string]any{"type": "string", "description": "Role display name"},
				"key":         map[string]any{"type": "string", "description": "Unique role key (e.g. 'erp_manager')"},
				"description": map[string]any{"type": "string"},
			},
			"required": []string{"name", "key"},
		},
		RequiredScopes: []string{"roles:write"},
		Handler: func(ctx context.Context, c *client.Client, args map[string]any) (any, error) {
			body := map[string]any{
				"name":        argStr(args, "name"),
				"key":         argStr(args, "key"),
				"description": argStr(args, "description"),
			}
			var result any
			if err := c.Post(ctx, "/api/v1/roles", body, &result); err != nil {
				return nil, err
			}
			return result, nil
		},
	},
	{
		Name:        "delete_role",
		Description: "Delete a role by its ID",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"role_id": map[string]any{"type": "string", "description": "Role UUID"},
			},
			"required": []string{"role_id"},
		},
		RequiredScopes: []string{"roles:write"},
		Handler: func(ctx context.Context, c *client.Client, args map[string]any) (any, error) {
			var result any
			if err := c.Delete(ctx, fmt.Sprintf("/api/v1/roles/%s", argStr(args, "role_id")), &result); err != nil {
				return nil, err
			}
			return result, nil
		},
	},
	{
		Name:        "remove_user_role",
		Description: "Remove a role assignment from a user",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"user_id": map[string]any{"type": "string", "description": "User UUID"},
				"role_id": map[string]any{"type": "string", "description": "Role UUID"},
			},
			"required": []string{"user_id", "role_id"},
		},
		RequiredScopes: []string{"roles:manage"},
		Handler: func(ctx context.Context, c *client.Client, args map[string]any) (any, error) {
			var result any
			if err := c.Delete(ctx, fmt.Sprintf("/api/v1/users/%s/roles/%s", argStr(args, "user_id"), argStr(args, "role_id")), &result); err != nil {
				return nil, err
			}
			return result, nil
		},
	},
	{
		Name:        "delete_policy",
		Description: "Delete an access control policy by ID",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"policy_id": map[string]any{"type": "string", "description": "Policy UUID"},
			},
			"required": []string{"policy_id"},
		},
		RequiredScopes: []string{"policies:write"},
		Handler: func(ctx context.Context, c *client.Client, args map[string]any) (any, error) {
			var result any
			if err := c.Delete(ctx, fmt.Sprintf("/api/v1/policies/%s", argStr(args, "policy_id")), &result); err != nil {
				return nil, err
			}
			return result, nil
		},
	},
	{
		Name:        "list_organizations",
		Description: "List all organizations in the tenant",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"limit": map[string]any{"type": "integer", "description": "Results per page (default 50)"},
				"page":  map[string]any{"type": "integer", "description": "Page number (default 1)"},
			},
		},
		RequiredScopes: []string{"orgs:read"},
		Handler: func(ctx context.Context, c *client.Client, args map[string]any) (any, error) {
			page := argInt(args, "page", 1)
			limit := argInt(args, "limit", 50)
			var result any
			if err := c.Get(ctx, fmt.Sprintf("/api/v1/orgs?page=%d&limit=%d", page, limit), &result); err != nil {
				return nil, err
			}
			return result, nil
		},
	},
	{
		Name:        "create_organization",
		Description: "Create a new organization within the tenant",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name":      map[string]any{"type": "string"},
				"parent_id": map[string]any{"type": "string", "description": "Parent org UUID (optional for root)"},
			},
			"required": []string{"name"},
		},
		RequiredScopes: []string{"orgs:write"},
		Handler: func(ctx context.Context, c *client.Client, args map[string]any) (any, error) {
			body := map[string]any{"name": argStr(args, "name")}
			if pid := argStr(args, "parent_id"); pid != "" {
				body["parent_id"] = pid
			}
			var result any
			if err := c.Post(ctx, "/api/v1/orgs", body, &result); err != nil {
				return nil, err
			}
			return result, nil
		},
	},
	{
		Name:        "list_oauth_clients",
		Description: "List all OAuth/OIDC clients registered in the tenant",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"limit": map[string]any{"type": "integer", "description": "Results per page (default 50)"},
				"page":  map[string]any{"type": "integer", "description": "Page number (default 1)"},
			},
		},
		RequiredScopes: []string{"oauth:read"},
		Handler: func(ctx context.Context, c *client.Client, args map[string]any) (any, error) {
			page := argInt(args, "page", 1)
			limit := argInt(args, "limit", 50)
			var result any
			if err := c.Get(ctx, fmt.Sprintf("/api/v1/oauth/clients?page=%d&limit=%d", page, limit), &result); err != nil {
				return nil, err
			}
			return result, nil
		},
	},
	{
		Name:        "list_sessions",
		Description: "List active user sessions in the tenant",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"limit": map[string]any{"type": "integer", "description": "Results per page (default 50)"},
			},
		},
		RequiredScopes: []string{"users:read"},
		Handler: func(ctx context.Context, c *client.Client, args map[string]any) (any, error) {
			limit := argInt(args, "limit", 50)
			var result any
			if err := c.Get(ctx, fmt.Sprintf("/api/v1/auth/sessions?limit=%d", limit), &result); err != nil {
				return nil, err
			}
			return result, nil
		},
	},
	{
		Name:        "revoke_session",
		Description: "Revoke a user session by session ID or user ID",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"session_id": map[string]any{"type": "string", "description": "Session ID to revoke"},
				"user_id":    map[string]any{"type": "string", "description": "Revoke all sessions for this user"},
			},
		},
		RequiredScopes: []string{"users:write"},
		Handler: func(ctx context.Context, c *client.Client, args map[string]any) (any, error) {
			body := map[string]any{}
			if sid := argStr(args, "session_id"); sid != "" {
				body["session_id"] = sid
			}
			if uid := argStr(args, "user_id"); uid != "" {
				body["user_id"] = uid
			}
			var result any
			if err := c.Post(ctx, "/api/v1/auth/sessions/revoke", body, &result); err != nil {
				return nil, err
			}
			return result, nil
		},
	},
	{
		Name:        "get_tenant_info",
		Description: "Get current tenant information (name, slug, status, branding)",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{},
		},
		RequiredScopes: []string{"tenants:read"},
		Handler: func(ctx context.Context, c *client.Client, args map[string]any) (any, error) {
			var result any
			if err := c.Get(ctx, "/api/v1/tenants", &result); err != nil {
				return nil, err
			}
			return result, nil
		},
	},
}