package commands

import (
	"flag"
	"fmt"
	"os"

	"github.com/ggid/ggid/services/ggid-cli/internal/output"
)

// System handles system-level subcommands.
func System(ctx *Context, args []string) {
	if len(args) == 0 {
		systemUsage()
		os.Exit(1)
	}
	sub := args[0]
	rest := args[1:]

	switch sub {
	case "health":
		systemHealth(ctx, rest)
	case "status":
		systemStatus(ctx, rest)
	case "bootstrap":
		systemBootstrap(ctx, rest)
	case "initialized":
		systemInitialized(ctx, rest)
	case "routes":
		systemRoutes(ctx, rest)
	case "help", "--help", "-h":
		systemUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown system subcommand: %s\n\n", sub)
		systemUsage()
		os.Exit(1)
	}
}

func systemUsage() {
	fmt.Println(`USAGE: ggid system <subcommand> [flags]

SUBCOMMANDS:
  health                          Show system health overview
  status                          Show system status
  bootstrap --admin-user X        Bootstrap initial admin user
  initialized                     Check if system is initialized
  routes                          List gateway routes`)
}

func systemHealth(ctx *Context, args []string) {
	c := requireClient(ctx)
	var result map[string]any
	if err := c.Get("/api/v1/health", &result); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}

	if isJSON(ctx) {
		output.PrintJSON(result)
		return
	}

	// Print service health table.
	services := extractList(result, "services", "data")
	t := output.NewTable("SERVICE", "STATUS", "URL")
	for _, s := range services {
		t.AddRow(
			getStr(s, "name"),
			getStr(s, "status"),
			output.Truncate(getStr(s, "url"), 40),
		)
	}
	t.Print()
}

func systemStatus(ctx *Context, args []string) {
	c := requireClient(ctx)
	var result map[string]any
	if err := c.Get("/api/v1/system/status", &result); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}
	output.PrintJSON(result)
}

func systemBootstrap(ctx *Context, args []string) {
	fs := flag.NewFlagSet("system bootstrap", flag.ExitOnError)
	adminUser := fs.String("admin-user", "", "Admin username (required)")
	adminPass := fs.String("admin-password", "", "Admin password (required)")
	adminEmail := fs.String("admin-email", "", "Admin email")
	fs.Parse(args)

	if *adminUser == "" || *adminPass == "" {
		output.PrintError("--admin-user and --admin-password are required")
		os.Exit(1)
	}

	c := requireClient(ctx)
	body := map[string]any{
		"admin_username": *adminUser,
		"admin_password": *adminPass,
	}
	if *adminEmail != "" {
		body["admin_email"] = *adminEmail
	}

	var result map[string]any
	if err := c.Post("/api/v1/system/bootstrap", body, &result); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}
	output.PrintSuccess("System bootstrapped successfully")
	output.PrintJSON(result)
}

func systemInitialized(ctx *Context, args []string) {
	c := requireClient(ctx)
	var result map[string]any
	if err := c.Get("/api/v1/system/initialized", &result); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}
	if initialized, ok := result["initialized"].(bool); ok {
		if initialized {
			fmt.Println("System is initialized.")
		} else {
			fmt.Println("System is NOT initialized. Run 'ggid system bootstrap'.")
		}
	} else {
		output.PrintJSON(result)
	}
}

func systemRoutes(ctx *Context, args []string) {
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

	t := output.NewTable("PREFIX", "BACKEND", "TIMEOUT")
	for _, r := range routes {
		t.AddRow(
			output.Truncate(getStr(r, "prefix"), 30),
			output.Truncate(getStr(r, "backend"), 40),
			getStr(r, "read_timeout"),
		)
	}
	t.Print()
}
