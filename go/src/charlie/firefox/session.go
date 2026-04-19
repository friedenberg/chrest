package firefox

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"strings"

	"code.linenisgreat.com/chrest/go/src/bravo/bidi"
	"code.linenisgreat.com/chrest/go/src/bravo/cdp"
	"code.linenisgreat.com/chrest/go/src/charlie/launcher"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
)

// Session implements cdp.Session using a headless Firefox process via BiDi.
type Session struct {
	firefox      *launcher.Process
	conn         *bidi.Conn
	contextID    string
	capabilities bidiCapabilities
}

// bidiCapabilities captures the subset of the session.new response we
// carry forward into BrowserInfo. See WebDriver BiDi §6.4.1 for the
// full shape.
type bidiCapabilities struct {
	BrowserName    string `json:"browserName"`
	BrowserVersion string `json:"browserVersion"`
	PlatformName   string `json:"platformName"`
	UserAgent      string `json:"userAgent"`
}

// Verify interface compliance at compile time.
var _ cdp.Session = (*Session)(nil)

// NewSession launches headless Firefox and connects via BiDi.
func NewSession(ctx context.Context) (*Session, error) {
	ff, err := Launch(ctx)
	if err != nil {
		return nil, err
	}

	conn, err := bidi.Dial(ff.WSURL())
	if err != nil {
		ff.Close()
		return nil, err
	}

	s := &Session{firefox: ff, conn: conn}

	if err := s.initSession(); err != nil {
		s.Close()
		return nil, err
	}

	return s, nil
}

// initSession creates a BiDi session and discovers the default browsing context.
func (s *Session) initSession() error {
	// Create session.
	result, err := s.conn.Send("session.new", map[string]any{
		"capabilities": map[string]any{},
	})
	if err != nil {
		return errors.Wrap(err)
	}
	// Preserve capabilities for BrowserInfo. Non-fatal if shape surprises us.
	var sessResp struct {
		Capabilities bidiCapabilities `json:"capabilities"`
	}
	if err := json.Unmarshal(result, &sessResp); err == nil {
		s.capabilities = sessResp.Capabilities
	}

	// Get the default browsing context.
	result, err = s.conn.Send("browsingContext.getTree", map[string]any{})
	if err != nil {
		return errors.Wrap(err)
	}

	var tree struct {
		Contexts []struct {
			Context string `json:"context"`
		} `json:"contexts"`
	}
	if err := json.Unmarshal(result, &tree); err != nil {
		return errors.Wrap(err)
	}

	if len(tree.Contexts) == 0 {
		return errors.Errorf("no browsing contexts found")
	}

	s.contextID = tree.Contexts[0].Context
	return nil
}

func (s *Session) Navigate(ctx context.Context, url string) error {
	_, err := s.conn.Send("browsingContext.navigate", map[string]any{
		"context": s.contextID,
		"url":     url,
		"wait":    "complete",
	})
	if err != nil {
		return errors.Wrap(err)
	}
	return nil
}

func (s *Session) PrintToPDF(ctx context.Context, opts cdp.PDFOptions) (io.ReadCloser, error) {
	// BiDi browsingContext.print does not support DisplayHeaderFooter.
	if opts.DisplayHeaderFooter {
		log.Printf("bidi: DisplayHeaderFooter is not supported by Firefox/BiDi, ignoring")
	}

	params := map[string]any{
		"context":     s.contextID,
		"shrinkToFit": true,
	}

	if opts.Landscape {
		params["orientation"] = "landscape"
	}
	if opts.PrintBackground {
		params["background"] = true
	}
	if opts.PageRanges != "" {
		params["pageRanges"] = []string{opts.PageRanges}
	}

	// BiDi uses centimeters; CDP uses inches. Convert if set.
	// A value of 0 is valid (borderless), so use pointers or always set.
	page := map[string]any{}
	if opts.PaperWidth > 0 {
		page["width"] = opts.PaperWidth * 2.54
	}
	if opts.PaperHeight > 0 {
		page["height"] = opts.PaperHeight * 2.54
	}
	if len(page) > 0 {
		params["page"] = page
	}

	margin := map[string]any{}
	if opts.MarginTop > 0 {
		margin["top"] = opts.MarginTop * 2.54
	}
	if opts.MarginBottom > 0 {
		margin["bottom"] = opts.MarginBottom * 2.54
	}
	if opts.MarginLeft > 0 {
		margin["left"] = opts.MarginLeft * 2.54
	}
	if opts.MarginRight > 0 {
		margin["right"] = opts.MarginRight * 2.54
	}
	if len(margin) > 0 {
		params["margin"] = margin
	}

	result, err := s.conn.Send("browsingContext.print", params)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	return base64Field(result, "data")
}

func (s *Session) CaptureScreenshot(ctx context.Context, opts cdp.ScreenshotOptions) (io.ReadCloser, error) {
	params := map[string]any{
		"context": s.contextID,
	}

	if opts.Format == "jpeg" {
		params["format"] = map[string]any{
			"type":    "image/jpeg",
			"quality": opts.Quality,
		}
	}

	if opts.FullPage {
		params["origin"] = "document"
	}

	result, err := s.conn.Send("browsingContext.captureScreenshot", params)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	return base64Field(result, "data")
}

func (s *Session) CaptureSnapshot(ctx context.Context) (io.ReadCloser, error) {
	return nil, errors.Errorf("MHTML capture is not supported with Firefox/BiDi (Chrome-only)")
}

func (s *Session) AccessibilityTree(ctx context.Context) (io.ReadCloser, error) {
	return nil, errors.Errorf("accessibility tree capture is not supported with Firefox/BiDi (Chrome-only)")
}

func (s *Session) ExtractText(ctx context.Context) (io.ReadCloser, error) {
	result, err := s.conn.Send("script.evaluate", map[string]any{
		"expression":      "document.body.innerText",
		"target":          map[string]string{"context": s.contextID},
		"awaitPromise":    false,
		"resultOwnership": "none",
	})
	if err != nil {
		return nil, errors.Wrap(err)
	}

	var parsed struct {
		Result struct {
			Type  string `json:"type"`
			Value string `json:"value"`
		} `json:"result"`
	}
	if err := json.Unmarshal(result, &parsed); err != nil {
		return nil, errors.Wrapf(err, "raw BiDi result: %s", string(result))
	}

	return io.NopCloser(strings.NewReader(parsed.Result.Value)), nil
}

func (s *Session) BrowserInfo(ctx context.Context) (cdp.BrowserInfo, error) {
	info := cdp.BrowserInfo{
		Name:      "firefox",
		Version:   s.capabilities.BrowserVersion,
		UserAgent: s.capabilities.UserAgent,
		Platform:  s.capabilities.PlatformName,
		JSEngine:  "SpiderMonkey",
	}
	if s.firefox != nil {
		info.CommandLine = s.firefox.Args()
	}
	return info, nil
}

func (s *Session) Close() error {
	if s.conn != nil {
		// Best-effort session end.
		_, _ = s.conn.Send("session.end", map[string]any{})
		s.conn.Close()
	}
	if s.firefox != nil {
		s.firefox.Close()
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
		return nil, errors.Errorf("BiDi response missing %q field", field)
	}

	return io.NopCloser(
		base64.NewDecoder(base64.StdEncoding, strings.NewReader(data)),
	), nil
}
