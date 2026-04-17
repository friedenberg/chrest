package firefox

import (
	"bufio"
	"context"
	"log"
	"os"
	"os/exec"
	"regexp"
	"time"

	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
)

var wsURLPattern = regexp.MustCompile(`WebDriver BiDi listening on (ws://\S+)`)

// findFirefox locates a Firefox binary on PATH.
func findFirefox() (string, error) {
	for _, name := range []string{"firefox", "firefox-esr"} {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}
	return "", errors.Errorf("no Firefox binary found on PATH")
}

// Firefox manages a headless Firefox process.
type Firefox struct {
	cmd        *exec.Cmd
	wsURL      string
	profileDir string
}

// Launch starts a headless Firefox and returns the BiDi WebSocket URL.
func Launch(ctx context.Context) (*Firefox, error) {
	firefoxPath, err := findFirefox()
	if err != nil {
		return nil, err
	}

	profileDir, err := os.MkdirTemp("", "chrest-firefox-*")
	if err != nil {
		return nil, errors.Wrap(err)
	}

	log.Printf("launching: %s --headless --remote-debugging-port=0 --no-remote --profile %s about:blank", firefoxPath, profileDir)

	cmd := exec.CommandContext(ctx, firefoxPath,
		"--headless",
		"--remote-debugging-port=0",
		"--no-remote",
		"--profile", profileDir,
		"about:blank",
	)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, errors.Wrap(err)
	}

	if err := cmd.Start(); err != nil {
		return nil, errors.Wrap(err)
	}

	log.Printf("firefox started (pid %d), waiting for BiDi WebSocket URL...", cmd.Process.Pid)

	found := make(chan string, 1)
	done := make(chan error, 1)

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			log.Printf("firefox stderr: %s", line)
			if m := wsURLPattern.FindStringSubmatch(line); len(m) > 1 {
				found <- m[1]
				return
			}
		}
		if err := scanner.Err(); err != nil {
			done <- err
		} else {
			done <- errors.Errorf("firefox stderr closed without emitting BiDi URL")
		}
	}()

	select {
	case wsURL := <-found:
		log.Printf("firefox BiDi URL: %s", wsURL)
		return &Firefox{cmd: cmd, wsURL: wsURL, profileDir: profileDir}, nil
	case err := <-done:
		_ = cmd.Process.Kill()
		os.RemoveAll(profileDir)
		return nil, errors.Wrapf(err, "firefox exited before emitting BiDi URL")
	case <-time.After(10 * time.Second):
		_ = cmd.Process.Kill()
		os.RemoveAll(profileDir)
		return nil, errors.Errorf("timed out waiting for Firefox BiDi WebSocket URL")
	case <-ctx.Done():
		_ = cmd.Process.Kill()
		os.RemoveAll(profileDir)
		return nil, ctx.Err()
	}
}

// WSURL returns the BiDi WebSocket URL.
func (f *Firefox) WSURL() string {
	return f.wsURL
}

// Close kills the Firefox process and removes the temporary profile.
func (f *Firefox) Close() error {
	if f.cmd.Process != nil {
		_ = f.cmd.Process.Kill()
		_ = f.cmd.Wait()
	}
	if f.profileDir != "" {
		os.RemoveAll(f.profileDir)
	}
	return nil
}
