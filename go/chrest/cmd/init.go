package main

import (
	"code.linenisgreat.com/chrest/go/chrest"
	"code.linenisgreat.com/zit/go/zit/src/alfa/errors"
)

func CmdInit() (err error) {
	var c chrest.Config

	if c, err = chrest.ConfigDefault(); err != nil {
		err = errors.Wrap(err)
		return
	}

	if err = c.Write(); err != nil {
		err = errors.Wrap(err)
		return
	}

	return CmdInstall(c)
}
