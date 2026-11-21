// Package feed is the notices kind's reader for a vendor's RSS or Atom
// changelog feed (kinds/notices reads it, snapshots and cites it): one event per breaking or deprecation entry (or per name such an
// entry lists), quoting the line that says so. Rung 3 of the source ladder.
//
// A feed is prose, so each feed comes with a Reader that knows what its
// vendor's entries look like and which subject they describe. The rules are
// the same for every reader (precision over recall):
//
//   - An entry the vendor doesn't mark as breaking or deprecation (a category,
//     a heading) is skipped.
//   - A marked entry becomes an event only when the reader finds a typed
//     subject in it (a runner label, a model id in code) and a dated
//     sentence. Otherwise it is dropped, logged with why, never guessed.
//   - `breaking` needs the vendor to say the thing stops on a date ("will be
//     retired on", "will be shut down on"). Anything softer ("we recommend
//     updating before") is downgraded to `deprecation`.
//
// A feed is a live document, so the events cite a Wayback Machine snapshot
// of it, read back and checked against the live feed (kit.Cite), like every
// notices source.
package feed

import (
	"fmt"
	"log"
	"slices"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/extract"
	"github.com/patch-it-dev/patch-it-plugins/core/kit"
)

// Feed is one vendor feed and how to read its entries.
type Feed struct {
	API    string // the source id the events carry: "github/actions"
	Vendor string // for sentences: "GitHub"
	URL    string // the feed itself
	Read   Reader
}

// Reader turns one entry into findings: what it can name for sure, and what
// it drops. It returns nothing for an entry that isn't breaking or a
// deprecation.
type Reader func(e extract.Entry) []Finding

// Finding is one change a reader found in an entry, or one it dropped.
type Finding struct {
	ChangeType  event.ChangeType // breaking or deprecation
	Version     string           // per the subject's type: the removal or shutdown date
	Subject     event.Subject
	Description string
	Hint        string
	Excerpt     string // the entry's own lines that say so
	Entry       string // the entry's link, for the log
	Dropped     string // set: the entry was marked but this is why no event
}

// Findings reads every entry of a feed.
func (f Feed) Findings(b []byte) ([]Finding, error) {
	found, err := f.Find(b)
	if err != nil {
		return nil, err
	}
	return found.(Findings).Found, nil
}

// What the notices kind (kinds/notices) asks of a feed.

// Findings are what one read of a feed found: the feed's title, every
// entry's title, and the findings, dropped ones included.
type Findings struct {
	Title   string
	Entries []string
	Found   []Finding
}

// Count is how many entries the feed has: a feed keeps its newest entries,
// so their number stays about the same from one read to the next even when
// none of them is a finding.
func (fs Findings) Count() int { return len(fs.Entries) }

// Has says whether entry is the feed's own title or an entry's title. A feed
// drops its oldest entries as new ones come, so its title is the anchor that
// stays: it says the URL still serves the feed it did.
func (fs Findings) Has(entry string) bool {
	return fs.Title == entry || slices.Contains(fs.Entries, entry)
}

// MinRatio is the shape guard's ratio for a feed: its window of newest
// entries can shrink when a vendor trims it, so only a drop below half is a
// change of shape.
func (Feed) MinRatio() float64 { return 0.5 }

// Slug is the source id the events carry: "github/actions".
func (f Feed) Slug() string { return f.API }

// PageURL is the feed itself.
func (f Feed) PageURL() string { return f.URL }

// Find reads every entry of the feed.
func (f Feed) Find(b []byte) (kit.Notices, error) {
	entries, err := extract.ParseFeed(b)
	if err != nil {
		return nil, err
	}
	out := Findings{Title: extract.FeedTitle(b)}
	for _, e := range entries {
		out.Entries = append(out.Entries, e.Title)
		out.Found = append(out.Found, f.Read(e)...)
	}
	return out, nil
}

// Events are an event per finding, citing citeURL; one of a version-pair type
// (an api_operation) carries previous as previous_version, a dated notice
// none. A dropped finding is logged with why.
func (f Feed) Events(found kit.Notices, citeURL, previous string, now time.Time) ([]event.Event, []error) {
	var events []event.Event
	var rejected []error
	for _, fd := range found.(Findings).Found {
		if fd.Dropped != "" {
			log.Printf("%s: dropped %s: %s", f.API, fd.Entry, fd.Dropped)
			continue
		}
		e := event.Event{
			API:           f.API,
			Version:       fd.Version,
			ChangeType:    fd.ChangeType,
			Breaking:      fd.ChangeType == event.ChangeTypeBreaking,
			Subject:       fd.Subject,
			Description:   fd.Description,
			MigrationHint: fd.Hint,
			SourceURL:     citeURL,
			SourceExcerpt: fd.Excerpt,
			Confidence:    1.0,
			DetectedAt:    now,
		}
		if event.PairType(e.Subject.Type) {
			e.PreviousVersion = previous
		}
		e.Fingerprint = event.Fingerprint(e.API, e.Version, e.ChangeType, e.Subject, "")
		if err := e.Validate(); err != nil {
			rejected = append(rejected, fmt.Errorf("%s %s: %w", f.API, fd.Entry, err))
			continue
		}
		events = append(events, e)
	}
	return events, rejected
}
