package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"

	"code.linenisgreat.com/chrest/go/libs/dewey/golf/command"
	"code.linenisgreat.com/chrest/go/libs/dewey/golf/protocol"
	"code.linenisgreat.com/chrest/go/libs/dewey/golf/server"
	"code.linenisgreat.com/chrest/go/libs/dewey/golf/transport"
	"code.linenisgreat.com/chrest/go/src/alfa/prompter"

	"code.linenisgreat.com/chrest/go/libs/dewey/0/stack_frame"
	"code.linenisgreat.com/chrest/go/libs/dewey/bravo/errors"
	"code.linenisgreat.com/chrest/go/libs/dewey/charlie/ui"
	"code.linenisgreat.com/chrest/go/src/bravo/config"
	"code.linenisgreat.com/chrest/go/src/charlie/browser_items"
	"code.linenisgreat.com/chrest/go/src/charlie/markdown"
	"code.linenisgreat.com/chrest/go/src/delta/proxy"
	"code.linenisgreat.com/chrest/go/src/delta/resources"
	"code.linenisgreat.com/chrest/go/src/delta/tools"
)

func init() {
	log.SetPrefix("chrest ")
}

func main() {
	ctx := errors.MakeContextDefault()
	ctx.SetCancelOnSignals(syscall.SIGTERM)

	if err := ctx.Run(
		func(ctx errors.Context) {
			if err := run(ctx); err != nil {
				ctx.Cancel(err)
			}
		},
	); err != nil {
		var normalError stack_frame.ErrorStackTracer

		if errors.As(err, &normalError) && !normalError.ShouldShowStackTrace() {
			ui.Err().Printf("%s", normalError.Error())
		} else {
			ui.Err().Print(err)
		}
		// Fall through to a non-zero exit so shell callers and CI can
		// detect the failure. Without this, `chrest capture ... > out`
		// quietly produced a zero-byte file with exit 0 when the browser
		// binary was missing.
		os.Exit(1)
	}
}

func run(ctx errors.Context) (err error) {
	var c config.Config

	if c, err = config.Default(); err != nil {
		err = errors.Wrap(err)
		return
	}

	if err = c.Read(); err != nil {
		err = errors.Wrap(err)
		return
	}

	p := &proxy.BrowserProxy{Config: c}

	app := command.NewUtility("chrest", "Manage browsers via REST")
	app.Version = "0.1.0"
	app.MCPArgs = []string{"mcp"}

	tools.RegisterAll(app, p)
	registerClientCommand(app, c)
	registerReloadExtensionCommand(app, c)
	registerInitCommand(app)
	registerInstallMCPCommand(app)
	registerGeneratePluginCommand(app)
	registerCaptureBatchCommand(app)

	if len(os.Args) > 1 && os.Args[1] == "mcp" {
		if err = runMCP(ctx, app, p); err != nil {
			err = errors.Wrap(err)
			return
		}
		return
	}

	// Bypass dewey for client command — variadic StringArg
	// serialization is broken (purse-first#44)
	if len(os.Args) > 1 && os.Args[1] == "client" {
		if err = cmdClient(c, os.Getenv("CHREST_BROWSER"), false, os.Args[2:]); err != nil {
			err = errors.Wrap(err)
			return
		}
		return
	}

	// Bypass dewey for init command — flag parsing may also
	// be affected (purse-first#44)
	if len(os.Args) > 1 && os.Args[1] == "init" {
		if err = cmdInitDirect(ctx); err != nil {
			err = errors.Wrap(err)
			return
		}
		return
	}

	// Bypass dewey for capture: the Result/TextResult path in golf/command
	// buffers everything and appends a trailing newline via fmt.Println,
	// which corrupts binary output and bloats memory. See chrest#21 and the
	// upstream dewey gap in amarbel-llc/purse-first#55.
	if len(os.Args) > 1 && os.Args[1] == "capture" {
		if err = cmdCapture(ctx, p, os.Args[2:]); err != nil {
			err = errors.Wrap(err)
			return
		}
		return
	}

	// Bypass dewey for capture-batch: the contract is JSON-on-stdin and
	// JSON-on-stdout per RFC 0001, neither of which fits the Result path.
	// Batch-level failures surface as non-zero exit via the outer error
	// handler in main() so the orchestrator distinguishes them from
	// per-capture errors in the output JSON.
	if len(os.Args) > 1 && os.Args[1] == "capture-batch" {
		if err = cmdCaptureBatch(ctx, app.Version, os.Args[2:]); err != nil {
			err = errors.Wrap(err)
			return
		}
		return
	}

	if err = app.RunCLI(ctx, os.Args[1:], prompter.Prompter{}); err != nil {
		err = errors.Wrap(err)
		return
	}

	return
}

