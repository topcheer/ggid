package commands

import (
	"fmt"
	"os"

	"github.com/ggid/ggid/services/ggid-cli/internal/output"
)

// Monitoring shows monitoring and analytics data.
func Monitoring(ctx *Context, args []string) {
	if len(args) == 0 {
		monitoringUsage()
		os.Exit(1)
	}
	sub := args[0]
	rest := args[1:]

	switch sub {
	case "gateway":
		monitoringGateway(ctx, rest)
	case "routes":
		monitoringRoutes(ctx, rest)
	case "rate-limits":
		monitoringRateLimits(ctx, rest)
	case "analytics":
		monitoringAnalytics(ctx, rest)
	case "activity":
		monitoringActivity(ctx, rest)
	case "help", "--help", "-h":
		monitoringUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown monitoring subcommand: %s\n\n", sub)
		monitoringUsage()
		os.Exit(1)
	}
}

func monitoringUsage() {
	fmt.Println(`USAGE: ggid monitoring <subcommand> [flags]

SUBCOMMANDS:
  gateway       Gateway stats and metrics
  routes        Route configuration and health
  rate-limits   Rate limit tiers and status
  analytics     User analytics
  activity      User activity log`)
}

func monitoringGateway(ctx *Context, args []string) {
	c := requireClient(ctx)
	var result map[string]any
	if err := c.Get("/api/v1/gateway/stats", &result); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}
	output.PrintJSON(result)
}

func monitoringRoutes(ctx *Context, args []string) {
	c := requireClient(ctx)
	var result map[string]any
	if err := c.Get("/api/v1/gateway/routes", &result); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}

	routes := extractList(result, "routes", "data")
	if isJSON(ctx) {
		output.PrintJSON(routes)
		return
	}

	t := output.NewTable("PREFIX", "BACKEND", "HAS TIMEOUT", "READ TIMEOUT")
	for _, r := range routes {
		t.AddRow(
			output.Truncate(getStr(r, "prefix"), 30),
			output.Truncate(getStr(r, "backend"), 35),
			getStr(r, "has_timeout"),
			getStr(r, "read_timeout"),
		)
	}
	t.Print()
}

func monitoringRateLimits(ctx *Context, args []string) {
	c := requireClient(ctx)
	var result map[string]any
	if err := c.Get("/api/v1/gateway/rate-limits", &result); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}
	output.PrintJSON(result)
}

func monitoringAnalytics(ctx *Context, args []string) {
	c := requireClient(ctx)
	var result map[string]any
	if err := c.Get("/api/v1/analytics", &result); err != nil {
		// Fallback to dashboard stats.
		if err2 := c.Get("/api/v1/dashboard/stats", &result); err2 != nil {
			output.PrintError("%v", err2)
			os.Exit(1)
		}
	}
	output.PrintJSON(result)
}

func monitoringActivity(ctx *Context, args []string) {
	c := requireClient(ctx)
	var result map[string]any
	if err := c.Get("/api/v1/activity", &result); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}

	events := extractList(result, "activity", "events", "items", "data")
	if isJSON(ctx) {
		output.PrintJSON(events)
		return
	}

	t := output.NewTable("TIME", "EVENT", "ACTOR", "RESOURCE", "IP")
	for _, e := range events {
		t.AddRow(
			getStr(e, "timestamp"),
			output.Truncate(getStr(e, "event"), 20),
			output.Truncate(getStr(e, "actor"), 15),
			output.Truncate(getStr(e, "resource"), 20),
			getStr(e, "ip_address"),
		)
	}
	t.Print()
}
