package main

import (
	"code.linenisgreat.com/chrest/go/chrest/src/bravo/config"
	"code.linenisgreat.com/zit/go/zit/src/alfa/errors"
)

func CmdInit() (err error) {
	var c config.Config

	if c, err = config.ConfigDefault(); err != nil {
		err = errors.Wrap(err)
		return
	}

	if err = c.Write(); err != nil {
		err = errors.Wrap(err)
		return
	}

	return CmdInstall(c)
}
