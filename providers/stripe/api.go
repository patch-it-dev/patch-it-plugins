package stripe

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/core/fetch"
	"github.com/patch-it-dev/patch-it-plugins/core/github"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/core/versions"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi/adapter"
)

// Variables, not constants, only so tests can point them at a local server.
var (
	stripeReleasesURL = "https://api.github.com/repos/stripe/openapi/releases/latest"
	// stripeSpecURLTemplate is Stripe's recommended file: v1 + v2 endpoints.
	// Every release that names its API version (Versions) has it.
	stripeSpecURLTemplate = "https://raw.githubusercontent.com/stripe/openapi/%s/latest/openapi.spec3.json"
)

// Stripe is the reference adapter: the pattern every future adapter follows.
// It has no state, so its zero value is ready to use.
type Stripe struct{}

// Compile-time checks: Stripe is an adapter whose API versions are
// re-released.
var (
	_ adapter.Adapter    = Stripe{}
	_ adapter.Rereleased = Stripe{}
)

func (Stripe) API() string { return "stripe/api" }

// LatestTag asks GitHub for Stripe's newest openapi release tag, e.g. "v2468".
func (Stripe) LatestTag(ctx context.Context) (string, error) {
	tag, err := github.LatestRelease(ctx, github.Get, stripeReleasesURL)
	if err != nil {
		return "", fmt.Errorf("looking up stripe's latest release: %w", err)
	}
	return tag, nil
}

// Fetch downloads the spec pinned to tag and reads its declared API version
// out of info.version — the only version field the spec has.
func (Stripe) Fetch(ctx context.Context, tag string) (adapter.Spec, error) {
	url := fmt.Sprintf(stripeSpecURLTemplate, tag)
	body, err := fetch.Get(ctx, url, nil) // raw.githubusercontent.com needs no auth
	if err != nil {
		return adapter.Spec{}, fmt.Errorf("fetching stripe spec %s: %w", tag, err)
	}

	var parsed struct {
		Info struct {
			Version string `json:"version"`
		} `json:"info"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return adapter.Spec{}, fmt.Errorf("parsing stripe spec %s: %w", tag, err)
	}
	if parsed.Info.Version == "" {
		return adapter.Spec{}, fmt.Errorf("stripe spec %s has no info.version", tag)
	}

	return adapter.Spec{
		Content:   body,
		Version:   parsed.Info.Version,
		SourceURL: url,
	}, nil
}

// stripeSDKSpecVersion is the API version a stripe/openapi release's notes
// say its SDK spec carries, the version Stripe's SDKs pin (every release
// since v2191, 2026-03-11, checked 2026-11-20):
//
//	| openapi/spec3.sdk.{json,yaml} | 2026-11-18.endive |
var stripeSDKSpecVersion = regexp.MustCompile(`(?m)^\|\s*openapi/spec3\.sdk\.\{json,yaml\}\s*\|\s*(\d{4}-\d{2}-\d{2}(?:\.[a-z]+)?)\s*\|`)

// Versions are Stripe's API versions ("2026-11-18.endive"), oldest first, as
// stripe/openapi's releases name them in their notes: each is read at the
// newest release that carries it (Ref, URL) and dated by the first
// (Released). A release whose notes name no version (every one before v2191)
// is left out: its version would have to be guessed.
func (Stripe) Versions(ctx context.Context) ([]plugin.Version, error) {
	byID, err := apiVersions(ctx)
	if err != nil {
		return nil, err
	}
	vs := make([]plugin.Version, 0, len(byID))
	for id, rs := range byID {
		first, last := rs[0], rs[len(rs)-1]
		vs = append(vs, plugin.Version{ID: id, Released: first.Published, Ref: last.Tag, URL: last.URL})
	}
	slices.SortStableFunc(vs, func(a, b plugin.Version) int { return versions.Compare(a.ID, b.ID) })
	return vs, nil
}

// Releases are the releases whose notes name v, oldest first: Stripe keeps
// adding to an API version in later releases, and what they add is filed
// under v alone (adapter.Rereleased). The first one or two may not carry v
// in the full spec yet (their notes name the SDK spec's version).
func (Stripe) Releases(ctx context.Context, v plugin.Version) ([]string, error) {
	byID, err := apiVersions(ctx)
	if err != nil {
		return nil, err
	}
	var tags []string
	for _, r := range byID[v.ID] {
		tags = append(tags, r.Tag)
	}
	return tags, nil
}

// apiVersions are stripe/openapi's releases by the API version their notes
// name, oldest release first; drafts, prereleases and releases that name
// none are left out.
func apiVersions(ctx context.Context) (map[string][]stripeRelease, error) {
	rs, err := versions.GitHubPages[stripeRelease](ctx, github.Get, versions.GitHubAPI+"/repos/stripe/openapi/releases")
	if err != nil {
		return nil, fmt.Errorf("listing stripe/openapi's releases: %w", err)
	}
	slices.SortStableFunc(rs, func(a, b stripeRelease) int { return a.Published.Compare(b.Published) })
	byID := map[string][]stripeRelease{}
	for _, r := range rs {
		m := stripeSDKSpecVersion.FindStringSubmatch(r.Body)
		if r.Draft || r.Prerelease || r.Tag == "" || m == nil {
			continue
		}
		byID[m[1]] = append(byID[m[1]], r)
	}
	return byID, nil
}

// stripeRelease is a stripe/openapi release as GitHub's API lists it.
type stripeRelease struct {
	Tag        string    `json:"tag_name"`
	Draft      bool      `json:"draft"`
	Prerelease bool      `json:"prerelease"`
	Published  time.Time `json:"published_at"`
	URL        string    `json:"html_url"`
	Body       string    `json:"body"`
}

// ListsSpecVersions: Stripe's version ids are the specs' info.version.
func (Stripe) ListsSpecVersions() {}
