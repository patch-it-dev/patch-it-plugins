// Package algolia is what patch-it reads for Algolia: every source, registered
// below with its metadata, each read by its kind (kinds/<kind>). README.md
// says why each source and what it reads.
package algolia

import (
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/kinds/apisurface"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi"
)

func init() {
	plugin.RegisterProvider(plugin.Provider{
		ID: "algolia", Name: "Algolia", Category: plugin.CategorySearch, Homepage: "https://www.algolia.com",
		Sources: append(specs(),
			apisurface.PyPI(PythonSurface, plugin.LayerSDK, "https://github.com/algolia/algoliasearch-client-python"),
		),
	})
}

// The packages whose public API the apisurface sources read.
var (
	PythonSurface = apisurface.Package{ID: "algolia/python-surface", Name: "algoliasearch", Ecosystem: "pypi", Module: "algoliasearch"}
)

// specs are the API specs' sources, one per spec, each read by Algolia's
// adapter (specs.go). No Hints: Algolia's changelog is a hosted product
// feed, not keyed by operation.
func specs() []plugin.Source {
	var out []plugin.Source
	for _, spec := range []struct{ name, title string }{{"search", "Search"}, {"ingestion", "Ingestion"}, {"recommend", "Recommend"}} {
		out = append(out, plugin.Source{
			ID: spec.name + "-api", Layer: plugin.LayerAPI,
			Watches:  spec.title + " API operations deprecated, and deprecated operations removed, between releases of the spec",
			URLs:     []string{"https://github.com/algolia/api-clients-automation", "https://github.com/algolia/api-clients-automation/blob/main/specs/bundled/" + spec.name + ".yml"},
			Packages: []string{"pypi:algoliasearch", "npm:algoliasearch"},
			Plugin:   openapi.Plugin{Adapter: Algolia{Spec: spec.name}, Diff: algoliaDiff, Keep: algoliaKeep},
		})
	}
	return out
}
