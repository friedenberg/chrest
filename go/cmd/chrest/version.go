package main

import (
	"context"
	"encoding/json"
	"fmt"

	"code.linenisgreat.com/chrest/go/libs/dewey/golf/command"
)

func registerVersionCommand(app *command.Utility) {
	app.AddCommand(&command.Command{
		Name: "version",
		Description: command.Description{
			Short: "Print build identity (version+commit)",
		},
		RunCLI: func(ctx context.Context, args json.RawMessage) error {
			fmt.Printf("%s+%s\n", version, commit)
			return nil
		},
	})
}
