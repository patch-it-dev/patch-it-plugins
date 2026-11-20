// Command new-plugin scaffolds a source of a provider on one of the kinds,
// so a contributor only fills in what is specific to it. Run it from the
// repo root (or `make new-plugin ARGS='...'`):
//
//	go run ./cmd/new-plugin --provider acme --name Acme --category devtools \
//	    --homepage https://acme.example --source api --kind openapi \
//	    --url https://github.com/acme/openapi --package pypi:acme
//
// A flag left out is asked for when the command runs in a terminal; a
// second source of a registered provider needs only --provider, --source
// and --kind (the provider's name, category and homepage come from the
// registry). Run it once per source.
//
// It writes the source's definition in the kind's package (or the reader
// package the kind uses: plugins/openapi/adapter/, plugins/notices/retire/,
// plugins/notices/feed/, plugins/migrationguide/sdkguide/,
// plugins/migrationguide/members/), with the version wiring the kind needs
// from it (an OpenAPI adapter's Versions; every other kind lists, reads and
// extracts versions itself), adds the plugin.RegisterProvider block (the
// provider and the source with their metadata) to the kind's init, a test
// (registered, golden, drop) over placeholder sources with an empty golden
// file, a row in that testdata/README.md, providers/<page>.md from
// providers/_template.md, and the kind's import in plugins/all when it isn't
// there. `go test ./...` passes right after; every place to fill in says
// TODO(new-plugin).
//
// It changes nothing when the provider/source id or the first name is
// taken, or a file it would write exists. The old flags (--api, --provider
// as the name, --format, --source as the URL) still work for one release.
package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"text/template"
	"time"
	"unicode"

	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	_ "github.com/patch-it-dev/patch-it-plugins/plugins/all"
)

// Options are the command's flags.
type Options struct {
	Root string // the repo root

	// The provider.
	ProviderID string // lowercase slug: "acme"
	Provider   string // its name, as it writes it: "Acme"
	Category   string // one of plugin.Categories(); a registered provider's is kept
	Homepage   string // its own site

	// The source.
	SourceID string   // unique within the provider: "api", "python-sdk"
	Kind     string   // the kind that reads it: plugins/<Kind>
	Reader   string   // for a kind with several readers: "feed" for notices; "" is the kind's default
	Layer    string   // the source's layer (plugin.Layers()); "" is the template's default
	URLs     []string // the provider's pages or files it reads; none is a placeholder
	Packages []string // the packages it covers, "pypi:acme"
	Watches  string   // what the source watches, in a few words
	Page     string   // the provider page, providers/<Page>.md; "" is the first name. An existing page is shared, not rewritten
	API      string   // its first name, the old slug events carry in api; "" is <provider>-<source> (or the provider's id when they're the same)

	// Deprecated: Format is a template ("openapi", "notices/feed") or an old
	// format name ("feed"); Kind and Reader replace it. Source is the first
	// URL (URLs replaces it).
	Format string
	Source string

	Today time.Time
	// Go runs the go command afterwards: the golden file from the placeholders
	// (go test -update).
	Go bool
}

// list is a repeatable flag.
type list []string

func (l *list) String() string     { return strings.Join(*l, ",") }
func (l *list) Set(v string) error { *l = append(*l, v); return nil }

