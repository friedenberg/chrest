package install

import "fmt"

type JSONChromeOrChromium struct {
	JSONCommon
	AllowedOrigins []string `json:"allowed_origins"`
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
