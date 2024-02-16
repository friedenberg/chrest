package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"code.linenisgreat.com/chrest"
)

var (
	flagPort string
	flagSock string
)

func init() {
	log.SetPrefix("chrest-whitening ")
}

func main() {
	cmd := os.Args[1]

	switch cmd {
	default:
		log.Fatalf("unsupported command: %s\n", cmd)

	case "client":
		CmdClient()

	case "init":
		CmdInit()
		fallthrough

	case "install":
		CmdInstall()
	}
}

func CmdClient() {
	flag.Parse()
	var c chrest.Config
	var err error

	if err = c.Read(); err != nil {
		panic(err)
	}

	var sock string
	if sock, err = c.SocketPath(); err != nil {
		panic(err)
	}

	var resp *http.Response

	var conn net.Conn

	if conn, err = net.Dial("unix", sock); err != nil {
		return
	}

	// args := flag.Args()[1:]

	// if len(args) == 0 {
	// 	panic("no path provided")
	// }

	// path := args[0]
	// if path == "-" {
	if resp, err = ResponseFromStdin(conn); err != nil {
		panic(err)
	}
	// } else {
	// 	if resp, err = ResponseFromArgs(conn, args...); err != nil {
	// 		panic(err)
	// 	}
	// }

	io.Copy(os.Stdout, resp.Body)
}

func CmdInit() {
	var c chrest.Config
	var err error

	if c, err = chrest.Default(); err != nil {
		panic(err)
	}

	if err = c.Write(); err != nil {
		panic(err)
	}
}

func CmdInstall() {
	flag.Parse()

	args := flag.Args()[1:]

	if len(args) == 0 {
		panic(fmt.Sprintf("extension id required"))
	}

	var home string
	var err error

	if home, err = os.UserHomeDir(); err != nil {
		panic(err)
	}

	var ij InstallJSON

	if ij, err = MakeInstallJSON(
		filepath.Join(home, ".local", "bin", "chrest-cavity"),
		args...,
	); err != nil {
		panic(err)
	}

	var b []byte

	b, err = json.Marshal(ij)
	if err != nil {
		panic(err)
	}

	path := path.Join(
		home,
		"Library/Application Support/Google/Chrome/NativeMessagingHosts",
		"com.linenisgreat.code.chrest.json",
	)

	{
		err := os.WriteFile(
			path,
			b,
			0o666,
		)
		if err != nil {
			panic(err)
		}
	}
}
