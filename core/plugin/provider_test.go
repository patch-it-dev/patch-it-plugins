package plugin

import (
	"context"
	"strings"
	"testing"
)

type fake struct{ name string }

func (f fake) Name() string                                                   { return f.name }
func (fake) Read(context.Context, Env, Snapshot) (Snapshot, error)            { return Snapshot{}, nil }
func (fake) Extract(context.Context, Env, Snapshot, Snapshot) (Result, error) { return Result{}, nil }

// both is a plugin that is Versioned and Living: refused.
type both struct{ fake }

func (both) ListVersions(context.Context, Env) ([]Version, error)   { return nil, nil }
func (both) ReadAt(context.Context, Env, Version) (Snapshot, error) { return Snapshot{}, nil }

// neither only extracts: refused.
type neither struct{ name string }

func (n neither) Name() string { return n.name }
func (neither) Extract(context.Context, Env, Snapshot, Snapshot) (Result, error) {
	return Result{}, nil
}

// src is a source of the provider whose id is provider, named by its id.
func src(id, provider string) Source {
	return Source{ID: id, Layer: LayerAPI, Watches: "w", URLs: []string{"https://example.com/spec"}, Page: "providers/acme.md", Plugin: fake{provider + "/" + id}}
}

func acme(sources ...Source) Provider {
	return Provider{ID: "acme-test", Name: "Acme", Category: CategoryPayments, Homepage: "https://example.com", Sources: sources}
}

func TestRegisterProviderRegistersEachSourceByItsID(t *testing.T) {
	RegisterProvider(acme(src("api", "acme-test")))
	RegisterProvider(acme(src("sdk", "acme-test"))) // a second kind adds a source
	s, ok := LookupSource("acme-test/api")
	if !ok || s.FullID() != "acme-test/api" || s.Source.Kind != "plugin" {
		t.Errorf("LookupSource = %+v, %v", s, ok)
	}
	if p, ok := Lookup("acme-test/api"); !ok || p.Name() != "acme-test/api" {
		t.Errorf("Lookup = %v, %v", p, ok)
	}
	if _, ok := Lookup("acme-test-api"); ok {
		t.Error("found a source by a name that isn't its id")
	}
	var got []string
	for _, p := range Providers() {
		if p.ID == "acme-test" {
			for _, s := range p.Sources {
				got = append(got, s.ID)
			}
		}
	}
	if strings.Join(got, ",") != "api,sdk" {
		t.Errorf("acme-test's sources = %v", got)
	}
}

func TestRegisterProviderRefusesIncompleteMetadata(t *testing.T) {
	for name, p := range map[string]Provider{
		"named by another id": acme(func() Source {
			s := src("named", "acme-test")
			s.Plugin = fake{"acme-test-named"}
			return s
		}()),
		"no category":  {ID: "x1", Name: "X", Homepage: "https://x.example", Sources: []Source{src("api", "x1")}},
		"no sources":   {ID: "x2", Name: "X", Category: CategoryAI, Homepage: "https://x.example"},
		"bad homepage": {ID: "x3", Name: "X", Category: CategoryAI, Homepage: "x.example", Sources: []Source{src("api", "x3")}},
		"no watches": {ID: "x4", Name: "X", Category: CategoryAI, Homepage: "https://x.example", Sources: []Source{func() Source {
			s := src("api", "x4")
			s.Watches = ""
			return s
		}()}},
		"bad layer": {ID: "x5", Name: "X", Category: CategoryAI, Homepage: "https://x.example", Sources: []Source{func() Source {
			s := src("api", "x5")
			s.Layer = "API"
			return s
		}()}},
		"bad package": {ID: "x7", Name: "X", Category: CategoryAI, Homepage: "https://x.example", Sources: []Source{func() Source {
			s := src("api", "x7")
			s.Packages = []string{"pip:x"}
			return s
		}()}},
		"both shapes": {ID: "x8", Name: "X", Category: CategoryAI, Homepage: "https://x.example", Sources: []Source{func() Source {
			s := src("api", "x8")
			s.Plugin = both{fake{"x8/api"}}
			return s
		}()}},
		"neither shape": {ID: "x9", Name: "X", Category: CategoryAI, Homepage: "https://x.example", Sources: []Source{func() Source {
			s := src("api", "x9")
			s.Plugin = neither{"x9/api"}
			return s
		}()}},
		"wrong kind": {ID: "x6", Name: "X", Category: CategoryAI, Homepage: "https://x.example", Sources: []Source{func() Source {
			s := src("api", "x6")
			s.Kind = "openapi"
			return s
		}()}},
	} {
		if err := p.check(); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestRegisterProviderRefusesConflictingProvider(t *testing.T) {
	RegisterProvider(acme(src("one", "acme-test")))
	defer func() {
		if recover() == nil {
			t.Error("a provider registered again with another name didn't panic")
		}
	}()
	p := acme(src("two", "acme-test"))
	p.Name = "Acme Corp"
	RegisterProvider(p)
}

// A package names its sources.
func TestForPackage(t *testing.T) {
	s := src("pkg", "acme-test")
	s.Packages = []string{"pypi:acme-test", "npm:@acme/test"}
	RegisterProvider(acme(s))
	got := ForPackage("npm:@acme/test")
	if len(got) != 1 || got[0].FullID() != "acme-test/pkg" {
		t.Errorf("ForPackage = %+v", got)
	}
	if got := ForPackage("pypi:nobody"); len(got) != 0 {
		t.Errorf("ForPackage(pypi:nobody) = %+v", got)
	}
}
