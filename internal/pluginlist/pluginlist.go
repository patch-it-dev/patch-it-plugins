// Package pluginlist joins the plugin registry with the table in
// docs/plugins.md: the registry says which plugins exist and which format
// plugin each is built on, the table says who the provider is, what it
// watches, its source and its status. List checks that the two agree, so a
// plugin missing from the table (AGENTS.md rule 1) fails `make check`.
package pluginlist

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
)

// Table is the plugin list, relative to the repo root.
const Table = "docs/plugins.md"

// JSON is the generated list, relative to the repo root.
const JSON = "docs/plugins.json"

// The markers around the table in Table.
const (
	StartMarker = "<!-- plugins:start -->"
	EndMarker   = "<!-- plugins:end -->"
)

// Row is one row of the table, as written.
type Row struct {
	ID       string // the plugin id, without the backticks and link
	Page     string // the provider page it links to: "algolia" for providers/algolia.md
	Provider string
	Layer    string // as written: "API", "AI models"
	Watches  string
	Format   string
	Source   string // the cell, as written
	Status   string // the cell, as written: "live", "paused: why"
}

// Plugin is one entry of docs/plugins.json.
type Plugin struct {
	ID           string `json:"id"`
	Provider     string `json:"provider"`
	Layer        string `json:"layer"` // api, sdk, package, runtime, ci or ai-models
	Watches      string `json:"watches"`
	Format       string `json:"format"`
	SourceURL    string `json:"source_url"`
	Status       string `json:"status"`                // live, paused or planned
	StatusNote   string `json:"status_note,omitempty"` // why, for paused
	ProviderPage string `json:"provider_page"`         // relative to the repo root
}

// File is docs/plugins.json.
type File struct {
	Comment string   `json:"_comment"`
	Schema  string   `json:"$schema"`
	Count   int      `json:"count"`
	Plugins []Plugin `json:"plugins"`
}

var (
	cellID   = regexp.MustCompile("^\\[`([a-z0-9][a-z0-9.-]*)`\\]\\(https://github\\.com/patch-it-dev/patch-it-plugins/blob/main/providers/([^)]+)\\.md\\)$")
	cellFmt  = regexp.MustCompile("^`([a-z]+)`$")
	link     = regexp.MustCompile(`\]\((https?://[^)\s]+)\)`)
	statuses = []string{"live", "paused", "planned"}
)

// Layers are the kinds of dependency a plugin covers: the table's word to
// the JSON's.
var Layers = map[string]string{
	"API":       "api",
	"SDK":       "sdk",
	"package":   "package",
	"runtime":   "runtime",
	"CI":        "ci",
	"AI models": "ai-models",
}

// Statuses are the values of a plugin's status.
func Statuses() []string { return slices.Clone(statuses) }

// ReadTable reads the rows between the markers of the plugin list.
func ReadTable(md []byte) ([]Row, error) {
	start := bytes.Index(md, []byte(StartMarker))
	end := bytes.Index(md, []byte(EndMarker))
	if start < 0 || end < start {
		return nil, fmt.Errorf("%s: no %s ... %s block", Table, StartMarker, EndMarker)
	}
	var rows []Row
	lines := strings.Split(string(md[start+len(StartMarker):end]), "\n")
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if !strings.HasPrefix(l, "|") || strings.HasPrefix(l, "|---") || strings.HasPrefix(l, "| Plugin") {
			continue
		}
		cells := splitRow(l)
		if len(cells) != 7 {
			return nil, fmt.Errorf("%s: row %q has %d cells, want 7", Table, l, len(cells))
		}
		m := cellID.FindStringSubmatch(cells[0])
		if m == nil {
			return nil, fmt.Errorf("%s: plugin cell %q is not [`id`](https://github.com/patch-it-dev/patch-it-plugins/blob/main/providers/<id>.md)", Table, cells[0])
		}
		f := cellFmt.FindStringSubmatch(cells[4])
		if f == nil {
			return nil, fmt.Errorf("%s: %s: format cell %q is not `format`", Table, m[1], cells[4])
		}
		rows = append(rows, Row{ID: m[1], Page: m[2], Provider: cells[1], Layer: cells[2], Watches: cells[3], Format: f[1], Source: cells[5], Status: cells[6]})
	}
	return rows, nil
}

// splitRow splits "| a | b |" into its cells. A cell holding a pipe would
// have to escape it (\|); none does, and the docs site copies the table as is.
func splitRow(l string) []string {
	l = strings.TrimSuffix(strings.TrimPrefix(l, "|"), "|")
	var cells []string
	for _, c := range strings.Split(l, "|") {
		cells = append(cells, strings.TrimSpace(c))
	}
	return cells
}

// Format is the format plugin a registered plugin is built on: the name of
// the package under plugins/ its type is defined in.
func Format(p plugin.Plugin) string {
	t := reflect.TypeOf(p)
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return path.Base(t.PkgPath())
}

