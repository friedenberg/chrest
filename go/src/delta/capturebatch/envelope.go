package capturebatch

import "time"

// EnvelopeSchema is the `schema` constant in the envelope artifact.
//
// Stage 1 of chrest#22 emits a `v1-preview` envelope that is missing
// the RFC-required `http.status` / `http.headers` fields. This label
// is coordinated with nebulous on the session bus: v1-insisting
// consumers reject preview per RFC forward-compat rules (spec is
// self-enforcing), preview-tolerant consumers can opt in, and the
// RFC stays unchanged. Stage 1b bumps this constant to
// `web-capture-archive.envelope/v1` when the bidi event-subscription
// refactor (chrest#24) lands and `http.*` is populated.
const EnvelopeSchema = "web-capture-archive.envelope/v1-preview"

// EnvelopeMediaType is the Content-Type of the canonicalized envelope
// bytes. Stays stable across v1-preview → v1 — the discriminator is
// the `schema` field inside the blob.
const EnvelopeMediaType = "application/vnd.web-capture-archive.envelope+json"

// BuildEnvelope assembles the envelope artifact for a resolved capture
// and returns the JCS-canonicalized bytes.
//
// Per RFC 0001 §Envelope Artifact:
//   - `schema`, `url`, `captured_at` are required.
//   - `http.status`, `http.headers` are required by the RFC but NOT
//     populated here in stage 1 of chrest#22 — see EnvelopeSchema
//     doc comment for the v1-preview rationale.
//   - `stripped.<format>` is optional; the format normalizer returns
//     what it removed, or nil if nothing.
func BuildEnvelope(url string, capturedAt time.Time, stripped map[string]any) ([]byte, error) {
	doc := map[string]any{
		"schema":      EnvelopeSchema,
		"url":         url,
		"captured_at": capturedAt.UTC().Format("2006-01-02T15:04:05.000Z"),
	}
	if len(stripped) > 0 {
		doc["stripped"] = stripped
	}
	return Canonicalize(doc)
}
