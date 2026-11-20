package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/patch-it-dev/patch-it-plugins/core/fetch"
	"github.com/patch-it-dev/patch-it-plugins/core/github"
	"gopkg.in/yaml.v3"
)

// Variables, not constants, only so tests can point them at a local server.
var (
	flagsmithReleasesURL = "https://api.github.com/repos/Flagsmith/flagsmith/releases/latest"
	// flagsmithSpecURLTemplate is the OpenAPI 3.1 description of Flagsmith's
	// full REST API (the Admin/Management API and the SDK-facing endpoints
	// together), committed at the root of Flagsmith/flagsmith as openapi.yaml
	// (checked 2026-11-19). It's generated from the Django backend with
	// drf-spectacular (`make generate-full-openapi`) and regenerated on every
	// PR that touches api/** by a bot that commits the result back to the PR
	// branch (.github/workflows/api-fix-docs-artefacts.yml), so a release tag
	// always carries the spec for the code it actually ships. There's also a
	// narrower sdk/openapi.yaml, filtered down to the SDK-only endpoints
	// (openapi-filter-flagsmith-sdk.yml): this adapter reads the full file so
	// Admin/Management API removals aren't missed.
	flagsmithSpecURLTemplate = "https://raw.githubusercontent.com/Flagsmith/flagsmith/%s/openapi.yaml"
)

// Flagsmith adapts Flagsmith's own OpenAPI spec for the openapi plugin.
type Flagsmith struct{}

var _ Adapter = Flagsmith{}

func (Flagsmith) API() string { return "flagsmith/api" }

// LatestTag asks GitHub for Flagsmith's newest release tag, e.g. "v2.293.0".
// Flagsmith tags a release on (almost) every merge to main via release-please,
// so releases/latest is always close to HEAD.
func (Flagsmith) LatestTag(ctx context.Context) (string, error) {
	tag, err := github.LatestRelease(ctx, github.Get, flagsmithReleasesURL)
	if err != nil {
		return "", fmt.Errorf("looking up flagsmith's latest release: %w", err)
	}
	return tag, nil
}

// Fetch downloads the spec pinned to tag and converts it from YAML to JSON,
// since diff.Parse only reads JSON. This changes no provenance guarantee:
// normalize's excerpt already re-serializes whatever node it quotes rather
// than slicing the original bytes, for every adapter including Stripe's.
//
// Version is the release tag without its "v" ("2.293.0"), not info.version:
// that is a fixed "v1" in every release (checked 2026-11-19), so it can't
// tell two releases apart. The bare version is also what a self-hosted
// instance reports and what a customer tracks as "the Flagsmith version
// we're running".
func (Flagsmith) Fetch(ctx context.Context, tag string) (Spec, error) {
	url := fmt.Sprintf(flagsmithSpecURLTemplate, tag)
	body, err := fetch.Get(ctx, url, nil) // raw.githubusercontent.com needs no auth
	if err != nil {
		return Spec{}, fmt.Errorf("fetching flagsmith spec %s: %w", tag, err)
	}

	var doc any
	if err := yaml.Unmarshal(body, &doc); err != nil {
		return Spec{}, fmt.Errorf("parsing flagsmith spec %s as yaml: %w", tag, err)
	}
	content, err := json.Marshal(doc)
	if err != nil {
		return Spec{}, fmt.Errorf("converting flagsmith spec %s to json: %w", tag, err)
	}

	return Spec{
		Content:   content,
		Version:   strings.TrimPrefix(tag, "v"),
		SourceURL: url,
	}, nil
}
