// Package flagsmith is what patch-it reads for Flagsmith: every source, registered
// below with its metadata, each read by its kind (kinds/<kind>). README.md
// says why each source and what it reads.
package flagsmith

import (
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi"
)

func init() {
	plugin.RegisterProvider(plugin.Provider{
		ID: "flagsmith", Name: "Flagsmith", Category: plugin.CategoryDevTools, Homepage: "https://www.flagsmith.com",
		Sources: []plugin.Source{
			// No Hints: Flagsmith's release notes are product news, not keyed by
			// operation. RequestSide, same reason as GitLab's: a named request-body
			// schema shared by several operations (FeatureValue, SegmentOverride)
			// reads as its own removal, side "response" by the differ's default,
			// whether or not any operation that used it is actually gone.
			{
				ID: "api", Layer: plugin.LayerAPI, Watches: "REST API changes between releases of Flagsmith's own OpenAPI spec (removed endpoints and request parameters)",
				URLs:   []string{"https://github.com/Flagsmith/flagsmith/blob/main/openapi.yaml"},
				Plugin: openapi.Plugin{Adapter: Flagsmith{}, Keep: openapi.RequestSide},
			},
		},
	})
}
