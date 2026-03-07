package tools

import (
	"context"
	"encoding/json"

	"code.linenisgreat.com/chrest/go/src/delta/proxy"
	"github.com/amarbel-llc/purse-first/libs/go-mcp/command"
	"github.com/amarbel-llc/purse-first/libs/go-mcp/protocol"
)

func registerTabCommands(app *command.App, p *proxy.BrowserProxy) {
	app.AddCommand(&command.Command{
		Name:        "list-tabs",
		Description: command.Description{Short: "List all browser tabs"},
		Annotations: &protocol.ToolAnnotations{ReadOnlyHint: protocol.BoolPtr(true)},
		Run: func(ctx context.Context, args json.RawMessage, _ command.Prompter) (*command.Result, error) {
			result, err := p.RequestAllBrowsers(ctx, "GET", "/tabs", nil)
			if err != nil {
				return nil, err
			}
			return command.TextResult(result), nil
		},
	})

	app.AddCommand(&command.Command{
		Name:        "get-tab",
		Description: command.Description{Short: "Get a specific browser tab by ID"},
		Annotations: &protocol.ToolAnnotations{ReadOnlyHint: protocol.BoolPtr(true)},
		Params: []command.Param{
			{Name: "tab_id", Type: command.String, Required: true, Description: "Tab ID"},
		},
		Run: func(ctx context.Context, args json.RawMessage, _ command.Prompter) (*command.Result, error) {
			var p0 struct {
				TabID string `json:"tab_id"`
			}
			if err := json.Unmarshal(args, &p0); err != nil {
				return command.TextErrorResult(err.Error()), nil
			}
			result, err := p.RequestAllBrowsers(ctx, "GET", "/tabs/"+p0.TabID, nil)
			if err != nil {
				return nil, err
			}
			return command.TextResult(result), nil
		},
	})

	app.AddCommand(&command.Command{
		Name:        "create-tab",
		Description: command.Description{Short: "Create a new browser tab"},
		Params: []command.Param{
			{Name: "url", Type: command.String, Required: true, Description: "URL to open"},
			{Name: "window_id", Type: command.String, Description: "Window ID to create the tab in"},
			{Name: "active", Type: command.Bool, Description: "Whether the tab should be active"},
		},
		Run: func(ctx context.Context, args json.RawMessage, _ command.Prompter) (*command.Result, error) {
			var p0 struct {
				URL      string `json:"url"`
				WindowID string `json:"window_id"`
				Active   bool   `json:"active"`
			}
			if err := json.Unmarshal(args, &p0); err != nil {
				return command.TextErrorResult(err.Error()), nil
			}

			body := map[string]any{
				"url": p0.URL,
			}
			if p0.WindowID != "" {
				body["windowId"] = p0.WindowID
			}
			if p0.Active {
				body["active"] = true
			}

			result, err := p.RequestAllBrowsers(ctx, "POST", "/tabs", body)
			if err != nil {
				return nil, err
			}
			return command.TextResult(result), nil
		},
	})

	app.AddCommand(&command.Command{
		Name:        "update-tab",
		Description: command.Description{Short: "Update a browser tab"},
		Params: []command.Param{
			{Name: "tab_id", Type: command.String, Required: true, Description: "Tab ID"},
			{Name: "url", Type: command.String, Description: "New URL for the tab"},
			{Name: "active", Type: command.Bool, Description: "Whether the tab should be active"},
		},
		Run: func(ctx context.Context, args json.RawMessage, _ command.Prompter) (*command.Result, error) {
			var p0 struct {
				TabID  string `json:"tab_id"`
				URL    string `json:"url"`
				Active bool   `json:"active"`
			}
			if err := json.Unmarshal(args, &p0); err != nil {
				return command.TextErrorResult(err.Error()), nil
			}

			body := map[string]any{}
			if p0.URL != "" {
				body["url"] = p0.URL
			}
			if p0.Active {
				body["active"] = true
			}

			result, err := p.RequestAllBrowsers(ctx, "PATCH", "/tabs/"+p0.TabID, body)
			if err != nil {
				return nil, err
			}
			return command.TextResult(result), nil
		},
	})

	app.AddCommand(&command.Command{
		Name:        "close-tab",
		Description: command.Description{Short: "Close a browser tab"},
		Annotations: &protocol.ToolAnnotations{DestructiveHint: protocol.BoolPtr(true)},
		Params: []command.Param{
			{Name: "tab_id", Type: command.String, Required: true, Description: "Tab ID"},
		},
		Run: func(ctx context.Context, args json.RawMessage, _ command.Prompter) (*command.Result, error) {
			var p0 struct {
				TabID string `json:"tab_id"`
			}
			if err := json.Unmarshal(args, &p0); err != nil {
				return command.TextErrorResult(err.Error()), nil
			}
			result, err := p.RequestAllBrowsers(ctx, "DELETE", "/tabs/"+p0.TabID, nil)
			if err != nil {
				return nil, err
			}
			return command.TextResult(result), nil
		},
	})
}
