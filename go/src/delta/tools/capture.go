package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"code.linenisgreat.com/chrest/go/src/bravo/cdp"
	"code.linenisgreat.com/chrest/go/src/charlie/extension"
	"code.linenisgreat.com/chrest/go/src/charlie/firefox"
	"code.linenisgreat.com/chrest/go/src/charlie/headless"
	"code.linenisgreat.com/chrest/go/src/charlie/markdown"
	"code.linenisgreat.com/chrest/go/src/charlie/monolith"
	"code.linenisgreat.com/chrest/go/src/delta/proxy"
	"code.linenisgreat.com/chrest/go/libs/dewey/bravo/errors"
	"code.linenisgreat.com/chrest/go/libs/dewey/golf/command"
	"code.linenisgreat.com/chrest/go/libs/dewey/golf/protocol"
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
	formatHTMLOuter        = "html-outer"
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
	formatHTMLOuter,
}

const captureFormatsDesc = "pdf, screenshot-png, screenshot-jpeg, mhtml, a11y, text, html-monolith, html-outer, markdown-full, markdown-reader, markdown-selector"

var formatExtensions = map[string]string{
	formatPDF:              ".pdf",
	formatScreenshotPNG:    ".png",
	formatScreenshotJPEG:   ".jpeg",
	formatMHTML:            ".mhtml",
	formatA11y:             ".json",
	formatText:             ".txt",
	formatHTMLMonolith:     ".html",
	formatHTMLOuter:        ".html",
	formatMarkdownFull:     ".md",
	formatMarkdownReader:   ".md",
	formatMarkdownSelector: ".md",
}

func FormatExtension(format string) string {
	if ext, ok := formatExtensions[format]; ok {
		return ext
	}
	return ""
}

func ValidFormat(s string) bool {
	for _, f := range captureFormats {
		if f == s {
			return true
		}
	}
	return false
}

// readerEngineReadability runs extraction via the embedded Go
// Readability port. Default and only currently-supported engine.
const readerEngineReadability = "readability"

// readerEngineBrowser would use the browser's native reader view
// (Firefox about:reader). Declared here so the spec surface is stable;
// ConvertReader currently rejects this engine with not-yet-implemented.
const readerEngineBrowser = "browser"

// flexFloat accepts both JSON numbers (1.5) and JSON strings ("1.5").
// Needed because dewey lacks a FloatFlag type, so the MCP schema declares
// these params as "string" while the CLI uses flag.Float64Var directly.
type flexFloat struct {
	Value *float64
}

func (f *flexFloat) UnmarshalJSON(b []byte) error {
	var n float64
	if err := json.Unmarshal(b, &n); err == nil {
		f.Value = &n
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("expected number or string, got %s", string(b))
	}
	if s == "" {
		return nil
	}
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("invalid number %q: %w", s, err)
	}
	f.Value = &n
	return nil
}

func (f flexFloat) MarshalJSON() ([]byte, error) {
	if f.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(*f.Value)
}

// CaptureParams is the shared parameter struct for the `capture` command,
// used by both the MCP tool handler and the CLI bypass in
// go/cmd/chrest/capture.go.
type CaptureParams struct {
	Format       string    `json:"format"`
	URL          string    `json:"url"`
	TabID        string    `json:"tab-id"`
	Browser      string    `json:"browser"`
	Landscape    bool      `json:"landscape"`
	NoHeaders    bool      `json:"no-headers"`
	Background   bool      `json:"background"`
	Quality      int       `json:"quality"`
	FullPage     bool      `json:"full-page"`
	Selector     string    `json:"selector"`
	ReaderEngine string    `json:"reader-engine"`
	PaperWidth   flexFloat `json:"paper-width"`
	PaperHeight  flexFloat `json:"paper-height"`
	MarginTop    flexFloat `json:"margin-top"`
	MarginBottom flexFloat `json:"margin-bottom"`
	MarginLeft   flexFloat `json:"margin-left"`
	MarginRight    flexFloat `json:"margin-right"`
	ViewportWidth  int       `json:"viewport-width"`
}

