package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"

	"code.linenisgreat.com/chrest/go/src/alfa/prompter"
	"code.linenisgreat.com/chrest/go/libs/dewey/golf/command"
	"code.linenisgreat.com/chrest/go/libs/dewey/golf/protocol"
	"code.linenisgreat.com/chrest/go/libs/dewey/golf/server"
	"code.linenisgreat.com/chrest/go/libs/dewey/golf/transport"

	"code.linenisgreat.com/chrest/go/src/bravo/config"
	"code.linenisgreat.com/chrest/go/src/charlie/browser_items"
	"code.linenisgreat.com/chrest/go/src/delta/proxy"
	"code.linenisgreat.com/chrest/go/src/delta/resources"
	"code.linenisgreat.com/chrest/go/src/delta/tools"
	"code.linenisgreat.com/chrest/go/libs/dewey/0/stack_frame"
	"code.linenisgreat.com/chrest/go/libs/dewey/bravo/errors"
	"code.linenisgreat.com/chrest/go/libs/dewey/charlie/ui"
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
				"as plain text, reader-mode Markdown, and raw HTML. Returns the full " +
				"page \u2014 no summarization. Use instead of the built-in WebFetch when " +
				"you need complete, unsummarized content or when the page requires " +
				"JavaScript to render.",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"url":{"type":"string","description":"URL to fetch"}},"required":["url"]}`),
			Annotations: &protocol.ToolAnnotations{ReadOnlyHint: protocol.BoolPtr(true)},
		},
		func(ctx context.Context, args json.RawMessage) (*protocol.ToolCallResultV1, error) {
			var p0 struct {
				URL string `json:"url"`
			}
			if err := json.Unmarshal(args, &p0); err != nil {
				return protocol.ErrorResultV1(err.Error()), nil
			}

			results, err := tools.MultiExtract(ctx, p, tools.MultiExtractParams{
				URL:     p0.URL,
				Formats: []string{"text", "markdown-reader", "html-outer"},
			})
			if err != nil {
				return protocol.ErrorResultV1(err.Error()), nil
			}

			type fmtMeta struct {
				fragment string
				mime     string
			}
			meta := []fmtMeta{
				{"text", "text/plain; charset=utf-8"},
				{"markdown", "text/markdown; charset=utf-8"},
				{"html", "text/html; charset=utf-8"},
			}

			var blocks []protocol.ContentBlockV1
			var errs []string
			for i, r := range results {
				if r.Err != nil {
					errs = append(errs, fmt.Sprintf("%s: %s", r.Format, r.Err))
					continue
				}
				uri := fmt.Sprintf("web-fetch://%s#%s", p0.URL, meta[i].fragment)
				blocks = append(blocks, protocol.EmbeddedTextResourceContent(
					uri, string(r.Data), meta[i].mime,
				))
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
