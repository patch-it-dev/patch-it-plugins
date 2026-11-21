// Package catalog is the kind for a provider's own structured list of what
// it offers or is about to take away: a command catalog, a file of upcoming
// removals. Unlike openapi's typed spec, an entry here is opaque beyond its
// name (and, for a list of upcoming changes, its date and what to use
// instead). Each reader is a plugin of its own in this package, read
// version by version (plugin.Versioned): Redis's commands at each release
// tag (redis.go), GitHub's GraphQL upcoming changes at each commit
// (graphql.go).
//
// Precision over recall: a name that moves (a genuine rename) reads as one
// removal and one addition, same as any other. A reader never guesses a
// rename from a disappearance and an appearance landing in the same diff.
package catalog

import (
	"fmt"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
)

func init() {
	plugin.RegisterProvider(plugin.Provider{
		ID: "redis", Name: "Redis", Category: plugin.CategoryDatabases, Homepage: "https://redis.io",
		Sources: []plugin.Source{{
			ID: "commands", Layer: plugin.LayerRuntime, Page: "providers/redis.md",
			Watches: "Commands removed entirely between two Redis releases, from the command definitions in the server's own source (src/commands/) at each release tag",
			URLs:    []string{"https://github.com/redis/redis/tree/unstable/src/commands", "https://github.com/redis/redis/releases"},
			Plugin:  Redis,
		}},
	})
	plugin.RegisterProvider(plugin.Provider{
		ID: "github", Name: "GitHub", Category: plugin.CategoryDevTools, Homepage: "https://github.com",
		Sources: []plugin.Source{{
			ID: "graphql", Layer: plugin.LayerAPI, Page: "providers/github-graphql.md",
			Watches:  "GraphQL API fields, arguments, input fields and enum values GitHub announces it will remove, with the date, from the file its breaking-changes page is built from",
			URLs:     []string{"https://docs.github.com/en/graphql/overview/breaking-changes", "https://github.com/github/docs/blob/main/src/graphql/data/fpt/upcoming-changes.json"},
			Packages: []string{"npm:@octokit/graphql"},
			Plugin:   GitHubGraphQL,
		}},
	})
}

// finish fingerprints and validates the events a reader made, keeping the
// valid ones and the reasons the others failed. details, when given, are
// the fingerprints' detail, one per event: what separates two changes at
// one location.
func finish(api string, made []event.Event, details []string) (events []event.Event, rejected []error) {
	for i, e := range made {
		detail := ""
		if details != nil {
			detail = details[i]
		}
		e.Fingerprint = event.Fingerprint(e.API, e.Version, e.ChangeType, e.Subject, detail)
		if err := e.Validate(); err != nil {
			rejected = append(rejected, fmt.Errorf("%s %s: %w", api, e.Description, err))
			continue
		}
		events = append(events, e)
	}
	return events, rejected
}

// excerpt caps a quote at the contract's 2000 characters.
func excerpt(s string) string {
	if r := []rune(s); len(r) > 2000 {
		s = string(r[:1999]) + "…"
	}
	return s
}
