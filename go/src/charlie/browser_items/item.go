package browser_items

import (
	"fmt"

	"code.linenisgreat.com/chrest/go/src/bravo/config"
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
	Id               string `json:"id"`
	Type             string `json:"type"`
}

func (bi ItemId) String() string {
	return fmt.Sprintf("/%s/%s-%s", bi.BrowserId, bi.Type, bi.Id)
}
