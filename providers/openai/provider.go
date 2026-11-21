// Package openai is what patch-it reads for OpenAI: every source, registered
// below with its metadata, each read by its kind (kinds/<kind>). README.md
// says why each source and what it reads.
package openai

import (
	"github.com/patch-it-dev/patch-it-plugins/core/kit"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/kinds/apisurface"
	"github.com/patch-it-dev/patch-it-plugins/kinds/notices"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi"
)

func init() {
	plugin.RegisterProvider(plugin.Provider{
		ID: "openai", Name: "OpenAI", Category: plugin.CategoryAI, Homepage: "https://openai.com",
		Sources: []plugin.Source{
			// No Hints: OpenAI's changelog is product news, not keyed by operation.
			// Only operation deprecations and removals: see api.go.
			{
				ID: "api", Layer: plugin.LayerAPI,
				Watches:  "REST API operations deprecated, and deprecated operations removed, day by day in the spec OpenAI's SDKs are generated from",
				URLs:     []string{"https://github.com/openai/openai-openapi/blob/main/openapi.yaml"},
				Packages: []string{"pypi:openai", "npm:openai"},
				Plugin:   openapi.Plugin{Adapter: OpenAI{}, Diff: openaiDiff, Keep: openaiKeep},
			},
			{
				ID: "models", Layer: plugin.LayerAIModels,
				Watches:  "Model deprecations and shutdown dates (Chat Completions and Responses text models)",
				URLs:     []string{"https://developers.openai.com/api/docs/deprecations"},
				Packages: []string{"pypi:openai", "npm:openai"},
				// Anchors: two models in the page's "Past deprecations" table, which keeps
				// every shutdown (gpt-4-0613: Oct 23, 2026; gpt-3.5-turbo-0301: Sep 13, 2024).
				Shape:  plugin.Shape{Anchors: []plugin.Anchor{{Entry: "gpt-4-0613", Checked: "2026-11-21"}, {Entry: "gpt-3.5-turbo-0301", Checked: "2026-11-21"}}},
				Plugin: notices.Plugin{Page: Models, Archiver: kit.Wayback{}},
			},
			{
				ID: "deprecations", Layer: plugin.LayerAPI,
				Watches:  "REST API endpoints OpenAI shuts down, named by their path, with the shutdown date, from the deprecations page's feature and endpoint tables",
				URLs:     []string{"https://developers.openai.com/api/docs/deprecations"},
				Packages: []string{"pypi:openai", "npm:openai"},
				// Anchors: two endpoints shut down long ago, kept in the page's "Past
				// deprecations" table (/v1/edits: Jan 4, 2024; /v1/engines: Dec 3, 2022).
				Shape:  plugin.Shape{Anchors: []plugin.Anchor{{Entry: "/v1/edits", Checked: "2026-11-21"}, {Entry: "/v1/engines", Checked: "2026-11-21"}}},
				Plugin: notices.Plugin{Page: Deprecations, Archiver: kit.Wayback{}},
			},
			apisurface.PyPI(PythonSurface, plugin.LayerSDK, "https://github.com/openai/openai-python"),
			apisurface.NPM(NodeSurface, "https://github.com/openai/openai-node"),
		},
	})
}

// The packages whose public API the apisurface sources read.
var (
	NodeSurface   = apisurface.NPMPackage("openai/node-surface", "openai")
	PythonSurface = apisurface.Package{ID: "openai/python-surface", Name: "openai", Ecosystem: "pypi", Module: "openai"}
)
