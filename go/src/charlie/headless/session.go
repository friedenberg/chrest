package headless

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"strings"

	"code.linenisgreat.com/chrest/go/src/bravo/cdp"
	"code.linenisgreat.com/chrest/go/src/charlie/launcher"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
)

// Session implements cdp.Session using a headless Chrome process.
type Session struct {
	chrome *launcher.Process
	conn   *cdp.Conn
}

// Verify interface compliance at compile time.
var _ cdp.Session = (*Session)(nil)

// NewSession launches headless Chrome and connects via CDP.
func NewSession(ctx context.Context) (*Session, error) {
	chrome, err := Launch(ctx)
	if err != nil {
		return nil, err
	}

	conn, err := cdp.Dial(chrome.WSURL())
	if err != nil {
		chrome.Close()
		return nil, err
	}

	return &Session{chrome: chrome, conn: conn}, nil
}

func (s *Session) Navigate(ctx context.Context, url string) error {
	if _, err := s.conn.Send("Page.enable", nil); err != nil {
		return errors.Wrap(err)
	}

	_, err := s.conn.Send("Page.navigate", map[string]string{"url": url})
	if err != nil {
		return errors.Wrap(err)
	}

	// Wait for the page to finish loading by polling until frameStoppedLoading.
	// Simple approach: use Page.getNavigationHistory and check current entry.
	// TODO: listen for Page.loadEventFired event for a more robust solution.
	return nil
}

func (s *Session) PrintToPDF(ctx context.Context, opts cdp.PDFOptions) (io.ReadCloser, error) {
	result, err := s.conn.Send("Page.printToPDF", opts)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	return base64Field(result, "data")
}

func (s *Session) CaptureScreenshot(ctx context.Context, opts cdp.ScreenshotOptions) (io.ReadCloser, error) {
	params := map[string]any{}
	if opts.Format != "" {
		params["format"] = opts.Format
	}
	if opts.Quality > 0 {
		params["quality"] = opts.Quality
	}
	if opts.FullPage {
		params["captureBeyondViewport"] = true
	}

	result, err := s.conn.Send("Page.captureScreenshot", params)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	return base64Field(result, "data")
}

func (s *Session) CaptureSnapshot(ctx context.Context) (io.ReadCloser, error) {
	result, err := s.conn.Send("Page.captureSnapshot", map[string]string{"format": "mhtml"})
	if err != nil {
		return nil, errors.Wrap(err)
	}

	var parsed map[string]string
	if err := json.Unmarshal(result, &parsed); err != nil {
		return nil, errors.Wrap(err)
	}

	return io.NopCloser(strings.NewReader(parsed["data"])), nil
}

func (s *Session) AccessibilityTree(ctx context.Context) (io.ReadCloser, error) {
	result, err := s.conn.Send("Accessibility.getFullAXTree", nil)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	var buf bytes.Buffer
	if err := json.Indent(&buf, result, "", "  "); err != nil {
		return nil, errors.Wrap(err)
	}

	return io.NopCloser(&buf), nil
}

func (s *Session) ExtractText(ctx context.Context) (io.ReadCloser, error) {
	result, err := s.conn.Send("Runtime.evaluate", map[string]any{
		"expression":    "document.body.innerText",
		"returnByValue": true,
	})
	if err != nil {
		return nil, errors.Wrap(err)
	}

	var parsed struct {
		Result struct {
			Value string `json:"value"`
		} `json:"result"`
	}
	if err := json.Unmarshal(result, &parsed); err != nil {
		return nil, errors.Wrap(err)
	}

	return io.NopCloser(strings.NewReader(parsed.Result.Value)), nil
}

func (s *Session) Close() error {
	if s.conn != nil {
		s.conn.Close()
	}
	if s.chrome != nil {
		s.chrome.Close()
	}
	return nil
}

// base64Field extracts a base64-encoded field from a JSON result and
// returns a reader that decodes it.
func base64Field(result json.RawMessage, field string) (io.ReadCloser, error) {
	var parsed map[string]string
	if err := json.Unmarshal(result, &parsed); err != nil {
		return nil, errors.Wrap(err)
	}

	data, ok := parsed[field]
	if !ok {
		return nil, errors.Errorf("CDP response missing %q field", field)
	}

	return io.NopCloser(
		base64.NewDecoder(base64.StdEncoding, strings.NewReader(data)),
	), nil
}
