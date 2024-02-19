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

	"github.com/pkg/errors"
)

func init() {
	log.SetPrefix("chrest ")
}

func main() {
	var cmd string

	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	var err error

	switch cmd {
	default:
		var c Config

		if c, err = ConfigDefault(); err != nil {
			break
		}

		err = CmdServer(c)

	case "client":
		var c Config

		if err = c.Read(); err != nil {
			break
		}

		err = CmdClient(c)

	case "init":
		err = CmdInit()

	case "install":
		var c Config

		if err = c.Read(); err != nil {
			break
		}

		err = CmdInstall(c)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func CmdServer(c Config) (err error) {
	if err = c.Read(); err != nil {
		log.Fatal(err)
	}

	serverPath := c.ServerPath()

	var exe string

	if exe, err = os.Executable(); err != nil {
		log.Fatal(err)
	}

	if serverPath != exe {
		log.Fatalf("expected bin: %s, actual bin: %s", serverPath, exe)
	}

	var sock string

	if sock, err = c.SocketPath(); err != nil {
		log.Fatal(err)
	}

	log.Printf("starting server on %q", sock)

	socket := ServerSocket{SockPath: sock}

	if err = socket.Listen(); err != nil {
		log.Fatal(err)
	}

	server := http.Server{Handler: http.HandlerFunc(ServeHTTP)}
	server.Serve(socket.Listener)
	return
}

func CmdClient(c Config) (err error) {
	flag.Parse()

	var sock string
	if sock, err = c.SocketPath(); err != nil {
		return
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
		return
	}
	// } else {
	// 	if resp, err = ResponseFromArgs(conn, args...); err != nil {
	// 		panic(err)
	// 	}
	// }

	_, err = io.Copy(os.Stdout, resp.Body)

	if err != nil {
		return
	}

	return
}

func CmdInit() (err error) {
	var c Config

	if c, err = ConfigDefault(); err != nil {
		return
	}

	if err = c.Write(); err != nil {
		return
	}

	return CmdInstall(c)
}

// TODO use config
func CmdInstall(c Config) (err error) {
	flag.Parse()

	args := flag.Args()[1:]

	if len(args) < 1 {
		err = errors.Errorf("extension id(s) required")
		return
	}

	var exe string
	exe, err = os.Executable()
	if err != nil {
		return
	}

	err = nil

	newPath := c.ServerPath()

	err = Symlink(exe, newPath)
	if err != nil {
		return
	}

	var ij InstallJSON

	if ij, err = MakeInstallJSON(
		newPath,
		args...,
	); err != nil {
		return
	}

	var b []byte

	b, err = json.Marshal(ij)
	if err != nil {
		return
	}

	path := path.Join(
		c.Home,
		"Library/Application Support/Google/Chrome/NativeMessagingHosts",
		"com.linenisgreat.code.chrest.json",
	)

	err = os.WriteFile(
		path,
		b,
		0o666,
	)
	if err != nil {
		return
	}

	return
}
