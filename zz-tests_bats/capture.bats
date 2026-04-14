#!/usr/bin/env bats

# Integration tests for CDP capture commands.
# Requires chromium on PATH.

setup() {
  load "$(dirname "$BATS_TEST_FILE")/common.bash"
  FIXTURE="file://$(cd "$(dirname "$BATS_TEST_FILE")" && pwd)/fixtures/test.html"
}

function capture_pdf_returns_pdf_bytes { # @test
  result=$("$CHREST_BIN" capture-pdf --url "$FIXTURE" | head -c 5)
  [ "$result" = "%PDF-" ]
}

function capture_screenshot_returns_png_bytes { # @test
  result=$("$CHREST_BIN" capture-screenshot --url "$FIXTURE" | head -c 4 | xxd -p)
  [ "$result" = "89504e47" ]
}

function capture_text_extracts_text { # @test
  result=$("$CHREST_BIN" capture-text --url "$FIXTURE")
  echo "$result" | grep -q "Hello from chrest"
}

function capture_a11y_returns_json { # @test
  result=$("$CHREST_BIN" capture-a11y --url "$FIXTURE")
  echo "$result" | jq -e '.nodes | length > 0'
}

function capture_mhtml_returns_mhtml { # @test
  result=$("$CHREST_BIN" capture-mhtml --url "$FIXTURE" | head -20)
  echo "$result" | grep -q "Content-Type"
}
