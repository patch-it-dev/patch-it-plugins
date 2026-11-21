package catalog

import (
	"context"
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
}

func graphqlPages() kittest.Pages {
	pages := kittest.Pages{}
	for _, v := range graphqlCommits {
		pages[GitHubGraphQL.FileURL(v.Ref)] = "testdata/github-graphql-upcoming-changes-" + v.Ref[:8] + ".json"
	}
	return pages
}

func graphqlPair(t *testing.T, from, to plugin.Version) []event.Event {
	t.Helper()
	return kittest.Versions(t, kittest.Env(t, graphqlPages()), GitHubGraphQL, from, to)
}

// The golden test: three consecutive commits, two pairs. 2026-03-03 adds
// Artifact.databaseId and CheckAnnotation.databaseId (removed on
// 2027-01-01, in the file's order), 2026-05-22 adds User.viewerRelevantRepositories (2026-10-01).
// Every entry the commits share is no event.
func TestGraphQLGolden(t *testing.T) {
	var got []event.Event
	var names []string
	for _, pair := range versions.Pairs(graphqlCommits[3:]) {
		for _, e := range graphqlPair(t, pair[0], pair[1]) {
			got = append(got, e)
			names = append(names, fmt.Sprintf("%s->%s %s.%s", e.PreviousVersion, e.Version, e.Subject.Schema, e.Subject.Property))
		}
	}
	want := "2026-02-13->2026-03-03 CheckAnnotation.databaseId | 2026-02-13->2026-03-03 Artifact.databaseId | 2026-03-03->2026-05-22 User.viewerRelevantRepositories"
	if strings.Join(names, " | ") != want {
		t.Errorf("got  %s\nwant %s", strings.Join(names, " | "), want)
	}
	for _, e := range got {
		if e.ChangeType != event.ChangeTypeDeprecation || e.Subject.Kind != event.KindSchemaProperty || !strings.Contains(e.SourceExcerpt, `"criticality":"breaking"`) || strings.Contains(e.SourceExcerpt, "owner") {
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
	from, _ := GitHubGraphQL.ReadAt(context.Background(), kittest.Env(t, graphqlPages()), graphqlCommits[0])
	to, _ := GitHubGraphQL.ReadAt(context.Background(), kittest.Env(t, graphqlPages()), graphqlCommits[1])
	a, _ := entries(from.Content)
	b, _ := entries(to.Content)
	if len(b)-len(a) != 17 {
		t.Errorf("the re-added entries: %d, want 17", len(b)-len(a))
	}
}

// An input object's field and a root field's argument are request
// parameters of POST /graphql; any other location is a property of its type.
func TestGraphQLLocations(t *testing.T) {
	for loc, want := range map[string]string{
		"AddPullRequestReviewInput.comments":     "request_parameter POST /graphql AddPullRequestReviewInput.comments",
		"Query.sponsorables.dependencyEcosystem": "request_parameter POST /graphql Query.sponsorables.dependencyEcosystem",
		"RepositoryPermission.TRIAGE_PLUS":       "schema_property RepositoryPermission TRIAGE_PLUS",
		"Issue.stateReason.enableDuplicate":      "schema_property Issue stateReason.enableDuplicate",
	} {
		s, ok := GitHubGraphQL.subject(loc)
		got := strings.Join(strings.Fields(fmt.Sprintf("%s %s %s %s", s.Kind, s.Operation, s.Schema, s.Property)), " ")
		if !ok || got != want {
			t.Errorf("%s: %q, want %q", loc, got, want)
		}
	}
	if _, ok := GitHubGraphQL.subject("NoDot"); ok {
		t.Error("a location without a field isn't read")
	}
}

func TestGraphQLVersionsAreTheFilesCommits(t *testing.T) {
	env := kittest.Env(t, kittest.Pages{
		"https://api.github.com/repos/github/docs/commits?path=src%2Fgraphql%2Fdata%2Ffpt%2Fupcoming-changes.json&per_page=100&page=1": "testdata/github-commits-upcoming-changes-excerpt.json",
	})
	vs, err := GitHubGraphQL.ListVersions(context.Background(), env)
	if err != nil || len(vs) != 3 || vs[0].Ref != graphqlCommits[3].Ref || vs[2].ID != "2026-05-22" {
		t.Errorf("%v %v", vs, err)
	}
}
