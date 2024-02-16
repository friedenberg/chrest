package main

import (
	"log"
	"net/http"

	"code.linenisgreat.com/chrest"
)

func init() {
	log.SetPrefix("chrest-cavity ")
}

func main() {
	var c chrest.Config
	var err error

	if err = c.Read(); err != nil {
		log.Fatal(err)
	}

	var sock string

	if sock, err = c.SocketPath(); err != nil {
		log.Fatal(err)
	}

	log.Printf("starting server on %q", sock)

	socket := ServerSocket{SockPath: sock}

	if err = socket.Listen(); err != nil {
		log.Fatal(err)
	}

	server := http.Server{Handler: http.HandlerFunc(ServeHTTP)}
	server.Serve(socket.Listener)
}
