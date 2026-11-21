// Package kit is what source plugins share beyond the core: the Wayback
// Machine client a living page is cited through (Wayback, Cite), and what a
// reader of dated notices returns (Notices). Fetching with ETag and
// Last-Modified caching is fetch's Cache, handed to every plugin as
// plugin.Env.Get; the test kit is kittest.
//
// kit is not a plugin: plugins import it, it imports no plugin.
package kit

import (
	"context"
	"fmt"
	"reflect"
)

// Cite snapshots a live page and checks the snapshot holds what the plugin
// read from the live page: read turns bytes into whatever the plugin makes
// events from, and the snapshot's must deep-equal want. The events are then
// made from the snapshot and cite snapURL, so every excerpt is on the page it
// cites. No snapshot, or one that says something else (the page changed in
// between), is an error: no provenance, no event.
func Cite[T any](ctx context.Context, a Archiver, pageURL string, read func([]byte) (T, error), want T) (snapURL string, snap []byte, got T, err error) {
	snapURL, snap, err = a.Snapshot(ctx, pageURL)
	if err != nil {
		return "", nil, got, fmt.Errorf("no events without a snapshot to cite: %w", err)
	}
	got, err = read(snap)
	if err != nil {
		return "", nil, got, fmt.Errorf("reading the snapshot %s: %w", snapURL, err)
	}
	if !reflect.DeepEqual(got, want) {
		return "", nil, got, fmt.Errorf("the snapshot %s doesn't hold what %s says now; run again", snapURL, pageURL)
	}
	return snapURL, snap, got, nil
}
