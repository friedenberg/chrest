package headless

import (
	"bufio"
	"context"
	"os/exec"
	"regexp"
	"time"

	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
)

var wsURLPattern = regexp.MustCompile(`DevTools listening on (ws://\S+)`)

// findChrome locates a Chrome/Chromium binary on PATH.
func findChrome() (string, error) {
	for _, name := range []string{"chromium", "google-chrome-stable", "google-chrome"} {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}
	return "", errors.Errorf("no Chrome/Chromium binary found on PATH")
}

// Chrome manages a headless Chrome process.
type Chrome struct {
	cmd   *exec.Cmd
	wsURL string
}

// Launch starts a headless Chrome and returns the DevTools WebSocket URL.
func Launch(ctx context.Context) (*Chrome, error) {
	chromePath, err := findChrome()
	if err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, chromePath,
		"--headless",
		"--remote-debugging-port=0",
		"--disable-gpu",
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-extensions",
		"about:blank",
	)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, errors.Wrap(err)
	}

	if err := cmd.Start(); err != nil {
		return nil, errors.Wrap(err)
	}

	// Read stderr for the DevTools WebSocket URL.
	found := make(chan string, 1)

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			if m := wsURLPattern.FindStringSubmatch(scanner.Text()); len(m) > 1 {
				found <- m[1]
				return
			}
		}
	}()

	select {
	case wsURL := <-found:
		return &Chrome{cmd: cmd, wsURL: wsURL}, nil
	case <-time.After(10 * time.Second):
		_ = cmd.Process.Kill()
		return nil, errors.Errorf("timed out waiting for Chrome DevTools URL")
	case <-ctx.Done():
		_ = cmd.Process.Kill()
		return nil, ctx.Err()
	}
}

// WSURL returns the WebSocket debugging URL.
func (c *Chrome) WSURL() string {
	return c.wsURL
}

// Close kills the Chrome process.
func (c *Chrome) Close() error {
	if c.cmd.Process != nil {
		_ = c.cmd.Process.Kill()
		_ = c.cmd.Wait()
	}
	return nil
}
