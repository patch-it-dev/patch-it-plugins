package vercel

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/patch-it-dev/patch-it-plugins/core/diff"
	"github.com/patch-it-dev/patch-it-plugins/core/fetch"
	"github.com/patch-it-dev/patch-it-plugins/core/github"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi/adapter"
)

// Variables, not constants, only so tests can point them at a local server.
var (
	vercelReleasesURL = "https://api.github.com/repos/vercel/sdk/releases/latest"
	// vercelSpecURLTemplate is the REST API spec @vercel/sdk is generated from,
	// committed at the root of vercel/sdk and regenerated on every release.
	// Vercel also serves it live at https://openapi.vercel.sh/, but that URL
	// has no versions to pin or replay. The SDK repo's copy is the same spec,
	// a release behind at most (on 2026-11-18 the live one had 3 paths more
	// out of 356), and tagged on every release.
	vercelSpecURLTemplate = "https://raw.githubusercontent.com/vercel/sdk/%s/vercel-spec.json"
)

// Vercel adapts the OpenAPI spec bundled in vercel/sdk for the openapi plugin.
type Vercel struct{}

var _ adapter.Adapter = Vercel{}

func (Vercel) API() string { return "vercel/api" }

// LatestTag asks GitHub for vercel/sdk's newest release tag, e.g. "v1.28.65".
// The canary tags (v14.0.0-canary.9) aren't releases, so releases/latest never
// returns them (checked 2026-11-18).
func (Vercel) LatestTag(ctx context.Context) (string, error) {
	tag, err := github.LatestRelease(ctx, github.Get, vercelReleasesURL)
	if err != nil {
		return "", fmt.Errorf("looking up vercel's latest release: %w", err)
	}
	return tag, nil
}

// Fetch downloads the spec pinned to tag. It's JSON already, so it's passed
// through unchanged.
//
// Version is the release tag without its "v", not info.version: that is a
// fixed "0.0.1" in every release (checked 2026-11-18). The tag is what a
// customer pins in package.json ("@vercel/sdk": "1.28.65"). Vercel's REST API
// has no version of its own: each endpoint carries its own in the path
// (/v13/deployments, /v6/user/tokens).
func (Vercel) Fetch(ctx context.Context, tag string) (adapter.Spec, error) {
	url := fmt.Sprintf(vercelSpecURLTemplate, tag)
	body, err := fetch.Get(ctx, url, nil) // raw.githubusercontent.com needs no auth
	if err != nil {
		return adapter.Spec{}, fmt.Errorf("fetching vercel spec %s: %w", tag, err)
	}
	if !json.Valid(body) {
		return adapter.Spec{}, fmt.Errorf("vercel spec %s is not valid json", tag)
	}
	return adapter.Spec{Content: body, Version: strings.TrimPrefix(tag, "v"), SourceURL: url}, nil
}

// Versions are vercel/sdk's releases, canaries left out (as LatestTag does).
func (Vercel) Versions(ctx context.Context) ([]plugin.Version, error) {
	return adapter.Releases(ctx, "vercel/sdk", func(tag string) bool { return !strings.Contains(tag, "canary") })
}

// vercelDiff is how Vercel's spec says what it says:
//
//   - Most deprecations are in the summary, not the flag. Of the 22 operations
//     deprecated between 2024-11 and 2026-11 in vercel/sdk's spec history, 16
//     said so only in words: "Check the price for a domain (deprecated)",
//     "Deprecated: Use DELETE /v1/drives/:name. ...".
//   - Renaming a path parameter isn't a change for callers, and Vercel does it:
//     DELETE /v2/sandboxes/drives/{name} became .../{nameOrId} in v1.28.65.
var vercelDiff = diff.Options{PathTemplates: true, DeprecatedSummary: vercelDeprecatedSummary}

// vercelDeprecatedWords are the two ways every word-only deprecation in the
// history is written: a summary that starts "Deprecated:" or ends
// "(deprecated)". Not "deprecat" anywhere, nor a bare leading "Deprecated",
// which would match an operation about deprecated things ("Deprecated
// runtimes for a project").
var vercelDeprecatedWords = regexp.MustCompile(`(?i)^deprecated:|\(deprecated\)\s*$`)

func vercelDeprecatedSummary(summary string) bool {
	return vercelDeprecatedWords.MatchString(summary)
}

// vercelKeep keeps operation deprecations, and removals of operations the old
// spec had marked deprecated. Everything else is dropped, because in
// vercel/sdk's spec history (207 releases, 2024-11 to 2026-11) it was mostly
// not a change to the API:
//
//   - 108 operations left the spec, and only 5 had been marked deprecated: the
//     four Legacy Domains endpoints, which Vercel did shut down on 2025-11-09
//     (they now answer "This endpoint has been sunsetted"), and the drives
//     rename PathTemplates already pairs up. Of the other 103, 16 came back in
//     a later release, and the edge-config ones were the product's rename to
//     Global Config.
//   - The response side reported 67,048 removed properties, about a thousand a
//     release, from the generated schemas reshuffling.
//   - The request side reported 478 breaking property changes, most of them
//     union members compared by position as their order moved; outside unions
//     there were still 142, and 27 of those reverted later.
//
// Additions are dropped too: in this history a new operation was as often a
// moved or re-added one (16 of the 103 came back), and a new property the
// generated schemas reshuffling, so an "added" here isn't one (precision over
// recall).
func vercelKeep(c diff.Change) bool { return openapi.DeprecationsOnly(vercelDiff)(c) }
