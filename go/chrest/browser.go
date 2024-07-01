package chrest

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"path"
	"time"

	"code.linenisgreat.com/zit/go/zit/src/alfa/errors"
)

// TODO rename to browser and make structure

type Browser struct {
	Config
}

type BrowserRequest struct {
	Method string
	Path   string
	Body   io.ReadCloser
}

type ResponseWithParsedJSONBody struct {
	*http.Response
	ParsedJSONBody JSONAnything
}

func (b Browser) Request(
	req BrowserRequest,
) (resp ResponseWithParsedJSONBody, err error) {
	var httpReq *http.Request

	if httpReq, err = http.NewRequest(
		req.Method,
		path.Join("http://localhost", req.Path),
		req.Body,
	); err != nil {
		err = errors.Wrap(err)
		return
	}

	if resp, err = b.HTTPRequest(httpReq); err != nil {
		err = errors.Wrap(err)
		return
	}

	return
}

func (b Browser) HTTPRequest(
	req *http.Request,
) (resp ResponseWithParsedJSONBody, err error) {
	ctx, cancel := context.WithDeadline(
		context.Background(),
		// TODO add default timeout to Config
		time.Now().Add(time.Duration(1e9)),
	)

	defer cancel()

	if resp, err = b.HTTPRequestWithContext(
		ctx,
		req,
	); err != nil {
		err = errors.Wrap(err)
		return
	}

	return
}

// TODO figure out which method retunrs err == io.EOF and set err to nil
func (b Browser) HTTPRequestWithContext(
	ctx context.Context,
	req *http.Request,
) (resp ResponseWithParsedJSONBody, err error) {
	var sock string

	if sock, err = b.SocketPath(); err != nil {
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
