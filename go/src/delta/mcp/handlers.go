package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"sync"

	"code.linenisgreat.com/chrest/go/src/charlie/browser_items"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type GetWindowArgs struct {
	WindowID string `json:"window_id"`
}

type CreateWindowArgs struct {
	URLs      []string `json:"urls,omitempty"`
	Focused   bool     `json:"focused,omitempty"`
	Incognito bool     `json:"incognito,omitempty"`
}

type UpdateWindowArgs struct {
	WindowID string `json:"window_id"`
	Focused  bool   `json:"focused,omitempty"`
	State    string `json:"state,omitempty"`
}

type CloseWindowArgs struct {
	WindowID string `json:"window_id"`
}

type GetTabArgs struct {
	TabID string `json:"tab_id"`
}

type CreateTabArgs struct {
	URL      string `json:"url"`
	WindowID string `json:"window_id,omitempty"`
	Active   bool   `json:"active,omitempty"`
}

type UpdateTabArgs struct {
	TabID  string `json:"tab_id"`
	URL    string `json:"url,omitempty"`
	Active bool   `json:"active,omitempty"`
}

type CloseTabArgs struct {
	TabID string `json:"tab_id"`
}

type ManageItemsArgs struct {
	Added   []ItemArg `json:"added,omitempty"`
	Deleted []ItemArg `json:"deleted,omitempty"`
	Focused []ItemArg `json:"focused,omitempty"`
}

type ItemArg struct {
	ID    string `json:"id,omitempty"`
	URL   string `json:"url,omitempty"`
	Title string `json:"title,omitempty"`
}

type RestoreStateArgs struct {
	State json.RawMessage `json:"state"`
}

func (s *Server) handleBrowserInfo(
	ctx context.Context,
	req *mcp.CallToolRequest,
	_ any,
) (*mcp.CallToolResult, any, error) {
	result, err := s.requestAllBrowsers(ctx, "GET", "/", nil)
	if err != nil {
		return nil, nil, err
	}
	return textResult(result), nil, nil
}

func (s *Server) handleListExtensions(
	ctx context.Context,
	req *mcp.CallToolRequest,
	_ any,
) (*mcp.CallToolResult, any, error) {
	result, err := s.requestAllBrowsers(ctx, "GET", "/extensions", nil)
	if err != nil {
		return nil, nil, err
	}
	return textResult(result), nil, nil
}

func (s *Server) handleListWindows(
	ctx context.Context,
	req *mcp.CallToolRequest,
	_ any,
) (*mcp.CallToolResult, any, error) {
	result, err := s.requestAllBrowsers(ctx, "GET", "/windows", nil)
	if err != nil {
		return nil, nil, err
	}
	return textResult(result), nil, nil
}

func (s *Server) handleGetWindow(
	ctx context.Context,
	req *mcp.CallToolRequest,
	args GetWindowArgs,
) (*mcp.CallToolResult, any, error) {
	result, err := s.requestAllBrowsers(ctx, "GET", "/windows/"+args.WindowID, nil)
	if err != nil {
		return nil, nil, err
	}
	return textResult(result), nil, nil
}

func (s *Server) handleCreateWindow(
	ctx context.Context,
	req *mcp.CallToolRequest,
	args CreateWindowArgs,
) (*mcp.CallToolResult, any, error) {
	body := map[string]any{}
	if len(args.URLs) > 0 {
		body["url"] = args.URLs[0]
	}
	if args.Focused {
		body["focused"] = true
	}
	if args.Incognito {
		body["incognito"] = true
	}

	result, err := s.requestAllBrowsers(ctx, "POST", "/windows", body)
	if err != nil {
		return nil, nil, err
	}
	return textResult(result), nil, nil
}

