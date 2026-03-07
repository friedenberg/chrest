package main

import (
	"context"
	"encoding/json"
	"os"

	"code.linenisgreat.com/chrest/go/src/alfa/browser"
	"code.linenisgreat.com/chrest/go/src/alfa/symlink"
	"code.linenisgreat.com/chrest/go/src/bravo/config"
	"code.linenisgreat.com/chrest/go/src/charlie/install"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"github.com/amarbel-llc/purse-first/libs/go-mcp/command"
)

func registerInitCommand(app *command.App) {
	app.AddCommand(&command.Command{
		Name:   "init",
		Hidden: true,
		Description: command.Description{
			Short: "Initialize configuration and install native messaging host",
		},
		Params: []command.Param{
			{Name: "browser", Type: command.String, Description: "The browser to use by default"},
			{Name: "chrome", Type: command.String, Description: "The chrome IDs to install for"},
			{Name: "firefox", Type: command.String, Description: "The Firefox IDs to install for"},
		},
		RunCLI: func(ctx context.Context, args json.RawMessage) error {
			return cmdInit(ctx, args)
		},
	})
}

func cmdInit(ctx context.Context, args json.RawMessage) (err error) {
	var params struct {
		Browser string `json:"browser"`
		Chrome  string `json:"chrome"`
		Firefox string `json:"firefox"`
	}

	if err = json.Unmarshal(args, &params); err != nil {
		err = errors.Wrap(err)
		return
	}

	var initConfig config.Config

	if initConfig, err = config.Default(); err != nil {
		err = errors.Wrap(err)
		return
	}

	if params.Browser != "" {
		if err = initConfig.DefaultBrowser.Set(params.Browser); err != nil {
			err = errors.Wrap(err)
			return
		}
	}

	var idsChrome install.IdSet
	var idsFirefox install.IdSet

	idsChrome.Browser = browser.Chrome
	idsFirefox.Browser = browser.Firefox

	if params.Chrome != "" {
		if err = idsChrome.Set(params.Chrome); err != nil {
			err = errors.Wrap(err)
			return
		}
	}

	if params.Firefox != "" {
		if err = idsFirefox.Set(params.Firefox); err != nil {
			err = errors.Wrap(err)
			return
		}
	}

	errCtx := errors.MakeContext(ctx)

	// TODO do not overwrite config if it exists
	if err = initConfig.Write(errCtx); err != nil {
		err = errors.Wrap(err)
		return
	}

	var exe string

	if exe, err = os.Executable(); err != nil {
		err = errors.Wrap(err)
		return
	}

	newPath := initConfig.ServerPath()

	if err = symlink.Symlink(exe, newPath); err != nil {
		err = errors.Wrap(err)
		return
	}

	for _, idSet := range []install.IdSet{idsChrome, idsFirefox} {
		if _, _, err = idSet.Install(
			initConfig,
		); err != nil {
			err = errors.Wrap(err)
			return
		}
	}

	return
}
