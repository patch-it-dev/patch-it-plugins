// Package anthropic is what patch-it reads for Anthropic: every source, registered
// below with its metadata, each read by its kind (kinds/<kind>). README.md
// says why each source and what it reads.
package anthropic

import (
	"strings"

	"github.com/patch-it-dev/patch-it-plugins/core/kit"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/kinds/apisurface"
	"github.com/patch-it-dev/patch-it-plugins/kinds/migrationguide"
	"github.com/patch-it-dev/patch-it-plugins/kinds/migrationguide/sdkguide"
	"github.com/patch-it-dev/patch-it-plugins/kinds/notices"
	"github.com/patch-it-dev/patch-it-plugins/kinds/notices/retire"
)

func init() {
	plugin.RegisterProvider(plugin.Provider{
		ID: "anthropic", Name: "Anthropic", Category: plugin.CategoryAI, Homepage: "https://www.anthropic.com",
		Sources: []plugin.Source{
			{
				ID: "models", Layer: plugin.LayerAIModels,
				Watches:  "Model deprecations and retirement dates (Messages API models)",
				URLs:     []string{"https://platform.claude.com/docs/en/about-claude/model-deprecations"},
				Packages: []string{"pypi:anthropic", "npm:@anthropic-ai/sdk"},
				// Anchors: two retired models in the page's tables, which keep every
				// retirement (claude-3-opus-20240229: Jan 5, 2026; claude-2.1: Jul 21, 2025).
				Shape:  plugin.Shape{Anchors: []plugin.Anchor{{Entry: "claude-3-opus-20240229", Checked: "2026-11-21"}, {Entry: "claude-2.1", Checked: "2026-11-21"}}},
				Plugin: notices.Plugin{Page: Models, Archiver: kit.Wayback{}},
			},
			{
				ID: "python-sdk", Layer: plugin.LayerSDK,
				Watches:  "Request parameters each new major of the Python SDK removes from `messages.create()` and friends, from MIGRATION.md at the major's tag",
				URLs:     []string{"https://github.com/anthropics/anthropic-sdk-python/blob/main/MIGRATION.md", "https://pypi.org/project/anthropic/"},
				Packages: []string{"pypi:anthropic"},
				Plugin:   migrationguide.Plugin{Guide: PythonSDK},
			},
			apisurface.PyPI(PythonSurface, plugin.LayerSDK, "https://github.com/anthropics/anthropic-sdk-python"),
			apisurface.NPM(TypescriptSurface, "https://github.com/anthropics/anthropic-sdk-typescript"),
		},
	})
}

// Models is Anthropic's model deprecations page (checked 2026-10-13): every model in
// its deprecation history is a Messages API model.
var Models = retire.Page{
	API:      "anthropic/models",
	Provider: "anthropic",
	Vendor:   "Anthropic",
	URL:      "https://platform.claude.com/docs/en/about-claude/model-deprecations.md",
	Keep:     func(name string) bool { return strings.HasPrefix(name, "claude-") },
}

// PythonSDK is anthropic-sdk-python's MIGRATION.md, unpinned: read at the
// first release of each major (Major 0; At pins it).
var PythonSDK = sdkguide.Guide{
	API:     "anthropic/python-sdk",
	Package: "anthropic",
	Repo:    "anthropics/anthropic-sdk-python",
	Methods: "`messages.create()`",
	Symbols: []string{"messages.create", "messages.stream", "messages.parse"},
}

// The packages whose public API the apisurface sources read.
var (
	PythonSurface     = apisurface.Package{ID: "anthropic/python-surface", Name: "anthropic", Ecosystem: "pypi", Module: "anthropic"}
	TypescriptSurface = apisurface.NPMPackage("anthropic/typescript-surface", "@anthropic-ai/sdk")
)