func main() {
	var o Options
	var urls, packages list
	var provider, source string
	flag.StringVar(&o.Root, "root", ".", "the repo root")
	flag.StringVar(&provider, "provider", "", "the provider's id, e.g. acme (with --format: its name, the old meaning)")
	flag.StringVar(&o.Provider, "name", "", "the provider's name as it writes it, e.g. Acme (a registered provider's is kept)")
	flag.StringVar(&o.Category, "category", "", "the provider's category: "+strings.Join(categories(), ", "))
	flag.StringVar(&o.Homepage, "homepage", "", "the provider's own site, https://...")
	flag.StringVar(&source, "source", "", "the source's id within the provider, e.g. api or python-sdk (with --format: its URL, the old meaning)")
	flag.StringVar(&o.Kind, "kind", "", "the kind that reads it: "+strings.Join(kindNames(), ", "))
	flag.StringVar(&o.Reader, "reader", "", "the kind's reader, for a kind with several: "+readerHelp())
	flag.StringVar(&o.Layer, "layer", "", "the source's layer: api, sdk, package, runtime, ci, ai-models or infra (default: the kind's)")
	flag.Var(&urls, "url", "a page or file the source reads, https://... (repeatable; default: a placeholder to fill in)")
	flag.Var(&packages, "package", "a package the source covers, ecosystem:name, e.g. pypi:acme or npm:@acme/sdk (repeatable)")
	flag.StringVar(&o.Watches, "watches", "", "what it watches, in a few words (default: a placeholder)")
	flag.StringVar(&o.Page, "page", "", "the provider page, providers/<page>.md (default: the first name); a second source of a provider may share the existing page")
	flag.StringVar(&o.API, "api", "", "the source's first name, the slug events carry in api (default: <provider>-<source>)")
	flag.StringVar(&o.Format, "format", "", "deprecated, for one release: the old format names ("+strings.Join(legacyNames(), ", ")+"), with --api, --provider <Name> and --source <URL>")
	flag.Parse()
	o.URLs, o.Packages = urls, packages
	if o.Format != "" {
		o.Provider, o.Source = provider, source
	} else {
		o.ProviderID, o.SourceID = provider, source
		if term() {
			if err := ask(&o, bufio.NewReader(os.Stdin), os.Stdout); err != nil {
				fmt.Fprintln(os.Stderr, "new-plugin:", err)
				os.Exit(1)
			}
		}
	}
	o.Today = time.Now().UTC()
	o.Go = true
	if err := Run(o, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "new-plugin:", err)
		os.Exit(1)
	}
}

// formatNames are the templates: a kind, or kind/reader.
func formatNames() []string {
	var names []string
	for n := range formats {
		names = append(names, n)
	}
	slices.Sort(names)
	return names
}

// kindNames are the kinds a template scaffolds.
func kindNames() []string {
	var names []string
	for n := range formats {
		k, _, _ := strings.Cut(n, "/")
		if !slices.Contains(names, k) {
			names = append(names, k)
		}
	}
	slices.Sort(names)
	return names
}

func readerHelp() string {
	var parts []string
	for _, k := range kindNames() {
		var rs []string
		for _, n := range formatNames() {
			if kind, r, ok := strings.Cut(n, "/"); ok && kind == k {
				rs = append(rs, r)
			}
		}
		if len(rs) > 0 {
			parts = append(parts, k+": "+strings.Join(rs, "|")+" (default "+defaultReaders[k]+")")
		}
	}
	return strings.Join(parts, "; ")
}

func legacyNames() []string {
	var names []string
	for n := range legacyFormats {
		names = append(names, n)
	}
	slices.Sort(names)
	return names
}

func categories() []string {
	var out []string
	for _, c := range plugin.Categories() {
		out = append(out, string(c))
	}
	return out
}

// template is the template o asks for: from --kind and --reader, or the
// deprecated --format.
func (o Options) template() (string, error) {
	if o.Format != "" {
		if t, ok := legacyFormats[o.Format]; ok {
			return t, nil
		}
		if _, ok := formats[o.Format]; ok {
			return o.Format, nil
		}
		return "", fmt.Errorf("--format %q: want one of %s", o.Format, strings.Join(legacyNames(), ", "))
	}
	if !slices.Contains(kindNames(), o.Kind) {
		return "", fmt.Errorf("--kind %q: want one of %s", o.Kind, strings.Join(kindNames(), ", "))
	}
	if _, ok := formats[o.Kind]; ok && o.Reader == "" {
		return o.Kind, nil
	}
	r := cmp(o.Reader, defaultReaders[o.Kind])
	if _, ok := formats[o.Kind+"/"+r]; !ok {
		return "", fmt.Errorf("--reader %q: kind %s's readers are %s", o.Reader, o.Kind, readerHelp())
	}
	return o.Kind + "/" + r, nil
}

