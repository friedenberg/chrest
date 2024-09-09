package main

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"

	"code.linenisgreat.com/chrest/go/chrest/src/alfa/browser"
	"code.linenisgreat.com/chrest/go/chrest/src/alfa/symlink"
	"code.linenisgreat.com/chrest/go/chrest/src/bravo/config"
	"code.linenisgreat.com/chrest/go/chrest/src/bravo/install"
	"code.linenisgreat.com/zit/go/zit/src/alfa/errors"
)

func CmdInit() (err error) {
	var (
		initConfig config.Config
		idsChrome  install.IdSet
		idsFirefox install.IdSet
	)

	flag.Var(
		&initConfig.DefaultBrowser,
		"browser",
		"the browser to use by default",
	)

	idsChrome.Browser = browser.Chrome
	idsFirefox.Browser = browser.Firefox

	flag.Var(
		&idsChrome,
		"chrome",
		"the chrome IDs to install for",
	)

	flag.Var(
		&idsFirefox,
		"firefox",
		"the Firefox IDs to install for",
	)

	if initConfig, err = config.Default(); err != nil {
		err = errors.Wrap(err)
		return
	}

	flag.Parse()

	if err = initConfig.Write(); err != nil {
		err = errors.Wrap(err)
		return
	}

	var exe string
	if exe, err = os.Executable(); err != nil {
		err = errors.Wrap(err)
		return
	}

	err = nil

	newPath := initConfig.ServerPath()

	if err = symlink.Symlink(exe, newPath); err != nil {
		err = errors.Wrap(err)
		return
	}

	var ijs map[string]any

	if ijs, err = install.MakeJSON(
		initConfig,
		idsChrome,
		idsFirefox,
	); err != nil {
		err = errors.Wrap(err)
		return
	}

	for loc, ij := range ijs {
		var b []byte

		if b, err = json.Marshal(ij); err != nil {
			err = errors.Wrap(err)
			return
		}

		if err = os.MkdirAll(filepath.Dir(loc), 0o700); err != nil {
			err = errors.Wrap(err)
			return
		}

		if err = os.WriteFile(loc, b, 0o666); err != nil {
			err = errors.Wrap(err)
			return
		}
	}

	return
}
