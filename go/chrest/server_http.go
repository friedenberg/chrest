package chrest

import (
	"encoding/json"
	"io"
	"net/http"

	"code.linenisgreat.com/zit/go/zit/src/alfa/errors"
)

type (
	JSONAnything          = interface{}
	JSONObject            = map[string]JSONAnything
	ServerRequestJSONBody JSONObject
)

func NewRequest(in *http.Request, body JSONAnything) (out ServerRequestJSONBody) {
	out = map[string]interface{}{
		"path":   in.URL.Path,
		"method": in.Method,
		"body":   body,
	}

	return
}

func ServeHTTP(w http.ResponseWriter, req *http.Request) {
	enc := json.NewEncoder(w)

	dec := json.NewDecoder(req.Body)

	w.Header().Set("Content-Type", "application/json")

	var err error

	var m JSONAnything

	err = dec.Decode(&m)

	if err == io.EOF {
		err = nil
	}

	if err != nil {
		flushError(err, enc, w, req)
	}

	_, err = WriteToChrome(NewRequest(req, m))

	if err != nil {
		flushError(err, enc, w, req)
	}

	var res JSONObject

	_, err = ReadFromChrome(&res)

	if err != nil && err != io.EOF {
		flushError(err, enc, w, req)
	}

	headers, ok := res["headers"].(map[string]interface{})

	if !ok {
		flushError(
			errors.Errorf("expected %T but got %T", headers, res["headers"]),
			enc,
			w,
			req,
		)
	}

	for k, v := range headers {
		vs, ok := v.(string)

		if !ok {
			flushError(
				errors.Errorf("expected %T but got %T", vs, v),
				enc,
				w,
				req,
			)
		}

		w.Header().Add(k, vs)
	}

	w.WriteHeader(int(res["status"].(float64)))

	b, ok := res["body"]

	if !ok {
		return
	}

	switch bjo := b.(type) {
	case JSONObject:
		if len(bjo) == 0 {
			return
		}

	case []JSONObject:
		if len(bjo) == 0 {
			return
		}
	}

	err = enc.Encode(b)

	if err != nil {
		flushError(err, enc, w, req)
	}
}

const StatusBadBoy = http.StatusBadRequest

func flushError(
	err error,
	enc *json.Encoder,
	w http.ResponseWriter,
	req *http.Request,
) {
	w.WriteHeader(StatusBadBoy)

	type errResponse struct {
		Error string `json:"error"`
	}

	res := errResponse{
		Error: err.Error(),
	}

	err = enc.Encode(res)

	if err != nil {
		panic(err)
	}
}
