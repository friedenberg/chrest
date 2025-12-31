package config

import (
	"os"

	"code.linenisgreat.com/chrest/go/src/alfa/browser"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
)

func Default() (config Config, err error) {
	config.DefaultBrowser.Browser = browser.Firefox

	if config.Home, err = os.UserHomeDir(); err != nil {
		err = errors.Wrap(err)
		return
	}

	return
}
