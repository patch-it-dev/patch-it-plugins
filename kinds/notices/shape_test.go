package notices

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/kit"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
)

// lines is a page of notices, one name per line; an empty page has none to
// read.
type lines struct{ ratio float64 }

type names []string

func (n names) Count() int            { return len(n) }
func (n names) Has(entry string) bool { return slices.Contains(n, entry) }

func (lines) Slug() string    { return "acme/models" }
func (lines) PageURL() string { return "https://x.test/deprecations" }
func (lines) Find(page []byte) (kit.Notices, error) {
	var out names
	for _, l := range strings.Split(strings.TrimSpace(string(page)), "\n") {
		if l != "" {
			out = append(out, l)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no table: %w", plugin.ErrNothingToRead)
	}
	return out, nil
}
func (lines) Events(kit.Notices, string, string, time.Time) ([]event.Event, []error) { return nil, nil }

// withRatio is a reader with a ratio of its own, as a feed has.
type withRatio struct{ lines }

func (withRatio) MinRatio() float64 { return 0.5 }

// sameArchive serves the page it is given as the snapshot.
type sameArchive struct{ page *string }

func (a sameArchive) Snapshot(context.Context, string) (string, []byte, error) {
	return "https://web.archive.org/web/1/x", []byte(*a.page), nil
}

func page(n int, extra ...string) string {
	var out []string
	for i := 0; i < n; i++ {
		out = append(out, fmt.Sprintf("model-%d", i))
	}
	return strings.Join(append(out, extra...), "\n")
}

// read reads now after a read of before, with the source's shape.
func read(pg Page, shape plugin.Shape, before, now string) error {
	cur := before
	env := plugin.Env{Now: time.Now, Get: func(context.Context, string) ([]byte, error) { return []byte(cur), nil }}
	p := Plugin{Page: pg, Archiver: sameArchive{&cur}}.WithShape(shape).(plugin.Living)
	last, err := p.Read(context.Background(), env, plugin.Snapshot{})
	if err != nil {
		return fmt.Errorf("first read: %w", err)
	}
	cur = now
	_, err = p.Read(context.Background(), env, last)
	return err
}

func TestTheShapeGuardOnALivingPage(t *testing.T) {
	anchored := plugin.Shape{Anchors: []plugin.Anchor{{Entry: "model-0", Checked: "2026-11-21"}}}
	for _, c := range []struct {
		name        string
		pg          Page
		shape       plugin.Shape
		before, now string
		changed     bool
	}{
		{"unchanged count", lines{}, plugin.Shape{}, page(40), page(40, "new-model"), false},
		{"a renamed column hid most rows", lines{}, plugin.Shape{}, page(40), page(10), true},
		{"a redesigned page with no table", lines{}, plugin.Shape{}, page(40), "", true},
		{"a few rows fewer, under the kind's 90%", lines{}, plugin.Shape{}, page(40), page(30), true},
		{"the same, with the source's own ratio", lines{}, plugin.Shape{MinRatio: 0.7}, page(40), page(30), false},
		{"a feed's window shrank a little", withRatio{}, plugin.Shape{}, page(40), page(30), false},
		{"a feed lost most entries", withRatio{}, plugin.Shape{}, page(40), page(10), true},
		{"the anchor is there", lines{}, anchored, page(40), page(41), false},
		{"the anchor is gone", lines{}, anchored, page(40), strings.Replace(page(41), "model-0\n", "", 1), true},
	} {
		err := read(c.pg, c.shape, c.before, c.now)
		if errors.Is(err, plugin.ErrShapeChanged) != c.changed {
			t.Errorf("%s: %v, want changed %v", c.name, err, c.changed)
		}
	}
	// An anchor is checked on a first read too.
	cur := page(5)
	env := plugin.Env{Now: time.Now, Get: func(context.Context, string) ([]byte, error) { return []byte(cur), nil }}
	p := Plugin{Page: lines{}, Archiver: sameArchive{&cur}}.WithShape(plugin.Shape{Anchors: []plugin.Anchor{{Entry: "gpt-4", Checked: "2026-11-21"}}}).(plugin.Living)
	if _, err := p.Read(context.Background(), env, plugin.Snapshot{}); !errors.Is(err, plugin.ErrShapeChanged) {
		t.Errorf("first read without the anchor: %v", err)
	}
	// Built by hand, no Shape: no guard.
	if _, err := (Plugin{Page: lines{}, Archiver: sameArchive{&cur}}).Read(context.Background(), env, plugin.Snapshot{}); err != nil {
		t.Errorf("unguarded: %v", err)
	}
}
