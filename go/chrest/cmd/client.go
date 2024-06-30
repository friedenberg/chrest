package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"code.linenisgreat.com/chrest/go/chrest"
	"code.linenisgreat.com/zit/go/zit/src/alfa/errors"
	"code.linenisgreat.com/zit/go/zit/src/charlie/files"
)

var (
	printFullRequest   *bool
	clientAddFlagsOnce sync.Once
)

func ClientAddFlags() {
	printFullRequest = flag.Bool(
		"full-request",
		false,
		"print the full request including headers",
	)
}

func CmdServerReload(c chrest.Config) (err error) {
	var exe string

	if exe, err = os.Executable(); err != nil {
		err = errors.Wrap(err)
		return
	}

	os.Args = []string{exe, "POST", "/runtime/reload"}

	fmt.Println("reloading server")

	if err = CmdClient(c); err != nil {
		if errors.Is(err, io.ErrUnexpectedEOF) {
			err = nil
		} else {
			err = errors.Wrap(err)
			return
		}
	}

	os.Args[1] = "GET"

	for {
		fmt.Println("waiting for server to come back up")

		if err = CmdClient(c); err != nil {
			if errors.Is(err, io.ErrUnexpectedEOF) {
				err = nil
				time.Sleep(1e8)
				continue
			} else {
				err = errors.Wrap(err)
				return
			}
		}

		break
	}

	return
}

func CmdClient(c chrest.Config) (err error) {
	clientAddFlagsOnce.Do(ClientAddFlags)
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
