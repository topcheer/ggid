package tools

import (
	"context"
	"fmt"

	"github.com/ggid/ggid/services/mcp/internal/client"
)

// Extended admin tools covering remaining Console functionality.

var extendedAdminTools = []Tool{
	// === OAuth 客户端管理 ===
	{
		Name:        "create_oauth_client",
		Description: "Register a new OAuth/OIDC client application (DCR-style)",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name":            map[string]any{"type": "string", "description": "Client display name"},
				"redirect_uris":  map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				"grant_types":     map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "e.g. [\"authorization_code\",\"refresh_token\"]"},
				"scopes":          map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "e.g. [\"openid\",\"profile\",\"email\"]"},
				"token_endpoint_auth_method": map[string]any{"type": "string", "description": "none, client_secret_basic, client_secret_post"},
			},
			"required": []string{"name"},
		},
		RequiredScopes: []string{"oauth:write"},
		Handler: func(ctx context.Context, c *client.Client, args map[string]any) (any, error) {
			body := map[string]any{
				"name": argStr(args, "name"),
			}
			if v := argStrSlice(args, "redirect_uris"); len(v) > 0 {
				body["redirect_uris"] = v
			}
			if v := argStrSlice(args, "grant_types"); len(v) > 0 {
				body["grant_types"] = v
			}
			if v := argStrSlice(args, "scopes"); len(v) > 0 {
				body["scopes"] = v
			}
			if v := argStr(args, "token_endpoint_auth_method"); v != "" {
				body["token_endpoint_auth_method"] = v
			}
			var result any
			if err := c.Post(ctx, "/api/v1/oauth/clients", body, &result); err != nil {
				return nil, err
			}
			return result, nil
		},
	},
	{
		Name:        "delete_oauth_client",
		Description: "Delete an OAuth client by client_id",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"client_id": map[string]any{"type": "string", "description": "OAuth client_id"},
			},
			"required": []string{"client_id"},
		},
		RequiredScopes: []string{"oauth:write"},
		Handler: func(ctx context.Context, c *client.Client, args map[string]any) (any, error) {
			var result any
			if err := c.Delete(ctx, fmt.Sprintf("/api/v1/oauth/clients/%s", argStr(args, "client_id")), &result); err != nil {
				return nil, err
			}
			return result, nil
		},
	},

	// === 权限管理 ===
	{
		Name:        "list_permissions",
		Description: "List all available permissions in the system",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{},
		},
		RequiredScopes: []string{"policies:read"},
		Handler: func(ctx context.Context, c *client.Client, args map[string]any) (any, error) {
			var result any
			if err := c.Get(ctx, "/api/v1/permissions", &result); err != nil {
				return nil, err
			}
			return result, nil
		},
	},
	{
		Name:        "create_permission",
		Description: "Create a new custom permission/scope",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"key":         map[string]any{"type": "string", "description": "Permission key (e.g. 'reports:read')"},
				"name":        map[string]any{"type": "string", "description": "Display name"},
				"description": map[string]any{"type": "string"},
			},
			"required": []string{"key", "name"},
		},
		RequiredScopes: []string{"policies:write"},
		Handler: func(ctx context.Context, c *client.Client, args map[string]any) (any, error) {
			body := map[string]any{
				"key":         argStr(args, "key"),
				"name":        argStr(args, "name"),
				"description": argStr(args, "description"),
			}
			var result any
			if err := c.Post(ctx, "/api/v1/permissions", body, &result); err != nil {
				return nil, err
			}
			return result, nil
		},
	},
	{
		Name:        "assign_permission_to_role",
		Description: "Add permissions to a role (role-permission binding)",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"role_id":     map[string]any{"type": "string", "description": "Role UUID"},
				"permissions": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Permission keys to add"},
			},
			"required": []string{"role_id", "permissions"},
		},
		RequiredScopes: []string{"roles:write"},
		Handler: func(ctx context.Context, c *client.Client, args map[string]any) (any, error) {
			body := map[string]any{
				"permissions": argStrSlice(args, "permissions"),
			}
			var result any
			if err := c.Post(ctx, fmt.Sprintf("/api/v1/roles/%s/permissions", argStr(args, "role_id")), body, &result); err != nil {
				return nil, err
			}
			return result, nil
		},
	},
	{
		Name:        "list_role_permissions",
		Description: "List all permissions assigned to a role",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"role_id": map[string]any{"type": "string", "description": "Role UUID"},
			},
			"required": []string{"role_id"},
		},
		RequiredScopes: []string{"roles:read"},
		Handler: func(ctx context.Context, c *client.Client, args map[string]any) (any, error) {
			var result any
			if err := c.Get(ctx, fmt.Sprintf("/api/v1/roles/%s/permissions", argStr(args, "role_id")), &result); err != nil {
				return nil, err
			}
			return result, nil
		},
	},

	// === Webhook 管理 ===
	{
		Name:        "list_webhooks",
		Description: "List all webhooks configured for the tenant",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{},
		},
		RequiredScopes: []string{"users:read"},
		Handler: func(ctx context.Context, c *client.Client, args map[string]any) (any, error) {
			var result any
			if err := c.Get(ctx, "/api/v1/webhooks", &result); err != nil {
				return nil, err
			}
			return result, nil
		},
	},
	{
		Name:        "create_webhook",
		Description: "Create a new webhook endpoint for event notifications",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"url":    map[string]any{"type": "string", "description": "Webhook endpoint URL"},
				"events": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Event types to subscribe to"},
			},
			"required": []string{"url"},
		},
		RequiredScopes: []string{"users:write"},
		Handler: func(ctx context.Context, c *client.Client, args map[string]any) (any, error) {
			body := map[string]any{
				"url": argStr(args, "url"),
			}
			if v := argStrSlice(args, "events"); len(v) > 0 {
				body["events"] = v
			}
			var result any
			if err := c.Post(ctx, "/api/v1/webhooks", body, &result); err != nil {
				return nil, err
			}
			return result, nil
		},
	},
	{
		Name:        "delete_webhook",
		Description: "Delete a webhook by ID",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"webhook_id": map[string]any{"type": "string"},
			},
			"required": []string{"webhook_id"},
		},
		RequiredScopes: []string{"users:write"},
		Handler: func(ctx context.Context, c *client.Client, args map[string]any) (any, error) {
			var result any
			if err := c.Delete(ctx, fmt.Sprintf("/api/v1/webhooks/%s", argStr(args, "webhook_id")), &result); err != nil {
				return nil, err
			}
			return result, nil
		},
	},

	// === API Key 管理 ===
	{
		Name:        "list_api_keys",
		Description: "List all API keys for the tenant",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{},
		},
		RequiredScopes: []string{"oauth:read"},
		Handler: func(ctx context.Context, c *client.Client, args map[string]any) (any, error) {
			var result any
			if err := c.Get(ctx, "/api/v1/api-keys", &result); err != nil {
				return nil, err
			}
			return result, nil
		},
	},
	{
		Name:        "create_api_key",
		Description: "Create a new API key for M2M authentication",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name":        map[string]any{"type": "string", "description": "API key name"},
				"scopes":      map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Permission scopes"},
				"expires_in":  map[string]any{"type": "integer", "description": "Expiry in seconds (0 = no expiry)"},
			},
			"required": []string{"name"},
		},
		RequiredScopes: []string{"oauth:write"},
		Handler: func(ctx context.Context, c *client.Client, args map[string]any) (any, error) {
			body := map[string]any{
				"name": argStr(args, "name"),
			}
			if v := argStrSlice(args, "scopes"); len(v) > 0 {
				body["scopes"] = v
			}
			if v := argInt(args, "expires_in", 0); v > 0 {
				body["expires_in"] = v
			}
			var result any
			if err := c.Post(ctx, "/api/v1/api-keys", body, &result); err != nil {
				return nil, err
			}
			return result, nil
		},
	},
	{
		Name:        "delete_api_key",
		Description: "Revoke/delete an API key by ID",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"key_id": map[string]any{"type": "string"},
			},
			"required": []string{"key_id"},
		},
		RequiredScopes: []string{"oauth:write"},
		Handler: func(ctx context.Context, c *client.Client, args map[string]any) (any, error) {
			var result any
			if err := c.Delete(ctx, fmt.Sprintf("/api/v1/api-keys/%s", argStr(args, "key_id")), &result); err != nil {
				return nil, err
			}
			return result, nil
		},
	},

	// === IdP 配置 ===
	{
		Name:        "list_idp_configs",
		Description: "List all identity provider configurations (SAML, LDAP, social, SCIM)",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{},
		},
		RequiredScopes: []string{"tenants:read"},
		Handler: func(ctx context.Context, c *client.Client, args map[string]any) (any, error) {
			var result any
			if err := c.Get(ctx, "/api/v1/idp/config", &result); err != nil {
				return nil, err
			}
			return result, nil
		},
	},
	{
		Name:        "list_social_connectors",
		Description: "List available social login connectors (Google, GitHub, Microsoft, etc.)",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{},
		},
		RequiredScopes: []string{"tenants:read"},
		Handler: func(ctx context.Context, c *client.Client, args map[string]any) (any, error) {
			var result any
			if err := c.Get(ctx, "/api/v1/auth/social/connectors", &result); err != nil {
				return nil, err
			}
			return result, nil
		},
	},
	{
		Name:        "list_saml_configs",
		Description: "List SAML IdP configurations for the tenant",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{},
		},
		RequiredScopes: []string{"tenants:read"},
		Handler: func(ctx context.Context, c *client.Client, args map[string]any) (any, error) {
			var result any
			if err := c.Get(ctx, "/saml/config", &result); err != nil {
				return nil, err
			}
			return result, nil
		},
	},
	{
		Name:        "list_scim_configs",
		Description: "List SCIM provisioning configurations and sync status",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{},
		},
		RequiredScopes: []string{"tenants:read"},
		Handler: func(ctx context.Context, c *client.Client, args map[string]any) (any, error) {
			var result any
			if err := c.Get(ctx, "/api/v1/identity/scim/config", &result); err != nil {
				return nil, err
			}
			return result, nil
		},
	},
	{
		Name:        "sync_scim",
		Description: "Trigger a SCIM sync to provision/deprovision users from external IdP",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{},
		},
		RequiredScopes: []string{"tenants:write"},
		Handler: func(ctx context.Context, c *client.Client, args map[string]any) (any, error) {
			var result any
			if err := c.Post(ctx, "/api/v1/identity/scim/config/sync", nil, &result); err != nil {
				return nil, err
			}
			return result, nil
		},
	},

	// === 条件访问 / 安全策略 ===
	{
		Name:        "list_conditional_access_policies",
		Description: "List conditional access policies (IP-based, device-based, time-based rules)",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{},
		},
		RequiredScopes: []string{"policies:read"},
		Handler: func(ctx context.Context, c *client.Client, args map[string]any) (any, error) {
			var result any
			if err := c.Get(ctx, "/api/v1/policies/conditional-access", &result); err != nil {
				return nil, err
			}
			return result, nil
		},
	},
	{
		Name:        "create_conditional_access_policy",
		Description: "Create a conditional access policy (IP restrictions, device posture, time-based access)",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name":        map[string]any{"type": "string"},
				"description": map[string]any{"type": "string"},
				"effect":      map[string]any{"type": "string", "enum": []string{"allow", "deny"}},
				"conditions":  map[string]any{"type": "object", "description": "Conditions object with ip_ranges, device_posture, time_window, etc."},
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
			if v, ok := args["conditions"]; ok {
				body["conditions"] = v
			}
			var result any
			if err := c.Post(ctx, "/api/v1/policies/conditional-access", body, &result); err != nil {
				return nil, err
			}
			return result, nil
		},
	},

	// === 密码策略 / MFA ===
	{
		Name:        "get_password_policy",
		Description: "Get current password policy (complexity, expiry, history rules)",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{},
		},
		RequiredScopes: []string{"tenants:read"},
		Handler: func(ctx context.Context, c *client.Client, args map[string]any) (any, error) {
			var result any
			if err := c.Get(ctx, "/api/v1/auth/password/policy", &result); err != nil {
				return nil, err
			}
			return result, nil
		},
	},
	{
		Name:        "update_password_policy",
		Description: "Update password policy (min length, complexity, expiry, history)",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"min_length":      map[string]any{"type": "integer"},
				"require_upper":   map[string]any{"type": "boolean"},
				"require_lower":   map[string]any{"type": "boolean"},
				"require_digit":   map[string]any{"type": "boolean"},
				"require_symbol":  map[string]any{"type": "boolean"},
				"expiry_days":    map[string]any{"type": "integer"},
				"history_count":  map[string]any{"type": "integer"},
			},
		},
		RequiredScopes: []string{"tenants:write"},
		Handler: func(ctx context.Context, c *client.Client, args map[string]any) (any, error) {
			body := map[string]any{}
			for _, k := range []string{"min_length", "require_upper", "require_lower", "require_digit", "require_symbol", "expiry_days", "history_count"} {
				if v, ok := args[k]; ok {
					body[k] = v
				}
			}
			var result any
			if err := c.Patch(ctx, "/api/v1/auth/password-policy", body, &result); err != nil {
				return nil, err
			}
			return result, nil
		},
	},
	{
		Name:        "list_mfa_methods",
		Description: "List available MFA methods (TOTP, SMS, email, WebAuthn/passkey)",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{},
		},
		RequiredScopes: []string{"tenants:read"},
		Handler: func(ctx context.Context, c *client.Client, args map[string]any) (any, error) {
			var result any
			if err := c.Get(ctx, "/api/v1/auth/mfa/methods", &result); err != nil {
				return nil, err
			}
			return result, nil
		},
	},

	// === 部门 / 团队 ===
	{
		Name:        "list_departments",
		Description: "List all departments in the organization",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"org_id": map[string]any{"type": "string", "description": "Organization UUID"},
			},
		},
		RequiredScopes: []string{"orgs:read"},
		Handler: func(ctx context.Context, c *client.Client, args map[string]any) (any, error) {
			path := "/api/v1/departments"
			if oid := argStr(args, "org_id"); oid != "" {
				path += "?org_id=" + oid
			}
			var result any
			if err := c.Get(ctx, path, &result); err != nil {
				// Fallback: try /api/v1/orgs/departments
				if err2 := c.Get(ctx, "/api/v1/orgs/departments", &result); err2 != nil {
					return nil, err
				}
			}
			return result, nil
		},
	},
	{
		Name:        "list_teams",
		Description: "List all teams in the organization",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"org_id": map[string]any{"type": "string", "description": "Organization UUID (optional)"},
			},
		},
		RequiredScopes: []string{"orgs:read"},
		Handler: func(ctx context.Context, c *client.Client, args map[string]any) (any, error) {
			path := "/api/v1/teams"
			if oid := argStr(args, "org_id"); oid != "" {
				path += "?org_id=" + oid
			}
			var result any
			if err := c.Get(ctx, path, &result); err != nil {
				return nil, err
			}
			return result, nil
		},
	},

	// === 用户导入/导出 ===
	{
		Name:        "export_users",
		Description: "Export all users in the tenant to CSV/JSON",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"format": map[string]any{"type": "string", "enum": []string{"csv", "json"}, "description": "Export format (default csv)"},
			},
		},
		RequiredScopes: []string{"users:read"},
		Handler: func(ctx context.Context, c *client.Client, args map[string]any) (any, error) {
			path := "/api/v1/users/export"
			if f := argStr(args, "format"); f != "" {
				path += "?format=" + f
			}
			var result any
			if err := c.Get(ctx, path, &result); err != nil {
				return nil, err
			}
			return result, nil
		},
	},

	// === Feature Flags ===
	{
		Name:        "list_feature_flags",
		Description: "List all feature flags for the tenant",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{},
		},
		RequiredScopes: []string{"tenants:read"},
		Handler: func(ctx context.Context, c *client.Client, args map[string]any) (any, error) {
			var result any
			if err := c.Get(ctx, "/api/v1/admin/feature-flags", &result); err != nil {
				return nil, err
			}
			return result, nil
		},
	},
	{
		Name:        "set_feature_flag",
		Description: "Set a feature flag value (enable/disable/toggle)",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name":  map[string]any{"type": "string", "description": "Feature flag name"},
				"value": map[string]any{"type": "boolean"},
			},
			"required": []string{"name", "value"},
		},
		RequiredScopes: []string{"tenants:write"},
		Handler: func(ctx context.Context, c *client.Client, args map[string]any) (any, error) {
			body := map[string]any{
				"name":  argStr(args, "name"),
				"value": args["value"],
			}
			var result any
			if err := c.Post(ctx, "/api/v1/admin/feature-flags", body, &result); err != nil {
				return nil, err
			}
			return result, nil
		},
	},
}