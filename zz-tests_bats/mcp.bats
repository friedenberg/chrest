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
