
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
  for tool in browser-info list-windows get-window list-tabs get-tab list-extensions items-get state-get read-resource capture-pdf capture-screenshot capture-mhtml capture-a11y capture-text; do
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

[group: 'explore']
explore-setup browser="chrome":
  just build
  go/build/release/chrest init --browser {{browser}} --name primary

explore-run browser="chrome":
  #!/usr/bin/env bash
  set -euo pipefail
  if [ "{{browser}}" = "firefox" ]; then
    web-ext run --target firefox-desktop --source-dir extension/dist-firefox
  else
    web-ext run --target chromium --source-dir extension/dist-chrome --start-url "chrome://extensions"
  fi

explore-capture command="capture-text" browser="firefox" url="https://example.com": build-go
  timeout 30 go/build/release/chrest {{command}} --browser {{browser}} --url {{url}}

explore-client +httpie_args:
  go/build/release/chrest client {{httpie_args}}