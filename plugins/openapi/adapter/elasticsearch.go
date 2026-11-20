package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/patch-it-dev/patch-it-plugins/core/fetch"
	"github.com/patch-it-dev/patch-it-plugins/core/github"
)

// Variables, not constants, only so tests can point them at a local server.
var (
	// esReleasesURL is Elasticsearch's own releases, newest first by date (an
	// 8.19 patch can come after a 9.5 one), so the newest version is computed,
	// not taken from releases/latest.
	esReleasesURL = "https://api.github.com/repos/elastic/elasticsearch/releases?per_page=100"
	// esBranchesURL lists every branch of elastic/elasticsearch-specification
	// in one response (the branches API pages; this doesn't): one per minor
	// version, "9.5".
	esBranchesURL = "https://api.github.com/repos/elastic/elasticsearch-specification/git/matching-refs/heads/"
	// esCommitURL is a branch's head commit.
	esCommitURL = "https://api.github.com/repos/elastic/elasticsearch-specification/commits/%s"
	// esSpecURLTemplate is the OpenAPI spec generated from the specification
	// (the one Elastic's clients and API reference are built from) at one
	// commit. The OpenAPI output starts with the 8.14 branch.
	esSpecURLTemplate = "https://raw.githubusercontent.com/elastic/elasticsearch-specification/%s/output/openapi/elasticsearch-openapi.json"
)

// Elasticsearch adapts the Elasticsearch REST API spec in
// elastic/elasticsearch-specification for the openapi plugin, minor version
// by minor version. A tag here is a minor version's branch ("9.5").
type Elasticsearch struct{}

var _ Adapter = Elasticsearch{}

func (Elasticsearch) API() string { return "elasticsearch" }

var (
	esRelease = regexp.MustCompile(`^v(\d+)\.(\d+)\.(\d+)$`)
	esBranch  = regexp.MustCompile(`^(\d+)\.(\d+)$`)
)

// LatestTag is the branch of the newest minor version Elastic has released:
// not the newest branch, which is the next, unreleased one (9.6 on
// 2026-11-19, when the newest release was 9.5.6), whose spec can still
// change before it ships.
func (Elasticsearch) LatestTag(ctx context.Context) (string, error) {
	newest, branches, err := esMinors(ctx)
	if err != nil {
		return "", err
	}
	want := fmt.Sprintf("%d.%d", newest[0], newest[1])
	if slices.Contains(branches, want) {
		return want, nil
	}
	return "", fmt.Errorf("elasticsearch %s is released, but elasticsearch-specification has no %s branch", want, want)
}

// esMinors is the newest minor Elastic has released and every minor branch
// ("9.5") of elasticsearch-specification.
func esMinors(ctx context.Context) (newest [2]int, branches []string, err error) {
	body, err := fetch.Get(ctx, esReleasesURL, github.Headers())
	if err != nil {
		return newest, nil, fmt.Errorf("listing elasticsearch releases: %w", err)
	}
	var releases []struct {
		Tag        string `json:"tag_name"`
		Prerelease bool   `json:"prerelease"`
		Draft      bool   `json:"draft"`
	}
	if err := json.Unmarshal(body, &releases); err != nil {
		return newest, nil, fmt.Errorf("parsing github releases: %w", err)
	}
	newest = [2]int{-1, -1}
	for _, r := range releases {
		m := esRelease.FindStringSubmatch(r.Tag)
		if m == nil || r.Prerelease || r.Draft {
			continue
		}
		v := [2]int{atoi(m[1]), atoi(m[2])}
		if v[0] > newest[0] || v[0] == newest[0] && v[1] > newest[1] {
			newest = v
		}
	}
	if newest[0] < 0 {
		return newest, nil, fmt.Errorf("github lists no elasticsearch release")
	}

	body, err = fetch.Get(ctx, esBranchesURL, github.Headers())
	if err != nil {
		return newest, nil, fmt.Errorf("listing elasticsearch-specification branches: %w", err)
	}
	var refs []struct {
		Ref string `json:"ref"`
	}
	if err := json.Unmarshal(body, &refs); err != nil {
		return newest, nil, fmt.Errorf("parsing github branch refs: %w", err)
	}
	for _, r := range refs {
		if b := strings.TrimPrefix(r.Ref, "refs/heads/"); esBranch.MatchString(b) {
			branches = append(branches, b)
		}
	}
	return newest, branches, nil
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

// Fetch downloads the spec at the head of the branch, pinned to that commit:
// a branch moves (fixes land on it after the release), a commit doesn't.
//
// Version is the minor's first release, "9.5.0": what a repo pins, and the
// elasticsearch clients' versions follow the server's (elasticsearch-py 9.x
// for Elasticsearch 9.x). info.version in the spec is empty.
func (Elasticsearch) Fetch(ctx context.Context, branch string) (Spec, error) {
	if !esBranch.MatchString(branch) {
		return Spec{}, fmt.Errorf("%q isn't an elasticsearch-specification minor branch", branch)
	}
	sha, err := github.Head(ctx, github.Get, fmt.Sprintf(esCommitURL, branch))
	if err != nil {
		return Spec{}, fmt.Errorf("looking up the head of elasticsearch-specification %s: %w", branch, err)
	}
	url := fmt.Sprintf(esSpecURLTemplate, sha)
	body, err := fetch.Get(ctx, url, nil) // raw.githubusercontent.com needs no auth
	if err != nil {
		return Spec{}, fmt.Errorf("fetching the elasticsearch %s spec: %w", branch, err)
	}
	if !json.Valid(body) {
		return Spec{}, fmt.Errorf("the elasticsearch %s spec is not valid json", branch)
	}
	return Spec{Content: body, Version: branch + ".0", SourceURL: url}, nil
}
