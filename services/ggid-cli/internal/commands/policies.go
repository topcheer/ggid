package commands

import (
	"flag"
	"fmt"
	"os"

	"github.com/ggid/ggid/services/ggid-cli/internal/output"
)

// Policies handles policy management subcommands.
func Policies(ctx *Context, args []string) {
	if len(args) == 0 {
		policiesUsage()
		os.Exit(1)
	}
	sub := args[0]
	rest := args[1:]

	switch sub {
	case "list", "ls":
		policiesList(ctx, rest)
	case "get":
		policiesGet(ctx, rest)
	case "create":
		policiesCreate(ctx, rest)
	case "delete", "rm":
		policiesDelete(ctx, rest)
	case "check":
		policiesCheck(ctx, rest)
	case "help", "--help", "-h":
		policiesUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown policies subcommand: %s\n\n", sub)
		policiesUsage()
		os.Exit(1)
	}
}

func policiesUsage() {
	fmt.Println(`USAGE: ggid policies <subcommand> [flags]

SUBCOMMANDS:
  list [--page N] [--size N]           List policies
  get <id>                              Get policy details
  create --name X [--description Y]     Create a policy
  delete <id>                           Delete a policy
  check --user ID --action X --res Y    Check if action is allowed`)
}

func policiesList(ctx *Context, args []string) {
	fs := flag.NewFlagSet("policies list", flag.ExitOnError)
	page := fs.Int("page", 1, "Page number")
	size := fs.Int("size", 20, "Page size")
	fs.Parse(args)

	c := requireClient(ctx)
	path := fmt.Sprintf("/api/v1/policies?page=%d&page_size=%d", *page, *size)
	var result map[string]any
	if err := c.Get(path, &result); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}

	policies := extractList(result, "policies", "items", "data")
	if isJSON(ctx) {
		output.PrintJSON(policies)
		return
	}

	t := output.NewTable("ID", "NAME", "DESCRIPTION", "EFFECT", "ENABLED")
	for _, p := range policies {
		t.AddRow(
			getStr(p, "id"),
			output.Truncate(getStr(p, "name"), 25),
			output.Truncate(getStr(p, "description"), 30),
			getStr(p, "effect"),
			getStr(p, "enabled"),
		)
	}
	t.Print()
}

func policiesGet(ctx *Context, args []string) {
	if len(args) == 0 {
		output.PrintError("policy ID required: ggid policies get <id>")
		os.Exit(1)
	}
	c := requireClient(ctx)
	var result map[string]any
	if err := c.Get("/api/v1/policies/"+args[0], &result); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}
	output.PrintJSON(result)
}

func policiesCreate(ctx *Context, args []string) {
	fs := flag.NewFlagSet("policies create", flag.ExitOnError)
	name := fs.String("name", "", "Policy name (required)")
	description := fs.String("description", "", "Policy description")
	effect := fs.String("effect", "allow", "Policy effect: allow or deny")
	fs.Parse(args)

	if *name == "" {
		output.PrintError("--name is required")
		os.Exit(1)
	}

	c := requireClient(ctx)
	body := map[string]any{
		"name":   *name,
		"effect": *effect,
	}
	if *description != "" {
		body["description"] = *description
	}

	var result map[string]any
	if err := c.Post("/api/v1/policies", body, &result); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}
	output.PrintSuccess("Policy created: %s", getStr(result, "id"))
}

func policiesDelete(ctx *Context, args []string) {
	if len(args) == 0 {
		output.PrintError("policy ID required: ggid policies delete <id>")
		os.Exit(1)
	}
	c := requireClient(ctx)
	if err := c.Delete("/api/v1/policies/" + args[0]); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}
	output.PrintSuccess("Policy deleted: %s", args[0])
}

func policiesCheck(ctx *Context, args []string) {
	fs := flag.NewFlagSet("policies check", flag.ExitOnError)
	userID := fs.String("user", "", "User ID (required)")
	action := fs.String("action", "", "Action to check (required)")
	resource := fs.String("resource", "", "Resource (required)")
	fs.Parse(args)

	if *userID == "" || *action == "" || *resource == "" {
		output.PrintError("--user, --action, and --resource are required")
		os.Exit(1)
	}

	c := requireClient(ctx)
	body := map[string]any{
		"user_id":  *userID,
		"action":   *action,
		"resource": *resource,
	}
	var result map[string]any
	if err := c.Post("/api/v1/policies/check", body, &result); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}

	allowed := false
	if a, ok := result["allowed"].(bool); ok {
		allowed = a
	}
	if allowed {
		output.PrintSuccess("ALLOWED: action %q on resource %q", *action, *resource)
	} else {
		fmt.Println("DENIED: action \"" + *action + "\" on resource \"" + *resource + "\"")
	}
}
