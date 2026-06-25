package document

import "fmt"

// ParseFunc builds a Document from a file's bytes.
type ParseFunc func(path string, raw []byte) (*Document, error)

var parsers = map[Format]ParseFunc{
	Markdown: ParseMarkdown,
	Data:     ParseData,
}

// Parse dispatches to the parser registered for format.
func Parse(format Format, path string, raw []byte) (*Document, error) {
	p, ok := parsers[format]
	if !ok {
		return nil, fmt.Errorf("no parser for format %q", format)
	}
	return p(path, raw)
}
