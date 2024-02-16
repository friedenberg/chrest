package chrest

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
)

type Message struct {
	bytes.Buffer
	Content map[string]interface{}
}

func (m *Message) WriteToChrome() (n int64, err error) {
	w := os.Stdout

	var b []byte

	b, err = json.Marshal(m.Content)

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

func (m *Message) WriteTo(w io.Writer) (n int64, err error) {
	var b []byte

	b, err = json.Marshal(m.Content)

	if err != nil {
		return
	}

	var n1 int
	n1, err = WriteAllOrDieTrying(w, b)
	n += int64(n1)

	if err != nil {
		return
	}

	return
}

func (m *Message) ReadFromChrome() (n int64, err error) {
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

	err = json.Unmarshal(b.Bytes(), &m.Content)

	if err != nil {
		return
	}

	return
}

func (m *Message) ReadFrom(r io.Reader) (n int64, err error) {
	return
}
