package mcp

import (
	"context"
	"fmt"
	"net/http"

	"code.linenisgreat.com/chrest/go/src/bravo/config"
	"code.linenisgreat.com/chrest/go/src/charlie/browser_items"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Server struct {
	mcpServer *mcp.Server
	proxy     browser_items.BrowserProxy
	scopes    ScopeConfig
}

func NewServer(c config.Config, scopes ScopeConfig) *Server {
	mcpServer := mcp.NewServer(
		&mcp.Implementation{
			Name:    "chrest",
			Version: "1.0.0",
		},
		nil,
	)

	s := &Server{
		mcpServer: mcpServer,
		proxy:     browser_items.BrowserProxy{Config: c},
		scopes:    scopes,
	}

	s.registerTools()

	return s
}

func (s *Server) RunStdio(ctx context.Context) (err error) {
	if err = s.mcpServer.Run(ctx, &mcp.StdioTransport{}); err != nil {
		err = errors.Wrap(err)
		return
	}

	return
}

func (s *Server) RunSSE(ctx context.Context, port int) (err error) {
	handler := mcp.NewSSEHandler(
		func(r *http.Request) *mcp.Server {
			return s.mcpServer
		},
		nil,
	)

	addr := fmt.Sprintf(":%d", port)

	if err = http.ListenAndServe(addr, handler); err != nil {
		err = errors.Wrap(err)
		return
	}

	return
}
