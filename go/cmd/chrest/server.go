package main

import (
	"os"

	"code.linenisgreat.com/chrest/go/src/bravo/config"
	"code.linenisgreat.com/chrest/go/src/bravo/server"
	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/bravo/ui"
)

func CmdServer(ctx interfaces.ActiveContext, c config.Config) (err error) {
	defer ui.Err().Print("shut down complete")

	ui.Err().Printf("args: %q", os.Args)

	if err = c.Read(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	srv := server.Server{
		ActiveContext: ctx,
	}

	srv.Initialize()
	srv.Serve()

	return err
}
