package commands

import (
	"flag"
	"fmt"
	"os"

	"github.com/ggid/ggid/services/ggid-cli/internal/output"
)

// Organizations handles organization management subcommands.
func Organizations(ctx *Context, args []string) {
	if len(args) == 0 {
		orgsUsage()
		os.Exit(1)
	}
	sub := args[0]
	rest := args[1:]

	switch sub {
	case "list", "ls":
		orgsList(ctx, rest)
	case "get":
		orgsGet(ctx, rest)
	case "create":
		orgsCreate(ctx, rest)
	case "delete", "rm":
		orgsDelete(ctx, rest)
	case "tree":
		orgsTree(ctx, rest)
	case "members":
		orgsMembers(ctx, rest)
	case "help", "--help", "-h":
		orgsUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown orgs subcommand: %s\n\n", sub)
		orgsUsage()
		os.Exit(1)
	}
}

func orgsUsage() {
	fmt.Println(`USAGE: ggid orgs <subcommand> [flags]

SUBCOMMANDS:
  list [--page N] [--size N]          List organizations
  get <id>                             Get organization details
  create --name X [--parent ID]        Create an organization
  delete <id>                          Delete an organization
  tree                                 Show organization tree
  members <id>                         List members of an organization`)
}

func orgsList(ctx *Context, args []string) {
	fs := flag.NewFlagSet("orgs list", flag.ExitOnError)
	page := fs.Int("page", 1, "Page number")
	size := fs.Int("size", 20, "Page size")
	fs.Parse(args)

	c := requireClient(ctx)
	path := fmt.Sprintf("/api/v1/orgs?page=%d&page_size=%d", *page, *size)
	var result map[string]any
	if err := c.Get(path, &result); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}

	orgs := extractList(result, "organizations", "orgs", "items", "data")
	if isJSON(ctx) {
		output.PrintJSON(orgs)
		return
	}

	t := output.NewTable("ID", "NAME", "PARENT", "TYPE", "MEMBERS")
	for _, o := range orgs {
		t.AddRow(
			getStr(o, "id"),
			output.Truncate(getStr(o, "name"), 25),
			getStr(o, "parent_id"),
			getStr(o, "type"),
			getStr(o, "member_count"),
		)
	}
	t.Print()
}

func orgsGet(ctx *Context, args []string) {
	if len(args) == 0 {
		output.PrintError("organization ID required: ggid orgs get <id>")
		os.Exit(1)
	}
	c := requireClient(ctx)
	var result map[string]any
	if err := c.Get("/api/v1/orgs/"+args[0], &result); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}
	output.PrintJSON(result)
}

func orgsCreate(ctx *Context, args []string) {
	fs := flag.NewFlagSet("orgs create", flag.ExitOnError)
	name := fs.String("name", "", "Organization name (required)")
	parentID := fs.String("parent", "", "Parent organization ID")
	orgType := fs.String("type", "", "Organization type (department, team, etc.)")
	description := fs.String("description", "", "Description")
	fs.Parse(args)

	if *name == "" {
		output.PrintError("--name is required")
		os.Exit(1)
	}

	c := requireClient(ctx)
	body := map[string]any{
		"name": *name,
	}
	if *parentID != "" {
		body["parent_id"] = *parentID
	}
	if *orgType != "" {
		body["type"] = *orgType
	}
	if *description != "" {
		body["description"] = *description
	}

	var result map[string]any
	if err := c.Post("/api/v1/orgs", body, &result); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}
	output.PrintSuccess("Organization created: %s", getStr(result, "id"))
}

func orgsDelete(ctx *Context, args []string) {
	if len(args) == 0 {
		output.PrintError("organization ID required: ggid orgs delete <id>")
		os.Exit(1)
	}
	c := requireClient(ctx)
	if err := c.Delete("/api/v1/orgs/" + args[0]); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}
	output.PrintSuccess("Organization deleted: %s", args[0])
}

func orgsTree(ctx *Context, args []string) {
	c := requireClient(ctx)
	var result map[string]any
	if err := c.Get("/api/v1/orgs/tree", &result); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}
	output.PrintJSON(result)
}

func orgsMembers(ctx *Context, args []string) {
	if len(args) == 0 {
		output.PrintError("organization ID required: ggid orgs members <id>")
		os.Exit(1)
	}
	c := requireClient(ctx)
	var result map[string]any
	if err := c.Get("/api/v1/orgs/"+args[0]+"/members", &result); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}
	members := extractList(result, "members", "items", "data")
	if isJSON(ctx) {
		output.PrintJSON(members)
		return
	}
	t := output.NewTable("USER ID", "USERNAME", "EMAIL", "ROLE")
	for _, m := range members {
		t.AddRow(
			getStr(m, "user_id"),
			getStr(m, "username"),
			getStr(m, "email"),
			getStr(m, "role"),
		)
	}
	t.Print()
}
