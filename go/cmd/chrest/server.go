package main

import (
	"os"
	"syscall"

	"code.linenisgreat.com/chrest/go/src/bravo/config"
	"code.linenisgreat.com/chrest/go/src/bravo/server"
	"code.linenisgreat.com/dodder/go/src/_/stack_frame"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/bravo/ui"
)

func CmdServer(c config.Config) (err error) {
	defer ui.Err().Print("shut down complete")

	ui.Err().Printf("args: %q", os.Args)

	if err = c.Read(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	ctx := errors.MakeContextDefault()
	ctx.SetCancelOnSignals(syscall.SIGTERM)

	if err := ctx.Run(
		func(ctx errors.Context) {
			srv := server.Server{
				ActiveContext: ctx,
			}

			srv.Initialize()
			srv.Serve()
		},
	); err != nil {
		var normalError stack_frame.ErrorStackTracer

		if errors.As(err, &normalError) && !normalError.ShouldShowStackTrace() {
			ui.Err().Printf("%s", normalError.Error())
		} else {
			if err != nil {
				ui.Err().Print(err)
			}
		}
	}

	return err
}
