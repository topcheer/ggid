package commands

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/ggid/ggid/services/ggid-cli/internal/output"
)

// OAuthClients handles OAuth client management subcommands.
func OAuthClients(ctx *Context, args []string) {
	if len(args) == 0 {
		oauthUsage()
		os.Exit(1)
	}

	// Support "oauth clients ..." prefix.
	if args[0] == "clients" {
		args = args[1:]
	}
	if len(args) == 0 {
		oauthUsage()
		os.Exit(1)
	}

	sub := args[0]
	rest := args[1:]

	switch sub {
	case "list", "ls":
		oauthClientsList(ctx, rest)
	case "get":
		oauthClientsGet(ctx, rest)
	case "create":
		oauthClientsCreate(ctx, rest)
	case "delete", "rm":
		oauthClientsDelete(ctx, rest)
	case "rotate-secret":
		oauthClientsRotateSecret(ctx, rest)
	case "help", "--help", "-h":
		oauthUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown oauth subcommand: %s\n\n", sub)
		oauthUsage()
		os.Exit(1)
	}
}

func oauthUsage() {
	fmt.Println(`USAGE: ggid oauth clients <subcommand> [flags]

SUBCOMMANDS:
  list                                List OAuth clients
  get <id>                            Get client details
  create --name X [...]               Create an OAuth client
  delete <id>                         Delete an OAuth client
  rotate-secret <id>                  Rotate client secret`)
}

func oauthClientsList(ctx *Context, args []string) {
	c := requireClient(ctx)
	var result map[string]any
	if err := c.Get("/api/v1/oauth/clients", &result); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}

	clients := extractList(result, "clients", "items", "data")
	if isJSON(ctx) {
		output.PrintJSON(clients)
		return
	}

	t := output.NewTable("CLIENT ID", "NAME", "TYPE", "GRANT TYPES", "ENABLED")
	for _, cl := range clients {
		grants := ""
		if g, ok := cl["grant_types"].([]any); ok {
			strs := make([]string, 0, len(g))
			for _, x := range g {
				if s, ok := x.(string); ok {
					strs = append(strs, s)
				}
			}
			grants = strings.Join(strs, ", ")
		}
		t.AddRow(
			getStr(cl, "client_id"),
			output.Truncate(getStr(cl, "name"), 20),
			getStr(cl, "type"),
			output.Truncate(grants, 30),
			getStr(cl, "enabled"),
		)
	}
	t.Print()
}

func oauthClientsGet(ctx *Context, args []string) {
	if len(args) == 0 {
		output.PrintError("client ID required: ggid oauth clients get <id>")
		os.Exit(1)
	}
	c := requireClient(ctx)
	var result map[string]any
	if err := c.Get("/api/v1/oauth/clients/"+args[0], &result); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}
	output.PrintJSON(result)
}

func oauthClientsCreate(ctx *Context, args []string) {
	fs := flag.NewFlagSet("oauth clients create", flag.ExitOnError)
	name := fs.String("name", "", "Client name (required)")
	clientType := fs.String("type", "confidential", "Client type: confidential or public")
	grantTypes := fs.String("grant-types", "client_credentials", "Comma-separated grant types")
	redirectURIs := fs.String("redirect-uris", "", "Comma-separated redirect URIs")
	scopes := fs.String("scopes", "openid profile email", "Space-separated scopes")
	fs.Parse(args)

	if *name == "" {
		output.PrintError("--name is required")
		os.Exit(1)
	}

	c := requireClient(ctx)
	body := map[string]any{
		"client_name": *name,
		"type":        *clientType,
		"grant_types": strings.Split(*grantTypes, ","),
	}
	if *redirectURIs != "" {
		body["redirect_uris"] = strings.Split(*redirectURIs, ",")
	}
	if *scopes != "" {
		body["scopes"] = strings.Split(*scopes, " ")
	}

	var result map[string]any
	if err := c.Post("/api/v1/oauth/clients", body, &result); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}

	if clientID := getStr(result, "client_id"); clientID != "" {
		// Check for nested client object.
		if clientObj, ok := result["client"].(map[string]any); ok {
			clientID = getStr(clientObj, "client_id")
			if clientID == "" {
				clientID = getStr(clientObj, "id")
			}
		}
		output.PrintSuccess("OAuth client created: %s", clientID)
	}
	output.PrintJSON(result)
}

func oauthClientsDelete(ctx *Context, args []string) {
	if len(args) == 0 {
		output.PrintError("client ID required: ggid oauth clients delete <id>")
		os.Exit(1)
	}
	c := requireClient(ctx)
	if err := c.Delete("/api/v1/oauth/clients/" + args[0]); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}
	output.PrintSuccess("OAuth client deleted: %s", args[0])
}

func oauthClientsRotateSecret(ctx *Context, args []string) {
	if len(args) == 0 {
		output.PrintError("client ID required: ggid oauth clients rotate-secret <id>")
		os.Exit(1)
	}
	c := requireClient(ctx)
	var result map[string]any
	if err := c.Post("/api/v1/oauth/clients/"+args[0]+"/rotate-secret", nil, &result); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}
	output.PrintSuccess("Secret rotated for client: %s", args[0])
	if newSecret := getStr(result, "client_secret"); newSecret != "" {
		fmt.Printf("New secret: %s\n", newSecret)
	}
}
