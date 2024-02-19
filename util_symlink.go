package main

import (
	"os"
)

func Symlink(oldPath, newPath string) (err error) {
	if _, err = os.Lstat(newPath); err == nil {
		err = os.Remove(newPath)
		if err != nil {
			return
		}
	}

	err = os.Symlink(oldPath, newPath)
	if err != nil {
		return
	}

	return
}
