package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os/exec"
)

func main() {
	log.SetPrefix("chrome-json ")

	flagPort := flag.String("port", "3001", "port to serve from")
	flagSock := flag.String("sock", "chrome.sock", "socket to serve from")

	log.Printf("starting server on %s", *flagSock)

	socket := ServerSocket{SockPath: *flagSock}
	socket.Listen()

	socat := exec.Command(
		"socat",
		fmt.Sprintf("TCP4-LISTEN:%s,fork,bind=127.0.0.1", *flagPort),
		fmt.Sprintf("UNIX-CONNECT:%s", *flagSock),
	)

	socat.Start()

	server := http.Server{Handler: http.HandlerFunc(ServeHTTP)}
	server.Serve(socket.Listener)
}
