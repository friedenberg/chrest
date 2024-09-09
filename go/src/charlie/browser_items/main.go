package browser_items

import (
	"bufio"
	"context"
	"net"
	"net/http"
	"time"

	"code.linenisgreat.com/chrest/go/src/bravo/config"
	"code.linenisgreat.com/zit/go/zit/src/alfa/errors"
)

type BrowserProxy struct {
	config.Config
}

func (b BrowserProxy) Get(
	req BrowserRequestGet,
) (resp HTTPResponseWithRequestPayloadGet, err error) {
	var httpReq *http.Request

	if httpReq, err = req.MakeHTTPRequest(); err != nil {
		err = errors.Wrap(err)
		return
	}

	if resp.Response, err = b.httpRequest(httpReq); err != nil {
		err = errors.Wrap(err)
		return
	}

	defer errors.DeferredCloser(&err, resp.Response.Body)

	if err = resp.parseResponse(); err != nil {
		err = errors.Wrap(err)
		return
	}

	return
}

func (b BrowserProxy) Put(
	req BrowserRequestPut,
) (resp HTTPResponseWithRequestPayloadPut, err error) {
	var httpReq *http.Request

	if httpReq, err = req.MakeHTTPRequest(); err != nil {
		err = errors.Wrap(err)
		return
	}

	if resp.Response, err = b.httpRequest(httpReq); err != nil {
		err = errors.Wrap(err)
		return
	}

	defer errors.DeferredCloser(&err, resp.Response.Body)

	if err = resp.parseResponse(); err != nil {
		err = errors.Wrap(err)
		return
	}

	return
}

func (b BrowserProxy) httpRequest(
	httpReq *http.Request,
) (resp *http.Response, err error) {
	ctx, cancel := context.WithDeadline(
		context.Background(),
		// TODO add default timeout to Config
		time.Now().Add(time.Duration(1e9)),
	)

	defer cancel()

	var sock string
	if sock, err = b.GetSocketPathForBrowserId(b.DefaultBrowser); err != nil {
		err = errors.Wrap(err)
		return
	}

	var dialer net.Dialer

	var conn net.Conn

	if conn, err = dialer.DialContext(ctx, "unix", sock); err != nil {
		err = errors.Wrap(err)
		return
	}

	if err = httpReq.Write(conn); err != nil {
		err = errors.Wrap(err)
		return
	}

	if resp, err = http.ReadResponse(bufio.NewReader(conn), httpReq); err != nil {
		err = errors.Wrap(err)
		return
	}

	return
}
