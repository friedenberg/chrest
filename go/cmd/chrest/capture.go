package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"code.linenisgreat.com/chrest/go/src/delta/proxy"
	"code.linenisgreat.com/chrest/go/src/delta/tools"
)

const defaultCaptureTimeout = 60 * time.Second

// cmdCapture is the CLI-side handler for `chrest capture`. It bypasses
// command.RunCLI so the capture bytes can stream straight to the output
// (os.Stdout or the --output file) without the trailing newline that
// fmt.Println(r.Text) would append in the standard Result path (see
// issue #21). The command is still registered on the command.Utility for
// MCP use — that path buffers into a TextResult.
//
// The underlying dewey gap (no streaming Result variant) is tracked in
// amarbel-llc/purse-first#55. Once that lands, this bypass can be removed
// and the capture command can rely on a single Run handler for both CLI
// and MCP transports.
func cmdCapture(ctx context.Context, p *proxy.BrowserProxy, args []string) (err error) {
	// Ignore SIGPIPE so that a downstream reader closing the pipe early
	// (e.g. `chrest capture | head -c 4`) turns into an EPIPE on the next
	// write instead of killing the program. Without this, Go's default
	// SIGPIPE handler on fd 1/2 terminates chrest before StreamCapture's
	// deferred session.Close runs — leaving Firefox and its content
	// processes orphaned and blocking the bats shutdown.
	signal.Ignore(syscall.SIGPIPE)

	fs := flag.NewFlagSet("capture", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: chrest capture --format <kind> [flags]")
		fmt.Fprintln(os.Stderr, "  Formats: pdf, screenshot-png, screenshot-jpeg, mhtml, a11y, text, html-monolith")
		fs.PrintDefaults()
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "See also: chrest capture-batch  (JSON-stdin batch interface per RFC 0001)")
	}

	var params tools.CaptureParams
	var timeout time.Duration
	var output string
	fs.StringVar(&params.Format, "format", "", "Output format: pdf, screenshot-png, screenshot-jpeg, mhtml, a11y, text, html-monolith")
	fs.StringVar(&params.URL, "url", "", "URL to capture")
	fs.StringVar(&params.TabID, "tab-id", "", "Tab ID to capture (uses extension debugger instead of headless)")
	fs.StringVar(&params.Browser, "browser", "firefox", "Browser backend: firefox (default) or chrome")
	fs.BoolVar(&params.Landscape, "landscape", false, "PDF only: use landscape orientation")
	fs.BoolVar(&params.NoHeaders, "no-headers", false, "PDF only: disable header and footer")
	fs.BoolVar(&params.Background, "background", false, "PDF only: print background graphics")
	fs.IntVar(&params.Quality, "quality", 0, "screenshot-jpeg only: JPEG quality (0-100)")
	fs.BoolVar(&params.FullPage, "full-page", false, "screenshot-png / screenshot-jpeg: capture the full scrollable page")
	fs.DurationVar(&timeout, "timeout", defaultCaptureTimeout, "Abort and tear down the browser if the capture takes longer than this (0 disables)")
	fs.StringVar(&output, "output", "", "Write capture to this path (atomic tmpfile + rename). If unset, stream to stdout.")

	if err = fs.Parse(args); err != nil {
		return err
	}

	if params.Format == "" {
		fs.Usage()
		return fmt.Errorf("--format is required")
	}

	if err = params.Validate(); err != nil {
		return err
	}

	// Self-contained deadline. When it fires, ctx cancels — which
	// propagates into exec.CommandContext (SIGKILL to the browser
	// parent) and into any ctx-aware I/O inside StreamCapture. This
	// guards against hangs the external test harness doesn't notice:
	// Navigate stuck on a loading page, BiDi read stuck waiting for a
	// browser response, etc.
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	var w io.Writer = os.Stdout
	if output != "" {
		// Same-directory tmpfile so the final rename is atomic (same
		// filesystem guarantee). Leading dot keeps the in-flight file
		// out of shell glob expansions.
		f, cerr := os.CreateTemp(filepath.Dir(output), ".chrest-capture-*")
		if cerr != nil {
			return fmt.Errorf("create tmpfile: %w", cerr)
		}
		tmpPath := f.Name()
		w = f
		defer func() {
			// Close before rename: some filesystems don't like renaming
			// an open file, and we want Close's error to count as a
			// capture failure.
			if closeErr := f.Close(); closeErr != nil && err == nil {
				err = closeErr
			}
			if err != nil {
				os.Remove(tmpPath) // best-effort; don't mask the real error
				return
			}
			if rerr := os.Rename(tmpPath, output); rerr != nil {
				err = rerr
				os.Remove(tmpPath)
			}
		}()
	}

	err = tools.StreamCapture(ctx, p, params, w)
	if errors.Is(err, context.DeadlineExceeded) {
		err = fmt.Errorf("capture timed out after %s", timeout)
	}
	return
}
