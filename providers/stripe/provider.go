// Package stripe is what patch-it reads for Stripe: every source, registered
// below with its metadata, each read by its kind (kinds/<kind>). README.md
// says why each source and what it reads.
package stripe

import (
	"github.com/patch-it-dev/patch-it-plugins/core/hint"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/kinds/apisurface"
	"github.com/patch-it-dev/patch-it-plugins/kinds/changelog"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi"
)

func init() {
	plugin.RegisterProvider(plugin.Provider{
		ID: "stripe", Name: "Stripe", Category: plugin.CategoryPayments, Homepage: "https://stripe.com",
		Sources: []plugin.Source{
			{
				ID: "api", Layer: plugin.LayerAPI,
				Watches:  "REST API changes between OpenAPI spec releases (removed and renamed endpoints, parameters and fields), with migration hints from Stripe's API changelog",
				URLs:     []string{"https://github.com/stripe/openapi"},
				Packages: []string{"pypi:stripe", "npm:stripe"},
				Plugin:   openapi.Plugin{Adapter: Stripe{}, Hints: hint.Stripe{}},
			},
			changelog.SDK("python", Python, "pypi:stripe"),
			changelog.SDK("node", Node, "npm:stripe"),
			changelog.SDK("go", Go, "go:github.com/stripe/stripe-go/v87"),
			changelog.SDK("java", Java, "maven:com.stripe:stripe-java"),
			changelog.SDK("ruby", Ruby, "rubygems:stripe"),
			changelog.SDK("dotnet", Dotnet, "nuget:Stripe.net"),
			changelog.SDK("php", PHP, "packagist:stripe/stripe-php"),
			apisurface.PyPI(PythonSurface, plugin.LayerSDK, "https://github.com/stripe/stripe-python"),
			apisurface.NPM(NodeSurface, "https://github.com/stripe/stripe-node"),
		},
	})
}

// Stripe's SDKs: one CHANGELOG.md each, "## <a id="X-Y-Z"></a>X.Y.Z - date"
// (older ones "## X.Y.Z - date"), every breaking bullet marked "⚠️"; each
// release tagged vX.Y.Z. A release that changes the API version the SDK
// pins says so in its section, "This release changes the pinned API version
// to `2026-11-18.endive`." (kinds/changelog/pins.go): its MapsTo is that
// stripe/api version.
var (
	Python = changelog.File{API: "stripe/python", Package: "stripe", Ecosystem: "pypi", Module: "stripe",
		Repo: "stripe/stripe-python", TagPrefix: "v", Path: "CHANGELOG.md", Pins: "stripe/api"}
	Node = changelog.File{API: "stripe/node", Package: "stripe", Ecosystem: "npm",
		Repo: "stripe/stripe-node", TagPrefix: "v", Path: "CHANGELOG.md", Pins: "stripe/api"}
	Go = changelog.File{API: "stripe/go", Package: "github.com/stripe/stripe-go", Ecosystem: "go",
		Repo: "stripe/stripe-go", TagPrefix: "v", Path: "CHANGELOG.md", Pins: "stripe/api"}
	Java = changelog.File{API: "stripe/java", Package: "com.stripe:stripe-java", Ecosystem: "maven",
		Repo: "stripe/stripe-java", TagPrefix: "v", Path: "CHANGELOG.md", Pins: "stripe/api"}
	Ruby = changelog.File{API: "stripe/ruby", Package: "stripe", Ecosystem: "rubygems",
		Repo: "stripe/stripe-ruby", TagPrefix: "v", Path: "CHANGELOG.md", Pins: "stripe/api"}
	Dotnet = changelog.File{API: "stripe/dotnet", Package: "Stripe.net", Ecosystem: "nuget",
		Repo: "stripe/stripe-dotnet", TagPrefix: "v", Path: "CHANGELOG.md", Pins: "stripe/api"}
	PHP = changelog.File{API: "stripe/php", Package: "stripe/stripe-php", Ecosystem: "packagist",
		Repo: "stripe/stripe-php", TagPrefix: "v", Path: "CHANGELOG.md", Pins: "stripe/api"}
)

// The packages whose public API the apisurface sources read.
var (
	NodeSurface   = apisurface.NPMPackage("stripe/node-surface", "stripe")
	PythonSurface = apisurface.Package{ID: "stripe/python-surface", Name: "stripe", Ecosystem: "pypi", Module: "stripe"}
)
