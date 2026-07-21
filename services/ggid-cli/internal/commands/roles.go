package commands

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/ggid/ggid/services/ggid-cli/internal/output"
)

// Roles handles role management subcommands.
func Roles(ctx *Context, args []string) {
	if len(args) == 0 {
		rolesUsage()
		os.Exit(1)
	}
	sub := args[0]
	rest := args[1:]

	switch sub {
	case "list", "ls":
		rolesList(ctx, rest)
	case "get":
		rolesGet(ctx, rest)
	case "create":
		rolesCreate(ctx, rest)
	case "delete", "rm":
		rolesDelete(ctx, rest)
	case "assign":
		rolesAssign(ctx, rest)
	case "revoke":
		rolesRevoke(ctx, rest)
	case "help", "--help", "-h":
		rolesUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown roles subcommand: %s\n\n", sub)
		rolesUsage()
		os.Exit(1)
	}
}

func rolesUsage() {
	fmt.Println(`USAGE: ggid roles <subcommand> [flags]

SUBCOMMANDS:
  list                                    List roles
  get <id>                                Get role details
  create --name X [--description Y]       Create a role
  delete <id>                             Delete a role
  assign --user ID --role ID              Assign a role to a user
  revoke --user ID --role ID              Revoke a role from a user`)
}

func rolesList(ctx *Context, args []string) {
	c := requireClient(ctx)
	var result map[string]any
	if err := c.Get("/api/v1/roles", &result); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}

	roles := extractList(result, "roles", "items", "data")
	if isJSON(ctx) {
		output.PrintJSON(roles)
		return
	}

	t := output.NewTable("ID", "NAME", "DESCRIPTION", "PERMISSIONS")
	for _, r := range roles {
		perms := ""
		if p, ok := r["permissions"].([]any); ok {
			strs := make([]string, 0, len(p))
			for _, x := range p {
				if s, ok := x.(string); ok {
					strs = append(strs, s)
				}
			}
			perms = strings.Join(strs, ", ")
		}
		t.AddRow(
			getStr(r, "id"),
			output.Truncate(getStr(r, "name"), 20),
			output.Truncate(getStr(r, "description"), 30),
			output.Truncate(perms, 40),
		)
	}
	t.Print()
}

func rolesGet(ctx *Context, args []string) {
	if len(args) == 0 {
		output.PrintError("role ID required: ggid roles get <id>")
		os.Exit(1)
	}
	c := requireClient(ctx)
	var result map[string]any
	if err := c.Get("/api/v1/roles/"+args[0], &result); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}
	output.PrintJSON(result)
}

func rolesCreate(ctx *Context, args []string) {
	fs := flag.NewFlagSet("roles create", flag.ExitOnError)
	name := fs.String("name", "", "Role name (required)")
	description := fs.String("description", "", "Role description")
	permissions := fs.String("permissions", "", "Comma-separated permissions")
	fs.Parse(args)

	if *name == "" {
		output.PrintError("--name is required")
		os.Exit(1)
	}

	c := requireClient(ctx)
	body := map[string]any{
		"name": *name,
	}
	if *description != "" {
		body["description"] = *description
	}
	if *permissions != "" {
		body["permissions"] = strings.Split(*permissions, ",")
	}

	var result map[string]any
	if err := c.Post("/api/v1/roles", body, &result); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}
	output.PrintSuccess("Role created: %s", getStr(result, "id"))
}

func rolesDelete(ctx *Context, args []string) {
	if len(args) == 0 {
		output.PrintError("role ID required: ggid roles delete <id>")
		os.Exit(1)
	}
	c := requireClient(ctx)
	if err := c.Delete("/api/v1/roles/" + args[0]); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}
	output.PrintSuccess("Role deleted: %s", args[0])
}

func rolesAssign(ctx *Context, args []string) {
	fs := flag.NewFlagSet("roles assign", flag.ExitOnError)
	userID := fs.String("user", "", "User ID (required)")
	roleID := fs.String("role", "", "Role ID (required)")
	fs.Parse(args)

	if *userID == "" || *roleID == "" {
		output.PrintError("--user and --role are required")
		os.Exit(1)
	}

	c := requireClient(ctx)
	body := map[string]any{"role_id": *roleID}
	if err := c.Post("/api/v1/users/"+*userID+"/roles", body, nil); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}
	output.PrintSuccess("Role %s assigned to user %s", *roleID, *userID)
}

func rolesRevoke(ctx *Context, args []string) {
	fs := flag.NewFlagSet("roles revoke", flag.ExitOnError)
	userID := fs.String("user", "", "User ID (required)")
	roleID := fs.String("role", "", "Role ID (required)")
	fs.Parse(args)

	if *userID == "" || *roleID == "" {
		output.PrintError("--user and --role are required")
		os.Exit(1)
	}

	c := requireClient(ctx)
	if err := c.Delete(fmt.Sprintf("/api/v1/users/%s/roles/%s", *userID, *roleID)); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}
	output.PrintSuccess("Role %s revoked from user %s", *roleID, *userID)
}
