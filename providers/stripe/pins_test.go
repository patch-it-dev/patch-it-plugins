package stripe

import (
	"context"
	"errors"
	"maps"
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/kinds/changelog"
)

// Each Stripe SDK's changelog at its newest tag, trimmed to the releases
// that state their pinned API version in each wording the SDKs use, and one
// that states none (testdata/README.md): what each release pins, exactly as
// stated, and nothing for the release that doesn't say.
func TestStripeReleasesStateWhatTheyPin(t *testing.T) {
	for _, c := range []struct {
		file changelog.File
		tag  string
		want map[string]string
	}{
		{Python, "17.1.0", map[string]string{"17.1.0": "2026-11-18.endive"}}, // 17.0.0 says nothing
		{Node, "23.2.0", map[string]string{"23.2.0": "2026-11-18.endive", "22.0.0": "2026-03-25.dahlia"}},
		{Go, "87.2.0", map[string]string{"87.2.0": "2026-11-18.endive", "86.0.0": "2026-05-27.dahlia"}},
		{Java, "34.1.0", map[string]string{"34.1.0": "2026-10-28.endive", "33.0.0": "2026-05-27.dahlia"}},
		{Ruby, "21.1.0", map[string]string{"21.1.0": "2026-11-18.endive"}},
		{Dotnet, "53.2.0", map[string]string{"53.2.0": "2026-11-18.endive", "52.0.0": "2026-05-27.dahlia"}},
		{PHP, "23.1.0", map[string]string{"23.1.0": "2026-11-18.endive", "21.0.0": "2026-06-24.dahlia"}},
	} {
		t.Run(c.file.API, func(t *testing.T) {
			if c.file.Pins != "stripe/api" {
				t.Errorf("Pins %q, want stripe/api", c.file.Pins)
			}
			got := changelog.Pins(kittest.Read(t, "testdata/pins/"+strings.ReplaceAll(c.file.API, "/", "-")+"-"+c.tag+"-CHANGELOG.md"))
			if !maps.Equal(got, c.want) {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}

// Listed, each version carries what its own section states it pins, read
// from the changelog at the newest version's tag; a version whose section
// says nothing pins nothing.
func TestListedVersionsCarryWhatTheyPin(t *testing.T) {
	env := kittest.Env(t, kittest.Pages{
		"https://api.github.com/repos/stripe/stripe-go/tags?per_page=100&page=1": "testdata/versions/stripe-go-tags-pins.json",
		Go.At("87.2.0").URL(): "testdata/pins/stripe-go-87.2.0-CHANGELOG.md",
	})
	vs, err := changelog.Plugin{File: Go}.ListVersions(context.Background(), env)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, v := range vs {
		got = append(got, v.ID+"="+v.MapsTo.API+"@"+v.MapsTo.Version)
	}
	want := "86.0.0=stripe/api@2026-05-27.dahlia 86.4.2=@ 87.2.0=stripe/api@2026-11-18.endive"
	if strings.Join(got, " ") != want {
		t.Errorf("got  %s\nwant %s", strings.Join(got, " "), want)
	}
}

// A changelog that can't be fetched at the newest two tags leaves the listing
// as it is, pinning nothing; a source without Pins never fetches it.
func TestPinsAreReadOnlyWhenTheyCanBe(t *testing.T) {
	tags := kittest.Read(t, "testdata/versions/stripe-go-tags-pins.json")
	var fetched []string
	env := plugin.Env{Get: func(_ context.Context, url string) ([]byte, error) {
		fetched = append(fetched, url)
		if strings.Contains(url, "/tags?") {
			return tags, nil
		}
		return nil, errors.New("503")
	}}
	vs, err := changelog.Plugin{File: Go}.ListVersions(context.Background(), env)
	if err != nil || len(vs) != 3 {
		t.Fatalf("%v, %d versions", err, len(vs))
	}
	if len(fetched) != 3 { // the tags, then the newest two releases' changelogs
		t.Errorf("fetched %v", fetched)
	}
	for _, v := range vs {
		if !v.MapsTo.IsZero() {
			t.Errorf("%s pins %v with no changelog read", v.ID, v.MapsTo)
		}
	}
	unpinned := Go
	unpinned.Pins = ""
	fetched = nil
	if _, err := (changelog.Plugin{File: unpinned}).ListVersions(context.Background(), env); err != nil || len(fetched) != 1 {
		t.Errorf("without Pins: %v, fetched %v", err, fetched)
	}
}

// When the newest release's tag can't be read yet, the one before's
// changelog says what the earlier releases pin.
func TestPinsFallBackToTheTagBefore(t *testing.T) {
	tags := kittest.Read(t, "testdata/versions/stripe-go-tags-pins.json")
	file := kittest.Read(t, "testdata/pins/stripe-go-87.2.0-CHANGELOG.md")
	env := plugin.Env{Get: func(_ context.Context, url string) ([]byte, error) {
		switch {
		case strings.Contains(url, "/tags?"):
			return tags, nil
		case url == Go.At("86.4.2").URL():
			return file, nil
		}
		return nil, errors.New("404")
	}}
	vs, err := changelog.Plugin{File: Go}.ListVersions(context.Background(), env)
	if err != nil || len(vs) != 3 || vs[0].MapsTo.Version != "2026-05-27.dahlia" {
		t.Errorf("%v %+v", err, vs)
	}
}
