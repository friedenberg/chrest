package main

import (
	"io"
	"unsafe"

	"github.com/pkg/errors"
)

func Int32ToByteArray(i int32) [4]byte {
	return *(*[unsafe.Sizeof(i)]byte)(unsafe.Pointer(&i))
}

func ByteArrayToInt32(arr [4]byte) int32 {
	val := int32(0)
	size := len(arr)

	for i := 0; i < size; i++ {
		*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(&val)) + uintptr(i))) = arr[i]
	}

	return val
}

func ReadInt32(r io.Reader, i *int32) (n int64, err error) {
	var b [4]byte

  var n1 int
	n1, err = ReadAllOrDieTrying(r, b[:])
  n += int64(n1)

	if err != nil {
		return
	}

	*i = ByteArrayToInt32(b)

	return
}

func WriteAllOrDieTrying(w io.Writer, b []byte) (n int, err error) {
	var acc int

	for n < len(b) {
		acc, err = w.Write(b[n:])
		n += acc
		if err != nil {
			return
		}
	}

	return
}

func ReadAllOrDieTrying(r io.Reader, b []byte) (n int, err error) {
	var acc int

	for n < len(b) {
		acc, err = r.Read(b[n:])
		n += acc

		if err != nil {
			break
		}
	}

	switch err {
	case io.EOF:
		if n < len(b) && n > 0 {
			err = errors.Wrapf(
				io.ErrUnexpectedEOF,
				"Expected %d, got %d",
				len(b),
				n,
			)
		}

	case nil:
	default:
	}

	return
}
