package changelog

import (
	"errors"
	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
)

// A Go module's events name the module as code imports it at that major.
func TestAGoModuleIsNamedByItsMajor(t *testing.T) {
	f := File{Package: "github.com/acme/acme-go", Ecosystem: "go"}
	for release, want := range map[string]string{"73.0.0": "github.com/acme/acme-go/v73", "1.4.0": "github.com/acme/acme-go", "0.31.0": "github.com/acme/acme-go"} {
		if got := f.At(release).installed(); got != want {
			t.Errorf("%s: %s, want %s", release, got, want)
		}
	}
}

// A release's notes on GitHub are read whole: any heading level starts a
// section, a code fence is skipped, and notes for another tag or none at
// all are never read as "no breaking changes".
func TestReleaseNotesAreReadWhole(t *testing.T) {
	f := File{Package: "acme", Repo: "acme/acme", TagPrefix: "v", Notes: true}.At("2.0.0")
	body := "## BREAKING CHANGES\\n\\n* `acme.old()` has been removed\\n\\n```\\n- `acme.example()` has been removed\\n```\\n\\n### Features\\n\\n* add `acme.new()`\\n"
	_, fs, err := f.Read([]byte(`{"tag_name": "v2.0.0", "name": "v2.0.0", "body": "` + body + `"}`))
	if err != nil {
		t.Fatal(err)
	}
	// The removal, and the feature that names what it adds.
	if len(fs) != 2 || fs[0].Dropped != "" || strings.Join(fs[0].Change.Symbols, ",") != "acme.old" ||
		fs[1].Change.Change != event.PackageAddedSymbol || strings.Join(fs[1].Change.Symbols, ",") != "acme.new" {
		t.Errorf("findings %+v", fs)
	}
	if f.Cite() != "https://github.com/acme/acme/releases/tag/v2.0.0" || f.URL() != "https://api.github.com/repos/acme/acme/releases/tags/v2.0.0" {
		t.Errorf("cite %s, url %s", f.Cite(), f.URL())
	}
	if _, _, err := f.Read([]byte(`{"tag_name": "v1.9.0", "body": "x"}`)); err == nil {
		t.Error("another release's notes: want an error")
	}
	if _, _, err := f.Read([]byte(`{"tag_name": "v2.0.0", "body": ""}`)); !errors.Is(err, plugin.ErrNothingToRead) {
		t.Errorf("no notes: %v, want plugin.ErrNothingToRead", err)
	}
}

// A rename whose new name the same release renames again or removes points
// to where the chain ends: never to a name the release took away.
func TestRenamesAreFollowedToTheirEnd(t *testing.T) {
	f := File{Package: "acme", Repo: "acme/acme", TagPrefix: "v", Notes: true}.At("2.0.0")
	body := "## BREAKING CHANGES\\n\\n" +
		"* `acme.a()` has been renamed to `acme.b()`\\n* `acme.b()` has been renamed to `acme.c()`\\n" +
		"* `acme.x()` has been renamed to `acme.y()`\\n* `acme.y()` has been removed\\n" +
		"* `acme.p()` has been renamed to `acme.q()`\\n* `acme.q()` has been renamed to `acme.p()`\\n" +
		"## Deprecations\\n\\n* Deprecated `acme.d` in favor of `acme.y`\\n"
	_, fs, err := f.Read([]byte(`{"tag_name": "v2.0.0", "body": "` + body + `"}`))
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, f := range fs {
		got = append(got, strings.Join(f.Change.Symbols, ",")+">"+f.Change.Replacement)
	}
	want := "acme.a>acme.c acme.b>acme.c acme.x> acme.y> acme.p> acme.q> acme.d>acme.y"
	if strings.Join(got, " ") != want {
		t.Errorf("got  %s\nwant %s", strings.Join(got, " "), want)
	}
}
