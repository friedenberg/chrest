#!/usr/bin/env bats

# Integration tests for Firefox headless capture via WebDriver BiDi.
# Requires firefox on PATH. Uses local fixture files only (no network).

setup() {
  load "$(dirname "$BATS_TEST_FILE")/common.bash"

  # Inline a minimal HTML fixture so it works inside the sandbox without
  # depending on file path resolution.
  cat >"$BATS_TEST_TMPDIR/test.html" <<'EOF'
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
  result=$("$CHREST_BIN" capture --format text --browser firefox --url "$FIXTURE")
  echo "$result" | grep -q "Hello from chrest"
}

function firefox_capture_pdf_returns_pdf_bytes { # @test
  result=$("$CHREST_BIN" capture --format pdf --browser firefox --url "$FIXTURE" | head -c 5)
  [ "$result" = "%PDF-" ]
}

function firefox_capture_screenshot_returns_png_bytes { # @test
  result=$("$CHREST_BIN" capture --format screenshot-png --browser firefox --url "$FIXTURE" | head -c 4 | xxd -p)
  [ "$result" = "89504e47" ]
}

function firefox_capture_mhtml_returns_unsupported_error { # @test
  run "$CHREST_BIN" capture --format mhtml --browser firefox --url "$FIXTURE"
  echo "$output" | grep -qi "not supported"
}

function firefox_capture_a11y_returns_unsupported_error { # @test
  run "$CHREST_BIN" capture --format a11y --browser firefox --url "$FIXTURE"
  echo "$output" | grep -qi "not supported"
}

# Regression: PNG must end exactly at its IEND chunk. A trailing newline
# byte from fmt.Println would push the tail past 49454e44ae426082. See #21.
function firefox_capture_screenshot_has_no_trailing_newline { # @test
  "$CHREST_BIN" capture --format screenshot-png --browser firefox --url "$FIXTURE" > "$BATS_TEST_TMPDIR/out.png"
  tail=$(tail -c 8 "$BATS_TEST_TMPDIR/out.png" | xxd -p)
  [ "$tail" = "49454e44ae426082" ]
}

# Regression: PDF output ends with %%EOF + one trailing newline from the PDF
# itself. The CLI must not append a second newline. See #21.
function firefox_capture_pdf_has_no_trailing_newline { # @test
  "$CHREST_BIN" capture --format pdf --browser firefox --url "$FIXTURE" > "$BATS_TEST_TMPDIR/out.pdf"
  # last 6 bytes should be "%%EOF\n" (25 25 45 4f 46 0a), NOT "%EOF\n\n"
  tail=$(tail -c 6 "$BATS_TEST_TMPDIR/out.pdf" | xxd -p)
  [ "$tail" = "2525454f460a" ]
}
