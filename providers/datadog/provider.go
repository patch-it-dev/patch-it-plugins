// Package datadog is what patch-it reads for Datadog: every source, registered
// below with its metadata, each read by its kind (kinds/<kind>). README.md
// says why each source and what it reads.
package datadog

import (
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/kinds/apisurface"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi"
)

func init() {
	plugin.RegisterProvider(plugin.Provider{
		ID: "datadog", Name: "Datadog", Category: plugin.CategoryMonitoring, Homepage: "https://www.datadoghq.com",
		Sources: []plugin.Source{
			// No Hints yet: Datadog's changelog isn't Stripe's per-parameter table
			// shape, so there's no hint.Source for it to read.
			{
				ID: "api", Layer: plugin.LayerAPI,
				Watches:  "API v2 changes between OpenAPI spec releases",
				URLs:     []string{"https://github.com/DataDog/datadog-api-client-python"},
				Packages: []string{"pypi:datadog-api-client", "npm:@datadog/datadog-api-client"},
				Plugin:   openapi.Plugin{Adapter: Datadog{}},
			},
			apisurface.PyPI(DdtracePythonSurface, plugin.LayerSDK, "https://github.com/DataDog/dd-trace-py"),
		},
	})
}

// The packages whose public API the apisurface sources read.
var (
	DdtracePythonSurface = apisurface.Package{ID: "datadog/ddtrace-python-surface", Name: "ddtrace", Ecosystem: "pypi", Module: "ddtrace"}
)
