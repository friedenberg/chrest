
default: build test

build: build-go build-extension

reload: build
  go/build/release/chrest install jbcogiaaaaikinoljmplilmcnicpfoek
  chrest reload-extension

build-go:
  just go/build

build-extension:
  just extension/build

test: test-go test-mcp test-mcp-bats

test-go:
  just go/tests-go

mcp-bin := "go/build/release/chrest mcp"
mcp-inspect := "npx @modelcontextprotocol/inspector --cli"

test-mcp: build
  #!/usr/bin/env bash
  set -euo pipefail
  tools=$({{mcp-inspect}} --method tools/list {{mcp-bin}})
  resources=$({{mcp-inspect}} --method resources/list {{mcp-bin}})
  templates=$({{mcp-inspect}} --method resources/templates/list {{mcp-bin}})
  # Verify listings return valid JSON
  echo "$tools" | jq -e '.tools | length > 0'
  echo "$resources" | jq -e '.resources | length > 0'
  echo "$templates" | jq -e '.resourceTemplates | length > 0'
  # Verify readOnlyHint annotations
  for tool in browser-info list-windows get-window list-tabs get-tab list-extensions items-get state-get read-resource; do
    echo "$tools" | jq -e --arg t "$tool" '.tools[] | select(.name == $t) | .annotations.readOnlyHint == true' \
      || { echo "FAIL: $tool missing readOnlyHint"; exit 1; }
  done
  # Verify destructiveHint annotations
  for tool in close-window close-tab state-restore items-put; do
    echo "$tools" | jq -e --arg t "$tool" '.tools[] | select(.name == $t) | .annotations.destructiveHint == true' \
      || { echo "FAIL: $tool missing destructiveHint"; exit 1; }
  done
  echo "All MCP validations passed"

test-mcp-bats:
  bats --bin-dir go/build/release/ --allow-unix-sockets --allow-local-binding zz-tests_bats/

dev-install-mcp: build
  go/build/release/chrest install-mcp

demo:
  vhs demo/demo.tape
