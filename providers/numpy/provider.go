// Package numpy is what patch-it reads for NumPy: every source, registered
// below with its metadata, each read by its kind (kinds/<kind>). README.md
// says why each source and what it reads.
package numpy

import (
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/kinds/apisurface"
	"github.com/patch-it-dev/patch-it-plugins/kinds/migrationguide"
	"github.com/patch-it-dev/patch-it-plugins/kinds/migrationguide/members"
)

func init() {
	plugin.RegisterProvider(plugin.Provider{
		ID: "numpy", Name: "NumPy", Category: plugin.CategoryPackages, Homepage: "https://numpy.org",
		Sources: []plugin.Source{
			{
				ID: "numpy", Layer: plugin.LayerPackage, Watches: "Members a new major removes or renames in the main namespace, from that major's migration guide at its first tag",
				URLs:     []string{"https://numpy.org/devdocs/numpy_2_0_migration_guide.html", "https://pypi.org/project/numpy/"},
				Packages: []string{"pypi:numpy"},
				Plugin:   migrationguide.Plugin{Guide: Numpy},
			},
			apisurface.PyPI(PythonSurface, plugin.LayerPackage, "https://github.com/numpy/numpy"),
		},
	})
}

// Numpy is NumPy's migration guide to a major, "Main namespace" table
// (doc/source/numpy_2_0_migration_guide.rst at v2.0.0, checked 2026-11-04;
// no other major has one yet, checked 2026-11-20).
var Numpy = members.Guide{
	API:     "numpy/numpy",
	Package: "numpy",
	Module:  "np",
	Repo:    "numpy/numpy",
	Path:    "doc/source/numpy_{major}_0_migration_guide.rst",
	Section: "Main namespace",
	Effect:  "code using it raises AttributeError",
}

// The packages whose public API the apisurface sources read.
var (
	PythonSurface = apisurface.Package{ID: "numpy/python-surface", Name: "numpy", Ecosystem: "pypi", Module: "numpy"}
)
