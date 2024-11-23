package main

import (
	"context"
	"os"
	"syscall"

	"code.linenisgreat.com/chrest/go/src/bravo/config"
	"code.linenisgreat.com/chrest/go/src/bravo/server"
	"code.linenisgreat.com/zit/go/zit/src/alfa/errors"
	"code.linenisgreat.com/zit/go/zit/src/bravo/ui"
)

func CmdServer(c config.Config) (err error) {
	defer ui.Err().Print("shut down complete")

	ui.Err().Printf("args: %q", os.Args)

	if err = c.Read(); err != nil {
		err = errors.Wrap(err)
		return
	}

	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)

	srv := server.Server{
		Cancel: cancel,
	}

	if err = srv.Initialize(ctx); err != nil {
		err = errors.Wrap(err)
		return
	}

	errors.MakeSignalWatchChannelAndCancelContextIfNecessary(
		cancel,
		syscall.SIGTERM,
	)

	if err = srv.Serve(ctx); err != nil {
		err = errors.Wrap(err)
		return
	}

	if err := context.Cause(ctx); err != nil {
		var normalError errors.StackTracer

		if errors.As(err, &normalError) && !normalError.ShouldShowStackTrace() {
			ui.Err().Printf("%s", normalError.Error())
		} else {
			if err != nil {
				ui.Err().Print(err)
			}
		}
	}

	return
}
