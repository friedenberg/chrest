package install

import (
	"code.linenisgreat.com/chrest/go/src/bravo/config"
	"code.linenisgreat.com/zit/go/zit/src/alfa/errors"
)

type JSONCommon struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Path        string `json:"path"`
	Type        string `json:"type"`
}

func MakeJSON(c config.Config, ids ...IdSet) (ijs map[string]any, err error) {
	ijs = make(map[string]any, len(ids))

	for _, is := range ids {
		var ij any
		var loc string

		if loc, ij, err = is.MakeJSON(c); err != nil {
			err = errors.Wrap(err)
			return
		}

		ijs[loc] = ij
	}

	return
}
