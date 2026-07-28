package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Providers handles provider configuration subcommands.
func Providers(ctx *Context, args []string) {
	if len(args) == 0 {
		providersUsage()
		os.Exit(1)
	}
	switch args[0] {
	case "status":
		providersStatus(ctx, args[1:])
	case "config":
		providersConfig(ctx, args[1:])
	case "help", "--help", "-h":
		providersUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown providers subcommand: %s\n\n", args[0])
		providersUsage()
		os.Exit(1)
	}
}

func providersUsage() {
	fmt.Println(`Provider configuration management

Usage:
  ggid providers <command> [flags]

Commands:
  status                     Show auth method availability
  config get <key>           Get hierarchical config value
  config set <key> <json>    Set config value at specified scope
  config list                List all provider configs
  config delete <key>        Delete config at specified scope

Config Keys:
  sms_provider, email_provider, password_policy, session_config,
  token_config, cors_config, mfa_enforcement, branding, audit_retention

Scope Flags:
  --scope instance|tenant|client
  --tenant <id>              Tenant ID (required for tenant/client scope)
  --client <id>              Client ID (required for client scope)

Examples:
  ggid providers status
  ggid providers config get sms_provider --tenant abc-123
  ggid providers config set sms_provider '{"provider":"twilio","from":"+123"}' --scope instance`)
}

func providersStatus(ctx *Context, args []string) {
	var result map[string]interface{}
	if err := ctx.Client.Get("/api/v1/providers/status", &result); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if ctx.OutputFormat == "json" {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return
	}

	fmt.Println("Authentication Method Availability:")
	fmt.Println()
	for key, val := range result {
		if methods, ok := val.(map[string]interface{}); ok {
			fmt.Printf("  %s:\n", key)
			for method, available := range methods {
				check := "x"
				if avail, ok := available.(bool); ok && avail {
					check = "OK"
				}
				fmt.Printf("    %-20s [%s]\n", method, check)
			}
		} else {
			fmt.Printf("  %-20s %v\n", key, val)
		}
	}
}

func providersConfig(ctx *Context, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: ggid providers config <get|set|list|delete> [args]")
		os.Exit(1)
	}
	switch args[0] {
	case "get":
		providersConfigGet(ctx, args[1:])
	case "set":
		providersConfigSet(ctx, args[1:])
	case "list":
		providersConfigList(ctx, args[1:])
	case "delete", "rm":
		providersConfigDelete(ctx, args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown config action: %s\n", args[0])
		os.Exit(1)
	}
}

func parseScopeFlags(args []string) (scope, tenantID, clientID string, remaining []string) {
	scope = "instance"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--scope":
			if i+1 < len(args) {
				scope = args[i+1]
				i++
			}
		case "--tenant":
			if i+1 < len(args) {
				tenantID = args[i+1]
				i++
			}
		case "--client":
			if i+1 < len(args) {
				clientID = args[i+1]
				i++
			}
		default:
			remaining = append(remaining, args[i])
		}
	}
	return
}

func buildQueryParams(scope, tenantID, clientID string) string {
	params := []string{"scope=" + scope}
	if tenantID != "" {
		params = append(params, "tenant_id="+tenantID)
	}
	if clientID != "" {
		params = append(params, "client_id="+clientID)
	}
	return "?" + strings.Join(params, "&")
}

func providersConfigGet(ctx *Context, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: ggid providers config get <key> [--scope ...]")
		os.Exit(1)
	}
	key := args[0]
	scope, tenantID, clientID, _ := parseScopeFlags(args[1:])

	path := fmt.Sprintf("/api/v1/providers/config/%s%s", key, buildQueryParams(scope, tenantID, clientID))
	var result map[string]interface{}
	if err := ctx.Client.Get(path, &result); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if ctx.OutputFormat == "json" {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return
	}

	fmt.Printf("Config: %s (scope: %s)\n\n", key, scope)
	prettyPrintMap(result, "  ")
}

func providersConfigSet(ctx *Context, args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: ggid providers config set <key> '<json>' [--scope ...]")
		os.Exit(1)
	}
	key := args[0]
	jsonStr := args[1]
	scope, tenantID, clientID, _ := parseScopeFlags(args[2:])

	var configValue interface{}
	if err := json.Unmarshal([]byte(jsonStr), &configValue); err != nil {
		fmt.Fprintf(os.Stderr, "invalid JSON: %v\n", err)
		os.Exit(1)
	}

	body := map[string]interface{}{
		"scope_type": scope,
		"config":     configValue,
		"enabled":    true,
	}
	if tenantID != "" {
		body["tenant_id"] = tenantID
	}
	if clientID != "" {
		body["client_id"] = clientID
	}

	path := fmt.Sprintf("/api/v1/providers/config/%s", key)
	var resp interface{}
	if err := ctx.Client.Put(path, body, &resp); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Set %s at scope '%s'\n", key, scope)
}

func providersConfigList(ctx *Context, args []string) {
	scope, tenantID, clientID, _ := parseScopeFlags(args)

	path := "/api/v1/providers/config" + buildQueryParams(scope, tenantID, clientID)
	var configs []map[string]interface{}
	if err := ctx.Client.Get(path, &configs); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if ctx.OutputFormat == "json" {
		data, _ := json.MarshalIndent(configs, "", "  ")
		fmt.Println(string(data))
		return
	}

	if len(configs) == 0 {
		fmt.Println("No provider configs found.")
		return
	}

	fmt.Printf("%-20s %-10s %-15s %-8s %-20s\n", "KEY", "SCOPE", "PROVIDER", "ENABLED", "UPDATED")
	fmt.Println(strings.Repeat("-", 80))
	for _, c := range configs {
		key, _ := c["config_key"].(string)
		sc, _ := c["scope_type"].(string)
		pt, _ := c["provider_type"].(string)
		enabled, _ := c["enabled"].(bool)
		updated, _ := c["updated_at"].(string)
		if len(updated) > 19 {
			updated = updated[:19]
		}
		en := "no"
		if enabled {
			en = "yes"
		}
		fmt.Printf("%-20s %-10s %-15s %-8s %-20s\n", key, sc, pt, en, updated)
	}
}

func providersConfigDelete(ctx *Context, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: ggid providers config delete <key> [--scope ...]")
		os.Exit(1)
	}
	key := args[0]
	scope, tenantID, clientID, _ := parseScopeFlags(args[1:])

	path := fmt.Sprintf("/api/v1/providers/config/%s%s", key, buildQueryParams(scope, tenantID, clientID))
	if err := ctx.Client.Delete(path); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Deleted %s at scope '%s'\n", key, scope)
}

func prettyPrintMap(m map[string]interface{}, indent string) {
	for key, val := range m {
		switch v := val.(type) {
		case map[string]interface{}:
			fmt.Printf("%s%s:\n", indent, key)
			prettyPrintMap(v, indent+"  ")
		default:
			fmt.Printf("%s%-20s %v\n", indent, key+":", v)
		}
	}
}
