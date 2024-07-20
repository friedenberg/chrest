package install

import (
	"fmt"
	"path/filepath"

	"code.linenisgreat.com/chrest/go/chrest/src/alfa/browser"
	"code.linenisgreat.com/zit/go/zit/src/alfa/errors"
)

type JSONCommon struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Path        string `json:"path"`
	Type        string `json:"type"`
}

type JSONChromeOrChromium struct {
	JSONCommon
	AllowedOrigins []string `json:"allowed_origins"`
}

type JSONFirefox struct {
	JSONCommon
	AllowedExtensions []string `json:"allowed_extensions"`
}

func MakeJSON(
	p string,
	b browser.Browser,
	ids ...string,
) (ij any, err error) {
	if p, err = filepath.Abs(p); err != nil {
		err = errors.Wrap(err)
		return
	}

	switch b {
	case browser.Chrome, browser.Chromium:
		ij, err = makeJSONChromeOrChromium(p, ids...)

	case browser.Firefox:
		ij, err = makeJSONFirefox(p, ids...)

	default:
		err = errors.Errorf("unsupported browser: %s", b)
		return
	}

	return
}

func makeJSONChromeOrChromium(
	p string,
	ids ...string,
) (ij JSONChromeOrChromium, err error) {
	for i, id := range ids {
		ids[i] = fmt.Sprintf("chrome-extension://%s/", id)
	}

	ij = JSONChromeOrChromium{
		JSONCommon: JSONCommon{
			Name:        "com.linenisgreat.code.chrest",
			Description: "HTTP or socket server for management",
			Path:        p,
			Type:        "stdio",
		},
		AllowedOrigins: ids,
	}

	return
}

func makeJSONFirefox(
	p string,
	ids ...string,
) (ij JSONFirefox, err error) {
	ij = JSONFirefox{
		JSONCommon: JSONCommon{
			Name:        "com.linenisgreat.code.chrest",
			Description: "HTTP or socket server for management",
			Path:        p,
			Type:        "stdio",
		},
		AllowedExtensions: ids,
	}

	return
}

func MakeInstallJSON(
	bt browser.Browser,
	p string,
	ids ...string,
) (ij any, err error) {
	if p, err = filepath.Abs(p); err != nil {
		err = errors.Wrap(err)
		return
	}

	ij = JSONFirefox{
		JSONCommon: JSONCommon{
			Name:        "com.linenisgreat.code.chrest",
			Description: "HTTP or socket server for management",
			Path:        p,
			Type:        "stdio",
		},
		AllowedExtensions: ids,
	}

	return
}
