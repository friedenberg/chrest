
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
  for tool in browser-info list-windows get-window list-tabs get-tab list-extensions items-get state-get read-resource capture; do
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
  #!/usr/bin/env bash
  # Wrap bats in a hard wall-clock timeout because bats has been observed
  # to hang on post-test shutdown in bwrap --unshare-pid sandboxes after
  # several Firefox captures, even though every individual test passes.
  # Root cause is still open; this guard keeps `just test` finite.
  # We validate the TAP output ourselves: if the plan line `1..N` is
  # present and every line 1..N is `ok`, the suite succeeded regardless
  # of whether bats itself exited cleanly.
  set +e
  out=$(mktemp)
  trap 'rm -f "$out"' EXIT
  timeout --preserve-status 120 \
    bats --bin-dir go/build/release/ --allow-unix-sockets --allow-local-binding zz-tests_bats/ \
    > >(tee "$out") 2>&1
  bats_rc=$?
  expected=$(grep -m1 -E '^1\.\.[0-9]+$' "$out" | sed 's/^1\.\.//')
  passing=$(grep -cE '^ok [0-9]+ ' "$out")
  failing=$(grep -cE '^not ok [0-9]+ ' "$out")
  if [ -z "$expected" ]; then
    echo "FAIL: no TAP plan line (bats exit $bats_rc)"; exit 1
  fi
  if [ "$failing" -gt 0 ]; then
    echo "FAIL: $failing test(s) failed"; exit 1
  fi
  if [ "$passing" -ne "$expected" ]; then
    echo "FAIL: expected $expected, saw $passing ok (bats exit $bats_rc)"; exit 1
  fi
  echo "PASS: $expected tests ok (bats exit $bats_rc)"

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

explore-capture format="text" browser="firefox" url="https://example.com" output="":
  #!/usr/bin/env bash
  set -euo pipefail
  nix build
  if [ -n "{{output}}" ]; then
    timeout 30 result/bin/chrest capture --format {{format}} --browser {{browser}} --url "{{url}}" > "{{output}}"
    echo "wrote {{output}}"
  else
    timeout 30 result/bin/chrest capture --format {{format}} --browser {{browser}} --url "{{url}}"
  fi

explore-client +httpie_args:
  go/build/release/chrest client {{httpie_args}}