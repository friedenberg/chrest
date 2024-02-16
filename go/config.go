package chrest

import (
	"bufio"
	"encoding/json"
	"log"
	"os"
	"path"
)

type Config struct {
	Port string `json:"port"`
}

func (c Config) SocketPath() (v string, err error) {
	var dir string

	if dir, err = StateDirectory(); err != nil {
		return
	}

	v = path.Join(dir, "chrest.sock")

	return
}

func Default() (c Config, err error) {
	c.Port = "3001"

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

func ConfigDirectory() (v string, err error) {
	v = os.Getenv("XDG_CONFIG_HOME")

	if v == "" {
		v, err = os.UserHomeDir()
		if err != nil {
			return
		}

		v = path.Join(v, ".config")
	}

	v = path.Join(v, "chrest")

	return
}

func (c *Config) Read() (err error) {
	var dir string

	if dir, err = ConfigDirectory(); err != nil {
		return
	}

	p := path.Join(dir, "config.json")

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
	var dir string

	if dir, err = ConfigDirectory(); err != nil {
		return
	}

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
