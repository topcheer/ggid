package commands

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ggid/ggid/services/ggid-cli/internal/output"
)

// Audit handles audit log subcommands.
func Audit(ctx *Context, args []string) {
	if len(args) == 0 {
		auditUsage()
		os.Exit(1)
	}
	sub := args[0]
	rest := args[1:]

	switch sub {
	case "events", "list", "ls":
		auditEvents(ctx, rest)
	case "dashboard":
		auditDashboard(ctx, rest)
	case "help", "--help", "-h":
		auditUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown audit subcommand: %s\n\n", sub)
		auditUsage()
		os.Exit(1)
	}
}

func auditUsage() {
	fmt.Println(`USAGE: ggid audit <subcommand> [flags]

SUBCOMMANDS:
  events [--page N] [--size N] [--type T] [--status S]   Query audit events
  dashboard                                                Show audit dashboard`)
}

func auditEvents(ctx *Context, args []string) {
	fs := flag.NewFlagSet("audit events", flag.ExitOnError)
	page := fs.Int("page", 1, "Page number")
	size := fs.Int("size", 20, "Page size")
	eventType := fs.String("type", "", "Filter by event type")
	status := fs.String("status", "", "Filter by status")
	startDate := fs.String("from", "", "Start date (RFC3339)")
	endDate := fs.String("to", "", "End date (RFC3339)")
	fs.Parse(args)

	c := requireClient(ctx)

	path := fmt.Sprintf("/api/v1/audit/events?page=%d&page_size=%d", *page, *size)
	if *eventType != "" {
		path += "&event_type=" + *eventType
	}
	if *status != "" {
		path += "&status=" + *status
	}
	if *startDate != "" {
		path += "&start_date=" + *startDate
	}
	if *endDate != "" {
		path += "&end_date=" + *endDate
	}

	var result map[string]any
	if err := c.Get(path, &result); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}

	events := extractList(result, "events", "items", "data")
	if isJSON(ctx) {
		output.PrintJSON(events)
		return
	}

	t := output.NewTable("TIME", "TYPE", "STATUS", "ACTOR ID", "RESOURCE")
	for _, e := range events {
		ts := ""
		if v, ok := e["timestamp"].(string); ok {
			if parsed, err := time.Parse(time.RFC3339, v); err == nil {
				ts = parsed.Format("2006-01-02 15:04:05")
			} else {
				ts = output.Truncate(v, 19)
			}
		}
		if ts == "" {
			ts = getStr(e, "created_at")
		}
		t.AddRow(
			ts,
			output.Truncate(getStr(e, "event_type"), 15),
			output.Truncate(getStr(e, "status"), 8),
			output.Truncate(getStr(e, "actor_id"), 15),
			output.Truncate(getStr(e, "resource_type"), 15),
		)
	}
	t.Print()

	if total, ok := result["total_count"].(float64); ok {
		fmt.Printf("\nTotal: %d\n", int(total))
	}
}

func auditDashboard(ctx *Context, args []string) {
	c := requireClient(ctx)
	var result map[string]any
	if err := c.Get("/api/v1/audit/dashboard", &result); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}
	output.PrintJSON(result)
}
