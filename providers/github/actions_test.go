package github

import (
	"errors"
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/extract"
	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/kinds/notices/feed"
	"github.com/patch-it-dev/patch-it-plugins/kinds/notices/noticestest"
)

// The feed as saved on 2026-11-14 (testdata/README.md) through the whole
// plugin: live feed and snapshot are the same copy, so the events cite the
// snapshot.
func TestActionsGolden(t *testing.T) {
	noticestest.Golden(t, Actions, "testdata/github-changelog-actions.rss.xml")
}

// The URL serving another feed (the label renamed or dropped, the changelog's
// main feed served instead) reads fine as a feed: only the anchor, the
// feed's own title, says it isn't the Actions label any more.
func TestActionsAnotherFeedChangedShape(t *testing.T) {
	saved := string(kittest.Read(t, "testdata/github-changelog-actions.rss.xml"))
	src, _ := plugin.LookupSource(Actions.API)
	for name, page := range map[string]string{
		"saved":        saved,
		"another feed": strings.Replace(saved, "<title>Use Case: actions - GitHub Changelog</title>", "<title>The GitHub Blog</title>", 1),
	} {
		found, err := Actions.Find([]byte(page))
		if err != nil {
			t.Fatal(err)
		}
		err = src.Source.Shape.CheckAnchors(found.Has)
		if changed := errors.Is(err, plugin.ErrShapeChanged); changed != (name != "saved") {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// What the feed's marked entries came to, kept and dropped: the golden files
// pin the events, this pins that the unsure ones were dropped, not guessed.
func TestActionsUnsureEntriesAreDropped(t *testing.T) {
	for _, c := range []struct {
		feed    feed.Feed
		file    string
		dropped []string // a substring of each drop reason, in order
	}{
		{Actions, "testdata/github-changelog-actions.rss.xml", []string{
			"arm32-linux-windows-retirement",                    // no runner label: no runtime to name
			"pull_request_target-blocked-by-default-december-2", // a behavior change, no subject type fits
		}},
	} {
		fs, err := c.feed.Findings(kittest.Read(t, c.file))
		if err != nil {
			t.Fatal(err)
		}
		var drops []string
		for _, f := range fs {
			if f.Dropped != "" {
				drops = append(drops, f.Entry+" "+f.Dropped)
			}
		}
		if len(drops) != len(c.dropped) {
			t.Errorf("%s: dropped %d, want %d:\n%s", c.feed.API, len(drops), len(c.dropped), strings.Join(drops, "\n"))
			continue
		}
		for i, want := range c.dropped {
			if !strings.Contains(drops[i], want) {
				t.Errorf("%s: drop %d is %q, want %q", c.feed.API, i, drops[i], want)
			}
		}
	}
}

// Breaking only when the vendor says the thing stops on the date; softer
// words downgrade to a deprecation, and no date drops the entry.
func TestBreakingNeedsTheVendorToSayItStops(t *testing.T) {
	entry := func(sentence string) extract.Entry {
		return extract.Entry{Title: "Ubuntu 22.04 runner image retirement", Link: "https://x.test/e", Categories: []string{"Retired"},
			HTML: "<p>" + sentence + "</p><p>This deprecation includes the following labels:</p><ul><li><code>ubuntu-22.04</code></li></ul>"}
	}
	for sentence, want := range map[string]string{
		"The Ubuntu 22.04 runner image will be retired on March 1, 2027.":     "breaking 2027-03-01",
		"The Ubuntu 22.04 runner image will be removed on March 1, 2027.":     "breaking 2027-03-01",
		"The Ubuntu 22.04 runner image will be deprecated on March 1, 2027.":  "deprecated 2027-03-01",
		"Please move off the Ubuntu 22.04 runner image before March 1, 2027.": "deprecated 2027-03-01",
		"The Ubuntu 22.04 runner image will be retired in the coming months.": "dropped",
		"Announced March 1, 2027: the Ubuntu 22.04 image is being looked at.": "dropped",
	} {
		fs := readGitHubActions(entry(sentence))
		got := "dropped"
		if len(fs) == 1 && fs[0].Dropped == "" {
			got = string(fs[0].ChangeType) + " " + fs[0].Version
		}
		if got != want {
			t.Errorf("%q: %s, want %s", sentence, got, want)
		}
	}
	unmarked := entry("The Ubuntu 22.04 runner image will be retired on March 1, 2027.")
	unmarked.Categories = []string{"Improvement"}
	if fs := readGitHubActions(unmarked); len(fs) != 0 {
		t.Errorf("an entry the vendor didn't mark: %+v", fs)
	}
}
