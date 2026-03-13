package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
	"github.com/amarbel-llc/purse-first/libs/go-mcp/command"
)

func registerInstallMCPCommand(app *command.App) {
	app.AddCommand(&command.Command{
		Name: "install-mcp",
		Description: command.Description{
			Short: "Install chrest as a Claude Code MCP server",
		},
		RunCLI: func(ctx context.Context, args json.RawMessage) error {
			return cmdInstallMCP()
		},
	})
}

type mcpServerConfig struct {
	Type    string   `json:"type"`
	Command string   `json:"command"`
	Args    []string `json:"args,omitempty"`
}

type mcpJSON struct {
	MCPServers map[string]mcpServerConfig `json:"mcpServers"`
}

func cmdInstallMCP() (err error) {
	exe, err := os.Executable()
	if err != nil {
		return errors.Wrap(err)
	}

	if exe, err = filepath.EvalSymlinks(exe); err != nil {
		return errors.Wrap(err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return errors.Wrap(err)
	}

	mcpPath := filepath.Join(home, ".claude", "mcp.json")

	var config mcpJSON

	if data, readErr := os.ReadFile(mcpPath); readErr == nil {
		if err = json.Unmarshal(data, &config); err != nil {
			return errors.Wrap(err)
		}
	}

	if config.MCPServers == nil {
		config.MCPServers = make(map[string]mcpServerConfig)
	}

	config.MCPServers["chrest"] = mcpServerConfig{
		Type:    "stdio",
		Command: exe,
		Args:    []string{"mcp"},
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return errors.Wrap(err)
	}

	if err = os.MkdirAll(filepath.Dir(mcpPath), 0o700); err != nil {
		return errors.Wrap(err)
	}

	if err = os.WriteFile(mcpPath, append(data, '\n'), 0o644); err != nil {
		return errors.Wrap(err)
	}

	ui.Out().Printf("installed chrest MCP server to %s", mcpPath)

	return
}
