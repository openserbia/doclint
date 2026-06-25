package document

import "testing"

func TestParseData_AllFormats(t *testing.T) {
	cases := map[string][]byte{
		"x.yaml": []byte("name: a\ncount: 2\n"),
		"x.yml":  []byte("name: a\ncount: 2\n"),
		"x.toml": []byte("name = \"a\"\ncount = 2\n"),
		"x.json": []byte(`{"name":"a","count":2}`),
	}
	for path, raw := range cases {
		doc, err := ParseData(path, raw)
		if err != nil {
			t.Fatalf("ParseData(%s): %v", path, err)
		}
		if doc.Format != Data {
			t.Errorf("%s: format = %q, want data", path, doc.Format)
		}
		if doc.Frontmatter["name"] != "a" {
			t.Errorf("%s: name = %v, want a", path, doc.Frontmatter["name"])
		}
	}
}

func TestParseData_Unsupported(t *testing.T) {
	if _, err := ParseData("x.txt", []byte("hi")); err == nil {
		t.Error("expected error for unsupported data extension")
	}
}
