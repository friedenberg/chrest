package command

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type devMcpConfig struct {
	McpServers map[string]pluginMcpServer `json:"mcpServers"`
}

type devMappingFile struct {
	Server     string         `json:"server"`
	ToolPrefix string         `json:"tool_prefix"`
	Mappings   []mappingEntry `json:"mappings"`
}

func resolveBuildDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolving executable: %w", err)
	}

	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return "", fmt.Errorf("resolving symlinks: %w", err)
	}

	// Binary is at {buildDir}/bin/{name}, so buildDir is two levels up
	return filepath.Dir(filepath.Dir(resolved)), nil
}

func generateDevMCP(buildDir, projectDir, suffix string) error {
	// Read plugin.json
	pluginDirs, err := filepath.Glob(filepath.Join(buildDir, "share", "purse-first", "*", ".claude-plugin", "plugin.json"))
	if err != nil {
		return fmt.Errorf("finding plugin.json: %w", err)
	}
	if len(pluginDirs) == 0 {
		return fmt.Errorf("no plugin.json found in %s/share/purse-first/*/.claude-plugin/", buildDir)
	}

	pluginPath := pluginDirs[0]
	pluginData, err := os.ReadFile(pluginPath)
	if err != nil {
		return fmt.Errorf("reading plugin.json: %w", err)
	}

	var manifest pluginManifest
	if err := json.Unmarshal(pluginData, &manifest); err != nil {
		return fmt.Errorf("parsing plugin.json: %w", err)
	}

	name := manifest.Name
	serverKey := name + "-" + suffix

	// Find the binary
	binDir := filepath.Join(buildDir, "bin")
	entries, err := os.ReadDir(binDir)
	if err != nil {
		return fmt.Errorf("reading bin directory: %w", err)
	}
	if len(entries) == 0 {
		return fmt.Errorf("no binaries found in %s", binDir)
	}
	binaryPath := filepath.Join(binDir, entries[0].Name())

	// Get MCP args from the plugin manifest
	var mcpArgs []string
	for _, server := range manifest.McpServers {
		mcpArgs = server.Args
		break
	}

	// 1. Merge our server entry into the project-local .mcp.json.
	// Read-modify-write rather than overwrite so we don't clobber
	// other servers the project already registered (e.g. spinclass).
	mcpPath := filepath.Join(projectDir, ".mcp.json")
	mcpRaw, err := readUserMCPConfig(mcpPath)
	if err != nil {
		return err
	}
	mcpServers, _ := mcpRaw["mcpServers"].(map[string]any)
	if mcpServers == nil {
		mcpServers = make(map[string]any)
	}
	mcpServers[serverKey] = map[string]any{
		"type":    "stdio",
		"command": binaryPath,
		"args":    mcpArgs,
	}
	mcpRaw["mcpServers"] = mcpServers
	if err := writeUserMCPConfig(mcpPath, mcpRaw); err != nil {
		return err
	}

	// 2. Write .purse-first/<name>.json with tool_prefix
	mappingsPath := filepath.Join(buildDir, "share", "purse-first", name, "mappings.json")
	// Write .purse-first/<name>.json if mappings.json exists in build artifacts.
	// Not all plugins have mappings, so a missing file is not an error.
	mappingsData, err := os.ReadFile(mappingsPath)
	if err == nil {
		var sourceMappings mappingFile
		if err := json.Unmarshal(mappingsData, &sourceMappings); err != nil {
			return fmt.Errorf("parsing mappings.json: %w", err)
		}

		devMappings := devMappingFile{
			Server:     sourceMappings.Server,
			ToolPrefix: "mcp__" + serverKey,
			Mappings:   sourceMappings.Mappings,
		}

		purseDir := filepath.Join(projectDir, ".purse-first")
		if err := os.MkdirAll(purseDir, 0o755); err != nil {
			return fmt.Errorf("creating .purse-first/: %w", err)
		}

		devData, err := json.MarshalIndent(devMappings, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling dev mappings: %w", err)
		}
		devData = append(devData, '\n')

		if err := os.WriteFile(filepath.Join(purseDir, name+".json"), devData, 0o644); err != nil {
			return fmt.Errorf("writing dev mappings: %w", err)
		}
	}

	return nil
}

// cleanDevMCP removes only the dev entry written by generateDevMCP,
// preserving any other servers the project registered. The mappings
// file is named per-plugin so it can be deleted unconditionally; the
// .purse-first/ directory is removed only if it ends up empty.
func cleanDevMCP(projectDir, serverKey, name string) error {
	mcpPath := filepath.Join(projectDir, ".mcp.json")
	if mcpRaw, err := readUserMCPConfig(mcpPath); err == nil {
		if mcpServers, ok := mcpRaw["mcpServers"].(map[string]any); ok {
			delete(mcpServers, serverKey)
			if len(mcpServers) == 0 {
				delete(mcpRaw, "mcpServers")
			} else {
				mcpRaw["mcpServers"] = mcpServers
			}
			if len(mcpRaw) == 0 {
				_ = os.Remove(mcpPath)
			} else if err := writeUserMCPConfig(mcpPath, mcpRaw); err != nil {
				return err
			}
		}
	}

	purseDir := filepath.Join(projectDir, ".purse-first")
	_ = os.Remove(filepath.Join(purseDir, name+".json"))
	if entries, err := os.ReadDir(purseDir); err == nil && len(entries) == 0 {
		_ = os.Remove(purseDir)
	}
	return nil
}

func (u *Utility) addDevMCPCommand() {
	u.AddCommand(&Command{
		Name:   "dev-mcp",
		Hidden: true,
		Description: Description{
			Short: "Generate project-local MCP config for dev testing",
		},
		OldParams: []OldParam{
			{Name: "suffix", Type: String, Description: "Suffix for the MCP server name", Default: "dev"},
			{Name: "clean", Type: Bool, Description: "Remove generated dev artifacts", Default: false},
		},
		RunCLI: func(ctx context.Context, args json.RawMessage) error {
			projectDir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("resolving project dir: %w", err)
			}

			var p struct {
				Suffix string `json:"suffix"`
				Clean  bool   `json:"clean"`
			}
			p.Suffix = "dev"
			if len(args) > 0 {
				if err := json.Unmarshal(args, &p); err != nil {
					return fmt.Errorf("parsing dev-mcp args: %w", err)
				}
			}

			if p.Clean {
				serverKey := u.Name + "-" + p.Suffix
				return cleanDevMCP(projectDir, serverKey, u.Name)
			}

			buildDir, err := resolveBuildDir()
			if err != nil {
				return err
			}
			if err := generateDevMCP(buildDir, projectDir, p.Suffix); err != nil {
				return err
			}
			fmt.Printf("wrote .mcp.json with %s-%s server pointing at %s/bin/\n",
				u.Name, p.Suffix, buildDir)
			return nil
		},
	})
}
