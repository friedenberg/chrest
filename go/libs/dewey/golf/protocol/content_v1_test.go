package protocol

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestEmbeddedTextResourceContent_EmptyTextRoundTripsAsExplicitField
// pins the marshal contract that closes chrest#65: when the helper is
// called with an empty `text`, the JSON output MUST include a
// `"text": ""` field rather than eliding it. Earlier versions used
// `string` + `json:",omitempty"`, which dropped the field entirely
// and made the resource look like neither a TextResourceContents nor
// a BlobResourceContents to MCP clients.
func TestEmbeddedTextResourceContent_EmptyTextRoundTripsAsExplicitField(t *testing.T) {
	block := EmbeddedTextResourceContent("web-fetch://example.com#markdown", "", "text/markdown")

	out, err := json.Marshal(block)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(out)

	if !strings.Contains(got, `"text":""`) {
		t.Errorf("marshaled output missing explicit empty text field; got %s", got)
	}
	if strings.Contains(got, `"blob"`) {
		t.Errorf("marshaled output should not include blob field for text variant; got %s", got)
	}
}

// TestEmbeddedBlobResourceContent_EmptyBlobRoundTripsAsExplicitField
// is the symmetric pin for the blob variant.
func TestEmbeddedBlobResourceContent_EmptyBlobRoundTripsAsExplicitField(t *testing.T) {
	block := EmbeddedBlobResourceContent("web-fetch://example.com#bin", "", "application/octet-stream")

	out, err := json.Marshal(block)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(out)

	if !strings.Contains(got, `"blob":""`) {
		t.Errorf("marshaled output missing explicit empty blob field; got %s", got)
	}
	if strings.Contains(got, `"text"`) {
		t.Errorf("marshaled output should not include text field for blob variant; got %s", got)
	}
}

// TestEmbeddedTextResourceContent_NonEmptyTextStillMarshals confirms
// the round-trip still works for the common case.
func TestEmbeddedTextResourceContent_NonEmptyTextStillMarshals(t *testing.T) {
	block := EmbeddedTextResourceContent("web-fetch://example.com#markdown", "hello", "text/markdown")

	out, err := json.Marshal(block)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(out)

	if !strings.Contains(got, `"text":"hello"`) {
		t.Errorf("expected text field with value hello; got %s", got)
	}
}
