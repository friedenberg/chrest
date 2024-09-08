package install

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"code.linenisgreat.com/chrest/go/chrest/src/alfa/browser"
	"code.linenisgreat.com/chrest/go/chrest/src/bravo/config"
	"code.linenisgreat.com/zit/go/zit/src/alfa/errors"
)

type IdSet struct {
	browser.Browser
	Ids []string
}

func (is IdSet) String() string {
	var sb strings.Builder

	for i, v := range is.Ids {
		if i > 0 {
			sb.WriteRune(',')
			sb.WriteString(v)
		}
	}

	return sb.String()
}

func (is *IdSet) Set(v string) (err error) {
	is.Ids = append(is.Ids, v)
	return
}

func (is IdSet) GetDefaultId() string {
	switch is.Browser {
	case browser.Chrome, browser.Chromium:
		return "faeaeoifckcedjniagocclagdbbkifgo"

	case browser.Firefox:
		return "chrest@code.linenisgreat.com"

	default:
		panic(errors.Errorf("unsupported browser: %s", is.Browser))
	}
}

type JSONFirefox struct {
	JSONCommon
	AllowedExtensions []string `json:"allowed_extensions"`
}

type JSONChromeOrChromium struct {
	JSONCommon
	AllowedOrigins []string `json:"allowed_origins"`
}

func (is *IdSet) MakeJSON(c config.Config) (loc string, ij any, err error) {
	dir := path.Join(
		c.Home,
		GetUserPath(is.Browser),
	)

	loc = path.Join(
		dir,
		"com.linenisgreat.code.chrest.json",
	)

	if loc, err = filepath.Abs(loc); err != nil {
		err = errors.Wrap(err)
		return
	}

	serverPath := c.ServerPath()

	if len(is.Ids) == 0 {
		is.Ids = append(is.Ids, is.GetDefaultId())
	}

	switch is.Browser {
	case browser.Chrome, browser.Chromium:
		ids := make([]string, len(is.Ids))
		for i, id := range is.Ids {
			ids[i] = fmt.Sprintf("chrome-extension://%s/", id)
		}

		ij = JSONChromeOrChromium{
			JSONCommon: JSONCommon{
				Name:        "com.linenisgreat.code.chrest",
				Description: "HTTP or socket server for management",
				Path:        serverPath,
				Type:        "stdio",
			},
			AllowedOrigins: ids,
		}

	case browser.Firefox:
		ij = JSONFirefox{
			JSONCommon: JSONCommon{
				Name:        "com.linenisgreat.code.chrest",
				Description: "HTTP or socket server for management",
				Path:        serverPath,
				Type:        "stdio",
			},
			AllowedExtensions: is.Ids,
		}

	default:
		err = errors.Errorf("unsupported browser: %s", is.Browser)
		return
	}

	return
}
