package plugin

import (
	"context"
	"time"
)

// A source that has versions (a spec repo's tags, a package's releases on
// PyPI or npm, a guide per major) is read in three parts, so whoever runs it
// can cover every version, not only the newest:
//
//	versions, _ := s.ListVersions(ctx, env)            // oldest first
//	a, _ := s.ReadAt(ctx, env, versions[i])
//	b, _ := s.ReadAt(ctx, env, versions[i+1])
//	result, _ := s.Extract(ctx, env, a, b)            // what changed from a to b
//
// Every consecutive pair is one Extract; a released version never changes, so
// a pair is done once.
//
// A living page (a deprecations page, a feed: the notices kind) is also read
// as it is now by Run, which compares it with the last baseline and cites a
// fresh Web Archive snapshot of it; its history is its Web Archive captures,
// listed as versions.

// Version is one version of a source.
type Version struct {
	// ID is the version as events name it: "2.0.0", "2026-11-19", "2025-03-31.basil".
	ID string
	// Released is when it came out; zero when the source doesn't say.
	Released time.Time
	// Ref is what to read it at: a tag, a commit SHA, a branch, a date.
	Ref string
	// URL is where a person sees this version (a release page, a commit).
	URL string
}

// Snapshot is a source as it was at one version: what ReadAt read.
type Snapshot struct {
	Version   Version
	Content   []byte
	SourceURL string // the exact document read, pinned to the version
}

// IsZero is true for the snapshot before the first version: nothing.
func (s Snapshot) IsZero() bool { return s.Version.ID == "" && s.Content == nil }

// VersionLister lists every version a source has, oldest first.
type VersionLister interface {
	ListVersions(ctx context.Context, env Env) ([]Version, error)
}

// Reader reads a source at one version.
type Reader interface {
	ReadAt(ctx context.Context, env Env, v Version) (Snapshot, error)
}

// Extractor turns two snapshots into the changes between them. from may be
// zero (no earlier version read). The events carry to's version and, when from
// isn't zero, previous_version set to from's. It stores nothing: Result has no
// Baselines (Run adds its own).
type Extractor interface {
	Extract(ctx context.Context, env Env, from, to Snapshot) (Result, error)
}

// Versioned is a plugin read version by version.
type Versioned interface {
	Plugin
	VersionLister
	Reader
	Extractor
}
