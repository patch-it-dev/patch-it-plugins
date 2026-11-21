package kittest

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sync"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
)

// Sink is where a kit run stores the events a plugin found. Mem is the one
// the kit uses; patch-it's hosted ingestion service stores the same way.
type Sink interface {
	// SaveEvents stores events, skipping any whose fingerprint is already
	// stored, and returns how many were new. An invalid event stores nothing.
	SaveEvents(events []event.Event) (int, error)
	// All is every stored event in feed order, each with its id.
	All() ([]event.Event, error)
}

// Mem is a Sink in memory that stores like the hosted service's database does, so a
// golden file is the same whichever stored the events: events are deduped by
// fingerprint and get ids 1, 2, 3... in the order they were saved, each is
// stored as its JSON (what the feed serves). Ingest's conformance test stores
// every plugin's golden through the real store to keep the two equal.
type Mem struct {
	mu     sync.Mutex
	events []event.Event
	seen   map[string]bool
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

// Extract extracts from -> to with p and stores the events in sink. rejected
// reports whether any change failed to become a valid event. Unlike
// Extracted it doesn't fail the test, for tests that expect an error.
func Extract(ctx context.Context, env plugin.Env, sink Sink, p plugin.Plugin, from, to plugin.Snapshot) (rejected bool, err error) {
	r, err := p.Extract(ctx, env, from, to)
	if err != nil {
		return false, fmt.Errorf("%s: %w", p.Name(), err)
	}
	if _, err := sink.SaveEvents(r.Events); err != nil {
		return false, fmt.Errorf("%s: saving events: %w", p.Name(), err)
	}
	return len(r.Rejected) > 0, nil
}

// ReadPage is one read of a living page against last, the snapshot kept from
// the read before (zero on a first read), as the hosted ingestion service
// does it: Read; when the page is the same as last (by CompareHash), nothing
// more; otherwise Extract from last to the new snapshot and store the events.
// It returns the snapshot to keep for the next read: cur, or last unchanged.
// A failed read or extract keeps last.
func ReadPage(ctx context.Context, env plugin.Env, sink Sink, p plugin.Living, last plugin.Snapshot) (kept plugin.Snapshot, rejected bool, err error) {
	cur, err := p.Read(ctx, env, last)
	if err != nil {
		return last, false, fmt.Errorf("%s: %w", p.Name(), err)
	}
	if !last.IsZero() && cur.CompareHash == last.CompareHash {
		return last, false, nil
	}
	if rejected, err = Extract(ctx, env, sink, p, last, cur); err != nil {
		return last, false, err
	}
	return cur, rejected, nil
}

// Pair reads two versions of a versioned plugin (by id, read at the same
// ref), extracts the change between them and stores its events in sink: a
// version pair as patch-it's ingestion service does it. Unlike Versions it
// doesn't fail the test.
func Pair(ctx context.Context, env plugin.Env, sink Sink, p plugin.Versioned, from, to string) (rejected bool, err error) {
	a, err := p.ReadAt(ctx, env, plugin.Version{ID: from, Ref: from})
	if err != nil {
		return false, fmt.Errorf("%s: %w", p.Name(), err)
	}
	b, err := p.ReadAt(ctx, env, plugin.Version{ID: to, Ref: to})
	if err != nil {
		return false, fmt.Errorf("%s: %w", p.Name(), err)
	}
	return Extract(ctx, env, sink, p, a, b)
}
