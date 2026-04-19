#!/usr/bin/env bats

# Integration tests for CDP capture commands.
# Requires a working headless Chrome/Chromium on PATH.
# Tests are skipped if headless Chrome crashes (e.g. kernel 6.17 + Chrome
# seccomp/crashpad incompatibility).

setup() {
  load "$(dirname "$BATS_TEST_FILE")/common.bash"
  FIXTURE="file://$(cd "$(dirname "$BATS_TEST_FILE")" && pwd)/fixtures/test.html"

  # Skip all capture tests if headless Chrome is non-functional.
  chrome="$(command -v chromium || command -v google-chrome-stable || command -v google-chrome || true)"
  if [ -z "$chrome" ]; then
    skip "no Chrome/Chromium found on PATH"
  fi
  if ! timeout 5 "$chrome" --headless=new --no-sandbox --dump-dom about:blank >/dev/null 2>&1; then
    skip "headless Chrome not functional (crash or timeout)"
  fi
}

function capture_pdf_returns_pdf_bytes { # @test
  result=$("$CHREST_BIN" capture --format pdf --url "$FIXTURE" | head -c 5)
  [ "$result" = "%PDF-" ]
}

function capture_screenshot_returns_png_bytes { # @test
  result=$("$CHREST_BIN" capture --format screenshot-png --url "$FIXTURE" | head -c 4 | xxd -p)
  [ "$result" = "89504e47" ]
}

function capture_text_extracts_text { # @test
  result=$("$CHREST_BIN" capture --format text --url "$FIXTURE")
  echo "$result" | grep -q "Hello from chrest"
}

function capture_a11y_returns_json { # @test
  result=$("$CHREST_BIN" capture --format a11y --url "$FIXTURE")
  echo "$result" | jq -e '.nodes | length > 0'
}

function capture_mhtml_returns_mhtml { # @test
  result=$("$CHREST_BIN" capture --format mhtml --url "$FIXTURE" | head -20)
  echo "$result" | grep -q "Content-Type"
}
