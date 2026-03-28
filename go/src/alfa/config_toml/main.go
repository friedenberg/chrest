package config_toml

import (
	"code.linenisgreat.com/chrest/go/src/alfa/browser"
)

//go:generate tommy generate
type Config struct {
	DefaultBrowser BrowserId `toml:"default-browser"`
}

type BrowserId struct {
	Browser browser.Browser `toml:"browser"`
	Id      string          `toml:"id,omitempty"`
}
