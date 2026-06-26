// Package cache memoizes per-file lint findings keyed by a content + config hash,
// so re-linting unchanged files is near-instant. It is used only by plain `lint`
// (fix/diff/fmt always run fresh); the key folds in everything that can change a
// file's findings, so a stale result is never returned.
package cache

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"os"
	"path/filepath"
	"sync"

	"github.com/openserbia/doclint/pkg/rule"
)

// schema is bumped to invalidate every entry when the on-disk format or the
// finding shape changes.
const schema = "v1"

const (
	dirMode  = 0o755
	fileMode = 0o644
	fileName = "findings.gob"
)

// Cache is a content-addressed store of per-file findings.
type Cache struct {
	dir     string
	mu      sync.Mutex
	entries map[string][]rule.Finding
	hits    int
	dirty   bool
}

// Open loads the cache from dir, returning an empty cache if it is absent or
// unreadable (a corrupt cache is never fatal — it just rebuilds).
func Open(dir string) *Cache {
	c := &Cache{dir: dir, entries: map[string][]rule.Finding{}}
	f, err := os.Open(c.path()) //nolint:gosec // path is the configured cache dir
	if err != nil {
		return c
	}
	defer func() { _ = f.Close() }()
	var loaded map[string][]rule.Finding
	if gob.NewDecoder(f).Decode(&loaded) == nil && loaded != nil {
		c.entries = loaded
	}
	return c
}

func (c *Cache) path() string { return filepath.Join(c.dir, fileName) }

// Key derives the cache key for a file from the invariants that affect its
// findings: the schema version, the doclint version, a hash of the resolved
// config, the file path, and the file content.
func Key(version, configHash, path string, content []byte) string {
	h := sha256.New()
	for _, p := range []string{schema, version, configHash, path} {
		_, _ = h.Write([]byte(p))
		_, _ = h.Write([]byte{0})
	}
	_, _ = h.Write(content)
	return hex.EncodeToString(h.Sum(nil))
}

// Get returns the cached findings for key, if present.
func (c *Cache) Get(key string) ([]rule.Finding, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	f, ok := c.entries[key]
	if ok {
		c.hits++
	}
	return f, ok
}

// Put stores findings for key.
func (c *Cache) Put(key string, findings []rule.Finding) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = findings
	c.dirty = true
}

// Save persists the cache to disk when it changed.
func (c *Cache) Save() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.dirty {
		return nil
	}
	if err := os.MkdirAll(c.dir, dirMode); err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(c.entries); err != nil {
		return err
	}
	return os.WriteFile(c.path(), buf.Bytes(), fileMode) //nolint:gosec // cache file in the configured dir
}

// Hits reports how many lookups were served from the cache this run.
func (c *Cache) Hits() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.hits
}

// Entries reports how many file results are stored.
func (c *Cache) Entries() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.entries)
}

// Clean removes the on-disk cache file.
func (c *Cache) Clean() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = map[string][]rule.Finding{}
	err := os.Remove(c.path())
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// DefaultDir returns the per-user cache directory for doclint.
func DefaultDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "doclint"), nil
}
