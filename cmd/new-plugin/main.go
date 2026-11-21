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
// Everything goes in the provider's folder, providers/<provider>/: the
// source's definition, <source>.go, using the kind's exported types (an
// OpenAPI adapter with its Versions; a feed or a page with its reader; a
// changelog, a guide or a package as a value), its registration in
// provider.go (a new provider's provider.go, or one more source in the
// existing one's RegisterProvider), a test (registered, golden, drop) over
// placeholder sources with an empty golden file, rows in testdata/README.md,
// a section in the provider's README.md (a new provider's from
// docs/provider-template.md), and, for a new provider, its import in
// providers/all. `go test ./...` passes right after; every place to fill in
// says TODO(new-plugin).
//
// It changes nothing when the provider/source id is taken, or a file it
// would write exists.
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
	"strconv"
	"strings"
	"text/template"
	"time"
	"unicode"

	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	_ "github.com/patch-it-dev/patch-it-plugins/providers/all"
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
	Kind     string   // the kind that reads it: kinds/<Kind>
	Reader   string   // for a kind with several readers: "feed" for notices; "" is the kind's default
	Layer    string   // the source's layer (plugin.Layers()); "" is the template's default
	URLs     []string // the provider's pages or files it reads; none is a placeholder
	Packages []string // the packages it covers, "pypi:acme"
	Watches  string   // what the source watches, in a few words

	Today time.Time
	// Go runs the go command afterwards: the golden file from the placeholders
	// (go test -update).
	Go bool

	tmpl string // the template: a kind, or kind/reader
	name string // the golden file's name: <provider>-<source>, or the provider's id when they're the same
	url  string // the first URL
}

// list is a repeatable flag.
type list []string

func (l *list) String() string     { return strings.Join(*l, ",") }
func (l *list) Set(v string) error { *l = append(*l, v); return nil }

