package commands

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ggid/ggid/services/ggid-cli/internal/client"
	"github.com/ggid/ggid/services/ggid-cli/internal/config"
	"github.com/ggid/ggid/services/ggid-cli/internal/output"
)

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

	// Decode JWT payload to show claims (especially sub = user identity).
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
				// Highlight user identity fields.
				if sub, ok := claims["sub"]; ok {
					fmt.Printf("\nUser ID:    %v\n", sub)
				}
				if name, ok := claims["name"]; ok {
					fmt.Printf("Name:       %v\n", name)
				}
				if email, ok := claims["email"]; ok {
					fmt.Printf("Email:      %v\n", email)
				}
				if scopes, ok := claims["scope"]; ok {
					fmt.Printf("Scopes:     %v\n", scopes)
				}
				if sid, ok := claims["sid"]; ok {
					fmt.Printf("Session ID: %v\n", sid)
				}

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
			fmt.Printf("\nToken expired: %s (will auto-refresh on next command)\n", expires.Format(time.RFC3339))
		}
	}
}

// requireClient returns an authenticated API client or exits with an error.
// If the token is expired, it attempts to refresh using the refresh_token grant.
func requireClient(ctx *Context) *client.Client {
	if ctx.Client == nil {
		output.PrintError("not logged in. Run 'ggid login' first.")
		os.Exit(1)
	}

	// Check token expiry — try to auto-refresh using refresh_token grant.
	if client.IsTokenExpired(ctx.Config.ExpiresAt) {
		// Try refresh_token grant first (preferred for device flow).
		if ctx.Config.RefreshToken != "" && ctx.Config.ClientID != "" && ctx.Config.ServerURL != "" {
			tokenResp, err := client.RefreshToken(
				ctx.Config.ServerURL,
				ctx.Config.ConsoleTenantID,
				ctx.Config.ClientID,
				ctx.Config.RefreshToken,
			)
			if err == nil && tokenResp.AccessToken != "" {
				ctx.Config.AccessToken = tokenResp.AccessToken
				ctx.Config.ExpiresAt = time.Now().Unix() + int64(tokenResp.ExpiresIn)
				if tokenResp.RefreshToken != "" {
					ctx.Config.RefreshToken = tokenResp.RefreshToken
				}
				// Save refreshed config.
				_ = config.Save(ctx.Config)
				// Update the client with new token.
				ctx.Client = client.New(
					ctx.Config.ServerURL,
					ctx.Config.ConsoleTenantID,
					ctx.Config.AccessToken,
				)
			} else {
				output.PrintError("session expired and token refresh failed. Run 'ggid login' to re-authenticate.")
				os.Exit(1)
			}
		} else {
			output.PrintError("session expired. Run 'ggid login' to re-authenticate.")
			os.Exit(1)
		}
	}
	return ctx.Client
}

// isJSON returns true if output should be JSON format.
func isJSON(ctx *Context) bool {
	return ctx.OutputFormat == "json"
}

// extractList extracts a list of objects from a response map.
func extractList(result map[string]any, keys ...string) []map[string]any {
	for _, key := range keys {
		if raw, ok := result[key]; ok {
			if arr, ok := raw.([]any); ok {
				list := make([]map[string]any, 0, len(arr))
				for _, item := range arr {
					if m, ok := item.(map[string]any); ok {
						list = append(list, m)
					}
				}
				return list
			}
		}
	}
	return nil
}

// getStr safely extracts a string value from a map.
func getStr(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		switch s := v.(type) {
		case string:
			return s
		case float64:
			return fmt.Sprintf("%v", s)
		case json.Number:
			return s.String()
		case bool:
			if s {
				return "true"
			}
			return "false"
		default:
			return fmt.Sprintf("%v", v)
		}
	}
	return ""
}
