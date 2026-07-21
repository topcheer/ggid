package commands

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ggid/ggid/services/ggid-cli/internal/client"
	"github.com/ggid/ggid/services/ggid-cli/internal/config"
	"github.com/ggid/ggid/services/ggid-cli/internal/output"
)

// Login authenticates the CLI by:
// 1. Registering via DCR in the console tenant
// 2. Exchanging client credentials for an access token
// 3. Storing credentials in ~/.ggid/config.json
func Login(ctx *Context, args []string) {
	fs := flag.NewFlagSet("login", flag.ExitOnError)
	serverURL := fs.String("server", "", "Gateway URL (default: http://localhost:8080)")
	tenantID := fs.String("tenant", client.ConsoleTenantID, "Console tenant ID")
	clientName := fs.String("name", "ggid-cli", "DCR client name")
	fs.Parse(args)

	url := *serverURL
	if url == "" {
		url = ctx.ServerURL
	}
	if url == "" {
		url = "http://localhost:8080"
	}

	fmt.Printf("Connecting to %s ...\n", url)

	// Step 1: DCR registration.
	fmt.Printf("Registering via DCR in tenant %s...\n", *tenantID)
	dcrResp, err := client.RegisterViaDCR(url, *tenantID, *clientName)
	if err != nil {
		output.PrintError("DCR registration failed: %v", err)
		os.Exit(1)
	}
	fmt.Printf("  Registered client: %s (id: %s)\n", dcrResp.ClientName, dcrResp.ClientID)

	// Step 2: Exchange client credentials for token.
	fmt.Println("Exchanging client credentials for access token...")
	tokenResp, err := client.GetClientCredentialsToken(url, *tenantID, dcrResp.ClientID, dcrResp.ClientSecret)
	if err != nil {
		output.PrintError("Token exchange failed: %v", err)
		os.Exit(1)
	}
	fmt.Printf("  Got access token (expires in %ds)\n", tokenResp.ExpiresIn)

	// Step 3: Save config.
	cfg := &config.Config{
		ServerURL:       url,
		ConsoleTenantID: *tenantID,
		ClientID:        dcrResp.ClientID,
		ClientSecret:    dcrResp.ClientSecret,
		AccessToken:     tokenResp.AccessToken,
		ExpiresAt:       time.Now().Unix() + int64(tokenResp.ExpiresIn),
		OutputFormat:    "table",
	}
	if err := config.Save(cfg); err != nil {
		output.PrintError("Cannot save config: %v", err)
		os.Exit(1)
	}

	output.PrintSuccess("Login successful. Credentials saved to ~/.ggid/config.json")
}

// Logout clears the stored credentials.
func Logout(ctx *Context, args []string) {
	if err := config.Delete(); err != nil {
		output.PrintError("Cannot delete config: %v", err)
		os.Exit(1)
	}
	output.PrintSuccess("Logged out. Credentials removed.")
}

// Whoami shows the current identity and token information.
func Whoami(ctx *Context, args []string) {
	cfg := ctx.Config
	if cfg.AccessToken == "" {
		output.PrintError("not logged in. Run 'ggid login' first.")
		os.Exit(1)
	}

	fmt.Println("=== GGID CLI Session ===")
	fmt.Printf("Server:     %s\n", cfg.ServerURL)
	fmt.Printf("Tenant:     %s\n", cfg.ConsoleTenantID)
	fmt.Printf("Client ID:  %s\n", cfg.ClientID)

	// Decode JWT payload to show claims.
	parts := strings.Split(cfg.AccessToken, ".")
	if len(parts) >= 2 {
		payload := parts[1]
		// Add padding if needed.
		for len(payload)%4 != 0 {
			payload += "="
		}
		if decoded, err := base64.URLEncoding.DecodeString(payload); err == nil {
			var claims map[string]any
			if json.Unmarshal(decoded, &claims) == nil {
				fmt.Println("\n=== Token Claims ===")
				for k, v := range claims {
					fmt.Printf("  %s: %v\n", k, v)
				}
			}
		}
	}

	if cfg.ExpiresAt > 0 {
		expires := time.Unix(cfg.ExpiresAt, 0)
		remaining := time.Until(expires).Round(time.Second)
		if remaining > 0 {
			fmt.Printf("\nToken expires: %s (%s remaining)\n", expires.Format(time.RFC3339), remaining)
		} else {
			fmt.Printf("\nToken expired: %s (run 'ggid login' to refresh)\n", expires.Format(time.RFC3339))
		}
	}
}

// requireClient returns an authenticated API client or exits with an error.
func requireClient(ctx *Context) *client.Client {
	if ctx.Client == nil {
		output.PrintError("not logged in. Run 'ggid login' first.")
		os.Exit(1)
	}
	// Check token expiry — try to auto-refresh via client_credentials.
	if client.IsTokenExpired(ctx.Config.ExpiresAt) {
		if ctx.Config.ClientID != "" && ctx.Config.ClientSecret != "" && ctx.Config.ServerURL != "" {
			// Auto-refresh: exchange client credentials for a new token.
			tokenResp, err := client.GetClientCredentialsToken(
				ctx.Config.ServerURL,
				ctx.Config.ConsoleTenantID,
				ctx.Config.ClientID,
				ctx.Config.ClientSecret,
			)
			if err == nil && tokenResp.AccessToken != "" {
				ctx.Config.AccessToken = tokenResp.AccessToken
				ctx.Config.ExpiresAt = time.Now().Unix() + int64(tokenResp.ExpiresIn)
				// Save refreshed config.
				_ = config.Save(ctx.Config)
				// Update the client with new token.
				ctx.Client = client.New(
					ctx.Config.ServerURL,
					ctx.Config.ConsoleTenantID,
					ctx.Config.AccessToken,
				)
			} else {
				output.PrintError("token expired and auto-refresh failed. Run 'ggid login' to re-authenticate.")
				os.Exit(1)
			}
		} else {
			output.PrintError("token expired. Run 'ggid login' to re-authenticate.")
			os.Exit(1)
		}
	}
	return ctx.Client
}

// parseGlobalFlags extracts common flags from args.
func parseGlobalFlags(args []string) ([]string, string) {
	// Check for --json or --table at the end of args.
	filtered := make([]string, 0, len(args))
	formatOverride := ""
	for _, a := range args {
		switch a {
		case "--json":
			formatOverride = "json"
		case "--table":
			formatOverride = "table"
		default:
			filtered = append(filtered, a)
		}
	}
	return filtered, formatOverride
}

// printData outputs data in the configured format.
func printData(ctx *Context, formatOverride string, data any) {
	format := ctx.OutputFormat
	if formatOverride != "" {
		format = formatOverride
	}
	if format == "json" {
		output.PrintJSON(data)
	} else {
		output.PrintJSON(data) // default to JSON for complex objects
	}
}
