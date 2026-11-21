package adapter

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/core/fetch"
	"github.com/patch-it-dev/patch-it-plugins/core/github"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/core/versions"
)

// Every version of each spec, oldest first (Lister): the repo's releases for
// the adapters whose LatestTag is releases/latest, the commits that changed
// the spec for those that follow a file, and each other adapter's own
// listing (GitHub's dated versions, GitLab's release tags, Elasticsearch's
// released minors, Sentry's releases, Algolia's release tags, Appwrite's
// spec commits). Every adapter is a Lister.

func releases(ctx context.Context, repo string, keep func(string) bool) ([]plugin.Version, error) {
	return versions.GitHubReleases(ctx, github.Get, repo, keep)
}

// commits are the days a file changed, each read at the day's last commit
// (Daily).
func commits(ctx context.Context, repo, path string) ([]plugin.Version, error) {
	vs, err := versions.GitHubCommits(ctx, github.Get, repo, path)
	if err != nil {
		return nil, err
	}
	return Daily(vs), nil
}

// Versions are stripe/openapi's releases.
func (Stripe) Versions(ctx context.Context) ([]plugin.Version, error) {
	return releases(ctx, "stripe/openapi", nil)
}

// Versions are datadog-api-client-python's releases.
func (Datadog) Versions(ctx context.Context) ([]plugin.Version, error) {
	return releases(ctx, "DataDog/datadog-api-client-python", nil)
}

// Versions are vercel/sdk's releases, canaries left out (as LatestTag does).
func (Vercel) Versions(ctx context.Context) ([]plugin.Version, error) {
	return releases(ctx, "vercel/sdk", func(tag string) bool { return !strings.Contains(tag, "canary") })
}

// Versions are Flagsmith's releases.
func (Flagsmith) Versions(ctx context.Context) ([]plugin.Version, error) {
	return releases(ctx, "Flagsmith/flagsmith", nil)
}

// Versions are the commits that changed the Management API spec.
func (Supabase) Versions(ctx context.Context) ([]plugin.Version, error) {
	return commits(ctx, "supabase/supabase", "apps/docs/spec/api_v1_openapi.json")
}

// Versions are the commits that changed Neon's spec.
func (Neon) Versions(ctx context.Context) ([]plugin.Version, error) {
	return commits(ctx, "neondatabase/neon-pkgs", "packages/sdk/spec/neon-openapi.json")
}

// Versions are the commits that changed Convex's deployment API spec.
func (Convex) Versions(ctx context.Context) ([]plugin.Version, error) {
	return commits(ctx, "get-convex/convex-backend", "npm-packages/@convex-dev/platform/deployment-openapi.json")
}

// Versions are the commits that changed PlanetScale's spec.
func (PlanetScale) Versions(ctx context.Context) ([]plugin.Version, error) {
	return commits(ctx, planetscaleRepo, planetscaleSpecPath)
}

var (
	_ Lister = Stripe{}
	_ Lister = Datadog{}
	_ Lister = Vercel{}
	_ Lister = Flagsmith{}
	_ Lister = Supabase{}
	_ Lister = Neon{}
	_ Lister = Convex{}
	_ Lister = PlanetScale{}
)

// Versions are every API version GitHub supports, oldest first: each is a
// spec file of its own, and what a client sends as X-GitHub-Api-Version.
// GitHub publishes no spec for a version it no longer supports.
func (GitHub) Versions(ctx context.Context) ([]plugin.Version, error) {
	body, err := fetch.Get(ctx, githubVersionsURL, github.Headers())
	if err != nil {
		return nil, fmt.Errorf("looking up github's supported api versions: %w", err)
	}
	var ids []string
	if err := json.Unmarshal(body, &ids); err != nil {
		return nil, fmt.Errorf("parsing github versions response: %w", err)
	}
	slices.Sort(ids)
	var out []plugin.Version
	for _, id := range ids {
		released, _ := time.Parse(time.DateOnly, id)
		out = append(out, plugin.Version{ID: id, Released: released, Ref: id, URL: "https://docs.github.com/rest/about-the-rest-api/api-versions"})
	}
	return out, nil
}

// gitlabFirstSpec is the first release whose tag carries openapi_v3.yaml.
var gitlabFirstSpec = [3]int{18, 10, 0}

