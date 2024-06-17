package chrest

import (
	"bufio"
	"encoding/json"
	"io"
	"net"
	"net/http"

	"golang.org/x/xerrors"
)

func ResponseFromReader(httpRequestReader io.Reader, conn net.Conn) (resp *http.Response, err error) {
	var req *http.Request

	if req, err = http.ReadRequest(bufio.NewReader(httpRequestReader)); err != nil {
		err = xerrors.Errorf("failed to read request: %w", err)
		return
	}

	if err = req.Write(conn); err != nil {
		err = xerrors.Errorf("failed to write to socket: %w", err)
		return
	}

	if resp, err = http.ReadResponse(bufio.NewReader(conn), req); err != nil {
		err = xerrors.Errorf("failed to read response: %w", err)
		return
	}

	return
}

// TODO figure out which method retunrs err == io.EOF and set err to nil
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
		if err == io.EOF {
			err = nil
		}

		return
	}

	dec := json.NewDecoder(bufio.NewReader(resp.Body))

	if err = dec.Decode(&response); err != nil {
		return
	}

	return
}
