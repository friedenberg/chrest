package main

import (
	"fmt"
	"log"
	"os"
	"sync"
	"syscall"

	"code.linenisgreat.com/chrest/go/src/bravo/config"
	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/_/stack_frame"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/bravo/ui"
)

var addFlagsOnce sync.Once

func init() {
	log.SetPrefix("chrest ")
}

func main() {
	ctx := errors.MakeContextDefault()
	ctx.SetCancelOnSignals(syscall.SIGTERM)

	if err := ctx.Run(
		func(ctx errors.Context) {
			if err := run(ctx); err != nil {
				ctx.Cancel(err)
			}
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
}

func run(ctx interfaces.ActiveContext) (err error) {
	var cmd string

	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	for i, arg := range os.Args {
		if arg == cmd {
			os.Args = append(os.Args[:i], os.Args[i+1:]...)
			break
		}
	}

	var c config.Config

	if c, err = config.Default(); err != nil {
		err = errors.Wrap(err)
		return
	}

	switch cmd {
	case "":
		fmt.Println("Usage: chrest <command> [options]")
		fmt.Println()
		fmt.Println("Commands:")
		fmt.Println("  init              Initialize configuration and install native messaging host")
		fmt.Println("  client            Forward HTTP request from stdin to browser")
		fmt.Println("  items-get         Get browser items (tabs, bookmarks, history)")
		fmt.Println("  items-put         Put browser items")
		fmt.Println("  mcp               Run MCP server")
		fmt.Println("  reload-extension  Reload the browser extension")
		return

	case "reload-extension":
		if err = c.Read(); err != nil {
			err = errors.Wrap(err)
			return
		}

		if err = CmdReloadExtension(c); err != nil {
			err = errors.Wrap(err)
			return
		}

	case "client":
		if err = c.Read(); err != nil {
			err = errors.Wrap(err)
			return
		}

		if err = CmdClient(c); err != nil {
			err = errors.Wrap(err)
			return
		}

	case "items-get":
		if err = c.Read(); err != nil {
			err = errors.Wrap(err)
			return
		}

		if err = CmdItemsGet(c); err != nil {
			err = errors.Wrap(err)
			return
		}

	case "items-put":
		if err = c.Read(); err != nil {
			err = errors.Wrap(err)
			return
		}

		if err = CmdItemsPut(c); err != nil {
			err = errors.Wrap(err)
			return
		}

	case "init":
		if err = CmdInit(ctx); err != nil {
			err = errors.Wrap(err)
			return
		}

	case "mcp":
		if err = c.Read(); err != nil {
			err = errors.Wrap(err)
			return
		}

		if err = CmdMcp(ctx, c); err != nil {
			err = errors.Wrap(err)
			return
		}

		// TODO
		// case "demo":
		// 	if err = c.Read(); err != nil {
		// 		err = errors.Wrap(err)
		// 		return
		// 	}

		// err = CmdDemo(c)
	}

	return
}
