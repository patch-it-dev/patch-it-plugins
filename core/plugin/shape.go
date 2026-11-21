package plugin

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// ErrShapeChanged is what a plugin returns, wrapped, when a read doesn't look
// like the source it reads any more: a page that lost most of what the read
// before found, a page without an entry its source declares it always has
// (an anchor), or a versioned source whose newest releases all have nothing
// to read after ones that did. The source most likely changed its layout, or
// something else is served in its place, so the read is refused: never
// stored, never read as removals. errors.Is finds it through any wrapping,
// and whoever runs the plugin reports it (patch-it's hosted ingestion
// service opens an issue in this repo).
var ErrShapeChanged = errors.New("the source's shape changed")

// Shape is what a source declares about its own reads, for the shape guard
// every kind runs (Guarded). The zero value is the kind's defaults, which fit
// most sources: a source sets a field only with a reason, in a comment next
// to it in its provider.go.
type Shape struct {
	// MinRatio: a read that finds fewer than MinRatio times what the read
	// before found fails with ErrShapeChanged. 0 is the kind's default (its
	// DefaultMinRatio). Compared only when the read before found at least
	// MinItemsCompared; a read that finds nothing after one that found
	// anything always fails.
	MinRatio float64
	// Anchors are one or two entries a living source's page always lists,
	// so a read without one fails with ErrShapeChanged. Each is declared by
	// hand after checking the live page, with the date checked: never taken
	// from a saved copy. Living sources only.
	Anchors []Anchor
	// NothingToReadAfter: a versioned source whose newest this many releases
	// in a row have nothing to read (ErrNothingToRead), after one that had
	// something, is a shape signal (NothingToRead). 0 is
	// DefaultNothingToReadAfter.
	NothingToReadAfter int
}

// Anchor is one entry a living page always lists, as its kind's reader names
// an entry (a model id on a deprecations page, an API version on an
// end-of-life table, the feed's own title or an entry's title on a feed).
type Anchor struct {
	Entry string
	// Checked is the day the entry was seen on the live page: YYYY-MM-DD.
	Checked string
}

const (
	// MinItemsCompared is the smallest read before that MinRatio is
	// compared with: below it, a few legitimate removals are a large share.
	MinItemsCompared = 10
	// DefaultNothingToReadAfter is Shape.NothingToReadAfter's default.
	DefaultNothingToReadAfter = 3
	// MaxAnchors is how many anchors a source may declare: one or two
	// entries that won't leave the page, not a copy of it.
	MaxAnchors = 2
)

// Guarded is a plugin that runs the shape guard with its source's Shape.
// Every kind is one: RegisterProvider refuses a plugin that isn't, and hands
// each registered plugin its source's Shape through WithShape. A plugin
// built by hand and never given a Shape (a golden test over trimmed copies,
// far smaller than the real releases) runs no guard: the guard's methods
// are nil-safe, and a kind keeps a *Shape that is nil until WithShape.
type Guarded interface {
	Plugin
	// WithShape is the plugin with s as its source's Shape.
	WithShape(s Shape) Plugin
}

// CheckCount is the shape guard on how much a read found: now items, against
// before found by the read before (before < 0: there was none). kindDefault
// is the kind's ratio, used when the source sets none. what names the items
// for the error ("operations", "retirements").
func (s *Shape) CheckCount(what string, kindDefault float64, before, now int) error {
	if s == nil || before <= 0 {
		return nil
	}
	if now == 0 {
		return fmt.Errorf("%w: the read found no %s, the read before found %d", ErrShapeChanged, what, before)
	}
	ratio := s.MinRatio
	if ratio == 0 {
		ratio = kindDefault
	}
	if before >= MinItemsCompared && float64(now) < ratio*float64(before) {
		return fmt.Errorf("%w: the read found %d %s, the read before found %d (less than %.0f%%)", ErrShapeChanged, now, what, before, ratio*100)
	}
	return nil
}

// CheckAnchors is the shape guard on what a read found: every anchor the
// source declares must be among it (has).
func (s *Shape) CheckAnchors(has func(entry string) bool) error {
	if s == nil {
		return nil
	}
	var missing []string
	for _, a := range s.Anchors {
		if !has(a.Entry) {
			missing = append(missing, fmt.Sprintf("%q (seen on %s)", a.Entry, a.Checked))
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("%w: the page no longer lists %s", ErrShapeChanged, strings.Join(missing, ", "))
	}
	return nil
}

// NothingToRead is the shape guard on a versioned source's releases, for
// whoever runs it: read says, for each release extracted, oldest first,
// whether it had something to read (false: ErrNothingToRead). When the newest
// NothingToReadAfter (or more) in a row had nothing, after one that had
// something, the source probably moved its changelog or guide or changed its
// layout: the error wraps ErrShapeChanged, not ErrNothingToRead. A source
// that has never had anything to read isn't flagged: nothing says it ever
// could be.
func (s Shape) NothingToRead(read []bool) error {
	n := s.NothingToReadAfter
	if n == 0 {
		n = DefaultNothingToReadAfter
	}
	run := 0
	for i := len(read) - 1; i >= 0; i-- {
		if read[i] {
			if run >= n {
				return fmt.Errorf("%w: the newest %d releases in a row have nothing to read, after one that had", ErrShapeChanged, run)
			}
			return nil
		}
		run++
	}
	return nil
}

var day = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// check is what a Shape must be, for a source of the given shape ("living"
// or "versioned").
func (s Shape) check(shape string) error {
	switch {
	case s.MinRatio < 0 || s.MinRatio > 1:
		return fmt.Errorf("Shape.MinRatio %v isn't between 0 and 1", s.MinRatio)
	case s.NothingToReadAfter < 0:
		return fmt.Errorf("Shape.NothingToReadAfter %d is negative", s.NothingToReadAfter)
	case len(s.Anchors) > MaxAnchors:
		return fmt.Errorf("%d anchors, at most %d", len(s.Anchors), MaxAnchors)
	case len(s.Anchors) > 0 && shape != "living":
		return errors.New("anchors are for living pages; a versioned source's releases are guarded by count and NothingToRead")
	}
	for _, a := range s.Anchors {
		if strings.TrimSpace(a.Entry) == "" || !day.MatchString(a.Checked) {
			return fmt.Errorf("anchor %q: an entry and the day it was checked (YYYY-MM-DD) are both required", a.Entry)
		}
	}
	return nil
}
