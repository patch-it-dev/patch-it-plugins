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

// Slug is the events' api: "openai-models".
func (p Page) Slug() string { return p.API }

// PageURL is the page, as markdown.
func (p Page) PageURL() string { return p.URL }

// Find reads every retirement on the page.
func (p Page) Find(page []byte) (kit.Notices, error) {
	rs, err := Parse(page, p.Keep)
	return Retirements(rs), err
}

// Events are a v1 event per retired model, citing citeURL (the kind
// converts them to v2: core/migrate reads this package, so it can't be
// called from here).
func (p Page) Events(found kit.Notices, citeURL string, now time.Time) ([]event.ChangeEvent, []event.EventV2, []error) {
	v1, errs := Events(p, found.(Retirements), citeURL, now)
	return v1, nil, errs
}

// ShapeGuard is on: past retirements stay on the page, so a read with far
// fewer than the last one means the table changed shape, not that models
// came back.
func (Page) ShapeGuard() bool { return true }
