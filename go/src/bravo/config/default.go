package config

import (
	"os"

	"code.linenisgreat.com/chrest/go/src/alfa/browser"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
)

func Default() (config Config, err error) {
	config.DefaultBrowser.Browser = browser.Firefox

	if config.Home, err = os.UserHomeDir(); err != nil {
		err = errors.Wrap(err)
		return
	}

	return
}
