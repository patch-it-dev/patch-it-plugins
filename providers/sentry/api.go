package sentry

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/core/diff"
	"github.com/patch-it-dev/patch-it-plugins/core/fetch"
	"github.com/patch-it-dev/patch-it-plugins/core/github"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/core/versions"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi/adapter"
)

// sentryAsset is the spec's file name in both places it's published:
// Sentry's public API spec with every $ref resolved.
const sentryAsset = "openapi-derefed.json"

var (
	// sentryRelease is a getsentry/sentry API spec release tag.
	sentryRelease = regexp.MustCompile(`^api@(\d+)\.(\d+)\.(\d+)$`)
	// sentrySchemaTag is a getsentry/sentry-api-schema release tag.
	sentrySchemaTag = regexp.MustCompile(`^\d+\.\d+\.\d+$`)
	// sentrySchemaCommit is a full getsentry/sentry-api-schema commit SHA.
	sentrySchemaCommit = regexp.MustCompile(`^[0-9a-f]{40}$`)
)

// Sentry adapts Sentry's public API spec for the openapi plugin. A tag is a
// getsentry/sentry release, "api@2.16.0" (where the spec is published now),
// or, to replay the history, a getsentry/sentry-api-schema release tag,
// "0.224.0", or commit SHA.
//
//   - getsentry/sentry publishes the spec as an asset of each api@X.Y.Z
//     release, from api@2.5.0 (2026-11-14) on.
//   - getsentry/sentry-api-schema published it before that, as
//     openapi-derefed.json committed to the repo: commits from 2020-08-25,
//     release tags 0.1.0 to 2.4.0 from 2026-02-10. The repo is archived;
//     its 2.4.0 is byte for byte api@2.5.0.
//
// The zero value reads GitHub. The fields point it at another server, for
// tests.
type Sentry struct {
	APIBase      string // GitHub's REST API; "" is https://api.github.com
	DownloadBase string // release downloads; "" is https://github.com
	RawBase      string // files at a tag or commit; "" is https://raw.githubusercontent.com
}

var _ adapter.Adapter = Sentry{}

func (Sentry) API() string { return "sentry/api" }

func (s Sentry) api(path string) string {
	return cmp.Or(s.APIBase, "https://api.github.com") + path
}

// LatestTag is the highest api@X.Y.Z release of getsentry/sentry that has the
// spec attached, drafts and prereleases left out.
//
// The releases listing is newest first by date, and getsentry/sentry
// releases other things too (the server, "26.11.0"; npm packages,
// "scraps@0.5.0"). GitHub lists releases in an order that isn't version
// order, so the newest is computed from the version.
func (s Sentry) LatestTag(ctx context.Context) (string, error) {
	body, err := fetch.Get(ctx, s.api("/repos/getsentry/sentry/releases?per_page=100"), github.Headers())
	if err != nil {
		return "", fmt.Errorf("listing sentry releases: %w", err)
	}
	var releases []sentryReleaseInfo
	if err := json.Unmarshal(body, &releases); err != nil {
		return "", fmt.Errorf("parsing github releases: %w", err)
	}
	type release struct {
		tag string
		v   [3]int
	}
	var found []release
	for _, r := range releases {
		m := sentryRelease.FindStringSubmatch(r.Tag)
		if m == nil || r.Draft || r.Prerelease || !r.hasSpec() {
			continue
		}
		rel := release{tag: r.Tag}
		for i := range rel.v {
			rel.v[i], _ = strconv.Atoi(m[i+1])
		}
		found = append(found, rel)
	}
	if len(found) == 0 {
		return "", fmt.Errorf("github lists no sentry api@ release with %s", sentryAsset)
	}
	newest := slices.MaxFunc(found, func(a, b release) int {
		return cmp.Or(cmp.Compare(a.v[0], b.v[0]), cmp.Compare(a.v[1], b.v[1]), cmp.Compare(a.v[2], b.v[2]))
	})
	return newest.tag, nil
}

type sentryReleaseInfo struct {
	Tag         string    `json:"tag_name"`
	Draft       bool      `json:"draft"`
	Prerelease  bool      `json:"prerelease"`
	PublishedAt time.Time `json:"published_at"`
	Assets      []struct {
		Name string `json:"name"`
	} `json:"assets"`
}

func (r sentryReleaseInfo) hasSpec() bool {
	for _, a := range r.Assets {
		if a.Name == sentryAsset {
			return true
		}
	}
	return false
}

