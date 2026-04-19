#!/usr/bin/env bats

# Integration tests for `chrest capture-batch` (RFC 0001 — Web Capture
# Archive Protocol, capturer role).

setup() {
  load "$(dirname "$BATS_TEST_FILE")/common.bash"

  # Minimal HTML fixture written into the bats tempdir so the bwrap
  # sandbox can read it.
  cat >"$BATS_TEST_TMPDIR/test.html" <<'EOF'
<!doctype html>
<html><head><title>Test</title></head>
<body><h1>Hello from chrest</h1></body>
</html>
EOF
  FIXTURE="file://$BATS_TEST_TMPDIR/test.html"

  # Stub writer: read stdin, emit a deterministic JSON result line.
  # Simulates the madder writer interface without requiring madder in
  # the test closure.
  cat >"$BATS_TEST_TMPDIR/stub-writer.sh" <<'EOF'
#!/usr/bin/env bash
size=$(wc -c)
echo "{\"id\":\"blake2b256-stub-${size}\",\"size\":${size}}"
EOF
  chmod +x "$BATS_TEST_TMPDIR/stub-writer.sh"
  STUB_WRITER="$BATS_TEST_TMPDIR/stub-writer.sh"

  firefox="$(command -v firefox || command -v firefox-esr || true)"
  if [ -z "$firefox" ]; then
    skip "no Firefox found on PATH"
  fi
  if ! timeout 5 "$firefox" --headless --version >/dev/null 2>&1; then
    skip "headless Firefox not functional"
  fi
}

function capture_batch_rejects_bad_schema { # @test
  input='{"schema":"wrong/v1","writer":{"cmd":["/bin/true"]},"url":"about:blank","captures":[]}'
  run bash -c "echo '$input' | timeout 10 '$CHREST_BIN' capture-batch"
  [ "$status" -ne 0 ]
  echo "$output" | grep -qi "schema"
}

function capture_batch_split_true_pdf_emits_all_three_artifacts { # @test
  # Stage 3 of chrest#22: PDF normalizer via pdfcpu. Expect payload
  # (normalized), envelope, and spec refs all populated. Normalized
  # PDF has /CreationDate, /ModDate, /Producer, /Creator stripped
  # plus the trailer /ID zeroed.
  input=$(
    cat <<JSON
{
  "schema": "web-capture-archive/v1",
  "writer": {"cmd": ["$STUB_WRITER"]},
  "url": "$FIXTURE",
  "defaults": {"browser": "firefox", "split": true},
  "captures": [{"name": "pdf", "format": "pdf"}]
}
JSON
  )
  result=$(echo "$input" | timeout 60 "$CHREST_BIN" capture-batch)
  echo "$result" | jq -e '.captures[0].error == null'
  echo "$result" | jq -e '.captures[0].payload.normalized == true'
  echo "$result" | jq -e '.captures[0].payload.media_type  == "application/pdf"'
  echo "$result" | jq -e '.captures[0].envelope.media_type == "application/vnd.web-capture-archive.envelope+json"'
  echo "$result" | jq -e '.captures[0].spec.media_type     == "application/vnd.web-capture-archive.spec+json"'
  echo "$result" | jq -e '.captures[0].payload.size > 100'
}

function capture_batch_split_true_mhtml_returns_not_implemented { # @test
  # mhtml and a11y are the only formats left unimplemented for split=true.
  input=$(
    cat <<JSON
{
  "schema": "web-capture-archive/v1",
  "writer": {"cmd": ["$STUB_WRITER"]},
  "url": "$FIXTURE",
  "defaults": {"browser": "firefox", "split": true},
  "captures": [{"name": "m", "format": "mhtml"}]
}
JSON
  )
  result=$(echo "$input" | timeout 30 "$CHREST_BIN" capture-batch)
  echo "$result" | jq -e '.captures[0].error.kind == "not-implemented"'
}

