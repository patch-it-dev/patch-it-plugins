// Package appwrite is what patch-it reads for Appwrite: every source, registered
// below with its metadata, each read by its kind (kinds/<kind>). README.md
// says why each source and what it reads.
package appwrite

import (
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi"
)

func init() {
	plugin.RegisterProvider(plugin.Provider{
		ID: "appwrite", Name: "Appwrite", Category: plugin.CategoryHosting, Homepage: "https://appwrite.io",
		Sources: []plugin.Source{
			// No Hints: Appwrite's changelog is product news, not keyed by operation.
			// Operation deprecations only, SDK method renames left out: see
			// api.go.
			{
				ID: "api", Layer: plugin.LayerAPI,
				Watches:  "REST API operations a server SDK calls that are deprecated between commits of the spec, SDK method renames left out",
				URLs:     []string{"https://github.com/appwrite/specs"},
				Packages: []string{"pypi:appwrite", "npm:node-appwrite"},
				Plugin:   openapi.Plugin{Adapter: Appwrite{}, Diff: appwriteDiff, Keep: appwriteKeep},
			},
		},
	})
}
