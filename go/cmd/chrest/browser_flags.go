package main

import (
	"os"
	"strings"

	"code.linenisgreat.com/chrest/go/src/bravo/config"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
)

// browserIdFlags is a repeatable flag type that collects multiple browser IDs.
// When empty, defaults to all browsers. When populated, filters to specified browsers.
type browserIdFlags []config.BrowserId

func (b *browserIdFlags) String() string {
	if b == nil || len(*b) == 0 {
		return ""
	}

	strs := make([]string, len(*b))
	for i, bid := range *b {
		strs[i] = bid.String()
	}
	return strings.Join(strs, ",")
}

func (b *browserIdFlags) Set(value string) error {
	var bid config.BrowserId
	if err := bid.Set(value); err != nil {
		return errors.Wrap(err)
	}
	*b = append(*b, bid)
	return nil
}

// IsEmpty returns true if no browsers have been specified via flags.
func (b browserIdFlags) IsEmpty() bool {
	return len(b) == 0
}

// GetSockets returns socket paths for the specified browsers.
// If no browsers specified, returns all available sockets.
func (b browserIdFlags) GetSockets(c config.Config) ([]string, error) {
	if len(b) == 0 {
		return c.GetAllSockets()
	}

	socks := make([]string, 0, len(b))
	for _, bid := range b {
		sock, err := c.GetSocketPathForBrowserId(bid)
		if err != nil {
			return nil, errors.Wrap(err)
		}
		socks = append(socks, sock)
	}
	return socks, nil
}

// GetSocketForSingle returns a socket path for a single browser.
// If no browser specified, uses the default browser from config.
// Returns an error if multiple browsers are specified.
func (b browserIdFlags) GetSocketForSingle(c config.Config) (string, error) {
	if len(b) > 1 {
		return "", errors.Errorf("expected at most one browser, got %d", len(b))
	}

	var bid config.BrowserId
	if len(b) == 1 {
		bid = b[0]
	}

	return c.GetSocketPathForBrowserId(bid)
}

// ApplyEnvironment applies the CHREST_BROWSER environment variable if no
// browsers have been specified via flags.
func (b *browserIdFlags) ApplyEnvironment() error {
	if len(*b) > 0 {
		return nil // CLI flags take precedence
	}

	envBrowser := os.Getenv("CHREST_BROWSER")
	if envBrowser == "" {
		return nil
	}

	return b.Set(envBrowser)
}
