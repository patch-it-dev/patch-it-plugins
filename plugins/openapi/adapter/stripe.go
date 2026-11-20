package adapter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/patch-it-dev/patch-it-plugins/core/fetch"
	"github.com/patch-it-dev/patch-it-plugins/core/github"
)

// Variables, not constants, only so tests can point them at a local server.
var (
	stripeReleasesURL = "https://api.github.com/repos/stripe/openapi/releases/latest"
	// stripeSpecURLTemplate is Stripe's recommended file: v1 + v2 endpoints.
	stripeSpecURLTemplate = "https://raw.githubusercontent.com/stripe/openapi/%s/latest/openapi.spec3.json"
	// stripeLegacySpecURLTemplate is v1 only, but it's the only file that
	// exists on tags from before Stripe added the latest/ directory (e.g.
	// v207, from 2022) — still updated on every release since.
	stripeLegacySpecURLTemplate = "https://raw.githubusercontent.com/stripe/openapi/%s/openapi/spec3.json"
)

// Stripe is the reference adapter: the pattern every future adapter follows.
// It has no state, so its zero value is ready to use.
type Stripe struct{}

// Compile-time check that Stripe actually satisfies Adapter.
var _ Adapter = Stripe{}

func (Stripe) API() string { return "stripe" }

// LatestTag asks GitHub for Stripe's newest openapi release tag, e.g. "v2468".
func (Stripe) LatestTag(ctx context.Context) (string, error) {
	tag, err := github.LatestRelease(ctx, github.Get, stripeReleasesURL)
	if err != nil {
		return "", fmt.Errorf("looking up stripe's latest release: %w", err)
	}
	return tag, nil
}

// Fetch downloads the spec pinned to tag and reads its declared API version
// out of info.version — the only version field the spec has. Tags old enough
// to predate the latest/ directory fall back to the legacy, v1-only file.
func (Stripe) Fetch(ctx context.Context, tag string) (Spec, error) {
	url := fmt.Sprintf(stripeSpecURLTemplate, tag)
	body, err := fetch.Get(ctx, url, nil) // raw.githubusercontent.com needs no auth

	var statusErr *fetch.StatusError
	if errors.As(err, &statusErr) && statusErr.StatusCode == http.StatusNotFound {
		url = fmt.Sprintf(stripeLegacySpecURLTemplate, tag)
		body, err = fetch.Get(ctx, url, nil)
	}
	if err != nil {
		return Spec{}, fmt.Errorf("fetching stripe spec %s: %w", tag, err)
	}

	var parsed struct {
		Info struct {
			Version string `json:"version"`
		} `json:"info"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return Spec{}, fmt.Errorf("parsing stripe spec %s: %w", tag, err)
	}
	if parsed.Info.Version == "" {
		return Spec{}, fmt.Errorf("stripe spec %s has no info.version", tag)
	}

	return Spec{
		Content:   body,
		Version:   parsed.Info.Version,
		SourceURL: url,
	}, nil
}
