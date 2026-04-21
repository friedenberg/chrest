#!/usr/bin/env bats

# Exploratory tests for chrest MCP server behavior under various socket
# conditions.

setup() {
  load "$(dirname "$BATS_TEST_FILE")/common.bash"
  setup_test_home
}

teardown() {
  teardown_test_home
}

function initialize_responds_with_no_sockets { # @test
  result=$(printf '%s\n' "$INIT_MSG" | run_mcp)
  echo "$result" | jq -e 'select(.id == 1) | .result.serverInfo.name == "chrest"'
}

function tools_list_returns_tools_with_no_sockets { # @test
  result=$(printf '%s\n' "$INIT_MSG" "$INITIALIZED_MSG" \
    '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}' | run_mcp)
  echo "$result" | grep '"id":2' | jq -e '.result.tools | length > 0'
}

function resources_list_returns_resources_with_no_sockets { # @test
  result=$(printf '%s\n' "$INIT_MSG" "$INITIALIZED_MSG" \
    '{"jsonrpc":"2.0","id":2,"method":"resources/list","params":{}}' | run_mcp)
  echo "$result" | grep '"id":2' | jq -e '.result.resources | length > 0'
}

function tools_call_list_windows_returns_quickly_with_no_sockets { # @test
  result=$(printf '%s\n' "$INIT_MSG" "$INITIALIZED_MSG" \
    '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"list-windows","arguments":{}}}' | run_mcp)
  echo "$result" | grep '"id":2' | jq -e '.result or .error'
}

function web_fetch_appears_in_tools_list { # @test
  result=$(printf '%s\n' "$INIT_MSG" "$INITIALIZED_MSG" \
    '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}' | run_mcp)
  echo "$result" | grep '"id":2' | jq -e '
    [.result.tools[] | select(.name == "web-fetch")] | length == 1
  '
}

# V1 negotiation exposes annotations (readOnlyHint). The shared INIT_MSG
# uses protocol 2025-03-26 which falls back to V0 where annotations are
# dropped, so this test uses the V1 version string explicitly.
function web_fetch_annotations_visible_under_v1 { # @test
  v1_init='{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test","version":"0.0.1"}}}'
  result=$(printf '%s\n' "$v1_init" "$INITIALIZED_MSG" \
    '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}' | run_mcp)
  tool=$(echo "$result" | grep '"id":2' | jq '[.result.tools[] | select(.name == "web-fetch")] | first')
  echo "DEBUG V1 tool: $tool" >&2
  echo "$tool" | jq -e '.annotations.readOnlyHint == true'
}

function web_fetch_rejects_missing_url { # @test
  result=$(printf '%s\n' "$INIT_MSG" "$INITIALIZED_MSG" \
    '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"web-fetch","arguments":{}}}' | run_mcp)
  echo "$result" | grep '"id":2' | jq -e '.result.isError == true'
}

function web_fetch_defaults_to_markdown_with_resource_links { # @test
  firefox="$(command -v firefox || command -v firefox-esr || true)"
  if [ -z "$firefox" ]; then
    skip "no Firefox found on PATH"
  fi
  if ! timeout 5 "$firefox" --headless --version >/dev/null 2>&1; then
    skip "headless Firefox not functional"
  fi

  cat >"$BATS_TEST_TMPDIR/fetch.html" <<'FIXTURE'
<!doctype html>
<html><head><title>Fetch Test</title></head>
<body><h1>Hello web-fetch</h1><p>Body text.</p></body>
</html>
FIXTURE
  url="file://$BATS_TEST_TMPDIR/fetch.html"

  result=$(printf '%s\n' "$INIT_MSG" "$INITIALIZED_MSG" \
    "{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/call\",\"params\":{\"name\":\"web-fetch\",\"arguments\":{\"url\":\"$url\"}}}" |
    timeout 20 "$CHREST_BIN" mcp)

  resp=$(echo "$result" | grep '"id":2')
  # 3 blocks: 1 embedded resource (markdown) + 2 resource_links (text, html)
  echo "$resp" | jq -e '.result.content | length == 3'
  # The embedded resource is markdown (fragment #markdown in URI)
  echo "$resp" | jq -e '.result.content[] | select(.type == "resource") | .resource.uri | test("#markdown$")'
  # The other two are resource_links
  echo "$resp" | jq -e '[.result.content[] | select(.type == "resource_link")] | length == 2'
}

function web_fetch_format_text_returns_text_inline { # @test
  firefox="$(command -v firefox || command -v firefox-esr || true)"
  if [ -z "$firefox" ]; then
    skip "no Firefox found on PATH"
  fi
  if ! timeout 5 "$firefox" --headless --version >/dev/null 2>&1; then
    skip "headless Firefox not functional"
  fi

  cat >"$BATS_TEST_TMPDIR/fetch.html" <<'FIXTURE'
<!doctype html>
<html><head><title>Fetch Test</title></head>
<body><h1>Hello web-fetch</h1><p>Body text.</p></body>
</html>
FIXTURE
  url="file://$BATS_TEST_TMPDIR/fetch.html"

  result=$(printf '%s\n' "$INIT_MSG" "$INITIALIZED_MSG" \
    "{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/call\",\"params\":{\"name\":\"web-fetch\",\"arguments\":{\"url\":\"$url\",\"format\":\"text\"}}}" |
    timeout 20 "$CHREST_BIN" mcp)

  resp=$(echo "$result" | grep '"id":2')
  # The embedded resource is text (fragment #text in URI)
  echo "$resp" | jq -e '.result.content[] | select(.type == "resource") | .resource.uri | test("#text$")'
  # Markdown and html are resource_links
  echo "$resp" | jq -e '[.result.content[] | select(.type == "resource_link")] | length == 2'
}

function tools_call_list_windows_handles_stale_socket { # @test
  # Override XDG_STATE_HOME to a short /tmp path to avoid AF_UNIX 108-char limit
  short_state="$(mktemp -d /tmp/chrest-bats.XXXXXX)"
  export XDG_STATE_HOME="$short_state"
  mkdir -p "$short_state/chrest"
  sock_path="$short_state/chrest/firefox-stale.sock"

  # Create a socket file that nothing is listening on
  python3 -c "
import socket
s = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
s.bind('$sock_path')
s.close()
"

  result=$(printf '%s\n' "$INIT_MSG" "$INITIALIZED_MSG" \
    '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"list-windows","arguments":{}}}' | run_mcp)
  rm -rf "$short_state"
  echo "$result" | grep '"id":2' | jq -e '.result or .error'
}
