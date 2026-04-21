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

# Per-test timeout on every chrest invocation: in the bwrap --unshare-pid
# sandbox, sequences of Firefox launches have exhibited a post-test bats
# hang whose root cause is still open. Bounding each chrest call with
# `timeout` ensures bats cannot stall indefinitely waiting on a child.
FIREFOX_TEST_TIMEOUT=15

function firefox_capture_text_extracts_text { # @test
  result=$(timeout "$FIREFOX_TEST_TIMEOUT" "$CHREST_BIN" capture --format text --browser firefox --url "$FIXTURE")
  echo "$result" | grep -q "Hello from chrest"
}

function firefox_capture_pdf_returns_pdf_bytes { # @test
  result=$(timeout "$FIREFOX_TEST_TIMEOUT" "$CHREST_BIN" capture --format pdf --browser firefox --url "$FIXTURE" | head -c 5)
  [ "$result" = "%PDF-" ]
}

function firefox_capture_screenshot_returns_png_bytes { # @test
  result=$(timeout "$FIREFOX_TEST_TIMEOUT" "$CHREST_BIN" capture --format screenshot-png --browser firefox --url "$FIXTURE" | head -c 4 | xxd -p)
  [ "$result" = "89504e47" ]
}

function firefox_capture_mhtml_returns_unsupported_error { # @test
  run timeout "$FIREFOX_TEST_TIMEOUT" "$CHREST_BIN" capture --format mhtml --browser firefox --url "$FIXTURE"
  echo "$output" | grep -qi "not supported"
}

function firefox_capture_a11y_returns_unsupported_error { # @test
  run timeout "$FIREFOX_TEST_TIMEOUT" "$CHREST_BIN" capture --format a11y --browser firefox --url "$FIXTURE"
  echo "$output" | grep -qi "not supported"
}

# Regression: PNG must end exactly at its IEND chunk. A trailing newline
# byte from fmt.Println would push the tail past 49454e44ae426082. See #21.
function firefox_capture_screenshot_has_no_trailing_newline { # @test
  timeout "$FIREFOX_TEST_TIMEOUT" "$CHREST_BIN" capture --format screenshot-png --browser firefox --url "$FIXTURE" >"$BATS_TEST_TMPDIR/out.png"
  tail=$(tail -c 8 "$BATS_TEST_TMPDIR/out.png" | xxd -p)
  [ "$tail" = "49454e44ae426082" ]
}

# Regression: PDF output ends with %%EOF + one trailing newline from the PDF
# itself. The CLI must not append a second newline. See #21.
function firefox_capture_pdf_has_no_trailing_newline { # @test
  timeout "$FIREFOX_TEST_TIMEOUT" "$CHREST_BIN" capture --format pdf --browser firefox --url "$FIXTURE" >"$BATS_TEST_TMPDIR/out.pdf"
  # last 6 bytes should be "%%EOF\n" (25 25 45 4f 46 0a), NOT "%EOF\n\n"
  tail=$(tail -c 6 "$BATS_TEST_TMPDIR/out.pdf" | xxd -p)
  [ "$tail" = "2525454f460a" ]
}

# html-monolith: pipe the rendered DOM through monolith, which inlines every
# asset as data: URIs and returns a self-contained .html document. Skip the
# case gracefully if the monolith binary isn't on PATH (chrest#26).
function firefox_capture_html_monolith_returns_html { # @test
  if ! command -v monolith >/dev/null 2>&1; then
    skip "monolith binary not found on PATH"
  fi
  timeout "$FIREFOX_TEST_TIMEOUT" "$CHREST_BIN" capture --format html-monolith --browser firefox --url "$FIXTURE" >"$BATS_TEST_TMPDIR/out.html"
  # Monolith preserves the <html> tag and the document content. Look for
  # the fixture's h1 text, not a byte prefix — monolith may emit a
  # preamble (doctype, comments) before <html>.
  grep -q "Hello from chrest" "$BATS_TEST_TMPDIR/out.html"
  grep -qi "<html" "$BATS_TEST_TMPDIR/out.html"
}

