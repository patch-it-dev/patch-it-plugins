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
func (f fake) WithShape(Shape) Plugin                                         { return f }

// unguarded is a living plugin that doesn't run the shape guard: refused.
type unguarded struct{ name string }

func (u unguarded) Name() string                                        { return u.name }
func (unguarded) Read(context.Context, Env, Snapshot) (Snapshot, error) { return Snapshot{}, nil }
func (unguarded) Extract(context.Context, Env, Snapshot, Snapshot) (Result, error) {
	return Result{}, nil
}

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
	return Source{ID: id, Layer: LayerAPI, Watches: "w", URLs: []string{"https://example.com/spec"}, Plugin: fake{provider + "/" + id}}
}

func acme(sources ...Source) Provider { return acmeAs("acme-test", sources...) }

func acmeAs(id string, sources ...Source) Provider {
	return Provider{ID: id, Name: "Acme", Category: CategoryPayments, Homepage: "https://example.com", Sources: sources}
}

func TestRegisterProviderRegistersEachSourceByItsID(t *testing.T) {
	RegisterProvider(acme(src("api", "acme-test"), src("sdk", "acme-test")))
	s, ok := LookupSource("acme-test/api")
	if !ok || s.FullID() != "acme-test/api" || s.Source.Kind != "plugin" || s.Source.Page != "providers/acme-test/README.md" {
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
		"no shape guard": {ID: "x11", Name: "X", Category: CategoryAI, Homepage: "https://x.example", Sources: []Source{func() Source {
			s := src("api", "x11")
			s.Plugin = unguarded{"x11/api"}
			return s
		}()}},
		"undated anchor": {ID: "x12", Name: "X", Category: CategoryAI, Homepage: "https://x.example", Sources: []Source{func() Source {
			s := src("api", "x12")
			s.Shape.Anchors = []Anchor{{Entry: "gpt-4"}}
			return s
		}()}},
		"three anchors": {ID: "x13", Name: "X", Category: CategoryAI, Homepage: "https://x.example", Sources: []Source{func() Source {
			s := src("api", "x13")
			s.Shape.Anchors = []Anchor{{"a", "2026-11-21"}, {"b", "2026-11-21"}, {"c", "2026-11-21"}}
			return s
		}()}},
		"ratio above 1": {ID: "x14", Name: "X", Category: CategoryAI, Homepage: "https://x.example", Sources: []Source{func() Source {
			s := src("api", "x14")
			s.Shape.MinRatio = 1.5
			return s
		}()}},
		"wrong page": {ID: "x10", Name: "X", Category: CategoryAI, Homepage: "https://x.example", Sources: []Source{func() Source {
			s := src("api", "x10")
			s.Page = "providers/x10.md" // not providers/x10/README.md
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

// A provider registers every source at once: a second registration, even of
// other sources, is a programming error.
func TestRegisterProviderRefusesASecondRegistration(t *testing.T) {
	RegisterProvider(acmeAs("acme-twice", src("one", "acme-twice")))
	defer func() {
		if recover() == nil {
			t.Error("a provider registered twice didn't panic")
		}
	}()
	RegisterProvider(acmeAs("acme-twice", src("two", "acme-twice")))
}

// A package names its sources.
func TestForPackage(t *testing.T) {
	s := src("pkg", "acme-pkg")
	s.Packages = []string{"pypi:acme-test", "npm:@acme/test"}
	RegisterProvider(acmeAs("acme-pkg", s))
	got := ForPackage("npm:@acme/test")
	if len(got) != 1 || got[0].FullID() != "acme-pkg/pkg" {
		t.Errorf("ForPackage = %+v", got)
	}
	if got := ForPackage("pypi:nobody"); len(got) != 0 {
		t.Errorf("ForPackage(pypi:nobody) = %+v", got)
	}
}
