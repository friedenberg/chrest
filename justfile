
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

# End-to-end sanity for `chrest capture-batch` using the RFC 0001
# fixture + writer stub landed in ~/eng/aim/ by the nebulous session.
# Pipes the example batch input through chrest, pretty-prints the
# output JSON, and echoes any stderr chrest emitted. Intended to be
# re-run after chrest changes to verify the cross-session contract
# still matches.
explore-capture-batch input="/home/sasha/eng/aim/fixtures/batch-input.example.json":
  #!/usr/bin/env bash
  set -euo pipefail
  err=$(mktemp)
  trap 'rm -f "$err"' EXIT
  go/build/release/chrest capture-batch < "{{input}}" 2>"$err" | jq '.'
  if [ -s "$err" ]; then
    echo "--- chrest stderr ---" >&2
    cat "$err" >&2
  fi

# Print chrest's help text (both top-level and per-command) so we can
# verify command discoverability after any registration changes.
explore-help subcommand="":
  #!/usr/bin/env bash
  set -euo pipefail
  if [ -n "{{subcommand}}" ]; then
    go/build/release/chrest {{subcommand}} --help
  else
    go/build/release/chrest --help
  fi

# Run chrest-jcs on a shared byte-stability fixture and compare the
# sha256 against the remote implementation's hash. Output file lives
# next to the input in the aim/ directory so other sessions can diff
# it. Hash printed to stdout and written beside the output file.
explore-jcs-fixture vector="jcs-spec-vector-1" expected="":
  #!/usr/bin/env bash
  set -euo pipefail
  fixtures=/home/sasha/eng/aim/fixtures
  input="$fixtures/{{vector}}.input.json"
  output="$fixtures/{{vector}}.chrest.canonical.json"
  if [ ! -f "$input" ]; then
    echo "missing input: $input" >&2
    exit 1
  fi
  go/build/release/chrest-jcs < "$input" > "$output"
  got=$(sha256sum "$output" | awk '{print $1}')
  echo "output=$output"
  echo "sha256=$got"
  if [ -n "{{expected}}" ]; then
    if [ "$got" = "{{expected}}" ]; then
      echo "MATCH (expected $got)"
    else
      echo "MISMATCH (expected {{expected}}, got $got)" >&2
      exit 2
    fi
  fi
