package main

import (
	"os"

	"code.linenisgreat.com/chrest/go/src/bravo/config"
	"code.linenisgreat.com/chrest/go/src/bravo/server"
	"code.linenisgreat.com/zit/go/zit/src/alfa/errors"
	"code.linenisgreat.com/zit/go/zit/src/bravo/quiter"
	"code.linenisgreat.com/zit/go/zit/src/bravo/ui"
)

func CmdServer(c config.Config) (err error) {
	ui.Err().Printf("args: %q", os.Args)

	if err = c.Read(); err != nil {
		err = errors.Wrap(err)
		return
	}

	srv := server.Server{}

	wg := quiter.MakeErrorWaitGroupParallel()

	wg.Do(srv.Serve)

	if err = wg.GetError(); err != nil {
		err = errors.Wrap(err)
		return
	}

	return
}
