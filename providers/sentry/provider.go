// Package sentry is what patch-it reads for Sentry: every source, registered
// below with its metadata, each read by its kind (kinds/<kind>). README.md
// says why each source and what it reads.
package sentry

import (
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/kinds/apisurface"
	"github.com/patch-it-dev/patch-it-plugins/kinds/changelog"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi"
)

func init() {
	plugin.RegisterProvider(plugin.Provider{
		ID: "sentry", Name: "Sentry", Category: plugin.CategoryMonitoring, Homepage: "https://sentry.io",
		Sources: []plugin.Source{
			// No Hints: Sentry's changelog is product news, not keyed by operation.
			// Operation deprecations only, most said in the description: see
			// sentry.go.
			{
				ID: "api", Layer: plugin.LayerAPI, Watches: "REST API operations deprecated between releases of the spec, most of them in the operation's description",
				URLs:   []string{"https://github.com/getsentry/sentry/releases"},
				Plugin: openapi.Plugin{Adapter: Sentry{}, Diff: sentryDiff, Keep: sentryKeep},
			},
			changelog.SDK("python-changelog", PythonChangelog, "pypi:sentry-sdk"),
			changelog.SDK("javascript-changelog", JavascriptChangelog, "npm:@sentry/node", "npm:@sentry/browser"),
			changelog.SDK("go", Go, "go:github.com/getsentry/sentry-go"),
			changelog.SDK("dotnet", Dotnet, "nuget:Sentry"),
			apisurface.PyPI(PythonSurface, plugin.LayerSDK, "https://github.com/getsentry/sentry-python"),
			apisurface.NPM(NodeSurface, "https://github.com/getsentry/sentry-javascript"),
		},
	})
}

// Sentry's SDKs: CHANGELOG.md, "## X.Y.Z", with "### Breaking Changes" /
// "### Removed" / "### Deprecations" sections or conventional-commit bullets
// ("feat(core)!: Remove `x`"); tagged X.Y.Z, except sentry-go's vX.Y.Z.
var (
	PythonChangelog = changelog.File{API: "sentry/python-changelog", Package: "sentry-sdk", Ecosystem: "pypi", Module: "sentry_sdk",
		Repo: "getsentry/sentry-python", Path: "CHANGELOG.md"}
	JavascriptChangelog = changelog.File{API: "sentry/javascript-changelog", Package: "@sentry/node", Ecosystem: "npm",
		Repo: "getsentry/sentry-javascript", Path: "CHANGELOG.md"}
	Go = changelog.File{API: "sentry/go", Package: "github.com/getsentry/sentry-go", Ecosystem: "go", Module: "sentry",
		Repo: "getsentry/sentry-go", TagPrefix: "v", Path: "CHANGELOG.md"}
	Dotnet = changelog.File{API: "sentry/dotnet", Package: "Sentry", Ecosystem: "nuget",
		Repo: "getsentry/sentry-dotnet", Path: "CHANGELOG.md"}
)

// The packages whose public API the apisurface sources read.
var (
	NodeSurface   = apisurface.NPMPackage("sentry/node-surface", "@sentry/node")
	PythonSurface = apisurface.Package{ID: "sentry/python-surface", Name: "sentry-sdk", Ecosystem: "pypi", Module: "sentry_sdk"}
)
