package tools

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"code.linenisgreat.com/chrest/go/libs/dewey/bravo/errors"
	"code.linenisgreat.com/chrest/go/src/charlie/firefox"
	"code.linenisgreat.com/chrest/go/src/charlie/markdown"
	"code.linenisgreat.com/chrest/go/src/charlie/monolith"
)

type MultiExtractParams struct {
	URL           string
	Formats       []string
	Selector      string
	ReaderEngine  string
	Quality       int
	FullPage      bool
	ViewportWidth int
}

type FormatResult struct {
	Format string
	Data   []byte
	Err    error
}

func MultiExtract(
	ctx context.Context,
	params MultiExtractParams,
) ([]FormatResult, error) {
	if params.URL == "" {
		return nil, fmt.Errorf("URL is required")
	}
	if len(params.Formats) == 0 {
		return nil, fmt.Errorf("at least one format is required")
	}
	for _, f := range params.Formats {
		if !ValidFormat(f) {
			return nil, fmt.Errorf("unknown format %q", f)
		}
	}

	session, err := openCaptureSession(ctx, params.URL)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	defer session.Close()

	if params.ViewportWidth > 0 {
		if err := session.SetViewport(ctx, params.ViewportWidth, defaultViewportHeight); err != nil {
			return nil, errors.Wrap(err)
		}
	}

	if err := session.Navigate(ctx, params.URL); err != nil {
		return nil, errors.Wrap(err)
	}

	return multiExtractFromSession(ctx, session, params), nil
}

// captureSession is the subset of *firefox.Session used by the internal
// extract helpers. Defined here so tests can inject a mock without depending
// on the real Firefox binary.
type captureSession interface {
	Navigate(ctx context.Context, url string) error
	SetViewport(ctx context.Context, width, height int) error
	GetDocumentHTML(ctx context.Context) (io.ReadCloser, error)
	ExtractText(ctx context.Context) (io.ReadCloser, error)
	PrintToPDF(ctx context.Context, opts firefox.PDFOptions) (io.ReadCloser, error)
	CaptureScreenshot(ctx context.Context, opts firefox.ScreenshotOptions) (io.ReadCloser, error)
	CaptureSnapshot(ctx context.Context) (io.ReadCloser, error)
	AccessibilityTree(ctx context.Context) (io.ReadCloser, error)
	BrowserInfo(ctx context.Context) (firefox.BrowserInfo, error)
	LastNavigationHTTP() (firefox.HTTPResponse, bool)
	Close() error
}

func multiExtractFromSession(
	ctx context.Context,
	s captureSession,
	params MultiExtractParams,
) []FormatResult {
	results := make([]FormatResult, len(params.Formats))

	var domBytes []byte
	var domErr error
	if anyFormatNeedsDOM(params.Formats) {
		var rc io.ReadCloser
		rc, domErr = s.GetDocumentHTML(ctx)
		if domErr == nil {
			domBytes, domErr = io.ReadAll(rc)
			rc.Close()
		}
	}

	for i, format := range params.Formats {
		results[i].Format = format
		results[i].Data, results[i].Err = extractOne(ctx, s, format, params, domBytes, domErr)
	}

	return results
}

var domFormats = map[string]bool{
	formatHTMLOuter:        true,
	formatHTMLMonolith:     true,
	formatMarkdownFull:     true,
	formatMarkdownReader:   true,
	formatMarkdownSelector: true,
}

func anyFormatNeedsDOM(formats []string) bool {
	for _, f := range formats {
		if domFormats[f] {
			return true
		}
	}
	return false
}

func extractOne(
	ctx context.Context,
	s captureSession,
	format string,
	params MultiExtractParams,
	domBytes []byte,
	domErr error,
) ([]byte, error) {
	switch format {
	case formatText:
		return readAllCloser(s.ExtractText(ctx))

	case formatHTMLOuter:
		if domErr != nil {
			return nil, domErr
		}
		return domBytes, nil

	case formatHTMLMonolith:
		if domErr != nil {
			return nil, domErr
		}
		return readAllCloser(monolith.Process(ctx, bytes.NewReader(domBytes), params.URL))

	case formatMarkdownFull:
		if domErr != nil {
			return nil, domErr
		}
		return readAllCloser(markdown.ConvertFull(ctx, bytes.NewReader(domBytes)))

	case formatMarkdownReader:
		if domErr != nil {
			return nil, domErr
		}
		return readAllCloser(markdown.ConvertReader(ctx, bytes.NewReader(domBytes), params.URL))

	case formatMarkdownSelector:
		if domErr != nil {
			return nil, domErr
		}
		return readAllCloser(markdown.ConvertSelector(ctx, bytes.NewReader(domBytes), params.Selector))

	case formatPDF:
		return readAllCloser(s.PrintToPDF(ctx, firefox.PDFOptions{}))

	case formatScreenshotPNG:
		return readAllCloser(s.CaptureScreenshot(ctx, firefox.ScreenshotOptions{
			Format:   "png",
			FullPage: params.FullPage,
		}))

	case formatScreenshotJPEG:
		return readAllCloser(s.CaptureScreenshot(ctx, firefox.ScreenshotOptions{
			Format:   "jpeg",
			Quality:  params.Quality,
			FullPage: params.FullPage,
		}))

	case formatMHTML:
		return readAllCloser(s.CaptureSnapshot(ctx))

	case formatA11y:
		return readAllCloser(s.AccessibilityTree(ctx))

	default:
		return nil, fmt.Errorf("unknown format %q", format)
	}
}

func readAllCloser(rc io.ReadCloser, err error) ([]byte, error) {
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}