var (
	validID   = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)
	validURL  = regexp.MustCompile(`^https://[^\s|()]+$`)
	cellSafe  = regexp.MustCompile(`^[^|\n"\\` + "`" + `]+$`)
	goosArchs = []string{"aix", "android", "darwin", "dragonfly", "freebsd", "hurd", "illumos", "ios", "js", "linux", "nacl", "netbsd", "openbsd", "plan9", "solaris", "wasip1", "windows", "zos",
		"386", "amd64", "arm", "arm64", "loong64", "mips", "mipsle", "mips64", "mips64le", "ppc64", "ppc64le", "riscv64", "s390x", "wasm", "test"}
)

// data is what the templates fill in.
type data struct {
	ID, Ident, Lower, Provider, Source, File, Format, Kind, Module, PyPI string
	Slug                                                                 string // the provider, lowercase: a model subject's provider
	SourceFile, DropFile, PluginOf, Ingest, Drop                         string
	Imports                                                              []string
}

// Run scaffolds the source, or changes nothing and says why.
func Run(o Options, out io.Writer) error {
	tmpl, err := o.template()
	if err != nil {
		return err
	}
	o.Format = tmpl
	f := formats[tmpl]
	if err := o.resolve(); err != nil {
		return err
	}
	if !validID.MatchString(o.API) || o.API == "all" {
		return fmt.Errorf("--api %q: want lowercase letters, digits and single hyphens, starting with a letter (acme, acme-models), and not \"all\"", o.API)
	}
	if o.Provider == "" || !cellSafe.MatchString(o.Provider) {
		return fmt.Errorf("--provider %q: want the provider's name, without |, quotes, backslashes or backticks", o.Provider)
	}
	if o.Layer == "" {
		o.Layer = f.layer
	}
	if !slices.Contains(plugin.Layers(), plugin.Layer(o.Layer)) {
		return fmt.Errorf("--layer %q: want one of %v", o.Layer, plugin.Layers())
	}
	if o.Source == "" && len(o.URLs) > 0 {
		o.Source = o.URLs[0]
	}
	if o.Source == "" {
		o.Source = "https://example.com/TODO-" + o.API + "-source"
	}
	if len(o.URLs) == 0 {
		o.URLs = []string{o.Source}
	}
	for _, u := range o.URLs {
		if !validURL.MatchString(u) {
			return fmt.Errorf("--url %q: want an https URL", u)
		}
	}
	if len(o.Packages) == 0 && f.kind == "apisurface" {
		o.Packages = []string{"pypi:" + o.ProviderID}
	}
	for _, p := range o.Packages {
		eco, name, ok := strings.Cut(p, ":")
		if !ok || !slices.Contains(plugin.Ecosystems(), eco) || name == "" || !cellSafe.MatchString(name) || strings.ContainsAny(name, " \t") {
			return fmt.Errorf("--package %q: want ecosystem:name, the ecosystem one of %v", p, plugin.Ecosystems())
		}
	}
	if o.Watches == "" {
		o.Watches = "TODO: what it watches, in a few words"
	} else if !cellSafe.MatchString(o.Watches) {
		return fmt.Errorf("--watches: no |, quotes, backslashes or backticks")
	}

	sharedPage := false
	if o.Page == "" {
		o.Page = o.API
	} else if !validID.MatchString(o.Page) {
		return fmt.Errorf("--page %q: want the name of a page in providers/, without .md", o.Page)
	} else if _, err := os.Stat(filepath.Join(o.Root, "providers", o.Page+".md")); err == nil {
		sharedPage = true
	}

	// Taken? Check everything before writing anything.
	if _, ok := plugin.Lookup(o.API); ok {
		return fmt.Errorf("a source named %q is already registered: pick another --api (the first name)", o.API)
	}
	if _, ok := plugin.LookupSource(o.ProviderID + "/" + o.SourceID); ok {
		return fmt.Errorf("source %s/%s is already registered: pick another --source", o.ProviderID, o.SourceID)
	}

	d := data{ID: o.API, Provider: o.Provider, Source: o.Source, Format: o.Format, Kind: f.kind, PyPI: o.API}
	for _, p := range o.Packages {
		if name, ok := strings.CutPrefix(p, "pypi:"); ok {
			d.PyPI = name
			break
		}
	}
	d.Module = strings.ReplaceAll(strings.ToLower(d.PyPI), "-", "_")
	d.Ident = ident(o.API)
	d.Lower = string(unicode.ToLower(rune(d.Ident[0]))) + d.Ident[1:]
	d.File = fileBase(o.API)
	d.Slug = strings.Trim(regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(strings.ToLower(o.Provider), "-"), "-")
	d.SourceFile = fill(f.source, d)
	d.DropFile = dropName(d.SourceFile)
	d.PluginOf = fill(f.pluginOf, d)
	d.Ingest = fill(f.ingest, d)
	d.Drop = fill(f.drop, d)
	d.Imports = append(slices.Clone(f.imports), "testing", "github.com/patch-it-dev/patch-it-plugins/core/kit/kittest")
	slices.Sort(d.Imports)

	pkgDir := "plugins/" + f.kind
	writes := map[string][]byte{}
	add := func(path string, b []byte) { writes[path] = b }

	def, err := gofmt(fill(f.def, d))
	if err != nil {
		return fmt.Errorf("the %s template: %w", o.Format, err)
	}
	add(filepath.Join(f.defDir, d.File+".go"), def)
	test, err := gofmt(fill(testFile, d))
	if err != nil {
		return fmt.Errorf("the test template: %w", err)
	}
	add(filepath.Join(pkgDir, d.File+"_test.go"), test)
	for name, content := range f.fixtures {
		add(filepath.Join(pkgDir, "testdata", fill(name, d)), []byte(fill(content, d)))
	}
	drop := f.dropFile
	if drop == "" {
		drop = f.fixtures[f.source]
	}
	add(filepath.Join(pkgDir, "testdata", d.DropFile), []byte(fill(drop, d)))
	// What the placeholders give; rewritten with -update below, and again
	// once the source is real.
	add(filepath.Join(pkgDir, "testdata", "golden", o.API+".json"), []byte("[]\n"))

	if !sharedPage {
		page, err := providerPage(o)
		if err != nil {
			return err
		}
		add(filepath.Join("providers", o.Page+".md"), page)
	}

	for path := range writes {
		if _, err := os.Stat(filepath.Join(o.Root, path)); err == nil {
			return fmt.Errorf("%s already exists: pick another --api, or remove what's left of an earlier run", path)
		}
	}
	for _, dir := range []string{f.defDir, pkgDir} {
		if err := identsFree(filepath.Join(o.Root, dir), d.Ident, d.Lower, "read"+d.Ident, "Test"+d.Ident+"Golden"); err != nil {
			return err
		}
	}

	// The edits to existing files.
	mainFile, registered, err := register(o.Root, pkgDir, providerBlock(o, d, fill(f.register, d)))
	if err != nil {
		return err
	}
	edits := map[string][]byte{mainFile: registered}
	if all, ok, err := importInAll(o.Root, f.kind); err != nil {
		return err
	} else if ok {
		edits[filepath.Join("plugins", "all", "all.go")] = all
	}
	if m := f.mapEntry; m != nil {
		if edits[m.file], err = addMapEntry(filepath.Join(o.Root, m.file), m.varName, fill(m.entry, d)); err != nil {
			return err
		}
	}
	readmePath := filepath.Join(pkgDir, "testdata", "README.md")
	readme, err := os.ReadFile(filepath.Join(o.Root, readmePath))
	if err != nil {
		return err
	}
	if edits[readmePath], err = readmeRows(readme, []string{d.SourceFile, d.DropFile}, o.Source); err != nil {
		return fmt.Errorf("%s: %w", readmePath, err)
	}

	// Write.
	var paths []string
	for path, b := range writes {
		full := filepath.Join(o.Root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(full, b, 0o644); err != nil {
			return err
		}
		paths = append(paths, path+" (new)")
	}
	for path, b := range edits {
		if err := os.WriteFile(filepath.Join(o.Root, path), b, 0o644); err != nil {
			return err
		}
		paths = append(paths, path)
	}
	slices.Sort(paths)
	fmt.Fprintf(out, "Scaffolded %s/%s (%s, first name %s, kind %s):\n", o.ProviderID, o.SourceID, o.Provider, o.API, o.Format)
	for _, p := range paths {
		fmt.Fprintln(out, "  "+p)
	}
	if o.Go {
		for _, args := range [][]string{
			{"test", "./" + pkgDir, "-run", "^Test" + d.Ident + "Golden$", "-update"},
		} {
			cmd := exec.Command("go", args...)
			cmd.Dir = o.Root
			if b, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("the files are written, but `go %s` failed: %w\n%s", strings.Join(args, " "), err, b)
			}
		}
		fmt.Fprintf(out, "  %s (from the placeholders)\n", filepath.Join(pkgDir, "testdata", "golden", o.API+".json"))
	}
	fmt.Fprint(out, fill(nextSteps, struct {
		data
		Extra, Page string
		Shared      bool
	}{d, fill(f.next, d), o.Page, sharedPage}))
	return nil
}

