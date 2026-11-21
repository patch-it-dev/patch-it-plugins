// Package supabase is what patch-it reads for Supabase: every source, registered
// below with its metadata, each read by its kind (kinds/<kind>). README.md
// says why each source and what it reads.
package supabase

import (
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/kinds/apisurface"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi"
)

func init() {
	plugin.RegisterProvider(plugin.Provider{
		ID: "supabase", Name: "Supabase", Category: plugin.CategoryDatabases, Homepage: "https://supabase.com",
		Sources: []plugin.Source{
			// No Hints: Supabase's changelog is platform news, not keyed by
			// operation. Only operation deprecations and removals: see supabase.go.
			{
				ID: "management-api", Layer: plugin.LayerAPI, Watches: "Management API operations deprecated, and deprecated operations removed, between commits of the spec",
				URLs:   []string{"https://github.com/supabase/supabase"},
				Plugin: openapi.Plugin{Adapter: Supabase{}, Diff: supabaseDiff, Keep: supabaseKeep},
			},
			apisurface.PyPI(PythonSurface, plugin.LayerSDK, "https://github.com/supabase/supabase-py"),
			apisurface.NPM(JSSurface, "https://github.com/supabase/supabase-js"),
		},
	})
}

// The packages whose public API the apisurface sources read.
var (
	JSSurface     = apisurface.NPMPackage("supabase/js-surface", "@supabase/supabase-js")
	PythonSurface = apisurface.Package{ID: "supabase/python-surface", Name: "supabase", Ecosystem: "pypi", Module: "supabase"}
)
