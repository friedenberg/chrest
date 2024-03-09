package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
)

func WriteToChrome(m interface{}) (n int64, err error) {
	w := os.Stdout

	var b []byte

	b, err = json.Marshal(m)

	if err != nil {
		return
	}

	i := int32(len(b))
	// TODO overflow safe
	ml := Int32ToByteArray(i)

	var n1 int
	n1, err = WriteAllOrDieTrying(w, ml[:])
	n += int64(n1)

	if err != nil {
		return
	}

	n1, err = WriteAllOrDieTrying(w, b)
	n += int64(n1)

	if err != nil {
		return
	}

	return
}

func ReadFromChrome(m interface{}) (n int64, err error) {
	r := os.Stdin

	var ml int32

	var n1 int64
	n1, err = ReadInt32(r, &ml)
	n += int64(n1)

	if err != nil {
		return
	}

	var b bytes.Buffer

	n1, err = io.CopyN(&b, r, int64(ml))
	n += n1

	if err != nil {
		return
	}

	err = json.Unmarshal(b.Bytes(), &m)

	if err != nil {
		return
	}

	return
}