// Versions are every stable GitLab release with the OpenAPI 3 spec
// (v18.10.0-ee on), oldest first by version: patch releases of an older
// minor (18.11.14) sit before the next minor's first release. The tags
// listing is read page by page, newest version first, until it passes
// v18.10.0-ee.
func (g GitLab) Versions(ctx context.Context) ([]plugin.Version, error) {
	type tag struct {
		Name   string `json:"name"`
		Commit struct {
			Created time.Time `json:"created_at"`
		} `json:"commit"`
	}
	var out []plugin.Version
	for page := 1; page <= versions.MaxPages; page++ {
		body, err := fetch.Get(ctx, fmt.Sprintf("%s&page=%d", cmp.Or(g.TagsURL, gitlabTagsURL), page), nil)
		if err != nil {
			return nil, fmt.Errorf("listing gitlab's tags: %w", err)
		}
		var tags []tag
		if err := json.Unmarshal(body, &tags); err != nil {
			return nil, fmt.Errorf("parsing gitlab tags response: %w", err)
		}
		older := false
		for _, t := range tags {
			m := gitlabRelease.FindStringSubmatch(t.Name)
			if m == nil {
				continue
			}
			v := [3]int{atoi(m[1]), atoi(m[2]), atoi(m[3])}
			if slices.Compare(v[:], gitlabFirstSpec[:]) < 0 {
				older = true
				continue
			}
			out = append(out, plugin.Version{ID: gitlabVersion(t.Name), Released: t.Commit.Created.UTC(), Ref: t.Name,
				URL: "https://gitlab.com/gitlab-org/gitlab/-/tags/" + t.Name})
		}
		if older || len(tags) < 100 {
			break
		}
	}
	slices.SortStableFunc(out, func(a, b plugin.Version) int { return versions.Compare(a.ID, b.ID) })
	return out, nil
}

// esFirstSpec is the first minor whose branch has the OpenAPI output.
var esFirstSpec = [2]int{8, 14}

// Versions are every released minor with the OpenAPI output (8.14 on), its
// branch the ref: up to the newest minor Elastic has released (LatestTag),
// never the next, unreleased one.
func (e Elasticsearch) Versions(ctx context.Context) ([]plugin.Version, error) {
	newest, branches, err := esMinors(ctx)
	if err != nil {
		return nil, err
	}
	var out []plugin.Version
	for _, b := range branches {
		m := esBranch.FindStringSubmatch(b)
		v := [2]int{atoi(m[1]), atoi(m[2])}
		if slices.Compare(v[:], esFirstSpec[:]) < 0 || slices.Compare(v[:], newest[:]) > 0 {
			continue
		}
		out = append(out, plugin.Version{ID: b + ".0", Ref: b, URL: "https://github.com/elastic/elasticsearch-specification/tree/" + b})
	}
	slices.SortStableFunc(out, func(a, b plugin.Version) int { return versions.Compare(a.ID, b.ID) })
	return out, nil
}

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

// Versions are every release tag, oldest first. A day with several releases
// is ordered by each tag's commit date (LatestTag's rule), one lookup per
// tag on such a day only.
func (a Algolia) Versions(ctx context.Context) ([]plugin.Version, error) {
	tags, err := algoliaTags(ctx)
	if err != nil {
		return nil, err
	}
	days := map[string]int{}
	for _, t := range tags {
		days[algoliaTag.FindStringSubmatch(t)[1]]++
	}
	var out []plugin.Version
	for _, t := range tags {
		day := algoliaTag.FindStringSubmatch(t)[1]
		released, _ := time.Parse(time.DateOnly, day)
		if days[day] > 1 {
			if released, err = algoliaCommitDate(ctx, t); err != nil {
				return nil, err
			}
		}
		out = append(out, plugin.Version{ID: day, Released: released.UTC(), Ref: t,
			URL: "https://github.com/algolia/api-clients-automation/releases/tag/" + t})
	}
	slices.SortStableFunc(out, func(a, b plugin.Version) int { return a.Released.Compare(b.Released) })
	return out, nil
}

// Versions are the days appwrite/specs' default branch changed specs/,
// oldest first, each at the day's last commit (Daily): the ID is the date
// (Fetch's Version), the Ref the SHA, read at the newest minor's directory.
func (a Appwrite) Versions(ctx context.Context) ([]plugin.Version, error) {
	cs, err := versions.GitHubPages[struct {
		SHA    string `json:"sha"`
		URL    string `json:"html_url"`
		Commit struct {
			Committer struct {
				Date time.Time `json:"date"`
			} `json:"committer"`
		} `json:"commit"`
	}](ctx, github.Get, a.api("/commits?path=specs"))
	if err != nil {
		return nil, fmt.Errorf("listing the commits that changed appwrite's specs: %w", err)
	}
	var out []plugin.Version
	for _, c := range cs {
		if !appwriteCommit.MatchString(c.SHA) || c.Commit.Committer.Date.IsZero() {
			continue
		}
		at := c.Commit.Committer.Date.UTC()
		out = append(out, plugin.Version{ID: at.Format(time.DateOnly), Released: at, Ref: c.SHA, URL: c.URL})
	}
	slices.SortStableFunc(out, func(a, b plugin.Version) int { return a.Released.Compare(b.Released) })
	return Daily(out), nil
}

var (
	_ Lister = GitHub{}
	_ Lister = GitLab{}
	_ Lister = Elasticsearch{}
	_ Lister = Sentry{}
	_ Lister = Algolia{}
	_ Lister = Appwrite{}
)

// Daily keeps one version per day of vs (oldest first, IDs dates, as
// versions.GitHubCommits lists them): the day's last commit, so a spec
// committed several times a day has one version per day, each ID once.
func Daily(vs []plugin.Version) []plugin.Version {
	var out []plugin.Version
	for _, v := range vs {
		if n := len(out); n > 0 && out[n-1].ID == v.ID {
			out[n-1] = v
			continue
		}
		out = append(out, v)
	}
	return out
}

var _ Lister = OpenAI{}
