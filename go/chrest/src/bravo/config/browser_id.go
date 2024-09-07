package config

import (
	"fmt"
	"strings"

	"code.linenisgreat.com/chrest/go/chrest/src/alfa/browser"
	"code.linenisgreat.com/zit/go/zit/src/alfa/errors"
)

type BrowserId struct {
	Browser browser.Browser `json:"browser"`
	Id      string          `json:"id"`
}

func (bi BrowserId) IsEmpty() bool {
	return bi.Browser == "" && bi.Id == ""
}

func (bi *BrowserId) Set(v string) (err error) {
	v = strings.TrimSpace(strings.ToLower(v))

	head, tail, ok := strings.Cut(v, "-")

	if !ok {
		err = errors.Errorf("unsupported id: %q", v)
		return
	}

	bi.Id = tail

	if err = bi.Browser.Set(head); err != nil {
		err = errors.Wrap(err)
		return
	}

	return
}

func (bi BrowserId) String() string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "%s", bi.Browser)

	if bi.Id == "" {
		return sb.String()
	}

	fmt.Fprintf(&sb, "-%s", bi.Id)

	return sb.String()
}
