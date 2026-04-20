# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

Chrest is a CLI tool and browser extension that enables managing Chrome/Firefox via REST. It consists of:
1. **Browser Extension** (`extension/`) - JavaScript service worker that exposes browser APIs as REST endpoints
2. **Native Messaging Host & CLI** (`go/`) - Go binary that communicates with the extension via Chrome Native Messaging

## Build Commands

All commands use justfiles. Run from appropriate directory or use root justfile.
Ad-hoc debug/exploration recipes live under `[group: 'explore']` in the root
justfile — prefer adding recipes there over writing one-off shell scripts.

### Full Build
```bash
just build              # builds both go and extension
just reload             # builds and reinstalls extension
just test               # runs test-go + test-mcp + test-mcp-bats
just test-mcp           # validates MCP tools, resources, and annotations
just test-mcp-bats      # BATS integration suite against a real unix socket
just dev-install-mcp    # build + install MCP server to ~/.claude.json
just demo               # generate VHS demo GIF
```

`test-mcp-bats` is wall-clock bounded (120s timeout) and validates success by
parsing TAP output — bats has been observed to hang on shutdown in bwrap
`--unshare-pid` sandboxes even when every test passes. Root cause still open.

`sweatfile` wires `pre-merge = "just"` — spinclass merge runs the full suite
before merging a worktree branch back to master.

### Go (from `go/` directory)
```bash
just build              # builds debug and release binaries to build/ for every cmd/*
just build-nix          # builds via nix
just tests-go           # run tests: go test -v ./...
just check              # run govulncheck and go vet
just codemod-go-fmt     # format with goimports and gofumpt
just update-go          # update dependencies
just add-dep <pkg>      # go get <pkg> + go mod tidy (keeps gomod2nix in sync)
```

`go/build` builds three binaries: `chrest` (the main CLI + native messaging
host + MCP server), `chrest-jcs` (standalone JCS canonicalizer, used for
cross-implementation byte-stability fixtures), and `chrest-server`.

