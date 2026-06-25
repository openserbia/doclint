package document

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

// ParseData loads a YAML/TOML/JSON data file into Document.Frontmatter (the
// whole file becomes the parsed map). Data files have no body or AST.
func ParseData(path string, raw []byte) (*Document, error) {
	doc := &Document{
		Path:        path,
		Format:      Data,
		Raw:         raw,
		Lines:       SplitLines(raw),
		Frontmatter: map[string]any{},
	}
	m := map[string]any{}
	switch ext := filepath.Ext(path); ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(raw, &m); err != nil {
			return nil, fmt.Errorf("parse yaml %s: %w", path, err)
		}
	case ".toml":
		if err := toml.Unmarshal(raw, &m); err != nil {
			return nil, fmt.Errorf("parse toml %s: %w", path, err)
		}
	case ".json":
		if err := json.Unmarshal(raw, &m); err != nil {
			return nil, fmt.Errorf("parse json %s: %w", path, err)
		}
	default:
		return nil, fmt.Errorf("unsupported data extension %q", ext)
	}
	doc.Frontmatter = m
	return doc, nil
}
