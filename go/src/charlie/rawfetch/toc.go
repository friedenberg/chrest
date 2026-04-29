package rawfetch

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"code.linenisgreat.com/chrest/go/src/charlie/markdown"
)

var (
	atxHeadingRE  = regexp.MustCompile(`^(#{1,6})\s+(.+?)(?:\s+#+)?\s*$`)
	slugReplaceRE = regexp.MustCompile(`[^a-z0-9]+`)
)

// ExtractMarkdownTOCFromText scans plain markdown text for ATX
// headings (# through ######), skipping lines inside fenced code
// blocks, and returns synthesized markdown.Heading entries with
// slugified ids. Suffix `-N` is appended on slug collisions.
func ExtractMarkdownTOCFromText(body []byte) []markdown.Heading {
	var out []markdown.Heading
	seen := map[string]int{}
	inFence := false
	scanner := bufio.NewScanner(bytes.NewReader(body))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimLeft(line, " \t")
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		m := atxHeadingRE.FindStringSubmatch(trimmed)
		if m == nil {
			continue
		}
		level := len(m[1])
		text := strings.TrimSpace(m[2])
		base := slugify(text)
		id := base
		if n := seen[base]; n > 0 {
			id = fmt.Sprintf("%s-%d", base, n+1)
		}
		seen[base]++
		out = append(out, markdown.Heading{ID: id, Text: text, Level: level})
	}
	return out
}

func slugify(text string) string {
	s := strings.ToLower(text)
	s = slugReplaceRE.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}