const nextSteps = `
Next steps (every place to fill in says TODO(new-plugin): grep -rn 'TODO(new-plugin)' .):
  1. Save a real copy of the source over the placeholder:
       curl -fsSL -o plugins/{{.Kind}}/testdata/{{.SourceFile}} '<the source URL>'
     and a realistic input that must not become an event as plugins/{{.Kind}}/testdata/{{.DropFile}}.
     Fill in their rows in plugins/{{.Kind}}/testdata/README.md (URL, date fetched, license, quote).
  2. Fill in the definition (the TODOs in the file next to the Register line for {{.Ident}}).{{if .Extra}}
     {{.Extra}}{{end}}
  3. Write the golden file, then read it like code:
       go test ./plugins/{{.Kind}} -run Test{{.Ident}} -update
  4. {{if .Shared}}Add {{.ID}} to providers/{{.Page}}.md (shared with the provider's other plugins){{else}}Fill in providers/{{.Page}}.md{{end}}, above all "Why this source", and the source's
     metadata in its RegisterProvider block (what it watches, its URLs; a new provider's category and
     homepage). go run ./cmd/list-plugins prints the list with it; there is no list to edit.
  5. make check, then open the PR (git commit -s).
`

func fill(tmpl string, d any) string {
	var b strings.Builder
	if err := template.Must(template.New("").Parse(tmpl)).Execute(&b, d); err != nil {
		panic(err) // the templates are constants: a failure is a bug here
	}
	return b.String()
}

