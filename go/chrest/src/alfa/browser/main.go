package browser

import (
	"strings"

	"code.linenisgreat.com/zit/go/zit/src/alfa/errors"
)

type Browser string

const (
	Chrome   Browser = "chrome"
	Chromium Browser = "chromium"
	Firefox  Browser = "firefox"
)

func (b *Browser) String() string {
	return string(*b)
}

func (b *Browser) Set(v string) (err error) {
	*b = Browser(strings.TrimSpace(strings.ToLower(v)))

	switch *b {
	case Chrome:
	case Chromium:
	case Firefox:

	default:
		err = errors.Errorf("unsupported browser: %q", b)
		return
	}

	return
}
