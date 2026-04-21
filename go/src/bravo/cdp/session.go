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
	// GetDocumentHTML returns the rendered DOM serialized as
	// document.documentElement.outerHTML. Backends that do not support
	// script evaluation return a wrapped "not supported" error.
	GetDocumentHTML(ctx context.Context) (io.ReadCloser, error)
	// BrowserInfo returns identity fields for the live browser. Used by
	// capture-batch to populate the spec artifact fingerprint. Fields
	// may be empty if the backend didn't advertise them; callers treat
	// empty strings as "not available."
	BrowserInfo(ctx context.Context) (BrowserInfo, error)
	// SetViewport resizes the browsing context viewport before
	// navigation. Width and height are in CSS pixels. Backends that
	// do not support viewport control return an error.
	SetViewport(ctx context.Context, width, height int) error
	// LastNavigationHTTP returns the HTTP response observed for the
	// top-level document on the most recent Navigate call. Second
	// return is false when no response was observed (no navigate yet,
	// or the backend doesn't support event capture). Used by
	// capture-batch to populate envelope http.* fields per RFC 0001.
	LastNavigationHTTP() (HTTPResponse, bool)
	Close() error
}

// HTTPResponse describes the top-level document response captured
// during a Navigate. Populated by backends that support network-event
// subscription (Firefox/BiDi today). Per RFC 0001 §Envelope Artifact
// this becomes the envelope's `http.*` fields.
type HTTPResponse struct {
	URL      string       // final URL after redirects
	Status   int          // HTTP status code of the final response
	Headers  []HTTPHeader // preserves order and duplicates
	TimingMs int64        // responseEnd - fetchStart in ms; 0 if not measured
}

// HTTPHeader is a single response header. Represented as a list of
// name/value pairs rather than a map so the envelope preserves both
// header order and duplicates (Set-Cookie commonly appears multiple
// times).
type HTTPHeader struct {
	Name  string
	Value string
}

// BrowserInfo is the identity surface of the live browser.
type BrowserInfo struct {
	Name        string // "firefox" | "chrome"
	Version     string
	UserAgent   string
	Platform    string
	JSEngine    string   // "SpiderMonkey" | "V8"
	CommandLine []string // argv the browser was launched with (excludes argv[0]); may be nil
}

type PDFOptions struct {
	Landscape           bool     `json:"landscape,omitempty"`
	DisplayHeaderFooter bool     `json:"displayHeaderFooter,omitempty"`
	PrintBackground     bool     `json:"printBackground,omitempty"`
	PaperWidth          *float64 `json:"paperWidth,omitempty"`
	PaperHeight         *float64 `json:"paperHeight,omitempty"`
	MarginTop           *float64 `json:"marginTop,omitempty"`
	MarginBottom        *float64 `json:"marginBottom,omitempty"`
	MarginLeft          *float64 `json:"marginLeft,omitempty"`
	MarginRight         *float64 `json:"marginRight,omitempty"`
	PageRanges          string   `json:"pageRanges,omitempty"`
}

type ScreenshotOptions struct {
	Format   string `json:"format,omitempty"`
	Quality  int    `json:"quality,omitempty"`
	FullPage bool   `json:"-"`
}
