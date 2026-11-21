package neon

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
	// neonCommitsURL lists the commits that changed the Neon API spec in
	// neondatabase/neon-pkgs, newest first; with per_page=1, the newest.
	neonCommitsURL = "https://api.github.com/repos/neondatabase/neon-pkgs/commits?path=packages/sdk/spec/neon-openapi.json&per_page=1"
	// neonCommitURL is one commit, for its date.
	neonCommitURL = "https://api.github.com/repos/neondatabase/neon-pkgs/commits/%s"
	// neonSpecURLTemplate is the Neon API spec (console.neon.tech/api/v2) at
	// one commit. Neon serves it live at
	// https://neon.com/api_spec/release/v2.json, which has no versions; the
	// copy in neondatabase/neon-pkgs is pulled from there and committed when
	// a daily drift check sees it change (23 commits from 2026-06-24 to
	// 2026-11-17), and a commit never changes.
	neonSpecURLTemplate = "https://raw.githubusercontent.com/neondatabase/neon-pkgs/%s/packages/sdk/spec/neon-openapi.json"
)

// Neon adapts the Neon API spec in neondatabase/neon-pkgs for the openapi
// plugin. The repo tags no spec releases, so a tag here is the commit that
// last changed the spec.
type Neon struct{}

var _ adapter.Adapter = Neon{}

func (Neon) API() string { return "neon/api" }

// LatestTag is the SHA of the newest commit that changed the spec.
func (Neon) LatestTag(ctx context.Context) (string, error) {
	sha, err := github.LatestCommit(ctx, github.Get, neonCommitsURL)
	if err != nil {
		return "", fmt.Errorf("looking up neon's latest spec commit: %w", err)
	}
	return sha, nil
}

// Fetch downloads the spec at commit sha and moves the server URL's path into
// the paths: see ServerBasePath.
//
// Version is the commit's date (YYYY-MM-DD): the API's only version is the
// `/api/v2` in its URL (info.version a fixed "v2" in all 23 commits, checked
// 2026-11-19), so the date the spec changed is the closest thing to "the API
// version the change arrived in".
func (Neon) Fetch(ctx context.Context, sha string) (adapter.Spec, error) {
	fullSHA, date, err := github.Commit(ctx, github.Get, fmt.Sprintf(neonCommitURL, sha))
	if err != nil {
		return adapter.Spec{}, fmt.Errorf("looking up neon spec commit %s: %w", sha, err)
	}
	if fullSHA == "" {
		return adapter.Spec{}, fmt.Errorf("github commit %s has no sha or date", sha)
	}
	// Pin to the full SHA even when given a short one: the citation must
	// name exactly one document.
	url := fmt.Sprintf(neonSpecURLTemplate, fullSHA)
	body, err := fetch.Get(ctx, url, nil) // raw.githubusercontent.com needs no auth
	if err != nil {
		return adapter.Spec{}, fmt.Errorf("fetching neon spec %s: %w", sha, err)
	}
	if !json.Valid(body) {
		return adapter.Spec{}, fmt.Errorf("neon spec %s is not valid json", sha)
	}
	// The server is "https://console.neon.tech/api/v2" and the paths don't
	// repeat /api/v2 (all 23 commits, checked 2026-11-19). Moved into the
	// paths, an operation is named by the path a caller sends, and a spec
	// that ever moved it (as Sentry's did) can't read as every deprecated
	// operation removed.
	content, err := adapter.ServerBasePath(body)
	if err != nil {
		return adapter.Spec{}, fmt.Errorf("neon spec %s: %w", sha, err)
	}
	return adapter.Spec{Content: content, Version: date.UTC().Format("2006-01-02"), SourceURL: url}, nil
}

// SourcePointer: see BasePathPointer.
func (Neon) SourcePointer(content []byte) func(string) string {
	return adapter.BasePathPointer(content)
}

var _ adapter.Reshaper = Neon{}

// Versions are the commits that changed Neon's spec.
func (Neon) Versions(ctx context.Context) ([]plugin.Version, error) {
	return adapter.Commits(ctx, "neondatabase/neon-pkgs", "packages/sdk/spec/neon-openapi.json")
}

// neonDiff is how the Neon API spec says what it says:
//
//   - Deprecations use the `deprecated` flag, and only the flag: in the 23
//     commits of neondatabase/neon-pkgs's copy (2026-06-24 to 2026-11-17) no
//     operation is described as deprecated without it. (Two unflagged
//     operations deprecate a parameter in words; those aren't operation
//     deprecations.)
//     A flagged operation's description says why or when ("Deprecated: to
//     test a branch's already-saved configuration, use
//     `sendNeonAuthEmailProviderTest`", "Removal scheduled for March 1,
//     2026"), and every one carries an `x-sunset` date, so the event quotes
//     the whole operation, not the bare `true`.
//   - Path parameters could be renamed without the URL changing; none was in
//     this history, but matching by path template costs nothing here and
//     keeps a rename from reading as a removal.
var neonDiff = diff.Options{PathTemplates: true, QuoteDeprecatedOperation: true}

// neonKeep keeps operation deprecations, and removals of operations the old
// spec had marked deprecated. In the spec's history that's 1 event: POST
// /projects/{project_id}/branches/{branch_id}/auth/send_test_email,
// deprecated on 2026-08-23. Everything else is dropped:
//
//   - One operation left the spec, not deprecated first: POST
//     /projects/{project_id}/branches/{branch_id}/recover on 2026-08-09.
//     neon-pkgs's own PR for it (#408) says it "was not removed from the API.
//     It was removed from the published spec": it still routes.
//   - Request and response property changes, new operations: not changes a
//     caller has to fix, or not ones the spec's history shows reliably.
var neonKeep = openapi.DeprecationsOnly(neonDiff)
