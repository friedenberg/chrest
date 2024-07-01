package main

import (
	"fmt"
	"log"
	"os"

	"code.linenisgreat.com/chrest/go/chrest"
)

func init() {
	log.SetPrefix("chrest ")
}

func main() {
	var cmd string

	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	for i, x := range os.Args {
		if x == cmd {
			os.Args = append(os.Args[:i], os.Args[i+1:]...)
			break
		}
	}

	var err error

	switch cmd {
	default:
		var c chrest.Config

		if c, err = chrest.ConfigDefault(); err != nil {
			break
		}

		err = CmdServer(c)

	case "reload-extension":
		var c chrest.Config

		if err = c.Read(); err != nil {
			break
		}

		if err = CmdReloadExtension(c); err != nil {
			break
		}

	case "client":
		var c chrest.Config

		if err = c.Read(); err != nil {
			break
		}

		err = CmdClient(c)

	case "init":
		err = CmdInit()

	case "install":
		var c chrest.Config

		if err = c.Read(); err != nil {
			break
		}

		err = CmdInstall(c)

	case "demo":
		var c chrest.Config

		if err = c.Read(); err != nil {
			break
		}

		// TODO
		// err = CmdDemo(c)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
