package main

import (
	"bufio"
	"encoding/json"
	"log"
	"os"
	"path"
	"path/filepath"
)

type Config struct {
	Home string `json:"-"`
	Port string `json:"port"`
}

func (c Config) ServerPath() string {
	return filepath.Join(c.Home, ".local", "bin", "chrest")
}

func (c Config) SocketPath() (v string, err error) {
	var dir string

	if dir, err = StateDirectory(); err != nil {
		return
	}

	v = path.Join(dir, "chrest.sock")

	return
}

func ConfigDefault() (c Config, err error) {
	c.Port = "3001"

	if c.Home, err = os.UserHomeDir(); err != nil {
		return
	}

	return
}

func StateDirectory() (v string, err error) {
	v = os.Getenv("XDG_STATE_HOME")

	if v == "" {
		v, err = os.UserHomeDir()
		if err != nil {
			return
		}

		v = path.Join(v, ".local", "state")
	}

	v = path.Join(v, "chrest")

	return
}

func (c Config) Directory() (v string) {
	v = os.Getenv("XDG_CONFIG_HOME")

	if v == "" {
		v = path.Join(c.Home, ".config")
	}

	v = path.Join(v, "chrest")

	return
}

func (c *Config) Read() (err error) {
	if c.Home, err = os.UserHomeDir(); err != nil {
		return
	}

	p := path.Join(c.Directory(), "config.json")

	var f *os.File

	if f, err = os.Open(p); err != nil {
		return
	}

	defer f.Close()

	dec := json.NewDecoder(bufio.NewReader(f))

	if err = dec.Decode(c); err != nil {
		return
	}

	return
}

func (c *Config) Write() (err error) {
	dir := c.Directory()
	p := path.Join(dir, "config.json")

	var f *os.File

	if f, err = os.CreateTemp("", ""); err != nil {
		return
	}

	defer f.Close()

	w := bufio.NewWriter(f)

	enc := json.NewEncoder(w)

	if err = enc.Encode(c); err != nil {
		return
	}

	if err = w.Flush(); err != nil {
		return
	}

	log.Print(f.Name(), p)

	if err = os.MkdirAll(dir, 0o700); err != nil {
		return
	}

	if err = os.Rename(f.Name(), p); err != nil {
		return
	}

	return
}
