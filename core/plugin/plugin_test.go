package plugin

import (
	"context"
	"slices"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
)

// testPlugin is a source defined only here: registering it and running it
// touches nothing outside this file.
type testPlugin struct {
	name string
	err  error
}

func (p testPlugin) Name() string { return p.name }

func (p testPlugin) Run(_ context.Context, env Env) (Result, error) {
	if p.err != nil {
		return Result{}, p.err
	}
	subject := event.Subject{Type: event.SubjectAPIOperation, Kind: event.KindOperation, Side: event.SideRequest, Operation: "POST /v1/things"}
	e := event.Event{
		Fingerprint:   event.Fingerprint(p.name, "2", event.ChangeTypeNewFeature, subject, ""),
		API:           p.name,
		Version:       "2",
		ChangeType:    event.ChangeTypeNewFeature,
		Subject:       subject,
		Description:   "New operation POST /v1/things.",
		SourceURL:     "https://example.test/spec/2.json",
		SourceExcerpt: `{"post":{}}`,
		Confidence:    1,
		DetectedAt:    env.Now().UTC(),
	}
	content := []byte(`{"version":"2"}`)
	return Result{
		Label:     "v2",
		Events:    []event.Event{e},
		Baselines: []Spec{Baseline(p.name, "2", "v2", "https://example.test/spec/2.json", content, env.Now())},
	}, nil
}

func TestASourceRegisteredInATestIsFound(t *testing.T) {
	register(t, "test-registered")
	if p, ok := Lookup("test-registered/api"); !ok || p.Name() != "test-registered/api" || !slices.Contains(Names(), "test-registered/api") {
		t.Fatal("not registered")
	}
}

func TestNamesAreSorted(t *testing.T) {
	register(t, "test-b")
	register(t, "test-a")
	if !slices.IsSorted(Names()) {
		t.Errorf("Names() = %v, want sorted", Names())
	}
}

// register registers a test provider with one source, api, for the rest of
// the test only, so the test can run again (-count).
func register(t *testing.T, provider string) {
	t.Helper()
	RegisterProvider(Provider{ID: provider, Name: provider, Category: CategoryDevTools, Homepage: "https://example.test",
		Sources: []Source{{ID: "api", Layer: LayerAPI, Watches: "w", URLs: []string{"https://example.test/spec"}, Page: "providers/test.md", Plugin: testPlugin{name: provider + "/api"}}}})
	t.Cleanup(func() {
		mu.Lock()
		defer mu.Unlock()
		delete(registry, provider+"/api")
		delete(sources, provider+"/api")
		delete(providers, provider)
	})
}

func TestTwoSourcesCantShareAnID(t *testing.T) {
	register(t, "test-dup")
	defer func() {
		if recover() == nil {
			t.Error("a second source test-dup/api was registered")
		}
	}()
	register(t, "test-dup")
}
