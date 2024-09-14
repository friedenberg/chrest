package browser_items

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"code.linenisgreat.com/zit/go/zit/src/alfa/errors"
)

type RequestPayloadPut struct {
	Deleted []Item `json:"deleted"`
	Added   []Item `json:"added"`
}

type RequestPayloadGet []Item

type (
	BrowserRequestGet struct{}
	BrowserRequestPut RequestPayloadPut
)

func (BrowserRequestGet) MakeHTTPRequest() (req *http.Request, err error) {
	if req, err = http.NewRequest(
		"GET",
		"/items",
		nil,
	); err != nil {
		err = errors.Wrap(err)
		return
	}

	return
}

func (br BrowserRequestPut) MakeHTTPRequest() (req *http.Request, err error) {
	b := bytes.NewBuffer(nil)

	if req, err = http.NewRequest(
		"PUT",
		"/items",
		io.NopCloser(b),
	); err != nil {
		err = errors.Wrap(err)
		return
	}

	enc := json.NewEncoder(b)

	if err = enc.Encode(br); err != nil {
		err = errors.Wrap(err)
		return
	}

	return
}

type HTTPResponseWithRequestPayloadPut struct {
	*http.Response
	RequestPayloadPut
}

func (resp *HTTPResponseWithRequestPayloadPut) parseResponse() (err error) {
	dec := json.NewDecoder(resp.Response.Body)

	if err = dec.Decode(&resp.RequestPayloadPut); err != nil {
		err = errors.Wrapf(err, "Response: %#v", resp.Response.Body)
		return
	}

	return
}

type HTTPResponseWithRequestPayloadGet struct {
	*http.Response
	RequestPayloadGet
}

func (resp *HTTPResponseWithRequestPayloadGet) parseResponse() (err error) {
	dec := json.NewDecoder(resp.Response.Body)

	if err = dec.Decode(&resp.RequestPayloadGet); err != nil {
		err = errors.Wrapf(err, "Response: %#v", resp)
		return
	}

	return
}
