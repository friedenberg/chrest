package tools

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"code.linenisgreat.com/chrest/go/src/bravo/cdp"
	"code.linenisgreat.com/chrest/go/src/charlie/markdown"
	"code.linenisgreat.com/chrest/go/src/charlie/monolith"
	"code.linenisgreat.com/chrest/go/src/delta/proxy"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
)

type MultiExtractParams struct {
	URL          string
	Browser      string
	Formats      []string
	Selector     string
	ReaderEngine string
}

type FormatResult struct {
	Format string
	Data   []byte
	Err    error
}

func MultiExtract(
	ctx context.Context,
	p *proxy.BrowserProxy,
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

	session, err := openCaptureSession(ctx, p, params.URL, "", params.Browser)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	defer session.Close()

	if err := session.Navigate(ctx, params.URL); err != nil {
		return nil, errors.Wrap(err)
	}

	return multiExtractFromSession(ctx, session, params), nil
}

func multiExtractFromSession(
	ctx context.Context,
	s cdp.Session,
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
	s cdp.Session,
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
		return readAllCloser(s.PrintToPDF(ctx, cdp.PDFOptions{}))

	case formatScreenshotPNG:
		return readAllCloser(s.CaptureScreenshot(ctx, cdp.ScreenshotOptions{Format: "png"}))

	case formatScreenshotJPEG:
		return readAllCloser(s.CaptureScreenshot(ctx, cdp.ScreenshotOptions{Format: "jpeg"}))

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
