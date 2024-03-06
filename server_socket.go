package main

import (
	"encoding/json"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
)

type ServerSocket struct {
	SockPath string
	Address  *net.UnixAddr
	Listener *net.UnixListener
}

func (s *ServerSocket) Listen() (err error) {
	dir := filepath.Dir(s.SockPath)

	if err = os.MkdirAll(dir, 0o700); err != nil {
		return
	}

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
		var m JsonAnything

		err = dec.Decode(&m)

		if err != nil {
			panic(err)
		}

		_, err = WriteToChrome(m)

		if err != nil {
			panic(err)
		}

		var resp JsonObject

		_, err = ReadFromChrome(&resp)

		if err != nil && err != io.EOF {
			panic(err)
		}

		err = enc.Encode(resp)

		if err != nil {
			panic(err)
		}
	}
}
