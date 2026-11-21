// Package textual is what patch-it reads for Textual: every source, registered
// below with its metadata, each read by its kind (kinds/<kind>). README.md
// says why each source and what it reads.
package textual

import (
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/kinds/apisurface"
	"github.com/patch-it-dev/patch-it-plugins/kinds/changelog"
)

func init() {
	plugin.RegisterProvider(plugin.Provider{
		ID: "textual", Name: "Textual", Category: plugin.CategoryPackages, Homepage: "https://textual.textualize.io",
		Sources: []plugin.Source{
			{
				ID: "textual", Layer: plugin.LayerPackage, Watches: "Names removed, renamed or deprecated in each release, from the release's section of CHANGELOG.md at its tag",
				URLs:     []string{"https://github.com/Textualize/textual/blob/main/CHANGELOG.md", "https://pypi.org/project/textual/"},
				Packages: []string{"pypi:textual"},
				Plugin:   changelog.Plugin{File: Textual},
			},
			apisurface.PyPI(PythonSurface, plugin.LayerPackage, "https://github.com/Textualize/textual"),
		},
	})
}

// Textual is Textual's Keep a Changelog CHANGELOG.md (checked 2026-11-14 at
// v0.86.0: "## [0.86.0] - 2024-11-16", "### Removed", and "Breaking change:"
// bullets under "### Changed"), tagged v<version>.
var Textual = changelog.File{
	API: "textual/textual", Package: "textual", Ecosystem: "pypi", Module: "textual",
	Repo: "Textualize/textual", TagPrefix: "v", Path: "CHANGELOG.md",
}

// The packages whose public API the apisurface sources read.
var (
	PythonSurface = apisurface.Package{ID: "textual/python-surface", Name: "textual", Ecosystem: "pypi", Module: "textual"}
)
