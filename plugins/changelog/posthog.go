package changelog

import "github.com/patch-it-dev/patch-it-plugins/core/plugin"

func init() {
	const page = "providers/posthog-go.md"
	plugin.RegisterProvider(plugin.Provider{
		ID: "posthog", Name: "PostHog", Category: plugin.CategoryAnalytics, Homepage: "https://posthog.com",
		Sources: []plugin.Source{sdk(page, "go", PostHogGo, "go:github.com/posthog/posthog-go")},
	})
}

// PostHogGo is posthog-go's changesets CHANGELOG.md: "## X.Y.Z" with "###
// Major Changes", each bullet starting with its commit; tagged vX.Y.Z.
var PostHogGo = File{API: "posthog/go", Package: "github.com/posthog/posthog-go", Ecosystem: "go", Module: "posthog",
	Repo: "PostHog/posthog-go", TagPrefix: "v", Path: "CHANGELOG.md"}
