// Package neon is what patch-it reads for Neon: every source, registered
// below with its metadata, each read by its kind (kinds/<kind>). README.md
// says why each source and what it reads.
package neon

import (
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/kinds/apisurface"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi"
)

func init() {
	plugin.RegisterProvider(plugin.Provider{
		ID: "neon", Name: "Neon", Category: plugin.CategoryDatabases, Homepage: "https://neon.com",
		Sources: []plugin.Source{
			// No Hints: Neon's changelog is a weekly digest, not keyed by operation.
			// Only operation deprecations and removals: see neon.go.
			{
				ID: "api", Layer: plugin.LayerAPI, Watches: "Neon API operations deprecated, and deprecated operations removed, between commits of the spec",
				URLs:     []string{"https://github.com/neondatabase/neon-pkgs"},
				Packages: []string{"npm:@neondatabase/api-client"},
				Plugin:   openapi.Plugin{Adapter: Neon{}, Diff: neonDiff, Keep: neonKeep},
			},
			apisurface.NPM(ServerlessSurface, "https://github.com/neondatabase/serverless"),
		},
	})
}

// The packages whose public API the apisurface sources read.
var (
	ServerlessSurface = apisurface.NPMPackage("neon/serverless-surface", "@neondatabase/serverless")
)
