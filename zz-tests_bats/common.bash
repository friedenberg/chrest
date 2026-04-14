bats_load_library bats-support
bats_load_library bats-assert
bats_load_library bats-assert-additions
bats_load_library bats-island
bats_load_library bats-emo

: "${CHREST_BIN:=$(command -v chrest)}"
require_bin CHREST_BIN chrest

TIMEOUT_SEC=5

INIT_MSG='{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"0.0.1"}}}'
INITIALIZED_MSG='{"jsonrpc":"2.0","method":"notifications/initialized"}'

# Send JSON-RPC messages (one per line) to chrest mcp, collect stdout.
# Usage: printf '%s\n' "$msg1" "$msg2" | run_mcp
run_mcp() {
  timeout "$TIMEOUT_SEC" "$CHREST_BIN" mcp
}
