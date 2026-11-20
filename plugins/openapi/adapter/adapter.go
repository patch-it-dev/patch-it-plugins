// Package adapter defines the per-API source adapter interface and holds
// each API's adapter, starting with Stripe. An adapter only knows where its
// API's spec lives and how to fetch it — fetching mechanics live in
// fetch/, and diffing/normalizing/storing are shared, not adapter code.
package adapter

import (
	"context"

	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
)

// Spec is one fetched spec: its raw bytes, the API version it declares, and
// the pinned URL it came from. It is not yet parsed or stored — that is
// diff.Parse's and the baseline's (plugin.Spec) job, respectively.
type Spec struct {
	Content   []byte
	Version   string
	SourceURL string
}

// Adapter is the shape every source adapter follows. Community and private
// adapters implement the same three methods.
type Adapter interface {
	API() string                                         // lowercase slug, e.g. "stripe"
	LatestTag(ctx context.Context) (string, error)       // newest release tag
	Fetch(ctx context.Context, tag string) (Spec, error) // spec for that exact tag
}

// Reshaper is an adapter whose Fetch changes the spec's structure, not only its
// encoding, before the differ reads it (GitLab's inlined request bodies). A
// pointer into what the differ read may then name no node in the published
// file, so SourcePointer turns it back into one that does. It is given the
// fetched content and returns the mapping for that spec.
type Reshaper interface {
	SourcePointer(content []byte) func(ptr string) string
}

// Lister is an adapter that can list every version of its spec, oldest first
// (a version's Ref is what Fetch takes): what lets a spec be diffed release by
// release, not only newest against last. One without it is read at LatestTag
// only.
type Lister interface {
	Versions(ctx context.Context) ([]plugin.Version, error)
}
