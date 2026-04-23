#!/usr/bin/env bats

# Integration tests for CDP capture commands.
# Requires a working headless Chrome/Chromium on PATH.
# Tests are skipped if headless Chrome crashes (e.g. kernel 6.17 + Chrome
# seccomp/crashpad incompatibility).

setup() {
  load "$(dirname "$BATS_TEST_FILE")/common.bash"
  FIXTURE="file://$(cd "$(dirname "$BATS_TEST_FILE")" && pwd)/fixtures/test.html"

  # chrest#14: headless Chrome CDP-over-websocket is non-functional on
  # this host (kernel 6.17 seccomp/crashpad incompatibility). Probes
  # like `chrome --dump-dom` pass but the real CDP handshake fails with
  # "bad status", so the per-test probe was unreliable. Skip the whole
  # file until Chrome is fixed upstream.
  skip "headless Chrome CDP not functional (chrest#14)"
}

function capture_pdf_returns_pdf_bytes { # @test
  result=$("$CHREST_BIN" capture --format pdf --browser chrome --url "$FIXTURE" | head -c 5)
  [ "$result" = "%PDF-" ]
}

function capture_screenshot_returns_png_bytes { # @test
  result=$("$CHREST_BIN" capture --format screenshot-png --browser chrome --url "$FIXTURE" | head -c 4 | xxd -p)
  [ "$result" = "89504e47" ]
}

function capture_text_extracts_text { # @test
  result=$("$CHREST_BIN" capture --format text --browser chrome --url "$FIXTURE")
  echo "$result" | grep -q "Hello from chrest"
}

function capture_a11y_returns_json { # @test
  result=$("$CHREST_BIN" capture --format a11y --browser chrome --url "$FIXTURE")
  echo "$result" | jq -e '.nodes | length > 0'
}

function capture_mhtml_returns_mhtml { # @test
  result=$("$CHREST_BIN" capture --format mhtml --browser chrome --url "$FIXTURE" | head -20)
  echo "$result" | grep -q "Content-Type"
}
