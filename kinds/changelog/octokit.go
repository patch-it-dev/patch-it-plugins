package changelog

import "github.com/patch-it-dev/patch-it-plugins/core/plugin"

func init() {
	const page = "providers/github-octokit.md"
	plugin.RegisterProvider(plugin.Provider{
		ID: "github", Name: "GitHub", Category: plugin.CategoryDevTools, Homepage: "https://github.com",
		Sources: []plugin.Source{
			sdk(page, "octokit-rest", OctokitRest, "npm:@octokit/rest"),
			sdk(page, "octokit-net", OctokitNet, "nuget:Octokit"),
		},
	})
}

// Octokit's SDKs keep no changelog file: each GitHub release's notes are it
// (semantic-release's "### BREAKING CHANGES" for rest.js, "Breaking
// changes" lists for octokit.net); tagged vX.Y.Z.
var (
	OctokitRest = File{API: "github/octokit-rest", Package: "@octokit/rest", Ecosystem: "npm",
		Repo: "octokit/rest.js", TagPrefix: "v", Notes: true}
	OctokitNet = File{API: "github/octokit-net", Package: "Octokit", Ecosystem: "nuget",
		Repo: "octokit/octokit.net", TagPrefix: "v", Notes: true}
)
