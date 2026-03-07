package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"syscall"
	"time"

	"code.linenisgreat.com/chrest/go/src/bravo/config"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
	"github.com/amarbel-llc/purse-first/libs/go-mcp/command"
)

func registerReloadExtensionCommand(app *command.App, c config.Config) {
	app.AddCommand(&command.Command{
		Name:        "reload-extension",
		Description: command.Description{Short: "Reload the browser extension"},
		RunCLI: func(ctx context.Context, args json.RawMessage) error {
			return cmdReloadExtension(c)
		},
	})
}

func cmdReloadExtension(c config.Config) (err error) {
	var exe string

	if exe, err = os.Executable(); err != nil {
		err = errors.Wrap(err)
		return
	}

	os.Args = []string{exe, "client", "POST", "/runtime/reload"}

	fmt.Println("reloading server")

	if err = cmdClient(c); err != nil {
		if errors.Is(err, io.ErrUnexpectedEOF) {
			err = nil
		} else {
			err = errors.Wrap(err)
			return
		}
	}

	os.Args[2] = "GET"

	for {
		fmt.Println("waiting for server to come back up")

		if err = cmdClient(c); err != nil {
			if errors.Is(err, io.ErrUnexpectedEOF) {
				err = nil
				time.Sleep(1e8)
				continue
			} else if errors.IsErrno(err, syscall.ECONNREFUSED) {
				ui.Err().Print("Browser failed to restart extension. It will need to be restarted manually.")
				err = nil
				return
			} else {
				err = errors.Wrap(err)
				return
			}
		}

		break
	}

	return
}