# Regression: monolith emits "</html>\n" (one trailing newline from
# the HTML itself). The CLI must not append a second newline. Mirrors
# the PDF trailing-newline check above; see #21.
function firefox_capture_html_monolith_has_no_trailing_newline { # @test
  if ! command -v monolith >/dev/null 2>&1; then
    skip "monolith binary not found on PATH"
  fi
  timeout "$FIREFOX_TEST_TIMEOUT" "$CHREST_BIN" capture --format html-monolith --browser firefox --url "$FIXTURE" >"$BATS_TEST_TMPDIR/out.html"
  # last 8 bytes should be "</html>\n" (3c 2f 68 74 6d 6c 3e 0a), NOT "</html>\n\n"
  tail=$(tail -c 8 "$BATS_TEST_TMPDIR/out.html" | od -An -t x1 | tr -d ' \n')
  [ "$tail" = "3c2f68746d6c3e0a" ]
}

# html-outer: raw outerHTML from the rendered DOM, no asset inlining.
function firefox_capture_html_outer_returns_html { # @test
  timeout "$FIREFOX_TEST_TIMEOUT" "$CHREST_BIN" capture --format html-outer --browser firefox --url "$FIXTURE" >"$BATS_TEST_TMPDIR/out.html"
  grep -q "Hello from chrest" "$BATS_TEST_TMPDIR/out.html"
  grep -qi "<html" "$BATS_TEST_TMPDIR/out.html"
}

# Default backend is firefox (no --browser flag). Proves a user can reach
# the firefox path without explicit opt-in.
function capture_default_backend_is_firefox { # @test
  result=$(timeout "$FIREFOX_TEST_TIMEOUT" "$CHREST_BIN" capture --format text --url "$FIXTURE")
  echo "$result" | grep -q "Hello from chrest"
}

# --output writes the capture to a file atomically and exits 0 on success.
function capture_output_flag_writes_file { # @test
  out="$BATS_TEST_TMPDIR/out.txt"
  timeout "$FIREFOX_TEST_TIMEOUT" "$CHREST_BIN" capture --format text --browser firefox --url "$FIXTURE" --output "$out"
  [ -f "$out" ]
  grep -q "Hello from chrest" "$out"
  # No leftover tmpfile next to the target.
  tmp_count=$(find "$BATS_TEST_TMPDIR" -maxdepth 1 -name '.chrest-capture-*' | wc -l)
  [ "$tmp_count" = "0" ]
}

# --output with a failing capture (unknown browser backend) must:
#   - exit non-zero (proves the top-level exit-code fix)
#   - leave no file at the target path
#   - leave no tmpfile behind
function capture_output_atomic_cleanup_on_failure { # @test
  out="$BATS_TEST_TMPDIR/should-not-exist.txt"
  run timeout "$FIREFOX_TEST_TIMEOUT" "$CHREST_BIN" capture --format text --browser bogus --url "$FIXTURE" --output "$out"
  [ "$status" -ne 0 ]
  [ ! -f "$out" ]
  tmp_count=$(find "$BATS_TEST_TMPDIR" -maxdepth 1 -name '.chrest-capture-*' | wc -l)
  [ "$tmp_count" = "0" ]
}

# A richer fixture with navigation / article / footer regions so
# readability has enough scoring signal to strip the boilerplate. The
# default fixture from setup() is too small to give a meaningful
# reader-mode extraction.
function _write_markdown_fixture {
  cat >"$BATS_TEST_TMPDIR/article.html" <<'EOF'
<!doctype html>
<html>
<head><title>Chrest Markdown Fixture</title></head>
<body>
<nav>Site Nav - <a href="/">Home</a> - <a href="/x">X</a></nav>
<main>
<article>
<h1>The Real Article Title</h1>
<p>First paragraph of the real article body. It needs enough sentences for readability to score it as the main content region. Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore.</p>
<p>Second paragraph with more substantive text to reinforce the score. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.</p>
<p>Third paragraph rounds out the article. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur.</p>
</article>
</main>
<footer>Site Footer - Copyright 2026</footer>
</body>
</html>
EOF
}

