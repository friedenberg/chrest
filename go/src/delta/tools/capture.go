package tools

import (
	"context"
	"encoding/json"
	"io"

	"code.linenisgreat.com/chrest/go/src/bravo/cdp"
	"code.linenisgreat.com/chrest/go/src/charlie/headless"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/golf/command"
	"github.com/amarbel-llc/purse-first/libs/dewey/golf/protocol"
)

func registerCaptureCommands(app *command.Utility) {
	url := command.StringFlag{Name: "url", Required: true, Description: "URL to capture"}

	// capture-pdf
	app.AddCommand(&command.Command{
		Name:        "capture-pdf",
		Description: command.Description{Short: "Capture a web page as PDF"},
		Annotations: &protocol.ToolAnnotations{ReadOnlyHint: protocol.BoolPtr(true)},
		Params: []command.Param{
			url,
			command.BoolFlag{Name: "landscape", Description: "Use landscape orientation"},
			command.BoolFlag{Name: "no-headers", Description: "Disable header and footer"},
			command.BoolFlag{Name: "background", Description: "Print background graphics"},
		},
		Run: func(ctx context.Context, args json.RawMessage, _ command.Prompter) (*command.Result, error) {
			var p0 struct {
				URL        string `json:"url"`
				Landscape  bool   `json:"landscape"`
				NoHeaders  bool   `json:"no-headers"`
				Background bool   `json:"background"`
			}
			if err := json.Unmarshal(args, &p0); err != nil {
				return command.TextErrorResult(err.Error()), nil
			}

			return withSession(ctx, p0.URL, func(s cdp.Session) (io.ReadCloser, error) {
				return s.PrintToPDF(ctx, cdp.PDFOptions{
					Landscape:           p0.Landscape,
					DisplayHeaderFooter: !p0.NoHeaders,
					PrintBackground:     p0.Background,
				})
			})
		},
	})

	// capture-screenshot
	app.AddCommand(&command.Command{
		Name:        "capture-screenshot",
		Description: command.Description{Short: "Capture a web page as an image"},
		Annotations: &protocol.ToolAnnotations{ReadOnlyHint: protocol.BoolPtr(true)},
		Params: []command.Param{
			url,
			command.StringFlag{Name: "format", Description: "Image format: png (default) or jpeg"},
			command.IntFlag{Name: "quality", Description: "JPEG quality (0-100)"},
			command.BoolFlag{Name: "full-page", Description: "Capture the full scrollable page"},
		},
		Run: func(ctx context.Context, args json.RawMessage, _ command.Prompter) (*command.Result, error) {
			var p0 struct {
				URL      string `json:"url"`
				Format   string `json:"format"`
				Quality  int    `json:"quality"`
				FullPage bool   `json:"full-page"`
			}
			if err := json.Unmarshal(args, &p0); err != nil {
				return command.TextErrorResult(err.Error()), nil
			}

			return withSession(ctx, p0.URL, func(s cdp.Session) (io.ReadCloser, error) {
				return s.CaptureScreenshot(ctx, cdp.ScreenshotOptions{
					Format:   p0.Format,
					Quality:  p0.Quality,
					FullPage: p0.FullPage,
				})
			})
		},
	})

	// capture-mhtml
	app.AddCommand(&command.Command{
		Name:        "capture-mhtml",
		Description: command.Description{Short: "Capture a web page as MHTML archive"},
		Annotations: &protocol.ToolAnnotations{ReadOnlyHint: protocol.BoolPtr(true)},
		Params:      []command.Param{url},
		Run: func(ctx context.Context, args json.RawMessage, _ command.Prompter) (*command.Result, error) {
			var p0 struct {
				URL string `json:"url"`
			}
			if err := json.Unmarshal(args, &p0); err != nil {
				return command.TextErrorResult(err.Error()), nil
			}

			return withSession(ctx, p0.URL, func(s cdp.Session) (io.ReadCloser, error) {
				return s.CaptureSnapshot(ctx)
			})
		},
	})

	// capture-a11y
	app.AddCommand(&command.Command{
		Name:        "capture-a11y",
		Description: command.Description{Short: "Capture the accessibility tree of a web page"},
		Annotations: &protocol.ToolAnnotations{ReadOnlyHint: protocol.BoolPtr(true)},
		Params:      []command.Param{url},
		Run: func(ctx context.Context, args json.RawMessage, _ command.Prompter) (*command.Result, error) {
			var p0 struct {
				URL string `json:"url"`
			}
			if err := json.Unmarshal(args, &p0); err != nil {
				return command.TextErrorResult(err.Error()), nil
			}

			return withSession(ctx, p0.URL, func(s cdp.Session) (io.ReadCloser, error) {
				return s.AccessibilityTree(ctx)
			})
		},
	})

	// capture-text
	app.AddCommand(&command.Command{
		Name:        "capture-text",
		Description: command.Description{Short: "Extract plain text from a web page"},
		Annotations: &protocol.ToolAnnotations{ReadOnlyHint: protocol.BoolPtr(true)},
		Params:      []command.Param{url},
		Run: func(ctx context.Context, args json.RawMessage, _ command.Prompter) (*command.Result, error) {
			var p0 struct {
				URL string `json:"url"`
			}
			if err := json.Unmarshal(args, &p0); err != nil {
				return command.TextErrorResult(err.Error()), nil
			}

			return withSession(ctx, p0.URL, func(s cdp.Session) (io.ReadCloser, error) {
				return s.ExtractText(ctx)
			})
		},
	})
}

// withSession launches headless Chrome, navigates to the URL, runs the capture
// function, and returns the result as a command.Result.
func withSession(
	ctx context.Context,
	url string,
	capture func(cdp.Session) (io.ReadCloser, error),
) (*command.Result, error) {
	session, err := headless.NewSession(ctx)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	defer session.Close()

	if err := session.Navigate(ctx, url); err != nil {
		return nil, errors.Wrap(err)
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
