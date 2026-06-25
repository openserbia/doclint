package document

import (
	"bytes"
	"fmt"

	"github.com/adrg/frontmatter"
)

// ParseMarkdown builds a markdown Document: it extracts YAML/TOML/JSON
// frontmatter into a map and records where the body begins.
func ParseMarkdown(path string, raw []byte) (*Document, error) {
	doc := &Document{
		Path:        path,
		Format:      Markdown,
		Raw:         raw,
		Lines:       SplitLines(raw),
		Frontmatter: map[string]any{},
	}

	var matter map[string]any
	rest, err := frontmatter.Parse(bytes.NewReader(raw), &matter)
	if err != nil {
		return nil, fmt.Errorf("parse frontmatter in %s: %w", path, err)
	}
	if matter != nil {
		doc.Frontmatter = matter
	}
	doc.Body = rest
	// BodyOffset = where `rest` starts inside raw. frontmatter.Parse returns the
	// trailing body verbatim, so locate it from the end to stay delimiter-agnostic.
	doc.BodyOffset = len(raw) - len(rest)
	return doc, nil
}
