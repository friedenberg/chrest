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

	// The native host process is killed when the extension reloads.
	// Wait for Chrome to relaunch it and for the new socket to appear.
	maxRetries := 30
	for i := range maxRetries {
		fmt.Printf("waiting for server to come back up (%d/%d)\n", i+1, maxRetries)
		time.Sleep(500 * time.Millisecond)

		if err = cmdClient(c); err != nil {
			if errors.Is(err, io.ErrUnexpectedEOF) ||
				errors.IsErrno(err, syscall.ECONNREFUSED) ||
				errors.IsErrno(err, syscall.ENOENT) {
				err = nil
				continue
			}

			err = errors.Wrap(err)
			return
		}

		break
	}

	if err == nil && maxRetries > 0 {
		fmt.Println("extension reloaded successfully")
	}

	return
}
