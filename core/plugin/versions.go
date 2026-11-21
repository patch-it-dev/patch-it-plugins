package plugin

import (
	"context"
	"time"
)

// A source is read in one of two shapes. Both end in Extract, so whoever runs
// them treats every source as pairs of snapshots.
//
// A source that has versions (a spec repo's tags, a package's releases on
// PyPI or npm, a guide per major, a file's commits) is Versioned, so whoever
// runs it can cover every version, not only the newest:
//
//	versions, _ := s.ListVersions(ctx, env)            // oldest first
//	a, _ := s.ReadAt(ctx, env, versions[i])
//	b, _ := s.ReadAt(ctx, env, versions[i+1])
//	result, _ := s.Extract(ctx, env, a, b)            // what changed from a to b
//
// Every consecutive pair is one Extract; a released version never changes, so
// a pair is done once.
//
// A page with no versions of its own (a deprecations page, a feed: the
// notices kind) is Living: it is read as it is now, against the snapshot kept
// from the read before, and each read that differs is a new version, dated:
//
//	cur, _ := s.Read(ctx, env, last)                  // last: the kept snapshot, or zero
//	if cur.CompareHash != last.CompareHash {
//		result, _ := s.Extract(ctx, env, last, cur)   // what the page says now that it didn't
//		// store the events, keep cur as the snapshot
//	}

// Version is one version of a source.
type Version struct {
	// ID is the version as events name it: "2.0.0", "2026-11-19",
	// "2025-03-31.basil"; a living page's read is dated (RFC 3339, UTC).
	ID string
	// Released is when it came out (a living page: when it was read); zero
	// when the source doesn't say.
	Released time.Time
	// Ref is what to read it at: a tag, a commit SHA, a branch, a date.
	Ref string
	// URL is where a person sees this version (a release page, a commit, a
	// Wayback Machine snapshot).
	URL string
	// MapsTo is the version of another source this one pins, when it pins
	// one: an SDK release and the API version it calls ("stripe/api",
	// "2025-03-31"). Zero when it pins none or the source doesn't say. The
	// plugin reads it from the release itself (its notes, the version
	// constant in its code), never from a table kept by hand; it is how a
	// feed query on the SDK's versions also gets the API's changes.
	MapsTo Pin
}

// Pin is one version of another source: what a release pins.
type Pin struct {
	API     string // the source's provider/source id: "stripe/api"
	Version string // as that source names it: "2025-03-31"
}

// IsZero is true when nothing is pinned.
func (p Pin) IsZero() bool { return p.API == "" && p.Version == "" }

// Snapshot is a source as it was at one version: what ReadAt or Read read.
type Snapshot struct {
	Version   Version
	Content   []byte
	SourceURL string // the exact document read, pinned to the version
	// CompareHash is how a living page's next Read tells it unchanged: the
	// hash of the page as fetched, which may differ from Content (a page
	// cited through a Wayback Machine snapshot keeps the snapshot's bytes).
	// Empty for a versioned source: a released version never changes.
	CompareHash string
}

// IsZero is true for the snapshot before the first version: nothing.
func (s Snapshot) IsZero() bool { return s.Version.ID == "" && s.Content == nil }

// Versioned is a plugin read version by version.
type Versioned interface {
	Plugin
	// ListVersions is every version the source has, oldest first.
	ListVersions(ctx context.Context, env Env) ([]Version, error)
	// ReadAt reads the source at one version.
	ReadAt(ctx context.Context, env Env, v Version) (Snapshot, error)
}

// Living is a plugin read as it is now: a page that only shows its latest
// state.
type Living interface {
	Plugin
	// Read reads the page now. last is the snapshot kept from the read
	// before (zero on a first read); when the page is the same as then (by
	// CompareHash), Read returns last as is, having cost one GET. Otherwise
	// it returns a new snapshot, dated now, citing a copy of the page that
	// stays as read (a Wayback Machine snapshot), with its CompareHash set.
	Read(ctx context.Context, env Env, last Snapshot) (Snapshot, error)
}
