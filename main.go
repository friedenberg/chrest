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
	"os/exec"
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
		for i, x := range os.Args {
			if x == "client" {
				os.Args = append(os.Args[:i], os.Args[i+1:]...)
				break
			}
		}

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

	case "demo":
		var c Config

		if err = c.Read(); err != nil {
			break
		}

		err = CmdDemo(c)
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
	printFullRequest := flag.Bool("full-request", false, "print the full request including headers")
	flag.Parse()

	var sock string
	if sock, err = c.SocketPath(); err != nil {
		return
	}

	cmdHttpArgs := append([]string{"--offline"}, flag.Args()...)
	cmdHttp := exec.Command("http", cmdHttpArgs...)
	cmdHttp.Stdin = os.Stdin

	var stdout io.ReadCloser

	if stdout, err = cmdHttp.StdoutPipe(); err != nil {
		return
	}

	// TODO error message when http is missing
	if err = cmdHttp.Start(); err != nil {
		return
	}

	var resp *http.Response

	var conn net.Conn

	if conn, err = net.Dial("unix", sock); err != nil {
		return
	}

	if *printFullRequest {
		_, err = io.Copy(os.Stdout, conn)

		if err != nil {
			return
		}

		return
	}

	if resp, err = ResponseFromReader(stdout, conn); err != nil {
		return
	}

	if err = cmdHttp.Wait(); err != nil {
		return
	}

	cmdJq := exec.Command("jq")
	cmdJq.Stdin = resp.Body
	cmdJq.Stdout = os.Stdout

	// TODO error message when jq is missing
	if err = cmdJq.Run(); err != nil {
		return
	}

	if resp.StatusCode >= 400 {
		err = errors.Errorf("http error: %s", http.StatusText(resp.StatusCode))
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

func CmdDemo(c Config) (err error) {
	return
	// flag.Parse()

	// var sock string
	// if sock, err = c.SocketPath(); err != nil {
	// 	return
	// }

	// var resp *http.Response

	// var conn net.Conn

	// if conn, err = net.Dial("unix", sock); err != nil {
	// 	return
	// }

	// script := flag.Args()[1]

	// if resp, err = ResponseFromStdin(conn); err != nil {
	// 	return
	// }
	// // } else {
	// // 	if resp, err = ResponseFromArgs(conn, args...); err != nil {
	// // 		panic(err)
	// // 	}
	// // }

	// _, err = io.Copy(os.Stdout, resp.Body)

	// if err != nil {
	// 	return
	// }

	// if resp.StatusCode >= 400 {
	// 	err = errors.Errorf("http error: %s", http.StatusText(resp.StatusCode))
	// 	return
	// }

	// return
}
