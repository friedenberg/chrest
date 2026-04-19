package capturebatch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"code.linenisgreat.com/chrest/go/src/bravo/cdp"
	"code.linenisgreat.com/chrest/go/src/charlie/firefox"
	"code.linenisgreat.com/chrest/go/src/charlie/headless"
	"code.linenisgreat.com/chrest/go/src/delta/tools"
)

// PayloadMediaTypes maps each supported capture format to the media
// type recorded on the payload ArtifactRef. RFC 0001 §Payload Artifact.
var PayloadMediaTypes = map[string]string{
	"text":       "text/plain; charset=utf-8",
	"pdf":        "application/pdf",
	"screenshot": "image/png",
	"mhtml":      "multipart/related",
	"a11y":       "application/json",
}

// Options configure the runner; most come from Input.
type Options struct {
	CapturerVersion string
	Writer          WriterSpec
	URL             string
	Defaults        *CaptureDefaults
}

// Run executes every capture in order and returns the batch output.
// The runner never fails fatally on per-capture errors — they become
// OutputCapture.Error entries. Batch-level failures (e.g. writer.cmd
// empty) are returned as errors.
func Run(ctx context.Context, inputCaptures []InputCapture, opts Options) (Output, error) {
	if len(opts.Writer.Cmd) == 0 {
		return Output{}, fmt.Errorf("writer.cmd MUST have at least one element")
	}
	if opts.URL == "" {
		return Output{}, fmt.Errorf("url MUST be a non-empty string")
	}

	host := GatherHost()

	out := Output{
		Schema: OutputSchema,
		Capturer: CapturerInfo{
			Name:    CapturerName,
			Version: opts.CapturerVersion,
		},
		Errors:   []Error{},
		Captures: make([]OutputCapture, 0, len(inputCaptures)),
	}

	for _, raw := range inputCaptures {
		resolved := Resolve(raw, opts.Defaults)
		out.Captures = append(out.Captures, runOne(ctx, resolved, opts, host))
	}

	return out, nil
}

func runOne(ctx context.Context, r Resolved, opts Options, host HostFingerprint) OutputCapture {
	entry := OutputCapture{Name: r.Name}

	if r.Split {
		entry.Error = &CaptureError{
			Kind:    "not-implemented",
			Message: "split=true normalization will land in follow-up",
		}
		return entry
	}

	mediaType, ok := PayloadMediaTypes[r.Format]
	if !ok {
		entry.Error = &CaptureError{
			Kind:    "bad-format",
			Message: fmt.Sprintf("unknown capture format %q", r.Format),
		}
		return entry
	}

	session, err := openSession(ctx, r.Browser)
	if err != nil {
		entry.Error = &CaptureError{Kind: "session-open-failed", Message: err.Error()}
		return entry
	}
	defer session.Close()

	if err := session.Navigate(ctx, opts.URL); err != nil {
		entry.Error = &CaptureError{Kind: "navigate-failed", Message: err.Error()}
		return entry
	}

	payloadRef, err := streamPayload(ctx, session, r, opts.Writer, mediaType)
	if err != nil {
		entry.Error = &CaptureError{Kind: "payload-write-failed", Message: err.Error()}
		return entry
	}
	entry.Payload = payloadRef

	specRef, err := writeSpec(ctx, session, r, host, opts)
	if err != nil {
		entry.Error = &CaptureError{Kind: "spec-write-failed", Message: err.Error()}
		return entry
	}
	entry.Spec = specRef

	return entry
}

func openSession(ctx context.Context, browser string) (cdp.Session, error) {
	switch browser {
	case "firefox":
		return firefox.NewSession(ctx)
	case "chrome", "":
		return headless.NewSession(ctx)
	default:
		return nil, fmt.Errorf("unknown browser backend %q", browser)
	}
}

func streamPayload(
	ctx context.Context,
	session cdp.Session,
	r Resolved,
	writer WriterSpec,
	mediaType string,
) (*ArtifactRef, error) {
	rc, err := runCaptureFormat(ctx, session, r)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	res, err := WriteThrough(ctx, writer.Cmd, rc)
	if err != nil {
		return nil, err
	}
	return &ArtifactRef{
		ID:        res.ID,
		Size:      res.Size,
		MediaType: mediaType,
	}, nil
}

// runCaptureFormat dispatches to the session method matching format.
// Mirrors tools.runCapture but operates directly on a cdp.Session —
// capture-batch holds the session over multiple operations (navigate,
// capture, BrowserInfo) so it doesn't use tools.StreamCapture wholesale.
func runCaptureFormat(ctx context.Context, s cdp.Session, r Resolved) (io.ReadCloser, error) {
	var opts tools.CaptureParams
	if len(r.Options) > 0 {
		_ = json.Unmarshal(r.Options, &opts) // best-effort field copy
	}
	switch r.Format {
	case "text":
		return s.ExtractText(ctx)
	case "pdf":
		return s.PrintToPDF(ctx, cdp.PDFOptions{
			Landscape:           opts.Landscape,
			DisplayHeaderFooter: !opts.NoHeaders,
			PrintBackground:     opts.Background,
		})
	case "screenshot":
		return s.CaptureScreenshot(ctx, cdp.ScreenshotOptions{
			Format:   "png",
			FullPage: opts.FullPage,
		})
	case "mhtml":
		return s.CaptureSnapshot(ctx)
	case "a11y":
		return s.AccessibilityTree(ctx)
	default:
		return nil, fmt.Errorf("unknown capture format %q", r.Format)
	}
}

func writeSpec(
	ctx context.Context,
	session cdp.Session,
	r Resolved,
	host HostFingerprint,
	opts Options,
) (*ArtifactRef, error) {
	browserInfo, _ := session.BrowserInfo(ctx) // best-effort; empty fields are fine
	// Echo the request's browser label if the session didn't populate it
	// (extension-proxied sessions currently return BrowserInfo{}).
	if browserInfo.Name == "" {
		browserInfo.Name = r.Browser
	}

	specBytes, err := BuildSpec(r, browserInfo, host, opts.CapturerVersion)
	if err != nil {
		return nil, fmt.Errorf("build spec: %w", err)
	}

	res, err := WriteThrough(ctx, opts.Writer.Cmd, bytes.NewReader(specBytes))
	if err != nil {
		return nil, err
	}
	return &ArtifactRef{
		ID:        res.ID,
		Size:      res.Size,
		MediaType: SpecMediaType,
	}, nil
}
