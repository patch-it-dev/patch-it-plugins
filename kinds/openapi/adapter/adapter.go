// Package adapter defines the interface a provider's OpenAPI source
// implements (its adapter, in providers/<provider>/; Stripe's is the
// reference) and the helpers adapters share: listing a repo's releases or a
// spec's commits, converting YAML, and moving a server's base path into the
// paths. An adapter only knows where its API's spec lives and how to fetch
// it: fetching mechanics live in core/fetch, and diffing, normalizing and
// storing are the openapi kind's, not adapter code.
package adapter

import (
	"context"

	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
)

// Spec is one fetched spec: its raw bytes, the API version it declares, and
// the pinned URL it came from. It is not yet parsed: that is diff.Parse's
// job, and keeping it is the runner's (the openapi plugin's ReadAt returns it
// as a plugin.Snapshot).
type Spec struct {
	Content   []byte
	Version   string
	SourceURL string
}

// Adapter is the shape every source adapter follows. Community and private
// adapters implement the same three methods.
type Adapter interface {
	API() string                                         // the source id events carry, e.g. "stripe/api"
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

// SpecVersioned is a Lister whose version ids are the specs' own
// info.version (Stripe's API versions): a spec read at a listed version must
// declare that version, or it's an error, never another version's content.
type SpecVersioned interface {
	Lister
	ListsSpecVersions()
}

// Rereleased is an adapter that publishes a version again after naming it:
// Stripe adds to an API version in later stripe/openapi releases, under the
// same version. Releases are the releases that name v, oldest first (what
// Fetch takes); the first whose spec declares v is where v starts. The
// changes into v are read from there, so nothing added inside v is filed
// with the changes between two versions; what v's later releases add is
// filed under v itself (previous_version = version), additions only.
type Rereleased interface {
	Releases(ctx context.Context, v plugin.Version) ([]string, error)
}
