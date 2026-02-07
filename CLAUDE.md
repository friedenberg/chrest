# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

Chrest is a CLI tool and browser extension that enables managing Chrome/Firefox via REST. It consists of:
1. **Browser Extension** (`extension/`) - JavaScript service worker that exposes browser APIs as REST endpoints
2. **Native Messaging Host & CLI** (`go/`) - Go binary that communicates with the extension via Chrome Native Messaging

## Build Commands

All commands use justfiles. Run from appropriate directory or use root justfile.

### Full Build
```bash
just build              # builds both go and extension
just reload             # builds and reinstalls extension
```

### Go (from `go/` directory)
```bash
just build              # builds debug and release binaries to build/
just build-nix          # builds via nix
just tests-go           # run tests: go test -v ./...
just check              # run govulncheck and go vet
just codemod-go-fmt     # format with goimports and gofumpt
just update-go          # update dependencies
```

### Extension (from `extension/` directory)
```bash
just build              # builds both chrome and firefox
just build-chrome       # builds chrome extension to dist-chrome/
just build-firefox      # builds firefox extension to dist-firefox/
just deploy-firefox     # sign and deploy to Firefox AMO
```

## Architecture

### Communication Flow
1. CLI sends HTTP requests to Unix socket (`$XDG_STATE_HOME/chrest/<browser-id>.sock`)
2. Go server (`go/src/bravo/server/`) forwards requests to browser extension via Native Messaging
3. Extension (`extension/src/main.js`) routes requests to handlers and returns HTTP responses
4. Extension uses mutex to serialize requests

### Go Package Structure (`go/src/`)
- `alfa/browser/` - Browser detection utilities
- `alfa/symlink/` - Symlink handling
- `bravo/client/` - HTTP client for browser proxy communication
- `bravo/server/` - Unix socket HTTP server, Native Messaging protocol
- `bravo/config/` - Configuration and state directory management
- `charlie/install/` - Native messaging host installation (platform-specific paths)
- `charlie/browser_items/` - Browser item types and operations

### CLI Commands (`go/cmd/chrest/main.go`)
- `chrest` (default) - Start native messaging server
- `chrest client` - Forward HTTP request from stdin to browser
- `chrest install <extension-id>` - Install native messaging host manifest
- `chrest reload-extension` - Reload the browser extension
- `chrest items-get` / `items-put` - Get/put browser items
- `chrest init` - Initialize configuration

### Extension REST Routes (`extension/src/routes.js`)
- `/` - Browser info
- `/windows`, `/windows/#WINDOW_ID` - Window CRUD
- `/tabs`, `/tabs/#TAB_ID` - Tab CRUD
- `/state` - Save/restore/clear browser state
- `/items` - Unified tabs, bookmarks, history
- `/bookmarks`, `/history` - Read-only access
- `/extensions` - List extensions
- `/runtime/reload` - Reload extension

## Development Environment

Uses nix flakes with direnv. The root `flake.nix` provides a dev shell with Go and JS tooling via `devenv-go` and `devenv-js` from `github:friedenberg/eng?dir=pkgs/alfa`.

## Usage Example

```bash
# Get all windows and tabs
http --ignore-stdin --offline localhost/windows | chrest client | jq

# Create new window with URL
http --ignore-stdin --offline localhost/windows urls[]=https://example.com | chrest client | jq

# Close a window
http --ignore-stdin --offline DELETE localhost/windows/1234 | chrest client
```
