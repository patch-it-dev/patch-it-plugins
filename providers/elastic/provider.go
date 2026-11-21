// Package elastic is what patch-it reads for Elastic: every source, registered
// below with its metadata, each read by its kind (kinds/<kind>). README.md
// says why each source and what it reads.
package elastic

import (
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/kinds/apisurface"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi"
)

func init() {
	plugin.RegisterProvider(plugin.Provider{
		ID: "elastic", Name: "Elastic", Category: plugin.CategorySearch, Homepage: "https://www.elastic.co",
		Sources: []plugin.Source{
			// No Hints: Elastic's breaking-changes page is prose per area, not keyed
			// by operation. See elasticsearch.go.
			{
				ID: "elasticsearch-api", Layer: plugin.LayerAPI, Watches: "Elasticsearch REST API operations deprecated, and deprecated operations removed, between released minor versions of the spec",
				URLs:     []string{"https://github.com/elastic/elasticsearch-specification"},
				Packages: []string{"pypi:elasticsearch", "npm:@elastic/elasticsearch"},
				Plugin:   openapi.Plugin{Adapter: Elasticsearch{}, Diff: elasticsearchDiff, Keep: elasticsearchKeep, RemovalsNeedANewVersion: true},
			},
			apisurface.PyPI(PythonSurface, plugin.LayerSDK, "https://github.com/elastic/elasticsearch-py"),
		},
	})
}

// The packages whose public API the apisurface sources read.
var (
	PythonSurface = apisurface.Package{ID: "elastic/python-surface", Name: "elasticsearch", Ecosystem: "pypi", Module: "elasticsearch"}
)
