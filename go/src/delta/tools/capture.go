package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"code.linenisgreat.com/chrest/go/src/bravo/cdp"
	"code.linenisgreat.com/chrest/go/src/charlie/extension"
	"code.linenisgreat.com/chrest/go/src/charlie/firefox"
	"code.linenisgreat.com/chrest/go/src/charlie/headless"
	"code.linenisgreat.com/chrest/go/src/charlie/markdown"
	"code.linenisgreat.com/chrest/go/src/charlie/monolith"
	"code.linenisgreat.com/chrest/go/src/delta/proxy"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/golf/command"
	"github.com/amarbel-llc/purse-first/libs/dewey/golf/protocol"
)

const (
	formatPDF              = "pdf"
	formatScreenshotPNG    = "screenshot-png"
	formatScreenshotJPEG   = "screenshot-jpeg"
	formatMHTML            = "mhtml"
	formatA11y             = "a11y"
	formatText             = "text"
	formatHTMLMonolith     = "html-monolith"
	formatMarkdownFull     = "markdown-full"
	formatMarkdownReader   = "markdown-reader"
	formatMarkdownSelector = "markdown-selector"
)

var captureFormats = []string{
	formatPDF,
	formatScreenshotPNG,
	formatScreenshotJPEG,
	formatMHTML,
	formatA11y,
	formatText,
	formatHTMLMonolith,
	formatMarkdownFull,
	formatMarkdownReader,
	formatMarkdownSelector,
}

const captureFormatsDesc = "pdf, screenshot-png, screenshot-jpeg, mhtml, a11y, text, html-monolith, markdown-full, markdown-reader, markdown-selector"

// readerEngineReadability runs extraction via the embedded Go
// Readability port. Default and only currently-supported engine.
const readerEngineReadability = "readability"

// readerEngineBrowser would use the browser's native reader view
// (Firefox about:reader). Declared here so the spec surface is stable;
// ConvertReader currently rejects this engine with not-yet-implemented.
const readerEngineBrowser = "browser"

// CaptureParams is the shared parameter struct for the `capture` command,
// used by both the MCP tool handler and the CLI bypass in
// go/cmd/chrest/capture.go.
type CaptureParams struct {
	Format       string `json:"format"`
	URL          string `json:"url"`
	TabID        string `json:"tab-id"`
	Browser      string `json:"browser"`
	Landscape    bool   `json:"landscape"`
	NoHeaders    bool   `json:"no-headers"`
	Background   bool   `json:"background"`
	Quality      int    `json:"quality"`
	FullPage     bool   `json:"full-page"`
	Selector     string `json:"selector"`
	ReaderEngine string `json:"reader-engine"`
}

// Validate rejects format-specific flags used with an incompatible --format,
// so mistakes surface instead of being silently ignored.
func (p CaptureParams) Validate() error {
	pdfOnly := p.Landscape || p.NoHeaders || p.Background
	if pdfOnly && p.Format != formatPDF {
		return fmt.Errorf("--landscape, --no-headers, --background are only valid with --format %s", formatPDF)
	}
	if p.Quality != 0 && p.Format != formatScreenshotJPEG {
		return fmt.Errorf("--quality is only valid with --format %s", formatScreenshotJPEG)
	}
	if p.FullPage && p.Format != formatScreenshotPNG && p.Format != formatScreenshotJPEG {
		return fmt.Errorf("--full-page is only valid with --format %s or %s", formatScreenshotPNG, formatScreenshotJPEG)
	}
	if p.Selector != "" && p.Format != formatMarkdownSelector {
		return fmt.Errorf("--selector is only valid with --format %s", formatMarkdownSelector)
	}
	if p.Format == formatMarkdownSelector && p.Selector == "" {
		return fmt.Errorf("--selector is required with --format %s", formatMarkdownSelector)
	}
	if p.ReaderEngine != "" {
		if p.Format != formatMarkdownReader {
			return fmt.Errorf("--reader-engine is only valid with --format %s", formatMarkdownReader)
		}
		switch p.ReaderEngine {
		case readerEngineReadability:
			// ok
		case readerEngineBrowser:
			return fmt.Errorf("--reader-engine=%s is not yet implemented (tracked as a follow-up under chrest#29); use %s", readerEngineBrowser, readerEngineReadability)
		default:
			return fmt.Errorf("--reader-engine must be one of %q, %q (got %q)", readerEngineReadability, readerEngineBrowser, p.ReaderEngine)
		}
	}
	return nil
}

