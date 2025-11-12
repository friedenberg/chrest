package config

import (
	"os"

	"code.linenisgreat.com/chrest/go/src/alfa/browser"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
)

func Default() (c Config, err error) {
	c.DefaultBrowser.Browser = browser.Firefox

	if c.Home, err = os.UserHomeDir(); err != nil {
		err = errors.Wrap(err)
		return
	}

	return
}
