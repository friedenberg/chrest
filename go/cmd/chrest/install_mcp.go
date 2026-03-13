package main

import (
	"context"
	"encoding/json"

	"github.com/amarbel-llc/purse-first/libs/go-mcp/command"
)

func registerInstallMCPCommand(app *command.App) {
	app.AddCommand(&command.Command{
		Name: "install-mcp",
		Description: command.Description{
			Short: "Install chrest as a Claude Code MCP server",
		},
		RunCLI: func(ctx context.Context, args json.RawMessage) error {
			return app.InstallMCP()
		},
	})
}
