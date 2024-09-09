package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"

	"code.linenisgreat.com/chrest/go/src/bravo/client"
	"code.linenisgreat.com/chrest/go/src/bravo/config"
	"code.linenisgreat.com/zit/go/zit/src/alfa/errors"
	"code.linenisgreat.com/zit/go/zit/src/charlie/files"
)

func CmdDebug(c config.Config) (err error) {
	flag.Parse()

	var sock string
	if sock, err = c.SocketPathDebug(); err != nil {
		return
	}

	cmdHttpArgs := append([]string{"--offline"}, "GET", "localhost")
	cmdHttp := exec.Command("http", cmdHttpArgs...)
	cmdHttp.Stdin = os.Stdin

	var httpieStdout io.ReadCloser

	if httpieStdout, err = cmdHttp.StdoutPipe(); err != nil {
		err = errors.Wrap(err)
		return
	}

	// if httpieStderr, err = cmdHttp.StderrPipe(); err != nil {
	// 	err = errors.Wrap(err)
	// 	return
	// }

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

	// if _, err = io.Copy(os.Stderr, httpieStderr); err != nil {
	// 	err = errors.Errorf("failed to write request to stdout: %w", err)
	// 	return
	// }

	if resp, err = client.ResponseFromReader(httpieStdout, conn); err != nil {
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
		if errors.IsBrokenPipe(err) {
			err = nil
		} else {
			err = errors.Wrap(err)
		}

		return
	}

	if resp.StatusCode >= 400 {
		err = errors.Errorf("http error: %s", http.StatusText(resp.StatusCode))
		return
	}

	return
}
