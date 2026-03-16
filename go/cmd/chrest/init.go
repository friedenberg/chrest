package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"code.linenisgreat.com/chrest/go/src/alfa/symlink"
	"code.linenisgreat.com/chrest/go/src/bravo/config"
	"code.linenisgreat.com/chrest/go/src/charlie/install"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	tap "github.com/amarbel-llc/bob/packages/tap-dancer/go"
	"github.com/amarbel-llc/purse-first/libs/go-mcp/command"
)

func registerInitCommand(app *command.App) {
	app.AddCommand(&command.Command{
		Name: "init",
		Description: command.Description{
			Short: "Initialize configuration and install native messaging host",
		},
		Params: []command.Param{
			{Name: "browser", Type: command.String, Description: "Default browser (chrome or firefox)"},
			{Name: "extension-id", Type: command.String, Description: "Custom extension ID (uses default if omitted)"},
		},
		Run: func(ctx context.Context, args json.RawMessage, p command.Prompter) (*command.Result, error) {
			return nil, cmdInit(ctx, args, p)
		},
	})
}

func cmdInit(ctx context.Context, args json.RawMessage, p command.Prompter) (err error) {
	var params struct {
		Browser     string `json:"browser"`
		ExtensionId string `json:"extension-id"`
	}

	if err = json.Unmarshal(args, &params); err != nil {
		err = errors.Wrap(err)
		return
	}

	if params.Browser == "" {
		var idx int
		options := []string{"chrome", "firefox"}

		if idx, err = p.Select("Default browser", options); err != nil {
			err = errors.Errorf("--browser is required when not interactive")
			return
		}

		params.Browser = options[idx]
	}

	w := tap.NewColorWriter(os.Stderr, true)
	defer w.Plan()

	var initConfig config.Config

	if initConfig, err = config.Default(); err != nil {
		w.NotOk("Read default config", map[string]string{"error": err.Error()})
		err = errors.Wrap(err)
		return
	}

	if err = initConfig.DefaultBrowser.Set(params.Browser); err != nil {
		w.NotOk(
			fmt.Sprintf("Set default browser to %s", params.Browser),
			map[string]string{"error": err.Error()},
		)
		err = errors.Wrap(err)
		return
	}

	errCtx := errors.MakeContext(ctx)

	if err = initConfig.Write(errCtx); err != nil {
		w.NotOk("Write config", map[string]string{"error": err.Error()})
		err = errors.Wrap(err)
		return
	}

	w.Ok(fmt.Sprintf("Wrote config to %s", initConfig.Directory()))

	var exe string

	if exe, err = os.Executable(); err != nil {
		w.NotOk("Find executable path", map[string]string{"error": err.Error()})
		err = errors.Wrap(err)
		return
	}

	serverBinary := filepath.Join(filepath.Dir(exe), "chrest-server")

	if _, err = os.Stat(serverBinary); err != nil {
		w.NotOk(
			fmt.Sprintf("Find chrest-server binary at %s", serverBinary),
			map[string]string{"error": err.Error()},
		)
		err = errors.Errorf("chrest-server not found adjacent to chrest at %s", serverBinary)
		return
	}

	serverPath := initConfig.ServerPath()

	if err = symlink.Symlink(serverBinary, serverPath); err != nil {
		w.NotOk(
			fmt.Sprintf("Symlink chrest-server to %s", serverPath),
			map[string]string{"error": err.Error()},
		)
		err = errors.Wrap(err)
		return
	}

	w.Ok(fmt.Sprintf("Symlinked chrest-server to %s", serverPath))

	var idSet install.IdSet

	if err = idSet.Browser.Set(params.Browser); err != nil {
		err = errors.Wrap(err)
		return
	}

	if params.ExtensionId != "" {
		if err = idSet.Set(params.ExtensionId); err != nil {
			err = errors.Wrap(err)
			return
		}
	}

	extensionId := params.ExtensionId
	if extensionId == "" {
		extensionId = idSet.GetDefaultId()
	}

	if _, _, err = idSet.Install(initConfig); err != nil {
		w.NotOk(
			fmt.Sprintf("Install native messaging host for %s", params.Browser),
			map[string]string{"error": err.Error()},
		)
		err = errors.Wrap(err)
		return
	}

	w.Ok(fmt.Sprintf(
		"Installed native messaging host for %s (extension: %s)",
		params.Browser,
		extensionId,
	))

	return
}
