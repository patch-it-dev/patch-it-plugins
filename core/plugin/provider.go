package plugin

import (
	"cmp"
	"fmt"
	"net/url"
	"path"
	"reflect"
	"regexp"
	"slices"
	"strings"
)

// A provider is whose API, SDK or package it is (Stripe, OpenAI, Kubernetes),
// in one category. It has one or more sources: each one place patch-it reads
// (a spec repo, a deprecations page, a changelog), read by one kind (the
// plugins/<kind> package whose plugin runs it), at one layer of a customer's
// stack. A source's id is "<provider>/<source>": "stripe/api",
// "algolia/search-api", "openai/models".
//
// Every source is registered with its metadata, in code, by the kind's
// package (RegisterProvider). That registration is the only list of what
// patch-it watches: `go run ./cmd/list-plugins` prints it, the Release
// workflow attaches it to each release as plugins.json, and patch-it's hosted
// ingestion service serves it. There is no list to keep by hand.

// Category is what a provider is, for grouping: one per provider.
type Category string

// The categories. A new one is added here, in review.
const (
	CategoryAI             Category = "ai"
	CategoryPayments       Category = "payments"
	CategoryInfrastructure Category = "infrastructure"
	CategoryDatabases      Category = "databases"
	CategoryMonitoring     Category = "monitoring"
	CategoryAnalytics      Category = "analytics"
	CategoryAuth           Category = "auth"
	CategoryMessaging      Category = "messaging"
	CategoryHosting        Category = "hosting"
	CategorySearch         Category = "search"
	CategoryDevTools       Category = "devtools"
	CategoryCRM            Category = "crm"
	CategoryPackages       Category = "packages"
)

// Categories are every category, in display order.
func Categories() []Category {
	return []Category{
		CategoryAI, CategoryPayments, CategoryInfrastructure, CategoryDatabases,
		CategoryMonitoring, CategoryAnalytics, CategoryAuth, CategoryMessaging,
		CategoryHosting, CategorySearch, CategoryDevTools, CategoryCRM, CategoryPackages,
	}
}

// Layer is which part of a customer's stack a source covers.
type Layer string

// The layers.
const (
	LayerAPI      Layer = "api"
	LayerSDK      Layer = "sdk"
	LayerPackage  Layer = "package"
	LayerRuntime  Layer = "runtime"
	LayerCI       Layer = "ci"
	LayerAIModels Layer = "ai-models"
	LayerInfra    Layer = "infra"
)

// Ecosystems are the package registries a Source's Packages name.
func Ecosystems() []string {
	return []string{"pypi", "npm", "go", "maven", "nuget", "rubygems", "cargo", "packagist"}
}

// Layers are every layer.
func Layers() []Layer {
	return []Layer{LayerAPI, LayerSDK, LayerPackage, LayerRuntime, LayerCI, LayerAIModels, LayerInfra}
}

// Provider is one provider and the sources a kind's package registers for it.
// A provider with sources in several kinds (Anthropic: a deprecations page and
// an SDK migration guide) is registered by each, with the same ID, Name,
// Category and Homepage; the registry merges them.
type Provider struct {
	ID       string   // lowercase slug: "algolia"
	Name     string   // as the provider writes it: "Algolia"
	Category Category // what it is
	Homepage string   // the provider's own site
	Sources  []Source
}

// Source is one place a provider's changes are read from.
type Source struct {
	// ID is the source's part of its id, unique within the provider:
	// "search-api" in "algolia/search-api".
	ID string
	// Kind is the kind that reads it: the plugins/<kind> package Plugin is
	// defined in. RegisterProvider fills it in; a value set by hand must
	// match.
	Kind string
	// Layer is which part of a customer's stack it covers.
	Layer Layer
	// Watches is what changes it reports, in a few words.
	Watches string
	// URLs are the provider's own pages or files it reads, absolute; the
	// first is the one a person should look at.
	URLs []string
	// Page is the source's page in this repo: "providers/algolia.md" (why
	// this source, what it reads and ignores, its known gaps).
	Page string
	// Packages are the published packages whose changes this source
	// covers, as "<ecosystem>:<name>" ("pypi:stripe", "npm:@vercel/sdk"):
	// the package itself for a package or SDK source, the provider's
	// official SDKs for an API source. What lets a lockfile entry find its
	// sources without a table kept elsewhere. Only names checked by hand;
	// none is fine when no package maps to the source.
	Packages []string
	// Plugin runs it. Its Name is the source's id ("algolia/search-api"),
	// which every event's `api` carries.
	Plugin Plugin
}

// SourceInfo is a registered source with its provider: what the list prints.
type SourceInfo struct {
	Provider Provider // without Sources
	Source   Source
}

// FullID is "<provider>/<source>".
func (s SourceInfo) FullID() string { return s.Provider.ID + "/" + s.Source.ID }

var (
	providers = map[string]Provider{}   // by provider ID, Sources merged
	sources   = map[string]SourceInfo{} // by full ID
	slug      = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
)

