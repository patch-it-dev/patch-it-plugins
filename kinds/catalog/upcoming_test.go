package catalog

import (
	"fmt"
	"strings"
	"testing"
)

// A location is the type, the field and, when it names one, the argument.
func TestUpcomingChangesLocations(t *testing.T) {
	for loc, want := range map[string]string{
		"AddPullRequestReviewInput.comments":     "AddPullRequestReviewInput comments",
		"Query.sponsorables.dependencyEcosystem": "Query sponsorables dependencyEcosystem",
		"RepositoryPermission.TRIAGE_PLUS":       "RepositoryPermission TRIAGE_PLUS",
		"Issue.stateReason.enableDuplicate":      "Issue stateReason enableDuplicate",
	} {
		s, ok := graphql.subject(loc)
		got := strings.Join(strings.Fields(fmt.Sprintf("%s %s %s", s.TypeName, s.Field, s.Argument)), " ")
		if !ok || s.Type != "graphql_field" || got != want {
			t.Errorf("%s: %q, want %q", loc, got, want)
		}
	}
	for _, loc := range []string{"NoDot", "A.b.c.d", "A..b"} {
		if _, ok := graphql.subject(loc); ok {
			t.Errorf("%s: read, want dropped", loc)
		}
	}
}

// graphql is an UpcomingChanges like GitHub's GraphQL API's.
var graphql = UpcomingChanges{API: "github/graphql", Vendor: "GitHub", Repo: "github/docs", Path: "src/graphql/data/fpt/upcoming-changes.json"}