function firefox_capture_markdown_full_includes_boilerplate { # @test
  _write_markdown_fixture
  result=$(timeout "$FIREFOX_TEST_TIMEOUT" "$CHREST_BIN" capture --format markdown-full --browser firefox --url "file://$BATS_TEST_TMPDIR/article.html")
  # Full page markdown should contain the heading AND the boilerplate.
  echo "$result" | grep -q "# The Real Article Title"
  echo "$result" | grep -q "Site Nav"
  echo "$result" | grep -q "Site Footer"
}

function firefox_capture_markdown_reader_strips_boilerplate { # @test
  _write_markdown_fixture
  result=$(timeout "$FIREFOX_TEST_TIMEOUT" "$CHREST_BIN" capture --format markdown-reader --browser firefox --url "file://$BATS_TEST_TMPDIR/article.html")
  # Reader-mode should keep the article body and drop the nav / footer.
  echo "$result" | grep -q "First paragraph of the real article body"
  if echo "$result" | grep -q "Site Nav"; then
    echo "reader kept nav boilerplate:" >&2
    echo "$result" >&2
    return 1
  fi
  if echo "$result" | grep -q "Site Footer"; then
    echo "reader kept footer boilerplate:" >&2
    echo "$result" >&2
    return 1
  fi
}

function firefox_capture_markdown_selector_scopes { # @test
  _write_markdown_fixture
  result=$(timeout "$FIREFOX_TEST_TIMEOUT" "$CHREST_BIN" capture --format markdown-selector --selector "footer" --browser firefox --url "file://$BATS_TEST_TMPDIR/article.html")
  # Footer content should be present; article body should be absent.
  echo "$result" | grep -q "Site Footer"
  if echo "$result" | grep -q "First paragraph of the real article body"; then
    echo "selector leaked article body:" >&2
    echo "$result" >&2
    return 1
  fi
}

# Negative-path tests: master's exit-code fix means chrest capture now
# returns non-zero on validation/handler errors, so we assert on both
# $status and the diagnostic text.
function firefox_capture_markdown_selector_rejects_empty { # @test
  _write_markdown_fixture
  run timeout "$FIREFOX_TEST_TIMEOUT" "$CHREST_BIN" capture --format markdown-selector --browser firefox --url "file://$BATS_TEST_TMPDIR/article.html"
  [ "$status" -ne 0 ]
  echo "$output" | grep -qi "selector is required"
}

function firefox_capture_markdown_selector_no_match { # @test
  _write_markdown_fixture
  run timeout "$FIREFOX_TEST_TIMEOUT" "$CHREST_BIN" capture --format markdown-selector --selector ".does-not-exist" --browser firefox --url "file://$BATS_TEST_TMPDIR/article.html"
  [ "$status" -ne 0 ]
  echo "$output" | grep -q ".does-not-exist"
}

function firefox_capture_markdown_reader_browser_engine_unimplemented { # @test
  run timeout "$FIREFOX_TEST_TIMEOUT" "$CHREST_BIN" capture --format markdown-reader --reader-engine browser --browser firefox --url "$FIXTURE"
  [ "$status" -ne 0 ]
  echo "$output" | grep -qi "not yet implemented"
}

# Multi-format: comma-separated --format writes each to a directory.
function firefox_capture_multi_format_writes_directory { # @test
  out="$BATS_TEST_TMPDIR/multi"
  timeout "$FIREFOX_TEST_TIMEOUT" "$CHREST_BIN" capture --format text,html-outer --browser firefox --url "$FIXTURE" --output "$out"
  [ -d "$out" ]
  [ -f "$out/text.txt" ]
  [ -f "$out/html-outer.html" ]
  grep -q "Hello from chrest" "$out/text.txt"
  grep -qi "<html" "$out/html-outer.html"
}

# Multi-format without --output must fail.
function firefox_capture_multi_format_requires_output { # @test
  run timeout "$FIREFOX_TEST_TIMEOUT" "$CHREST_BIN" capture --format text,html-outer --browser firefox --url "$FIXTURE"
  [ "$status" -ne 0 ]
  echo "$output" | grep -qi "output.*required"
}
