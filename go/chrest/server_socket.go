package chrest

import (
	"log"
	"net"
	"os"
	"path/filepath"

	"code.linenisgreat.com/zit/go/zit/src/alfa/errors"
)

type ServerSocket struct {
	SockPath string
	Address  *net.UnixAddr
	Listener *net.UnixListener
}

func (s *ServerSocket) Listen() (err error) {
	dir := filepath.Dir(s.SockPath)

	if err = os.MkdirAll(dir, 0o700); err != nil {
		err = errors.Wrap(err)
		return
	}

	log.Printf("removing old socket: %s", s.SockPath)

	os.Remove(s.SockPath)

	if s.Address, err = net.ResolveUnixAddr("unix", s.SockPath); err != nil {
		err = errors.Wrap(err)
		return
	}

	if s.Listener, err = net.ListenUnix("unix", s.Address); err != nil {
		err = errors.Wrap(err)
		return
	}

	log.Printf("listening: %s", s.SockPath)

	return
}