func (s *Server) handleUpdateWindow(
	ctx context.Context,
	req *mcp.CallToolRequest,
	args UpdateWindowArgs,
) (*mcp.CallToolResult, any, error) {
	body := map[string]any{}
	if args.Focused {
		body["focused"] = true
	}
	if args.State != "" {
		body["state"] = args.State
	}

	result, err := s.requestAllBrowsers(ctx, "PUT", "/windows/"+args.WindowID, body)
	if err != nil {
		return nil, nil, err
	}
	return textResult(result), nil, nil
}

func (s *Server) handleCloseWindow(
	ctx context.Context,
	req *mcp.CallToolRequest,
	args CloseWindowArgs,
) (*mcp.CallToolResult, any, error) {
	result, err := s.requestAllBrowsers(ctx, "DELETE", "/windows/"+args.WindowID, nil)
	if err != nil {
		return nil, nil, err
	}
	return textResult(result), nil, nil
}

func (s *Server) handleListTabs(
	ctx context.Context,
	req *mcp.CallToolRequest,
	_ any,
) (*mcp.CallToolResult, any, error) {
	result, err := s.requestAllBrowsers(ctx, "GET", "/tabs", nil)
	if err != nil {
		return nil, nil, err
	}
	return textResult(result), nil, nil
}

func (s *Server) handleGetTab(
	ctx context.Context,
	req *mcp.CallToolRequest,
	args GetTabArgs,
) (*mcp.CallToolResult, any, error) {
	result, err := s.requestAllBrowsers(ctx, "GET", "/tabs/"+args.TabID, nil)
	if err != nil {
		return nil, nil, err
	}
	return textResult(result), nil, nil
}

func (s *Server) handleCreateTab(
	ctx context.Context,
	req *mcp.CallToolRequest,
	args CreateTabArgs,
) (*mcp.CallToolResult, any, error) {
	body := map[string]any{
		"url": args.URL,
	}
	if args.WindowID != "" {
		body["windowId"] = args.WindowID
	}
	if args.Active {
		body["active"] = true
	}

	result, err := s.requestAllBrowsers(ctx, "POST", "/tabs", body)
	if err != nil {
		return nil, nil, err
	}
	return textResult(result), nil, nil
}

func (s *Server) handleUpdateTab(
	ctx context.Context,
	req *mcp.CallToolRequest,
	args UpdateTabArgs,
) (*mcp.CallToolResult, any, error) {
	body := map[string]any{}
	if args.URL != "" {
		body["url"] = args.URL
	}
	if args.Active {
		body["active"] = true
	}

	result, err := s.requestAllBrowsers(ctx, "PATCH", "/tabs/"+args.TabID, body)
	if err != nil {
		return nil, nil, err
	}
	return textResult(result), nil, nil
}

func (s *Server) handleCloseTab(
	ctx context.Context,
	req *mcp.CallToolRequest,
	args CloseTabArgs,
) (*mcp.CallToolResult, any, error) {
	result, err := s.requestAllBrowsers(ctx, "DELETE", "/tabs/"+args.TabID, nil)
	if err != nil {
		return nil, nil, err
	}
	return textResult(result), nil, nil
}

func (s *Server) handleGetBrowserItems(
	ctx context.Context,
	req *mcp.CallToolRequest,
	_ any,
) (*mcp.CallToolResult, any, error) {
	socks, err := s.getSockets()
	if err != nil {
		return nil, nil, errors.Wrap(err)
	}

	resp, err := s.proxy.GetForSockets(ctx, browser_items.BrowserRequestGet{}, socks)
	if err != nil {
		return nil, nil, errors.Wrap(err)
	}

	jsonBytes, err := json.MarshalIndent(resp.RequestPayloadGet, "", "  ")
	if err != nil {
		return nil, nil, errors.Wrap(err)
	}

	return textResult(string(jsonBytes)), nil, nil
}

