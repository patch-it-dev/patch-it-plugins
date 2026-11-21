package changelog

import "github.com/patch-it-dev/patch-it-plugins/core/plugin"

func init() {
	const page = "providers/sentry-sdks.md"
	plugin.RegisterProvider(plugin.Provider{
		ID: "sentry", Name: "Sentry", Category: plugin.CategoryMonitoring, Homepage: "https://sentry.io",
		Sources: []plugin.Source{
			sdk(page, "python-changelog", SentryPython, "pypi:sentry-sdk"),
			sdk(page, "javascript-changelog", SentryJavaScript, "npm:@sentry/node", "npm:@sentry/browser"),
			sdk(page, "go", SentryGo, "go:github.com/getsentry/sentry-go"),
			sdk(page, "dotnet", SentryDotnet, "nuget:Sentry"),
		},
	})
}

// Sentry's SDKs: CHANGELOG.md, "## X.Y.Z", with "### Breaking Changes" /
// "### Removed" / "### Deprecations" sections or conventional-commit bullets
// ("feat(core)!: Remove `x`"); tagged X.Y.Z, except sentry-go's vX.Y.Z.
var (
	SentryPython = File{API: "sentry/python-changelog", Package: "sentry-sdk", Ecosystem: "pypi", Module: "sentry_sdk",
		Repo: "getsentry/sentry-python", Path: "CHANGELOG.md"}
	SentryJavaScript = File{API: "sentry/javascript-changelog", Package: "@sentry/node", Ecosystem: "npm",
		Repo: "getsentry/sentry-javascript", Path: "CHANGELOG.md"}
	SentryGo = File{API: "sentry/go", Package: "github.com/getsentry/sentry-go", Ecosystem: "go", Module: "sentry",
		Repo: "getsentry/sentry-go", TagPrefix: "v", Path: "CHANGELOG.md"}
	SentryDotnet = File{API: "sentry/dotnet", Package: "Sentry", Ecosystem: "nuget",
		Repo: "getsentry/sentry-dotnet", Path: "CHANGELOG.md"}
)
