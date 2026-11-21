// Package convex is what patch-it reads for Convex: every source, registered
// below with its metadata, each read by its kind (kinds/<kind>). README.md
// says why each source and what it reads.
package convex

import (
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/kinds/apisurface"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi"
)

func init() {
	plugin.RegisterProvider(plugin.Provider{
		ID: "convex", Name: "Convex", Category: plugin.CategoryDatabases, Homepage: "https://www.convex.dev",
		Sources: []plugin.Source{
			// No Hints: Convex has no API changelog; its news feed is product news.
			// Only operation deprecations and removals: see deployment_api.go.
			{
				ID: "deployment-api", Layer: plugin.LayerAPI,
				Watches: "Deployment API operations deprecated, and deprecated operations removed, between commits of the spec",
				URLs:    []string{"https://github.com/convex/convex-backend"},
				Plugin:  openapi.Plugin{Adapter: Convex{}, Diff: convexDiff, Keep: convexKeep},
			},
			apisurface.NPM(JSSurface, "https://github.com/convex/convex-js"),
		},
	})
}

// The packages whose public API the apisurface sources read.
var (
	JSSurface = apisurface.NPMPackage("convex/js-surface", "convex")
)
