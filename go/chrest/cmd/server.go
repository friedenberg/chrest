package main

import (
	"log"
	"net/http"
	"os"

	"code.linenisgreat.com/chrest/go/chrest"
)

func CmdServer(c chrest.Config) (err error) {
	if err = c.Read(); err != nil {
		log.Fatal(err)
	}

	serverPath := c.ServerPath()

	var exe string

	if exe, err = os.Executable(); err != nil {
		log.Fatal(err)
	}

	if serverPath != exe {
		log.Fatalf("expected bin: %s, actual bin: %s", serverPath, exe)
	}

	var sock string

	if sock, err = c.SocketPath(); err != nil {
		log.Fatal(err)
	}

	log.Printf("starting server on %q", sock)

	socket := chrest.ServerSocket{SockPath: sock}

	if err = socket.Listen(); err != nil {
		log.Fatal(err)
	}

	server := http.Server{Handler: http.HandlerFunc(chrest.ServeHTTP)}
	server.Serve(socket.Listener)
	return
}