// Fetch downloads the spec of an api@X.Y.Z release (the release's asset), or
// of a sentry-api-schema tag or commit (the file at it), and moves the
// server URL's path into the paths: see ServerBasePath.
//
// Version is the release's publication date, or a commit's date
// (YYYY-MM-DD): the API has no version of its own (`/api/0`, info.version
// "v0"), and the spec's own version numbers are a package's (@sentry/api),
// not the API's.
func (s Sentry) Fetch(ctx context.Context, tag string) (adapter.Spec, error) {
	raw := cmp.Or(s.RawBase, "https://raw.githubusercontent.com") + "/getsentry/sentry-api-schema/" + tag + "/" + sentryAsset
	var url string
	var date time.Time
	switch {
	case sentryRelease.MatchString(tag):
		r, err := sentryReleaseMeta(ctx, s.api("/repos/getsentry/sentry/releases/tags/"+tag), tag)
		if err != nil {
			return adapter.Spec{}, err
		}
		if !r.hasSpec() {
			return adapter.Spec{}, fmt.Errorf("sentry release %s has no %s", tag, sentryAsset)
		}
		// Spelled as GitHub's own links spell it: the tag's @ as %40.
		url = cmp.Or(s.DownloadBase, "https://github.com") + "/getsentry/sentry/releases/download/" +
			strings.ReplaceAll(tag, "@", "%40") + "/" + sentryAsset
		date = r.PublishedAt
	case sentrySchemaTag.MatchString(tag):
		r, err := sentryReleaseMeta(ctx, s.api("/repos/getsentry/sentry-api-schema/releases/tags/"+tag), tag)
		if err != nil {
			return adapter.Spec{}, err
		}
		url, date = raw, r.PublishedAt
	case sentrySchemaCommit.MatchString(tag):
		meta, err := fetch.Get(ctx, s.api("/repos/getsentry/sentry-api-schema/commits/"+tag), github.Headers())
		if err != nil {
			return adapter.Spec{}, fmt.Errorf("looking up sentry-api-schema commit %s: %w", tag, err)
		}
		var c struct {
			Commit struct {
				Committer struct {
					Date time.Time `json:"date"`
				} `json:"committer"`
			} `json:"commit"`
		}
		if err := json.Unmarshal(meta, &c); err != nil {
			return adapter.Spec{}, fmt.Errorf("parsing github commit %s: %w", tag, err)
		}
		url, date = raw, c.Commit.Committer.Date
	default:
		return adapter.Spec{}, fmt.Errorf("%q is neither a sentry api@X.Y.Z release, a sentry-api-schema tag nor a full commit sha", tag)
	}
	if date.IsZero() {
		return adapter.Spec{}, fmt.Errorf("sentry %s: github gives no date", tag)
	}
	body, err := fetch.Get(ctx, url, nil) // public downloads need no auth
	if err != nil {
		return adapter.Spec{}, fmt.Errorf("fetching the sentry %s spec: %w", tag, err)
	}
	content, err := adapter.ServerBasePath(body)
	if err != nil {
		return adapter.Spec{}, fmt.Errorf("sentry %s: %w", tag, err)
	}
	return adapter.Spec{Content: content, Version: date.UTC().Format("2006-01-02"), SourceURL: url}, nil
}

func sentryReleaseMeta(ctx context.Context, url, tag string) (sentryReleaseInfo, error) {
	meta, err := fetch.Get(ctx, url, github.Headers())
	if err != nil {
		return sentryReleaseInfo{}, fmt.Errorf("looking up sentry release %s: %w", tag, err)
	}
	var r sentryReleaseInfo
	if err := json.Unmarshal(meta, &r); err != nil {
		return sentryReleaseInfo{}, fmt.Errorf("parsing github release %s: %w", tag, err)
	}
	if r.Draft || r.Prerelease {
		return sentryReleaseInfo{}, fmt.Errorf("sentry %s is a draft or a prerelease", tag)
	}
	return r, nil
}

// SourcePointer: see BasePathPointer.
func (Sentry) SourcePointer(content []byte) func(string) string {
	return adapter.BasePathPointer(content)
}

var _ adapter.Reshaper = Sentry{}

