package main

import (
	"bufio"
	"bytes"
	"fmt"
	"go/format"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
)

// resolve fills in the provider and source from the registry and the
// flags: a registered provider's name, category and homepage, the first
// name, and, for the deprecated flags, the ids from the name and --api.
func (o *Options) resolve() error {
	if o.ProviderID == "" && o.Provider != "" { // the old flags: --provider is the name
		o.ProviderID = strings.Trim(regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(strings.ToLower(o.Provider), "-"), "-")
		for _, p := range plugin.Providers() {
			if strings.EqualFold(p.Name, o.Provider) {
				o.ProviderID = p.ID
			}
		}
		if o.SourceID == "" {
			o.SourceID = strings.TrimPrefix(o.API, o.ProviderID+"-")
		}
	}
	if !validID.MatchString(o.ProviderID) {
		return fmt.Errorf("--provider %q: want the provider's id, lowercase letters, digits and single hyphens (acme)", o.ProviderID)
	}
	if !validID.MatchString(o.SourceID) {
		return fmt.Errorf("--source %q: want the source's id within the provider, lowercase letters, digits and single hyphens (api, python-sdk)", o.SourceID)
	}
	if i := slices.IndexFunc(plugin.Providers(), func(p plugin.Provider) bool { return p.ID == o.ProviderID }); i >= 0 {
		p := plugin.Providers()[i]
		switch {
		case o.Provider != "" && !strings.EqualFold(o.Provider, p.Name):
			return fmt.Errorf("--name %q: provider %s is registered as %q", o.Provider, p.ID, p.Name)
		case o.Category != "" && o.Category != string(p.Category):
			return fmt.Errorf("--category %q: provider %s is registered in %q", o.Category, p.ID, p.Category)
		case o.Homepage != "" && o.Homepage != p.Homepage:
			return fmt.Errorf("--homepage %q: provider %s is registered with %q", o.Homepage, p.ID, p.Homepage)
		}
		o.Provider, o.Category, o.Homepage = p.Name, string(p.Category), p.Homepage
	} else {
		if o.Category != "" && !slices.Contains(categories(), o.Category) {
			return fmt.Errorf("--category %q: want one of %s", o.Category, strings.Join(categories(), ", "))
		}
		if o.Homepage != "" && !validURL.MatchString(o.Homepage) {
			return fmt.Errorf("--homepage %q: want an https URL", o.Homepage)
		}
	}
	if o.API == "" {
		o.API = o.ProviderID + "-" + o.SourceID
		if o.SourceID == o.ProviderID {
			o.API = o.ProviderID
		}
	}
	return nil
}

// term is true when stdin is a terminal: flags left out are asked for.
func term() bool {
	fi, err := os.Stdin.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

// ask fills in, from a person at a terminal, what the flags left out: only
// when one of the provider id, the source id or the kind is missing (a full
// command line asks nothing).
func ask(o *Options, in *bufio.Reader, out io.Writer) error {
	if o.ProviderID != "" && o.SourceID != "" && o.Kind != "" {
		return nil
	}
	prompt := func(field *string, question, def string) error {
		if *field != "" {
			return nil
		}
		if def != "" {
			question += " [" + def + "]"
		}
		fmt.Fprint(out, question+": ")
		line, _ := in.ReadString('\n') // no answer (end of input) is the default
		*field = cmp(strings.TrimSpace(line), def)
		return nil
	}
	list := func(field *[]string, question string) error {
		if len(*field) > 0 {
			return nil
		}
		var answer string
		if err := prompt(&answer, question+" (comma-separated, empty for none)", ""); err != nil {
			return err
		}
		for _, v := range strings.Split(answer, ",") {
			if v = strings.TrimSpace(v); v != "" {
				*field = append(*field, v)
			}
		}
		return nil
	}
	if err := prompt(&o.ProviderID, "Provider id (acme)", ""); err != nil {
		return err
	}
	if !slices.ContainsFunc(plugin.Providers(), func(p plugin.Provider) bool { return p.ID == o.ProviderID }) {
		for _, q := range []struct {
			field    *string
			question string
		}{
			{&o.Provider, "Provider name, as it writes it (Acme)"},
			{&o.Category, "Category (" + strings.Join(categories(), ", ") + ")"},
			{&o.Homepage, "Homepage (https://...)"},
		} {
			if err := prompt(q.field, q.question, ""); err != nil {
				return err
			}
		}
	}
	if err := prompt(&o.SourceID, "Source id within the provider (api, python-sdk)", ""); err != nil {
		return err
	}
	if err := prompt(&o.Kind, "Kind ("+strings.Join(kindNames(), ", ")+"; docs/which-kind.md)", ""); err != nil {
		return err
	}
	if def := defaultReaders[o.Kind]; def != "" {
		if err := prompt(&o.Reader, "Reader ("+readerHelp()+")", def); err != nil {
			return err
		}
	}
	if err := prompt(&o.Layer, "Layer (api, sdk, package, runtime, ci, ai-models, infra; empty for the kind's default)", ""); err != nil {
		return err
	}
	if err := list(&o.URLs, "URLs it reads (https://...)"); err != nil {
		return err
	}
	if err := list(&o.Packages, "Packages it covers (pypi:acme, npm:@acme/sdk)"); err != nil {
		return err
	}
	return prompt(&o.Watches, "What it watches, in a few words", "")
}

func cmp(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// importInAll adds plugins/<kind> to plugins/all's imports when it isn't
// there (a new kind), so a program importing all runs it. ok is false when
// it's already there.
func importInAll(root, kind string) ([]byte, bool, error) {
	path := filepath.Join(root, "plugins", "all", "all.go")
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, false, err
	}
	imp := `"github.com/patch-it-dev/patch-it-plugins/plugins/` + kind + `"`
	if bytes.Contains(src, []byte(imp)) {
		return nil, false, nil
	}
	out := bytes.Replace(src, []byte("import (\n"), []byte("import (\n\t_ "+imp+"\n"), 1)
	b, err := format.Source(out)
	return b, err == nil, err
}
