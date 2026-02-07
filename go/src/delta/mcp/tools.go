package mcp

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (s *Server) registerTools() {
	s.registerBrowserInfoTools()
	s.registerWindowTools()
	s.registerTabTools()
	s.registerItemTools()
	s.registerStateTools()
}

func (s *Server) registerBrowserInfoTools() {
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "browser_info",
		Description: "Get browser information including type, version, and platform details from all connected browsers",
	}, s.handleBrowserInfo)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "list_extensions",
		Description: "List all installed browser extensions from all connected browsers",
	}, s.handleListExtensions)
}

func (s *Server) registerWindowTools() {
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "list_windows",
		Description: "List all browser windows with their tabs from all connected browsers",
	}, s.handleListWindows)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "get_window",
		Description: "Get details of a specific window by ID",
	}, s.handleGetWindow)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "create_window",
		Description: "Create a new browser window with optional URLs",
	}, s.handleCreateWindow)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "update_window",
		Description: "Update properties of a specific window (focus, state, etc.)",
	}, s.handleUpdateWindow)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "close_window",
		Description: "Close a browser window by ID",
	}, s.handleCloseWindow)
}

func (s *Server) registerTabTools() {
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "list_tabs",
		Description: "List all tabs across all windows from all connected browsers",
	}, s.handleListTabs)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "get_tab",
		Description: "Get details of a specific tab by ID",
	}, s.handleGetTab)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "create_tab",
		Description: "Create a new tab with the specified URL",
	}, s.handleCreateTab)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "update_tab",
		Description: "Update a tab's URL or other properties",
	}, s.handleUpdateTab)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "close_tab",
		Description: "Close a tab by ID",
	}, s.handleCloseTab)
}

func (s *Server) registerItemTools() {
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "get_browser_items",
		Description: "Get all browser items (tabs, bookmarks, history) from all connected browsers",
	}, s.handleGetBrowserItems)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "manage_browser_items",
		Description: "Add, delete, or focus browser items across all connected browsers",
	}, s.handleManageBrowserItems)
}

func (s *Server) registerStateTools() {
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "get_browser_state",
		Description: "Get the current browser state (windows and tabs) for saving/restoring",
	}, s.handleGetBrowserState)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "restore_browser_state",
		Description: "Restore a previously saved browser state (create windows and tabs)",
	}, s.handleRestoreBrowserState)
}
