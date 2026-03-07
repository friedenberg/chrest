package main

import (
	"context"
	"encoding/json"
	"os"

	"github.com/amarbel-llc/purse-first/libs/go-mcp/command"
)

func registerGeneratePluginCommand(app *command.App) {
	app.AddCommand(&command.Command{
		Name:   "generate-plugin",
		Hidden: true,
		Description: command.Description{
			Short: "Generate purse-first plugin artifacts",
		},
		RunCLI: func(ctx context.Context, args json.RawMessage) error {
			return app.HandleGeneratePlugin(os.Args[2:], os.Stdout)
		},
	})
}
