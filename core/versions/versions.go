// Package versions lists a source's versions from where they're published:
// a GitHub repo's releases, tags or the commits that changed a file, and a
// package's releases on PyPI or npm. Every list comes back oldest first, as
// plugin.Version. (A living page has no versions to list: plugin.Living.)
//
// Each function takes how to fetch (a plugin's env.Get, or github.Get for the
// GitHub API with its token), so tests serve saved responses.
package versions

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
)

// Getter fetches one URL: plugin.Env.Get, or github.Get.
type Getter func(ctx context.Context, url string) ([]byte, error)

// The services' base URLs; variables only so a test can point them elsewhere.
var (
	GitHubAPI = "https://api.github.com"
	PyPI      = "https://pypi.org"
	NPM       = "https://registry.npmjs.org"
)

// MaxPages caps how many pages of 100 a GitHub list reads (1,000 entries).
const MaxPages = 10

// GitHubPages reads a GitHub list endpoint (base, with or without a query)
// page by page, 100 at a time, until a short page or MaxPages: for a list
// the functions here don't cover (a release's assets, another host's base).
func GitHubPages[T any](ctx context.Context, get Getter, base string) ([]T, error) {
	return pages[T](ctx, get, base)
}

// pages reads a GitHub list endpoint page by page until a short page.
func pages[T any](ctx context.Context, get Getter, base string) ([]T, error) {
	sep := "?"
	if strings.Contains(base, "?") {
		sep = "&"
	}
	var all []T
	for page := 1; page <= MaxPages; page++ {
		body, err := get(ctx, fmt.Sprintf("%s%sper_page=100&page=%d", base, sep, page))
		if err != nil {
			return nil, err
		}
		var got []T
		if err := json.Unmarshal(body, &got); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", base, err)
		}
		all = append(all, got...)
		if len(got) < 100 {
			break
		}
	}
	return all, nil
}

// GitHubReleases are a repo's published releases ("owner/name"), drafts and
// prereleases left out, oldest first by publication date. keep, when set,
// keeps only the tags it accepts.
func GitHubReleases(ctx context.Context, get Getter, repo string, keep func(tag string) bool) ([]plugin.Version, error) {
	rs, err := pages[struct {
		Tag        string    `json:"tag_name"`
		Draft      bool      `json:"draft"`
		Prerelease bool      `json:"prerelease"`
		Published  time.Time `json:"published_at"`
		URL        string    `json:"html_url"`
	}](ctx, get, GitHubAPI+"/repos/"+repo+"/releases")
	if err != nil {
		return nil, fmt.Errorf("listing %s's releases: %w", repo, err)
	}
	var out []plugin.Version
	for _, r := range rs {
		if r.Draft || r.Prerelease || r.Tag == "" || (keep != nil && !keep(r.Tag)) {
			continue
		}
		out = append(out, plugin.Version{ID: r.Tag, Released: r.Published, Ref: r.Tag, URL: r.URL})
	}
	slices.SortStableFunc(out, func(a, b plugin.Version) int { return a.Released.Compare(b.Released) })
	return out, nil
}

// GitHubTags are a repo's tags that match pattern; its first submatch, when
// it has one, is the version's ID ("^v(\d+\.\d+\.\d+)$" makes "v1.2.3" 1.2.3).
// GitHub's tag list has no dates, so they're sorted by Compare on the ID.
func GitHubTags(ctx context.Context, get Getter, repo string, pattern *regexp.Regexp) ([]plugin.Version, error) {
	ts, err := pages[struct {
		Name string `json:"name"`
	}](ctx, get, GitHubAPI+"/repos/"+repo+"/tags")
	if err != nil {
		return nil, fmt.Errorf("listing %s's tags: %w", repo, err)
	}
	var out []plugin.Version
	for _, t := range ts {
		m := pattern.FindStringSubmatch(t.Name)
		if m == nil {
			continue
		}
		id := t.Name
		if len(m) > 1 && m[1] != "" {
			id = m[1]
		}
		out = append(out, plugin.Version{ID: id, Ref: t.Name, URL: "https://github.com/" + repo + "/releases/tag/" + url.PathEscape(t.Name)})
	}
	slices.SortStableFunc(out, func(a, b plugin.Version) int { return Compare(a.ID, b.ID) })
	return out, nil
}

// GitHubCommits are the commits on the default branch that changed path,
// oldest first: a version per change, for a file the repo never tags. The ID
// is the commit's date (YYYY-MM-DD), the Ref its SHA.
func GitHubCommits(ctx context.Context, get Getter, repo, path string) ([]plugin.Version, error) {
	cs, err := pages[struct {
		SHA    string `json:"sha"`
		URL    string `json:"html_url"`
		Commit struct {
			Committer struct {
				Date time.Time `json:"date"`
			} `json:"committer"`
		} `json:"commit"`
	}](ctx, get, GitHubAPI+"/repos/"+repo+"/commits?path="+url.QueryEscape(path))
	if err != nil {
		return nil, fmt.Errorf("listing the commits that changed %s in %s: %w", path, repo, err)
	}
	var out []plugin.Version
	for _, c := range cs {
		if c.SHA == "" || c.Commit.Committer.Date.IsZero() {
			continue
		}
		at := c.Commit.Committer.Date.UTC()
		out = append(out, plugin.Version{ID: at.Format("2006-01-02"), Released: at, Ref: c.SHA, URL: c.URL})
	}
	slices.SortStableFunc(out, func(a, b plugin.Version) int { return a.Released.Compare(b.Released) })
	return out, nil
}

