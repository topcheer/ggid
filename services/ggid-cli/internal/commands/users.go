package commands

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ggid/ggid/services/ggid-cli/internal/output"
)

// Users handles all user management subcommands.
func Users(ctx *Context, args []string) {
	if len(args) == 0 {
		usersUsage()
		os.Exit(1)
	}

	sub := args[0]
	rest := args[1:]

	switch sub {
	case "list", "ls":
		usersList(ctx, rest)
	case "get":
		usersGet(ctx, rest)
	case "create":
		usersCreate(ctx, rest)
	case "update":
		usersUpdate(ctx, rest)
	case "delete", "rm":
		usersDelete(ctx, rest)
	case "lock":
		usersLock(ctx, rest)
	case "unlock":
		usersUnlock(ctx, rest)
	case "help", "--help", "-h":
		usersUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown users subcommand: %s\n\n", sub)
		usersUsage()
		os.Exit(1)
	}
}

func usersUsage() {
	fmt.Println(`USAGE: ggid users <subcommand> [flags]

SUBCOMMANDS:
  list [--page N] [--size N] [--search STR]   List users
  get <id>                                     Get user details
  create --username X --email Y --password Z   Create a new user
  update <id> [--email Y] [--status S] [...]   Update a user
  delete <id>                                  Delete a user
  lock <id>                                    Lock a user account
  unlock <id>                                  Unlock a user account`)
}

func usersList(ctx *Context, args []string) {
	fs := flag.NewFlagSet("users list", flag.ExitOnError)
	page := fs.Int("page", 1, "Page number")
	size := fs.Int("size", 20, "Page size")
	search := fs.String("search", "", "Search query")
	status := fs.String("status", "", "Filter by status")
	fs.Parse(args)

	c := requireClient(ctx)

	path := fmt.Sprintf("/api/v1/users?page=%d&page_size=%d", *page, *size)
	if *search != "" {
		path += "&search=" + *search
	}
	if *status != "" {
		path += "&status=" + *status
	}

	var result map[string]any
	if err := c.Get(path, &result); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}

	// Extract users from response.
	users := extractList(result, "users", "items")

	if isJSON(ctx) {
		output.PrintJSON(users)
		return
	}

	t := output.NewTable("ID", "USERNAME", "EMAIL", "STATUS", "DISPLAY NAME", "CREATED")
	for _, u := range users {
		created := ""
		if v, ok := u["created_at"].(string); ok {
			if parsed, err := time.Parse(time.RFC3339, v); err == nil {
				created = parsed.Format("2006-01-02 15:04")
			} else {
				created = output.Truncate(v, 19)
			}
		}
		t.AddRow(
			getStr(u, "id"),
			getStr(u, "username"),
			getStr(u, "email"),
			getStr(u, "status"),
			output.Truncate(getStr(u, "display_name"), 20),
			created,
		)
	}
	t.Print()

	if total, ok := result["total_count"].(float64); ok {
		fmt.Printf("\nTotal: %d\n", int(total))
	} else if total, ok := result["total"].(float64); ok {
		fmt.Printf("\nTotal: %d\n", int(total))
	}
}

func usersGet(ctx *Context, args []string) {
	if len(args) == 0 {
		output.PrintError("user ID required: ggid users get <id>")
		os.Exit(1)
	}
	c := requireClient(ctx)
	var result map[string]any
	if err := c.Get("/api/v1/users/"+args[0], &result); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}
	output.PrintJSON(result)
}

func usersCreate(ctx *Context, args []string) {
	fs := flag.NewFlagSet("users create", flag.ExitOnError)
	username := fs.String("username", "", "Username (required)")
	email := fs.String("email", "", "Email (required)")
	password := fs.String("password", "", "Password (required)")
	displayName := fs.String("display-name", "", "Display name")
	phone := fs.String("phone", "", "Phone number")
	locale := fs.String("locale", "", "Locale (e.g. en-US)")
	timezone := fs.String("timezone", "", "Timezone")
	roleIDs := fs.String("roles", "", "Comma-separated role IDs")
	fs.Parse(args)

	if *username == "" || *email == "" || *password == "" {
		output.PrintError("--username, --email, and --password are required")
		os.Exit(1)
	}

	c := requireClient(ctx)

	body := map[string]any{
		"username": *username,
		"email":    *email,
		"password": *password,
	}
	if *displayName != "" {
		body["display_name"] = *displayName
	}
	if *phone != "" {
		body["phone"] = *phone
	}
	if *locale != "" {
		body["locale"] = *locale
	}
	if *timezone != "" {
		body["timezone"] = *timezone
	}
	if *roleIDs != "" {
		body["role_ids"] = strings.Split(*roleIDs, ",")
	}

	var result map[string]any
	if err := c.Post("/api/v1/users", body, &result); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}
	output.PrintSuccess("User created: %s", getStr(result, "id"))
	output.PrintJSON(result)
}

func usersUpdate(ctx *Context, args []string) {
	if len(args) == 0 {
		output.PrintError("user ID required: ggid users update <id> [flags]")
		os.Exit(1)
	}
	userID := args[0]
	fs := flag.NewFlagSet("users update", flag.ExitOnError)
	email := fs.String("email", "", "Email")
	displayName := fs.String("display-name", "", "Display name")
	phone := fs.String("phone", "", "Phone number")
	status := fs.String("status", "", "Status (active, inactive, locked)")
	locale := fs.String("locale", "", "Locale")
	timezone := fs.String("timezone", "", "Timezone")
	fs.Parse(args[1:])

	body := map[string]any{}
	if *email != "" {
		body["email"] = *email
	}
	if *displayName != "" {
		body["display_name"] = *displayName
	}
	if *phone != "" {
		body["phone"] = *phone
	}
	if *status != "" {
		body["status"] = *status
	}
	if *locale != "" {
		body["locale"] = *locale
	}
	if *timezone != "" {
		body["timezone"] = *timezone
	}
	if len(body) == 0 {
		output.PrintError("no fields specified to update")
		os.Exit(1)
	}

	c := requireClient(ctx)
	var result map[string]any
	if err := c.Put("/api/v1/users/"+userID, body, &result); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}
	output.PrintSuccess("User updated: %s", userID)
}

func usersDelete(ctx *Context, args []string) {
	if len(args) == 0 {
		output.PrintError("user ID required: ggid users delete <id>")
		os.Exit(1)
	}
	c := requireClient(ctx)
	if err := c.Delete("/api/v1/users/" + args[0]); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}
	output.PrintSuccess("User deleted: %s", args[0])
}

func usersLock(ctx *Context, args []string) {
	if len(args) == 0 {
		output.PrintError("user ID required: ggid users lock <id>")
		os.Exit(1)
	}
	c := requireClient(ctx)
	if err := c.Post("/api/v1/users/"+args[0]+"/lock", nil, nil); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}
	output.PrintSuccess("User locked: %s", args[0])
}

func usersUnlock(ctx *Context, args []string) {
	if len(args) == 0 {
		output.PrintError("user ID required: ggid users unlock <id>")
		os.Exit(1)
	}
	c := requireClient(ctx)
	if err := c.Post("/api/v1/users/"+args[0]+"/unlock", nil, nil); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}
	output.PrintSuccess("User unlocked: %s", args[0])
}
