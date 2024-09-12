package browser_items

import (
	"net/url"

	"code.linenisgreat.com/zit/go/zit/src/alfa/errors"
)

type Url struct {
	url.URL
}

func (u *Url) Set(v string) (err error) {
	return
}

func (u *Url) MarshalText() (b []byte, err error) {
	if b, err = u.URL.MarshalBinary(); err != nil {
		err = errors.Wrap(err)
		return
	}

	return
}

func (u *Url) UnmarshalText(b []byte) (err error) {
	if err = u.URL.UnmarshalBinary(b); err != nil {
		err = errors.Wrap(err)
		return
	}

	return
}
