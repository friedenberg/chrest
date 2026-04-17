#!/usr/bin/env bats

# Integration tests for Firefox headless capture via WebDriver BiDi.
# Requires firefox on PATH. Uses local fixture files only (no network).

setup() {
  load "$(dirname "$BATS_TEST_FILE")/common.bash"

  # Inline a minimal HTML fixture so it works inside the sandbox without
  # depending on file path resolution.
  cat > "$BATS_TEST_TMPDIR/test.html" <<'EOF'
<!doctype html>
<html><head><title>Test</title></head>
<body><h1>Hello from chrest</h1></body>
</html>
EOF
  FIXTURE="file://$BATS_TEST_TMPDIR/test.html"

  firefox="$(command -v firefox || command -v firefox-esr || true)"
  if [ -z "$firefox" ]; then
    skip "no Firefox found on PATH"
  fi
  if ! timeout 5 "$firefox" --headless --version >/dev/null 2>&1; then
    skip "headless Firefox not functional"
  fi
}

function firefox_capture_text_extracts_text { # @test
  result=$("$CHREST_BIN" capture-text --browser firefox --url "$FIXTURE")
  echo "$result" | grep -q "Hello from chrest"
}

function firefox_capture_pdf_returns_pdf_bytes { # @test
  result=$("$CHREST_BIN" capture-pdf --browser firefox --url "$FIXTURE" | head -c 5)
  [ "$result" = "%PDF-" ]
}

function firefox_capture_screenshot_returns_png_bytes { # @test
  result=$("$CHREST_BIN" capture-screenshot --browser firefox --url "$FIXTURE" | head -c 4 | xxd -p)
  [ "$result" = "89504e47" ]
}

function firefox_capture_mhtml_returns_unsupported_error { # @test
  run "$CHREST_BIN" capture-mhtml --browser firefox --url "$FIXTURE"
  echo "$output" | grep -qi "not supported"
}

function firefox_capture_a11y_returns_unsupported_error { # @test
  run "$CHREST_BIN" capture-a11y --browser firefox --url "$FIXTURE"
  echo "$output" | grep -qi "not supported"
}
