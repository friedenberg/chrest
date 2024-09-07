package install

import (
	"code.linenisgreat.com/chrest/go/chrest/src/alfa/browser"
	"code.linenisgreat.com/zit/go/zit/src/alfa/errors"
)

func GetId(b browser.Browser) string {
	switch b {
	case browser.Chrome, browser.Chromium:
		panic("not implemented")

	case browser.Firefox:
		return "chrest@code.linenisgreat.com"

	default:
		panic(errors.Errorf("unsupported browser: %s", b))
	}
}
