package cache

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/openserbia/doclint/pkg/rule"
)

func TestCacheRoundTrip(t *testing.T) {
	dir := t.TempDir()
	c := Open(dir)
	key := Key("v1", "cfg", "a.md", []byte("content"))
	if _, ok := c.Get(key); ok {
		t.Fatal("empty cache should miss")
	}
	want := []rule.Finding{{Rule: "r", Path: "a.md", Line: 1, Col: 1, Message: "m", Severity: rule.Warning, Safety: rule.Safe}}
	c.Put(key, want)
	if err := c.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Reopen: the entry persists across processes.
	c2 := Open(dir)
	got, ok := c2.Get(key)
	if !ok || len(got) != 1 || got[0].Rule != "r" || got[0].Safety != rule.Safe {
		t.Errorf("reopened cache miss/mismatch: ok=%v got=%+v", ok, got)
	}
	if c2.Entries() != 1 {
		t.Errorf("entries = %d, want 1", c2.Entries())
	}

	// A different content / version / config hash yields a different key → miss.
	if _, ok := c2.Get(Key("v1", "cfg", "a.md", []byte("changed"))); ok {
		t.Error("changed content should miss")
	}
	if _, ok := c2.Get(Key("v2", "cfg", "a.md", []byte("content"))); ok {
		t.Error("changed version should miss")
	}

	// Clean removes the on-disk file.
	if err := c2.Clean(); err != nil {
		t.Fatalf("clean: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, fileName)); !os.IsNotExist(err) {
		t.Error("clean should remove the cache file")
	}
}
