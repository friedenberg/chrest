package main

import (
	"flag"
	"strings"

	"code.linenisgreat.com/chrest/go/src/bravo/config"
	mcpserver "code.linenisgreat.com/chrest/go/src/delta/mcp"
	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
)

type scopeFlags []string

func (s *scopeFlags) String() string {
	return strings.Join(*s, ",")
}

func (s *scopeFlags) Set(value string) error {
	*s = append(*s, value)
	return nil
}

var (
	mcpTransport *string
	mcpPort      *int
	mcpScopes    scopeFlags
)

func mcpAddFlags() {
	mcpTransport = flag.String("transport", "stdio", "Transport type: stdio or sse")
	mcpPort = flag.Int("port", 8080, "Port for SSE transport")
	flag.Var(&mcpScopes, "scope", "Permission scope (repeatable, format: scope:level, e.g., tabs:read)")
}

func CmdMcp(ctx interfaces.ActiveContext, c config.Config) (err error) {
	addFlagsOnce.Do(mcpAddFlags)
	flag.Parse()

	// Start with defaults
	scopes := mcpserver.DefaultScopes()

	// Override from config file
	scopes.MergeFrom(c.MCP.Scopes)

	// Override from CLI flags
	if err = scopes.MergeFromFlags(mcpScopes); err != nil {
		err = errors.Wrap(err)
		return
	}

	server := mcpserver.NewServer(c, scopes)

	switch *mcpTransport {
	case "stdio":
		if err = server.RunStdio(ctx); err != nil {
			err = errors.Wrap(err)
			return
		}

	case "sse":
		if err = server.RunSSE(ctx, *mcpPort); err != nil {
			err = errors.Wrap(err)
			return
		}

	default:
		err = errors.Errorf("unknown transport: %s", *mcpTransport)
		return
	}

	return
}
