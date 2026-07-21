package commands

import (
	"fmt"
	"os"
)

// Completion generates shell completion scripts.
func Completion(ctx *Context, args []string) {
	if len(args) == 0 {
		fmt.Println(`USAGE: ggid completion <shell>

Generate shell completion script.

SHELLS:
  bash    Bash completion
  zsh     Zsh completion
  fish    Fish completion

INSTALL:
  # Bash (add to ~/.bashrc):
  eval "$(ggid completion bash)"

  # Zsh (add to ~/.zshrc):
  source <(ggid completion zsh)

  # Fish:
  ggid completion fish > ~/.config/fish/completions/ggid.fish`)
		return
	}

	shell := args[0]
	switch shell {
	case "bash":
		printBashCompletion()
	case "zsh":
		printZshCompletion()
	case "fish":
		printFishCompletion()
	default:
		fmt.Fprintf(os.Stderr, "unsupported shell: %s (use: bash, zsh, fish)\n", shell)
		os.Exit(1)
	}
}

func getCommands() []string {
	return []string{
		"login", "logout", "whoami", "version",
		"users", "roles", "orgs", "organizations",
		"audit", "policies", "policy",
		"oauth", "tenants", "tenant",
		"sessions", "session",
		"system", "webhooks", "webhook",
		"dashboard", "api-keys", "apikeys",
		"settings", "setting",
		"security", "monitoring",
		"completion", "help",
	}
}

func getSubCommands() map[string][]string {
	return map[string][]string{
		"users":       {"list", "get", "create", "update", "delete", "lock", "unlock"},
		"roles":       {"list", "get", "create", "delete", "assign", "revoke"},
		"orgs":        {"list", "get", "create", "delete", "tree", "members"},
		"audit":       {"events", "dashboard"},
		"policies":    {"list", "get", "create", "delete", "check"},
		"oauth":       {"clients"},
		"tenants":     {"list", "get", "create", "delete", "resolve", "suspend", "activate"},
		"sessions":    {"list", "revoke"},
		"system":      {"health", "status", "bootstrap", "initialized", "routes"},
		"webhooks":    {"list", "create", "delete", "test", "catalog"},
		"api-keys":    {"list", "create", "delete"},
		"settings":    {"get", "branding", "feature-flags", "password-policy", "mfa", "scim", "ldap"},
		"security":    {"sessions", "cae", "threats", "posture", "risk-score"},
		"monitoring":  {"gateway", "routes", "rate-limits", "analytics", "activity"},
	}
}

func printBashCompletion() {
	cmds := getCommands()
	subs := getSubCommands()

	fmt.Println(`# bash completion for ggid-cli
_ggid() {
    local cur prev words cword
    _init_completion || return

    local commands="` + joinStrings(cmds, " ") + `"

    if [ $cword -eq 1 ]; then
        COMPREPLY=( $(compgen -W "$commands" -- "$cur") )
        return
    fi

    case ${words[1]} in`)

	for cmd, subCmds := range subs {
		fmt.Printf(`
        %s)
            COMPREPLY=( $(compgen -W "%s" -- "$cur") )
            return;;`, cmd, joinStrings(subCmds, " "))
	}

	fmt.Println(`
    esac

    # File completion for certain flags
    case "$cur" in
        --*) COMPREPLY=( $(compgen -W "--json --table --help --server" -- "$cur") ); return;;
    esac
}
complete -F _ggid ggid`)
}

func printZshCompletion() {
	cmds := getCommands()
	subs := getSubCommands()

	fmt.Print(`#compdef ggid
# zsh completion for ggid-cli

_ggid() {
    local commands=(
`)
	for _, c := range cmds {
		fmt.Printf("        '%s'\n", c)
	}
	fmt.Println(`    )

    if [ $CURRENT -eq 2 ]; then
        _describe 'command' commands
        return
    fi

    local cmd=${words[2]}
    case $cmd in`)

	for cmd, subCmds := range subs {
		fmt.Printf(`
        (%s)
            local -a subcmds=(\n`, cmd)
		for _, sc := range subCmds {
			fmt.Printf("                '%s'\n", sc)
		}
		fmt.Println("            )")
		fmt.Println("            _describe 'subcommand' subcmds")
		fmt.Println("            ;;")
	}

	fmt.Println(`    esac
}

_ggid "$@"`)
}

func printFishCompletion() {
	cmds := getCommands()
	subs := getSubCommands()

	for _, c := range cmds {
		fmt.Printf("complete -c ggid -n '__fish_use_subcommand' -a '%s'\n", c)
	}

	for cmd, subCmds := range subs {
		for _, sc := range subCmds {
			fmt.Printf("complete -c ggid -n '__fish_seen_subcommand_from %s' -a '%s'\n", cmd, sc)
		}
	}
}

func joinStrings(strs []string, sep string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}
