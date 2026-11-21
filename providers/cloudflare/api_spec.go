package cloudflare

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/patch-it-dev/patch-it-plugins/core/diff"
	"github.com/patch-it-dev/patch-it-plugins/core/fetch"
	"github.com/patch-it-dev/patch-it-plugins/core/github"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi/adapter"
)

// Variables, not constants, only so tests can point them at a local server.
var (
	cloudflareRepo = "cloudflare/api-schemas"
	cloudflarePath = "openapi.json"
	// cloudflareCommitsURL lists the commits that changed the spec, newest
	// first; with per_page=1, the newest.
	cloudflareCommitsURL = "https://api.github.com/repos/cloudflare/api-schemas/commits?path=openapi.json&per_page=1"
	// cloudflareCommitURL is one commit, for its date.
	cloudflareCommitURL = "https://api.github.com/repos/cloudflare/api-schemas/commits/%s"
	// cloudflareSpecURLTemplate is the OpenAPI description of the whole
	// Cloudflare API (api.cloudflare.com/client/v4) at one commit of
	// cloudflare/api-schemas (BSD-3-Clause), the spec Cloudflare's SDKs and
	// Terraform provider are generated from. A bot commits it as "Update
	// schema: openapi.json" several times a day and the repo has no tags
	// (checked 2026-11-21), so the commits are the versions. About 30 MB;
	// openapi.yaml beside it is the same spec.
	cloudflareSpecURLTemplate = "https://raw.githubusercontent.com/cloudflare/api-schemas/%s/openapi.json"
)

// Cloudflare adapts the Cloudflare API's spec in cloudflare/api-schemas for
// the openapi plugin. A tag here is a commit SHA.
type Cloudflare struct{}

var _ adapter.Adapter = Cloudflare{}

func (Cloudflare) API() string { return "cloudflare/api-spec" }

// LatestTag is the SHA of the newest commit that changed the spec.
func (Cloudflare) LatestTag(ctx context.Context) (string, error) {
	sha, err := github.LatestCommit(ctx, github.Get, cloudflareCommitsURL)
	if err != nil {
		return "", fmt.Errorf("looking up cloudflare's latest spec commit: %w", err)
	}
	return sha, nil
}

// Versions are the days the spec changed, each read at that day's last
// commit (Daily).
func (Cloudflare) Versions(ctx context.Context) ([]plugin.Version, error) {
	return adapter.Commits(ctx, cloudflareRepo, cloudflarePath)
}

// Fetch downloads the spec at commit sha and moves the server's /client/v4
// into the paths: see ServerBasePath. Version is the commit's date
// (YYYY-MM-DD): info.version is a fixed "4.0.0" (checked 2026-11-21).
func (Cloudflare) Fetch(ctx context.Context, sha string) (adapter.Spec, error) {
	fullSHA, date, err := github.Commit(ctx, github.Get, fmt.Sprintf(cloudflareCommitURL, sha))
	if err != nil {
		return adapter.Spec{}, fmt.Errorf("looking up cloudflare spec commit %s: %w", sha, err)
	}
	if fullSHA == "" {
		return adapter.Spec{}, fmt.Errorf("github commit %s has no sha", sha)
	}
	url := fmt.Sprintf(cloudflareSpecURLTemplate, fullSHA)
	body, err := fetch.Get(ctx, url, nil) // raw.githubusercontent.com needs no auth
	if err != nil {
		return adapter.Spec{}, fmt.Errorf("fetching cloudflare spec %s: %w", sha, err)
	}
	if !json.Valid(body) {
		return adapter.Spec{}, fmt.Errorf("cloudflare spec %s is not valid json", sha)
	}
	// The one server is "https://api.cloudflare.com/client/v4" and the
	// paths don't repeat it: moved, an operation is named by the path a
	// caller sends.
	content, err := adapter.ServerBasePath(body)
	if err != nil {
		return adapter.Spec{}, fmt.Errorf("cloudflare spec %s: %w", sha, err)
	}
	return adapter.Spec{Content: content, Version: date.UTC().Format("2006-01-02"), SourceURL: url}, nil
}

// SourcePointer: see BasePathPointer.
func (Cloudflare) SourcePointer(content []byte) func(string) string {
	return adapter.BasePathPointer(content)
}

var (
	_ adapter.Reshaper = Cloudflare{}
	_ adapter.Lister   = Cloudflare{}
)

// cloudflareDiff is how Cloudflare's spec says what it says:
//
//   - Most deprecations use the `deprecated` flag (329 operations in the spec
//     of 2026-11-20), and a few only the summary, written as Vercel's are: "List
//     datasets (deprecated)", "Deprecated: Retrieve legacy embed code HTML"
//     (11 of them). Other summaries and descriptions that say "deprecated"
//     are about a parameter or a model, not the operation, so the
//     description isn't read.
//   - Path parameters are renamed without the URL changing:
//     /zones/{zone_identifier}/analytics/dashboard became
//     /zones/{zone_id}/analytics/dashboard on 2026-10-30. Matching by path
//     template keeps that from reading as a removal.
var cloudflareDiff = diff.Options{PathTemplates: true, QuoteDeprecatedOperation: true, DeprecatedSummary: cloudflareDeprecatedSummary}

// cloudflareDeprecatedWords: a summary that starts "Deprecated:" or ends
// "(deprecated)". Not "deprecat" anywhere ("Omitting `datasetId` is
// deprecated" is about a parameter).
var cloudflareDeprecatedWords = regexp.MustCompile(`(?i)^deprecated:|\(deprecated\)\s*$`)

func cloudflareDeprecatedSummary(summary string) bool {
	return cloudflareDeprecatedWords.MatchString(summary)
}

// cloudflareKeep keeps operation deprecations, and removals of operations the
// old spec had marked deprecated. Everything else is dropped: a bot commits
// the 30 MB spec several times a day, and an operation can leave it without
// leaving the API or without notice (between 2026-10-29 and 2026-10-30 POST
// /accounts/move and four GET /tenants/{tenant_id} operations left it, never
// marked deprecated: the drop test), while property changes in the
// generated schemas are mostly the generator's.
var cloudflareKeep = openapi.DeprecationsOnly(cloudflareDiff)
