package retire

import (
	"time"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/kit"
)

// What the notices kind (plugins/notices) asks of a deprecations page.

// Retirements are what one read of a page found.
type Retirements []Retirement

// Count is how many retirements the page lists.
func (rs Retirements) Count() int { return len(rs) }

// Slug is the source id the events carry: "openai/models".
func (p Page) Slug() string { return p.API }

// PageURL is the page, as markdown.
func (p Page) PageURL() string { return p.URL }

// Find reads every retirement on the page.
func (p Page) Find(page []byte) (kit.Notices, error) {
	rs, err := Parse(p.Format.Pipes(page), p.Keep)
	return Retirements(rs), err
}

// Events are an event per retired model, citing citeURL. A retirement is a
// dated notice: no previous_version.
func (p Page) Events(found kit.Notices, citeURL, _ string, now time.Time) ([]event.Event, []error) {
	return Events(p, found.(Retirements), citeURL, now)
}

// ShapeGuard is on: past retirements stay on the page, so a read with far
// fewer than the last one means the table changed shape, not that models
// came back.
func (Page) ShapeGuard() bool { return true }