func runMCP(ctx context.Context, app *command.Utility, p *proxy.BrowserProxy) error {
	t := transport.NewStdio(os.Stdin, os.Stdout)
	registry := server.NewToolRegistryV1()
	app.RegisterMCPToolsV1(registry)

	itemsProxy := browser_items.BrowserProxy{Config: p.Config}
	itemResources := resources.NewItemResources(p, itemsProxy)

	const (
		mimeText     = "text/plain; charset=utf-8"
		mimeMarkdown = "text/markdown; charset=utf-8"
		mimeHTML     = "text/html; charset=utf-8"
	)

	type fetchCacheEntry struct {
		HTML           []byte
		Text           []byte
		MarkdownReader []byte
		TOC            []markdown.Heading
		FetchedAt      time.Time
	}

	var fetchCache sync.Map // URL (no fragment) → *fetchCacheEntry

	// Bridge tool: exposes resources as a tool for subagent access
	registry.Register(
		protocol.ToolV1{
			Name:        "read-resource",
			Description: "Read a chrest resource by URI (e.g. chrest://items, chrest://items/1)",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"uri":{"type":"string","description":"Resource URI to read"}},"required":["uri"]}`),
			Annotations: &protocol.ToolAnnotations{ReadOnlyHint: protocol.BoolPtr(true)},
		},
		func(ctx context.Context, args json.RawMessage) (*protocol.ToolCallResultV1, error) {
			var p0 struct {
				URI string `json:"uri"`
			}
			if err := json.Unmarshal(args, &p0); err != nil {
				return protocol.ErrorResultV1(err.Error()), nil
			}
			if strings.HasPrefix(p0.URI, "web-fetch://") {
				url, fragment := splitWebFetchURI(p0.URI)
				if url == "" {
					return protocol.ErrorResultV1("invalid web-fetch URI: " + p0.URI), nil
				}
				v, ok := fetchCache.Load(url)
				if !ok {
					return protocol.ErrorResultV1("resource not found (page may need to be fetched first): " + p0.URI), nil
				}
				entry := v.(*fetchCacheEntry)
				switch fragment {
				case "text":
					return &protocol.ToolCallResultV1{
						Content: []protocol.ContentBlockV1{
							protocol.TextContentV1(string(entry.Text)),
						},
					}, nil
				case "markdown":
					return &protocol.ToolCallResultV1{
						Content: []protocol.ContentBlockV1{
							protocol.TextContentV1(string(entry.MarkdownReader)),
						},
					}, nil
				case "html":
					return &protocol.ToolCallResultV1{
						Content: []protocol.ContentBlockV1{
							protocol.TextContentV1(string(entry.HTML)),
						},
					}, nil
				case "toc":
					return &protocol.ToolCallResultV1{
						Content: []protocol.ContentBlockV1{
							protocol.TextContentV1(markdown.FormatTOC(entry.TOC, url)),
						},
					}, nil
				case "markdown-selector":
					return protocol.ErrorResultV1("selector-derived resources are not re-readable; call web-fetch with the selector arg to regenerate"), nil
				default:
					return protocol.ErrorResultV1("unknown web-fetch fragment: " + fragment + " (expected text, markdown, html, or toc)"), nil
				}
			}
			result, err := itemResources.ReadResource(ctx, p0.URI)
			if err != nil {
				return protocol.ErrorResultV1(err.Error()), nil
			}
			if len(result.Contents) == 0 {
				return protocol.ErrorResultV1("no content"), nil
			}
			return &protocol.ToolCallResultV1{
				Content: []protocol.ContentBlockV1{
					protocol.TextContentV1(result.Contents[0].Text),
				},
			}, nil
		},
	)

	registry.Register(
		protocol.ToolV1{
			Name: "web-fetch",
			Description: "Fetch a web page via headless Firefox and return its content " +
				"as reader-mode Markdown (default). All three base formats (text, markdown, html) " +
				"are always rendered; non-selected formats are returned as resource_link URIs " +
				"that can be read via read-resource. The first content block is always a TOC " +
				"listing every `#id` anchor found on the page — pass one of those ids as " +
				"`selector` to trim the returned markdown to that section only. Results are " +
				"cached per URL for the lifetime of the MCP session; pass `refresh: true` to " +
				"force a re-fetch. Use instead of the built-in WebFetch when you need complete, " +
				"unsummarized content or when the page requires JavaScript to render.",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"url":{"type":"string","description":"URL to fetch"},"format":{"type":"string","description":"Format to return inline: 'markdown' (default), 'text', or 'html'. Other formats are returned as resource_link URIs.","enum":["markdown","text","html"]},"selector":{"type":"string","description":"Optional CSS selector to trim the returned markdown to a specific section (e.g. '#chap-stdenv', 'article#main'). Requires format=markdown. On a miss, the tool returns a diagnostic plus a resource_link to the full page - it never silently returns empty. The TOC content block (always first) lists the #id selectors available on this page."},"refresh":{"type":"boolean","description":"Force a re-fetch even if this URL was fetched earlier in the session. Default false: reuse cached HTML. Selector-only re-runs against the same URL never relaunch Firefox."}},"required":["url"]}`),
			Annotations: &protocol.ToolAnnotations{ReadOnlyHint: protocol.BoolPtr(true)},
		},
		func(ctx context.Context, args json.RawMessage) (*protocol.ToolCallResultV1, error) {
			var p0 struct {
				URL      string `json:"url"`
				Format   string `json:"format"`
				Selector string `json:"selector"`
				Refresh  bool   `json:"refresh"`
			}
			if err := json.Unmarshal(args, &p0); err != nil {
				return protocol.ErrorResultV1(err.Error()), nil
			}
			if p0.Format == "" {
				p0.Format = "markdown"
			}
			if p0.Selector != "" && p0.Format != "markdown" {
				return protocol.ErrorResultV1("selector is only supported with format=markdown; got format=" + p0.Format), nil
			}

			var entry *fetchCacheEntry
			if !p0.Refresh {
				if v, ok := fetchCache.Load(p0.URL); ok {
					entry = v.(*fetchCacheEntry)
				}
			}
			if entry == nil {
				results, err := tools.MultiExtract(ctx, p, tools.MultiExtractParams{
					URL:     p0.URL,
					Formats: []string{"text", "markdown-reader", "html-outer"},
				})
				if err != nil {
					return protocol.ErrorResultV1(err.Error()), nil
				}

				entry = &fetchCacheEntry{FetchedAt: time.Now()}
				var errs []string
				for _, r := range results {
					if r.Err != nil {
						errs = append(errs, fmt.Sprintf("%s: %s", r.Format, r.Err))
						continue
					}
					switch r.Format {
					case "text":
						entry.Text = r.Data
					case "markdown-reader":
						entry.MarkdownReader = r.Data
					case "html-outer":
						entry.HTML = r.Data
					}
				}
				if entry.Text == nil && entry.MarkdownReader == nil && entry.HTML == nil {
					return protocol.ErrorResultV1("all formats failed: " + strings.Join(errs, "; ")), nil
				}
				if entry.HTML != nil {
					toc, tocErr := markdown.ExtractTOC(bytes.NewReader(entry.HTML))
					if tocErr != nil {
						log.Printf("web-fetch: ExtractTOC failed for %s: %v", p0.URL, tocErr)
					} else {
						entry.TOC = toc
					}
				}
				fetchCache.Store(p0.URL, entry)
			}

			tocBlock := protocol.TextContentV1(markdown.FormatTOC(entry.TOC, p0.URL))

			textURI := fmt.Sprintf("web-fetch://%s#text", p0.URL)
			markdownURI := fmt.Sprintf("web-fetch://%s#markdown", p0.URL)
			htmlURI := fmt.Sprintf("web-fetch://%s#html", p0.URL)

			if p0.Selector == "" {
				blocks := []protocol.ContentBlockV1{tocBlock}
				switch p0.Format {
				case "markdown":
					blocks = append(blocks,
						protocol.EmbeddedTextResourceContent(markdownURI, string(entry.MarkdownReader), mimeMarkdown),
						protocol.ResourceLinkContent(textURI, "Plain text", "", mimeText),
						protocol.ResourceLinkContent(htmlURI, "Raw HTML", "", mimeHTML),
					)
				case "text":
					blocks = append(blocks,
						protocol.EmbeddedTextResourceContent(textURI, string(entry.Text), mimeText),
						protocol.ResourceLinkContent(markdownURI, "Reader-mode Markdown", "", mimeMarkdown),
						protocol.ResourceLinkContent(htmlURI, "Raw HTML", "", mimeHTML),
					)
				case "html":
					blocks = append(blocks,
						protocol.EmbeddedTextResourceContent(htmlURI, string(entry.HTML), mimeHTML),
						protocol.ResourceLinkContent(markdownURI, "Reader-mode Markdown", "", mimeMarkdown),
						protocol.ResourceLinkContent(textURI, "Plain text", "", mimeText),
					)
				default:
					return protocol.ErrorResultV1("unknown format: " + p0.Format), nil
				}
				return &protocol.ToolCallResultV1{Content: blocks}, nil
			}

			// Selector path. Use the section-expanding variant so an
			// `#id` selector that matches a heading returns the whole
			// section (heading + following siblings up to the next
			// heading of equal-or-higher level) rather than just the
			// heading element.
			rc, err := markdown.ConvertSelectorSection(bytes.NewReader(entry.HTML), p0.Selector)
			if err != nil {
				if errors.Is(err, markdown.ErrSelectorNoMatch) {
					diag := fmt.Sprintf(
						"selector %q matched no element on %s.\nThe TOC above lists the anchors that are present on this page.\nFull markdown is available via the resource_link below; call read-resource on web-fetch://%s#markdown to fetch it.",
						p0.Selector, p0.URL, p0.URL,
					)
					return &protocol.ToolCallResultV1{
						Content: []protocol.ContentBlockV1{
							tocBlock,
							protocol.TextContentV1(diag),
							protocol.ResourceLinkContent(markdownURI, "Reader-mode Markdown", "", mimeMarkdown),
							protocol.ResourceLinkContent(textURI, "Plain text", "", mimeText),
							protocol.ResourceLinkContent(htmlURI, "Raw HTML", "", mimeHTML),
						},
					}, nil
				}
				return protocol.ErrorResultV1(err.Error()), nil
			}
			trimmed, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return protocol.ErrorResultV1(err.Error()), nil
			}

			selectorURI := fmt.Sprintf("web-fetch://%s#markdown-selector", p0.URL)
			return &protocol.ToolCallResultV1{
				Content: []protocol.ContentBlockV1{
					tocBlock,
					protocol.EmbeddedTextResourceContent(selectorURI, string(trimmed), mimeMarkdown),
					protocol.ResourceLinkContent(markdownURI, "Reader-mode Markdown", "", mimeMarkdown),
					protocol.ResourceLinkContent(textURI, "Plain text", "", mimeText),
					protocol.ResourceLinkContent(htmlURI, "Raw HTML", "", mimeHTML),
				},
			}, nil
		},
	)

	srv, err := server.New(t, server.Options{
		ServerName:    app.Name,
		ServerVersion: app.Version,
		Tools:         registry,
		Resources:     itemResources,
	})
	if err != nil {
		return errors.Wrap(err)
	}

	return srv.Run(ctx)
}

// splitWebFetchURI parses "web-fetch://<url>#<fragment>" into its url and
// fragment halves. The URL itself may contain '#' characters if the caller
// passed a fragmented URL, so the last '#' is treated as the delimiter
// (matching the format emitted by the web-fetch handler). Returns empty
// strings if the URI doesn't carry the web-fetch:// prefix or lacks a '#'.
func splitWebFetchURI(uri string) (url, fragment string) {
	const prefix = "web-fetch://"
	if !strings.HasPrefix(uri, prefix) {
		return "", ""
	}
	rest := uri[len(prefix):]
	idx := strings.LastIndex(rest, "#")
	if idx < 0 {
		return "", ""
	}
	return rest[:idx], rest[idx+1:]
}
