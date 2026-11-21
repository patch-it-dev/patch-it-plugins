// Package urllib3 is what patch-it reads for urllib3: every source, registered
// below with its metadata, each read by its kind (kinds/<kind>). README.md
// says why each source and what it reads.
package urllib3

import (
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/kinds/apisurface"
	"github.com/patch-it-dev/patch-it-plugins/kinds/changelog"
)

func init() {
	plugin.RegisterProvider(plugin.Provider{
		ID: "urllib3", Name: "urllib3", Category: plugin.CategoryPackages, Homepage: "https://urllib3.readthedocs.io",
		Sources: []plugin.Source{
			{
				ID: "urllib3", Layer: plugin.LayerPackage,
				Watches:  "Names removed or deprecated in each release, from the release's section of CHANGES.rst at its tag",
				URLs:     []string{"https://github.com/urllib3/urllib3/blob/main/CHANGES.rst", "https://pypi.org/project/urllib3/"},
				Packages: []string{"pypi:urllib3"},
				Plugin:   changelog.Plugin{File: Urllib3},
			},
			apisurface.PyPI(PythonSurface, plugin.LayerPackage, "https://github.com/urllib3/urllib3"),
		},
	})
}

// Urllib3 is urllib3's CHANGES.rst (checked 2026-11-14 at 2.0.0: "2.0.0
// (2023-04-26)" over "=====", "Removed" and "Deprecated" over "-----"),
// tagged <version>.
var Urllib3 = changelog.File{
	API: "urllib3/urllib3", Package: "urllib3", Ecosystem: "pypi", Module: "urllib3",
	Repo: "urllib3/urllib3", Path: "CHANGES.rst",
}

// The packages whose public API the apisurface sources read.
var (
	PythonSurface = apisurface.Package{ID: "urllib3/python-surface", Name: "urllib3", Ecosystem: "pypi", Module: "urllib3"}
)
