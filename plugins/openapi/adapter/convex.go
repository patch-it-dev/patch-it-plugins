package adapter

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/patch-it-dev/patch-it-plugins/core/fetch"
	"github.com/patch-it-dev/patch-it-plugins/core/github"
)

// Variables, not constants, only so tests can point them at a local server.
var (
	// convexCommitsURL lists the commits that changed the Deployment API
	// spec in get-convex/convex-backend, newest first; with per_page=1, the
	// newest.
	convexCommitsURL = "https://api.github.com/repos/get-convex/convex-backend/commits?path=npm-packages/@convex-dev/platform/deployment-openapi.json&per_page=1"
	// convexCommitURL is one commit, for its date.
	convexCommitURL = "https://api.github.com/repos/get-convex/convex-backend/commits/%s"
	// convexSpecURLTemplate is the Deployment API spec at one commit: 68
	// commits from 2025-08-19 to 2026-11-16 (checked 2026-11-19), and a
	// commit never changes.
	convexSpecURLTemplate = "https://raw.githubusercontent.com/get-convex/convex-backend/%s/npm-packages/@convex-dev/platform/deployment-openapi.json"
)

// Convex adapts the Convex Deployment API spec (`{deployment-url}/api/v1`)
// in get-convex/convex-backend for the openapi plugin. A tag here is the
// commit that last changed the spec, as for Supabase.
//
// The repo does tag releases, `precompiled-YYYY-MM-DD-<sha>` (1,213 of
// them), but they are daily builds of the self-hosted backend, not releases
// of the API: several a day, ordered only by commit date, the spec unchanged
// in most. The spec at a tag is the spec at the last commit before it that
// changed the file, so reading those commits sees every state a tag would,
// with one GitHub call for the newest instead of paging through the tags.
type Convex struct{}

var _ Adapter = Convex{}

func (Convex) API() string { return "convex/deployment-api" }

// LatestTag is the SHA of the newest commit that changed the spec.
func (Convex) LatestTag(ctx context.Context) (string, error) {
	sha, err := github.LatestCommit(ctx, github.Get, convexCommitsURL)
	if err != nil {
		return "", fmt.Errorf("looking up convex's latest spec commit: %w", err)
	}
	return sha, nil
}

// Fetch downloads the spec at commit sha.
//
// Version is the commit's committer date (YYYY-MM-DD, UTC): the Deployment
// API has no version of its own (`/api/v1`, info.version a fixed "1.0.0" in
// every commit, checked 2026-11-19), so the date the spec changed is the
// closest thing to "the API version the change arrived in".
func (Convex) Fetch(ctx context.Context, sha string) (Spec, error) {
	fullSHA, date, err := github.Commit(ctx, github.Get, fmt.Sprintf(convexCommitURL, sha))
	if err != nil {
		return Spec{}, fmt.Errorf("looking up convex spec commit %s: %w", sha, err)
	}
	if fullSHA == "" {
		return Spec{}, fmt.Errorf("github commit %s has no sha or date", sha)
	}
	// Pin to the full SHA even when given a short one: the citation must
	// name exactly one document.
	url := fmt.Sprintf(convexSpecURLTemplate, fullSHA)
	body, err := fetch.Get(ctx, url, nil) // raw.githubusercontent.com needs no auth
	if err != nil {
		return Spec{}, fmt.Errorf("fetching convex spec %s: %w", sha, err)
	}
	if !json.Valid(body) {
		return Spec{}, fmt.Errorf("convex spec %s is not valid json", sha)
	}
	// The server is "{deployment-url}/api/v1" and the paths leave out /api/v1
	// (all 68 commits, checked 2026-11-19). Moved into the paths, an
	// operation is named by the path a caller sends, and a spec that ever
	// moved it (as Sentry's did) can't read as every deprecated operation
	// removed.
	content, err := ServerBasePath(body)
	if err != nil {
		return Spec{}, fmt.Errorf("convex spec %s: %w", sha, err)
	}
	return Spec{Content: content, Version: date.UTC().Format("2006-01-02"), SourceURL: url}, nil
}

// SourcePointer: see BasePathPointer.
func (Convex) SourcePointer(content []byte) func(string) string { return BasePathPointer(content) }

var _ Reshaper = Convex{}
