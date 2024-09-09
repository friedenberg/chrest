package browser_items

import (
	"fmt"
	"net/url"

	"code.linenisgreat.com/chrest/go/src/bravo/config"
	"code.linenisgreat.com/zit/go/zit/src/alfa/errors"
)

type Item struct {
	Id         ItemId `json:"id"`
	Url        string `json:"url"`
	Date       string `json:"date"`
	Title      string `json:"title"`
	ExternalId string `json:"external-id"` // external to the browser, so for us, it's actually our id
}

func (i Item) GetUrl() (u *url.URL, err error) {
	ur := i.Url

	if ur == "" {
		err = errors.Errorf("empty url")
		return
	}

	if u, err = url.Parse(ur); err != nil {
		err = errors.Wrap(err)
		return
	}

	return
}

type ItemId struct {
	config.BrowserId `json:"browser-id"`
	Id               string `json:"id"`
	Type             string `json:"type"`
}

func (bi ItemId) String() string {
	return fmt.Sprintf("/%s/%s-%s", bi.BrowserId, bi.Type, bi.Id)
}
