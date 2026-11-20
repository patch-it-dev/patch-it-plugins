package plugin

import (
	"context"
	"strings"
	"testing"
)

type fake struct{ name string }

func (f fake) Name() string                           { return f.name }
func (fake) Run(context.Context, Env) (Result, error) { return Result{}, nil }

func src(id, slug string) Source {
	return Source{ID: id, Layer: LayerAPI, Watches: "w", URLs: []string{"https://example.com/spec"}, Page: "providers/acme.md", Plugin: fake{slug}}
}

func acme(sources ...Source) Provider {
	return Provider{ID: "acme-test", Name: "Acme", Category: CategoryPayments, Homepage: "https://example.com", Sources: sources}
}

func TestRegisterProviderResolvesBothIDs(t *testing.T) {
	RegisterProvider(acme(src("api", "acme-test-api")))
	RegisterProvider(acme(src("sdk", "acme-test-sdk"))) // a second kind adds a source
	for _, id := range []string{"acme-test/api", "acme-test-api"} {
		s, ok := LookupSource(id)
		if !ok || s.FullID() != "acme-test/api" || s.Alias() != "acme-test-api" || s.Source.Kind != "plugin" {
			t.Errorf("LookupSource(%q) = %+v, %v", id, s, ok)
		}
		if p, ok := Lookup(id); !ok || p.Name() != "acme-test-api" {
			t.Errorf("Lookup(%q) = %v, %v", id, p, ok)
		}
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
	RegisterProvider(acme(src("one", "acme-test-one")))
	defer func() {
		if recover() == nil {
			t.Error("a provider registered again with another name didn't panic")
		}
	}()
	p := acme(src("two", "acme-test-two"))
	p.Name = "Acme Corp"
	RegisterProvider(p)
}

// A kind merged into another still names it: a source registered with the
// old name passes the check, and the old name reads as the new one.
func TestMergedKindsKeepWorking(t *testing.T) {
	for old, now := range mergedKinds {
		if got := CanonicalKind(old); got != now {
			t.Errorf("CanonicalKind(%q) = %q, want %q", old, got, now)
		}
	}
	if got := CanonicalKind("openapi"); got != "openapi" {
		t.Errorf("CanonicalKind(openapi) = %q", got)
	}
}
