package main

import (
	"context"
	"encoding/json"
	"os"

	"code.linenisgreat.com/chrest/go/libs/dewey/golf/command"
)

func registerGeneratePluginCommand(app *command.Utility) {
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
