package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/core/fetch"
	"github.com/patch-it-dev/patch-it-plugins/core/github"
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

var _ Adapter = Algolia{}

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
func (a Algolia) Fetch(ctx context.Context, tag string) (Spec, error) {
	m := algoliaTag.FindStringSubmatch(tag)
	if m == nil {
		return Spec{}, fmt.Errorf("%q isn't an algolia release tag", tag)
	}
	url := fmt.Sprintf(algoliaSpecURLTemplate, tag, a.Spec)
	body, err := fetch.Get(ctx, url, nil) // raw.githubusercontent.com needs no auth
	if err != nil {
		return Spec{}, fmt.Errorf("fetching algolia %s spec %s: %w", a.Spec, tag, err)
	}
	var doc any
	if err := yaml.Unmarshal(body, &doc); err != nil {
		return Spec{}, fmt.Errorf("parsing algolia %s spec %s as yaml: %w", a.Spec, tag, err)
	}
	content, err := json.Marshal(doc)
	if err != nil {
		return Spec{}, fmt.Errorf("converting algolia %s spec %s to json: %w", a.Spec, tag, err)
	}
	return Spec{Content: content, Version: m[1], SourceURL: url}, nil
}
