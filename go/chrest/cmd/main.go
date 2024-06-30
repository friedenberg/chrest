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
	"path/filepath"

	"code.linenisgreat.com/chrest/go/chrest"
	"code.linenisgreat.com/zit/go/zit/src/alfa/errors"
	"code.linenisgreat.com/zit/go/zit/src/charlie/files"
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
		var c chrest.Config

		if c, err = chrest.ConfigDefault(); err != nil {
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

		err = CmdDemo(c)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func CmdServer(c chrest.Config) (err error) {
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

	socket := chrest.ServerSocket{SockPath: sock}

	if err = socket.Listen(); err != nil {
		log.Fatal(err)
	}

	server := http.Server{Handler: http.HandlerFunc(chrest.ServeHTTP)}
	server.Serve(socket.Listener)
	return
}

func CmdClient(c chrest.Config) (err error) {
	printFullRequest := flag.Bool(
		"full-request",
		false,
		"print the full request including headers",
	)

	flag.Parse()

	var sock string
	if sock, err = c.SocketPath(); err != nil {
		return
	}

	cmdHttpArgs := append([]string{"--offline"}, flag.Args()...)
	cmdHttpArgs[2] = filepath.Join("localhost", cmdHttpArgs[2])
	cmdHttp := exec.Command("http", cmdHttpArgs...)
	cmdHttp.Stdin = os.Stdin

	var httpieStdout, httpieStderr io.ReadCloser

	if httpieStdout, err = cmdHttp.StdoutPipe(); err != nil {
		err = errors.Wrap(err)
		return
	}

	if httpieStderr, err = cmdHttp.StderrPipe(); err != nil {
		err = errors.Wrap(err)
		return
	}

	// TODO error message when http is missing
	if err = cmdHttp.Start(); err != nil {
		err = errors.Wrap(err)
		return
	}

	var resp *http.Response

	var conn net.Conn

	if conn, err = net.Dial("unix", sock); err != nil {
		err = errors.Wrap(err)
		return
	}

	if *printFullRequest {
		if _, err = io.Copy(os.Stdout, httpieStdout); err != nil {
			err = errors.Errorf("failed to write request to stdout: %w", err)
			return
		}

		if _, err = io.Copy(os.Stderr, httpieStderr); err != nil {
			err = errors.Errorf("failed to write request to stdout: %w", err)
			return
		}

		if err = cmdHttp.Wait(); err != nil {
			err = errors.Errorf("httpie failed: %w", err)
			return
		}

		return
	}

	if resp, err = chrest.ResponseFromReader(httpieStdout, conn); err != nil {
		err = errors.Wrap(err)
		return
	}

	if err = cmdHttp.Wait(); err != nil {
		err = errors.Errorf("waiting for httpie failed: %w", err)
		return
	}

	if files.IsTty(os.Stdout) {
		for k, vs := range resp.Header {
			for _, v := range vs {
				fmt.Printf("%s: %s\n", k, v)
			}
		}

		fmt.Println()
	}

	cmdJq := exec.Command("jq")
	cmdJq.Stdin = resp.Body
	cmdJq.Stdout = os.Stdout

	// TODO error message when jq is missing
	if err = cmdJq.Run(); err != nil {
		err = errors.Wrap(err)
		return
	}

	if resp.StatusCode >= 400 {
		err = errors.Errorf("http error: %s", http.StatusText(resp.StatusCode))
		return
	}

	return
}

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

// TODO use config
func CmdInstall(c chrest.Config) (err error) {
	flag.Parse()

	args := flag.Args()[1:]

	if len(args) < 1 {
		err = errors.Errorf("extension id(s) required")
		return
	}

	var exe string
	if exe, err = os.Executable(); err != nil {
		err = errors.Wrap(err)
		return
	}

	err = nil

	newPath := c.ServerPath()

	if err = chrest.Symlink(exe, newPath); err != nil {
		err = errors.Wrap(err)
		return
	}

	var ij chrest.InstallJSON

	if ij, err = chrest.MakeInstallJSON(
		newPath,
		args...,
	); err != nil {
		err = errors.Wrap(err)
		return
	}

	var b []byte

	if b, err = json.Marshal(ij); err != nil {
		err = errors.Wrap(err)
		return
	}

	path := path.Join(
		c.Home,
		"Library/Application Support/Google/Chrome/NativeMessagingHosts",
		"com.linenisgreat.code.chrest.json",
	)

	if err = os.WriteFile(
		path,
		b,
		0o666,
	); err != nil {
		err = errors.Wrap(err)
		return
	}

	return
}

func CmdDemo(c chrest.Config) (err error) {
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