func gofmt(src string) ([]byte, error) { return format.Source([]byte(src)) }

// ident is the id as a Go name: "acme-models" -> "AcmeModels".
func ident(id string) string {
	var b strings.Builder
	for _, part := range strings.Split(id, "-") {
		b.WriteString(strings.ToUpper(part[:1]) + part[1:])
	}
	return b.String()
}

// fileBase is the id as a file name: hyphens become underscores, and a last
// part Go would read as a build constraint (_linux, _arm64, _test) gets
// "_plugin" after it.
func fileBase(id string) string {
	base := strings.ReplaceAll(id, "-", "_")
	parts := strings.Split(base, "_")
	if len(parts) > 1 && slices.Contains(goosArchs, parts[len(parts)-1]) {
		base += "_plugin"
	}
	return base
}

// dropName is "x.rss.xml" -> "x-drop.rss.xml".
func dropName(name string) string {
	base, ext, _ := strings.Cut(name, ".")
	return base + "-drop." + ext
}

// identsFree fails if the package in dir already declares one of names.
func identsFree(dir string, names ...string) error {
	pkgs, err := parser.ParseDir(token.NewFileSet(), dir, nil, parser.SkipObjectResolution)
	if err != nil {
		return err
	}
	for _, pkg := range pkgs {
		for path, file := range pkg.Files {
			for _, decl := range file.Decls {
				for _, n := range declNames(decl) {
					if slices.Contains(names, n) {
						return fmt.Errorf("%s already declares %s: pick another --api", path, n)
					}
				}
			}
		}
	}
	return nil
}

