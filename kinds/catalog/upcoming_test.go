package catalog

import (
	"fmt"
	"strings"
	"testing"
)

// An input object's field and a root field's argument are request
// parameters of POST /graphql; any other location is a property of its type.
func TestUpcomingChangesLocations(t *testing.T) {
	for loc, want := range map[string]string{
		"AddPullRequestReviewInput.comments":     "request_parameter POST /graphql AddPullRequestReviewInput.comments",
		"Query.sponsorables.dependencyEcosystem": "request_parameter POST /graphql Query.sponsorables.dependencyEcosystem",
		"RepositoryPermission.TRIAGE_PLUS":       "schema_property RepositoryPermission TRIAGE_PLUS",
		"Issue.stateReason.enableDuplicate":      "schema_property Issue stateReason.enableDuplicate",
	} {
		s, ok := graphql.subject(loc)
		got := strings.Join(strings.Fields(fmt.Sprintf("%s %s %s %s", s.Kind, s.Operation, s.Schema, s.Property)), " ")
		if !ok || got != want {
			t.Errorf("%s: %q, want %q", loc, got, want)
		}
	}
	if _, ok := graphql.subject("NoDot"); ok {
		t.Error("a location without a field isn't read")
	}
}

// graphql is an UpcomingChanges like GitHub's GraphQL API's.
var graphql = UpcomingChanges{API: "github/graphql", Vendor: "GitHub", Repo: "github/docs", Path: "src/graphql/data/fpt/upcoming-changes.json", Endpoint: "POST /graphql"}