`build-nix-gomod` fails loudly if `gomod2nix.toml` drifts during regeneration
— a silent regen-then-lose loop caused a broken nix build during the pdfcpu
add. Always stage the regenerated toml.

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
- `bravo/bidi/` - WebDriver BiDi transport. Background `readLoop` owns all
  reads; routes response frames to per-request channels and fans events out
  to `Subscribe(methods)` consumers. Prerequisite for capture envelope `http.*`
  fields (chrest#24).
- `bravo/cdp/` - `cdp.Session` interface used by the capture pipeline, plus
  shared `HTTPResponse` / `HTTPHeader` types. Headers are `[]HTTPHeader`
  (name/value pairs) rather than a map so the envelope preserves order and
  duplicates (e.g. `Set-Cookie`).
- `charlie/install/` - Native messaging host installation (platform-specific paths)
- `charlie/browser_items/` - Browser item types and operations
- `charlie/extension/` - Extension-driven CDP backend (uses `chrome.debugger`).
- `charlie/firefox/` - Firefox/BiDi capture backend. Subscribes to
  `network.responseCompleted`, drains stale events before each navigate, and
  populates `LastNavigationHTTP()` — enables envelope v1 emission.
- `charlie/headless/` - Headless Chrome/CDP capture backend. No network-event
  wiring yet, so envelopes from this backend are v1-preview.
- `charlie/launcher/` - Browser process launching.
- `delta/proxy/` - Multi-browser proxy (fan-out requests to all sockets)
- `delta/tools/` - MCP tool definitions with annotations
- `delta/resources/` - MCP paginated resources (`chrest://items`, `chrest://items/{page}`)
- `delta/capturebatch/` - See "Capture Pipeline" below.

### CLI Commands (`go/cmd/chrest/main.go`)
- `chrest` (default) - Start native messaging server
- `chrest client` - Forward HTTP request from stdin to browser
- `chrest install <extension-id>` - Install native messaging host manifest
- `chrest install-mcp` - Install MCP server config to `~/.claude.json`
- `chrest reload-extension` - Reload the browser extension
- `chrest items-get` / `items-put` - Get/put browser items
- `chrest init` - Initialize configuration (browser, name, extension-id)
- `chrest mcp` - Start MCP server (stdio transport)
- `chrest capture --format <kind>` - Single-page capture. Formats: `pdf`,
  `screenshot-png`, `screenshot-jpeg`, `mhtml`, `a11y`, `text`. Has a
  `--timeout` flag (default 60s) backed by a deadline context.
- `chrest capture-batch` - RFC 0001 capturer role (MVP, `split=false`). Reads
  a batch input JSON on stdin, runs captures sequentially, streams each
  artifact to a writer subprocess, and emits a result envelope on stdout.

### Other binaries (`go/cmd/`)
- `chrest-jcs` - Standalone JCS (RFC 8785) canonicalizer. Reads JSON on stdin,
  writes canonicalized bytes on stdout. Used for byte-stability cross-checks
  against the nebulous-side implementation.
- `chrest-server` - Native messaging host server binary.

### Capture Pipeline (`go/src/delta/capturebatch/`)

Implements the chrest side of the **Web Capture Archive Protocol (RFC 0001)**.
The RFC document itself lives in the nebulous session at `~/eng/aim/` — it is
not checked into this repo. Fixtures shared with the nebulous session also live
under `~/eng/aim/fixtures/` and are referenced by `explore-capture-batch` and
`explore-jcs-fixture`.

Schema tokens (see `types.go`, `envelope.go`, `spec.go`):
- Input/output schema: `web-capture-archive/v1`
- Envelope schema: `web-capture-archive.envelope/v1` when `http.status` +
  `http.headers` are populated (Firefox/BiDi backend), or
  `web-capture-archive.envelope/v1-preview` when they can't be (headless-CDP
  backend — can't observe network events today, chrest#24 follow-up).
  Preview-tolerant consumers opt in knowingly; v1-strict consumers reject.
- Capturer identifier: `chrest` (`CapturerName` constant).

Files of note:
- `runner.go` - Drives the batch; orchestrates per-capture fingerprint,
  normalize, spec+envelope emission, and writer fan-out.
- `envelope.go` / `spec.go` - Artifact shapes + constants. `spec.isolation`
  is omitted when unset (chrest#28); never emit `""` — absence means "not
  set", not "empty".
- `fingerprint.go` - Content-addressed hashing (blake2b-256).
- `jcs.go` + `jcs_test.go` - JSON Canonicalization Scheme used for
  deterministic envelope/spec bytes.
- `mhtml.go`, `pdf.go`, `png.go`, `normalize.go` - Format-specific
  normalizers. Normalizers strip non-deterministic bits and return a
  `stripped.<format>` sidecar describing what they removed.
- `writer.go` - Streams artifacts to the orchestrator-supplied writer
  subprocess (RFC 0001 `WriterSpec.Cmd`).

Known open issues:
- **chrest#27** — PDF byte-stability. Pdfcpu has map-iteration-dependent
  object numbering + stream placement; two normalizePDF calls on the same
  input can produce outputs differing in length by 1 byte with the first
  diff near offset 309. The `/Info` + `/ID` scrub landed in `pdf.go` but
  full determinism still requires pdfcpu-side work. `explore-pdf-inspect-info`
  recipe decompresses FlateDecode streams to inspect `/Info` placement.

### MCP Server (`chrest mcp`)
Exposes browser management as MCP tools and resources over stdio (JSON-RPC 2.0).

**Tools** — all browser tools (list-windows, create-tab, close-tab, etc.) plus:
- `read-resource` — bridge tool so subagents can access MCP resources via tools/call

**Resources:**
- `chrest://items` — paginated index (total count, page URIs)
- `chrest://items/{page}` — 100 items per page (tabs, bookmarks, history)
- Items are cached for 30s to handle concurrent reads

**Annotations:** read-only tools have `readOnlyHint`, destructive tools (close-*, state-restore, items-put) have `destructiveHint`. Validated by `just test-mcp`.

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

## Design Docs and Tests

- `docs/plans/` holds dated design docs and implementation plans (e.g.
  `2026-04-14-cdp-capture-commands-design.md`). Check here first when picking
  up in-flight feature work — the `design` + implementation file pair together
  is the contract for a given chunk of work.
- `zz-tests_bats/` - BATS integration tests that exercise `chrest` end-to-end
  against real unix sockets (`--allow-unix-sockets`). Suites: `capture.bats`,
  `capture_batch.bats`, `capture_firefox.bats`, `mcp.bats`. Run via
  `just test-mcp-bats`.

## Usage Example

```bash
# Get all windows and tabs
http --ignore-stdin --offline localhost/windows | chrest client | jq

# Create new window with URL
http --ignore-stdin --offline localhost/windows url[]=https://example.com | chrest client | jq

# Close a window
http --ignore-stdin --offline DELETE localhost/windows/1234 | chrest client
```
