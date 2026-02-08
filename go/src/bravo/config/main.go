package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/alfa/pool"
	"code.linenisgreat.com/dodder/go/src/foxtrot/fd"
)

type MCPConfig struct {
	Scopes map[string]string `json:"scopes,omitempty"`
}

type Config struct {
	DefaultBrowser BrowserId   `json:"default-browser"`
	MCP            MCPConfig   `json:"mcp,omitempty"`
	LoadedBrowsers []BrowserId `json:"-"`
	Home           string      `json:"-"`
}

func (config Config) ServerPath() string {
	return filepath.Join(config.Home, ".local", "bin", "chrest-server")
}

func StateDirectory() (value string, err error) {
	value = os.Getenv("XDG_STATE_HOME")

	if value == "" {
		if value, err = os.UserHomeDir(); err != nil {
			err = errors.Wrap(err)
			return
		}

		value = path.Join(value, ".local", "state")
	}

	value = path.Join(value, "chrest")

	return
}

func (config Config) GetSocketPathForBrowserId(
	id BrowserId,
) (sock string, err error) {
	if id.IsEmpty() {
		id = config.DefaultBrowser
	}

	var stateDir string

	if stateDir, err = StateDirectory(); err != nil {
		err = errors.Wrap(err)
		return
	}

	sock = path.Join(stateDir, fmt.Sprintf("%s.sock", id))

	return
}

func (config Config) GetAllSockets() (socks []string, err error) {
	var stateDir string

	if stateDir, err = StateDirectory(); err != nil {
		err = errors.Wrap(err)
		return
	}

	if socks, err = filepath.Glob(filepath.Join(stateDir, "*.sock")); err != nil {
		err = errors.Wrap(err)
		return
	}

	return
}

func (config Config) Directory() (v string) {
	v = os.Getenv("XDG_CONFIG_HOME")

	if v == "" {
		v = path.Join(config.Home, ".config")
	}

	v = path.Join(v, "chrest")

	return
}

func (config *Config) Read() (err error) {
	wg := errors.MakeWaitGroupParallel()

	wg.Do(config.readConfig)
	wg.Do(config.readLoadedBrowsers)

	if err = wg.GetError(); err != nil {
		err = errors.Wrap(err)
		return
	}

	return
}

func (config *Config) readConfig() (err error) {
	if config.Home, err = os.UserHomeDir(); err != nil {
		err = errors.Wrap(err)
		return
	}

	p := path.Join(config.Directory(), "config.json")

	var f *os.File

	if f, err = os.Open(p); err != nil {
		if errors.IsNotExist(err) {
			err = nil
		} else {
			err = errors.Wrap(err)
		}

		return
	}

	defer errors.DeferredCloser(&err, f)

	dec := json.NewDecoder(bufio.NewReader(f))

	if err = dec.Decode(config); err != nil {
		err = errors.Wrap(err)
		return
	}

	return
}

func (config *Config) readLoadedBrowsers() (err error) {
	var loadedBrowserPaths []string

	var stateDir string

	if stateDir, err = StateDirectory(); err != nil {
		err = errors.Wrap(err)
		return
	}

	if loadedBrowserPaths, err = filepath.Glob(filepath.Join(stateDir, "*.sock")); err != nil {
		err = errors.Wrap(err)
		return
	}

	for _, loadedBrowserPath := range loadedBrowserPaths {
		var id BrowserId

		if err = id.Set(fd.FileNameSansExt(loadedBrowserPath)); err != nil {
			err = errors.Wrap(err)
			return
		}

		config.LoadedBrowsers = append(config.LoadedBrowsers, id)
	}

	return
}

func (config *Config) Write(ctx interfaces.ActiveContext) (err error) {
	dir := config.Directory()
	path := path.Join(dir, "config.json")

	tempFileName := func() (tempFileName string) {
		var file *os.File

		if file, err = os.CreateTemp("", ""); err != nil {
			err = errors.Wrap(err)
			return
		}

		tempFileName = file.Name()

		defer errors.ContextMustClose(ctx, file)

		bufferedWriter, repool := pool.GetBufferedWriter(file)
		defer repool()

		defer errors.ContextMustFlush(ctx, bufferedWriter)

		enc := json.NewEncoder(bufferedWriter)

		if err = enc.Encode(config); err != nil {
			err = errors.Wrap(err)
			return
		}

		return
	}()

	log.Print(tempFileName, path)

	if err = os.MkdirAll(dir, 0o700); err != nil {
		err = errors.Wrap(err)
		return
	}

	if err = os.Rename(tempFileName, path); err != nil {
		err = errors.Wrap(err)
		return
	}

	return
}
