package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"

	"code.linenisgreat.com/chrest/go/src/bravo/config"
	"code.linenisgreat.com/zit/go/zit/src/alfa/errors"
	"code.linenisgreat.com/zit/go/zit/src/bravo/ui"
)

type Server struct {
	Address         *net.UnixAddr
	Listener        *net.UnixListener
	HTTPHandlerFunc http.HandlerFunc
	Cancel          context.CancelCauseFunc
}

func (s *Server) Initialize(
	ctx context.Context,
) (err error) {
	var msgIAm JSONObject
	var browserId string

	ui.Err().Printf("waiting for id from browser")

	if _, err = ReadFromBrowser(&msgIAm); err != nil {
		err = errors.Wrap(err)
		return
	}

	ui.Err().Printf("read from browser: %q", msgIAm)

	var ok bool

	if browserId, ok = msgIAm["browser_id"].(string); !ok {
		err = errors.Errorf(
			"expected string `browser_id` but got %T",
			msgIAm["browser_id"],
		)

		return
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

	return
}

func (s *Server) Serve(
	ctx context.Context,
) (err error) {
	handler := s.HTTPHandlerFunc

	if handler == nil {
		handler = http.HandlerFunc(s.ServeHTTP)
	}

	httpServer := http.Server{Handler: handler}

	go func() {
		<-ctx.Done()
		ui.Err().Print("shutting down")

		ctx, cancel := context.WithTimeoutCause(
			context.Background(),
			1e9, // 1 second
			errors.Errorf("shut down timeout"),
		)

		defer cancel()

		httpServer.Shutdown(ctx)
	}()

	if err = httpServer.Serve(s.Listener); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		} else {
			err = errors.Wrap(err)
		}

		return
	}

	return
}
