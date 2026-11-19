package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/patch-it-dev/patch-it-plugins/fetch"
)

// Variables, not constants, only so tests can point them at a local server.
var (
	vercelReleasesURL = "https://api.github.com/repos/vercel/sdk/releases/latest"
	// vercelSpecURLTemplate is the REST API spec @vercel/sdk is generated from,
	// committed at the root of vercel/sdk and regenerated on every release.
	// Vercel also serves it live at https://openapi.vercel.sh/, but that URL
	// has no versions to pin or replay. The SDK repo's copy is the same spec,
	// a release behind at most (on 2026-11-18 the live one had 3 paths more
	// out of 356), and tagged on every release.
	vercelSpecURLTemplate = "https://raw.githubusercontent.com/vercel/sdk/%s/vercel-spec.json"
)

// Vercel adapts the OpenAPI spec bundled in vercel/sdk for the openapi plugin.
type Vercel struct{}

var _ Adapter = Vercel{}

func (Vercel) API() string { return "vercel" }

// LatestTag asks GitHub for vercel/sdk's newest release tag, e.g. "v1.28.65".
// The canary tags (v14.0.0-canary.9) aren't releases, so releases/latest never
// returns them (checked 2026-11-18).
func (Vercel) LatestTag(ctx context.Context) (string, error) {
	body, err := fetch.Get(ctx, vercelReleasesURL, githubHeaders())
	if err != nil {
		return "", fmt.Errorf("looking up vercel's latest release: %w", err)
	}
	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.Unmarshal(body, &release); err != nil {
		return "", fmt.Errorf("parsing github release response: %w", err)
	}
	if release.TagName == "" {
		return "", fmt.Errorf("github release response had no tag_name")
	}
	return release.TagName, nil
}

// Fetch downloads the spec pinned to tag. It's JSON already, so it's passed
// through unchanged.
//
// Version is the release tag without its "v", not info.version: that is a
// fixed "0.0.1" in every release (checked 2026-11-18). The tag is what a
// customer pins in package.json ("@vercel/sdk": "1.28.65"). Vercel's REST API
// has no version of its own: each endpoint carries its own in the path
// (/v13/deployments, /v6/user/tokens).
func (Vercel) Fetch(ctx context.Context, tag string) (Spec, error) {
	url := fmt.Sprintf(vercelSpecURLTemplate, tag)
	body, err := fetch.Get(ctx, url, nil) // raw.githubusercontent.com needs no auth
	if err != nil {
		return Spec{}, fmt.Errorf("fetching vercel spec %s: %w", tag, err)
	}
	if !json.Valid(body) {
		return Spec{}, fmt.Errorf("vercel spec %s is not valid json", tag)
	}
	return Spec{Content: body, Version: strings.TrimPrefix(tag, "v"), SourceURL: url}, nil
}
