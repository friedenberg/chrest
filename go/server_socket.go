package main

import (
	"encoding/json"
	"io"
	"log"
	"net"
	"os"
)

type ServerSocket struct {
	SockPath string
	Address  *net.UnixAddr
	Listener *net.UnixListener
}

func (s *ServerSocket) Listen() (err error) {
	s.SockPath = "chrome.sock"
	log.Printf("removing old socket: %s", s.SockPath)

	os.Remove(s.SockPath)

	if s.Address, err = net.ResolveUnixAddr("unix", s.SockPath); err != nil {
		return
	}

	if s.Listener, err = net.ListenUnix("unix", s.Address); err != nil {
		return
	}

	log.Printf("listening: %s", s.SockPath)

  return
}

func (s *ServerSocket) acceptConn(conn *net.UnixConn) {
	log.Print("accepted conn")

	dec := json.NewDecoder(conn)
	enc := json.NewEncoder(conn)

	var err error

	for {
		var m Message

		err = dec.Decode(&m.Content)

		if err != nil {
			panic(err)
		}

		_, err = m.WriteToChrome()

		if err != nil {
			panic(err)
		}

		_, err = m.ReadFromChrome()

		if err != nil && err != io.EOF {
			panic(err)
		}

		err = enc.Encode(m.Content)

		if err != nil {
			panic(err)
		}
	}
}
