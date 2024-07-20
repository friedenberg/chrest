package main

import (
	"encoding/json"
	"flag"
	"os"
	"path"

	"code.linenisgreat.com/chrest/go/chrest/src/alfa/symlink"
	"code.linenisgreat.com/chrest/go/chrest/src/bravo/config"
	"code.linenisgreat.com/chrest/go/chrest/src/bravo/install"
	"code.linenisgreat.com/zit/go/zit/src/alfa/errors"
)

func CmdInstall(c config.Config) (err error) {
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

	if err = symlink.Symlink(exe, newPath); err != nil {
		err = errors.Wrap(err)
		return
	}

	var ij any

	if ij, err = install.MakeJSON(
		newPath,
		c.Browser,
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

	dir := path.Join(
		c.Home,
		install.GetUserPath(c.Browser),
	)

	path := path.Join(
		dir,
		"com.linenisgreat.code.chrest.json",
	)

	if err = os.MkdirAll(dir, 0o700); err != nil {
		err = errors.Wrap(err)
		return
	}

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
