package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"

	"code.linenisgreat.com/chrest/go/chrest/src/bravo/config"
	"code.linenisgreat.com/zit/go/zit/src/alfa/errors"
	"code.linenisgreat.com/zit/go/zit/src/bravo/ui"
)

type Server struct {
	Address         *net.UnixAddr
	Listener        *net.UnixListener
	HTTPHandlerFunc http.HandlerFunc

	chDone chan struct{}
}

func (s *Server) Serve() (err error) {
	s.chDone = make(chan struct{})

	var msgIAm JSONObject
	var browserId string

	for {
		ui.Err().Printf("waiting for id from browser")

		if _, err = ReadFromBrowser(&msgIAm); err != nil {
			err = errors.Wrap(err)
			return
		}

		ui.Err().Printf("read from browser: %q", msgIAm)

		var ok bool

		if browserId, ok = msgIAm["browser_id"].(string); ok {
			break
		}
	}

	var dir string

	if dir, err = config.StateDirectory(); err != nil {
		err = errors.Wrap(err)
		return
	}

	if err = os.MkdirAll(dir, 0o700); err != nil {
		err = errors.Wrap(err)
		return
	}

	pathSock := fmt.Sprintf("%s/%s.sock", dir, browserId)
	ui.Err().Printf("removing old socket: %s", pathSock)
	os.Remove(pathSock)

	ui.Err().Printf("starting server on %q", pathSock)

	if s.Address, err = net.ResolveUnixAddr("unix", pathSock); err != nil {
		err = errors.Wrap(err)
		return
	}

	if s.Listener, err = net.ListenUnix("unix", s.Address); err != nil {
		// TODO add sigil error
		err = errors.Wrap(err)
		return
	}

	ui.Err().Printf("listening: %s", pathSock)

	handler := s.HTTPHandlerFunc

	if handler == nil {
		handler = http.HandlerFunc(s.ServeHTTP)
	}

	server := http.Server{Handler: handler}

	go func() {
		<-s.chDone
		ui.Err().Print("shutting down")
		server.Shutdown(context.Background())
	}()

	if err = server.Serve(s.Listener); err != nil {
		err = errors.Wrap(err)
		return
	}

	return
}
