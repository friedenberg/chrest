package chrest

import (
	"fmt"
	"path/filepath"

	"code.linenisgreat.com/zit/go/zit/src/alfa/errors"
)

type InstallJSON struct {
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	Path           string   `json:"path"`
	Type           string   `json:"type"`
	AllowedOrigins []string `json:"allowed_origins"`
}

func MakeInstallJSON(p string, ids ...string) (ij InstallJSON, err error) {
	if p, err = filepath.Abs(p); err != nil {
		err = errors.Wrap(err)
		return
	}

	for i, id := range ids {
		ids[i] = fmt.Sprintf("chrome-extension://%s/", id)
	}

	ij = InstallJSON{
		Name:           "com.linenisgreat.code.chrest",
		Description:    "HTTP or socket server for management",
		Path:           p,
		Type:           "stdio",
		AllowedOrigins: ids,
	}

	return
}
