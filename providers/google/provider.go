// Package google is what patch-it reads for Google: every source, registered
// below with its metadata, each read by its kind (kinds/<kind>). README.md
// says why each source and what it reads.
package google

import (
	"github.com/patch-it-dev/patch-it-plugins/core/kit"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/kinds/apisurface"
	"github.com/patch-it-dev/patch-it-plugins/kinds/notices"
)

func init() {
	plugin.RegisterProvider(plugin.Provider{
		ID: "google", Name: "Google", Category: plugin.CategoryAI, Homepage: "https://cloud.google.com",
		Sources: []plugin.Source{
			{
				ID: "vertex-ai-models", Layer: plugin.LayerAIModels,
				Watches: "Generative AI model deprecations on Vertex AI",
				URLs:    []string{"https://docs.cloud.google.com/feeds/generative-ai-on-vertex-ai-release-notes.xml"},
				// Anchor: the feed's own title. A release-notes feed drops its oldest
				// entries, so no entry stays; the title says the URL still serves this feed.
				Shape:  plugin.Shape{Anchors: []plugin.Anchor{{Entry: "Generative AI on Vertex AI - Release notes", Checked: "2026-11-21"}}},
				Plugin: notices.Plugin{Page: VertexAIModels, Archiver: kit.Wayback{}},
			},
			{
				ID: "gemini-models", Layer: plugin.LayerAIModels,
				Watches:  "Gemini API model shutdown dates (the earliest Google gives) and their replacements, from Google's own deprecations table",
				URLs:     []string{"https://ai.google.dev/gemini-api/docs/deprecations", "https://aistudio.google.com/docs/deprecations"},
				Packages: []string{"pypi:google-genai", "npm:@google/genai"},
				// Anchors: two shut-down models in the page's tables, which keep past
				// shutdowns (gemini-2.0-flash: Jun 1, 2026; embedding-001: Oct 30, 2025).
				Shape:  plugin.Shape{Anchors: []plugin.Anchor{{Entry: "gemini-2.0-flash", Checked: "2026-11-21"}, {Entry: "embedding-001", Checked: "2026-11-21"}}},
				Plugin: notices.Plugin{Page: GeminiModels, Archiver: kit.Wayback{}},
			},
			{
				ID: "vertex-model-retirements", Layer: plugin.LayerAIModels,
				Watches:  "Vertex AI (Gemini Enterprise Agent Platform) model retirement dates and replacements, from Google Cloud's model versions tables",
				URLs:     []string{"https://docs.cloud.google.com/gemini-enterprise-agent-platform/models/model-versions"},
				Packages: []string{"pypi:google-genai", "pypi:google-cloud-aiplatform", "npm:@google/genai"},
				// Anchors: two retired models in the page's tables, which keep past
				// retirements (gemini-1.5-pro-002: Sep 24, 2025; text-bison: Apr 21, 2025).
				Shape:  plugin.Shape{Anchors: []plugin.Anchor{{Entry: "gemini-1.5-pro-002", Checked: "2026-11-21"}, {Entry: "text-bison", Checked: "2026-11-21"}}},
				Plugin: notices.Plugin{Page: VertexModelRetirements, Archiver: kit.Wayback{}},
			},
			apisurface.PyPI(GenaiPythonSurface, plugin.LayerSDK, "https://github.com/googleapis/python-genai"),
			apisurface.NPM(GenaiJSSurface, "https://github.com/googleapis/js-genai"),
		},
	})
}

// The packages whose public API the apisurface sources read.
var (
	GenaiJSSurface     = apisurface.NPMPackage("google/genai-js-surface", "@google/genai")
	GenaiPythonSurface = apisurface.Package{ID: "google/genai-python-surface", Name: "google-genai", Ecosystem: "pypi", Module: "google"}
)
