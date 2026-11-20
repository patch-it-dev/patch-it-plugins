package adapter

import (
	"context"
	"strings"

	"github.com/patch-it-dev/patch-it-plugins/core/github"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/core/versions"
)

// Every version of each spec that can be listed, oldest first (Lister): the
// repo's releases for the adapters whose LatestTag is releases/latest, the
// commits that changed the spec for those that follow a file. The others
// (GitHub's dated versions, GitLab, Elasticsearch, Sentry, Algolia, Appwrite)
// are read at LatestTag only, for now.

func releases(ctx context.Context, repo string, keep func(string) bool) ([]plugin.Version, error) {
	return versions.GitHubReleases(ctx, github.Get, repo, keep)
}

func commits(ctx context.Context, repo, path string) ([]plugin.Version, error) {
	return versions.GitHubCommits(ctx, github.Get, repo, path)
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
