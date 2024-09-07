package main

import (
	"flag"

	"code.linenisgreat.com/chrest/go/chrest/src/bravo/config"
	"code.linenisgreat.com/zit/go/zit/src/alfa/errors"
)

var InitConfig config.Config

func InitAddFlags() {
	flag.Var(
		&InitConfig.DefaultBrowser,
		"browser",
		"the browser to use by default",
	)
}

func CmdInit() (err error) {
	if InitConfig, err = config.Default(); err != nil {
		err = errors.Wrap(err)
		return
	}

	addFlagsOnce.Do(InitAddFlags)
	flag.Parse()

	if err = InitConfig.Write(); err != nil {
		err = errors.Wrap(err)
		return
	}

	if err = CmdInstall(InitConfig); err != nil {
		err = errors.Wrap(err)
		return
	}

	return
}
