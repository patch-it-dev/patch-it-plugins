package adapter

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/patch-it-dev/patch-it-plugins/core/fetch"
	"github.com/patch-it-dev/patch-it-plugins/core/github"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
)

// Variables, not constants, only so tests can point them at a local server.
var (
	cloudflareRepo = "cloudflare/api-schemas"
	cloudflarePath = "openapi.json"
	// cloudflareCommitsURL lists the commits that changed the spec, newest
	// first; with per_page=1, the newest.
	cloudflareCommitsURL = "https://api.github.com/repos/cloudflare/api-schemas/commits?path=openapi.json&per_page=1"
	// cloudflareCommitURL is one commit, for its date.
	cloudflareCommitURL = "https://api.github.com/repos/cloudflare/api-schemas/commits/%s"
	// cloudflareSpecURLTemplate is the OpenAPI description of the whole
	// Cloudflare API (api.cloudflare.com/client/v4) at one commit of
	// cloudflare/api-schemas (BSD-3-Clause), the spec Cloudflare's SDKs and
	// Terraform provider are generated from. A bot commits it as "Update
	// schema: openapi.json" several times a day and the repo has no tags
	// (checked 2026-11-21), so the commits are the versions. About 30 MB;
	// openapi.yaml beside it is the same spec.
	cloudflareSpecURLTemplate = "https://raw.githubusercontent.com/cloudflare/api-schemas/%s/openapi.json"
)

// Cloudflare adapts the Cloudflare API's spec in cloudflare/api-schemas for
// the openapi plugin. A tag here is a commit SHA.
type Cloudflare struct{}

var _ Adapter = Cloudflare{}

func (Cloudflare) API() string { return "cloudflare/api-spec" }

// LatestTag is the SHA of the newest commit that changed the spec.
func (Cloudflare) LatestTag(ctx context.Context) (string, error) {
	sha, err := github.LatestCommit(ctx, github.Get, cloudflareCommitsURL)
	if err != nil {
		return "", fmt.Errorf("looking up cloudflare's latest spec commit: %w", err)
	}
	return sha, nil
}

// Versions are the days the spec changed, each read at that day's last
// commit (Daily).
func (Cloudflare) Versions(ctx context.Context) ([]plugin.Version, error) {
	return commits(ctx, cloudflareRepo, cloudflarePath)
}

// Fetch downloads the spec at commit sha and moves the server's /client/v4
// into the paths: see ServerBasePath. Version is the commit's date
// (YYYY-MM-DD): info.version is a fixed "4.0.0" (checked 2026-11-21).
func (Cloudflare) Fetch(ctx context.Context, sha string) (Spec, error) {
	fullSHA, date, err := github.Commit(ctx, github.Get, fmt.Sprintf(cloudflareCommitURL, sha))
	if err != nil {
		return Spec{}, fmt.Errorf("looking up cloudflare spec commit %s: %w", sha, err)
	}
	if fullSHA == "" {
		return Spec{}, fmt.Errorf("github commit %s has no sha", sha)
	}
	url := fmt.Sprintf(cloudflareSpecURLTemplate, fullSHA)
	body, err := fetch.Get(ctx, url, nil) // raw.githubusercontent.com needs no auth
	if err != nil {
		return Spec{}, fmt.Errorf("fetching cloudflare spec %s: %w", sha, err)
	}
	if !json.Valid(body) {
		return Spec{}, fmt.Errorf("cloudflare spec %s is not valid json", sha)
	}
	// The one server is "https://api.cloudflare.com/client/v4" and the
	// paths don't repeat it: moved, an operation is named by the path a
	// caller sends.
	content, err := ServerBasePath(body)
	if err != nil {
		return Spec{}, fmt.Errorf("cloudflare spec %s: %w", sha, err)
	}
	return Spec{Content: content, Version: date.UTC().Format("2006-01-02"), SourceURL: url}, nil
}

// SourcePointer: see BasePathPointer.
func (Cloudflare) SourcePointer(content []byte) func(string) string {
	return BasePathPointer(content)
}

var (
	_ Reshaper = Cloudflare{}
	_ Lister   = Cloudflare{}
)
