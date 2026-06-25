package document

import (
	"testing"
)

func TestParseMarkdown_FrontmatterAndBody(t *testing.T) {
	raw := []byte("---\ntitle: Hello\ndraft: true\n---\n\n# Body\n")
	doc, err := ParseMarkdown("post.md", raw)
	if err != nil {
		t.Fatalf("ParseMarkdown: %v", err)
	}
	if doc.Format != Markdown {
		t.Errorf("Format = %q, want markdown", doc.Format)
	}
	if doc.Frontmatter["title"] != "Hello" {
		t.Errorf("title = %v, want Hello", doc.Frontmatter["title"])
	}
	if doc.Frontmatter["draft"] != true {
		t.Errorf("draft = %v, want true", doc.Frontmatter["draft"])
	}
	if len(doc.Body) == 0 || doc.BodyOffset == 0 {
		t.Errorf("body not extracted: offset=%d body=%q", doc.BodyOffset, doc.Body)
	}
}

func TestParseMarkdown_NoFrontmatter(t *testing.T) {
	raw := []byte("# Just a heading\n")
	doc, err := ParseMarkdown("x.md", raw)
	if err != nil {
		t.Fatalf("ParseMarkdown: %v", err)
	}
	if len(doc.Frontmatter) != 0 {
		t.Errorf("expected empty frontmatter, got %v", doc.Frontmatter)
	}
}
