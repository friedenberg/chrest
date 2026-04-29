#!/usr/bin/env bats

# End-to-end coverage for the web-fetch content-type-aware dispatcher.
# Pins the contract of the CHREST_WEB_FETCH_DISPATCH=bidi-intercept path
# (default) against real upstream URLs:
#   - HTML (example.com)         → existing TOC + body (regression).
#   - Raw .md (anthropic SDK)    → body + TOC built from raw markdown.
#   - Raw .toml (anthropic SDK)  → text body, no schema crash.
#   - 404 URL                    → structured HTTP 404 error.
#   - Binary tarball             → structured "binary content-type"
#                                  refusal, no MCP schema crash.
#
# These tests need real Firefox + network reachability. They skip
# cleanly when Firefox is missing on PATH.

setup() {
  load "$(dirname "$BATS_TEST_FILE")/common.bash"
  setup_test_home
}

teardown() {
  teardown_test_home
}

# Common skip gate; mirrors the pattern used by the web_fetch_* tests
# in mcp.bats.
require_firefox() {
  firefox="$(command -v firefox || command -v firefox-esr || true)"
  if [ -z "$firefox" ]; then
    skip "no Firefox found on PATH"
  fi
  if ! timeout 5 "$firefox" --headless --version >/dev/null 2>&1; then
    skip "headless Firefox not functional"
  fi
}

function web_fetch_html_url_regression { # @test
  require_firefox

  url="https://example.com"
  call=$(jq -nc --arg url "$url" '{jsonrpc:"2.0",id:2,method:"tools/call",params:{name:"web-fetch",arguments:{url:$url,format:"markdown"}}}')
  result=$(printf '%s\n' "$INIT_MSG" "$INITIALIZED_MSG" "$call" |
    timeout 60 "$CHREST_BIN" mcp)

  resp=$(echo "$result" | grep '"id":2')
  echo "$resp" | jq -e '.result.isError != true'
  # Default markdown layout: TOC + embedded markdown resource + 2 resource_links.
  echo "$resp" | jq -e '.result.content | length == 4'
  echo "$resp" | jq -e '.result.content[0].type == "text"'
  echo "$resp" | jq -e '.result.content[] | select(.type == "resource") | .resource.uri | test("#markdown$")'
  # The example.com body is delivered through the HTML/MultiExtract path.
  echo "$resp" | jq -e '.result.content[] | select(.type == "resource") | .resource.text | contains("Example Domain")'
}

function web_fetch_raw_md_url_returns_body_and_toc { # @test
  require_firefox

  url="https://raw.githubusercontent.com/anthropics/anthropic-sdk-python/78c73600b714fcb036893768df8ee122f33d4cb3/README.md"
  call=$(jq -nc --arg url "$url" '{jsonrpc:"2.0",id:2,method:"tools/call",params:{name:"web-fetch",arguments:{url:$url,format:"markdown"}}}')
  result=$(printf '%s\n' "$INIT_MSG" "$INITIALIZED_MSG" "$call" |
    timeout 60 "$CHREST_BIN" mcp)

  resp=$(echo "$result" | grep '"id":2')
  echo "$resp" | jq -e '.result.isError != true'
  # Embedded markdown resource carries the raw .md body verbatim.
  echo "$resp" | jq -e '.result.content[] | select(.type == "resource") | .resource.text | contains("# Claude SDK for Python")'
  # New capability: TOC populated with real heading anchors built from
  # the raw markdown text (Firefox would otherwise wrap the body in
  # <pre> with no <h*> elements and yield an empty TOC).
  echo "$resp" | jq -e '.result.content[0].text | (contains("#installation") or contains("#getting-started"))'
}

function web_fetch_raw_toml_url_returns_text_body { # @test
  require_firefox

  url="https://raw.githubusercontent.com/anthropics/anthropic-sdk-python/78c73600b714fcb036893768df8ee122f33d4cb3/pyproject.toml"
  call=$(jq -nc --arg url "$url" '{jsonrpc:"2.0",id:2,method:"tools/call",params:{name:"web-fetch",arguments:{url:$url,format:"text"}}}')
  result=$(printf '%s\n' "$INIT_MSG" "$INITIALIZED_MSG" "$call" |
    timeout 60 "$CHREST_BIN" mcp)

  resp=$(echo "$result" | grep '"id":2')
  echo "$resp" | jq -e '.result.isError != true'
  # The text slot contains the raw toml.
  echo "$resp" | jq -e '.result.content[] | select(.type == "resource") | .resource.uri | test("#text$")'
  echo "$resp" | jq -e '.result.content[] | select(.type == "resource") | .resource.text | contains("[project]")'
  # No headings in toml; TOC block should be present (text type) but not crash.
  echo "$resp" | jq -e '.result.content[0].type == "text"'
}

function web_fetch_404_url_returns_structured_http_error { # @test
  require_firefox

  url="https://raw.githubusercontent.com/anthropics/anthropic-sdk-python/78c73600b714fcb036893768df8ee122f33d4cb3/THIS-DOES-NOT-EXIST"
  call=$(jq -nc --arg url "$url" '{jsonrpc:"2.0",id:2,method:"tools/call",params:{name:"web-fetch",arguments:{url:$url}}}')
  result=$(printf '%s\n' "$INIT_MSG" "$INITIALIZED_MSG" "$call" |
    timeout 60 "$CHREST_BIN" mcp)

  resp=$(echo "$result" | grep '"id":2')
  # Structured error envelope, not a synthesized "page" body.
  echo "$resp" | jq -e '.result.isError == true'
  echo "$resp" | jq -e '.result.content[0].text | contains("HTTP 404")'
  # MCP schema validation must NOT have fired — no resource_link / resource
  # blocks should be present in an error envelope.
  echo "$resp" | jq -e '[.result.content[] | select(.type == "resource_link")] | length == 0'
  echo "$resp" | jq -e '[.result.content[] | select(.type == "resource")] | length == 0'
  echo "$resp" | jq -e '.result.content[0].text | contains("invalid_union") | not'
}

function web_fetch_binary_url_returns_structured_refusal { # @test
  require_firefox

  url="https://github.com/anthropics/anthropic-sdk-python/archive/refs/tags/v0.97.0.tar.gz"
  call=$(jq -nc --arg url "$url" '{jsonrpc:"2.0",id:2,method:"tools/call",params:{name:"web-fetch",arguments:{url:$url}}}')
  result=$(printf '%s\n' "$INIT_MSG" "$INITIALIZED_MSG" "$call" |
    timeout 60 "$CHREST_BIN" mcp)

  resp=$(echo "$result" | grep '"id":2')
  # Structured error, not a parsed document.
  echo "$resp" | jq -e '.result.isError == true'
  # Refusal mentions the binary content-type so the caller knows why.
  echo "$resp" | jq -e '.result.content[0].text | contains("binary content-type")'
  # Crucially: MCP schema validation did not fire on a malformed
  # content block. The "invalid_union" string is what shows up in the
  # client when SDK schema parsing rejects the result.
  echo "$resp" | jq -e '.result.content[0].text | contains("invalid_union") | not'
}
