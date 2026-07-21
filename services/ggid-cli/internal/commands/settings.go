package commands

import (
	"fmt"
	"os"

	"github.com/ggid/ggid/services/ggid-cli/internal/output"
)

// Settings handles settings management subcommands.
func Settings(ctx *Context, args []string) {
	if len(args) == 0 {
		settingsUsage()
		os.Exit(1)
	}
	sub := args[0]
	rest := args[1:]

	switch sub {
	case "get", "show":
		settingsGet(ctx, rest)
	case "branding":
		settingsBranding(ctx, rest)
	case "feature-flags", "flags":
		settingsFeatureFlags(ctx, rest)
	case "password-policy":
		settingsPasswordPolicy(ctx, rest)
	case "mfa":
		settingsMFA(ctx, rest)
	case "scim":
		settingsSCIM(ctx, rest)
	case "ldap":
		settingsLDAP(ctx, rest)
	case "help", "--help", "-h":
		settingsUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown settings subcommand: %s\n\n", sub)
		settingsUsage()
		os.Exit(1)
	}
}

func settingsUsage() {
	fmt.Println(`USAGE: ggid settings <subcommand> [flags]

SUBCOMMANDS:
  get                 Show all settings
  branding            View/update branding settings
  feature-flags       View/update feature flags
  password-policy     View/update password policy
  mfa                 View/update MFA settings
  scim                View SCIM provisioning settings
  ldap                View LDAP configuration`)
}

func settingsGet(ctx *Context, args []string) {
	c := requireClient(ctx)
	var result map[string]any
	if err := c.Get("/api/v1/settings", &result); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}
	output.PrintJSON(result)
}

func settingsBranding(ctx *Context, args []string) {
	c := requireClient(ctx)
	var result map[string]any
	if err := c.Get("/api/v1/settings/branding", &result); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}
	output.PrintJSON(result)
}

func settingsFeatureFlags(ctx *Context, args []string) {
	c := requireClient(ctx)
	var result map[string]any
	if err := c.Get("/api/v1/settings/feature-flags", &result); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}

	flags := extractList(result, "flags", "feature_flags", "items", "data")
	if isJSON(ctx) || flags == nil {
		output.PrintJSON(result)
		return
	}

	t := output.NewTable("FLAG", "ENABLED", "DESCRIPTION")
	for _, f := range flags {
		t.AddRow(
			output.Truncate(getStr(f, "key"), 30),
			getStr(f, "enabled"),
			output.Truncate(getStr(f, "description"), 40),
		)
	}
	t.Print()
}

func settingsPasswordPolicy(ctx *Context, args []string) {
	c := requireClient(ctx)
	var result map[string]any
	if err := c.Get("/api/v1/settings/password-policy", &result); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}
	output.PrintJSON(result)
}

func settingsMFA(ctx *Context, args []string) {
	c := requireClient(ctx)
	var result map[string]any
	if err := c.Get("/api/v1/settings/mfa", &result); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}
	output.PrintJSON(result)
}

func settingsSCIM(ctx *Context, args []string) {
	c := requireClient(ctx)
	var result map[string]any
	if err := c.Get("/api/v1/settings/scim", &result); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}
	output.PrintJSON(result)
}

func settingsLDAP(ctx *Context, args []string) {
	c := requireClient(ctx)
	var result map[string]any
	if err := c.Get("/api/v1/settings/ldap-config", &result); err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}
	output.PrintJSON(result)
}
