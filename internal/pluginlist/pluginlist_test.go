package pluginlist

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	_ "github.com/patch-it-dev/patch-it-plugins/plugins/all"
)

const root = "../.."

// The list is the registry: every registered source once, under its provider,
// with its old slug, and the counts agree.
func TestBuildListsEveryRegisteredSource(t *testing.T) {
	f := Build("")
	if f.Version != "main" {
		t.Errorf("version %q, want main", f.Version)
	}
	seen := map[string]bool{}
	n := 0
	for _, p := range f.List {
		for _, s := range p.Sources {
			n++
			if !strings.HasPrefix(s.ID, p.ID+"/") {
				t.Errorf("%s is listed under %s", s.ID, p.ID)
			}
			if !strings.HasPrefix(s.Page, Repo+"/blob/main/providers/") {
				t.Errorf("%s: page %s", s.ID, s.Page)
			}
			seen[s.Alias] = true
		}
	}
	if n != f.Sources || len(f.List) != f.Providers || n != len(plugin.Sources()) {
		t.Errorf("counts: %d sources listed, source_count %d, %d registered; %d providers, provider_count %d", n, f.Sources, len(plugin.Sources()), len(f.List), f.Providers)
	}
	for _, name := range plugin.Names() {
		if !seen[name] {
			t.Errorf("registered plugin %s isn't listed", name)
		}
	}
}

// One provider, several sources: Algolia's three APIs.
func TestAlgoliaIsOneProviderWithThreeSources(t *testing.T) {
	b, err := Marshal(Build("v1.2.3"))
	if err != nil {
		t.Fatal(err)
	}
	var f File
	if err := json.Unmarshal(b, &f); err != nil {
		t.Fatal(err)
	}
	for _, p := range f.List {
		if p.ID != "algolia" {
			continue
		}
		var ids []string
		for _, s := range p.Sources {
			ids = append(ids, s.ID+"="+s.Alias)
			if !strings.Contains(s.Page, "/blob/v1.2.3/") {
				t.Errorf("%s: page %s isn't at the release", s.ID, s.Page)
			}
		}
		if got := strings.Join(ids, " "); got != "algolia/ingestion-api=algolia-ingestion algolia/recommend-api=algolia-recommend algolia/search-api=algolia-search" {
			t.Errorf("algolia's sources: %s", got)
		}
		return
	}
	t.Error("no algolia provider")
}
