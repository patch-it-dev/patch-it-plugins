package changelog

import "github.com/patch-it-dev/patch-it-plugins/core/plugin"

func init() {
	const page = "providers/cloudflare-sdks.md"
	plugin.RegisterProvider(plugin.Provider{
		ID: "cloudflare", Name: "Cloudflare", Category: plugin.CategoryInfrastructure, Homepage: "https://www.cloudflare.com",
		Sources: []plugin.Source{
			sdk(page, "typescript", CloudflareTypeScript, "npm:cloudflare"),
			sdk(page, "python", CloudflarePython, "pypi:cloudflare"),
			sdk(page, "go", CloudflareGo, "go:github.com/cloudflare/cloudflare-go/v7"),
			sdk(page, "terraform", CloudflareTerraform),
		},
	})
}

// Cloudflare's Stainless SDKs and its Terraform provider: release-please
// CHANGELOG.md, "## X.Y.Z (date)" with "### ⚠ BREAKING CHANGES" and bullets
// that start with an area ("**accounts:** ..."); tagged vX.Y.Z.
var (
	CloudflareTypeScript = File{API: "cloudflare/typescript", Package: "cloudflare", Ecosystem: "npm",
		Repo: "cloudflare/cloudflare-typescript", TagPrefix: "v", Path: "CHANGELOG.md"}
	CloudflarePython = File{API: "cloudflare/python", Package: "cloudflare", Ecosystem: "pypi", Module: "cloudflare",
		Repo: "cloudflare/cloudflare-python", TagPrefix: "v", Path: "CHANGELOG.md"}
	CloudflareGo = File{API: "cloudflare/go", Package: "github.com/cloudflare/cloudflare-go", Ecosystem: "go",
		Repo: "cloudflare/cloudflare-go", TagPrefix: "v", Path: "CHANGELOG.md"}
	CloudflareTerraform = File{API: "cloudflare/terraform", Package: "cloudflare/cloudflare", Ecosystem: "terraform",
		Repo: "cloudflare/terraform-provider-cloudflare", TagPrefix: "v", Path: "CHANGELOG.md"}
)