func declNames(decl ast.Decl) []string {
	switch d := decl.(type) {
	case *ast.FuncDecl:
		if d.Recv == nil {
			return []string{d.Name.Name}
		}
	case *ast.GenDecl:
		var out []string
		for _, s := range d.Specs {
			switch s := s.(type) {
			case *ast.ValueSpec:
				for _, n := range s.Names {
					out = append(out, n.Name)
				}
			case *ast.TypeSpec:
				out = append(out, s.Name.Name)
			}
		}
		return out
	}
	return nil
}

// providerBlock turns a template's "plugin.Register(<plugin>)" line into
// the provider's registration with its metadata: a provider already
// registered (a second source of Algolia) keeps its name, category and
// homepage; a new one gets what the flags say, or placeholders that say
// TODO(new-plugin).
func providerBlock(o Options, d data, line string) string {
	expr := strings.TrimSuffix(strings.TrimPrefix(line, "plugin.Register("), ")")
	category := "plugin.CategoryDevTools, // TODO(new-plugin): the provider's category, one of plugin.Categories()\n\t\t"
	if o.Category != "" {
		category = fmt.Sprintf("plugin.Category(%q),", o.Category)
	}
	homepage := `"https://example.com/TODO-` + o.ProviderID + `-homepage", // TODO(new-plugin): the provider's own site` + "\n\t\t"
	if o.Homepage != "" {
		homepage = fmt.Sprintf("%q,", o.Homepage)
	}
	var urls, pkgs []string
	for _, u := range o.URLs {
		urls = append(urls, fmt.Sprintf("%q", u))
	}
	for _, p := range o.Packages {
		pkgs = append(pkgs, fmt.Sprintf("%q", p))
	}
	packages := ""
	if len(pkgs) > 0 {
		packages = fmt.Sprintf("\n\t\t\tPackages: []string{%s},", strings.Join(pkgs, ", "))
	}
	return fmt.Sprintf(`plugin.RegisterProvider(plugin.Provider{
		ID: %q, Name: %q,
		Category: %s Homepage: %s
		Sources: []plugin.Source{{
			ID: %q, Layer: %q, Page: %q,
			Watches: %q,
			URLs:    []string{%s},%s
			Plugin:  %s,
		}},
	})`, o.ProviderID, o.Provider, category, homepage, o.SourceID, o.Layer, "providers/"+o.Page+".md", o.Watches, strings.Join(urls, ", "), packages, expr)
}

// register adds line at the end of the init func that registers pkgDir's
// plugins, next to the others.
func register(root, pkgDir, line string) (string, []byte, error) {
	files, err := filepath.Glob(filepath.Join(root, pkgDir, "*.go"))
	if err != nil {
		return "", nil, err
	}
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return "", nil, err
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, src, parser.SkipObjectResolution)
		if err != nil {
			return "", nil, err
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name.Name != "init" || fn.Recv != nil || !bytes.Contains(src[fn.Pos()-1:fn.End()-1], []byte("plugin.Register")) {
				continue
			}
			at := fset.Position(fn.Body.Rbrace).Offset
			var b bytes.Buffer
			b.Write(src[:at])
			b.WriteString("\t" + line + "\n")
			b.Write(src[at:])
			out, err := format.Source(b.Bytes())
			if err != nil {
				return "", nil, fmt.Errorf("adding the Register line to %s: %w", path, err)
			}
			rel, _ := filepath.Rel(root, path)
			return rel, out, nil
		}
	}
	return "", nil, fmt.Errorf("found no init in %s that calls plugin.RegisterProvider", pkgDir)
}

