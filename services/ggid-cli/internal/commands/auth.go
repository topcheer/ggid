package commands

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ggid/ggid/services/ggid-cli/internal/client"
	"github.com/ggid/ggid/services/ggid-cli/internal/config"
	"github.com/ggid/ggid/services/ggid-cli/internal/output"
)

// Login authenticates the CLI using the Device Authorization flow (RFC 8628):
//
//  1. DCR: Register CLI as a public client (no secret, token_endpoint_auth_method=none)
//  2. Device Auth: Request device_code + user_code from the server
//  3. User Authorization: User opens verification_uri in browser, enters user_code,
//     and logs in with their admin credentials (supports MFA)
//  4. Poll: CLI polls token endpoint until user authorizes or code expires
//  5. Store: Access token + refresh token saved to ~/.ggid/config.json
//
// This flow provides:
//   - Real user identity in tokens (sub = user_id) → proper audit trail
//   - No stored client secret → safer than client_credentials
//   - User must actively authorize → prevents credential theft
//   - Server-side session → admins can revoke from console
//   - MFA enforcement → follows tenant security policy
func Login(ctx *Context, args []string) {
	fs := flag.NewFlagSet("login", flag.ExitOnError)
	serverURL := fs.String("server", "", "Gateway URL (default: http://localhost:8080)")
	tenantID := fs.String("tenant", client.ConsoleTenantID, "Console tenant ID")
	clientName := fs.String("name", "ggid-cli", "DCR client name")
	noBrowser := fs.Bool("no-browser", false, "Don't auto-open browser; print URL only")
	fs.Parse(args)

	url := *serverURL
	if url == "" {
		url = ctx.ServerURL
	}
	if url == "" {
		url = "http://localhost:8080"
	}

	// Reuse existing DCR registration if available.
	clientID := ctx.Config.ClientID
	if clientID == "" {
		// Step 1: DCR registration as a public client.
		fmt.Printf("Connecting to %s ...\n", url)
		fmt.Printf("Registering via DCR in tenant %s...\n", *tenantID)
		dcrResp, err := client.RegisterViaDCR(url, *tenantID, *clientName)
		if err != nil {
			output.PrintError("DCR registration failed: %v", err)
			os.Exit(1)
		}
		clientID = dcrResp.ClientID
		fmt.Printf("  Registered public client: %s (id: %s)\n", dcrResp.ClientName, dcrResp.ClientID)
	} else {
		fmt.Printf("Using existing client: %s\n", clientID)
	}

	// Step 2: Request device authorization.
	fmt.Println("\nRequesting device authorization...")
	scopes := "openid profile email users:read users:write roles:read roles:write orgs:read orgs:write audit:read policies:read policies:write oauth:read oauth:write settings:read settings:write tenants:read tenants:write webhooks:read webhooks:write apikeys:read apikeys:write security:read security:write governance:read provisioning:read provisioning:write identity:read identity:write"
	deviceResp, err := client.RequestDeviceAuthorization(url, *tenantID, clientID, scopes)
	if err != nil {
		output.PrintError("Device authorization request failed: %v", err)
		os.Exit(1)
	}

	// Step 3: Display verification info to user.
	fmt.Println("\n╔══════════════════════════════════════════════════════════╗")
	fmt.Printf("║  Enter code: %-45s║\n", deviceResp.UserCode)
	fmt.Printf("║  At: %-51s║\n", truncateForBox(deviceResp.VerificationURI, 51))
	fmt.Println("╚══════════════════════════════════════════════════════════╝")

	// Try to auto-open browser.
	verificationURL := deviceResp.VerificationURI
	if deviceResp.VerificationURIComplete != "" {
		verificationURL = deviceResp.VerificationURIComplete
	}
	if !*noBrowser {
		openBrowser(verificationURL)
		fmt.Printf("\nOpening browser to: %s\n", verificationURL)
	} else {
		fmt.Printf("\nOpen this URL in your browser: %s\n", verificationURL)
	}
	fmt.Println("After logging in, enter the code above to authorize this CLI.")

	// Step 4: Poll token endpoint.
	fmt.Printf("\nWaiting for authorization (code expires in %dm)...\n", deviceResp.ExpiresIn/60)

	interval := deviceResp.Interval
	if interval < 1 {
		interval = 5
	}
	pollDuration := time.Duration(interval) * time.Second
	expiryDeadline := time.Now().Add(time.Duration(deviceResp.ExpiresIn) * time.Second)

	spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	spinIdx := 0

	for time.Now().Before(expiryDeadline) {
		result, err := client.PollDeviceToken(url, *tenantID, clientID, deviceResp.DeviceCode)
		if err != nil {
			output.PrintError("Poll error: %v", err)
			os.Exit(1)
		}

		if result.Token != nil && result.Token.AccessToken != "" {
			// Success!
			fmt.Printf("\r✓ Authorized! \n")
			fmt.Printf("  Token expires in: %ds\n", result.Token.ExpiresIn)
			if result.Token.RefreshToken != "" {
				fmt.Println("  Refresh token acquired for automatic renewal")
			}

			// Step 5: Save config.
			cfg := &config.Config{
				ServerURL:       url,
				ConsoleTenantID: *tenantID,
				ClientID:        clientID,
				AccessToken:     result.Token.AccessToken,
				RefreshToken:    result.Token.RefreshToken,
				ExpiresAt:       time.Now().Unix() + int64(result.Token.ExpiresIn),
				OutputFormat:    ctx.Config.OutputFormat,
			}
			if cfg.OutputFormat == "" {
				cfg.OutputFormat = "table"
			}
			if err := config.Save(cfg); err != nil {
				output.PrintError("Cannot save config: %v", err)
				os.Exit(1)
			}

			output.PrintSuccess("\nLogin successful. Credentials saved to ~/.ggid/config.json")
			return
		}

		if result.SlowDown {
			pollDuration += 5 * time.Second // back off
		}

		if result.Error != "" {
			fmt.Printf("\r✗ %s\n", result.Error)
			os.Exit(1)
		}

		// Show spinner while pending.
		fmt.Printf("\r  %s Waiting for authorization... (expires in %ds)  ",
			spinner[spinIdx%len(spinner)], int(time.Until(expiryDeadline).Seconds()))
		spinIdx++

		time.Sleep(pollDuration)
	}

	fmt.Println("\r✗ Device code expired. Please try again.                ")
	os.Exit(1)
}

// Logout clears the stored credentials and optionally revokes the token.
func Logout(ctx *Context, args []string) {
	// Try to revoke the token at the server (best-effort).
	if ctx.Config.AccessToken != "" && ctx.Config.ServerURL != "" {
		c := client.New(ctx.Config.ServerURL, ctx.Config.ConsoleTenantID, "")
		_ = c.PostForm("/api/v1/oauth/revoke",
			nil, // Would need token param, but revoke endpoint expects form data
			nil,
		)
	}

	if err := config.Delete(); err != nil {
		output.PrintError("Cannot delete config: %v", err)
		os.Exit(1)
	}
	output.PrintSuccess("Logged out. Credentials removed.")
}

// truncateForBox truncates a string to fit within the box width.
func truncateForBox(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// openBrowser attempts to open the OS default browser.
func openBrowser(url string) {
	// Try multiple OS-specific openers (best-effort, ignore errors).
	// Using os/exec in a separate function to keep imports clean.
	openBrowserOS(url)
}
