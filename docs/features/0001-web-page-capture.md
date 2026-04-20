---
status: accepted
date: 2026-04-20
promotion-criteria:
---

# Web page capture

## Problem Statement

Preserving a web page at a moment in time — for archival, for offline reading, for feeding to downstream tools (LLMs, full-text search, reader apps) — requires a patchwork of shell scripts, browser plug-ins, and manual browser operations. Chrest already launches browsers and talks to them; it's the natural place to collapse those tasks into one tool that produces reproducible outputs in whichever format the consumer wants.

## Interface

Two top-level commands, one interactive and one batch-oriented:

### `chrest capture --format <kind> [flags]`

Single-page capture. Streams bytes straight to stdout (or to `--output <path>` atomically).

**Formats (10):**

| Format | Payload | Media type |
|---|---|---|
| `pdf` | PDF document from the browser's print pipeline | `application/pdf` |
| `screenshot-png` | Full-page or viewport PNG | `image/png` |
| `screenshot-jpeg` | Full-page or viewport JPEG (tunable `--quality`) | `image/jpeg` |
| `mhtml` | Chrome MHTML snapshot (Chrome backend only) | `multipart/related` |
| `a11y` | Accessibility tree JSON (Chrome backend only) | `application/json` |
| `text` | `document.body.innerText` | `text/plain; charset=utf-8` |
| `html-monolith` | Rendered DOM inlined by `monolith` — every asset as a `data:` URI | `text/html; charset=utf-8` |
| `markdown-full` | Rendered DOM converted to markdown | `text/markdown; charset=utf-8` |
| `markdown-reader` | Readability-extracted article converted to markdown | `text/markdown; charset=utf-8` |
| `markdown-selector` | CSS-selector-scoped element converted to markdown (requires `--selector`) | `text/markdown; charset=utf-8` |

**Backend selection:**

