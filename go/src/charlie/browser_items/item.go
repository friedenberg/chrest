package browser_items

import (
	"fmt"
	"strings"

	"code.linenisgreat.com/chrest/go/src/bravo/config"
	"code.linenisgreat.com/zit/go/zit/src/alfa/errors"
)

type Item struct {
	Id         ItemId `json:"id"`
	Url        Url    `json:"url"`
	Date       string `json:"date"`
	Title      string `json:"title"`
	ExternalId string `json:"external_id"` // external to the browser, so for us, it's actually our id
}

type ItemId struct {
	config.BrowserId `json:"browser"`
	Type             string `json:"type"`
	Id               string `json:"id"`
}

func (ii *ItemId) Set(v string) (err error) {
	v = strings.TrimPrefix(v, "/")
	head, tail, ok := strings.Cut(v, "/")

	if !ok {
		err = errors.Errorf(
			"expected format like `/<browser>-<name>/<tab|history|bookmark>-<id>` but got %q",
			head,
		)

		return
	}

	if err = ii.BrowserId.Set(head); err != nil {
		err = errors.Wrap(err)
		return
	}

	head, tail, ok = strings.Cut(tail, "-")

	if !ok {
		err = errors.Errorf(
			"expected format like `<tab|history|bookmark>-<id>` but got %q",
			head,
		)

		return
	}

	ii.Type = head
	ii.Id = tail

	return
}

func (bi ItemId) String() string {
	return fmt.Sprintf("/%s/%s-%s", bi.BrowserId, bi.Type, bi.Id)
}
