package apisurface

import (
	"strings"

	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
)

// PyPI is the source a PyPI package's public API is, its metadata read off
// pkg: repo is the package's source repository. A package's Module is the
// top-level directory its wheel installs (checked by hand on the package's
// newest wheel).
func PyPI(pkg Package, layer plugin.Layer, repo string) plugin.Source {
	_, id, _ := strings.Cut(pkg.ID, "/")
	return plugin.Source{
		ID: id, Layer: layer,
		Watches:  "Classes, functions, methods and parameters removed or renamed in the public API of the " + pkg.Name + " Python package, release by release, read from each release's wheel",
		URLs:     []string{"https://pypi.org/project/" + pkg.Name + "/", repo},
		Packages: []string{"pypi:" + pkg.Name},
		Plugin:   Plugin{Package: pkg},
	}
}

// NPMPackage is the npm package name, read from its TypeScript declarations,
// for the source id. An npm package's Module is its own name: symbols are
// named from the import path code writes
// ("@anthropic-ai/sdk/resources/messages.Messages.create").
func NPMPackage(id, name string) Package {
	return Package{ID: id, Name: name, Ecosystem: "npm", Module: name}
}

// NPM is the source an npm package's public API is, its metadata read off
// pkg: repo is the package's source repository, also any other packages
// ("npm:redis") whose API is pkg's.
func NPM(pkg Package, repo string, also ...string) plugin.Source {
	_, id, _ := strings.Cut(pkg.ID, "/")
	return plugin.Source{
		ID: id, Layer: plugin.LayerSDK,
		Watches:  "Exported classes, functions, methods, properties, types and parameters removed, or parameters no longer optional, in the public API of the " + pkg.Name + " npm package, release by release, read from each release's TypeScript declarations",
		URLs:     []string{"https://www.npmjs.com/package/" + pkg.Name, repo},
		Packages: append([]string{"npm:" + pkg.Name}, also...),
		Plugin:   Plugin{Package: pkg},
	}
}
