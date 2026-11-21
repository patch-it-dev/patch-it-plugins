// Package planetscale is what patch-it reads for PlanetScale: every source, registered
// below with its metadata, each read by its kind (kinds/<kind>). README.md
// says why each source and what it reads.
package planetscale

import (
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi"
)

func init() {
	plugin.RegisterProvider(plugin.Provider{
		ID: "planetscale", Name: "PlanetScale", Category: plugin.CategoryDatabases, Homepage: "https://planetscale.com",
		Sources: []plugin.Source{
			// No Hints: PlanetScale's changelog is product news, not keyed by
			// operation. Operation deprecations only, said in the description: see
			// api.go.
			{
				ID: "api", Layer: plugin.LayerAPI,
				Watches: "API operations deprecated, and deprecated operations removed, between commits of the spec, said in the operation's description",
				URLs:    []string{"https://github.com/planetscale/terraform-provider-planetscale"},
				Plugin:  openapi.Plugin{Adapter: PlanetScale{}, Diff: planetscaleDiff, Keep: planetscaleKeep},
			},
		},
	})
}
