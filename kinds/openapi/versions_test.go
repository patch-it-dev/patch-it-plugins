package openapi

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/core/versions"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi/adapter"
)

// listingAdapter is fakeAdapter that can also list its versions.
type listingAdapter struct {
	fakeAdapter
	list []plugin.Version
}

func (l listingAdapter) Versions(context.Context) ([]plugin.Version, error) { return l.list, nil }

var _ adapter.Lister = listingAdapter{}

// Three releases, read version by version: each consecutive pair is its own
// Extract, with to's version and from's as previous_version, and only what
// changed in that pair.
func TestEveryConsecutivePairIsExtracted(t *testing.T) {
	spec := func(v string, props ...string) string {
		var ps []string
		for _, p := range props {
			ps = append(ps, fmt.Sprintf("%q:{\"type\":\"string\"}", p))
		}
		return fmt.Sprintf(`{"openapi":"3.0.0","info":{"version":"version-%s"},"paths":{},"components":{"schemas":{"charge":{"type":"object","properties":{%s}}}}}`, v, strings.Join(ps, ","))
	}
	a := listingAdapter{
		fakeAdapter: fakeAdapter{latest: "v4", specs: map[string]string{
			"v1": spec("v1", "id", "a", "b", "c"),
			"v2": spec("v2", "id", "b", "c"), // a removed
			"v3": spec("v3", "id", "c"),      // b removed
			"v4": spec("v4", "id"),           // c removed
		}},
		list: []plugin.Version{{ID: "v1", Ref: "v1"}, {ID: "v2", Ref: "v2"}, {ID: "v3", Ref: "v3"}, {ID: "v4", Ref: "v4"}},
	}
	p := newPlugin(a, noChangelog)
	ctx, e := context.Background(), env(kittest.NewMem())
	vs, err := p.ListVersions(ctx, e)
	if err != nil || len(vs) != 4 {
		t.Fatalf("ListVersions = %v, %v", vs, err)
	}
	var got []string
	for _, pair := range versions.Pairs(vs) {
		from, err := p.ReadAt(ctx, e, pair[0])
		if err != nil {
			t.Fatal(err)
		}
		to, err := p.ReadAt(ctx, e, pair[1])
		if err != nil {
			t.Fatal(err)
		}
		res, err := p.Extract(ctx, e, from, to)
		if err != nil || len(res.Rejected) > 0 {
			t.Fatalf("Extract: %v %v", err, res.Rejected)
		}
		for _, ev := range res.Events {
			got = append(got, fmt.Sprintf("%s<-%s %s %s", ev.Version, ev.PreviousVersion, ev.Change, ev.Subject.Property))
		}
	}
	want := "version-v2<-version-v1 breaking a|version-v3<-version-v2 breaking b|version-v4<-version-v3 breaking c"
	if strings.Join(got, "|") != want {
		t.Errorf("got  %s\nwant %s", strings.Join(got, "|"), want)
	}
}

// An adapter that can't list is read at its newest version only.
func TestListVersionsFallsBackToTheNewest(t *testing.T) {
	vs, err := newPlugin(fakeAdapter{latest: "v2", specs: specs}, noChangelog).ListVersions(context.Background(), plugin.Env{})
	if err != nil || len(vs) != 1 || vs[0].Ref != "v2" {
		t.Errorf("got %v, %v", vs, err)
	}
}

// specVersioned lists the specs' own versions (as Stripe does).
type specVersioned struct{ fakeAdapter }

func (specVersioned) Versions(context.Context) ([]plugin.Version, error) { return nil, nil }
func (specVersioned) ListsSpecVersions()                                 {}

// For an adapter that lists the specs' own versions, a spec that declares
// another version than the one listed (Stripe's GA spec not yet out at the
// newest release carrying the version) is an error, never read as it.
func TestASpecVersionedReadMustDeclareItsVersion(t *testing.T) {
	p := Plugin{Adapter: specVersioned{fakeAdapter{specs: specs}}}
	s, err := p.ReadAt(context.Background(), plugin.Env{}, plugin.Version{ID: "version-v2", Ref: "v2"})
	if err != nil || s.Version.ID != "version-v2" {
		t.Fatalf("%v %v", s.Version, err)
	}
	if _, err := p.ReadAt(context.Background(), plugin.Env{}, plugin.Version{ID: "version-v3", Ref: "v2"}); err == nil || !strings.Contains(err.Error(), "declares version-v2") {
		t.Errorf("err = %v, want one naming what the spec declares", err)
	}
}
