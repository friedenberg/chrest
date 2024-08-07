package install

import (
	"path/filepath"

	"code.linenisgreat.com/chrest/go/chrest/src/alfa/browser"
	"code.linenisgreat.com/zit/go/zit/src/alfa/errors"
)

type JSONCommon struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Path        string `json:"path"`
	Type        string `json:"type"`
}

func MakeJSON(
	p string,
	b browser.Browser,
	ids ...string,
) (ij any, err error) {
	if p, err = filepath.Abs(p); err != nil {
		err = errors.Wrap(err)
		return
	}

	switch b {
	case browser.Chrome, browser.Chromium:
		ij, err = makeJSONChromeOrChromium(p, ids...)

	case browser.Firefox:
		ij, err = makeJSONFirefox(p, ids...)

	default:
		err = errors.Errorf("unsupported browser: %s", b)
		return
	}

	return
}
