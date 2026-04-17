package firefox

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"strings"

	"code.linenisgreat.com/chrest/go/src/bravo/bidi"
	"code.linenisgreat.com/chrest/go/src/bravo/cdp"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
)

// Session implements cdp.Session using a headless Firefox process via BiDi.
type Session struct {
	firefox   *Firefox
	conn      *bidi.Conn
	contextID string
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
	_, err := s.conn.Send("session.new", map[string]any{
		"capabilities": map[string]any{},
	})
	if err != nil {
		return errors.Wrap(err)
	}

	// Get the default browsing context.
	result, err := s.conn.Send("browsingContext.getTree", map[string]any{})
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
	params := map[string]any{
		"context": s.contextID,
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
	if opts.PaperWidth > 0 || opts.PaperHeight > 0 {
		page := map[string]any{}
		if opts.PaperWidth > 0 {
			page["width"] = opts.PaperWidth
		}
		if opts.PaperHeight > 0 {
			page["height"] = opts.PaperHeight
		}
		params["page"] = page
	}
	if opts.MarginTop > 0 || opts.MarginBottom > 0 || opts.MarginLeft > 0 || opts.MarginRight > 0 {
		margin := map[string]any{}
		if opts.MarginTop > 0 {
			margin["top"] = opts.MarginTop
		}
		if opts.MarginBottom > 0 {
			margin["bottom"] = opts.MarginBottom
		}
		if opts.MarginLeft > 0 {
			margin["left"] = opts.MarginLeft
		}
		if opts.MarginRight > 0 {
			margin["right"] = opts.MarginRight
		}
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