function capture_batch_split_true_screenshot_emits_all_three_artifacts { # @test
  # Stage 2 of chrest#22: PNG normalizer. Expect payload (normalized),
  # envelope, and spec refs all populated. Normalized payload should
  # differ in size from the raw capture because chunks get stripped.
  input=$(
    cat <<JSON
{
  "schema": "web-capture-archive/v1",
  "writer": {"cmd": ["$STUB_WRITER"]},
  "url": "$FIXTURE",
  "defaults": {"browser": "firefox", "split": true},
  "captures": [{"name": "shot", "format": "screenshot"}]
}
JSON
  )
  result=$(echo "$input" | timeout 30 "$CHREST_BIN" capture-batch)
  echo "$result" | jq -e '.captures[0].error == null'
  echo "$result" | jq -e '.captures[0].payload.normalized == true'
  echo "$result" | jq -e '.captures[0].payload.media_type  == "image/png"'
  echo "$result" | jq -e '.captures[0].envelope.media_type == "application/vnd.web-capture-archive.envelope+json"'
  echo "$result" | jq -e '.captures[0].spec.media_type     == "application/vnd.web-capture-archive.spec+json"'
  echo "$result" | jq -e '.captures[0].payload.size > 100'
}

function capture_batch_split_true_text_emits_all_three_artifacts { # @test
  # Stage 1 of chrest#22: text normalizer + partial envelope.
  # Expect payload (normalized), envelope, and spec refs all populated.
  input=$(
    cat <<JSON
{
  "schema": "web-capture-archive/v1",
  "writer": {"cmd": ["$STUB_WRITER"]},
  "url": "$FIXTURE",
  "defaults": {"browser": "firefox", "split": true},
  "captures": [{"name": "txt", "format": "text"}]
}
JSON
  )
  result=$(echo "$input" | timeout 30 "$CHREST_BIN" capture-batch)
  echo "$result" | jq -e '.captures[0].error == null'
  # All three artifact refs populated.
  echo "$result" | jq -e '.captures[0].payload.id  | startswith("blake2b256-stub-")'
  echo "$result" | jq -e '.captures[0].envelope.id | startswith("blake2b256-stub-")'
  echo "$result" | jq -e '.captures[0].spec.id     | startswith("blake2b256-stub-")'
  # Payload normalized flag set true.
  echo "$result" | jq -e '.captures[0].payload.normalized == true'
  echo "$result" | jq -e '.captures[0].payload.media_type  == "text/plain; charset=utf-8"'
  echo "$result" | jq -e '.captures[0].envelope.media_type == "application/vnd.web-capture-archive.envelope+json"'
  echo "$result" | jq -e '.captures[0].spec.media_type     == "application/vnd.web-capture-archive.spec+json"'
}

function capture_batch_split_false_text_emits_payload_and_spec { # @test
  input=$(
    cat <<JSON
{
  "schema": "web-capture-archive/v1",
  "writer": {"cmd": ["$STUB_WRITER"]},
  "url": "$FIXTURE",
  "defaults": {"browser": "firefox", "split": false},
  "captures": [{"name": "txt", "format": "text"}]
}
JSON
  )
  result=$(echo "$input" | timeout 30 "$CHREST_BIN" capture-batch)
  echo "$result" | jq -e '.captures[0].name == "txt"'
  echo "$result" | jq -e '.captures[0].payload.id | startswith("blake2b256-stub-")'
  echo "$result" | jq -e '.captures[0].payload.media_type == "text/plain; charset=utf-8"'
  echo "$result" | jq -e '.captures[0].spec.id | startswith("blake2b256-stub-")'
  echo "$result" | jq -e '.captures[0].spec.media_type == "application/vnd.web-capture-archive.spec+json"'
  # Envelope MUST be omitted when split=false.
  echo "$result" | jq -e '.captures[0].envelope == null'
  echo "$result" | jq -e '.captures[0].error == null'
}

function capture_batch_split_false_screenshot_emits_png { # @test
  input=$(
    cat <<JSON
{
  "schema": "web-capture-archive/v1",
  "writer": {"cmd": ["$STUB_WRITER"]},
  "url": "$FIXTURE",
  "defaults": {"browser": "firefox", "split": false},
  "captures": [{"name": "shot", "format": "screenshot"}]
}
JSON
  )
  result=$(echo "$input" | timeout 30 "$CHREST_BIN" capture-batch)
  echo "$result" | jq -e '.captures[0].payload.media_type == "image/png"'
  echo "$result" | jq -e '.captures[0].payload.size > 100'
}