func registerCaptureCommands(app *command.Utility, p *proxy.BrowserProxy) {
	app.AddCommand(&command.Command{
		Name:        "capture",
		Description: command.Description{Short: "Capture a web page in a given format"},
		Annotations: &protocol.ToolAnnotations{ReadOnlyHint: protocol.BoolPtr(true)},
		Params: []command.Param{
			command.StringFlag{
				Name:        "format",
				Description: "Output format: " + captureFormatsDesc,
				Required:    true,
				EnumValues:  captureFormats,
			},
			command.StringFlag{Name: "url", Description: "URL to capture"},
			command.StringFlag{Name: "tab-id", Description: "Tab ID to capture (uses extension debugger instead of headless)"},
			command.StringFlag{Name: "browser", Description: "Browser backend: firefox (default) or chrome"},
			command.BoolFlag{Name: "landscape", Description: "PDF only: use landscape orientation"},
			command.BoolFlag{Name: "no-headers", Description: "PDF only: disable header and footer"},
			command.BoolFlag{Name: "background", Description: "PDF only: print background graphics"},
			command.IntFlag{Name: "quality", Description: "screenshot-jpeg only: JPEG quality (0-100)"},
			command.BoolFlag{Name: "full-page", Description: "screenshot-png / screenshot-jpeg: capture the full scrollable page"},
			command.StringFlag{Name: "selector", Description: "markdown-selector only: CSS selector for the element to extract (first match wins)"},
			command.StringFlag{Name: "reader-engine", Description: "markdown-reader only: extraction engine (\"readability\" default; \"browser\" reserved/not-yet-implemented)"},
		},
		Run: func(ctx context.Context, args json.RawMessage, _ command.Prompter) (*command.Result, error) {
			var p0 CaptureParams
			if err := json.Unmarshal(args, &p0); err != nil {
				return command.TextErrorResult(err.Error()), nil
			}
			if err := p0.Validate(); err != nil {
				return command.TextErrorResult(err.Error()), nil
			}

			// MCP path: buffer into memory since Result carries bytes as a
			// text block. CLI goes through cmd/chrest's bypass, which streams
			// directly to os.Stdout.
			var buf bytes.Buffer
			if err := StreamCapture(ctx, p, p0, &buf); err != nil {
				return nil, err
			}
			return command.TextResult(buf.String()), nil
		},
	})
}

// StreamCapture runs a capture with the given parameters and copies the raw
// output bytes to w. The session is established, the URL (if any) navigated
// to, the capture performed, and the session closed before returning.
//
// Callers that need the bytes as a Go value (e.g. the MCP handler) can pass
// a *bytes.Buffer. Callers that want zero-copy output (e.g. the CLI) can
// pass os.Stdout.
func StreamCapture(
	ctx context.Context,
	p *proxy.BrowserProxy,
	params CaptureParams,
	w io.Writer,
) error {
	session, err := openCaptureSession(ctx, p, params.URL, params.TabID, params.Browser)
	if err != nil {
		return errors.Wrap(err)
	}
	defer session.Close()

	if params.URL != "" {
		if err := session.Navigate(ctx, params.URL); err != nil {
			return errors.Wrap(err)
		}
	}

	rc, err := runCapture(ctx, session, params)
	if err != nil {
		return errors.Wrap(err)
	}
	defer rc.Close()

	if _, err := io.Copy(w, rc); err != nil {
		return errors.Wrap(err)
	}
	return nil
}

func openCaptureSession(
	ctx context.Context,
	p *proxy.BrowserProxy,
	url string,
	tabID string,
	browserBackend string,
) (cdp.Session, error) {
	if tabID != "" {
		return extension.NewSession(ctx, p, tabID)
	}
	if url == "" {
		return nil, fmt.Errorf("--url is required when --tab-id is not specified")
	}
	switch browserBackend {
	case "", "firefox":
		return firefox.NewSession(ctx)
	case "chrome", "headless":
		return headless.NewSession(ctx)
	default:
		return nil, fmt.Errorf("unknown --browser value %q; expected firefox, chrome, or headless", browserBackend)
	}
}

func runCapture(ctx context.Context, s cdp.Session, params CaptureParams) (io.ReadCloser, error) {
	switch params.Format {
	case formatPDF:
		return s.PrintToPDF(ctx, cdp.PDFOptions{
			Landscape:           params.Landscape,
			DisplayHeaderFooter: !params.NoHeaders,
			PrintBackground:     params.Background,
		})
	case formatScreenshotPNG:
		return s.CaptureScreenshot(ctx, cdp.ScreenshotOptions{
			Format:   "png",
			FullPage: params.FullPage,
		})
	case formatScreenshotJPEG:
		return s.CaptureScreenshot(ctx, cdp.ScreenshotOptions{
			Format:   "jpeg",
			Quality:  params.Quality,
			FullPage: params.FullPage,
		})
	case formatMHTML:
		return s.CaptureSnapshot(ctx)
	case formatA11y:
		return s.AccessibilityTree(ctx)
	case formatText:
		return s.ExtractText(ctx)
	case formatHTMLMonolith:
		dom, err := s.GetDocumentHTML(ctx)
		if err != nil {
			return nil, err
		}
		defer dom.Close()
		return monolith.Process(ctx, dom, params.URL)
	case formatMarkdownFull:
		dom, err := s.GetDocumentHTML(ctx)
		if err != nil {
			return nil, err
		}
		defer dom.Close()
		return markdown.ConvertFull(ctx, dom)
	case formatMarkdownReader:
		dom, err := s.GetDocumentHTML(ctx)
		if err != nil {
			return nil, err
		}
		defer dom.Close()
		return markdown.ConvertReader(ctx, dom, params.URL)
	case formatMarkdownSelector:
		dom, err := s.GetDocumentHTML(ctx)
		if err != nil {
			return nil, err
		}
		defer dom.Close()
		return markdown.ConvertSelector(ctx, dom, params.Selector)
	default:
		return nil, fmt.Errorf("unknown --format value %q; expected one of %v", params.Format, captureFormats)
	}
}
