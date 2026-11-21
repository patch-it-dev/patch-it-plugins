package mongodb

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/patch-it-dev/patch-it-plugins/core/diff"
	"github.com/patch-it-dev/patch-it-plugins/core/fetch"
	"github.com/patch-it-dev/patch-it-plugins/core/github"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi/adapter"
)

// Variables, not constants, only so tests can point them at a local server.
var (
	atlasRepo = "mongodb/openapi"
	atlasPath = "openapi/v2.json"
	// atlasCommitsURL lists the commits that changed the spec, newest first;
	// with per_page=1, the newest.
	atlasCommitsURL = "https://api.github.com/repos/mongodb/openapi/commits?path=openapi/v2.json&per_page=1"
	// atlasCommitURL is one commit, for its date.
	atlasCommitURL = "https://api.github.com/repos/mongodb/openapi/commits/%s"
	// atlasSpecURLTemplate is the OpenAPI description of the MongoDB Atlas
	// Administration API v2 at one commit of mongodb/openapi: every
	// operation with every resource version it serves. MongoDB's release
	// workflow commits it ("ci(prod): Release OpenAPI Spec") a few times a
	// week, 768 commits from 2024-07-25 to 2026-11-19 (checked 2026-11-21);
	// the repo's v0.0.N tags mark tooling releases, not the spec.
	atlasSpecURLTemplate = "https://raw.githubusercontent.com/mongodb/openapi/%s/openapi/v2.json"
)

// MongoDBAtlas adapts the Atlas Administration API's spec in mongodb/openapi
// for the openapi plugin. A tag here is a commit SHA.
type MongoDBAtlas struct{}

var _ adapter.Adapter = MongoDBAtlas{}

func (MongoDBAtlas) API() string { return "mongodb/atlas-admin-spec" }

// LatestTag is the SHA of the newest commit that changed the spec.
func (MongoDBAtlas) LatestTag(ctx context.Context) (string, error) {
	sha, err := github.LatestCommit(ctx, github.Get, atlasCommitsURL)
	if err != nil {
		return "", fmt.Errorf("looking up mongodb atlas's latest spec commit: %w", err)
	}
	return sha, nil
}

// Versions are the days the spec changed, each read at that day's last
// commit (Daily).
func (MongoDBAtlas) Versions(ctx context.Context) ([]plugin.Version, error) {
	return adapter.Commits(ctx, atlasRepo, atlasPath)
}

// Fetch downloads the spec at commit sha. Version is the commit's date
// (YYYY-MM-DD): info.version is a fixed "2.0", and the resource versions
// (application/vnd.atlas.2025-03-12+json) are per operation, several in one
// spec. The server is https://cloud.mongodb.com with no path: the paths are
// already what a caller sends (/api/atlas/v2/...).
func (MongoDBAtlas) Fetch(ctx context.Context, sha string) (adapter.Spec, error) {
	fullSHA, date, err := github.Commit(ctx, github.Get, fmt.Sprintf(atlasCommitURL, sha))
	if err != nil {
		return adapter.Spec{}, fmt.Errorf("looking up mongodb atlas spec commit %s: %w", sha, err)
	}
	if fullSHA == "" {
		return adapter.Spec{}, fmt.Errorf("github commit %s has no sha", sha)
	}
	url := fmt.Sprintf(atlasSpecURLTemplate, fullSHA)
	body, err := fetch.Get(ctx, url, nil) // raw.githubusercontent.com needs no auth
	if err != nil {
		return adapter.Spec{}, fmt.Errorf("fetching mongodb atlas spec %s: %w", sha, err)
	}
	if !json.Valid(body) {
		return adapter.Spec{}, fmt.Errorf("mongodb atlas spec %s is not valid json", sha)
	}
	return adapter.Spec{Content: body, Version: date.UTC().Format("2006-01-02"), SourceURL: url}, nil
}

var _ adapter.Lister = MongoDBAtlas{}

// atlasDiff is how the Atlas Administration API's spec says what it says:
//
//   - Deprecations use the `deprecated` flag (53 operations on 2026-11-19).
//     Fourteen operations mention "deprecated" in their description without
//     it, each about the deprecated invitation flow ("This resource cannot
//     be used to view details about users invited via the deprecated
//     Invite One MongoDB Cloud User to Join One Project endpoint"), not the
//     operation itself, so the description isn't read.
//   - Matching by path template keeps a renamed path parameter from reading
//     as a removal.
var atlasDiff = diff.Options{PathTemplates: true, QuoteDeprecatedOperation: true}

// atlasKeep keeps operation deprecations, and removals of operations the old
// spec had marked deprecated (the tenant backup operations, deprecated and
// then gone between 2025-10 and 2026-11). Everything else is dropped:
// operations that leave the spec unmarked (GET /rest/unauth/version on
// 2024-08-07: the drop test), and schema and property changes, which the
// API's own changelog (mongodb/atlas-admin-api, kind changelog) reports per
// resource version.
var atlasKeep = openapi.DeprecationsOnly(atlasDiff)
