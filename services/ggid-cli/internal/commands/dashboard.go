package commands

import (
	"fmt"
	"os"

	"github.com/ggid/ggid/services/ggid-cli/internal/output"
)

// Dashboard shows aggregate dashboard statistics.
func Dashboard(ctx *Context, args []string) {
	c := requireClient(ctx)
	var result map[string]any
	if err := c.Get("/api/v1/dashboard/stats", &result); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}

	if isJSON(ctx) {
		output.PrintJSON(result)
		return
	}

	fmt.Println("=== GGID Dashboard ===")
	if v, ok := result["total_users"]; ok {
		fmt.Printf("Total Users:        %v\n", v)
	}
	if v, ok := result["active_sessions"]; ok {
		fmt.Printf("Active Sessions:    %v\n", v)
	}
	if v, ok := result["login_rate_per_hour"]; ok {
		fmt.Printf("Logins/Hour:        %v\n", v)
	}
	if v, ok := result["mfa_adoption_pct"]; ok {
		fmt.Printf("MFA Adoption:       %v%%\n", v)
	}
	// Print any additional fields.
	for k, v := range result {
		switch k {
		case "total_users", "active_sessions", "login_rate_per_hour", "mfa_adoption_pct":
			continue
		default:
			fmt.Printf("%s: %v\n", k, v)
		}
	}
}
