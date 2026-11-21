package cloudflare

import (
	"context"
	"fmt"
	"os"
	"slices"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi/adapter"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi/openapitest"
)

// cloudflareDays are the saved excerpts of cloudflare/api-schemas's
// openapi.json (testdata/README.md): day -> the day's last commit.
var cloudflareDays = map[string]string{
	"2026-10-29": "46a48d5eba1d8a839cd8cc5712d2b9368e21e654",
	"2026-10-30": "319dc756af2a642b3122b4db7e3b79ac1fbec7d8",
}

func cloudflareURL(day string) string {
	return "https://raw.githubusercontent.com/cloudflare/api-schemas/" + cloudflareDays[day] + "/openapi.json"
}

// fakeCloudflareAdapter serves the saved excerpts, cited by the real URL each
// was copied from and reshaped as Cloudflare.Fetch does.
type fakeCloudflareAdapter struct{}

func (fakeCloudflareAdapter) API() string { return "cloudflare/api-spec" }

func (fakeCloudflareAdapter) LatestTag(context.Context) (string, error) { return "2026-10-30", nil }

func (fakeCloudflareAdapter) Fetch(_ context.Context, day string) (adapter.Spec, error) {
	if _, ok := cloudflareDays[day]; !ok {
		return adapter.Spec{}, fmt.Errorf("no saved copy for %s", day)
	}
	body, err := os.ReadFile("testdata/cloudflare-" + day + ".json")
	if err != nil {
		return adapter.Spec{}, err
	}
	content, err := adapter.ServerBasePath(body)
	if err != nil {
		return adapter.Spec{}, err
	}
	return adapter.Spec{Content: content, Version: day, SourceURL: cloudflareURL(day)}, nil
}

func (fakeCloudflareAdapter) SourcePointer(content []byte) func(string) string {
	return adapter.BasePathPointer(content)
}

func TestCloudflareRegistersItself(t *testing.T) {
	kittest.Registered(t, openapi.Plugin{Adapter: Cloudflare{}})
}

// cloudflarePair stores the 2026-10-29 -> 2026-10-30 pair: the three
// Registrar domains operations, marked deprecated, left the spec, as did POST
// /accounts/move and GET /tenants/{tenant_id}, never marked.
func cloudflarePair(t *testing.T) []event.Event {
	t.Helper()
	ctx := context.Background()
	s := kittest.NewMem()
	p := openapi.Plugin{Adapter: fakeCloudflareAdapter{}, Diff: cloudflareDiff, Keep: cloudflareKeep}
	if rejected, err := kittest.Pair(ctx, openapitest.Env(s), s, p, "2026-10-29", "2026-10-30"); err != nil || rejected {
		t.Fatalf("pair: rejected=%v err=%v", rejected, err)
	}
	return kittest.Stored(t, s)
}

// The golden test: the removal of the deprecated Registrar domains
// operations, superseded by /registrar/registrations.
func TestCloudflareGolden(t *testing.T) {
	events := cloudflarePair(t)
	var ops []string
	for _, e := range events {
		ops = append(ops, e.Subject.Operation)
		if e.Change != event.ChangeBreaking || e.Version != "2026-10-30" || e.PreviousVersion != "2026-10-29" || e.SourceURL != cloudflareURL("2026-10-29") {
			t.Errorf("%s: %s %s<-%s cites %s", e.Subject.Operation, e.Change, e.Version, e.PreviousVersion, e.SourceURL)
		}
	}
	slices.Sort(ops)
	want := []string{
		"GET /client/v4/accounts/{account_id}/registrar/domains",
		"GET /client/v4/accounts/{account_id}/registrar/domains/{domain_name}",
		"PUT /client/v4/accounts/{account_id}/registrar/domains/{domain_name}",
	}
	if !slices.Equal(ops, want) {
		t.Errorf("operations %v, want %v", ops, want)
	}
	kittest.Golden(t, "testdata/golden/cloudflare-api-spec.json", events)
}

// The pinned drop test: POST /accounts/move and GET /tenants/{tenant_id} left
// the same spec without ever being marked deprecated: no event names them.
func TestCloudflareDropsRemovalsThatWerentDeprecated(t *testing.T) {
	for _, e := range cloudflarePair(t) {
		if e.Subject.Operation == "POST /client/v4/accounts/move" || e.Subject.Operation == "GET /client/v4/tenants/{tenant_id}" {
			t.Errorf("event for %s, never deprecated", e.Subject.Operation)
		}
	}
}
