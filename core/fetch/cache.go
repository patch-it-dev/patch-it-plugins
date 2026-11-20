package fetch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

// Cache is Get with HTTP validators: the first GET of a URL keeps the body
// with its ETag and Last-Modified, and every later GET sends them back
// (If-None-Match, If-Modified-Since). A 304 Not Modified returns the kept
// body, so a source that hasn't changed costs the vendor a few bytes, and the
// caller sees exactly what a plain Get would have returned.
//
// A response with neither validator isn't kept: there would be nothing to ask
// with. Entries live in Dir, one JSON file per URL, so they survive between
// runs; an empty Dir keeps them in memory for the process only. The zero
// Cache is ready to use.
type Cache struct {
	Dir string

	mu  sync.Mutex
	mem map[string]cacheEntry
}

type cacheEntry struct {
	URL          string `json:"url"`
	ETag         string `json:"etag,omitempty"`
	LastModified string `json:"last_modified,omitempty"`
	Body         []byte `json:"body"`
}

// Get is fetch.Get through the cache. headers are sent as with Get; the
// validators are added to them.
func (c *Cache) Get(ctx context.Context, url string, headers map[string]string) ([]byte, error) {
	cached, ok := c.load(url)
	h := map[string]string{}
	for k, v := range headers {
		h[k] = v
	}
	if ok {
		if cached.ETag != "" {
			h["If-None-Match"] = cached.ETag
		}
		if cached.LastModified != "" {
			h["If-Modified-Since"] = cached.LastModified
		}
	}
	r, err := get(ctx, url, h)
	if err != nil {
		return nil, err
	}
	if r.status == http.StatusNotModified {
		if !ok {
			// Only a request with validators may get a 304.
			return nil, fmt.Errorf("%s answered 304 to a request without validators", url)
		}
		return cached.Body, nil
	}
	e := cacheEntry{URL: url, ETag: r.header.Get("ETag"), LastModified: r.header.Get("Last-Modified"), Body: r.body}
	if e.ETag != "" || e.LastModified != "" {
		if err := c.save(e); err != nil {
			return nil, fmt.Errorf("caching %s: %w", url, err)
		}
	}
	return r.body, nil
}

func (c *Cache) file(url string) string {
	sum := sha256.Sum256([]byte(url))
	return filepath.Join(c.Dir, hex.EncodeToString(sum[:])+".json")
}

func (c *Cache) load(url string) (cacheEntry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Dir == "" {
		e, ok := c.mem[url]
		return e, ok
	}
	b, err := os.ReadFile(c.file(url))
	if err != nil {
		return cacheEntry{}, false
	}
	var e cacheEntry
	// A file that doesn't read back (a crash mid-write, a hash collision) is
	// a miss: the GET goes out unconditional and rewrites it.
	if json.Unmarshal(b, &e) != nil || e.URL != url {
		return cacheEntry{}, false
	}
	return e, true
}

func (c *Cache) save(e cacheEntry) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Dir == "" {
		if c.mem == nil {
			c.mem = map[string]cacheEntry{}
		}
		c.mem[e.URL] = e
		return nil
	}
	if err := os.MkdirAll(c.Dir, 0o755); err != nil && !errors.Is(err, fs.ErrExist) {
		return err
	}
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(c.Dir, ".tmp-*")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), c.file(e.URL))
}
