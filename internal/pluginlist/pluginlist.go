// Package pluginlist is the list of what patch-it watches, as JSON, built at
// run time from the plugin registry: every registered provider with its
// sources and their metadata. Nothing is kept by hand and nothing is
// committed: `go run ./cmd/list-plugins` prints it, and the Release workflow
// attaches it to each release as plugins.json (docs/plugins.schema.json
// describes it).
package pluginlist

import (
	"bytes"
	"encoding/json"

	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
)

// Repo is where a source's page is published.
const Repo = "https://github.com/patch-it-dev/patch-it-plugins"

// File is plugins.json.
type File struct {
	Comment    string     `json:"_comment"`
	Schema     string     `json:"$schema"`
	Version    string     `json:"version"` // the release it was built from; "main" between releases
	Providers  int        `json:"provider_count"`
	Sources    int        `json:"source_count"`
	Categories []string   `json:"categories"` // every category, in display order
	List       []Provider `json:"providers"`  // sorted by id
}

// Provider is one provider and its sources.
type Provider struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Category string   `json:"category"`
	Homepage string   `json:"homepage"`
	Sources  []Source `json:"sources"` // sorted by id
}

// Source is one source.
type Source struct {
	ID      string   `json:"id"`    // provider/source
	Alias   string   `json:"alias"` // the old slug: --api and every event's api until the contract switches to source ids
	Kind    string   `json:"kind"`
	Layer   string   `json:"layer"`
	Watches string   `json:"watches"`
	URLs    []string `json:"urls"`
	Page    string   `json:"page"` // absolute, at Version
}

// Build is the list of everything registered, with pages linked at ref (a
// release tag, or "main").
func Build(ref string) File {
	if ref == "" {
		ref = "main"
	}
	f := File{
		Comment: "Generated from the plugin registry by `go run ./cmd/list-plugins`. Not committed: each release carries it as an asset.",
		Schema:  Repo + "/blob/" + ref + "/docs/plugins.schema.json",
		Version: ref,
	}
	for _, c := range plugin.Categories() {
		f.Categories = append(f.Categories, string(c))
	}
	for _, p := range plugin.Providers() {
		out := Provider{ID: p.ID, Name: p.Name, Category: string(p.Category), Homepage: p.Homepage}
		for _, s := range p.Sources {
			out.Sources = append(out.Sources, Source{
				ID: p.ID + "/" + s.ID, Alias: s.Plugin.Name(), Kind: s.Kind, Layer: string(s.Layer),
				Watches: s.Watches, URLs: s.URLs, Page: Repo + "/blob/" + ref + "/" + s.Page,
			})
		}
		f.Sources += len(out.Sources)
		f.List = append(f.List, out)
	}
	f.Providers = len(f.List)
	return f
}

// Marshal is f as indented JSON, ending in a newline.
func Marshal(f File) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	err := enc.Encode(f)
	return buf.Bytes(), err
}
