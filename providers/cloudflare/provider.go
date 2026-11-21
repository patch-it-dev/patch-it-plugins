// Package cloudflare is what patch-it reads for Cloudflare: every source, registered
// below with its metadata, each read by its kind (kinds/<kind>). README.md
// says why each source and what it reads.
package cloudflare

import (
	"github.com/patch-it-dev/patch-it-plugins/core/kit"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/kinds/apisurface"
	"github.com/patch-it-dev/patch-it-plugins/kinds/changelog"
	"github.com/patch-it-dev/patch-it-plugins/kinds/notices"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi"
)

func init() {
	plugin.RegisterProvider(plugin.Provider{
		ID: "cloudflare", Name: "Cloudflare", Category: plugin.CategoryInfrastructure, Homepage: "https://www.cloudflare.com",
		Sources: []plugin.Source{
			// No Hints: Cloudflare's changelog is product news, not keyed by
			// operation. Only operation deprecations and removals: see api_spec.go.
			// (cloudflare/api is the deprecations feed, read by the notices kind, below.)
			{
				ID: "api-spec", Layer: plugin.LayerAPI,
				Watches:  "API operations deprecated, and deprecated operations removed, day by day in the spec Cloudflare's SDKs and Terraform provider are generated from",
				URLs:     []string{"https://github.com/cloudflare/api-schemas/blob/main/openapi.json"},
				Packages: []string{"pypi:cloudflare", "npm:cloudflare"},
				Plugin:   openapi.Plugin{Adapter: Cloudflare{}, Diff: cloudflareDiff, Keep: cloudflareKeep},
			},
			{
				ID: "api", Layer: plugin.LayerAPI,
				Watches:  "REST API operations Cloudflare deprecates, breaking on their end-of-life date when it gives one",
				URLs:     []string{"https://developers.cloudflare.com/fundamentals/api/reference/deprecations/index.xml"},
				Packages: []string{"pypi:cloudflare", "npm:cloudflare"},
				Plugin:   notices.Plugin{Page: API, Archiver: kit.Wayback{}},
			},
			changelog.SDK("typescript", Typescript, "npm:cloudflare"),
			changelog.SDK("python", Python, "pypi:cloudflare"),
			changelog.SDK("go", Go, "go:github.com/cloudflare/cloudflare-go/v7"),
			changelog.SDK("terraform", Terraform),
			apisurface.PyPI(PythonSurface, plugin.LayerSDK, "https://github.com/cloudflare/cloudflare-python"),
		},
	})
}

// Cloudflare's Stainless SDKs and its Terraform provider: release-please
// CHANGELOG.md, "## X.Y.Z (date)" with "### ⚠ BREAKING CHANGES" and bullets
// that start with an area ("**accounts:** ..."); tagged vX.Y.Z.
var (
	Typescript = changelog.File{API: "cloudflare/typescript", Package: "cloudflare", Ecosystem: "npm",
		Repo: "cloudflare/cloudflare-typescript", TagPrefix: "v", Path: "CHANGELOG.md"}
	Python = changelog.File{API: "cloudflare/python", Package: "cloudflare", Ecosystem: "pypi", Module: "cloudflare",
		Repo: "cloudflare/cloudflare-python", TagPrefix: "v", Path: "CHANGELOG.md"}
	Go = changelog.File{API: "cloudflare/go", Package: "github.com/cloudflare/cloudflare-go", Ecosystem: "go",
		Repo: "cloudflare/cloudflare-go", TagPrefix: "v", Path: "CHANGELOG.md"}
	Terraform = changelog.File{API: "cloudflare/terraform", Package: "cloudflare/cloudflare", Ecosystem: "terraform",
		Repo: "cloudflare/terraform-provider-cloudflare", TagPrefix: "v", Path: "CHANGELOG.md"}
)

// The packages whose public API the apisurface sources read.
var (
	PythonSurface = apisurface.Package{ID: "cloudflare/python-surface", Name: "cloudflare", Ecosystem: "pypi", Module: "cloudflare"}
)
