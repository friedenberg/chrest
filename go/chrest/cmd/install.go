package main

import (
	"encoding/json"
	"flag"
	"os"
	"path"

	"code.linenisgreat.com/chrest/go/chrest"
	"code.linenisgreat.com/zit/go/zit/src/alfa/errors"
)

// TODO use config
func CmdInstall(c chrest.Config) (err error) {
	flag.Parse()
	args := flag.Args()

	if len(args) < 1 {
		err = errors.Errorf("extension id(s) required")
		return
	}

	var exe string
	if exe, err = os.Executable(); err != nil {
		err = errors.Wrap(err)
		return
	}

	err = nil

	newPath := c.ServerPath()

	if err = chrest.Symlink(exe, newPath); err != nil {
		err = errors.Wrap(err)
		return
	}

	var ij chrest.InstallJSON

	if ij, err = chrest.MakeInstallJSON(
		newPath,
		args...,
	); err != nil {
		err = errors.Wrap(err)
		return
	}

	var b []byte

	if b, err = json.Marshal(ij); err != nil {
		err = errors.Wrap(err)
		return
	}

	path := path.Join(
		c.Home,
		"Library/Application Support/Google/Chrome/NativeMessagingHosts",
		"com.linenisgreat.code.chrest.json",
	)

	if err = os.WriteFile(
		path,
		b,
		0o666,
	); err != nil {
		err = errors.Wrap(err)
		return
	}

	return
}
