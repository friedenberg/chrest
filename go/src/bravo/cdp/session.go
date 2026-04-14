package cdp

import (
	"context"
	"io"
)

// Session abstracts a CDP connection. Implementations include headless Chrome
// (charlie/headless) and, in the future, extension debugger proxy.
type Session interface {
	Navigate(ctx context.Context, url string) error
	PrintToPDF(ctx context.Context, opts PDFOptions) (io.ReadCloser, error)
	CaptureScreenshot(ctx context.Context, opts ScreenshotOptions) (io.ReadCloser, error)
	CaptureSnapshot(ctx context.Context) (io.ReadCloser, error)
	AccessibilityTree(ctx context.Context) (io.ReadCloser, error)
	ExtractText(ctx context.Context) (io.ReadCloser, error)
	Close() error
}

type PDFOptions struct {
	Landscape           bool    `json:"landscape,omitempty"`
	DisplayHeaderFooter bool    `json:"displayHeaderFooter,omitempty"`
	PrintBackground     bool    `json:"printBackground,omitempty"`
	PaperWidth          float64 `json:"paperWidth,omitempty"`
	PaperHeight         float64 `json:"paperHeight,omitempty"`
	MarginTop           float64 `json:"marginTop,omitempty"`
	MarginBottom        float64 `json:"marginBottom,omitempty"`
	MarginLeft          float64 `json:"marginLeft,omitempty"`
	MarginRight         float64 `json:"marginRight,omitempty"`
	PageRanges          string  `json:"pageRanges,omitempty"`
}

type ScreenshotOptions struct {
	Format   string `json:"format,omitempty"`
	Quality  int    `json:"quality,omitempty"`
	FullPage bool   `json:"-"`
}
