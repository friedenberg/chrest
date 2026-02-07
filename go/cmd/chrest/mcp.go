package main

import (
	"flag"

	"code.linenisgreat.com/chrest/go/src/bravo/config"
	mcpserver "code.linenisgreat.com/chrest/go/src/delta/mcp"
	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
)

var (
	mcpTransport *string
	mcpPort      *int
)

func mcpAddFlags() {
	mcpTransport = flag.String("transport", "stdio", "Transport type: stdio or sse")
	mcpPort = flag.Int("port", 8080, "Port for SSE transport")
}

func CmdMcp(ctx interfaces.ActiveContext, c config.Config) (err error) {
	addFlagsOnce.Do(mcpAddFlags)
	flag.Parse()

	server := mcpserver.NewServer(c)

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
