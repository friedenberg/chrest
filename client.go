package chrest

import (
	"bufio"
	"encoding/json"
	"io"
	"net"
	"net/http"
)

func ResponseFromReader(r io.Reader, conn net.Conn) (resp *http.Response, err error) {
	var req *http.Request

	if req, err = http.ReadRequest(bufio.NewReader(r)); err != nil {
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

func AskChrome(c Config, req *http.Request) (response interface{}, err error) {
	var sock string

	if sock, err = c.SocketPath(); err != nil {
		return
	}

	var resp *http.Response

	var conn net.Conn

	if conn, err = net.Dial("unix", sock); err != nil {
		return
	}

	if err = req.Write(conn); err != nil {
		return
	}

	if resp, err = http.ReadResponse(bufio.NewReader(conn), req); err != nil {
		return
	}

	dec := json.NewDecoder(bufio.NewReader(resp.Body))

	if err = dec.Decode(&response); err != nil {
		return
	}

	return
}
