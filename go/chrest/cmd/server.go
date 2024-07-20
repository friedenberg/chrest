package main

import (
	"log"
	"net/http"

	"code.linenisgreat.com/chrest/go/chrest/src/bravo/config"
	"code.linenisgreat.com/chrest/go/chrest/src/bravo/server"
	"code.linenisgreat.com/zit/go/zit/src/alfa/errors"
)

func CmdServer(c config.Config) (err error) {
	if err = c.Read(); err != nil {
		err = errors.Wrap(err)
		return
	}

	// serverPath := c.ServerPath()

	// var exe string

	// if exe, err = os.Executable(); err != nil {
	// 	err = errors.Wrap(err)
	// 	return
	// }

	// if serverPath != exe {
	// 	err = errors.Errorf("expected bin: %s, actual bin: %s", serverPath, exe)
	// 	return
	// }

	var sock string

	if sock, err = c.SocketPath(); err != nil {
		err = errors.Wrap(err)
		return
	}

	log.Printf("starting server on %q", sock)

	socket := server.ServerSocket{SockPath: sock}

	if err = socket.Listen(); err != nil {
		err = errors.Wrap(err)
		return
	}

	server := http.Server{Handler: http.HandlerFunc(server.ServeHTTP)}

	if err = server.Serve(socket.Listener); err != nil {
		err = errors.Wrap(err)
		return
	}

	return
}
