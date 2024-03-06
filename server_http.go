package main

import (
	"encoding/json"
	"io"
	"net/http"
)

type (
	JsonAnything = interface{}
	JsonObject   = map[string]JsonAnything
	Request      JsonObject
)

func NewRequest(in *http.Request, body JsonAnything) (out Request) {
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

	var m JsonAnything

	err = dec.Decode(&m)

	if err == io.EOF {
		err = nil
	}

	if err != nil {
		panic(err)
	}

	_, err = WriteToChrome(NewRequest(req, m))

	if err != nil {
		panic(err)
	}

	var res JsonObject

	_, err = ReadFromChrome(&res)

	if err != nil && err != io.EOF {
		panic(err)
	}

	w.WriteHeader(int(res["status"].(float64)))

	b, ok := res["body"]

	if !ok {
		return
	}

	switch bjo := b.(type) {
	case JsonObject:
		if len(bjo) == 0 {
			return
		}

	case []JsonObject:
		if len(bjo) == 0 {
			return
		}
	}

	err = enc.Encode(b)

	if err != nil {
		panic(err)
	}
}
