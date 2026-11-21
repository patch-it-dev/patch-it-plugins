package changelog

import "github.com/patch-it-dev/patch-it-plugins/core/plugin"

// A provider's official SDKs (providers/<provider>/provider.go) are each
// read from its own changelog at every release (or, with Notes, from its
// release notes on GitHub). Every one was checked by hand: the file's path,
// the tag each release is cut at, and a release whose section this reader
// turns into events (each provider's test pins it).

// SDK is the source an SDK's changelog is, its metadata read off f: id is
// the source's part of its id ("python" in "stripe/python").
func SDK(id string, f File, packages ...string) plugin.Source {
	watches := "Names removed, renamed or deprecated in each release of " + f.Package + ", from the release's section of " + f.Path + " at its tag"
	first := "https://github.com/" + f.Repo + "/blob/HEAD/" + f.Path
	if f.Notes {
		watches = "Names removed, renamed or deprecated in each release of " + f.Package + ", from the release's notes on GitHub"
		first = "https://github.com/" + f.Repo + "/releases"
	}
	return plugin.Source{
		ID: id, Layer: layerOf(f), Watches: watches,
		URLs: []string{first}, Packages: packages, Plugin: Plugin{File: f},
	}
}

func layerOf(f File) plugin.Layer {
	if f.Ecosystem == "terraform" {
		return plugin.LayerInfra
	}
	return plugin.LayerSDK
}
