package algolia

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/core/diff"
	"github.com/patch-it-dev/patch-it-plugins/core/fetch"
	"github.com/patch-it-dev/patch-it-plugins/core/github"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi/adapter"
	"gopkg.in/yaml.v3"
)

// Variables, not constants, only so tests can point them at a local server.
var (
	// algoliaRefsURL lists every release tag of algolia/api-clients-automation,
	// "released-2026-11-17-31c309fe2a" (or "released-2023-08-22" before
	// 2024): the date, then the commit.
	algoliaRefsURL = "https://api.github.com/repos/algolia/api-clients-automation/git/matching-refs/tags/released-"
	// algoliaCommitURL is one tag's commit, for its date.
	algoliaCommitURL = "https://api.github.com/repos/algolia/api-clients-automation/commits/%s"
	// algoliaSpecURLTemplate is one API's bundled spec at one release tag: the
	// spec Algolia generates its API clients from.
	algoliaSpecURLTemplate = "https://raw.githubusercontent.com/algolia/api-clients-automation/%s/specs/bundled/%s.yml"
)

// Algolia adapts one of Algolia's REST API specs (Spec: "search",
// "ingestion", "recommend") in algolia/api-clients-automation for the openapi
// plugin, release by release.
type Algolia struct {
	Spec string
}

var _ adapter.Adapter = Algolia{}

func (a Algolia) API() string { return "algolia/" + a.Spec + "-api" }

var algoliaTag = regexp.MustCompile(`^released-(\d{4}-\d{2}-\d{2})(-[0-9a-f]+)?$`)

// LatestTag is the newest release tag. The name sorts by day, but a day can
// have several releases (three on 2024-09-19) whose commit suffix says
// nothing about their order, so those are told apart by their commit's date.
func (a Algolia) LatestTag(ctx context.Context) (string, error) {
	tags, err := algoliaTags(ctx)
	if err != nil {
		return "", err
	}
	var day string
	var sameDay []string
	for _, tag := range tags {
		m := algoliaTag.FindStringSubmatch(tag)
		switch {
		case m[1] > day:
			day, sameDay = m[1], []string{tag}
		case m[1] == day:
			sameDay = append(sameDay, tag)
		}
	}
	if len(sameDay) == 0 {
		return "", fmt.Errorf("github lists no algolia release tag")
	}
	if len(sameDay) == 1 {
		return sameDay[0], nil
	}
	var latest string
	var latestAt time.Time
	for _, tag := range sameDay {
		at, err := algoliaCommitDate(ctx, tag)
		if err != nil {
			return "", err
		}
		if at.After(latestAt) {
			latest, latestAt = tag, at
		}
	}
	return latest, nil
}

// algoliaTags are every release tag, in the listing's order.
func algoliaTags(ctx context.Context) ([]string, error) {
	body, err := fetch.Get(ctx, algoliaRefsURL, github.Headers())
	if err != nil {
		return nil, fmt.Errorf("listing algolia's release tags: %w", err)
	}
	var refs []struct {
		Ref string `json:"ref"`
	}
	if err := json.Unmarshal(body, &refs); err != nil {
		return nil, fmt.Errorf("parsing github's tag refs: %w", err)
	}
	var tags []string
	for _, r := range refs {
		if tag := strings.TrimPrefix(r.Ref, "refs/tags/"); algoliaTag.MatchString(tag) {
			tags = append(tags, tag)
		}
	}
	return tags, nil
}

func algoliaCommitDate(ctx context.Context, tag string) (time.Time, error) {
	_, date, err := github.Commit(ctx, github.Get, fmt.Sprintf(algoliaCommitURL, tag))
	if err != nil {
		return time.Time{}, fmt.Errorf("algolia tag %s: no commit date: %w", tag, err)
	}
	return date, nil
}

// Fetch downloads the spec at tag and converts it from YAML to JSON, which is
// what diff.Parse reads (as for Datadog; the excerpts are re-serialized from
// the node either way).
//
// Version is the release's date from the tag (2026-11-17): Algolia's REST API
// has no version beyond the /1/ in its paths, and each API client has its own
// version numbers, so the day the spec was released is the closest thing to
// "the API version the change arrived in".
func (a Algolia) Fetch(ctx context.Context, tag string) (adapter.Spec, error) {
	m := algoliaTag.FindStringSubmatch(tag)
	if m == nil {
		return adapter.Spec{}, fmt.Errorf("%q isn't an algolia release tag", tag)
	}
	url := fmt.Sprintf(algoliaSpecURLTemplate, tag, a.Spec)
	body, err := fetch.Get(ctx, url, nil) // raw.githubusercontent.com needs no auth
	if err != nil {
		return adapter.Spec{}, fmt.Errorf("fetching algolia %s spec %s: %w", a.Spec, tag, err)
	}
	var doc any
	if err := yaml.Unmarshal(body, &doc); err != nil {
		return adapter.Spec{}, fmt.Errorf("parsing algolia %s spec %s as yaml: %w", a.Spec, tag, err)
	}
	content, err := json.Marshal(doc)
	if err != nil {
		return adapter.Spec{}, fmt.Errorf("converting algolia %s spec %s to json: %w", a.Spec, tag, err)
	}
	return adapter.Spec{Content: content, Version: m[1], SourceURL: url}, nil
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

// algoliaDiff reads Algolia's specs like Supabase's: deprecations by the
// `deprecated` flag, quoting the whole operation, and path templates matched.
var algoliaDiff = diff.Options{PathTemplates: true, QuoteDeprecatedOperation: true}

// algoliaKeep keeps operation deprecations, and removals of operations
// marked deprecated first, of REST operations only. Algolia's specs also
// describe what only its API clients do, and a change to one of those isn't
// an API change:
//
//   - helpers, marked `x-helper: true`: /saveObjects, /waitForTask,
//     /replaceAllObjects, … (16 in the Search API on 2026-11-17);
//   - the "custom request" methods for any path, operationId customGet,
//     customPost, customPut and customDelete on /{path} (before 2024-03-25,
//     /1{path}).
//
// In every release from 2023-08-22 to 2026-11-17 this keeps 9 Search API
// deprecations (the multi-cluster /1/clusters endpoints, 2025-04-01) and 9
// Ingestion API ones (the /1/tasks endpoints, 2025-01-22, and PUT
// /1/tasks/{taskID}/disable, 2024-07-26). None of the 17 removals in that
// history was deprecated first; 13 were the client-only entries above (the
// custom-request methods in all three specs on 2024-03-25, and /saveObjects).
func algoliaKeep(c diff.Change) bool {
	// c.Node is the operation only because algoliaDiff sets
	// QuoteDeprecatedOperation (a removal carries the old operation anyway);
	// TestAlgoliaClientOnlyFilterThroughTheDiffer pins that.
	op, _ := c.Node.(map[string]any)
	if clientOnly(op) {
		return false
	}
	return openapi.DeprecationsOnly(algoliaDiff)(c)
}

// clientOnly is an Algolia spec entry for an API-client method, not a REST
// operation.
func clientOnly(op map[string]any) bool {
	if helper, _ := op["x-helper"].(bool); helper {
		return true
	}
	id, _ := op["operationId"].(string)
	switch id {
	case "customGet", "customPost", "customPut", "customDelete":
		return true
	}
	return false
}
