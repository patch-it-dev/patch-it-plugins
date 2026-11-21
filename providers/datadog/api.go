package datadog

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/patch-it-dev/patch-it-plugins/core/fetch"
	"github.com/patch-it-dev/patch-it-plugins/core/github"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi/adapter"
	"gopkg.in/yaml.v3"
)

// Variables, not constants, only so tests can point them at a local server.
var (
	datadogReleasesURL = "https://api.github.com/repos/DataDog/datadog-api-client-python/releases/latest"
	// datadogSpecURLTemplate is the v2 spec the Python, JS, Go, Java and Ruby
	// clients are all generated from. Datadog publishes no standalone spec
	// repo: the generator's output is bundled inside datadog-api-client-python
	// itself, regenerated on every release.
	datadogSpecURLTemplate = "https://raw.githubusercontent.com/DataDog/datadog-api-client-python/%s/.generator/schemas/v2/openapi.yaml"
)

// Datadog adapts datadog-api-client-python's bundled OpenAPI spec for the
// openapi plugin (diff.Parse expects JSON; the spec ships as YAML, converted
// in Fetch — see Fetch's comment).
type Datadog struct{}

var _ adapter.Adapter = Datadog{}

func (Datadog) API() string { return "datadog/api" }

// LatestTag asks GitHub for the client's newest release tag, e.g. "2.62.0".
// Datadog has no release process of its own to ask instead: the spec has no
// repo, so the client that's generated from it is the only thing that's
// tagged.
func (Datadog) LatestTag(ctx context.Context) (string, error) {
	tag, err := github.LatestRelease(ctx, github.Get, datadogReleasesURL)
	if err != nil {
		return "", fmt.Errorf("looking up datadog's latest release: %w", err)
	}
	return tag, nil
}

// Fetch downloads the spec pinned to tag and converts it from YAML to JSON,
// since diff.Parse only reads JSON. This changes no provenance guarantee:
// normalize's excerpt already re-serializes whatever node it quotes rather
// than slicing the original bytes, for every adapter including Stripe's.
//
// Version is the release tag, not info.version: unlike Stripe, Datadog's
// info.version is a fixed "1.0" that never moves between releases (checked
// 2026-11-13), so it can't tell two releases apart or sit in a fingerprint.
// The tag is also what a customer actually pins in requirements.txt
// (datadog-api-client==2.61.0), so the version gate can compare against it
// directly — no SDK-version-to-API-version table like Stripe's needed.
func (Datadog) Fetch(ctx context.Context, tag string) (adapter.Spec, error) {
	url := fmt.Sprintf(datadogSpecURLTemplate, tag)
	body, err := fetch.Get(ctx, url, nil) // raw.githubusercontent.com needs no auth
	if err != nil {
		return adapter.Spec{}, fmt.Errorf("fetching datadog spec %s: %w", tag, err)
	}

	var doc any
	if err := yaml.Unmarshal(body, &doc); err != nil {
		return adapter.Spec{}, fmt.Errorf("parsing datadog spec %s as yaml: %w", tag, err)
	}
	content, err := json.Marshal(doc)
	if err != nil {
		return adapter.Spec{}, fmt.Errorf("converting datadog spec %s to json: %w", tag, err)
	}

	return adapter.Spec{
		Content:   content,
		Version:   tag,
		SourceURL: url,
	}, nil
}

// Versions are datadog-api-client-python's releases.
func (Datadog) Versions(ctx context.Context) ([]plugin.Version, error) {
	return adapter.Releases(ctx, "DataDog/datadog-api-client-python", nil)
}