func (s *Server) handleManageBrowserItems(
	ctx context.Context,
	req *mcp.CallToolRequest,
	args ManageItemsArgs,
) (*mcp.CallToolResult, any, error) {
	browserReq := browser_items.BrowserRequestPut{
		Added:   make([]browser_items.Item, 0, len(args.Added)),
		Deleted: make([]browser_items.Item, 0, len(args.Deleted)),
		Focused: make([]browser_items.Item, 0, len(args.Focused)),
	}

	for _, item := range args.Added {
		var url browser_items.Url
		url.Set(item.URL)
		browserReq.Added = append(browserReq.Added, browser_items.Item{
			Url:   url,
			Title: item.Title,
		})
	}

	for _, item := range args.Deleted {
		browserReq.Deleted = append(browserReq.Deleted, browser_items.Item{
			ExternalId: item.ID,
		})
	}

	for _, item := range args.Focused {
		browserReq.Focused = append(browserReq.Focused, browser_items.Item{
			ExternalId: item.ID,
		})
	}

	socks, err := s.getSockets()
	if err != nil {
		return nil, nil, errors.Wrap(err)
	}

	resp, err := s.proxy.PutForSockets(ctx, browserReq, socks)
	if err != nil {
		return nil, nil, errors.Wrap(err)
	}

	jsonBytes, err := json.MarshalIndent(resp.RequestPayloadPut, "", "  ")
	if err != nil {
		return nil, nil, errors.Wrap(err)
	}

	return textResult(string(jsonBytes)), nil, nil
}

func (s *Server) handleGetBrowserState(
	ctx context.Context,
	req *mcp.CallToolRequest,
	_ any,
) (*mcp.CallToolResult, any, error) {
	result, err := s.requestAllBrowsers(ctx, "GET", "/state", nil)
	if err != nil {
		return nil, nil, err
	}
	return textResult(result), nil, nil
}

func (s *Server) handleRestoreBrowserState(
	ctx context.Context,
	req *mcp.CallToolRequest,
	args RestoreStateArgs,
) (*mcp.CallToolResult, any, error) {
	var state any
	if err := json.Unmarshal(args.State, &state); err != nil {
		return nil, nil, errors.Wrap(err)
	}

	result, err := s.requestAllBrowsers(ctx, "POST", "/state", state)
	if err != nil {
		return nil, nil, err
	}
	return textResult(result), nil, nil
}

func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
	}
}

func (s *Server) requestAllBrowsers(
	ctx context.Context,
	method string,
	path string,
	body any,
) (string, error) {
	socks, err := s.getSockets()
	if err != nil {
		return "", errors.Wrap(err)
	}

	if len(socks) == 0 {
		return "[]", nil
	}

	wg := errors.MakeWaitGroupParallel()
	var l sync.Mutex
	var allResults []any

	for _, sock := range socks {
		wg.Do(func() (err error) {
			result, err := s.requestOneBrowser(ctx, sock, method, path, body)
			if err != nil {
				return nil
			}

			l.Lock()
			defer l.Unlock()

			if arr, ok := result.([]any); ok {
				allResults = append(allResults, arr...)
			} else if result != nil {
				allResults = append(allResults, result)
			}

			return nil
		})
	}

	if err = wg.GetError(); err != nil {
		return "", errors.Wrap(err)
	}

	jsonBytes, err := json.MarshalIndent(allResults, "", "  ")
	if err != nil {
		return "", errors.Wrap(err)
	}

	return string(jsonBytes), nil
}

func (s *Server) requestOneBrowser(
	ctx context.Context,
	sock string,
	method string,
	path string,
	body any,
) (any, error) {
	var bodyReader *json.Encoder

	pr, pw := net.Pipe()

	if body != nil {
		bodyReader = json.NewEncoder(pw)
		go func() {
			bodyReader.Encode(body)
			pw.Close()
		}()
	} else {
		pw.Close()
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, path, pr)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	if body != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "unix", sock)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	defer conn.Close()

	if err = httpReq.Write(conn); err != nil {
		return nil, errors.Wrap(err)
	}

	resp, err := http.ReadResponse(bufio.NewReader(conn), httpReq)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 204 {
		return map[string]string{"status": "success"}, nil
	}

	var result any
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, errors.Wrap(err)
	}

	return result, nil
}
