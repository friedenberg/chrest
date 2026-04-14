package extension

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"strings"

	"code.linenisgreat.com/chrest/go/src/bravo/cdp"
	"code.linenisgreat.com/chrest/go/src/delta/proxy"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
)

// Session implements cdp.Session using the browser extension's
// chrome.debugger API, proxied through BrowserProxy.
type Session struct {
	proxy *proxy.BrowserProxy
	tabID string
}

// Verify interface compliance at compile time.
var _ cdp.Session = (*Session)(nil)

// NewSession creates a session that attaches to the given tab via the
// extension's chrome.debugger API.
func NewSession(ctx context.Context, p *proxy.BrowserProxy, tabID string) (*Session, error) {
	s := &Session{proxy: p, tabID: tabID}

	if _, err := s.proxy.RequestAllBrowsers(ctx, "POST", "/debugger/attach", map[string]any{
		"tabId": tabID,
	}); err != nil {
		return nil, errors.Wrap(err)
	}

	return s, nil
}

func (s *Session) sendCommand(ctx context.Context, method string, params any) (json.RawMessage, error) {
	body := map[string]any{
		"tabId":  s.tabID,
		"method": method,
	}

	if params != nil {
		body["params"] = params
	}

	result, err := s.proxy.RequestAllBrowsers(ctx, "POST", "/debugger/command", body)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	return json.RawMessage(result), nil
}

func (s *Session) Navigate(ctx context.Context, url string) error {
	_, err := s.sendCommand(ctx, "Page.navigate", map[string]string{"url": url})
	return err
}

func (s *Session) PrintToPDF(ctx context.Context, opts cdp.PDFOptions) (io.ReadCloser, error) {
	result, err := s.sendCommand(ctx, "Page.printToPDF", opts)
	if err != nil {
		return nil, err
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

	result, err := s.sendCommand(ctx, "Page.captureScreenshot", params)
	if err != nil {
		return nil, err
	}

	return base64Field(result, "data")
}

func (s *Session) CaptureSnapshot(ctx context.Context) (io.ReadCloser, error) {
	result, err := s.sendCommand(ctx, "Page.captureSnapshot", map[string]string{"format": "mhtml"})
	if err != nil {
		return nil, err
	}

	return textField(result, "data")
}

func (s *Session) AccessibilityTree(ctx context.Context) (io.ReadCloser, error) {
	result, err := s.sendCommand(ctx, "Accessibility.getFullAXTree", nil)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := json.Indent(&buf, result, "", "  "); err != nil {
		return nil, errors.Wrap(err)
	}

	return io.NopCloser(&buf), nil
}

func (s *Session) ExtractText(ctx context.Context) (io.ReadCloser, error) {
	result, err := s.sendCommand(ctx, "Runtime.evaluate", map[string]any{
		"expression":    "document.body.innerText",
		"returnByValue": true,
	})
	if err != nil {
		return nil, err
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
	// Best-effort detach — ignore errors since the tab may already be closed.
	_, _ = s.proxy.RequestAllBrowsers(context.Background(), "POST", "/debugger/detach", map[string]any{
		"tabId": s.tabID,
	})
	return nil
}

// base64Field extracts a base64-encoded field from a JSON result.
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

// textField extracts a plain text field from a JSON result.
func textField(result json.RawMessage, field string) (io.ReadCloser, error) {
	var parsed map[string]string
	if err := json.Unmarshal(result, &parsed); err != nil {
		return nil, errors.Wrap(err)
	}

	data, ok := parsed[field]
	if !ok {
		return nil, errors.Errorf("CDP response missing %q field", field)
	}

	return io.NopCloser(strings.NewReader(data)), nil
}
