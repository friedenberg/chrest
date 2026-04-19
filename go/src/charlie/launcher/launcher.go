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
	"strconv"
	"strings"
	"syscall"
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
		killProcessTree(p.cmd.Process.Pid)
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

// killProcessTree SIGKILLs the root process and every descendant.
//
// Two strategies, belt-and-suspenders:
//  1. Walk /proc/<pid>/task/<tid>/children recursively to catch descendants
//     even when they live in their own process groups (Firefox content
//     processes detach into their own groups, so a bare -pgid kill misses
//     them).
//  2. Also SIGKILL the root's process group, picking up anything the /proc
//     walk missed due to reparenting races.
//
// The root is stopped first so it cannot spawn new children during the walk.
// On a normal system, orphaned helpers would be reaped by init; inside
// bwrap --unshare-pid sandboxes there is no init, so unkilled descendants
// linger forever and block the sandbox from exiting.
func killProcessTree(rootPid int) {
	// Freeze the root so its listed children are stable during the walk.
	_ = syscall.Kill(rootPid, syscall.SIGSTOP)

	for _, pid := range collectDescendants(rootPid) {
		_ = syscall.Kill(pid, syscall.SIGKILL)
	}

	if pgid, err := syscall.Getpgid(rootPid); err == nil {
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
	}

	_ = syscall.Kill(rootPid, syscall.SIGKILL)
}

// collectDescendants returns every descendant PID of root, using Linux's
// /proc/<pid>/task/<tid>/children interface. Requires CONFIG_PROC_CHILDREN
// (default-y on recent kernels).
func collectDescendants(root int) []int {
	var result []int
	seen := make(map[int]bool)

	var walk func(int)
	walk = func(pid int) {
		tasks, err := os.ReadDir(fmt.Sprintf("/proc/%d/task", pid))
		if err != nil {
			return
		}
		for _, task := range tasks {
			data, err := os.ReadFile(fmt.Sprintf("/proc/%d/task/%s/children", pid, task.Name()))
			if err != nil {
				continue
			}
			for _, s := range strings.Fields(string(data)) {
				child, err := strconv.Atoi(s)
				if err != nil || seen[child] {
					continue
				}
				seen[child] = true
				result = append(result, child)
				walk(child)
			}
		}
	}
	walk(root)
	return result
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
	// New process group so we can SIGKILL the whole tree in Close.
	// Without this, browser helper processes (content, tab, utility, ...)
	// become orphans when we kill the parent and leak until init reaps
	// them — which never happens inside bwrap --unshare-pid sandboxes.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Disconnect the browser's stdin/stdout from our fds. If we inherit
	// them, the browser forks content processes that also inherit them
	// — including a pipe write-end when chrest is invoked via bash's
	// `run` / command substitution. That pipe never sees EOF until the
	// last content-process dies, which causes the bats harness to hang
	// on shutdown even after every test passes. stderr stays on a pipe
	// because the launcher scrapes it for the WebSocket URL.
	devNull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		if cleanup != nil {
			cleanup()
		}
		return nil, errors.Wrap(err)
	}
	defer devNull.Close()
	cmd.Stdin = devNull
	cmd.Stdout = devNull

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
		killProcessTree(cmd.Process.Pid)
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
