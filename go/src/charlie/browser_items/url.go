package browser_items

import (
	"net/url"

	"code.linenisgreat.com/zit/go/zit/src/alfa/errors"
)

type Url struct {
	String string  `json:"string"`
	Parts  url.URL `json:"parts"`
}

func (u *Url) Set(v string) (err error) {
	if err = u.Parts.UnmarshalBinary([]byte(v)); err != nil {
		err = errors.Wrap(err)
		return
	}

	u.String = v

	return
}

// func (u *Url) UnmarshalJSON(b []byte) (err error) {
// 	return
// }

// func (u *Url) MarshalJSON() (b []byte, err error) {
// 	return
// }

// func (u *Url) MarshalText() (b []byte, err error) {
// 	if b, err = u.Parts.MarshalBinary(); err != nil {
// 		err = errors.Wrap(err)
// 		return
// 	}

// 	return
// }

// func (u *Url) UnmarshalText(b []byte) (err error) {
// 	if err = u.Parts.UnmarshalBinary(b); err != nil {
// 		err = errors.Wrap(err)
// 		return
// 	}

// 	u.String = u.Parts.String()

// 	return
// }
