// Package commands implements all CLI subcommands.
package commands

import (
	"github.com/ggid/ggid/services/ggid-cli/internal/client"
	"github.com/ggid/ggid/services/ggid-cli/internal/config"
)

// Context carries shared state across all commands.
type Context struct {
	Config       *config.Config
	Client       *client.Client
	ServerURL    string
	OutputFormat string
	Version      string
}
