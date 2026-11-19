// Command new-plugin scaffolds a provider on one of the format plugins, so a
// contributor only fills in what is specific to the provider. Run it from the
// repo root:
//
//	go run ./cmd/new-plugin --api acme --provider Acme --format openapi
//
// It writes the provider's definition in the format's package (or the reader
// package the format uses: adapter/, retire/, sdkguide/), adds its
// plugin.Register line to plugins/<format>'s init, a test (registered, golden,
// drop) over placeholder sources with an empty golden file, a row in that
// testdata/README.md, providers/<api>.md from providers/_template.md and a
// row in docs/plugins.md, then regenerates docs/plugins.json. `go test ./...`
// passes right after; every place to fill in says TODO(new-plugin).
//
// It changes nothing when the id is taken (registered, in docs/plugins.md, or
// a file it would write exists).
package main

import (
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

	"github.com/patch-it-dev/patch-it-plugins/internal/pluginlist"
	"github.com/patch-it-dev/patch-it-plugins/plugin"
	_ "github.com/patch-it-dev/patch-it-plugins/plugins/all"
)

// Options are the command's flags.
type Options struct {
	Root     string // the repo root
	API      string // the plugin id: --api and every event's api
	Provider string // whose API, SDK or package
	Format   string // the format plugin
	Layer    string // the docs/plugins.md layer; "" is the format's default
	Source   string // the provider's source URL; "" is a placeholder
	Watches  string // the docs/plugins.md "what it watches" cell
	Page     string // the provider page, providers/<Page>.md; "" is the id. An existing page is shared, not rewritten
	Today    time.Time
	// Go runs the go command afterwards: the golden file from the placeholders
	// (go test -update) and docs/plugins.json (go run ./cmd/list-plugins).
	Go bool
}

func main() {
	var o Options
	flag.StringVar(&o.Root, "root", ".", "the repo root")
	flag.StringVar(&o.API, "api", "", "the plugin id, e.g. acme or acme-models (lowercase letters, digits, hyphens)")
	flag.StringVar(&o.Provider, "provider", "", "the provider's name, e.g. Acme")
	flag.StringVar(&o.Format, "format", "", "the format plugin: "+strings.Join(formatNames(), ", "))
	flag.StringVar(&o.Layer, "layer", "", "the layer in docs/plugins.md: API, SDK, package, runtime, CI or \"AI models\" (default: the format's)")
	flag.StringVar(&o.Source, "source", "", "the provider's source URL (default: a placeholder to fill in)")
	flag.StringVar(&o.Watches, "watches", "", "what it watches, in a few words (default: a placeholder)")
	flag.StringVar(&o.Page, "page", "", "the provider page, providers/<page>.md (default: the --api id); one page per provider, so a second plugin of a provider names the existing page")
	flag.Parse()
	o.Today = time.Now().UTC()
	o.Go = true
	if err := Run(o, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "new-plugin:", err)
		os.Exit(1)
	}
}

func formatNames() []string {
	var names []string
	for n := range formats {
		names = append(names, n)
	}
	slices.Sort(names)
	return names
}

var (
	validID   = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)
	validURL  = regexp.MustCompile(`^https://[^\s|()]+$`)
	cellSafe  = regexp.MustCompile(`^[^|\n` + "`" + `]+$`)
	goosArchs = []string{"aix", "android", "darwin", "dragonfly", "freebsd", "hurd", "illumos", "ios", "js", "linux", "nacl", "netbsd", "openbsd", "plan9", "solaris", "wasip1", "windows", "zos",
		"386", "amd64", "arm", "arm64", "loong64", "mips", "mipsle", "mips64", "mips64le", "ppc64", "ppc64le", "riscv64", "s390x", "wasm", "test"}
)

// data is what the templates fill in.
type data struct {
	ID, Ident, Lower, Provider, Source, File, Format string
	Slug                                             string // the provider, lowercase: a model subject's provider
	SourceFile, DropFile, PluginOf, Ingest, Drop     string
	Imports                                          []string
}

