package kittest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
)

// Sink is where a kit run stores what a plugin found, and where the plugin
// reads its last baseline back from (it is the run's Env.Baselines). Mem is
// the one the kit uses; patch-it's hosted ingestion service stores the same way.
type Sink interface {
	plugin.BaselineReader
	// SaveEvents stores events, skipping any whose fingerprint is already
	// stored, and returns how many were new. An invalid event stores nothing.
	SaveEvents(events []event.Event) (int, error)
	// SaveSpec stores a baseline, once per (api, sha256).
	SaveSpec(spec plugin.Spec) error
	// All is every stored event in feed order, each with its id.
	All() ([]event.Event, error)
}

// Mem is a Sink in memory that stores like the hosted service's database does, so a
// golden file is the same whichever stored the events: events are deduped by
// fingerprint and get ids 1, 2, 3... in the order they were saved, each is
// stored as its JSON (what the feed serves), and a baseline is the latest
// one, by fetch time, then by save order. Ingest's conformance test stores
// every plugin's golden through the real store to keep the two equal.
type Mem struct {
	mu     sync.Mutex
	events []event.Event
	seen   map[string]bool
	specs  []plugin.Spec // in save order
}

var _ Sink = (*Mem)(nil)

// NewMem is an empty store.
func NewMem() *Mem { return &Mem{seen: map[string]bool{}} }

func (m *Mem) SaveEvents(events []event.Event) (int, error) {
	// Like the store's transaction: one invalid event and none are stored.
	stored := make([]event.Event, 0, len(events))
	for _, e := range events {
		if err := e.Validate(); err != nil {
			return 0, fmt.Errorf("invalid event %s: %w", e.Fingerprint, err)
		}
		data, err := json.Marshal(e)
		if err != nil {
			return 0, fmt.Errorf("marshaling event %s: %w", e.Fingerprint, err)
		}
		var back event.Event
		if err := json.Unmarshal(data, &back); err != nil {
			return 0, fmt.Errorf("unmarshaling event %s: %w", e.Fingerprint, err)
		}
		stored = append(stored, back)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.seen == nil {
		m.seen = map[string]bool{}
	}
	saved := 0
	for _, e := range stored {
		if m.seen[e.Fingerprint] {
			continue
		}
		m.seen[e.Fingerprint] = true
		e.ID = int64(len(m.events) + 1)
		m.events = append(m.events, e)
		saved++
	}
	return saved, nil
}

func (m *Mem) All() ([]event.Event, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Clone(m.events), nil
}

// SaveSpec stores spec once per (api, sha256): the same content saved again
// is not a new baseline.
func (m *Mem) SaveSpec(spec plugin.Spec) error {
	spec.FetchedAt = spec.FetchedAt.UTC().Truncate(time.Microsecond) // what Postgres keeps: UTC, to the microsecond
	spec.Content = bytes.Clone(spec.Content)
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.specs {
		if s.API == spec.API && s.SHA256 == spec.SHA256 {
			return nil
		}
	}
	m.specs = append(m.specs, spec)
	return nil
}

// LastSpec is api's latest baseline, or nil on a first run.
func (m *Mem) LastSpec(api string) (*plugin.Spec, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var last *plugin.Spec
	for i := range m.specs {
		s := &m.specs[i]
		if s.API != api {
			continue
		}
		if last == nil || !s.FetchedAt.Before(last.FetchedAt) { // a later save wins a tie
			last = s
		}
	}
	if last == nil {
		return nil, nil
	}
	spec := *last
	spec.Content = bytes.Clone(spec.Content)
	return &spec, nil
}

// Run runs p once and stores what it found in sink, as the hosted service's runner does:
// the events first, then the baselines, so a run that fails in between leaves
// the baseline where it was. rejected reports whether any change failed to
// become a valid event. Unlike Ingest it doesn't fail the test, for tests that
// expect an error.
func Run(ctx context.Context, env plugin.Env, sink Sink, p plugin.Plugin) (rejected bool, err error) {
	r, err := p.Run(ctx, env)
	if err != nil {
		return false, fmt.Errorf("%s: %w", p.Name(), err)
	}
	return save(sink, p.Name(), r)
}

// Pair reads two versions of a versioned plugin (by id, read at the same
// ref), extracts the change between them and stores its events in sink: a
// version pair as patch-it's ingestion service does it.
func Pair(ctx context.Context, env plugin.Env, sink Sink, p plugin.Plugin, from, to string) (rejected bool, err error) {
	vp, ok := p.(plugin.Versioned)
	if !ok {
		return false, fmt.Errorf("%s isn't read version by version", p.Name())
	}
	a, err := vp.ReadAt(ctx, env, plugin.Version{ID: from, Ref: from})
	if err != nil {
		return false, fmt.Errorf("%s: %w", p.Name(), err)
	}
	b, err := vp.ReadAt(ctx, env, plugin.Version{ID: to, Ref: to})
	if err != nil {
		return false, fmt.Errorf("%s: %w", p.Name(), err)
	}
	r, err := vp.Extract(ctx, env, a, b)
	if err != nil {
		return false, fmt.Errorf("%s: %w", p.Name(), err)
	}
	return save(sink, p.Name(), r)
}

func save(sink Sink, name string, r plugin.Result) (bool, error) {
	if _, err := sink.SaveEvents(r.Events); err != nil {
		return false, fmt.Errorf("%s: saving events: %w", name, err)
	}
	for _, b := range r.Baselines {
		if err := sink.SaveSpec(b); err != nil {
			return false, fmt.Errorf("%s: saving %s: %w", name, b.Tag, err)
		}
	}
	return len(r.Rejected) > 0, nil
}
