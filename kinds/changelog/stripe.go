package changelog

import "github.com/patch-it-dev/patch-it-plugins/core/plugin"

func init() {
	const page = "providers/stripe-sdks.md"
	plugin.RegisterProvider(plugin.Provider{
		ID: "stripe", Name: "Stripe", Category: plugin.CategoryPayments, Homepage: "https://stripe.com",
		Sources: []plugin.Source{
			sdk(page, "python", StripePython, "pypi:stripe"),
			sdk(page, "node", StripeNode, "npm:stripe"),
			sdk(page, "go", StripeGo, "go:github.com/stripe/stripe-go/v87"),
			sdk(page, "java", StripeJava, "maven:com.stripe:stripe-java"),
			sdk(page, "ruby", StripeRuby, "rubygems:stripe"),
			sdk(page, "dotnet", StripeDotnet, "nuget:Stripe.net"),
			sdk(page, "php", StripePHP, "packagist:stripe/stripe-php"),
		},
	})
}

// Stripe's SDKs: one CHANGELOG.md each, "## <a id="X-Y-Z"></a>X.Y.Z - date"
// (older ones "## X.Y.Z - date"), every breaking bullet marked "⚠️"; each
// release tagged vX.Y.Z. A release that changes the API version the SDK
// pins says so in its section, "This release changes the pinned API version
// to `2026-11-18.endive`." (pins.go): its MapsTo is that stripe/api version.
var (
	StripePython = File{API: "stripe/python", Package: "stripe", Ecosystem: "pypi", Module: "stripe",
		Repo: "stripe/stripe-python", TagPrefix: "v", Path: "CHANGELOG.md", Pins: "stripe/api"}
	StripeNode = File{API: "stripe/node", Package: "stripe", Ecosystem: "npm",
		Repo: "stripe/stripe-node", TagPrefix: "v", Path: "CHANGELOG.md", Pins: "stripe/api"}
	StripeGo = File{API: "stripe/go", Package: "github.com/stripe/stripe-go", Ecosystem: "go",
		Repo: "stripe/stripe-go", TagPrefix: "v", Path: "CHANGELOG.md", Pins: "stripe/api"}
	StripeJava = File{API: "stripe/java", Package: "com.stripe:stripe-java", Ecosystem: "maven",
		Repo: "stripe/stripe-java", TagPrefix: "v", Path: "CHANGELOG.md", Pins: "stripe/api"}
	StripeRuby = File{API: "stripe/ruby", Package: "stripe", Ecosystem: "rubygems",
		Repo: "stripe/stripe-ruby", TagPrefix: "v", Path: "CHANGELOG.md", Pins: "stripe/api"}
	StripeDotnet = File{API: "stripe/dotnet", Package: "Stripe.net", Ecosystem: "nuget",
		Repo: "stripe/stripe-dotnet", TagPrefix: "v", Path: "CHANGELOG.md", Pins: "stripe/api"}
	StripePHP = File{API: "stripe/php", Package: "stripe/stripe-php", Ecosystem: "packagist",
		Repo: "stripe/stripe-php", TagPrefix: "v", Path: "CHANGELOG.md", Pins: "stripe/api"}
)
