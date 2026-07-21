package commands

import (
	"flag"
	"fmt"
	"os"

	"github.com/ggid/ggid/services/ggid-cli/internal/output"
)

// Sessions handles session management subcommands.
func Sessions(ctx *Context, args []string) {
	if len(args) == 0 {
		sessionsUsage()
		os.Exit(1)
	}
	sub := args[0]
	rest := args[1:]

	switch sub {
	case "list", "ls":
		sessionsList(ctx, rest)
	case "revoke":
		sessionsRevoke(ctx, rest)
	case "help", "--help", "-h":
		sessionsUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown sessions subcommand: %s\n\n", sub)
		sessionsUsage()
		os.Exit(1)
	}
}

func sessionsUsage() {
	fmt.Println(`USAGE: ggid sessions <subcommand> [flags]

SUBCOMMANDS:
  list [--page N] [--size N]     List active sessions
  revoke <id>                    Revoke a session`)
}

func sessionsList(ctx *Context, args []string) {
	fs := flag.NewFlagSet("sessions list", flag.ExitOnError)
	page := fs.Int("page", 1, "Page number")
	size := fs.Int("size", 20, "Page size")
	fs.Parse(args)

	c := requireClient(ctx)
	path := fmt.Sprintf("/api/v1/sessions?page=%d&page_size=%d", *page, *size)
	var result map[string]any
	if err := c.Get(path, &result); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}

	sessions := extractList(result, "sessions", "items", "data")
	if isJSON(ctx) {
		output.PrintJSON(sessions)
		return
	}

	t := output.NewTable("SESSION ID", "USER ID", "IP", "USER AGENT", "CREATED", "EXPIRES")
	for _, s := range sessions {
		t.AddRow(
			output.Truncate(getStr(s, "id"), 20),
			output.Truncate(getStr(s, "user_id"), 15),
			getStr(s, "ip_address"),
			output.Truncate(getStr(s, "user_agent"), 20),
			getStr(s, "created_at"),
			getStr(s, "expires_at"),
		)
	}
	t.Print()
}

func sessionsRevoke(ctx *Context, args []string) {
	if len(args) == 0 {
		output.PrintError("session ID required: ggid sessions revoke <id>")
		os.Exit(1)
	}
	c := requireClient(ctx)
	if err := c.Delete("/api/v1/sessions/" + args[0]); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}
	output.PrintSuccess("Session revoked: %s", args[0])
}
