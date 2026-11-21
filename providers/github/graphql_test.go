package github

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/core/versions"
)

// The commits of github/docs the saved files are at (testdata/README.md),
// oldest first, as GitHub's commit list names them.
var graphqlCommits = []plugin.Version{
	{ID: "2026-01-05", Ref: "fc26275f90e62249fabc0f287255b3e5e60cb6e8"},
	{ID: "2026-01-06", Ref: "54c26c43342f7818c4001af7eaf6a3295e3f822d"},
	{ID: "2026-01-08", Ref: "2706f417a6afec24c57482d62f70035ee35c10fb"},
	{ID: "2026-02-13", Ref: "fc7cedfe9c6b2c0a40fd213cd18c0afc774b67fc"},
	{ID: "2026-03-03", Ref: "3a14e541873e834edf32b43b71911fe80d109a70"},
	{ID: "2026-05-22", Ref: "a934f06d211bef668253e3826c118a4e559dc1ba"},
	{ID: "2026-06-23", Ref: "c5738e5815fff90d38f0166f7c9949bfb664e003"},
}

func graphqlPages() kittest.Pages {
	pages := kittest.Pages{}
	for _, v := range graphqlCommits {
		pages[GraphQL.FileURL(v.Ref)] = "testdata/github-graphql-upcoming-changes-" + v.Ref[:8] + ".json"
	}
	return pages
}

func graphqlPair(t *testing.T, from, to plugin.Version) []event.Event {
	t.Helper()
	return kittest.Versions(t, kittest.Env(t, graphqlPages()), GraphQL, from, to)
}

// The golden test: four consecutive commits, three pairs. 2026-03-03 adds
// CheckAnnotation.databaseId and Artifact.databaseId (removed on
// 2027-01-01, in the file's order), 2026-05-22 User.viewerRelevantRepositories
// and 2026-06-23 CopilotAgentTask.type (both 2026-10-01). Every entry the
// commits share is no event.
func TestGraphQLGolden(t *testing.T) {
	var got []event.Event
	var names []string
	for _, pair := range versions.Pairs(graphqlCommits[3:]) {
		for _, e := range graphqlPair(t, pair[0], pair[1]) {
			got = append(got, e)
			names = append(names, fmt.Sprintf("%s->%s %s.%s", e.PreviousVersion, pair[1].ID, e.Subject.Schema, e.Subject.Property))
		}
	}
	want := "2026-02-13->2026-03-03 CheckAnnotation.databaseId | 2026-02-13->2026-03-03 Artifact.databaseId | 2026-03-03->2026-05-22 User.viewerRelevantRepositories | 2026-05-22->2026-06-23 CopilotAgentTask.type"
	if strings.Join(names, " | ") != want {
		t.Errorf("got  %s\nwant %s", strings.Join(names, " | "), want)
	}
	for _, e := range got {
		if e.Change != event.ChangeDeprecated || e.Subject.Kind != event.KindSchemaProperty || !strings.Contains(e.SourceExcerpt, `"criticality":"breaking"`) || strings.Contains(e.SourceExcerpt, "owner") {
			t.Errorf("%+v", e)
		}
	}
	for i := range got {
		got[i].ID = 0
	}
	kittest.Golden(t, "testdata/golden/github-graphql.json", got)
}

// The pinned drop test: on 2026-01-05 the file lost 17 entries (removals
// dated 2024-07-01 to 2026-01-01, already made), got them back the next day
// and lost them again on 2026-01-08. Putting back a removal already past
// isn't an announcement, and an entry leaving the file says nothing: no
// event in either pair, though the second pair adds 17 entries.
func TestGraphQLRevertsAreNoAnnouncement(t *testing.T) {
	for _, pair := range versions.Pairs(graphqlCommits[:3]) {
		if got := graphqlPair(t, pair[0], pair[1]); len(got) != 0 {
			t.Errorf("%s -> %s: %d events, want 0", pair[0].ID, pair[1].ID, len(got))
		}
	}
	// The second pair does add entries: what the date check drops.
	from, _ := GraphQL.ReadAt(context.Background(), kittest.Env(t, graphqlPages()), graphqlCommits[0])
	to, _ := GraphQL.ReadAt(context.Background(), kittest.Env(t, graphqlPages()), graphqlCommits[1])
	if a, b := countEntries(t, from.Content), countEntries(t, to.Content); b-a != 17 {
		t.Errorf("the re-added entries: %d, want 17", b-a)
	}
}

func TestGraphQLVersionsAreTheFilesCommits(t *testing.T) {
	env := kittest.Env(t, kittest.Pages{
		"https://api.github.com/repos/github/docs/commits?path=src%2Fgraphql%2Fdata%2Ffpt%2Fupcoming-changes.json&per_page=100&page=1": "testdata/github-commits-upcoming-changes-excerpt.json",
	})
	vs, err := GraphQL.ListVersions(context.Background(), env)
	if err != nil || len(vs) != 3 || vs[0].Ref != graphqlCommits[3].Ref || vs[2].ID != "2026-05-22" {
		t.Errorf("%v %v", vs, err)
	}
}

// An entry a commit drops while still upcoming and a later commit restores
// is the same event: its version is the removal date, not the commit's, so
// the fingerprint matches and the store keeps one.
func TestGraphQLARestoredEntryIsTheSameEvent(t *testing.T) {
	first := graphqlPair(t, graphqlCommits[3], graphqlCommits[4])
	// 2026-03-03's file without its two additions, then the file again.
	from, err := GraphQL.ReadAt(context.Background(), kittest.Env(t, graphqlPages()), graphqlCommits[3])
	if err != nil {
		t.Fatal(err)
	}
	to, err := GraphQL.ReadAt(context.Background(), kittest.Env(t, graphqlPages()), graphqlCommits[4])
	if err != nil {
		t.Fatal(err)
	}
	from.Version = plugin.Version{ID: "2026-04-01", Ref: "dropped"}
	to.Version = plugin.Version{ID: "2026-04-02", Ref: "restored"}
	again, err := GraphQL.Extract(context.Background(), kittest.Env(t, nil), from, to)
	if err != nil || len(again.Events) != len(first) {
		t.Fatalf("%v, %d events", err, len(again.Events))
	}
	for i, e := range again.Events {
		if e.Version != "2027-01-01" || e.Fingerprint != first[i].Fingerprint {
			t.Errorf("%s: version %s, same fingerprint %v", e.Subject.Property, e.Version, e.Fingerprint == first[i].Fingerprint)
		}
	}
}

// countEntries is how many entries the file holds, on every date.
func countEntries(t *testing.T, body []byte) int {
	t.Helper()
	var byDate map[string][]json.RawMessage
	if err := json.Unmarshal(body, &byDate); err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, es := range byDate {
		n += len(es)
	}
	return n
}
