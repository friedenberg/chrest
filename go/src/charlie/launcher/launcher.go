package launcher

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"time"

	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
)

type DiscoveryKind int

const (
	StderrRegex DiscoveryKind = iota
	HTTPEndpoint
)

type DiscoveryStrategy struct {
	Kind    DiscoveryKind
	Pattern *regexp.Regexp
}

type BrowserConfig struct {
	BinaryNames []string
	Args        []string
	URL         string
	Discovery   DiscoveryStrategy
	TempProfile string
}

type Process struct {
	cmd     *exec.Cmd
	wsURL   string
	cleanup func()
}

func (p *Process) WSURL() string {
	return p.wsURL
}

func (p *Process) Close() error {
	var firstErr error

	if p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
		if err := p.cmd.Wait(); err != nil {
			if _, ok := err.(*exec.ExitError); !ok {
				firstErr = err
			}
		}
	}

	if p.cleanup != nil {
		p.cleanup()
	}

	return firstErr
}

func findBinary(names []string) (string, error) {
	for _, name := range names {
		if filepath.IsAbs(name) {
			if _, err := os.Stat(name); err == nil {
				return name, nil
			}
			continue
		}
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}
	return "", errors.Errorf("no browser binary found (tried %v)", names)
}

func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, errors.Wrap(err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port, nil
}

func Launch(ctx context.Context, cfg BrowserConfig) (*Process, error) {
	binaryPath, err := findBinary(cfg.BinaryNames)
	if err != nil {
		return nil, err
	}

	args := append([]string{}, cfg.Args...)
	var cleanup func()

	if cfg.TempProfile != "" {
		dir, err := os.MkdirTemp("", cfg.TempProfile+"*")
		if err != nil {
			return nil, errors.Wrap(err)
		}
		args = append(args, "--profile", dir)
		cleanup = func() {
			os.RemoveAll(dir)
		}
	}

	var port int
	if cfg.Discovery.Kind == HTTPEndpoint {
		port, err = freePort()
		if err != nil {
			if cleanup != nil {
				cleanup()
			}
			return nil, errors.Wrap(err)
		}
		args = append(args,
			fmt.Sprintf("--remote-debugging-port=%d", port),
			"--remote-allow-origins=*",
		)
	}

	args = append(args, cfg.URL)

	cmd := exec.CommandContext(ctx, binaryPath, args...)
	log.Printf("launching: %s %v", binaryPath, args)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		if cleanup != nil {
			cleanup()
		}
		return nil, errors.Wrap(err)
	}

	if err := cmd.Start(); err != nil {
		if cleanup != nil {
			cleanup()
		}
		return nil, errors.Wrap(err)
	}
	log.Printf("browser started (pid %d)", cmd.Process.Pid)

	killAndCleanup := func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		if cleanup != nil {
			cleanup()
		}
	}

	if cfg.Discovery.Kind == HTTPEndpoint {
		go func() {
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				log.Printf("browser stderr: %s", scanner.Text())
			}
		}()
	}

	var wsURL string

	switch cfg.Discovery.Kind {
	case StderrRegex:
		wsURL, err = discoverStderr(ctx, stderr, cfg.Discovery.Pattern)
	case HTTPEndpoint:
		wsURL, err = discoverHTTP(ctx, port)
	}

	if err != nil {
		killAndCleanup()
		return nil, err
	}

	log.Printf("browser WebSocket URL: %s", wsURL)
	return &Process{cmd: cmd, wsURL: wsURL, cleanup: cleanup}, nil
}

func discoverStderr(ctx context.Context, stderr interface{ Read([]byte) (int, error) }, pattern *regexp.Regexp) (string, error) {
	found := make(chan string, 1)
	done := make(chan error, 1)

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			log.Printf("browser stderr: %s", line)
			if m := pattern.FindStringSubmatch(line); len(m) > 1 {
				found <- m[1]
				return
			}
		}
		if err := scanner.Err(); err != nil {
			done <- err
		} else {
			done <- errors.Errorf("browser stderr closed without emitting WebSocket URL")
		}
	}()

	select {
	case wsURL := <-found:
		return wsURL, nil
	case err := <-done:
		return "", errors.Wrapf(err, "browser exited before emitting WebSocket URL")
	case <-time.After(10 * time.Second):
		return "", errors.Errorf("timed out waiting for browser WebSocket URL")
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func discoverHTTP(ctx context.Context, port int) (string, error) {
	url := fmt.Sprintf("http://127.0.0.1:%d/json/list", port)
	client := &http.Client{Timeout: 2 * time.Second}
	delay := 50 * time.Millisecond
	deadline := time.After(10 * time.Second)

	for {
		if wsURL, found := pollDevToolsEndpoint(client, url); found {
			return wsURL, nil
		}

		select {
		case <-deadline:
			return "", errors.Errorf("timed out waiting for Chrome DevTools endpoint at %s", url)
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(delay):
			if delay < 500*time.Millisecond {
				delay *= 2
			}
		}
	}
}

func pollDevToolsEndpoint(client *http.Client, url string) (string, bool) {
	resp, err := client.Get(url)
	if err != nil {
		return "", false
	}
	defer resp.Body.Close()

	var entries []struct {
		Type                 string `json:"type"`
		WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return "", false
	}

	for _, e := range entries {
		if e.Type == "page" && e.WebSocketDebuggerURL != "" {
			return e.WebSocketDebuggerURL, true
		}
	}
	return "", false
}