// Validate rejects format-specific flags used with an incompatible --format,
// so mistakes surface instead of being silently ignored.
func (p CaptureParams) Validate() error {
	pdfOnly := p.Landscape || p.NoHeaders || p.Background
	pdfOnly = pdfOnly || p.PaperWidth.Value != nil || p.PaperHeight.Value != nil
	pdfOnly = pdfOnly || p.MarginTop.Value != nil || p.MarginBottom.Value != nil
	pdfOnly = pdfOnly || p.MarginLeft.Value != nil || p.MarginRight.Value != nil
	if pdfOnly && p.Format != formatPDF {
		return fmt.Errorf("--landscape, --no-headers, --background, --paper-width, --paper-height, --margin-* are only valid with --format %s", formatPDF)
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
		Name: "capture",
		Description: command.Description{
			Short: "Capture a web page in a given format",
			Long: "Capture renders a web page and writes its content to stdout (or to a file " +
				"with --output). The --format flag selects the output format:\n\n" +
				"Document formats: pdf, mhtml, html-monolith, html-outer, text, a11y.\n" +
				"Screenshot formats: screenshot-png, screenshot-jpeg.\n" +
				"Semantic formats: markdown-full, markdown-reader, markdown-selector.\n\n" +
				"The default browser backend is Firefox (WebDriver BiDi). Pass --browser " +
				"chrome for headless Chrome (CDP). Some formats are backend-specific: mhtml " +
				"and a11y require Chrome; html-monolith and the markdown family require " +
				"Firefox.\n\n" +
				"When --output is set, the capture is written atomically via a same-directory " +
				"tmpfile and rename. The tmpfile is cleaned up on failure.\n\n" +
				"PDF captures accept paper dimension and margin overrides (in inches). A " +
				"margin of 0 is valid for borderless output. When unset, the browser default " +
				"is used (typically US Letter 8.5\"x11\" with ~0.4\" margins).",
		},
		Annotations: &protocol.ToolAnnotations{ReadOnlyHint: protocol.BoolPtr(true)},
		SeeAlso:     []string{"capture-batch"},
		Examples: []command.Example{
			{
				Description: "Save a page as PDF",
				Command:     "chrest capture --format pdf --url https://example.com --output page.pdf",
			},
			{
				Description: "Thermal printer: 57mm-wide paper, no side margins",
				Command:     "chrest capture --format pdf --paper-width 2.2409 --margin-left 0 --margin-right 0 --url https://example.com",
			},
			{
				Description: "Full-page screenshot",
				Command:     "chrest capture --format screenshot-png --full-page --url https://example.com --output page.png",
			},
			{
				Description: "Extract readable article content as Markdown",
				Command:     "chrest capture --format markdown-reader --url https://example.com/article",
			},
			{
				Description: "Capture with headless Chrome",
				Command:     "chrest capture --format pdf --browser chrome --url https://example.com",
			},
		},
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
			command.StringFlag{Name: "paper-width", Description: "PDF only: page width in inches (default: browser default, typically 8.5)"},
			command.StringFlag{Name: "paper-height", Description: "PDF only: page height in inches (default: browser default, typically 11)"},
			command.StringFlag{Name: "margin-top", Description: "PDF only: top margin in inches (0 for borderless)"},
			command.StringFlag{Name: "margin-bottom", Description: "PDF only: bottom margin in inches (0 for borderless)"},
			command.StringFlag{Name: "margin-left", Description: "PDF only: left margin in inches (0 for borderless)"},
			command.StringFlag{Name: "margin-right", Description: "PDF only: right margin in inches (0 for borderless)"},
			command.IntFlag{Name: "quality", Description: "screenshot-jpeg only: JPEG quality (0-100)"},
			command.BoolFlag{Name: "full-page", Description: "screenshot-png / screenshot-jpeg: capture the full scrollable page"},
			command.StringFlag{Name: "selector", Description: "markdown-selector only: CSS selector for the element to extract (first match wins)"},
			command.StringFlag{Name: "reader-engine", Description: "markdown-reader only: extraction engine (\"readability\" default; \"browser\" reserved/not-yet-implemented)"},
			command.IntFlag{Name: "viewport-width", Description: "Viewport width in CSS pixels (e.g. 576 for thermal printer). Affects layout of all formats."},
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

	if params.ViewportWidth > 0 {
		if err := session.SetViewport(ctx, params.ViewportWidth, 1024); err != nil {
			return errors.Wrap(err)
		}
	}

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
			PaperWidth:          params.PaperWidth.Value,
			PaperHeight:         params.PaperHeight.Value,
			MarginTop:           params.MarginTop.Value,
			MarginBottom:        params.MarginBottom.Value,
			MarginLeft:          params.MarginLeft.Value,
			MarginRight:         params.MarginRight.Value,
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
	case formatHTMLOuter:
		return s.GetDocumentHTML(ctx)
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