// RegisterProvider registers p's sources, each under its id, with its Plugin
// (Lookup). Incomplete metadata is a programming error, as a duplicate is.
func RegisterProvider(p Provider) {
	if err := p.check(); err != nil {
		panic("plugin: " + err.Error())
	}
	for i := range p.Sources {
		p.Sources[i].Kind = KindOf(p.Sources[i].Plugin)
	}
	mu.Lock()
	defer mu.Unlock()
	if have, ok := providers[p.ID]; ok {
		if have.Name != p.Name || have.Category != p.Category || have.Homepage != p.Homepage {
			panic(fmt.Sprintf("plugin: provider %q registered twice with different metadata", p.ID))
		}
	}
	for _, s := range p.Sources {
		if _, dup := sources[p.ID+"/"+s.ID]; dup {
			panic(fmt.Sprintf("plugin: source %q is already registered", p.ID+"/"+s.ID))
		}
	}
	have := providers[p.ID]
	merged := Provider{ID: p.ID, Name: p.Name, Category: p.Category, Homepage: p.Homepage, Sources: append(have.Sources, p.Sources...)}
	providers[p.ID] = merged
	bare := merged
	bare.Sources = nil
	for _, s := range p.Sources {
		id := p.ID + "/" + s.ID
		sources[id] = SourceInfo{Provider: bare, Source: s}
		registry[id] = s.Plugin
	}
}

// check is the metadata every provider and source must have.
func (p Provider) check() error {
	switch {
	case !slug.MatchString(p.ID):
		return fmt.Errorf("provider id %q isn't a lowercase slug", p.ID)
	case p.Name == "":
		return fmt.Errorf("provider %q has no Name", p.ID)
	case !slices.Contains(Categories(), p.Category):
		return fmt.Errorf("provider %q: category %q isn't one of plugin.Categories()", p.ID, p.Category)
	case !absolute(p.Homepage):
		return fmt.Errorf("provider %q: Homepage %q isn't an absolute URL", p.ID, p.Homepage)
	case len(p.Sources) == 0:
		return fmt.Errorf("provider %q has no sources", p.ID)
	}
	seen := map[string]bool{}
	for _, s := range p.Sources {
		id := p.ID + "/" + s.ID
		switch {
		case !slug.MatchString(s.ID):
			return fmt.Errorf("source id %q isn't a lowercase slug", id)
		case seen[s.ID]:
			return fmt.Errorf("source %q is listed twice", id)
		case s.Plugin == nil:
			return fmt.Errorf("source %q has no Plugin", id)
		case s.Plugin.Name() != id:
			return fmt.Errorf("source %q: its plugin's Name is %q; a plugin is named by its source's id", id, s.Plugin.Name())
		case !slices.Contains(Layers(), s.Layer):
			return fmt.Errorf("source %q: layer %q isn't one of plugin.Layers()", id, s.Layer)
		case strings.TrimSpace(s.Watches) == "":
			return fmt.Errorf("source %q: Watches is empty", id)
		case len(s.URLs) == 0:
			return fmt.Errorf("source %q has no URLs", id)
		case !strings.HasPrefix(s.Page, "providers/") || !strings.HasSuffix(s.Page, ".md"):
			return fmt.Errorf("source %q: Page %q isn't providers/<name>.md", id, s.Page)
		case s.Kind != "" && s.Kind != KindOf(s.Plugin):
			return fmt.Errorf("source %q: Kind %q, but its plugin is in plugins/%s", id, s.Kind, KindOf(s.Plugin))
		}
		for _, u := range s.URLs {
			if !absolute(u) {
				return fmt.Errorf("source %q: URL %q isn't absolute", id, u)
			}
		}
		for _, pkg := range s.Packages {
			eco, name, ok := strings.Cut(pkg, ":")
			if !ok || !slices.Contains(Ecosystems(), eco) || name == "" || strings.ContainsAny(name, " \t\n") {
				return fmt.Errorf("source %q: package %q isn't <ecosystem>:<name>, the ecosystem one of plugin.Ecosystems()", id, pkg)
			}
		}
		seen[s.ID] = true
	}
	return nil
}

func absolute(u string) bool {
	parsed, err := url.Parse(u)
	return err == nil && (parsed.Scheme == "https" || parsed.Scheme == "http") && parsed.Host != ""
}

// KindOf is the kind a plugin is: the name of the package its type is
// defined in (plugins/openapi -> "openapi").
func KindOf(p Plugin) string {
	t := reflect.TypeOf(p)
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return path.Base(t.PkgPath())
}

// Providers are the registered providers with their sources, sorted by id
// (sources by id within each).
func Providers() []Provider {
	mu.Lock()
	defer mu.Unlock()
	out := make([]Provider, 0, len(providers))
	for _, p := range providers {
		p.Sources = slices.Clone(p.Sources)
		slices.SortFunc(p.Sources, func(a, b Source) int { return cmp.Compare(a.ID, b.ID) })
		out = append(out, p)
	}
	slices.SortFunc(out, func(a, b Provider) int { return cmp.Compare(a.ID, b.ID) })
	return out
}

// Sources are the registered sources, sorted by full id.
func Sources() []SourceInfo {
	mu.Lock()
	defer mu.Unlock()
	out := make([]SourceInfo, 0, len(sources))
	for _, s := range sources {
		out = append(out, s)
	}
	slices.SortFunc(out, func(a, b SourceInfo) int { return cmp.Compare(a.FullID(), b.FullID()) })
	return out
}

// ForPackage are the sources that cover a published package
// ("pypi:stripe"), sorted by full id: how a lockfile entry finds what to
// ask the feed for.
func ForPackage(pkg string) []SourceInfo {
	var out []SourceInfo
	for _, s := range Sources() {
		if slices.Contains(s.Source.Packages, pkg) {
			out = append(out, s)
		}
	}
	return out
}

// LookupSource finds a source by its id ("algolia/search-api").
func LookupSource(id string) (SourceInfo, bool) {
	mu.Lock()
	defer mu.Unlock()
	s, ok := sources[id]
	return s, ok
}