- `--browser firefox` (default) — headless Firefox via WebDriver BiDi. Works for every format except `mhtml` and `a11y`.
- `--browser chrome` (alias: `headless`) — headless Chrome via CDP. Required for `mhtml` / `a11y`; blocked on kernel 6.17 SIGTRAP (chrest#14).
- `--tab-id <id>` — bypass the headless browser and drive an already-open tab via the extension debugger. Wiring incomplete (chrest#30).

**Flags:**

- `--url <url>` — page to navigate to (required unless `--tab-id`).
- `--output <path>` — atomic tmpfile + rename; no file left behind on failure.
- `--timeout <dur>` (default 60s) — deadline-backed context; cancels the browser and writer on expiry.
- `--selector <css>` — required for `markdown-selector`; first match wins.
- `--reader-engine <readability|browser>` — `markdown-reader` only. `readability` (default) uses the embedded Go Readability port. `browser` is reserved and rejects with `not-yet-implemented`.
- Format-specific flags: `--landscape`, `--no-headers`, `--background` (PDF), `--quality` (JPEG), `--full-page` (screenshots).

The CLI exits non-zero on any error.

### `chrest capture-batch`

JSON-stdin / JSON-stdout batch capture implementing the **Web Capture Archive Protocol (RFC 0001)**. The RFC document itself lives outside this repo; the capturer contract chrest implements is:

- Reads a single JSON document on stdin with shape `{schema, writer, url, defaults, captures[]}`.
- Runs every capture sequentially.
- Streams each artifact to the orchestrator-supplied `writer.cmd` subprocess for content-addressed storage.
- Emits a single JSON result envelope on stdout with per-capture `payload` / `spec` / `envelope` ArtifactRefs.

Schema tokens: input/output `web-capture-archive/v1`; spec artifacts `web-capture-archive.spec/v1`; envelope artifacts `web-capture-archive.envelope/v1` (when HTTP fields are populated by a network-event-capable backend) or `v1-preview` (when they can't be).

Per-capture options echo into the spec artifact's `capture.options` via JCS canonicalization so downstream consumers can reproduce the exact extraction parameters.

## Examples

Single-page captures piped to stdout or files:

    $ chrest capture --format pdf --url https://example.com > page.pdf

    $ chrest capture --format screenshot-png --full-page \
        --url https://en.wikipedia.org/wiki/Markdown \
        --output wiki.png

    $ chrest capture --format html-monolith \
        --url https://simonwillison.net/ \
        --output blog.html

    $ chrest capture --format markdown-reader \
        --url https://simonwillison.net/2026/Feb/15/gwtar/ \
        --output gwtar.md

    $ chrest capture --format markdown-selector --selector "#bodyContent" \
        --url https://en.wikipedia.org/wiki/Markdown \
        --output wiki.md

Drive an already-open tab (when extension debugger wiring is available):

    $ chrest capture --format text --tab-id 1531273367

Batch capture (RFC 0001):

    $ echo '{
        "schema":   "web-capture-archive/v1",
        "writer":   {"cmd": ["madder", "--format=json", "write", "--store", "archive"]},
        "url":      "https://en.wikipedia.org/wiki/Ferris_wheel",
        "defaults": {"browser": "firefox", "split": false},
        "captures": [
          {"name": "pdf",     "format": "pdf"},
          {"name": "md",      "format": "markdown-reader"},
          {"name": "archive", "format": "html-monolith"}
        ]
      }' | chrest capture-batch | jq

Every capture produces a `payload` ArtifactRef keyed on its blake2b-256 content hash plus a `spec` ArtifactRef echoing the resolved options — so the archive is content-addressed and re-derivable.

## Limitations

- **Chrome headless is blocked on Linux kernel 6.17 SIGTRAP** (chrest#14). `mhtml` and `a11y` are Chrome-only and therefore unavailable on affected hosts; other formats fall back to Firefox.
- **Extension debugger path (`--tab-id`) is not yet end-to-end functional** (chrest#30). Chrome doesn't re-read the native messaging manifest without a restart, and `web-ext run --target chromium` uses a temporary extension ID that defeats `allowed_origins`.
- **`markdown-selector` takes the first match only.** No `--selector-mode=all` or similar. Selector misses are a typed error that names the selector.
- **`--reader-engine=browser` is reserved but not implemented.** The Firefox `about:reader` engine is accepted as a valid flag value so the spec surface stays stable but rejects with `not-yet-implemented` at runtime.
- **`html-monolith` requires the `monolith` binary on `PATH`.** The nix-built `chrest` wraps it in via `flake.nix` `postFixup`; a `go install`-ed chrest relies on the user's PATH.
- **`capture-batch` only supports `split=false` for `html-monolith` and `markdown-*`** — no byte-stability normalizer has been wired for those formats. Existing formats (`text`, `pdf`, `screenshot`, `mhtml`) do support `split=true`.
- **Envelope `http.*` fields are populated only by the Firefox/BiDi backend.** Chrome/CDP can't yet observe network events in this pipeline (chrest#24 follow-up); captures from that backend emit envelope schema `v1-preview` instead of `v1`.
- **BiDi network-event buffer drops events on heavy pages** (chrest#33). Affects envelope fidelity for `split=true` captures of media-heavy pages; harmless for `split=false`.
- **No splitting of an `html-monolith` / `markdown-*` payload into a normalized form.** The payload is recorded verbatim; a future `split=true` path could strip asset bytes into the envelope and normalize the wrapper.

## More Information

- RFC 0001 (Web Capture Archive Protocol) — lives in the nebulous session at `~/eng/aim/`, not in this repo.
- Design document: `docs/plans/2026-04-14-cdp-capture-commands-design.md` — the original headless-sidecar architecture that introduced the `cdp.Session` interface. The markdown and html-monolith formats extend that design without changing the interface shape.
- Related issues: chrest#10 (original html-to-pdf migration, closed), chrest#11 (multi-format aggregator, closed, superseded), chrest#26 (html-monolith, closed), chrest#29 (markdown variants, closed), chrest#14 (Chrome SIGTRAP blocker), chrest#30 (extension debugger wiring), chrest#33 (BiDi buffer drops), chrest#34 (capture exit-code, closed).