// addMapEntry adds entry to the map literal varName is set to in path.
func addMapEntry(path, varName, entry string) ([]byte, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, src, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	var lit *ast.CompositeLit
	ast.Inspect(file, func(n ast.Node) bool {
		if v, ok := n.(*ast.ValueSpec); ok && len(v.Names) == 1 && v.Names[0].Name == varName && len(v.Values) == 1 {
			lit, _ = v.Values[0].(*ast.CompositeLit)
		}
		return lit == nil
	})
	if lit == nil {
		return nil, fmt.Errorf("%s: no map literal %s to add %s to", path, varName, entry)
	}
	at := fset.Position(lit.Rbrace).Offset
	var b bytes.Buffer
	b.Write(src[:at])
	if len(lit.Elts) > 0 && fset.Position(lit.Elts[len(lit.Elts)-1].End()).Line == fset.Position(lit.Rbrace).Line {
		b.WriteString(", " + entry)
	} else {
		b.WriteString("\t" + entry + ",\n")
	}
	b.Write(src[at:])
	return format.Source(b.Bytes())
}

// readmeRows adds a row per saved file to the first table of a testdata
// README whose first column is the file, with a TODO in every other column.
func readmeRows(readme []byte, files []string, source string) ([]byte, error) {
	lines := strings.Split(string(readme), "\n")
	header := -1
	for i, l := range lines {
		if strings.HasPrefix(l, "| File") {
			header = i
			break
		}
	}
	if header < 0 {
		// The format's sources live with its reader package (retire/,
		// sdkguide/); this package's own saved sources get a table of their own.
		for len(lines) > 0 && lines[len(lines)-1] == "" {
			lines = lines[:len(lines)-1]
		}
		lines = append(lines, "", "## Saved sources", "", "| File | URL | Saved | License | Quote |", "|---|---|---|---|---|", "")
		header = len(lines) - 3
	}
	end := header + 1
	for end < len(lines) && strings.HasPrefix(lines[end], "|") {
		end++
	}
	var cols []string
	for _, c := range strings.Split(strings.Trim(lines[header], "| "), "|") {
		cols = append(cols, strings.ToLower(strings.TrimSpace(c)))
	}
	var rows []string
	for _, file := range files {
		cells := []string{"`" + file + "`"}
		for _, c := range cols[1:] {
			switch {
			case strings.Contains(c, "url") || strings.Contains(c, "source"):
				cells = append(cells, "TODO(new-plugin): "+source)
			case strings.Contains(c, "saved") || strings.Contains(c, "date") || strings.Contains(c, "fetched") || strings.Contains(c, "checked"):
				cells = append(cells, "TODO(new-plugin): YYYY-MM-DD")
			case strings.Contains(c, "license"):
				cells = append(cells, "TODO(new-plugin): the license; kept whole, or trimmed and why")
			case strings.Contains(c, "quote"):
				cells = append(cells, "TODO(new-plugin): a short quote of the part the plugin reads")
			default:
				cells = append(cells, "TODO(new-plugin)")
			}
		}
		rows = append(rows, "| "+strings.Join(cells, " | ")+" |")
	}
	lines = slices.Insert(lines, end, rows...)
	return []byte(strings.Join(lines, "\n")), nil
}

func linkText(url string) string {
	if strings.Contains(url, "example.com/TODO") {
		return "TODO: the source"
	}
	return strings.TrimPrefix(url, "https://")
}

// providerPage is providers/_template.md for this provider.
func providerPage(o Options) ([]byte, error) {
	tmpl, err := os.ReadFile(filepath.Join(o.Root, "providers", "_template.md"))
	if err != nil {
		return nil, err
	}
	s := string(tmpl)
	s = strings.ReplaceAll(s, "<Provider>", o.Provider)
	s = strings.ReplaceAll(s, "<api>", o.API)
	s = strings.ReplaceAll(s, "`<URL>`", "<"+o.Source+">")
	s = strings.ReplaceAll(s, "`<YYYY-MM-DD>`", o.Today.Format("2006-01-02")+" (TODO(new-plugin): the date you checked it)")
	s = strings.Replace(s, "Copy this file to `providers/"+o.API+".md` and fill every section. Delete these instructions.",
		"TODO(new-plugin): fill every section, then delete this line. Scaffolded by `go run ./cmd/new-plugin`.", 1)
	return []byte(s), nil
}
