package supabase

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
	// supabaseCommitsURL lists the commits that changed the Management API
	// spec in supabase/supabase, newest first; with per_page=1, the newest.
	supabaseCommitsURL = "https://api.github.com/repos/supabase/supabase/commits?path=apps/docs/spec/api_v1_openapi.json&per_page=1"
	// supabaseCommitURL is one commit, for its date.
	supabaseCommitURL = "https://api.github.com/repos/supabase/supabase/commits/%s"
	// supabaseSpecURLTemplate is the Management API spec (api.supabase.com)
	// at one commit. Supabase serves it live at
	// https://api.supabase.com/api/v1-json, which has no versions; the docs
	// copy in supabase/supabase is the same spec, committed when it changes
	// (90 commits from 2024-05-31 to 2026-11-18), and a commit never changes.
	supabaseSpecURLTemplate = "https://raw.githubusercontent.com/supabase/supabase/%s/apps/docs/spec/api_v1_openapi.json"
)

// Supabase adapts the Supabase Management API spec in supabase/supabase for
// the openapi plugin. The repo tags no spec releases, so a tag here is the
// commit that last changed the spec.
type Supabase struct{}

var _ adapter.Adapter = Supabase{}

func (Supabase) API() string { return "supabase/management-api" }

// LatestTag is the SHA of the newest commit that changed the spec.
func (Supabase) LatestTag(ctx context.Context) (string, error) {
	sha, err := github.LatestCommit(ctx, github.Get, supabaseCommitsURL)
	if err != nil {
		return "", fmt.Errorf("looking up supabase's latest spec commit: %w", err)
	}
	return sha, nil
}

// Fetch downloads the spec at commit sha.
//
// Version is the commit's date (YYYY-MM-DD): the Management API has no
// version of its own (`/v1`, info.version a fixed "1.0.0", checked
// 2026-11-19), so the date the spec changed is the closest thing to "the API
// version the change arrived in".
func (Supabase) Fetch(ctx context.Context, sha string) (adapter.Spec, error) {
	fullSHA, date, err := github.Commit(ctx, github.Get, fmt.Sprintf(supabaseCommitURL, sha))
	if err != nil {
		return adapter.Spec{}, fmt.Errorf("looking up supabase spec commit %s: %w", sha, err)
	}
	if fullSHA == "" {
		return adapter.Spec{}, fmt.Errorf("github commit %s has no sha or date", sha)
	}
	// Pin to the full SHA even when given a short one: the citation must
	// name exactly one document.
	url := fmt.Sprintf(supabaseSpecURLTemplate, fullSHA)
	body, err := fetch.Get(ctx, url, nil) // raw.githubusercontent.com needs no auth
	if err != nil {
		return adapter.Spec{}, fmt.Errorf("fetching supabase spec %s: %w", sha, err)
	}
	if !json.Valid(body) {
		return adapter.Spec{}, fmt.Errorf("supabase spec %s is not valid json", sha)
	}
	return adapter.Spec{Content: body, Version: date.UTC().Format("2006-01-02"), SourceURL: url}, nil
}

// Versions are the commits that changed the Management API spec.
func (Supabase) Versions(ctx context.Context) ([]plugin.Version, error) {
	return adapter.Commits(ctx, "supabase/supabase", "apps/docs/spec/api_v1_openapi.json")
}

// supabaseDiff is how the Supabase Management API spec says what it says:
//
//   - Deprecations use the `deprecated` flag, and the description sometimes
//     says why ("This endpoint is deprecated - use the deploy endpoint."), so
//     the event quotes the whole operation, not the bare `true`.
//   - Path parameters get renamed, and that isn't a change for callers:
//     /v1/branches/{branch_id}/merge became .../{branch_id_or_ref}/merge on
//     2025-09-11.
var supabaseDiff = diff.Options{PathTemplates: true, QuoteDeprecatedOperation: true}

// supabaseKeep keeps operation deprecations, and removals of operations the
// old spec had marked deprecated. In the spec's history (90 commits,
// 2024-05-31 to 2026-11-18) that's 2 events: POST /v1/projects/{ref}/functions
// deprecated on 2025-02-19, GET /v1/projects/{ref}/analytics/endpoints/logs.all
// on 2026-06-29. Everything else is dropped:
//
//   - 22 operations left the spec, none of them deprecated first: 14
//     analytics warehouse operations on 2024-07-15, 7 path-parameter renames
//     (which PathTemplates pairs up anyway), and one that came back 12
//     days later. Without credentials every route answers 401, so whether a
//     removed one still works can't be checked from outside.
//   - Request and response property changes, new operations: not changes a
//     caller has to fix, or not ones the spec's history shows reliably.
var supabaseKeep = openapi.DeprecationsOnly(supabaseDiff)
