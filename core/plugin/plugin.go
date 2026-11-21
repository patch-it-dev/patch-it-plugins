// Package plugin is what a change source is: every source is a plugin behind
// one of two shapes, registered by its id. A plugin reads its source and
// returns what changed between two reads of it; it stores nothing itself.
// Whoever runs it (patch-it's hosted ingestion service, or the test kit)
// stores the events and keeps the latest snapshot. Adding a source is a
// RegisterProvider call in its kind's package; nothing here changes.
package plugin

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"slices"
	"sync"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
)

// Plugin is one place changes come from. Every plugin is exactly one of
// Versioned (a source with versions: tags, releases, commits) or Living (a
// page with no versions of its own, read as it is now); RegisterProvider
// refuses one that is both or neither. Either way, the changes are what
// Extract finds between two snapshots.
type Plugin interface {
	// Name is the source's id, "<provider>/<source>", which its events
	// carry in `api`: "stripe/api", "openai/models".
	Name() string
	// Extract turns two snapshots into the changes between them: the
	// events carry to's version. from is zero when nothing was read before
	// it (a living page's first read). It stores nothing.
	Extract(ctx context.Context, env Env, from, to Snapshot) (Result, error)
}

// ErrNothingToRead is what a reader returns, wrapped, when a release or a
// page has no part it can read: a changelog with no section for the
// release, a migration guide with no table of removals, a page with no
// table of notices. For a release that's normal (not every release has
// one), so the runner keeps the pair as skipped rather than retrying it;
// errors.Is finds it through any wrapping. Any other error (a fetch, a
// parse) is a failure.
var ErrNothingToRead = errors.New("nothing to read")

// Env is everything a plugin may reach outside itself, so a test can hand it
// the network and a clock of its own. A plugin that fetches through its own
// client (the OpenAPI plugin's adapter) ignores Get; its tests fake that
// client instead.
type Env struct {
	// Get fetches one URL. In both the hosted ingestion service and the test
	// kit, this is fetch.Cache.Get (kit/kittest's Pages.Get wraps saved files
	// the same way): a non-2xx, non-304 response fails with a *fetch.StatusError
	// naming the status, returned as is, so errors.As finds it. A plugin that
	// branches on the status (eol, when the provider's own site blocks a plain
	// GET) can rely on that.
	Get func(ctx context.Context, url string) ([]byte, error)
	Now func() time.Time
}

// Result is what one Extract found.
type Result struct {
	Events   []event.Event // valid events, to store
	Rejected []error       // changes that failed to become valid events
}

var (
	mu       sync.Mutex
	registry = map[string]Plugin{}
)

// Lookup finds a registered plugin by its source's id ("algolia/search-api").
func Lookup(id string) (Plugin, bool) {
	mu.Lock()
	defer mu.Unlock()
	p, ok := registry[id]
	return p, ok
}

// Names are the registered sources' ids, sorted. Sources lists them with
// their metadata.
func Names() []string {
	mu.Lock()
	defer mu.Unlock()
	names := make([]string, 0, len(registry))
	for n := range registry {
		names = append(names, n)
	}
	slices.Sort(names)
	return names
}

// SHA256 is the hex hash a living page is compared by (Snapshot.CompareHash).
func SHA256(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