// PyPIReleases are a package's final releases on PyPI (no pre-, dev- or
// yanked releases, none without files), oldest first by version. The Ref is
// the version; a plugin that reads a repo at a tag maps it.
func PyPIReleases(ctx context.Context, get Getter, pkg string) ([]plugin.Version, error) {
	body, err := get(ctx, PyPI+"/pypi/"+url.PathEscape(pkg)+"/json")
	if err != nil {
		return nil, fmt.Errorf("listing %s's releases on PyPI: %w", pkg, err)
	}
	var doc struct {
		Releases map[string][]struct {
			Uploaded time.Time `json:"upload_time_iso_8601"`
			Yanked   bool      `json:"yanked"`
		} `json:"releases"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("parsing PyPI's %s: %w", pkg, err)
	}
	var out []plugin.Version
	for v, files := range doc.Releases {
		if len(files) == 0 || !Final(v) {
			continue
		}
		var first time.Time
		yanked := true
		for _, f := range files {
			yanked = yanked && f.Yanked
			if first.IsZero() || f.Uploaded.Before(first) {
				first = f.Uploaded
			}
		}
		if yanked {
			continue
		}
		out = append(out, plugin.Version{ID: v, Released: first.UTC(), Ref: v, URL: "https://pypi.org/project/" + pkg + "/" + v + "/"})
	}
	slices.SortStableFunc(out, func(a, b plugin.Version) int { return Compare(a.ID, b.ID) })
	return out, nil
}

// NPMReleases are a package's published versions on npm, prereleases and
// deprecated versions left out, oldest first by version.
func NPMReleases(ctx context.Context, get Getter, pkg string) ([]plugin.Version, error) {
	body, err := get(ctx, NPM+"/"+strings.Replace(url.PathEscape(pkg), "%40", "@", 1))
	if err != nil {
		return nil, fmt.Errorf("listing %s's versions on npm: %w", pkg, err)
	}
	var doc struct {
		Versions map[string]struct {
			Deprecated any `json:"deprecated"`
		} `json:"versions"`
		Time map[string]time.Time `json:"time"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("parsing npm's %s: %w", pkg, err)
	}
	var out []plugin.Version
	for v, meta := range doc.Versions {
		if !Final(v) || meta.Deprecated != nil {
			continue
		}
		out = append(out, plugin.Version{ID: v, Released: doc.Time[v].UTC(), Ref: v, URL: "https://www.npmjs.com/package/" + pkg + "/v/" + v})
	}
	slices.SortStableFunc(out, func(a, b plugin.Version) int { return Compare(a.ID, b.ID) })
	return out, nil
}

var (
	final  = regexp.MustCompile(`^v?\d+(\.\d+)*$`)
	number = regexp.MustCompile(`\d+`)
)

// Final is true for a plain release number ("2.0.0", "v0.86.0", "11.0"), false
// for a pre-release, dev, post or local version ("2.0.0rc1", "1.0.0-beta.2").
func Final(v string) bool { return final.MatchString(v) }

// Compare orders two version IDs: their numbers part by part ("1.10" after
// "1.9"), dates as numbers too ("2026-01-02" after "2025-12-31"), then as
// strings when the numbers tie.
func Compare(a, b string) int {
	na, nb := number.FindAllString(a, -1), number.FindAllString(b, -1)
	for i := 0; i < len(na) && i < len(nb); i++ {
		x, _ := strconv.Atoi(na[i])
		y, _ := strconv.Atoi(nb[i])
		if c := cmp.Compare(x, y); c != 0 {
			return c
		}
	}
	if c := cmp.Compare(len(na), len(nb)); c != 0 {
		return c
	}
	return strings.Compare(a, b)
}

// Major is a version's first number ("2" for "2.0.0", "8" for "posthog-v8.1"),
// or -1.
func Major(v string) int {
	m := number.FindString(v)
	if m == "" {
		return -1
	}
	n, _ := strconv.Atoi(m)
	return n
}

// Pairs are the consecutive pairs of vs, oldest first: what a backfill
// extracts, one pair at a time.
func Pairs(vs []plugin.Version) [][2]plugin.Version {
	var out [][2]plugin.Version
	for i := 1; i < len(vs); i++ {
		out = append(out, [2]plugin.Version{vs[i-1], vs[i]})
	}
	return out
}