func main() {
	var o Options
	var urls, packages list
	flag.StringVar(&o.Root, "root", ".", "the repo root")
	flag.StringVar(&o.ProviderID, "provider", "", "the provider's id, e.g. acme")
	flag.StringVar(&o.Provider, "name", "", "the provider's name as it writes it, e.g. Acme (a registered provider's is kept)")
	flag.StringVar(&o.Category, "category", "", "the provider's category: "+strings.Join(categories(), ", "))
	flag.StringVar(&o.Homepage, "homepage", "", "the provider's own site, https://...")
	flag.StringVar(&o.SourceID, "source", "", "the source's id within the provider, e.g. api or python-sdk")
	flag.StringVar(&o.Kind, "kind", "", "the kind that reads it: "+strings.Join(kindNames(), ", "))
	flag.StringVar(&o.Reader, "reader", "", "the kind's reader, for a kind with several: "+readerHelp())
	flag.StringVar(&o.Layer, "layer", "", "the source's layer: api, sdk, package, runtime, ci, ai-models or infra (default: the kind's)")
	flag.Var(&urls, "url", "a page or file the source reads, https://... (repeatable; default: a placeholder to fill in)")
	flag.Var(&packages, "package", "a package the source covers, ecosystem:name, e.g. pypi:acme or npm:@acme/sdk (repeatable)")
	flag.StringVar(&o.Watches, "watches", "", "what it watches, in a few words (default: a placeholder)")
	flag.Parse()
	o.URLs, o.Packages = urls, packages
	if term() {
		if err := ask(&o, bufio.NewReader(os.Stdin), os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, "new-plugin:", err)
			os.Exit(1)
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

func categories() []string {
	var out []string
	for _, c := range plugin.Categories() {
		out = append(out, string(c))
	}
	return out
}

// template is the template o asks for: from --kind and --reader.
func (o Options) template() (string, error) {
	if !slices.Contains(kindNames(), o.Kind) {
		return "", fmt.Errorf("--kind %q: want one of %s", o.Kind, strings.Join(kindNames(), ", "))
	}
	if _, ok := formats[o.Kind]; ok && o.Reader == "" {
		return o.Kind, nil
	}
	r := cmp(o.Reader, defaultReaders[o.Kind])
	if o.Kind == "apisurface" && o.Reader == "" && slices.ContainsFunc(o.Packages, func(p string) bool { return strings.HasPrefix(p, "npm:") }) {
		r = "npm" // the package's ecosystem picks the reader
	}
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
	ID, Ident, Lower, ProviderID, Provider, Source, File, Format, Kind, Module, PyPI, NPM string
	Package                                                                               string // the provider's Go package: its id without hyphens
	Golden                                                                                string // the golden file's name, without .json
	Slug                                                                                  string // the provider, lowercase: a model subject's provider
	SourceFile, DropFile, PluginOf, Ingest, Drop                                          string
	Imports                                                                               []string
}

// Run scaffolds the source, or changes nothing and says why.
func Run(o Options, out io.Writer) error {
	tmpl, err := o.template()
	if err != nil {
		return err
	}
	o.tmpl = tmpl
	f := formats[tmpl]
	if err := o.resolve(); err != nil {
		return err
	}
	id := o.ProviderID + "/" + o.SourceID
	o.name = o.ProviderID + "-" + o.SourceID
	if o.SourceID == o.ProviderID {
		o.name = o.ProviderID
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
	if len(o.URLs) == 0 {
		o.URLs = []string{"https://example.com/TODO-" + o.name + "-source"}
	}
	o.url = o.URLs[0]
	for _, u := range o.URLs {
		if !validURL.MatchString(u) {
			return fmt.Errorf("--url %q: want an https URL", u)
		}
	}
	if len(o.Packages) == 0 && f.kind == "apisurface" {
		o.Packages = []string{strings.TrimPrefix(o.tmpl, "apisurface/") + ":" + o.ProviderID}
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

	// Taken? Check everything before writing anything.
	if _, ok := plugin.LookupSource(id); ok {
		return fmt.Errorf("source %s is already registered: pick another --source", id)
	}
	dir := filepath.Join("providers", o.ProviderID)
	_, statErr := os.Stat(filepath.Join(o.Root, dir, "provider.go"))
	newProvider := statErr != nil

	d := data{ID: id, ProviderID: o.ProviderID, Provider: o.Provider, Source: o.url, Format: o.tmpl, Kind: f.kind, PyPI: o.ProviderID, Golden: o.name}
	for _, p := range o.Packages {
		if name, ok := strings.CutPrefix(p, "pypi:"); ok {
			d.PyPI = name
			break
		}
	}
	d.NPM = o.ProviderID
	for _, p := range o.Packages {
		if name, ok := strings.CutPrefix(p, "npm:"); ok {
			d.NPM = name
			break
		}
	}
	d.Module = strings.ReplaceAll(strings.ToLower(d.PyPI), "-", "_")
	d.Package = strings.ReplaceAll(o.ProviderID, "-", "")
	d.Ident = ident(o.SourceID)
	d.Lower = string(unicode.ToLower(rune(d.Ident[0]))) + d.Ident[1:]
	d.File = fileBase(o.SourceID)
	d.Slug = strings.Trim(regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(strings.ToLower(o.Provider), "-"), "-")
	d.SourceFile = fill(f.source, d)
	d.DropFile = dropName(d.SourceFile)
	d.PluginOf = fill(f.pluginOf, d)
	d.Ingest = fill(f.ingest, d)
	d.Drop = fill(f.drop, d)
	d.Imports = append(slices.Clone(f.imports), "testing", module+"core/kit/kittest")
	slices.Sort(d.Imports)

	writes := map[string][]byte{}
	add := func(path string, b []byte) { writes[path] = b }

	def, err := gofmt(fill(f.def, d))
	if err != nil {
		return fmt.Errorf("the %s template: %w", o.tmpl, err)
	}
	add(filepath.Join(dir, d.File+".go"), def)
	test, err := gofmt(fill(testFile, d))
	if err != nil {
		return fmt.Errorf("the test template: %w", err)
	}
	add(filepath.Join(dir, d.File+"_test.go"), test)
	for name, content := range f.fixtures {
		add(filepath.Join(dir, "testdata", fill(name, d)), []byte(fill(content, d)))
	}
	drop := f.dropFile
	if drop == "" {
		drop = f.fixtures[f.source]
	}
	add(filepath.Join(dir, "testdata", d.DropFile), []byte(fill(drop, d)))
	// What the placeholders give; rewritten with -update below, and again
	// once the source is real.
	add(filepath.Join(dir, "testdata", "golden", o.name+".json"), []byte("[]\n"))

	for path := range writes {
		if _, err := os.Stat(filepath.Join(o.Root, path)); err == nil {
			return fmt.Errorf("%s already exists: pick another --source, or remove what's left of an earlier run", path)
		}
	}
	if !newProvider {
		if err := identsFree(filepath.Join(o.Root, dir), d.Ident, d.Lower, "read"+d.Ident, "Test"+d.Ident+"Golden"); err != nil {
			return err
		}
	}

	// The provider's registration, page and testdata README: new, or edited.
	source := sourceLiteral(o, fill(f.plugin, d))
	readmePath := filepath.Join(dir, "testdata", "README.md")
	edits := map[string][]byte{}
	if newProvider {
		reg, err := gofmt(providerFile(o, d, strings.TrimPrefix(source, "plugin.Source"), f.regImports))
		if err != nil {
			return fmt.Errorf("the provider.go template: %w", err)
		}
		add(filepath.Join(dir, "provider.go"), reg)
		page, err := providerPage(o)
		if err != nil {
			return err
		}
		add(filepath.Join(dir, "README.md"), page)
		rows, err := readmeRows([]byte(fill(testdataREADME, d)), []string{d.SourceFile, d.DropFile}, o.url)
		if err != nil {
			return err
		}
		add(readmePath, rows)
		all, err := importInAll(o.Root, o.ProviderID)
		if err != nil {
			return err
		}
		edits[filepath.Join("providers", "all", "all.go")] = all
	} else {
		regPath := filepath.Join(dir, "provider.go")
		reg, err := addSource(filepath.Join(o.Root, regPath), source, f.regImports)
		if err != nil {
			return err
		}
		edits[regPath] = reg
		pagePath := filepath.Join(dir, "README.md")
		page, err := os.ReadFile(filepath.Join(o.Root, pagePath))
		if err != nil {
			return err
		}
		section, err := pageSection(o)
		if err != nil {
			return err
		}
		edits[pagePath] = append(append(bytes.TrimRight(page, "\n"), "\n\n"...), section...)
		readme, err := os.ReadFile(filepath.Join(o.Root, readmePath))
		if err != nil {
			readme = []byte(fill(testdataREADME, d))
		}
		if edits[readmePath], err = readmeRows(readme, []string{d.SourceFile, d.DropFile}, o.url); err != nil {
			return fmt.Errorf("%s: %w", readmePath, err)
		}
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
	fmt.Fprintf(out, "Scaffolded %s (%s, kind %s):\n", id, o.Provider, o.tmpl)
	for _, p := range paths {
		fmt.Fprintln(out, "  "+p)
	}
	if o.Go {
		for _, args := range [][]string{
			{"test", "./" + filepath.ToSlash(dir), "-run", "^Test" + d.Ident + "Golden$", "-update"},
		} {
			cmd := exec.Command("go", args...)
			cmd.Dir = o.Root
			if b, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("the files are written, but `go %s` failed: %w\n%s", strings.Join(args, " "), err, b)
			}
		}
		fmt.Fprintf(out, "  %s (from the placeholders)\n", filepath.Join(dir, "testdata", "golden", o.name+".json"))
	}
	fmt.Fprint(out, fill(nextSteps, struct {
		data
		Extra string
	}{d, fill(f.next, d)}))
	return nil
}

const nextSteps = `
Next steps (every place to fill in says TODO(new-plugin): grep -rn 'TODO(new-plugin)' providers/{{.ProviderID}}):
  1. Save a real copy of the source over the placeholder:
       curl -fsSL -o providers/{{.ProviderID}}/testdata/{{.SourceFile}} '<the source URL>'
     and a realistic input that must not become an event as providers/{{.ProviderID}}/testdata/{{.DropFile}}.
     Fill in their rows in providers/{{.ProviderID}}/testdata/README.md (URL, date fetched, license, quote).
  2. Fill in the definition, providers/{{.ProviderID}}/{{.File}}.go.{{if .Extra}}
     {{.Extra}}{{end}}
  3. Write the golden file, then read it like code:
       go test ./providers/{{.ProviderID}} -run Test{{.Ident}} -update
  4. Fill in {{.ID}}'s section of providers/{{.ProviderID}}/README.md, above all "Why this source", and
     the source's metadata in providers/{{.ProviderID}}/provider.go (what it watches, its URLs; a new
     provider's category and homepage). go run ./cmd/list-plugins prints the list with it; there is no
     list to edit.
  5. make check, then open the PR (git commit -s).
`

// testdataREADME is a new provider's testdata/README.md, before its rows.
const testdataREADME = `# {{.Provider}}: saved copies and goldens

Real copies of what {{.Provider}} publishes, read by this folder's tests instead of the network: each row says where it was fetched, when, its license and a quote. The golden files under ` + "`golden/`" + ` are every event the tests must store; ` + "`go test ./providers/{{.ProviderID}} -update`" + ` rewrites them after an intended change, and the PR says why the new output is right.
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
						return fmt.Errorf("%s already declares %s: pick another --source", path, n)
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

// sourceLiteral is the source's entry in its provider's Sources, with its
// metadata and the plugin that runs it.
func sourceLiteral(o Options, pluginExpr string) string {
	var urls, pkgs []string
	for _, u := range o.URLs {
		urls = append(urls, fmt.Sprintf("%q", u))
	}
	for _, p := range o.Packages {
		pkgs = append(pkgs, fmt.Sprintf("%q", p))
	}
	packages := ""
	if len(pkgs) > 0 {
		packages = fmt.Sprintf("\n\tPackages: []string{%s},", strings.Join(pkgs, ", "))
	}
	return fmt.Sprintf(`plugin.Source{
	ID: %q, Layer: %q,
	Watches: %q,
	URLs:    []string{%s},%s
	Plugin:  %s,
}`, o.SourceID, o.Layer, o.Watches, strings.Join(urls, ", "), packages, pluginExpr)
}

// providerFile is a new provider's provider.go: its registration, with what
// the flags say about it, or placeholders that say TODO(new-plugin).
func providerFile(o Options, d data, source string, imports []string) string {
	category := "plugin.CategoryDevTools, // TODO(new-plugin): the provider's category, one of plugin.Categories()\n\t\t"
	if o.Category != "" {
		category = fmt.Sprintf("plugin.Category(%q),", o.Category)
	}
	homepage := `"https://example.com/TODO-` + o.ProviderID + `-homepage", // TODO(new-plugin): the provider's own site` + "\n\t\t"
	if o.Homepage != "" {
		homepage = fmt.Sprintf("%q,", o.Homepage)
	}
	imps := append([]string{module + "core/plugin"}, imports...)
	slices.Sort(imps)
	var b strings.Builder
	fmt.Fprintf(&b, "// Package %s is what patch-it reads for %s: every source, registered\n// below with its metadata, each read by its kind (kinds/<kind>). README.md\n// says why each source and what it reads.\npackage %s\n\nimport (\n", d.Package, o.Provider, d.Package)
	for _, imp := range slices.Compact(imps) {
		fmt.Fprintf(&b, "\t%q\n", imp)
	}
	fmt.Fprintf(&b, `)

func init() {
	plugin.RegisterProvider(plugin.Provider{
		ID: %q, Name: %q,
		Category: %s Homepage: %s
		Sources: []plugin.Source{
			%s,
		},
	})
}
`, o.ProviderID, o.Provider, category, homepage, source)
	return b.String()
}

// addSource adds source to the Sources of the RegisterProvider call in an
// existing provider.go, and the imports its plugin needs.
func addSource(path, source string, imports []string) ([]byte, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, src, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	// The Sources of the provider's one plugin.RegisterProvider call.
	var sources ast.Expr
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || sources != nil {
			return sources == nil
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if pkg, isIdent := sel.X.(*ast.Ident); !ok || !isIdent || pkg.Name != "plugin" || sel.Sel.Name != "RegisterProvider" || len(call.Args) != 1 {
			return true
		}
		lit, ok := call.Args[0].(*ast.CompositeLit)
		if !ok {
			return true
		}
		for _, el := range lit.Elts {
			if kv, ok := el.(*ast.KeyValueExpr); ok {
				if k, ok := kv.Key.(*ast.Ident); ok && k.Name == "Sources" {
					sources = kv.Value
				}
			}
		}
		return false
	})
	var at int
	switch v := sources.(type) {
	case *ast.CompositeLit: // []plugin.Source{...}
		at = fset.Position(v.Rbrace).Offset
		source = strings.TrimPrefix(source, "plugin.Source")
	case *ast.CallExpr: // append(specs(), ...)
		at = fset.Position(v.Rparen).Offset
	default:
		return nil, fmt.Errorf("%s: found no Sources in a plugin.RegisterProvider(plugin.Provider{...}) call", path)
	}
	var b bytes.Buffer
	b.Write(bytes.TrimRight(src[:at], " \t\n"))
	if !bytes.HasSuffix(bytes.TrimRight(src[:at], " \t\n"), []byte(",")) && !bytes.HasSuffix(bytes.TrimRight(src[:at], " \t\n"), []byte("{")) {
		b.WriteString(",")
	}
	b.WriteString("\n" + source + ",\n")
	b.Write(src[at:])
	out := b.Bytes()
	for _, imp := range imports {
		if out, err = addImport(out, imp); err != nil {
			return nil, err
		}
	}
	if out, err = format.Source(out); err != nil {
		return nil, fmt.Errorf("adding the source to %s: %w", path, err)
	}
	return out, nil
}

// addImport adds path to src's imports when it isn't there.
func addImport(src []byte, path string) ([]byte, error) {
	file, err := parser.ParseFile(token.NewFileSet(), "", src, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}
	for _, imp := range file.Imports {
		if p, _ := strconv.Unquote(imp.Path.Value); p == path {
			return src, nil
		}
	}
	q := strconv.Quote(path)
	if bytes.Contains(src, []byte("import (\n")) {
		return bytes.Replace(src, []byte("import (\n"), []byte("import (\n\t"+q+"\n"), 1), nil
	}
	end := int(file.Name.End()) - 1
	return slices.Concat(src[:end], []byte("\n\nimport "+q+"\n"), src[end:]), nil
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

// providerPage is a new provider's README.md: docs/provider-template.md
// for this provider and source.
func providerPage(o Options) ([]byte, error) {
	tmpl, err := os.ReadFile(filepath.Join(o.Root, "docs", "provider-template.md"))
	if err != nil {
		return nil, err
	}
	return []byte(fillPage(o, string(tmpl))), nil
}

// pageSection is the template's section for one source, added to the
// README.md of a provider that has one.
func pageSection(o Options) ([]byte, error) {
	tmpl, err := os.ReadFile(filepath.Join(o.Root, "docs", "provider-template.md"))
	if err != nil {
		return nil, err
	}
	s := string(tmpl)
	i := strings.Index(s, "\n## ")
	if i < 0 {
		return nil, fmt.Errorf("docs/provider-template.md has no source section (## ...)")
	}
	return []byte(fillPage(o, s[i+1:])), nil
}

func fillPage(o Options, s string) string {
	s = regexp.MustCompile(`(?s)<!--.*?-->\n\n`).ReplaceAllString(s, "TODO(new-plugin): fill every part of "+o.ProviderID+"/"+o.SourceID+"'s section, then delete this line. Scaffolded by `go run ./cmd/new-plugin`.\n\n")
	s = strings.ReplaceAll(s, "<Provider>", o.Provider)
	s = strings.ReplaceAll(s, "<What the source is>", "TODO(new-plugin): what it is")
	s = strings.ReplaceAll(s, "`<provider>/<source>`", "`"+o.ProviderID+"/"+o.SourceID+"`")
	s = strings.ReplaceAll(s, "`<kind>` (`kinds/<kind>`)", "`"+o.Kind+"` (`kinds/"+o.Kind+"`)")
	s = strings.ReplaceAll(s, "`<URL>`", "<"+o.url+">")
	s = strings.ReplaceAll(s, "`<YYYY-MM-DD>`", o.Today.Format("2006-01-02")+" (TODO(new-plugin): the date you checked it)")
	return s
}
