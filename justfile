
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

# Capture a small diverse page set across the three markdown variants so
# the output can be visually compared. Writes results to /tmp/md-samples/
# and echoes the list at the end. Uses the debug-tagged binary
# (go/build/release/chrest) because it's already built in the dev loop
# and firefox is on the dev shell PATH.
explore-markdown-samples:
  #!/usr/bin/env bash
  set -uo pipefail
  out_dir=/tmp/md-samples
  mkdir -p "$out_dir"
  CHREST=go/build/release/chrest
  capture() {
    local src=$1 url=$2 fmt=$3 sel=${4:-}
    local out="$out_dir/${src}-${fmt}.md"
    local -a selflag=()
    if [ -n "$sel" ]; then selflag=(--selector "$sel"); fi
    echo "== $src $fmt ==" >&2
    if timeout 45 "$CHREST" capture --format "$fmt" --browser firefox --url "$url" \
         "${selflag[@]}" --output "$out" >"$out_dir/${src}-${fmt}.stderr" 2>&1; then
      echo "  ok $(wc -c <"$out") bytes" >&2
    else
      echo "  FAIL (see $out_dir/${src}-${fmt}.stderr)" >&2
    fi
  }
  for fmt in markdown-full markdown-reader; do
    capture swblog https://simonwillison.net/2026/Feb/15/gwtar/ "$fmt"
    capture wiki   https://en.wikipedia.org/wiki/Markdown                                                                  "$fmt"
    capture mdn    https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Array/map              "$fmt"
    capture hn     https://news.ycombinator.com/item?id=46762667                                                            "$fmt"
  done
  capture swblog https://simonwillison.net/2026/Feb/15/gwtar/ markdown-selector article
  capture wiki   https://en.wikipedia.org/wiki/Markdown       markdown-selector '#bodyContent'
  capture mdn    https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Array/map markdown-selector main
  capture hn     https://news.ycombinator.com/item?id=46762667 markdown-selector '#hnmain'
  echo >&2
  echo "=== outputs ===" >&2
  ls -la "$out_dir"/*.md 2>/dev/null >&2 || true

[group: 'explore']
explore-vendor-dewey:
  #!/usr/bin/env bash
  set -euo pipefail
  src="/Users/sfriedenberg/.cache/go/pkg/mod/github.com/amarbel-llc/purse-first/libs/dewey@v0.0.4"
  dst="go/libs/dewey"
  pkgs=(
    "0/interfaces"
    "0/stack_frame"
    "0/primordial"
    "0/box_chars"
    "0/http_statuses"
    "alfa/pool"
    "alfa/cmp"
    "bravo/errors"
    "bravo/collections_slice"
    "charlie/ui"
    "charlie/ohio"
    "charlie/values"
    "charlie/flags"
    "charlie/quiter"
    "0/flag_policy"
    "delta/cli"
    "delta/collections_value"
    "golf/jsonrpc"
    "golf/transport"
    "golf/server"
    "golf/protocol"
    "golf/command"
  )
  for pkg in "${pkgs[@]}"; do
    mkdir -p "$dst/$pkg"
    for f in "$src/$pkg"/*.go; do
      [ -f "$f" ] || continue
      base=$(basename "$f")
      # Skip test files
      [[ "$base" == *_test.go ]] && continue
      cp "$f" "$dst/$pkg/$base"
    done
    echo "  copied $pkg ($(ls "$dst/$pkg"/*.go 2>/dev/null | wc -l) files)"
  done
  # Exclude golf/command/huh/ subpackage (charmbracelet dep, not used by chrest)
  echo "done — $dst populated"

[group: 'explore']
explore-mcp-v1-debug:
  #!/usr/bin/env bash
  set -euo pipefail
  v1_init='{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test","version":"0.0.1"}}}'
  notif='{"jsonrpc":"2.0","method":"notifications/initialized"}'
  list='{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}'
  result=$(printf '%s\n' "$v1_init" "$notif" "$list" | go/build/release/chrest mcp)
  echo "=== init response ==="
  echo "$result" | grep '"id":1' | jq .
  echo "=== tools/list response (web-fetch) ==="
  echo "$result" | grep '"id":2' | jq '[.result.tools[] | select(.name == "web-fetch")] | first'

[group: 'explore']
explore-rewrite-dewey-imports:
  #!/usr/bin/env bash
  set -euo pipefail
  old="github.com/amarbel-llc/purse-first/libs/dewey"
  new="code.linenisgreat.com/chrest/go/libs/dewey"
  # Rewrite vendored dewey files
  count=0
  while IFS= read -r f; do
    if grep -q "$old" "$f"; then
      sed -i'' "s|$old|$new|g" "$f"
      count=$((count + 1))
    fi
  done < <(find go/libs/dewey -name '*.go' -type f)
  echo "rewrote $count vendored files"
  # Rewrite chrest source files
  count2=0
  while IFS= read -r f; do
    if grep -q "$old" "$f"; then
      sed -i'' "s|$old|$new|g" "$f"
      count2=$((count2 + 1))
    fi
  done < <(find go/src go/cmd -name '*.go' -type f)
  echo "rewrote $count2 chrest source files"

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

# Drive chrest capture-batch against a real HTTP fixture and save
# every writer-stdin artifact to disk so envelope / spec / payload
# bytes can be visually reviewed. Use to sanity-check artifact shape
# against RFC 0001 after non-trivial capturebatch changes.
#
# Output goes under /tmp/chrest-envelope-review.<timestamp>/. Prints
# the batch output JSON + a categorized dump of every artifact.
explore-envelope-review format="text" browser="firefox" split="true":
  #!/usr/bin/env bash
  set -euo pipefail
  just build-go
  out_dir=$(mktemp -d "/tmp/chrest-envelope-review.XXXXXX")
  echo "review dir: $out_dir" >&2
  rec_dir="$out_dir/artifacts"
  mkdir -p "$rec_dir"

  # Recording writer: tee stdin to a file, emit a JSON ref.
  cat >"$out_dir/writer.sh" <<EOF
  #!/usr/bin/env bash
  out=\$(mktemp "$rec_dir/artifact.XXXXXX")
  cat > "\$out"
  size=\$(wc -c < "\$out")
  echo "{\"id\":\"blake2b256-rec-\$(basename \$out)\",\"size\":\$size}"
  EOF
  chmod +x "$out_dir/writer.sh"

  # Minimal HTML fixture in the same dir the server will serve.
  cat >"$out_dir/test.html" <<'HTML'
  <!doctype html>
  <html><head><title>envelope review</title></head>
  <body><h1>Hello from chrest</h1><p>Fixture for envelope-review recipe.</p></body>
  </html>
  HTML

  # Python http.server on an ephemeral port.
  port=$(python3 -c 'import socket;s=socket.socket();s.bind(("127.0.0.1",0));print(s.getsockname()[1]);s.close()')
  (cd "$out_dir" && python3 -m http.server "$port" >/dev/null 2>&1) &
  srv_pid=$!
  trap 'kill $srv_pid 2>/dev/null || true' EXIT
  for _ in $(seq 1 50); do
    curl -sf "http://127.0.0.1:$port/test.html" >/dev/null && break
    sleep 0.1
  done

  cat <<JSON > "$out_dir/input.json"
  {
    "schema": "web-capture-archive/v1",
    "writer": {"cmd": ["$out_dir/writer.sh"]},
    "url": "http://127.0.0.1:$port/test.html",
    "defaults": {"browser": "{{browser}}", "split": {{split}}},
    "captures": [{"name": "c", "format": "{{format}}"}]
  }
  JSON

  echo "=== batch output ===" >&2
  go/build/release/chrest capture-batch < "$out_dir/input.json" | tee "$out_dir/output.json" | jq '.'

  echo >&2
  echo "=== artifact classification ===" >&2
  for f in "$rec_dir"/artifact.*; do
    name=$(basename "$f")
    size=$(wc -c < "$f")
    magic=$(xxd -l 8 -p "$f" 2>/dev/null || true)
    # Try to decide artifact type from content.
    kind="unknown"
    if jq -e '.schema | startswith("web-capture-archive.envelope")' < "$f" >/dev/null 2>&1; then
      kind="envelope"
    elif jq -e '.schema | startswith("web-capture-archive.spec")' < "$f" >/dev/null 2>&1; then
      kind="spec"
    else
      case "$magic" in
        89504e47*) kind="payload-png" ;;
        25504446*) kind="payload-pdf" ;;
        *) kind="payload-other" ;;
      esac
    fi
    printf '%-18s %-8s %10s bytes  %s\n' "$name" "$kind" "$size" "$magic" >&2
  done

  echo >&2
  echo "=== pretty JSON artifacts ===" >&2
  for f in "$rec_dir"/artifact.*; do
    if jq -e 'type == "object"' < "$f" >/dev/null 2>&1; then
      echo "--- $(basename "$f") ---" >&2
      jq '.' < "$f" >&2
    fi
  done
  echo >&2
  echo "artifact files kept in: $rec_dir" >&2

# Decompress every FlateDecode stream in a PDF looking for one that
# contains the /Info dict fields (Producer / CreationDate / ModDate).
# Used while investigating chrest#27 — lets us see whether pdfcpu put
# the re-stamped /Info entries in plain text or inside a compressed
# object stream (answer: compressed). Keep as a debug tool.
explore-pdf-inspect-info pdf:
  #!/usr/bin/env python3
  import zlib, re
  with open("{{pdf}}", "rb") as f:
      b = f.read()
  found = False
  for m in re.finditer(rb"stream\r?\n(.*?)\r?\nendstream", b, re.DOTALL):
      try:
          dec = zlib.decompress(m.group(1))
      except Exception:
          continue
      if b"pdfcpu" in dec or b"CreationDate" in dec or b"ModDate" in dec:
          print("--- decompressed stream (len={} bytes) ---".format(len(dec)))
          print(dec[:2000].decode("latin-1", errors="replace"))
          found = True
  if not found:
      print("(no decompressed stream contained pdfcpu/CreationDate/ModDate)")

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
