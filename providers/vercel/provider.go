// Package vercel is what patch-it reads for Vercel: every source, registered
// below with its metadata, each read by its kind (kinds/<kind>). README.md
// says why each source and what it reads.
package vercel

import (
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi"
)

func init() {
	plugin.RegisterProvider(plugin.Provider{
		ID: "vercel", Name: "Vercel", Category: plugin.CategoryHosting, Homepage: "https://vercel.com",
		Sources: []plugin.Source{
			// No Hints: Vercel's changelog is product news, not keyed by API change.
			// Only operation deprecations and removals are kept: see vercelKeep.
			{
				ID: "api", Layer: plugin.LayerAPI, Watches: "REST API operations deprecated, and deprecated operations removed, between SDK releases of the spec",
				URLs:     []string{"https://github.com/vercel/sdk"},
				Packages: []string{"npm:@vercel/sdk"},
				Plugin:   openapi.Plugin{Adapter: Vercel{}, Diff: vercelDiff, Keep: vercelKeep},
			},
		},
	})
}