// List joins the registered plugins with the table under root and checks
// they agree: every registered plugin has a row with its format, every row is
// registered (unless it's planned), and every plugin has a provider page.
// The plugins come out sorted by id.
func List(root string) ([]Plugin, error) {
	md, err := os.ReadFile(filepath.Join(root, Table))
	if err != nil {
		return nil, err
	}
	rows, err := ReadTable(md)
	if err != nil {
		return nil, err
	}
	var errs []error
	byID := map[string]Row{}
	for _, r := range rows {
		if _, dup := byID[r.ID]; dup {
			errs = append(errs, fmt.Errorf("%s: %s has two rows", Table, r.ID))
		}
		byID[r.ID] = r
	}
	registered := map[string]string{}
	for _, name := range plugin.Names() {
		p, _ := plugin.Lookup(name)
		registered[name] = Format(p)
		if _, ok := byID[name]; !ok {
			errs = append(errs, fmt.Errorf("%s is registered but has no row in %s: add one (AGENTS.md rule 1), then run `make plugins-json`", name, Table))
		}
	}
	var out []Plugin
	for _, r := range rows {
		status, note, _ := strings.Cut(r.Status, ":")
		status, note = strings.TrimSpace(status), strings.TrimSpace(note)
		if !slices.Contains(statuses, status) {
			errs = append(errs, fmt.Errorf("%s: %s: status %q is not live, paused or planned", Table, r.ID, r.Status))
		}
		layer, known := Layers[r.Layer]
		if !known {
			errs = append(errs, fmt.Errorf("%s: %s: layer %q is not one of API, SDK, package, runtime, CI, AI models", Table, r.ID, r.Layer))
		}
		format, ok := registered[r.ID]
		switch {
		case !ok && status != "planned":
			errs = append(errs, fmt.Errorf("%s: %s has a row but no plugin registers it: mark it planned, or remove the row", Table, r.ID))
		case ok && format != r.Format:
			errs = append(errs, fmt.Errorf("%s: %s says format %q, but the plugin is in plugins/%s", Table, r.ID, r.Format, format))
		}
		// The row says what the source's registration says (moving to the
		// registration only: until then, both).
		if s, ok := plugin.LookupSource(r.ID); ok {
			switch {
			case s.Provider.Name != r.Provider:
				errs = append(errs, fmt.Errorf("%s: %s: provider %q, but its registration says %q", Table, r.ID, r.Provider, s.Provider.Name))
			case string(s.Source.Layer) != layer:
				errs = append(errs, fmt.Errorf("%s: %s: layer %q, but its registration says %q", Table, r.ID, layer, s.Source.Layer))
			case s.Source.Watches != r.Watches:
				errs = append(errs, fmt.Errorf("%s: %s: what it watches differs from its registration's Watches", Table, r.ID))
			case "providers/"+r.Page+".md" != s.Source.Page:
				errs = append(errs, fmt.Errorf("%s: %s: links providers/%s.md, but its registration's Page is %s", Table, r.ID, r.Page, s.Source.Page))
			}
		}
		// One page per provider: the algolia-* plugins share providers/algolia.md.
		page := "providers/" + r.Page + ".md"
		if _, err := os.Stat(filepath.Join(root, page)); err != nil {
			errs = append(errs, fmt.Errorf("%s: %s links to %s, which doesn't exist (copy providers/_template.md)", Table, r.ID, page))
		}
		src := link.FindStringSubmatch(r.Source)
		if src == nil {
			errs = append(errs, fmt.Errorf("%s: %s: the source cell has no absolute link", Table, r.ID))
			src = []string{"", ""}
		}
		out = append(out, Plugin{
			ID: r.ID, Provider: r.Provider, Layer: layer, Watches: r.Watches, Format: r.Format,
			SourceURL: src[1], Status: status, StatusNote: note, ProviderPage: page,
		})
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	slices.SortFunc(out, func(a, b Plugin) int { return strings.Compare(a.ID, b.ID) })
	return out, nil
}

// Readme is the README, relative to the repo root: its plugins badge is a
// static count that `make plugins-json` keeps right.
const Readme = "README.md"

var badge = regexp.MustCompile(`(img\.shields\.io/badge/plugins-)(\d+)(-)`)

// Badge is readme with its plugins badge set to n. ok is false when it has none.
func Badge(readme []byte, n int) (out []byte, ok bool) {
	if !badge.Match(readme) {
		return readme, false
	}
	return badge.ReplaceAll(readme, []byte("${1}"+strconv.Itoa(n)+"${3}")), true
}

// Marshal is docs/plugins.json for plugins, ending in a newline.
func Marshal(plugins []Plugin) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	err := enc.Encode(File{
		Comment: "Generated from the plugin registry and docs/plugins.md by `make plugins-json` (go run ./cmd/list-plugins). Don't edit by hand.",
		Schema:  "./plugins.schema.json",
		Count:   len(plugins),
		Plugins: plugins,
	})
	return buf.Bytes(), err
}