// Versions are every release of the spec, oldest first by date: the
// getsentry/sentry-api-schema release tags (0.1.0 to 2.4.0, the archived
// repo) and then getsentry/sentry's api@X.Y.Z releases with the spec
// attached. The ID is the release's date, as Fetch's Version. The schema
// repo's commits before its first tag aren't listed; Fetch still reads one
// by its SHA.
func (s Sentry) Versions(ctx context.Context) ([]plugin.Version, error) {
	var out []plugin.Version
	for _, src := range []struct {
		repo string
		keep func(sentryReleaseInfo) bool
	}{
		{"getsentry/sentry-api-schema", func(r sentryReleaseInfo) bool { return sentrySchemaTag.MatchString(r.Tag) }},
		{"getsentry/sentry", func(r sentryReleaseInfo) bool { return sentryRelease.MatchString(r.Tag) && r.hasSpec() }},
	} {
		rs, err := versions.GitHubPages[sentryReleaseInfo](ctx, github.Get, s.api("/repos/"+src.repo+"/releases"))
		if err != nil {
			return nil, fmt.Errorf("listing %s's releases: %w", src.repo, err)
		}
		for _, r := range rs {
			if r.Draft || r.Prerelease || !src.keep(r) {
				continue
			}
			at := r.PublishedAt.UTC()
			out = append(out, plugin.Version{ID: at.Format(time.DateOnly), Released: at, Ref: r.Tag,
				URL: "https://github.com/" + src.repo + "/releases/tag/" + strings.ReplaceAll(r.Tag, "@", "%40")})
		}
	}
	slices.SortStableFunc(out, func(a, b plugin.Version) int { return a.Released.Compare(b.Released) })
	return out, nil
}

// sentryDiff is how Sentry's spec says what it says:
//
//   - Deprecations are mostly in the operation's description, not the flag.
//     Over the spec's history (the 1,130 commits that changed it in
//     getsentry/sentry-api-schema, 2020-08-25 to 2026-11-14, then the 19
//     api@ releases to api@2.16.0) the flag deprecates 3 operations and the
//     description 13, those 3 included, never later than the flag: the 10
//     issue and metric alert-rule operations on 2026-01-22, never flagged;
//     POST .../user-feedback/ on 2024-07-31 and GET .../issues/ on
//     2025-07-25, both flagged on 2026-06-16; GET .../stats/, words and flag
//     together on 2022-03-01.
//   - The flag, when it's set, comes with the reason in the description, so
//     a flagged operation is quoted whole.
//   - Path parameters get renamed ({organization_slug} became
//     {organization_id_or_slug} on 2024-05-10), and that isn't a change for
//     callers.
var sentryDiff = diff.Options{PathTemplates: true, QuoteDeprecatedOperation: true, DeprecatedDescription: sentryDeprecatedDescription}

// sentryDeprecatedWords are the three ways the descriptions say it, over the
// whole history: a "## Deprecated" heading with nothing else on its line, a
// line starting "**Deprecated**:", and "This endpoint is deprecated" in any
// case ("*This endpoint is DEPRECATED."). Also "**Deprecated:**", the colon
// inside the bold: never used for a deprecation yet, but how Sentry writes
// its other notes ("**Experimental:**", 45 times in api@2.16.0). Not a heading
// with more words ("## Deprecated fields"), which would be about part of the
// operation. Not "deprecat" anywhere: that also matches
// three descriptions that aren't deprecations, GET .../events/'s "**Eventsv2
// Deprecation Note**" (about another endpoint; 2022-06-16, and again under
// the 2024-05-10 rename) and GET .../trace-items/attributes/'s "examples, and
// deprecation details" (2026-10-15).
var sentryDeprecatedWords = regexp.MustCompile(`(?im)^\s*#+\s*deprecated[^\p{L}\p{N}\n]*$|^\s*\*\*deprecated(\*\*:|:\*\*)|\bthis endpoint is deprecated\b`)

func sentryDeprecatedDescription(description string) bool {
	return sentryDeprecatedWords.MatchString(description)
}

// sentryKeep keeps operation deprecations only: not even a deprecated
// operation's removal. An operation leaving Sentry's spec means Sentry
// stopped documenting it, and the history doesn't show when, or whether, it
// stopped working.
//
//   - 157 operations left the spec (by path template, with /api/0 in every
//     path: see Sentry). 11 had been deprecated first: GET
//     /api/0/organizations/{organization_slug}/stats/, gone on 2024-01-22
//     and still routed in getsentry/sentry's src/sentry/api/urls.py on
//     2026-11-19, and the 10 alert-rule operations, gone on 2026-05-08 (the
//     day after Sentry's "Monitors & Alerts — Now Generally Available"),
//     with nothing from Sentry saying they stopped answering then.
//   - Compared as published, 289 more left on 2026-11-13, when /api/0
//     moved into the server URL, GET .../issues/ and POST .../user-feedback/
//     among them, both flagged deprecated.
//
// Property changes and new operations are dropped too, as for vercel.
func sentryKeep(c diff.Change) bool { return c.Kind == diff.OperationDeprecated }
