package commands

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/ggid/ggid/services/ggid-cli/internal/output"
)

// Webhooks handles webhook management subcommands.
func Webhooks(ctx *Context, args []string) {
	if len(args) == 0 {
		webhooksUsage()
		os.Exit(1)
	}
	sub := args[0]
	rest := args[1:]

	switch sub {
	case "list", "ls":
		webhooksList(ctx, rest)
	case "create":
		webhooksCreate(ctx, rest)
	case "delete", "rm":
		webhooksDelete(ctx, rest)
	case "test":
		webhooksTest(ctx, rest)
	case "catalog":
		webhooksCatalog(ctx, rest)
	case "help", "--help", "-h":
		webhooksUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown webhooks subcommand: %s\n\n", sub)
		webhooksUsage()
		os.Exit(1)
	}
}

func webhooksUsage() {
	fmt.Println(`USAGE: ggid webhooks <subcommand> [flags]

SUBCOMMANDS:
  list                               List webhooks
  create --url X --events a,b        Create a webhook
  delete <id>                        Delete a webhook
  test <id>                          Test fire a webhook
  catalog                            List available webhook event types`)
}

func webhooksList(ctx *Context, args []string) {
	c := requireClient(ctx)
	var result map[string]any
	if err := c.Get("/api/v1/webhooks", &result); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}

	webhooks := extractList(result, "webhooks", "items", "data")
	if isJSON(ctx) {
		output.PrintJSON(webhooks)
		return
	}

	t := output.NewTable("ID", "URL", "EVENTS", "ENABLED", "LAST DELIVERY")
	for _, w := range webhooks {
		events := ""
		if e, ok := w["events"].([]any); ok {
			strs := make([]string, 0, len(e))
			for _, x := range e {
				if s, ok := x.(string); ok {
					strs = append(strs, s)
				}
			}
			events = strings.Join(strs, ", ")
		}
		t.AddRow(
			output.Truncate(getStr(w, "id"), 20),
			output.Truncate(getStr(w, "url"), 35),
			output.Truncate(events, 25),
			getStr(w, "enabled"),
			getStr(w, "last_delivery_at"),
		)
	}
	t.Print()
}

func webhooksCreate(ctx *Context, args []string) {
	fs := flag.NewFlagSet("webhooks create", flag.ExitOnError)
	url := fs.String("url", "", "Webhook URL (required)")
	events := fs.String("events", "", "Comma-separated event types (required)")
	description := fs.String("description", "", "Description")
	fs.Parse(args)

	if *url == "" || *events == "" {
		output.PrintError("--url and --events are required")
		os.Exit(1)
	}

	c := requireClient(ctx)
	body := map[string]any{
		"url":    *url,
		"events": strings.Split(*events, ","),
	}
	if *description != "" {
		body["description"] = *description
	}

	var result map[string]any
	if err := c.Post("/api/v1/webhooks", body, &result); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}
	output.PrintSuccess("Webhook created: %s", getStr(result, "id"))
}

func webhooksDelete(ctx *Context, args []string) {
	if len(args) == 0 {
		output.PrintError("webhook ID required: ggid webhooks delete <id>")
		os.Exit(1)
	}
	c := requireClient(ctx)
	if err := c.Delete("/api/v1/webhooks/" + args[0]); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}
	output.PrintSuccess("Webhook deleted: %s", args[0])
}

func webhooksTest(ctx *Context, args []string) {
	if len(args) == 0 {
		output.PrintError("webhook ID required: ggid webhooks test <id>")
		os.Exit(1)
	}
	c := requireClient(ctx)
	var result map[string]any
	if err := c.Post("/api/v1/webhooks/"+args[0]+"/test", nil, &result); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}
	output.PrintSuccess("Webhook test sent")
	output.PrintJSON(result)
}

func webhooksCatalog(ctx *Context, args []string) {
	c := requireClient(ctx)
	var result map[string]any
	if err := c.Get("/api/v1/webhooks/events/catalog", &result); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}

	events := extractList(result, "events", "data")
	if isJSON(ctx) {
		output.PrintJSON(events)
		return
	}

	t := output.NewTable("EVENT TYPE", "DESCRIPTION", "PAYLOAD SCHEMA")
	for _, e := range events {
		t.AddRow(
			output.Truncate(getStr(e, "type"), 25),
			output.Truncate(getStr(e, "description"), 40),
			output.Truncate(getStr(e, "payload_schema"), 20),
		)
	}
	t.Print()
}
