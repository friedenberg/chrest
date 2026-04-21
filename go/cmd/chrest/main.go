package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"syscall"

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

	var fetchCache sync.Map // web-fetch:// URI → string content

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
				if v, ok := fetchCache.Load(p0.URI); ok {
					return &protocol.ToolCallResultV1{
						Content: []protocol.ContentBlockV1{
							protocol.TextContentV1(v.(string)),
						},
					}, nil
				}
				return protocol.ErrorResultV1("resource not found (page may need to be fetched first): " + p0.URI), nil
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

	type webFetchFormat struct {
		capture  string
		fragment string
		mime     string
		name     string
	}
	allFormats := []webFetchFormat{
		{"text", "text", "text/plain; charset=utf-8", "Plain text"},
		{"markdown-reader", "markdown", "text/markdown; charset=utf-8", "Reader-mode Markdown"},
		{"html-outer", "html", "text/html; charset=utf-8", "Raw HTML"},
	}

	registry.Register(
		protocol.ToolV1{
			Name: "web-fetch",
			Description: "Fetch a web page via headless Firefox and return its content " +
				"as reader-mode Markdown (default). All three formats (text, markdown, html) " +
				"are always rendered; non-selected formats are returned as resource URIs " +
				"that can be read via read-resource. Returns the full page — no summarization. " +
				"Use instead of the built-in WebFetch when you need complete, unsummarized " +
				"content or when the page requires JavaScript to render.",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"url":{"type":"string","description":"URL to fetch"},"format":{"type":"string","description":"Format to return inline: 'markdown' (default), 'text', or 'html'. Other formats are returned as resource_link URIs.","enum":["markdown","text","html"]}},"required":["url"]}`),
			Annotations: &protocol.ToolAnnotations{ReadOnlyHint: protocol.BoolPtr(true)},
		},
		func(ctx context.Context, args json.RawMessage) (*protocol.ToolCallResultV1, error) {
			var p0 struct {
				URL    string `json:"url"`
				Format string `json:"format"`
			}
			if err := json.Unmarshal(args, &p0); err != nil {
				return protocol.ErrorResultV1(err.Error()), nil
			}
			if p0.Format == "" {
				p0.Format = "markdown"
			}

			captures := make([]string, len(allFormats))
			for i, f := range allFormats {
				captures[i] = f.capture
			}

			results, err := tools.MultiExtract(ctx, p, tools.MultiExtractParams{
				URL:     p0.URL,
				Formats: captures,
			})
			if err != nil {
				return protocol.ErrorResultV1(err.Error()), nil
			}

			var blocks []protocol.ContentBlockV1
			var errs []string
			for i, r := range results {
				f := allFormats[i]
				uri := fmt.Sprintf("web-fetch://%s#%s", p0.URL, f.fragment)
				if r.Err != nil {
					errs = append(errs, fmt.Sprintf("%s: %s", r.Format, r.Err))
					continue
				}
				content := string(r.Data)
				fetchCache.Store(uri, content)
				if f.fragment == p0.Format {
					blocks = append(blocks, protocol.EmbeddedTextResourceContent(uri, content, f.mime))
				} else {
					blocks = append(blocks, protocol.ResourceLinkContent(uri, f.name, "", f.mime))
				}
			}
			if len(errs) > 0 {
				blocks = append(blocks,
					protocol.TextContentV1("partial errors: "+strings.Join(errs, "; ")),
				)
			}
			if len(blocks) == 0 {
				return protocol.ErrorResultV1("all formats failed: " + strings.Join(errs, "; ")), nil
			}

			return &protocol.ToolCallResultV1{Content: blocks}, nil
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
