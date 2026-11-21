package stripe

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/core/versions"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi/openapitest"
)

// savedSpecs serves the saved spec excerpts (testdata/openapi/) at
// stripeSpecURLTemplate's path for their tag, and the releases listing; it
// returns the specs' base URL, for the golden to cite raw.githubusercontent.com.
func savedSpecs(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tag, _, _ := strings.Cut(strings.TrimPrefix(r.URL.Path, "/"), "/")
		b, err := os.ReadFile("testdata/openapi/stripe-" + tag + "-openapi.spec3.trimmed.json")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Write(b)
	}))
	t.Cleanup(srv.Close)
	old := stripeSpecURLTemplate
	stripeSpecURLTemplate = srv.URL + "/%s/latest/openapi.spec3.json"
	t.Cleanup(func() { stripeSpecURLTemplate = old })

	rel := openapitest.Recorded(t, map[string]string{"/repos/stripe/openapi/releases?per_page=100&page=1": "stripe-openapi-releases-2026-09-30.json"})
	oldAPI := versions.GitHubAPI
	versions.GitHubAPI = rel.URL
	t.Cleanup(func() { versions.GitHubAPI = oldAPI })
	return srv.URL
}

// The golden test for additions inside one API version. 2026-09-30.endive
// is named by v2517's notes but first carried by the full spec at v2525; by
// v2526 (and still at v2592, the version's last release) Stripe had added two
// values to subscriptions_resource_paused_status_details_subscription.type
// without a new API version. The pair 2026-08-26.dahlia (v2516) ->
// 2026-09-30.endive (v2592) is read from v2525: the change between the two
// versions (retrieval_reference_number removed from
// payment_method_details_card_present, the paused-status schema added) is
// filed under previous_version 2026-08-26.dahlia, and the two values added
// inside the version under 2026-09-30.endive itself, additions only.
func TestStripeAdditionsInsideAVersionGolden(t *testing.T) {
	base := savedSpecs(t)
	ctx := context.Background()
	s := kittest.NewMem()
	env := openapitest.Env(s)
	p := openapi.Plugin{Adapter: Stripe{}}
	from, err := p.ReadAt(ctx, env, plugin.Version{ID: "2026-08-26.dahlia", Ref: "v2516"})
	if err != nil {
		t.Fatal(err)
	}
	to, err := p.ReadAt(ctx, env, plugin.Version{ID: "2026-09-30.endive", Ref: "v2592"})
	if err != nil {
		t.Fatal(err)
	}
	if rejected, err := kittest.Extract(ctx, env, s, p, from, to); err != nil || rejected {
		t.Fatalf("extract: rejected=%v err=%v", rejected, err)
	}
	events := kittest.Stored(t, s)
	inside := 0
	for _, e := range events {
		if e.PreviousVersion != e.Version {
			continue
		}
		inside++
		if e.Change != event.ChangeAdded || !strings.HasSuffix(e.SourceURL, "/v2592/latest/openapi.spec3.json") {
			t.Errorf("inside the version: %s %s from %s", e.Change, e.Description, e.SourceURL)
		}
	}
	if inside != 2 {
		t.Errorf("%d events inside 2026-09-30.endive, want the 2 values added", inside)
	}
	for i := range events {
		events[i].SourceURL = strings.Replace(events[i].SourceURL, base, "https://raw.githubusercontent.com/stripe/openapi", 1)
	}
	kittest.Golden(t, "testdata/golden/stripe-api-additions.json", events)
}

// Two releases of one API version: only what the newer adds, never what it
// removes (a version only gains things; a removal there is a correction to
// the spec, not a release). Played backwards on purpose, v2525's spec then
// v2516's under one version, so the pair has both: the paused-status schema
// gone (no event) and retrieval_reference_number back (an addition).
func TestStripeReleasesOfOneVersionAreAdditionsOnly(t *testing.T) {
	read := func(tag string) plugin.Snapshot {
		return plugin.Snapshot{
			Version:   plugin.Version{ID: "2026-09-30.endive", Ref: tag},
			Content:   kittest.Read(t, "testdata/openapi/stripe-"+tag+"-openapi.spec3.trimmed.json"),
			SourceURL: "https://raw.githubusercontent.com/stripe/openapi/" + tag + "/latest/openapi.spec3.json",
		}
	}
	events := kittest.Extracted(t, openapitest.Env(nil), openapi.Plugin{Adapter: Stripe{}}, read("v2525"), read("v2516"))
	if len(events) != 1 || events[0].Change != event.ChangeAdded || events[0].Subject.Property != "retrieval_reference_number" {
		t.Fatalf("got %d events %v, want retrieval_reference_number added only", len(events), events)
	}
}
