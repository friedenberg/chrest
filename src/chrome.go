package chrest

import (
	"bytes"
	"encoding/json"
	"io"
	"os"

	"code.linenisgreat.com/zit/go/zit/src/alfa/errors"
)

func WriteToChrome(m interface{}) (n int64, err error) {
	w := os.Stdout

	var b []byte

	if b, err = json.Marshal(m); err != nil {
		err = errors.Wrap(err)
		return
	}

	i := int32(len(b))
	// TODO overflow safe
	ml := Int32ToByteArray(i)

	var n1 int
	n1, err = WriteAllOrDieTrying(w, ml[:])
	n += int64(n1)

	if err != nil {
		err = errors.Wrap(err)
		return
	}

	n1, err = WriteAllOrDieTrying(w, b)
	n += int64(n1)

	if err != nil {
		err = errors.Wrap(err)
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
		err = errors.Wrap(err)
		return
	}

	var b bytes.Buffer

	n1, err = io.CopyN(&b, r, int64(ml))
	n += n1

	if err != nil {
		err = errors.Wrap(err)
		return
	}

	err = json.Unmarshal(b.Bytes(), &m)

	if err != nil {
		err = errors.Wrap(err)
		return
	}

	return
}
