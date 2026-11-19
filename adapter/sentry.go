package adapter

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/fetch"
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

var _ Adapter = Sentry{}

func (Sentry) API() string { return "sentry" }

func (s Sentry) api(path string) string {
	return cmp.Or(s.APIBase, "https://api.github.com") + path
}

// LatestTag is the highest api@X.Y.Z release of getsentry/sentry that has the
// spec attached, drafts and prereleases left out.
//
// The releases listing is newest first by date, and getsentry/sentry
// releases other things too (the server, "26.11.0"; npm packages,
// "scraps@0.5.0"). The api@ releases' dates aren't in version order
// (api@2.9.0 was published after api@2.12.0), so the newest is computed.
func (s Sentry) LatestTag(ctx context.Context) (string, error) {
	body, err := fetch.Get(ctx, s.api("/repos/getsentry/sentry/releases?per_page=100"), githubHeaders())
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
// server URL's path into the paths: see sentryBasePath.
//
// Version is the release's publication date, or a commit's date
// (YYYY-MM-DD): the API has no version of its own (`/api/0`, info.version
// "v0"), and the spec's own version numbers are a package's (@sentry/api),
// not the API's.
func (s Sentry) Fetch(ctx context.Context, tag string) (Spec, error) {
	raw := cmp.Or(s.RawBase, "https://raw.githubusercontent.com") + "/getsentry/sentry-api-schema/" + tag + "/" + sentryAsset
	var url string
	var date time.Time
	switch {
	case sentryRelease.MatchString(tag):
		r, err := sentryReleaseMeta(ctx, s.api("/repos/getsentry/sentry/releases/tags/"+tag), tag)
		if err != nil {
			return Spec{}, err
		}
		if !r.hasSpec() {
			return Spec{}, fmt.Errorf("sentry release %s has no %s", tag, sentryAsset)
		}
		// Spelled as GitHub's own links spell it: the tag's @ as %40.
		url = cmp.Or(s.DownloadBase, "https://github.com") + "/getsentry/sentry/releases/download/" +
			strings.ReplaceAll(tag, "@", "%40") + "/" + sentryAsset
		date = r.PublishedAt
	case sentrySchemaTag.MatchString(tag):
		r, err := sentryReleaseMeta(ctx, s.api("/repos/getsentry/sentry-api-schema/releases/tags/"+tag), tag)
		if err != nil {
			return Spec{}, err
		}
		url, date = raw, r.PublishedAt
	case sentrySchemaCommit.MatchString(tag):
		meta, err := fetch.Get(ctx, s.api("/repos/getsentry/sentry-api-schema/commits/"+tag), githubHeaders())
		if err != nil {
			return Spec{}, fmt.Errorf("looking up sentry-api-schema commit %s: %w", tag, err)
		}
		var c struct {
			Commit struct {
				Committer struct {
					Date time.Time `json:"date"`
				} `json:"committer"`
			} `json:"commit"`
		}
		if err := json.Unmarshal(meta, &c); err != nil {
			return Spec{}, fmt.Errorf("parsing github commit %s: %w", tag, err)
		}
		url, date = raw, c.Commit.Committer.Date
	default:
		return Spec{}, fmt.Errorf("%q is neither a sentry api@X.Y.Z release, a sentry-api-schema tag nor a full commit sha", tag)
	}
	if date.IsZero() {
		return Spec{}, fmt.Errorf("sentry %s: github gives no date", tag)
	}
	body, err := fetch.Get(ctx, url, nil) // public downloads need no auth
	if err != nil {
		return Spec{}, fmt.Errorf("fetching the sentry %s spec: %w", tag, err)
	}
	content, err := sentryBasePath(body)
	if err != nil {
		return Spec{}, fmt.Errorf("sentry %s: %w", tag, err)
	}
	return Spec{Content: content, Version: date.UTC().Format("2006-01-02"), SourceURL: url}, nil
}

func sentryReleaseMeta(ctx context.Context, url, tag string) (sentryReleaseInfo, error) {
	meta, err := fetch.Get(ctx, url, githubHeaders())
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

// sentryBasePathKey is the root key Fetch records the server path it moved
// into the paths under. The differ reads only paths and components, so the
// key is never diffed.
const sentryBasePathKey = "x-patch-it-base-path"

// sentryBasePath puts the server URL's path back at the front of every path.
//
// Until sentry-api-schema 2.0.1 the server was "https://{region}.sentry.io"
// and the paths started "/api/0/"; from 2.1.0 (2026-11-13) the server is
// "https://{region}.sentry.io/api/0" and the paths don't. The URLs a caller
// sends are the same, but compared as published, every operation (289 of
// them) reads as removed and added again. With the server's path moved into
// the paths, every spec names an operation by the path a caller sends,
// "GET /api/0/organizations/", as Sentry's API reference does.
//
// A spec whose server has no path is returned byte for byte. The values are
// otherwise unchanged (numbers kept as written); the path moved is recorded
// for SourcePointer.
func sentryBasePath(body []byte) ([]byte, error) {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	var doc map[string]any
	if err := dec.Decode(&doc); err != nil {
		return nil, fmt.Errorf("parsing the spec: %w", err)
	}
	base := ""
	for i, s := range asSlice(doc["servers"]) {
		u, _ := asMap(s)["url"].(string)
		p := serverPath(u)
		if i > 0 && p != base {
			return nil, fmt.Errorf("servers with different paths (%q, %q)", base, p)
		}
		base = p
	}
	if base == "" {
		return body, nil
	}
	paths := asMap(doc["paths"])
	moved := make(map[string]any, len(paths))
	for p, item := range paths {
		if !strings.HasPrefix(p, "/") {
			return nil, fmt.Errorf("path %q doesn't start with /", p)
		}
		moved[base+p] = item
	}
	doc["paths"] = moved
	doc[sentryBasePathKey] = base
	return json.Marshal(doc)
}

// serverPath is the path of a server URL ("/api/0" for
// "https://{region}.sentry.io/api/0"), without a trailing slash; "" for
// none. Read by hand: url.Parse refuses the {region} variable in the host.
func serverPath(u string) string {
	_, rest, ok := strings.Cut(u, "://")
	if !ok {
		return ""
	}
	i := strings.Index(rest, "/")
	if i < 0 {
		return ""
	}
	return strings.TrimRight(rest[i:], "/")
}

// SourcePointer turns a pointer into a path Fetch prefixed with the server's
// path back into one into the path as published. Every other pointer, and
// every pointer into a spec Fetch didn't change, is already right.
func (Sentry) SourcePointer(content []byte) func(string) string {
	var doc struct {
		Base string `json:"x-patch-it-base-path"`
	}
	_ = json.Unmarshal(content, &doc) // unreadable content: pointers stay as they are
	if doc.Base == "" {
		return func(ptr string) string { return ptr }
	}
	prefix := "/paths/" + pointerToken(doc.Base)
	return func(ptr string) string {
		rest, ok := strings.CutPrefix(ptr, prefix)
		// Every path starts with /, so what follows is "~1..."; anything
		// else is a pointer into a path that wasn't moved.
		if !ok || !strings.HasPrefix(rest, "~1") {
			return ptr
		}
		return "/paths/" + rest
	}
}

var _ Reshaper = Sentry{}

// asSlice is v as an array, or nil.
func asSlice(v any) []any {
	s, _ := v.([]any)
	return s
}