// Run scaffolds the plugin, or changes nothing and says why.
func Run(o Options, out io.Writer) error {
	f, ok := formats[o.Format]
	if !ok {
		return fmt.Errorf("--format %q: want one of %s", o.Format, strings.Join(formatNames(), ", "))
	}
	if !validID.MatchString(o.API) || o.API == "all" {
		return fmt.Errorf("--api %q: want lowercase letters, digits and single hyphens, starting with a letter (acme, acme-models), and not \"all\"", o.API)
	}
	if o.Provider == "" || !cellSafe.MatchString(o.Provider) {
		return fmt.Errorf("--provider %q: want the provider's name, without | or backticks", o.Provider)
	}
	if o.Layer == "" {
		o.Layer = f.layer
	}
	if _, ok := pluginlist.Layers[o.Layer]; !ok {
		return fmt.Errorf("--layer %q: want API, SDK, package, runtime, CI or \"AI models\"", o.Layer)
	}
	if o.Source == "" {
		o.Source = "https://example.com/TODO-" + o.API + "-source"
	} else if !validURL.MatchString(o.Source) {
		return fmt.Errorf("--source %q: want an https URL", o.Source)
	}
	if o.Watches == "" {
		o.Watches = "TODO: what it watches, in a few words"
	} else if !cellSafe.MatchString(o.Watches) {
		return fmt.Errorf("--watches: no | or backticks")
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
		return fmt.Errorf("a plugin named %q is already registered: pick another --api", o.API)
	}
	tablePath := filepath.Join(o.Root, pluginlist.Table)
	table, err := os.ReadFile(tablePath)
	if err != nil {
		return fmt.Errorf("%v (run from the repo root, or pass --root)", err)
	}
	rows, err := pluginlist.ReadTable(table)
	if err != nil {
		return err
	}
	for _, r := range rows {
		if r.ID == o.API {
			return fmt.Errorf("%s already has a row for %q: pick another --api", pluginlist.Table, o.API)
		}
	}

	d := data{ID: o.API, Provider: o.Provider, Source: o.Source, Format: o.Format}
	d.Ident = ident(o.API)
	d.Lower = string(unicode.ToLower(rune(d.Ident[0]))) + d.Ident[1:]
	d.File = fileBase(o.API)
	d.Slug = strings.Trim(regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(strings.ToLower(o.Provider), "-"), "-")
	d.SourceFile = fill(f.source, d)
	d.DropFile = dropName(d.SourceFile)
	d.PluginOf = fill(f.pluginOf, d)
	d.Ingest = fill(f.ingest, d)
	d.Drop = fill(f.drop, d)
	d.Imports = append(slices.Clone(f.imports), "testing", "github.com/patch-it-dev/patch-it-plugins/kit/kittest")
	slices.Sort(d.Imports)

	pkgDir := "plugins/" + o.Format
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
	mainFile, registered, err := register(o.Root, pkgDir, fill(f.register, d))
	if err != nil {
		return err
	}
	edits := map[string][]byte{mainFile: registered}
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
	if edits[pluginlist.Table], err = tableRow(table, o, d); err != nil {
		return err
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
	fmt.Fprintf(out, "Scaffolded %s (%s, format %s):\n", o.API, o.Provider, o.Format)
	for _, p := range paths {
		fmt.Fprintln(out, "  "+p)
	}
	if o.Go {
		for _, args := range [][]string{
			{"test", "./" + pkgDir, "-run", "^Test" + d.Ident + "Golden$", "-update"},
			{"run", "./cmd/list-plugins"},
		} {
			cmd := exec.Command("go", args...)
			cmd.Dir = o.Root
			if b, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("the files are written, but `go %s` failed: %w\n%s", strings.Join(args, " "), err, b)
			}
		}
		fmt.Fprintf(out, "  %s (from the placeholders)\n  docs/plugins.json (regenerated)\n", filepath.Join(pkgDir, "testdata", "golden", o.API+".json"))
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
       curl -fsSL -o plugins/{{.Format}}/testdata/{{.SourceFile}} '<the source URL>'
     and a realistic input that must not become an event as plugins/{{.Format}}/testdata/{{.DropFile}}.
     Fill in their rows in plugins/{{.Format}}/testdata/README.md (URL, date fetched, license, quote).
  2. Fill in the definition (the TODOs in the file next to the Register line for {{.Ident}}).{{if .Extra}}
     {{.Extra}}{{end}}
  3. Write the golden file, then read it like code:
       go test ./plugins/{{.Format}} -run Test{{.Ident}} -update
  4. {{if .Shared}}Add {{.ID}} to providers/{{.Page}}.md (shared with the provider's other plugins){{else}}Fill in providers/{{.Page}}.md{{end}}, above all "Why this source", and the row in docs/plugins.md
     (what it watches, the source; a plugin in the code is live), then: make plugins-json
  5. make check, then open the PR (git commit -s).
`

func fill(tmpl string, d any) string {
	var b strings.Builder
	template.Must(template.New("").Parse(tmpl)).Execute(&b, d)
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
			if !ok || fn.Name.Name != "init" || fn.Recv != nil || !bytes.Contains(src[fn.Pos()-1:fn.End()-1], []byte("plugin.Register(")) {
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
	return "", nil, fmt.Errorf("found no init in %s that calls plugin.Register", pkgDir)
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
			case strings.Contains(c, "url"):
				cells = append(cells, "TODO(new-plugin): "+source)
			case strings.Contains(c, "saved") || strings.Contains(c, "date") || strings.Contains(c, "fetched"):
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

// tableRow adds the plugin's row at the end of docs/plugins.md's table.
func tableRow(table []byte, o Options, d data) ([]byte, error) {
	end := bytes.Index(table, []byte(pluginlist.EndMarker))
	if end < 0 {
		return nil, fmt.Errorf("%s: no %s", pluginlist.Table, pluginlist.EndMarker)
	}
	row := fmt.Sprintf("| [`%s`](https://github.com/patch-it-dev/patch-it-plugins/blob/main/providers/%s.md) | %s | %s | %s | `%s` | [%s](%s) | live |\n",
		o.API, o.Page, o.Provider, o.Layer, o.Watches, o.Format, linkText(o.Source), o.Source)
	var b bytes.Buffer
	b.Write(table[:end])
	b.WriteString(row)
	b.Write(table[end:])
	return b.Bytes(), nil
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
