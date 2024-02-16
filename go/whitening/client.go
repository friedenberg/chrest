package main

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
)

func ResponseFromStdin(conn net.Conn) (resp *http.Response, err error) {
	var req *http.Request

	if req, err = http.ReadRequest(bufio.NewReader(os.Stdin)); err != nil {
		return
	}

	if err = req.Write(conn); err != nil {
		return
	}

	if resp, err = http.ReadResponse(bufio.NewReader(conn), req); err != nil {
		return
	}

	return
}

func ResponseFromArgs(conn net.Conn, args ...string) (resp *http.Response, err error) {
	panic("not implemented")

	client := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return conn, nil
			},
		},
	}

	if resp, err = client.Get(
		fmt.Sprintf("http://localhost/%s", args[0]),
	); err != nil {
		return
	}

	return
}
