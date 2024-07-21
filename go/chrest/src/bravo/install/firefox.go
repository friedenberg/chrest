package install

type JSONFirefox struct {
	JSONCommon
	AllowedExtensions []string `json:"allowed_extensions"`
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
