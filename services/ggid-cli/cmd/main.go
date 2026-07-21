// Package main implements the ggid-cli command line tool.
//
// ggid-cli is a comprehensive CLI for managing GGID instances remotely.
// Authentication uses Dynamic Client Registration (RFC 7591) to register
// in the console tenant, then exchanges client credentials for access tokens.
package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/ggid/ggid/services/ggid-cli/internal/client"
	"github.com/ggid/ggid/services/ggid-cli/internal/commands"
	"github.com/ggid/ggid/services/ggid-cli/internal/config"
)

const version = "1.0.0"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(0)
	}

	// Load config (may not exist yet for login).
	cfg, _ := config.Load()

	// Determine output format.
	outputFormat := cfg.OutputFormat
	if envFmt := os.Getenv("GGID_OUTPUT"); envFmt != "" {
		outputFormat = envFmt
	}
	if outputFormat == "" {
		outputFormat = "table"
	}

	// Parse global flags that may appear before the subcommand.
	serverURL := os.Getenv("GGID_SERVER_URL")
	if cfg.ServerURL != "" {
		serverURL = cfg.ServerURL
	}

	// Create the authenticated client if we have a token.
	var apiClient *client.Client
	if serverURL != "" {
		tenantID := cfg.ConsoleTenantID
		if tenantID == "" {
			tenantID = client.ConsoleTenantID
		}
		apiClient = client.New(serverURL, tenantID, cfg.AccessToken)
	}

	// Create the command context.
	ctx := &commands.Context{
		Config:       cfg,
		Client:       apiClient,
		ServerURL:    serverURL,
		OutputFormat: outputFormat,
		Version:      version,
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "login":
		commands.Login(ctx, args)
	case "logout":
		commands.Logout(ctx, args)
	case "whoami":
		commands.Whoami(ctx, args)
	case "version", "--version", "-v":
		fmt.Printf("ggid-cli v%s\n", version)
	case "help", "--help", "-h":
		usage()

	// Users
	case "users":
		commands.Users(ctx, args)
	case "user":
		commands.Users(ctx, args)

	// Roles
	case "roles":
		commands.Roles(ctx, args)
	case "role":
		commands.Roles(ctx, args)

	// Organizations
	case "orgs", "organizations":
		commands.Organizations(ctx, args)
	case "org", "organization":
		commands.Organizations(ctx, args)

	// Audit
	case "audit":
		commands.Audit(ctx, args)

	// Policies
	case "policies":
		commands.Policies(ctx, args)
	case "policy":
		commands.Policies(ctx, args)

	// OAuth Clients
	case "oauth":
		commands.OAuthClients(ctx, args)

	// Tenants
	case "tenants":
		commands.Tenants(ctx, args)
	case "tenant":
		commands.Tenants(ctx, args)

	// Sessions
	case "sessions":
		commands.Sessions(ctx, args)
	case "session":
		commands.Sessions(ctx, args)

	// System
	case "system":
		commands.System(ctx, args)

	// Webhooks
	case "webhooks":
		commands.Webhooks(ctx, args)
	case "webhook":
		commands.Webhooks(ctx, args)

	// Dashboard
	case "dashboard":
		commands.Dashboard(ctx, args)

	// API Keys
	case "api-keys", "apikeys", "apikey":
		commands.APIKeys(ctx, args)

	// Settings
	case "settings", "setting":
		commands.Settings(ctx, args)

	// Security
	case "security":
		commands.Security(ctx, args)

	// Monitoring
	case "monitoring":
		commands.Monitoring(ctx, args)

	// Shell completion
	case "completion":
		commands.Completion(ctx, args)

	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", cmd)
		usage()
		os.Exit(1)
	}
}

func usage() {
	buildTime := ""
	if bt, err := strconv.ParseInt("0", 10, 64); err == nil {
		_ = bt
		buildTime = time.Now().Format("2006-01-02")
	}
	_ = buildTime
	fmt.Println(`ggid-cli — GGID Identity & Access Management CLI v` + version + `

USAGE:
  ggid <command> [subcommand] [flags]

AUTHENTICATION:
  The CLI uses Dynamic Client Registration (RFC 7591) to register a public
  client in the console tenant, then authenticates via Device Authorization
  flow (RFC 8628) — the user opens a browser, enters a code, and logs in
  with their admin credentials. No stored secrets, proper audit trail.

  Run 'ggid login' to authenticate. Credentials are stored in ~/.ggid/config.json

COMMANDS:

  AUTH:
    login              Authenticate via device code flow (interactive)
    logout             Clear stored credentials
    whoami             Show current identity and token info
    version            Show CLI version

  USERS:
    users list         List users [--page N] [--size N] [--search STR]
    users get <id>     Get user details
    users create       Create a user (--username, --email, --password, ...)
    users update <id>  Update a user
    users delete <id>  Delete a user
    users lock <id>    Lock a user account
    users unlock <id>  Unlock a user account

  ROLES:
    roles list         List roles
    roles get <id>     Get role details
    roles create       Create a role (--name, --permissions)
    roles delete <id>  Delete a role

  ORGANIZATIONS:
    orgs list          List organizations
    orgs get <id>      Get organization details
    orgs create        Create an organization
    orgs delete <id>   Delete an organization

  AUDIT:
    audit events       Query audit events [--page N] [--size N]
    audit dashboard    Show audit dashboard

  POLICIES:
    policies list      List policies
    policies get <id>  Get policy details

  OAUTH CLIENTS:
    oauth clients list    List OAuth clients
    oauth clients get <id>  Get client details
    oauth clients create  Create an OAuth client
    oauth clients delete <id>  Delete an OAuth client

  TENANTS:
    tenants list       List tenants
    tenants get <id>   Get tenant details
    tenants create     Create a tenant

  SESSIONS:
    sessions list      List active sessions
    sessions revoke <id>  Revoke a session

  SYSTEM:
    system health      Show system health
    system status      Show system status
    system bootstrap   Bootstrap initial admin

  WEBHOOKS:
    webhooks list      List webhooks
    webhooks create    Create a webhook

  DASHBOARD:
    dashboard          Show aggregate dashboard stats

  API KEYS:
    api-keys list      List API keys
    api-keys create    Create an API key
    api-keys delete <id>  Revoke an API key

  SETTINGS:
    settings get       Show all settings
    settings branding  View branding
    settings flags     View feature flags
    settings mfa       View MFA settings
    settings ldap      View LDAP config

  SECURITY:
    security sessions  View session details
    security cae       View CAE monitor
    security threats   View threat dashboard
    security posture   View device posture

  MONITORING:
    monitoring gateway   Gateway stats
    monitoring routes    Route health
    monitoring activity  Activity log

GLOBAL FLAGS:
  --server URL    Gateway URL (first run prompts; saved to config)
  --json          Force JSON output
  --table         Force table output (default)

ENVIRONMENT VARIABLES:
  GGID_SERVER_URL     Gateway base URL
  GGID_OUTPUT         Output format: json or table`)
}
