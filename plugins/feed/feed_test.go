package feed

import (
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/kit/kittest"
)

// The two feeds as saved on 2026-11-14 (testdata/README.md), each through the
// whole plugin: live feed and snapshot are the same copy, so the events cite
// the snapshot, and kittest's second run over the unchanged feed asks for no
// second snapshot and stores nothing new.
func TestGolden(t *testing.T) {
	for _, c := range []struct {
		feed Feed
		file string
	}{
		{GitHubActions, "testdata/github-changelog-actions.rss.xml"},
		{VertexAIModels, "testdata/vertex-ai-generative-ai.atom.xml"},
	} {
		t.Run(c.feed.API, func(t *testing.T) {
			saved := kittest.Pages{c.feed.URL: c.file}
			archive := &kittest.Archive{T: t, Pages: saved}
			got := kittest.IngestWith(t, kittest.Env(t, saved), Plugin{Feed: c.feed, Archiver: archive})
			if archive.Calls != 1 {
				t.Errorf("%d snapshots, want 1", archive.Calls)
			}
			kittest.Golden(t, "testdata/golden/"+c.feed.API+".json", got)
		})
	}
}

// What each feed's marked entries came to, kept and dropped: the golden files
// pin the events, this pins that the unsure ones were dropped, not guessed.
func TestUnsureEntriesAreDropped(t *testing.T) {
	for _, c := range []struct {
		feed    Feed
		file    string
		dropped []string // a substring of each drop reason, in order
	}{
		{GitHubActions, "testdata/github-changelog-actions.rss.xml", []string{
			"arm32-linux-windows-retirement",                    // no runner label: no ci_runtime to name
			"pull_request_target-blocked-by-default-december-2", // a behavior change, no subject type fits
		}},
		{VertexAIModels, "testdata/vertex-ai-generative-ai.atom.xml", []string{
			"Vertex AI Extensions deprecation",
			"Anthropic's Claude 3 Haiku", // a product name in prose, not a model id
			"Anthropic's Claude 3.5 Haiku",
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

func TestParseRefusesWhatIsntAFeed(t *testing.T) {
	for name, b := range map[string]string{
		"html":     "<!doctype html><html><body>moved</body></html>",
		"no items": `<rss version="2.0"><channel><title>x</title></channel></rss>`,
		"no entry": `<feed xmlns="http://www.w3.org/2005/Atom"><id>x</id></feed>`,
	} {
		if _, err := Parse([]byte(b)); err == nil {
			t.Errorf("%s: want an error, never zero events read as a quiet week", name)
		}
	}
}

// RSS without content:encoded falls back to description; Atom takes the
// alternate link and the category terms.
func TestParseReadsBothFormats(t *testing.T) {
	r, err := Parse([]byte(`<rss version="2.0"><channel><item><title>A &amp; B</title><link>https://x.test/a</link>
		<category>Retired</category><description>&lt;p&gt;text&lt;/p&gt;</description></item></channel></rss>`))
	if err != nil || r[0].Title != "A & B" || r[0].HTML != "<p>text</p>" || r[0].Categories[0] != "Retired" || r[0].ID != "https://x.test/a" {
		t.Errorf("rss: %+v %v", r, err)
	}
	a, err := Parse([]byte(`<feed xmlns="http://www.w3.org/2005/Atom"><entry><id>i</id><title>T</title>
		<link rel="self" href="https://x.test/self"/><link rel="alternate" href="https://x.test/e"/>
		<category term="Deprecated"/><summary type="html">&lt;b&gt;s&lt;/b&gt;</summary></entry></feed>`))
	if err != nil || a[0].Link != "https://x.test/e" || a[0].HTML != "<b>s</b>" || a[0].Categories[0] != "Deprecated" {
		t.Errorf("atom: %+v %v", a, err)
	}
}

// Breaking only when the vendor says the thing stops on the date; softer
// words downgrade to a deprecation, and no date drops the entry.
func TestBreakingNeedsTheVendorToSayItStops(t *testing.T) {
	entry := func(sentence string) Entry {
		return Entry{Title: "Ubuntu 22.04 runner image retirement", Link: "https://x.test/e", Categories: []string{"Retired"},
			HTML: "<p>" + sentence + "</p><p>This deprecation includes the following labels:</p><ul><li><code>ubuntu-22.04</code></li></ul>"}
	}
	for sentence, want := range map[string]string{
		"The Ubuntu 22.04 runner image will be retired on March 1, 2027.":     "breaking 2027-03-01",
		"The Ubuntu 22.04 runner image will be removed on March 1, 2027.":     "breaking 2027-03-01",
		"The Ubuntu 22.04 runner image will be deprecated on March 1, 2027.":  "deprecation 2027-03-01",
		"Please move off the Ubuntu 22.04 runner image before March 1, 2027.": "deprecation 2027-03-01",
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
