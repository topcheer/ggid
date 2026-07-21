package commands

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/ggid/ggid/services/ggid-cli/internal/output"
)

// APIKeys handles API key management subcommands.
func APIKeys(ctx *Context, args []string) {
	if len(args) == 0 {
		apiKeysUsage()
		os.Exit(1)
	}
	sub := args[0]
	rest := args[1:]

	switch sub {
	case "list", "ls":
		apiKeysList(ctx, rest)
	case "create":
		apiKeysCreate(ctx, rest)
	case "delete", "rm":
		apiKeysDelete(ctx, rest)
	case "revoke":
		apiKeysDelete(ctx, rest)
	case "help", "--help", "-h":
		apiKeysUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown api-keys subcommand: %s\n\n", sub)
		apiKeysUsage()
		os.Exit(1)
	}
}

func apiKeysUsage() {
	fmt.Println(`USAGE: ggid api-keys <subcommand> [flags]

SUBCOMMANDS:
  list                           List API keys
  create --name X [--scopes a,b]  Create an API key
  delete <id>                    Revoke/delete an API key`)
}

func apiKeysList(ctx *Context, args []string) {
	c := requireClient(ctx)
	var result map[string]any
	if err := c.Get("/api/v1/api-keys", &result); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}

	keys := extractList(result, "api_keys", "keys", "items", "data")
	if isJSON(ctx) {
		output.PrintJSON(keys)
		return
	}

	t := output.NewTable("ID", "NAME", "SCOPES", "STATUS", "CREATED", "EXPIRES")
	for _, k := range keys {
		scopes := ""
		if s, ok := k["scopes"].([]any); ok {
			strs := make([]string, 0, len(s))
			for _, x := range s {
				if str, ok := x.(string); ok {
					strs = append(strs, str)
				}
			}
			scopes = strings.Join(strs, ", ")
		}
		t.AddRow(
			output.Truncate(getStr(k, "id"), 20),
			output.Truncate(getStr(k, "name"), 20),
			output.Truncate(scopes, 25),
			getStr(k, "status"),
			getStr(k, "created_at"),
			getStr(k, "expires_at"),
		)
	}
	t.Print()
}

func apiKeysCreate(ctx *Context, args []string) {
	fs := flag.NewFlagSet("api-keys create", flag.ExitOnError)
	name := fs.String("name", "", "Key name (required)")
	scopes := fs.String("scopes", "", "Comma-separated scopes")
	expiresIn := fs.String("expires-in", "", "Expiry duration (e.g. 720h)")
	fs.Parse(args)

	if *name == "" {
		output.PrintError("--name is required")
		os.Exit(1)
	}

	c := requireClient(ctx)
	body := map[string]any{
		"name": *name,
	}
	if *scopes != "" {
		body["scopes"] = strings.Split(*scopes, ",")
	}
	if *expiresIn != "" {
		body["expires_in"] = *expiresIn
	}

	var result map[string]any
	if err := c.Post("/api/v1/api-keys", body, &result); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}
	output.PrintSuccess("API key created: %s", getStr(result, "id"))
	if key := getStr(result, "key"); key != "" {
		fmt.Printf("Key: %s\n", key)
		fmt.Println("WARNING: Save this key now - it will not be shown again.")
	}
}

func apiKeysDelete(ctx *Context, args []string) {
	if len(args) == 0 {
		output.PrintError("API key ID required: ggid api-keys delete <id>")
		os.Exit(1)
	}
	c := requireClient(ctx)
	if err := c.Delete("/api/v1/api-keys/" + args[0]); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}
	output.PrintSuccess("API key revoked: %s", args[0])
}
