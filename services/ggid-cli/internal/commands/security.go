package commands

import (
	"fmt"
	"os"

	"github.com/ggid/ggid/services/ggid-cli/internal/output"
)

// Security handles security-related subcommands.
func Security(ctx *Context, args []string) {
	if len(args) == 0 {
		securityUsage()
		os.Exit(1)
	}
	sub := args[0]
	rest := args[1:]

	switch sub {
	case "sessions":
		securitySessions(ctx, rest)
	case "cae":
		securityCAE(ctx, rest)
	case "threats":
		securityThreats(ctx, rest)
	case "posture":
		securityPosture(ctx, rest)
	case "risk-score":
		securityRiskScore(ctx, rest)
	case "help", "--help", "-h":
		securityUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown security subcommand: %s\n\n", sub)
		securityUsage()
		os.Exit(1)
	}
}

func securityUsage() {
	fmt.Println(`USAGE: ggid security <subcommand> [flags]

SUBCOMMANDS:
  sessions        View security session details
  cae             View CAE (Continuous Access Evaluation) monitor
  threats         View threat dashboard
  posture         View device posture
  risk-score      View risk scores`)
}

func securitySessions(ctx *Context, args []string) {
	c := requireClient(ctx)
	var result map[string]any
	if err := c.Get("/api/v1/security/session-detail", &result); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}
	output.PrintJSON(result)
}

func securityCAE(ctx *Context, args []string) {
	c := requireClient(ctx)
	var result map[string]any
	if err := c.Get("/api/v1/security/cae-monitor", &result); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}
	output.PrintJSON(result)
}

func securityThreats(ctx *Context, args []string) {
	c := requireClient(ctx)
	var result map[string]any
	if err := c.Get("/api/v1/admin/threats/dashboard", &result); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}
	output.PrintJSON(result)
}

func securityPosture(ctx *Context, args []string) {
	c := requireClient(ctx)
	var result map[string]any
	if err := c.Get("/api/v1/devices/posture/policies", &result); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}
	output.PrintJSON(result)
}

func securityRiskScore(ctx *Context, args []string) {
	c := requireClient(ctx)
	var result map[string]any
	if err := c.Get("/api/v1/security/risk-score", &result); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}
	output.PrintJSON(result)
}
