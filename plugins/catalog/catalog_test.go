package catalog

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/fetch"
	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
)

// redisPair is the version pair from -> to over the saved listings (and the
// removed files at from's tag), as the hosted ingestion service extracts it.
func redisPair(t *testing.T, from, to string, files ...string) []event.Event {
	t.Helper()
	pages := kittest.Pages{
		Redis.listingURL(from): "testdata/redis-contents-src-commands-" + from + ".json",
		Redis.listingURL(to):   "testdata/redis-contents-src-commands-" + to + ".json",
	}
	for _, f := range files {
		pages[Redis.FileURL(from, f)] = "testdata/redis-" + from + "-" + f
	}
	return kittest.Versions(t, kittest.Env(t, pages), Redis, plugin.Version{ID: from, Ref: from}, plugin.Version{ID: to, Ref: to})
}

// The one removal in redis/redis's src/commands/ between any two tags from
// 7.0 to 8.12 (checked 2026-11-20): GCRA and GCRASETVALUE, added in the
// 8.8 milestone builds, are gone at 8.8-rc1. Final releases never removed a
// command in that span, so the golden pins this pre-release pair: the real
// files, the real listings.
func TestGolden(t *testing.T) {
	got := redisPair(t, "8.8-m03", "8.8-rc1", "gcra.json", "gcrasetvalue.json")
	var names []string
	for _, e := range got {
		names = append(names, e.Subject.Symbols[0])
		if e.SourceURL != Redis.FileURL("8.8-m03", strings.ToLower(e.Subject.Symbols[0])+".json") || !strings.Contains(e.SourceExcerpt, `"since":"8.8.0"`) {
			t.Errorf("%s: cites %s, quoting %.80q", e.Subject.Symbols[0], e.SourceURL, e.SourceExcerpt)
		}
	}
	if !slices.Equal(names, []string{"GCRA", "GCRASETVALUE"}) {
		t.Errorf("removed %v, want GCRA and GCRASETVALUE", names)
	}
	kittest.Golden(t, "testdata/golden/redis-commands.json", got)
}

// The pinned drop test: 8.8.0 -> 8.10.0 adds 17 command files (the array
// commands, INCREX…) and removes none. Nothing is an event, and nothing
// past the two listings is fetched.
func TestReleasesThatOnlyAddCommandsHaveNoEvents(t *testing.T) {
	if got := redisPair(t, "8.8.0", "8.10.0"); len(got) != 0 {
		t.Errorf("%d events, want 0", len(got))
	}
}

// A subcommand is named with its container, as code sends it.
func TestASubcommandIsNamedWithItsContainer(t *testing.T) {
	name, _, err := command([]byte(`{"KILL": {"summary": "x", "container": "CLIENT"}}`))
	if err != nil || name != "CLIENT KILL" {
		t.Errorf("%q, %v", name, err)
	}
	if _, _, err := command([]byte(`{"A": {}, "B": {}}`)); err == nil {
		t.Error("two commands in one file must be an error, not a guess")
	}
}

// Versions are the final releases on GitHub, by version number (a patch to
// an older line published later stays in its place), each read at its tag.
// A release before src/commands/ existed has nothing to read.
func TestVersionsAreFinalReleasesByVersion(t *testing.T) {
	env := kittest.Env(t, kittest.Pages{
		"https://api.github.com/repos/redis/redis/releases?per_page=100&page=1": "testdata/redis-releases-excerpt.json",
	})
	vs, err := Redis.ListVersions(context.Background(), env)
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, v := range vs {
		ids = append(ids, v.ID+"@"+v.Ref)
	}
	if got := strings.Join(ids, " "); got != "6.2.20@6.2.20 7.2.10@7.2.10 8.0.0@8.0.0 8.0.1@8.0.1" {
		t.Errorf("versions %s", got)
	}
	notFound := plugin.Env{Get: func(context.Context, string) ([]byte, error) { return nil, &fetch.StatusError{StatusCode: 404} }}
	if _, err := Redis.ReadAt(context.Background(), notFound, plugin.Version{ID: "6.2.20", Ref: "6.2.20"}); !errors.Is(err, plugin.ErrNothingToRead) {
		t.Errorf("no src/commands/ at the tag: %v, want plugin.ErrNothingToRead", err)
	}
}
