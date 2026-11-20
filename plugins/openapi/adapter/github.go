package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/patch-it-dev/patch-it-plugins/core/fetch"
	"github.com/patch-it-dev/patch-it-plugins/core/github"
)

// Variables, not constants, only so tests can point them at a local server.
var (
	// githubVersionsURL is GitHub's own meta endpoint, documented at
	// https://docs.github.com/rest/meta/meta#get-all-api-versions. It returns
	// every currently supported API version, e.g. ["2026-03-10","2022-11-28"],
	// not necessarily newest first (checked 2026-11-14).
	githubVersionsURL = "https://api.github.com/versions"
	// githubSpecURLTemplate is the bundled (not dereferenced) REST API
	// description for github.com itself, one file per supported API version.
	// It lives in the same repo that generates GitHub's own Octokit SDKs.
	githubSpecURLTemplate = "https://raw.githubusercontent.com/github/rest-api-description/main/descriptions/api.github.com/api.github.com.%s.json"
)

// GitHub adapts github/rest-api-description's per-API-version REST
// descriptions for the openapi plugin. Unlike Stripe and Datadog, there is no
// release tag to speak of: GitHub's own versioning scheme names each spec
// file after the dated API version it describes, and that string is also
// exactly what a client sends back on X-GitHub-Api-Version.
type GitHub struct{}

var _ Adapter = GitHub{}

func (GitHub) API() string { return "github" }

// LatestTag asks GitHub for every API version it currently supports and
// returns the newest, by string order (the dates sort correctly as text).
// GitHub returns the list newest-first, but that isn't documented, so this
// doesn't rely on it.
func (GitHub) LatestTag(ctx context.Context) (string, error) {
	body, err := fetch.Get(ctx, githubVersionsURL, github.Headers())
	if err != nil {
		return "", fmt.Errorf("looking up github's supported api versions: %w", err)
	}
	var versions []string
	if err := json.Unmarshal(body, &versions); err != nil {
		return "", fmt.Errorf("parsing github versions response: %w", err)
	}
	if len(versions) == 0 {
		return "", fmt.Errorf("github versions response was empty")
	}
	sort.Strings(versions)
	return versions[len(versions)-1], nil
}

// Fetch downloads the bundled spec for the given dated API version. It is
// already JSON (unlike Datadog's bundled YAML), so there is nothing to
// convert. Version is the dated version itself, not info.version: that field
// is a fixed package version ("1.1.4" at the time of checking) identical
// across every spec file, so it can't tell two API versions apart — the same
// problem Datadog's static info.version has.
func (GitHub) Fetch(ctx context.Context, tag string) (Spec, error) {
	url := fmt.Sprintf(githubSpecURLTemplate, tag)
	body, err := fetch.Get(ctx, url, nil) // raw.githubusercontent.com needs no auth
	if err != nil {
		return Spec{}, fmt.Errorf("fetching github spec %s: %w", tag, err)
	}
	if !json.Valid(body) {
		return Spec{}, fmt.Errorf("github spec %s is not valid json", tag)
	}

	return Spec{
		Content:   body,
		Version:   tag,
		SourceURL: url,
	}, nil
}
