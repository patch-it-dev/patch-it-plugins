package changelog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/core/versions"
	_ "github.com/patch-it-dev/patch-it-plugins/kinds/openapi" // registers stripe/api
)

// Every pin the seven Stripe SDK sources read from their saved changelogs is
// a version stripe/api lists, from its saved release listing
// (../openapi/adapter/testdata/versions/): the pin and the listing name an API
// version the same way, so a feed query on an SDK's versions finds the API's.
func TestStripePinsAreVersionsStripeAPILists(t *testing.T) {
	listing, err := os.ReadFile("../openapi/adapter/testdata/versions/stripe-openapi-releases.json")
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/stripe/openapi/releases" || r.URL.Query().Get("page") != "1" {
			t.Errorf("unexpected request %s", r.URL)
			http.NotFound(w, r)
			return
		}
		w.Write(listing)
	}))
	defer srv.Close()
	old := versions.GitHubAPI
	versions.GitHubAPI = srv.URL
	defer func() { versions.GitHubAPI = old }()

	api, ok := plugin.Lookup("stripe/api")
	if !ok {
		t.Fatal("stripe/api isn't registered")
	}
	vs, err := api.(plugin.Versioned).ListVersions(context.Background(), plugin.Env{})
	if err != nil {
		t.Fatal(err)
	}
	listed := map[string]bool{}
	for _, v := range vs {
		listed[v.ID] = true
	}

	n := 0
	for _, c := range []struct {
		file File
		tag  string
	}{
		{StripePython, "17.1.0"}, {StripeNode, "23.2.0"}, {StripeGo, "87.2.0"}, {StripeJava, "34.1.0"},
		{StripeRuby, "21.1.0"}, {StripeDotnet, "53.2.0"}, {StripePHP, "23.1.0"},
	} {
		for release, pin := range pins(kittest.Read(t, "testdata/pins/"+strings.ReplaceAll(c.file.API, "/", "-")+"-"+c.tag+"-CHANGELOG.md")) {
			n++
			if c.file.Pins != "stripe/api" || !listed[pin] {
				t.Errorf("%s %s pins %s %s, which stripe/api doesn't list", c.file.API, release, c.file.Pins, pin)
			}
		}
	}
	if n == 0 {
		t.Fatal("no pins read")
	}
}
