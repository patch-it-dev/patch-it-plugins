// Package plugin is what a change source is: every source is a plugin behind
// one interface, registered by name. A plugin reads its source and returns a
// Result; it stores nothing itself. Whoever runs it (patch-it's hosted
// ingestion service, or the test kit) stores the events, then the baselines.
// Adding a source is a RegisterProvider call in its kind's package; nothing
// here changes.
package plugin

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"sync"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
)

// Plugin is one place changes come from.
type Plugin interface {
	// Name is the source's id, "<provider>/<source>", which its events
	// carry in `api`: "stripe/api", "openai/models".
	Name() string
	// Run reads the source as it is now and returns what changed since the
	// baseline it saved last (env.Baselines.LastSpec). An error stores nothing.
	Run(ctx context.Context, env Env) (Result, error)
}

// Env is everything a plugin may reach outside itself, so a test can hand it
// baselines, the network and a clock of its own. A plugin that fetches through
// its own client (the OpenAPI plugin's adapter) ignores Get; its tests fake
// that client instead.
type Env struct {
	Baselines BaselineReader
	// Get fetches one URL. In both the hosted ingestion service and the test
	// kit, this is fetch.Cache.Get (kit/kittest's Pages.Get wraps saved files
	// the same way): a non-2xx, non-304 response fails with a *fetch.StatusError
	// naming the status, returned as is, so errors.As finds it. A plugin that
	// branches on the status (eol, when the provider's own site blocks a plain
	// GET) can rely on that.
	Get func(ctx context.Context, url string) ([]byte, error)
	Now func() time.Time
}

// BaselineReader is the one thing a plugin reads back: the baseline its last
// run saved. Ingest's store implements it; so does the test kit.
type BaselineReader interface {
	// LastSpec is api's latest baseline, or nil (and no error) on a first run.
	LastSpec(api string) (*Spec, error)
}

// Spec is one baseline: a fetched copy of what a source said, kept only so a
// later run can diff against it, or tell in one hash comparison that nothing
// changed.
type Spec struct {
	API       string
	Version   string
	Tag       string
	SHA256    string
	SourceURL string
	FetchedAt time.Time
	Content   []byte
}

// Result is what one run found.
type Result struct {
	Label     string        // what was read, for the summary: a tag, a snapshot, a URL
	Events    []event.Event // valid v2 events, to store
	Rejected  []error       // changes that failed to become valid events
	Baselines []Spec        // what the run read, saved after the events
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

// Baseline is what a plugin saves as the last thing it read, hashed so the
// next run can tell an unchanged source in one comparison.
func Baseline(api, version, tag, sourceURL string, content []byte, at time.Time) Spec {
	return Spec{
		API:       api,
		Version:   version,
		Tag:       tag,
		SHA256:    SHA256(content),
		SourceURL: sourceURL,
		FetchedAt: at,
		Content:   content,
	}
}

// SHA256 is the hex hash a baseline is compared by.
func SHA256(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
