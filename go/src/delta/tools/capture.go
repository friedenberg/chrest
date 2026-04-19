package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"code.linenisgreat.com/chrest/go/src/bravo/cdp"
	"code.linenisgreat.com/chrest/go/src/charlie/extension"
	"code.linenisgreat.com/chrest/go/src/charlie/firefox"
	"code.linenisgreat.com/chrest/go/src/charlie/headless"
	"code.linenisgreat.com/chrest/go/src/delta/proxy"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/golf/command"
	"github.com/amarbel-llc/purse-first/libs/dewey/golf/protocol"
)

const (
	formatPDF            = "pdf"
	formatScreenshotPNG  = "screenshot-png"
	formatScreenshotJPEG = "screenshot-jpeg"
	formatMHTML          = "mhtml"
	formatA11y           = "a11y"
	formatText           = "text"
)

var captureFormats = []string{
	formatPDF,
	formatScreenshotPNG,
	formatScreenshotJPEG,
	formatMHTML,
	formatA11y,
	formatText,
}

func registerCaptureCommands(app *command.Utility, p *proxy.BrowserProxy) {
	app.AddCommand(&command.Command{
		Name:        "capture",
		Description: command.Description{Short: "Capture a web page in a given format"},
		Annotations: &protocol.ToolAnnotations{ReadOnlyHint: protocol.BoolPtr(true)},
		Params: []command.Param{
			command.StringFlag{
				Name:        "format",
				Description: "Output format: pdf, screenshot-png, screenshot-jpeg, mhtml, a11y, text",
				Required:    true,
				EnumValues:  captureFormats,
			},
			command.StringFlag{Name: "url", Description: "URL to capture"},
			command.StringFlag{Name: "tab-id", Description: "Tab ID to capture (uses extension debugger instead of headless)"},
			command.StringFlag{Name: "browser", Description: "Browser backend: chrome (default) or firefox"},
			command.BoolFlag{Name: "landscape", Description: "PDF only: use landscape orientation"},
			command.BoolFlag{Name: "no-headers", Description: "PDF only: disable header and footer"},
			command.BoolFlag{Name: "background", Description: "PDF only: print background graphics"},
			command.IntFlag{Name: "quality", Description: "screenshot-jpeg only: JPEG quality (0-100)"},
			command.BoolFlag{Name: "full-page", Description: "screenshot-png / screenshot-jpeg: capture the full scrollable page"},
		},
		Run: func(ctx context.Context, args json.RawMessage, _ command.Prompter) (*command.Result, error) {
			var p0 struct {
				Format     string `json:"format"`
				URL        string `json:"url"`
				TabID      string `json:"tab-id"`
				Browser    string `json:"browser"`
				Landscape  bool   `json:"landscape"`
				NoHeaders  bool   `json:"no-headers"`
				Background bool   `json:"background"`
				Quality    int    `json:"quality"`
				FullPage   bool   `json:"full-page"`
			}
			if err := json.Unmarshal(args, &p0); err != nil {
				return command.TextErrorResult(err.Error()), nil
			}

			if err := validateCaptureFlags(p0.Format, p0.Landscape, p0.NoHeaders, p0.Background, p0.Quality, p0.FullPage); err != nil {
				return command.TextErrorResult(err.Error()), nil
			}

			return withSession(ctx, p, p0.URL, p0.TabID, p0.Browser, func(s cdp.Session) (io.ReadCloser, error) {
				switch p0.Format {
				case formatPDF:
					return s.PrintToPDF(ctx, cdp.PDFOptions{
						Landscape:           p0.Landscape,
						DisplayHeaderFooter: !p0.NoHeaders,
						PrintBackground:     p0.Background,
					})
				case formatScreenshotPNG:
					return s.CaptureScreenshot(ctx, cdp.ScreenshotOptions{
						Format:   "png",
						FullPage: p0.FullPage,
					})
				case formatScreenshotJPEG:
					return s.CaptureScreenshot(ctx, cdp.ScreenshotOptions{
						Format:   "jpeg",
						Quality:  p0.Quality,
						FullPage: p0.FullPage,
					})
				case formatMHTML:
					return s.CaptureSnapshot(ctx)
				case formatA11y:
					return s.AccessibilityTree(ctx)
				case formatText:
					return s.ExtractText(ctx)
				default:
					return nil, fmt.Errorf("unknown --format value %q; expected one of %v", p0.Format, captureFormats)
				}
			})
		},
	})
}

// validateCaptureFlags rejects format-specific flags used with an incompatible
// --format, so mistakes surface instead of being silently ignored.
func validateCaptureFlags(format string, landscape, noHeaders, background bool, quality int, fullPage bool) error {
	pdfOnly := landscape || noHeaders || background
	if pdfOnly && format != formatPDF {
		return fmt.Errorf("--landscape, --no-headers, --background are only valid with --format %s", formatPDF)
	}
	if quality != 0 && format != formatScreenshotJPEG {
		return fmt.Errorf("--quality is only valid with --format %s", formatScreenshotJPEG)
	}
	if fullPage && format != formatScreenshotPNG && format != formatScreenshotJPEG {
		return fmt.Errorf("--full-page is only valid with --format %s or %s", formatScreenshotPNG, formatScreenshotJPEG)
	}
	return nil
}

// withSession creates a capture session, optionally navigates to a URL, runs
// the capture function, and returns the result.
// Session selection: --tab-id uses extension debugger, --browser=firefox uses
// headless Firefox via BiDi, otherwise headless Chrome via CDP.
func withSession(
	ctx context.Context,
	p *proxy.BrowserProxy,
	url string,
	tabID string,
	browserBackend string,
	capture func(cdp.Session) (io.ReadCloser, error),
) (*command.Result, error) {
	var session cdp.Session
	var err error

	if tabID != "" {
		session, err = extension.NewSession(ctx, p, tabID)
	} else {
		if url == "" {
			return command.TextErrorResult("--url is required when --tab-id is not specified"), nil
		}
		if browserBackend == "firefox" {
			session, err = firefox.NewSession(ctx)
		} else {
			session, err = headless.NewSession(ctx)
		}
	}

	if err != nil {
		return nil, errors.Wrap(err)
	}
	defer session.Close()

	if url != "" {
		if err := session.Navigate(ctx, url); err != nil {
			return nil, errors.Wrap(err)
		}
	}

	rc, err := capture(session)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	return command.TextResult(string(data)), nil
}
