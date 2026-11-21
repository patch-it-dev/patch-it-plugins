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
				Plugin:   notices.Plugin{Page: Models, Archiver: kit.Wayback{}},
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
