package commands

import (
	"flag"
	"fmt"
	"os"

	"github.com/ggid/ggid/services/ggid-cli/internal/output"
)

// Tenants handles tenant management subcommands.
func Tenants(ctx *Context, args []string) {
	if len(args) == 0 {
		tenantsUsage()
		os.Exit(1)
	}
	sub := args[0]
	rest := args[1:]

	switch sub {
	case "list", "ls":
		tenantsList(ctx, rest)
	case "get":
		tenantsGet(ctx, rest)
	case "create":
		tenantsCreate(ctx, rest)
	case "delete", "rm":
		tenantsDelete(ctx, rest)
	case "resolve":
		tenantsResolve(ctx, rest)
	case "suspend":
		tenantsSuspend(ctx, rest)
	case "activate":
		tenantsActivate(ctx, rest)
	case "help", "--help", "-h":
		tenantsUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown tenants subcommand: %s\n\n", sub)
		tenantsUsage()
		os.Exit(1)
	}
}

func tenantsUsage() {
	fmt.Println(`USAGE: ggid tenants <subcommand> [flags]

SUBCOMMANDS:
  list                          List tenants
  get <id>                      Get tenant details
  create --name X --slug Y      Create a tenant
  delete <id>                   Delete a tenant
  resolve --slug Y              Resolve a tenant slug to UUID
  suspend <id>                  Suspend a tenant
  activate <id>                 Activate a suspended tenant`)
}

func tenantsList(ctx *Context, args []string) {
	c := requireClient(ctx)
	var result map[string]any
	if err := c.Get("/api/v1/tenants", &result); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}

	tenants := extractList(result, "tenants", "items", "data")
	if isJSON(ctx) {
		output.PrintJSON(tenants)
		return
	}

	t := output.NewTable("ID", "NAME", "SLUG", "STATUS", "CREATED")
	for _, tn := range tenants {
		t.AddRow(
			getStr(tn, "id"),
			output.Truncate(getStr(tn, "name"), 20),
			getStr(tn, "slug"),
			getStr(tn, "status"),
			getStr(tn, "created_at"),
		)
	}
	t.Print()
}

func tenantsGet(ctx *Context, args []string) {
	if len(args) == 0 {
		output.PrintError("tenant ID required: ggid tenants get <id>")
		os.Exit(1)
	}
	c := requireClient(ctx)
	var result map[string]any
	if err := c.Get("/api/v1/tenants/"+args[0], &result); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}
	output.PrintJSON(result)
}

func tenantsCreate(ctx *Context, args []string) {
	fs := flag.NewFlagSet("tenants create", flag.ExitOnError)
	name := fs.String("name", "", "Tenant name (required)")
	slug := fs.String("slug", "", "Tenant slug (required)")
	description := fs.String("description", "", "Tenant description")
	fs.Parse(args)

	if *name == "" || *slug == "" {
		output.PrintError("--name and --slug are required")
		os.Exit(1)
	}

	c := requireClient(ctx)
	body := map[string]any{
		"name": *name,
		"slug": *slug,
	}
	if *description != "" {
		body["description"] = *description
	}

	var result map[string]any
	if err := c.Post("/api/v1/org/tenants", body, &result); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}
	output.PrintSuccess("Tenant created: %s", getStr(result, "id"))
}

func tenantsDelete(ctx *Context, args []string) {
	if len(args) == 0 {
		output.PrintError("tenant ID required: ggid tenants delete <id>")
		os.Exit(1)
	}
	c := requireClient(ctx)
	if err := c.Delete("/api/v1/tenants/" + args[0]); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}
	output.PrintSuccess("Tenant deleted: %s", args[0])
}

func tenantsResolve(ctx *Context, args []string) {
	fs := flag.NewFlagSet("tenants resolve", flag.ExitOnError)
	slug := fs.String("slug", "", "Tenant slug (required)")
	fs.Parse(args)

	if *slug == "" {
		output.PrintError("--slug is required")
		os.Exit(1)
	}

	c := requireClient(ctx)
	var result map[string]any
	if err := c.Get("/api/v1/tenants/resolve?slug="+*slug, &result); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}
	output.PrintJSON(result)
}

func tenantsSuspend(ctx *Context, args []string) {
	if len(args) == 0 {
		output.PrintError("tenant ID required: ggid tenants suspend <id>")
		os.Exit(1)
	}
	c := requireClient(ctx)
	body := map[string]any{"tenant_id": args[0]}
	if err := c.Post("/api/v1/org/tenants/suspend", body, nil); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}
	output.PrintSuccess("Tenant suspended: %s", args[0])
}

func tenantsActivate(ctx *Context, args []string) {
	if len(args) == 0 {
		output.PrintError("tenant ID required: ggid tenants activate <id>")
		os.Exit(1)
	}
	c := requireClient(ctx)
	body := map[string]any{"tenant_id": args[0]}
	if err := c.Post("/api/v1/org/tenants/activate", body, nil); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}
	output.PrintSuccess("Tenant activated: %s", args[0])
}
