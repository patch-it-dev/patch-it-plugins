package planetscale

import (
	"cmp"
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
	"gopkg.in/yaml.v3"
)

const (
	// planetscaleRepo is PlanetScale's Terraform provider, which keeps a copy
	// of the PlanetScale API spec at planetscaleSpecPath.
	planetscaleRepo = "planetscale/terraform-provider-planetscale"
	// planetscaleSpecPath is the spec in that repo: a copy of
	// https://planetscale.com/docs/openapi.yaml, which has no versions,
	// committed when it changes.
	planetscaleSpecPath = "schemas/openapi.yaml"
)

// PlanetScale adapts the PlanetScale API spec for the openapi plugin. The
// repo tags no spec releases, so a tag here is the commit that last changed
// the spec, as for Supabase.
//
// The zero value reads GitHub. The fields point it at another server, for
// tests.
type PlanetScale struct {
	APIBase string // GitHub's REST API; "" is https://api.github.com
	RawBase string // files at a commit; "" is https://raw.githubusercontent.com
}

var _ adapter.Adapter = PlanetScale{}

func (PlanetScale) API() string { return "planetscale/api" }

func (p PlanetScale) api(path string) string {
	return cmp.Or(p.APIBase, "https://api.github.com") + "/repos/" + planetscaleRepo + path
}

// LatestTag is the SHA of the newest commit that changed the spec.
func (p PlanetScale) LatestTag(ctx context.Context) (string, error) {
	sha, err := github.LatestCommit(ctx, github.Get, p.api("/commits?path="+planetscaleSpecPath+"&per_page=1"))
	if err != nil {
		return "", fmt.Errorf("looking up planetscale's latest spec commit: %w", err)
	}
	return sha, nil
}

// Fetch downloads the spec at commit sha and converts it from YAML to JSON,
// which is what diff.Parse reads (as for Algolia).
//
// Version is the commit's date (YYYY-MM-DD): the API has no version of its
// own (`info.version` is a fixed "v1", checked 2026-11-19), so the date the
// spec changed is the closest thing to "the API version the change arrived
// in".
//
// A file that isn't a spec is an error, not an empty API: diffed against the
// last one, it would read as every operation removed. The commits that
// changed the file include one that deleted it (f33a0c3b35, 2025-12-04;
// it came back the same day), where the raw URL answers 404: fetch.Get's
// error. A file served 200 that has no paths (emptied, or a stub page like
// "404: Not Found", which YAML reads as a one-key mapping) is refused here.
func (p PlanetScale) Fetch(ctx context.Context, sha string) (adapter.Spec, error) {
	fullSHA, date, err := github.Commit(ctx, github.Get, p.api("/commits/"+sha))
	if err != nil {
		return adapter.Spec{}, fmt.Errorf("looking up planetscale spec commit %s: %w", sha, err)
	}
	if fullSHA == "" {
		return adapter.Spec{}, fmt.Errorf("github commit %s has no sha or date", sha)
	}
	// Pin to the full SHA even when given a short one: the citation must
	// name exactly one document.
	url := cmp.Or(p.RawBase, "https://raw.githubusercontent.com") + "/" + planetscaleRepo + "/" + fullSHA + "/" + planetscaleSpecPath
	body, err := fetch.Get(ctx, url, nil) // raw.githubusercontent.com needs no auth
	if err != nil {
		return adapter.Spec{}, fmt.Errorf("fetching planetscale spec %s: %w", sha, err)
	}
	var doc map[string]any
	if err := yaml.Unmarshal(body, &doc); err != nil {
		return adapter.Spec{}, fmt.Errorf("parsing planetscale spec %s as yaml: %w", sha, err)
	}
	if paths, _ := doc["paths"].(map[string]any); len(paths) == 0 {
		return adapter.Spec{}, fmt.Errorf("planetscale spec %s has no paths: not a spec", sha)
	}
	content, err := json.Marshal(doc)
	if err != nil {
		return adapter.Spec{}, fmt.Errorf("converting planetscale spec %s to json: %w", sha, err)
	}
	// The server is "https://api.planetscale.com/v1" and the paths leave out
	// /v1 (all 32 versions, checked 2026-11-19). Moved into the paths, an
	// operation is named by the path a caller sends, and a spec that ever
	// moved it (as Sentry's did) can't read as every deprecated operation
	// removed.
	if content, err = adapter.ServerBasePath(content); err != nil {
		return adapter.Spec{}, fmt.Errorf("planetscale spec %s: %w", sha, err)
	}
	return adapter.Spec{Content: content, Version: date.UTC().Format("2006-01-02"), SourceURL: url}, nil
}

// SourcePointer: see BasePathPointer.
func (PlanetScale) SourcePointer(content []byte) func(string) string {
	return adapter.BasePathPointer(content)
}

var _ adapter.Reshaper = PlanetScale{}

// Versions are the commits that changed PlanetScale's spec.
func (PlanetScale) Versions(ctx context.Context) ([]plugin.Version, error) {
	return adapter.Commits(ctx, planetscaleRepo, planetscaleSpecPath)
}

// planetscaleDiff is how the PlanetScale API spec says what it says:
//
//   - Deprecations are only in the description's first words, "Deprecated.
//     Use GET /organizations/:organization/databases/:database/branches/:branch/change-requests
//     instead.": the spec has never set the `deprecated` flag (none in any
//     of the 32 versions of schemas/openapi.yaml in
//     planetscale/terraform-provider-planetscale, 2025-11-11 to 2026-11-09,
//     nor in the live https://planetscale.com/docs/openapi.yaml on
//     2026-11-19). If it ever does, the operation is quoted whole, so the
//     words come with it.
//   - Path parameters get renamed, and that isn't a change for callers.
var planetscaleDiff = diff.Options{PathTemplates: true, QuoteDeprecatedOperation: true, DeprecatedDescription: planetscaleDeprecatedDescription}

// planetscaleDeprecatedWords is the way every deprecation in the history is
// written: the description starts "Deprecated." (the others start with what
// the operation does, or with an empty line, then "### Authorization"). Only
// at the start,
// so not a description that mentions deprecation further on; and with the
// period or a colon, so not "Deprecated, storage shrinking is part of
// storage autoscaling.", which the spec writes on properties.
var planetscaleDeprecatedWords = regexp.MustCompile(`(?i)^\s*deprecated[.:]`)

func planetscaleDeprecatedDescription(description string) bool {
	return planetscaleDeprecatedWords.MatchString(description)
}

// planetscaleKeep keeps operation deprecations, and removals of operations
// the old spec had marked deprecated. Over the 31 consecutive pairs of the
// spec's history that's 5 events, all on 2026-11-09 (fc13e09d): GET and
// DELETE .../bouncers/{bouncer}/resizes, GET .../neki-changes, GET and
// DELETE .../neki-changes/{id}. Everything else is dropped:
//
//   - 41 operations left the spec, none deprecated first. 6 of them, on
//     2026-10-03, were the read-only replica operations renamed to
//     dedicated read replicas, and planetscale/planetscale-go still sends
//     the old paths ("the deprecated read-only-replicas path", main on
//     2026-11-19): left the spec isn't stopped working.
//   - Property changes and new operations, as for vercel: 226 type changes
//     on 2025-12-06 alone, when the spec switched whole numbers to
//     `integer`.
var planetscaleKeep = openapi.DeprecationsOnly(planetscaleDiff)
