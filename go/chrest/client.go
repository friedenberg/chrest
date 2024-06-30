package chrest

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"

	"code.linenisgreat.com/zit/go/zit/src/alfa/errors"
)

func ResponseFromReader(
	httpRequestReader io.Reader,
	conn net.Conn,
) (resp *http.Response, err error) {
	var req *http.Request

	if req, err = http.ReadRequest(bufio.NewReader(httpRequestReader)); err != nil {
		err = errors.Errorf("failed to read request: %w", err)
		return
	}

	if err = req.Write(conn); err != nil {
		err = errors.Errorf("failed to write to socket: %w", err)
		return
	}

	if resp, err = http.ReadResponse(bufio.NewReader(conn), req); err != nil {
		err = errors.Errorf("failed to read response: %w", err)
		return
	}

	return
}

type ResponseWithParsedJSONBody struct {
	*http.Response
	ParsedJSONBody interface{}
}

// TODO figure out which method retunrs err == io.EOF and set err to nil
func AskChrome(
	ctx context.Context,
	c Config,
	req *http.Request,
) (resp ResponseWithParsedJSONBody, err error) {
	var sock string

	if sock, err = c.SocketPath(); err != nil {
		err = errors.Wrap(err)
		return
	}

	var dialer net.Dialer

	var conn net.Conn

	if conn, err = dialer.DialContext(ctx, "unix", sock); err != nil {
		err = errors.Wrap(err)
		return
	}

	if err = req.Write(conn); err != nil {
		err = errors.Wrap(err)
		return
	}

	if resp.Response, err = http.ReadResponse(bufio.NewReader(conn), req); err != nil {
		if err == io.EOF {
			err = nil
		} else {
			err = errors.Wrap(err)
		}

		return
	}

	dec := json.NewDecoder(bufio.NewReader(resp.Body))

	if err = dec.Decode(&resp.ParsedJSONBody); err != nil {
		err = errors.WrapExcept(err, io.EOF)
		return
	}

	return
}
