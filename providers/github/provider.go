// Package github is what patch-it reads for GitHub: every source, registered
// below with its metadata, each read by its kind (kinds/<kind>). README.md
// says why each source and what it reads.
package github

import (
	"github.com/patch-it-dev/patch-it-plugins/core/kit"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/kinds/catalog"
	"github.com/patch-it-dev/patch-it-plugins/kinds/changelog"
	"github.com/patch-it-dev/patch-it-plugins/kinds/notices"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi"
)

func init() {
	plugin.RegisterProvider(plugin.Provider{
		ID: "github", Name: "GitHub", Category: plugin.CategoryDevTools, Homepage: "https://github.com",
		Sources: []plugin.Source{
			// No Hints yet: GitHub's per-version CHANGELOG.md is prose with no table
			// at all, so there's nothing for hint.Source to key on. (GitHub's Actions
			// source is registered by plugins/notices.)
			{
				ID: "rest-api", Layer: plugin.LayerAPI, Watches: "REST API changes between dated API versions",
				URLs:   []string{"https://github.com/github/rest-api-description"},
				Plugin: openapi.Plugin{Adapter: GitHub{}},
			},
			{
				ID: "actions", Layer: plugin.LayerCI, Watches: "GitHub Actions retirements, such as runner images and the `runs-on` labels they remove",
				URLs:   []string{"https://github.blog/changelog/label/actions/feed/"},
				Plugin: notices.Plugin{Page: Actions, Archiver: kit.Wayback{}},
			},
			{
				ID: "graphql", Layer: plugin.LayerAPI, Watches: "GraphQL API fields, arguments, input fields and enum values GitHub announces it will remove, with the date, from the file its breaking-changes page is built from",
				URLs:     []string{"https://docs.github.com/en/graphql/overview/breaking-changes", "https://github.com/github/docs/blob/main/src/graphql/data/fpt/upcoming-changes.json"},
				Packages: []string{"npm:@octokit/graphql"},
				Plugin:   GraphQL,
			},
			changelog.SDK("octokit-rest", OctokitRest, "npm:@octokit/rest"),
			changelog.SDK("octokit-net", OctokitNet, "nuget:Octokit"),
		},
	})
}

// Octokit's SDKs keep no changelog file: each GitHub release's notes are it
// (semantic-release's "### BREAKING CHANGES" for rest.js, "Breaking
// changes" lists for octokit.net); tagged vX.Y.Z.
var (
	OctokitRest = changelog.File{API: "github/octokit-rest", Package: "@octokit/rest", Ecosystem: "npm",
		Repo: "octokit/rest.js", TagPrefix: "v", Notes: true}
	OctokitNet = changelog.File{API: "github/octokit-net", Package: "Octokit", Ecosystem: "nuget",
		Repo: "octokit/octokit.net", TagPrefix: "v", Notes: true}
)

// GraphQL is GitHub's GraphQL API's upcoming changes (checked
// 2026-11-21: 1,291 entries, every one "criticality": "breaking", 29
// commits since 2023-05-01).
var GraphQL = catalog.UpcomingChanges{
	API: "github/graphql", Vendor: "GitHub",
	Repo: "github/docs", Path: "src/graphql/data/fpt/upcoming-changes.json",
	Endpoint: "POST /graphql",
}
